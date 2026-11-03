package app

import (
	"math"
	"time"

	shirei "go.hasen.dev/shirei"
)

// clockSnapThreshold is the error above which the UI clock trusts the audio
// thread outright. Seek and loop wraps produce jumps much larger than this;
// ordinary audio-block granularity stays well below it.
const clockSnapThreshold = 0.25

// clockSlew is the fraction of the remaining error corrected per frame. The
// UI clock advances with wall time and is nudged toward the audio clock, so
// its motion stays smooth even though the audio thread publishes its position
// only once per buffer.
const clockSlew = 0.1

// SyncEngine copies the audio clock into the transport. Called once per frame
// before the UI is built.
func (a *App) SyncEngine() {
	if a.Engine == nil {
		return
	}
	snap := a.Engine.Snapshot()
	now := time.Now()

	// The engine publishes its position once per audio block. At a 120 Hz
	// display a 512-frame block is longer than a frame, so copying snap.Time
	// verbatim leaves the roll frozen on about a third of frames and then
	// makes it jump. Advance a UI clock with wall time and slew it toward the
	// audio position instead. Count-in does not advance the song position, so
	// it stays verbatim.
	if !snap.Playing || snap.CountIn {
		a.Transport.Time = snap.Time
		a.Transport.Playing = snap.Playing
		a.clockValid = false
		if snap.Playing {
			shirei.RequestNextFrame()
		}
		return
	}

	a.Transport.Time = a.advanceClock(snap.Time, now)
	a.Transport.Playing = true
	shirei.RequestNextFrame()
}

// advanceClock moves the UI clock toward the audio position. The result is
// always evaluated against the previous UI time, not the quantized snapshot,
// so the roll moves every frame instead of once per audio buffer.
func (a *App) advanceClock(snapTime float64, now time.Time) float64 {
	if !a.clockValid {
		a.clockTime = snapTime
		a.clockWall = now
		a.clockValid = true
		return a.clockTime
	}
	dt := now.Sub(a.clockWall).Seconds()
	switch {
	case dt < 0 || dt > clockSnapThreshold:
		a.clockTime = snapTime
	default:
		predicted := a.clockTime + dt*a.Transport.Speed
		err := snapTime - predicted
		if math.Abs(err) > clockSnapThreshold {
			a.clockTime = snapTime
		} else {
			a.clockTime = predicted + err*clockSlew
		}
	}
	a.clockWall = now
	return a.clockTime
}

// PushAudioState re-sends mute/solo, master level and metronome state after a
// UI change.
func (a *App) PushAudioState() {
	if a.Engine == nil || a.Song == nil {
		return
	}
	anySolo := a.Song.AnySolo()
	for i := range a.Song.Tracks {
		t := &a.Song.Tracks[i]
		audible := !t.Muted && !t.Excluded && (!anySolo || t.Solo)
		a.Engine.SetTrackAudible(i, audible)
	}
	a.Engine.SetMasterVolume(a.Prefs.MasterVolume)
	a.Engine.SetMasterTone(a.Prefs.MasterTone)
	a.Engine.SetCompressor(a.Prefs.MasterCompressor)
	a.Engine.SetMetronome(a.Prefs.Metronome, a.Prefs.MetronomeVolume)
	if a.Prefs.CountIn {
		a.Engine.SetCountIn(a.Prefs.CountInBars)
	} else {
		a.Engine.SetCountIn(0)
	}
	a.Engine.SetSpeed(a.Transport.Speed)
	a.Engine.SetTranspose(a.Transport.Transpose)
	if a.Transport.LoopEnabled {
		a.Engine.SetLoop(true, a.Transport.LoopStart, a.Transport.LoopEnd)
	}
}
