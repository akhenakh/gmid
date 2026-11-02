package app

import (
	"sort"

	"github.com/akhenakh/gmid/internal/song"

	shirei "go.hasen.dev/shirei"
)

// Engine is the playback contract the app drives. The realtime and offline
// engines both implement it.
type Engine interface {
	Play(from float64)
	Pause()
	Stop()
	Seek(float64)
	SetSpeed(float64)
	SetTranspose(int)
	SetLoop(enabled bool, start, end float64)
	SetMetronome(on bool, volume float64)
	SetMasterVolume(float64)
	SetTrackAudible(track int, audible bool)
	SetChannelMix(ch int, volume, pan, chorus, reverb float64)
	NoteOn(ch, key, vel int)
	NoteOff(ch, key int)
	Close()
}

// TogglePlay starts or pauses playback.
func (a *App) TogglePlay() {
	if a.Song == nil {
		return
	}
	if a.Transport.Playing {
		a.Transport.Playing = false
		if a.Engine != nil {
			a.Engine.Pause()
		}
		return
	}
	if a.Transport.Time < a.Song.OriginSec() || a.Transport.Time >= a.Song.DurationSec {
		a.Transport.Time = a.Song.OriginSec()
	}
	a.Transport.Playing = true
	if a.Engine != nil {
		a.Engine.Play(a.Transport.Time)
	}
	shirei.RequestNextFrame()
}

// Stop pauses and rewinds to the visible start.
func (a *App) Stop() {
	a.Transport.Playing = false
	if a.Engine != nil {
		a.Engine.Stop()
	}
	if a.Song != nil {
		a.Transport.Time = a.Song.OriginSec()
	}
	shirei.RequestNextFrame()
}

// Seek moves the playhead.
func (a *App) Seek(t float64) {
	if a.Song == nil {
		return
	}
	lo := a.Song.OriginSec()
	hi := a.Song.DurationSec
	if t < lo {
		t = lo
	}
	if t > hi {
		t = hi
	}
	a.Transport.Time = t
	if a.Engine != nil {
		a.Engine.Seek(t)
	}
	shirei.RequestNextFrame()
}

// ToggleLoop turns the loop region on or off, creating a region when none exists.
func (a *App) ToggleLoop() {
	if a.Song == nil {
		return
	}
	if !a.Transport.LoopEnabled {
		if a.Transport.LoopEnd <= a.Transport.LoopStart {
			start := a.Transport.Time
			end := start + 4
			if end > a.Song.DurationSec {
				end = a.Song.DurationSec
			}
			if end-start < 0.5 {
				start = a.Song.OriginSec()
				end = a.Song.DurationSec
			}
			a.Transport.LoopStart, a.Transport.LoopEnd = start, end
		}
		a.Transport.LoopEnabled = true
	} else {
		a.Transport.LoopEnabled = false
	}
	a.Transport.LoopActive = a.Transport.LoopEnabled
	if a.Engine != nil {
		a.Engine.SetLoop(a.Transport.LoopEnabled, a.Transport.LoopStart, a.Transport.LoopEnd)
	}
	shirei.RequestNextFrame()
}

// ToggleMetronome flips the click on or off.
func (a *App) ToggleMetronome() {
	a.Prefs.Metronome = !a.Prefs.Metronome
	_ = a.Prefs.Save()
	if a.Engine != nil {
		a.Engine.SetMetronome(a.Prefs.Metronome, a.Prefs.MetronomeVolume)
	}
	shirei.RequestNextFrame()
}

// SetSpeed changes the playback ratio (tempo without pitch change).
func (a *App) SetSpeed(v float64) {
	if v < 0.25 {
		v = 0.25
	}
	if v > 2 {
		v = 2
	}
	a.Transport.Speed = v
	if a.Engine != nil {
		a.Engine.SetSpeed(v)
	}
	shirei.RequestNextFrame()
}

// SetTranspose changes the key by semitones.
func (a *App) SetTranspose(n int) {
	if n < -24 {
		n = -24
	}
	if n > 24 {
		n = 24
	}
	a.Transport.Transpose = n
	if a.Engine != nil {
		a.Engine.SetTranspose(n)
	}
	shirei.RequestNextFrame()
}

// SoundingNotes returns the transposed notes sounding at the playhead across
// shown, keyboard-enabled tracks.
func (a *App) SoundingNotes(at float64) map[int]*song.Track {
	out := map[int]*song.Track{}
	if a.Song == nil {
		return out
	}
	for i := range a.Song.Tracks {
		t := &a.Song.Tracks[i]
		if !a.Song.IsShown(t) || !t.OnKeyboard || t.IsDrum {
			continue
		}
		for j := range t.Notes {
			n := &t.Notes[j]
			if n.StartSec <= at && at < n.EndSec {
				out[n.Key+a.Transport.Transpose] = t
			}
		}
	}
	return out
}

// StepThrough moves to the next or previous played moment (a note or chord
// onset), optionally to the next bar.
func (a *App) StepThrough(dir int, bar bool) {
	if a.Song == nil {
		return
	}
	if a.Transport.Playing {
		a.Transport.Playing = false
		if a.Engine != nil {
			a.Engine.Pause()
		}
	}
	onsets := a.playedOnsets()
	if len(onsets) == 0 {
		return
	}
	cur := a.Transport.Time
	target := cur
	if bar {
		bars := a.Song.BarStarts()
		if dir > 0 {
			for _, b := range bars {
				if b > cur+1e-4 {
					target = b
					break
				}
			}
		} else {
			for i := len(bars) - 1; i >= 0; i-- {
				if bars[i] < cur-1e-4 {
					target = bars[i]
					break
				}
			}
		}
		// Snap to the first played moment in that bar.
		for _, o := range onsets {
			if o >= target-1e-4 {
				target = o
				break
			}
		}
	} else if dir > 0 {
		for _, o := range onsets {
			if o > cur+1e-4 {
				target = o
				break
			}
		}
	} else {
		for i := len(onsets) - 1; i >= 0; i-- {
			if onsets[i] < cur-1e-4 {
				target = onsets[i]
				break
			}
		}
	}
	a.Seek(target)
	a.Transport.Playing = false
}

func (a *App) playedOnsets() []float64 {
	if a.Song == nil {
		return nil
	}
	var raw []float64
	for i := range a.Song.Tracks {
		t := &a.Song.Tracks[i]
		if !a.Song.IsShown(t) || t.IsDrum {
			continue
		}
		for j := range t.Notes {
			raw = append(raw, t.Notes[j].StartSec)
		}
	}
	sort.Float64s(raw)
	out := raw[:0]
	for _, v := range raw {
		if len(out) == 0 || v-out[len(out)-1] > 0.03 {
			out = append(out, v)
		}
	}
	return out
}
