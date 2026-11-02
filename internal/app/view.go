package app

import (
	"github.com/akhenakh/gmid/internal/hands"
	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

// ReapplySongView reapplies view-only preferences (lead cut, hand colours)
// without re-reading the file.
func (a *App) ReapplySongView() {
	if a.Song == nil {
		return
	}
	if a.Prefs.SkipEmptyBars {
		a.Song.ApplyLeadCut(a.Prefs.KeepDrumCountIn)
	} else {
		a.Song.Lead = song.LeadCut{}
	}
	smf.ApplyHandColors(a.Song, a.Prefs.HandLH, a.Prefs.HandRH)
	if a.Prefs.HandAssignment {
		hands.AssignSong(a.Song, hands.Defaults())
	} else {
		for i := range a.Song.Tracks {
			for j := range a.Song.Tracks[i].Notes {
				a.Song.Tracks[i].Notes[j].Hand = ""
			}
		}
	}
	a.Transport.Time = a.clampTime(a.Transport.Time)
	if a.Engine != nil {
		snap := a.Engine.Snapshot()
		a.Engine.Load(a.Song)
		a.Engine.Seek(snap.Time)
		if snap.Playing {
			a.Engine.Play(snap.Time)
		}
		a.PushAudioState()
	}
}

func (a *App) clampTime(t float64) float64 {
	if a.Song == nil {
		return 0
	}
	if t < a.Song.OriginSec() {
		t = a.Song.OriginSec()
	}
	if t > a.Song.DurationSec {
		t = a.Song.DurationSec
	}
	return t
}
