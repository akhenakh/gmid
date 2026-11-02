package app

import (
	"github.com/akhenakh/gmid/internal/hands"
	"github.com/akhenakh/gmid/internal/song"
)

// NormalizeTracks reindexes tracks and notes after editor transforms.
func (a *App) NormalizeTracks() {
	if a.Song == nil {
		return
	}
	var id int64 = 1
	for i := range a.Song.Tracks {
		t := &a.Song.Tracks[i]
		t.Index = i
		for j := range t.Notes {
			t.Notes[j].Track = i
			t.Notes[j].ID = id
			id++
		}
	}
	a.Song.NoteCount = 0
	for i := range a.Song.Tracks {
		a.Song.NoteCount += len(a.Song.Tracks[i].Notes)
	}
}

// MergeTrack folds a track into the previous one.
func (a *App) MergeTrack(index int) {
	if a.Song == nil || index <= 0 || index >= len(a.Song.Tracks) {
		return
	}
	prev := &a.Song.Tracks[index-1]
	t := a.Song.Tracks[index]
	prev.Notes = append(prev.Notes, t.Notes...)
	t.Notes = nil
	t.Excluded = true
	a.NormalizeTracks()
	a.ReloadEngine()
}

// SplitHands replaces a track with left- and right-hand tracks.
func (a *App) SplitHands(index int) {
	if a.Song == nil || index < 0 || index >= len(a.Song.Tracks) {
		return
	}
	hands.AssignSong(a.Song, hands.Defaults())
	src := a.Song.Tracks[index]
	var left, right []song.Note
	for _, n := range src.Notes {
		if n.Hand == "L" {
			left = append(left, n)
		} else {
			right = append(right, n)
		}
	}
	if len(left) == 0 || len(right) == 0 {
		a.Notify("Split needs notes on both hands")
		return
	}
	src.Notes = nil
	src.Excluded = true
	mk := func(name string, notes []song.Note, colour string) {
		tr := src
		tr.Name = name
		tr.Color = colour
		tr.Notes = notes
		tr.Excluded = false
		a.Song.Tracks = append(a.Song.Tracks, tr)
	}
	mk(src.Name+" · LH", left, a.Prefs.HandLH)
	mk(src.Name+" · RH", right, a.Prefs.HandRH)
	a.NormalizeTracks()
	a.ReloadEngine()
}

// ReloadEngine rebuilds the timeline after model edits.
func (a *App) ReloadEngine() {
	if a.Engine == nil || a.Song == nil {
		return
	}
	snap := a.Engine.Snapshot()
	a.Engine.Load(a.Song)
	a.Engine.Seek(snap.Time)
	if snap.Playing {
		a.Engine.Play(snap.Time)
	}
	a.PushAudioState()
}
