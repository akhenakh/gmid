package mcp

import (
	"github.com/akhenakh/gmid/internal/song"
	"github.com/akhenakh/gmid/internal/theory"
)

type songView struct {
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Identity  string      `json:"identity"`
	Duration  float64     `json:"durationSec"`
	Origin    float64     `json:"originSec"`
	PPQ       int         `json:"ppq"`
	BPM       float64     `json:"bpm"`
	Key       string      `json:"key"`
	Bars      int         `json:"bars"`
	Beats     int         `json:"beats"`
	NoteCount int         `json:"noteCount"`
	Suspect   bool        `json:"tempoSuspect"`
	BeatFixed bool        `json:"tapTempoFixed"`
	LeadBar   int         `json:"leadBar"`
	Tracks    []trackView `json:"tracks"`
}

type trackView struct {
	Index      int    `json:"index"`
	Name       string `json:"name"`
	GMName     string `json:"gmName"`
	Program    int    `json:"program"`
	Channel    int    `json:"channel"`
	Group      string `json:"group"`
	Color      string `json:"color"`
	Notes      int    `json:"notes"`
	IsDrum     bool   `json:"isDrum"`
	Muted      bool   `json:"muted"`
	Solo       bool   `json:"solo"`
	Visible    bool   `json:"visible"`
	OnKeyboard bool   `json:"onKeyboard"`
	Excluded   bool   `json:"excluded"`
}

type noteView struct {
	ID       int64   `json:"id"`
	Track    int     `json:"track"`
	Key      int     `json:"key"`
	Tick     int64   `json:"tick"`
	Duration int64   `json:"durationTicks"`
	Start    float64 `json:"startSec"`
	End      float64 `json:"endSec"`
	Velocity int     `json:"velocity"`
	Hand     string  `json:"hand,omitempty"`
	Conf     float32 `json:"handConfidence,omitempty"`
}

func viewSong(s *song.Song) songView {
	v := songView{}
	if s == nil {
		return v
	}
	v.Name = s.Name
	v.Path = s.Path
	v.Identity = s.Identity
	v.Duration = s.DurationSec
	v.Origin = s.OriginSec()
	v.PPQ = s.PPQ
	v.BPM = s.EffectiveBPM(s.OriginSec())
	v.Bars = len(s.Bars)
	v.Beats = len(s.Beats)
	v.NoteCount = s.NoteCount
	v.Suspect = s.TempoSuspect
	v.BeatFixed = s.BeatFixed
	v.LeadBar = s.Lead.Bar
	if len(s.Keys) > 0 {
		k := s.Keys[0]
		v.Key = theory.NoteName(k.Semitone, "english", "flat")
		if k.Minor {
			v.Key += "m"
		}
	}
	for i := range s.Tracks {
		v.Tracks = append(v.Tracks, viewTrack(s, i))
	}
	return v
}

func viewTrack(s *song.Song, i int) trackView {
	t := &s.Tracks[i]
	return trackView{
		Index: i, Name: t.Name, GMName: t.GMName, Program: t.Program,
		Channel: t.Channel, Group: t.Group, Color: t.Color, Notes: len(t.Notes),
		IsDrum: t.IsDrum, Muted: t.Muted, Solo: t.Solo, Visible: t.Visible,
		OnKeyboard: t.OnKeyboard, Excluded: t.Excluded,
	}
}

func viewNote(n *song.Note) noteView {
	return noteView{
		ID: n.ID, Track: n.Track, Key: n.Key, Tick: n.Tick,
		Duration: n.DurTicks, Start: n.StartSec, End: n.EndSec,
		Velocity: n.Velocity, Hand: n.Hand, Conf: n.Conf,
	}
}
