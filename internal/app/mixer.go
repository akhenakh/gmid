package app

import (
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
