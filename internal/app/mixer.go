package app

import (
	"math"

	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

// MixFor returns the console state for a channel.
func (a *App) MixFor(ch int) audio.ChannelMix {
	if a.Mixes == nil {
		a.Mixes = map[int]audio.ChannelMix{}
	}
	if m, ok := a.Mixes[ch]; ok {
		return m
	}
	m := audio.ChannelMix{Volume: 1, Pan: 0.5, Reverb: 0.4}
	if a.Song != nil {
		if c := a.Song.ChannelAt(ch); c != nil {
			m.Pan = float64(c.InitialPan) / 127
			m.Reverb = float64(c.InitialReverb) / 127
			m.Chorus = float64(c.InitialChorus) / 127
		}
	}
	a.Mixes[ch] = m
	return m
}

// SetMix updates one channel strip and sends it to the engine.
func (a *App) SetMix(ch int, m audio.ChannelMix) {
	if a.Mixes == nil {
		a.Mixes = map[int]audio.ChannelMix{}
	}
	a.Mixes[ch] = m
	a.MixesDirty = true
	if a.Engine != nil {
		a.Engine.SetChannelMix(ch, m.Volume, m.Pan, m.Chorus, m.Reverb)
	}
}

// CloseMixer closes the console, baking a changed mix into a versioned file.
func (a *App) CloseMixer() {
	if a.UI.MixerOpen && a.MixesDirty && a.Song != nil {
		a.BakeMixAndSave()
		a.MixesDirty = false
	}
	a.UI.MixerOpen = false
}

// BakeMixAndSave writes the current console mix into a versioned file.
func (a *App) BakeMixAndSave() {
	if a.Song == nil {
		return
	}
	for ch, m := range a.Mixes {
		c := a.Song.ChannelAt(ch)
		if c == nil {
			continue
		}
		c.InitialVolume = clampInt(int(m.Volume*100+0.5), 0, 127)
		c.InitialPan = clampInt(int(m.Pan*127+0.5), 0, 127)
		c.InitialChorus = clampInt(int(m.Chorus*127+0.5), 0, 127)
		c.InitialReverb = clampInt(int(m.Reverb*127+0.5), 0, 127)
		c.CC = bakeCC(c.CC, 7, c.InitialVolume)
		c.CC = bakeCC(c.CC, 10, c.InitialPan)
		c.CC = bakeCC(c.CC, 93, c.InitialChorus)
		c.CC = bakeCC(c.CC, 91, c.InitialReverb)
	}
	path, err := smf.SaveVersion(a.Song, a.Prefs.LibraryFolder, "mixer", "Saved console mixer")
	if err != nil {
		a.Notify("Save failed: " + err.Error())
		return
	}
	a.Notify("Saved " + path)
}

func bakeCC(events []song.CCEvent, which, value int) []song.CCEvent {
	out := events[:0]
	for _, e := range events {
		if e.Tick == 0 && e.CC == which {
			continue
		}
		out = append(out, e)
	}
	out = append(out, song.CCEvent{Tick: 0, Sec: 0, CC: which, Val: clampInt(value, 0, 127)})
	return out
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// UpdateChannelMeters returns a smoothed per-channel activity level in 0..1.
// The synth exposes only one mixed output, so the levels are derived from the
// notes and live keys scheduled on each channel, scaled by that channel's mix
// volume. Attack is immediate, release decays across frames like a peak meter.
func (a *App) UpdateChannelMeters() map[int]float64 {
	if a.channelMeter == nil {
		a.channelMeter = map[int]float64{}
	}
	sumSq := map[int]float64{}
	if a.Song != nil && a.Transport.Playing {
		at := a.Transport.Time
		for i := range a.Song.Tracks {
			t := &a.Song.Tracks[i]
			if !a.Song.IsShown(t) {
				continue
			}
			vol := a.MixFor(t.Channel).Volume
			for j := range t.Notes {
				n := &t.Notes[j]
				if n.StartSec <= at && at < n.EndSec {
					v := float64(n.Velocity) / 127 * vol
					sumSq[t.Channel] += v * v
				}
			}
		}
	}
	if len(a.LiveKeys) > 0 {
		vol := a.MixFor(15).Volume
		v := 100.0 / 127 * vol
		sumSq[15] += float64(len(a.LiveKeys)) * v * v
	}

	for ch, sq := range sumSq {
		inst := math.Sqrt(sq) * 0.5
		if inst > 1 {
			inst = 1
		}
		prev := a.channelMeter[ch]
		if inst >= prev {
			a.channelMeter[ch] = inst
		} else {
			a.channelMeter[ch] = prev*0.78 + inst*0.22
		}
	}
	for ch, prev := range a.channelMeter {
		if _, ok := sumSq[ch]; !ok {
			next := prev * 0.78
			if next < 0.01 {
				delete(a.channelMeter, ch)
			} else {
				a.channelMeter[ch] = next
			}
		}
	}
	out := make(map[int]float64, len(a.channelMeter))
	for ch, v := range a.channelMeter {
		out[ch] = v
	}
	return out
}
