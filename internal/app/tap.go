package app

import (
	"fmt"
	"sort"
	"time"

	shirei "go.hasen.dev/shirei"

	"github.com/akhenakh/gmid/internal/config"
)

// TapTempo records a tap and applies the fitted grid after four taps.
func (a *App) TapTempo() {
	now := time.Now()
	if !a.tapLast.IsZero() && now.Sub(a.tapLast) > 2500*time.Millisecond {
		a.tapWall = nil
		a.tapSong = nil
	}
	a.tapLast = now
	a.tapWall = append(a.tapWall, float64(now.UnixNano())/1e9)
	if a.Transport.Playing {
		a.tapSong = append(a.tapSong, a.Transport.Time)
	} else {
		a.tapSong = append(a.tapSong, a.Transport.Time)
	}
	if len(a.tapWall) >= 4 {
		a.applyTapFit()
	}
	shirei.RequestNextFrame()
}

// TapCount is the number of taps in the current session.
func (a *App) TapCount() int { return len(a.tapWall) }

func (a *App) applyTapFit() {
	if a.Song == nil || len(a.tapWall) < 4 {
		return
	}
	gaps := make([]float64, 0, len(a.tapWall)-1)
	for i := 1; i < len(a.tapWall); i++ {
		g := a.tapWall[i] - a.tapWall[i-1]
		if g > 0.15 && g < 2.5 {
			gaps = append(gaps, g)
		}
	}
	if len(gaps) < 3 {
		a.Notify("Tap a steady beat (at least 4 taps)")
		return
	}
	sort.Float64s(gaps)
	period := gaps[len(gaps)/2]
	// Anchor on the first tap's song position; beats line up with the music.
	anchor := a.Transport.Time
	startTaps := a.tapSong[0]
	if len(a.tapSong) >= 2 {
		// The beat clock at the first tap is startTaps; keep the phase.
		anchor = startTaps
	}
	num, _ := a.Song.SigAt(anchor)
	if num <= 0 {
		num = 4
	}
	a.Song.ApplyBeatFix(period, anchor, num)
	if a.Prefs.TempoFixes == nil {
		a.Prefs.TempoFixes = map[string]config.TempoFix{}
	}
	a.Prefs.TempoFixes[a.Song.Identity] = config.TempoFix{PeriodSec: period, AnchorSec: anchor, BeatsPerBar: num}
	_ = a.Prefs.Save()
	a.tapWall = nil
	a.tapSong = nil
	a.ReloadEngine()
	a.Notify(fmt.Sprintf("Tap tempo: %.0f BPM", 60/period))
}

// ClearTempoFix removes a song's tapped grid.
func (a *App) ClearTempoFix() {
	if a.Song == nil {
		return
	}
	delete(a.Prefs.TempoFixes, a.Song.Identity)
	_ = a.Prefs.Save()
	a.Song.ClearBeatFix()
	a.ReloadEngine()
	a.Notify("Tempo grid restored")
}

// ApplyStoredTempoFix installs a remembered correction after loading a song.
func (a *App) ApplyStoredTempoFix() {
	if a.Song == nil || a.Prefs.TempoFixes == nil {
		return
	}
	if fix, ok := a.Prefs.TempoFixes[a.Song.Identity]; ok && fix.PeriodSec > 0 {
		bpb := fix.BeatsPerBar
		if bpb < 1 {
			bpb = 4
		}
		a.Song.ApplyBeatFix(fix.PeriodSec, fix.AnchorSec, bpb)
	}
}
