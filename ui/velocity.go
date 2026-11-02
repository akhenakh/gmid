package ui

import (
	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/song"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// VelocityLane edits note velocities while the Note Editor is active.
func VelocityLane(a *app.App) {
	Container(Attrs(FixHeight(66), ExpandAcross(), Clip, NoAnimate,
		BackgroundVec(Col("#0c0c10")), Corners(6), BorderWidth(1),
		BorderColorVec(Alpha(ColBorder, .5))), func() {
		w := GetContentWidth()
		h := GetContentHeight()
		t := a.EditedTrack()
		if t == nil || w <= 0 {
			return
		}
		min, max := EffectiveKeyRange(a)
		layout := BuildKeyLayout(w, min, max)
		now := a.Transport.Time
		visible := 6.0 / a.Prefs.RollZoom
		lo, hi := rollWindow(now, visible)

		ProcessButtonEvents(false)
		if IsActive() {
			rect := GetScreenRect()
			pt := GetInputState().MousePoint
			x := pt[0] - rect.Origin[0]
			y := pt[1] - rect.Origin[1]
			vel := int((1 - (y-6)/(h-12)) * 127)
			if vel < 1 {
				vel = 1
			}
			if vel > 127 {
				vel = 127
			}
			if note := nearestVelocityNote(t, x, layout, min, max, lo, hi, a.Transport.Transpose); note != nil {
				a.SelectNote(note.ID, false)
				a.SetSelectedVelocity(vel)
			}
		}

		Caption("VELOCITY")
		for i := range t.Notes {
			n := &t.Notes[i]
			if n.EndSec < lo || n.StartSec > hi {
				continue
			}
			key := n.Key + a.Transport.Transpose
			if key < min || key > max {
				continue
			}
			k := layout[key-min]
			bh := float32(n.Velocity) / 127 * (h - 20)
			c := ColAmber
			if a.Edit.Selected[n.ID] {
				c = ColWhite
			}
			Element(Attrs(Float(k.X+k.W/2-2, h-6-bh), FixSize(4, bh), NoAnimate,
				BackgroundVec(Alpha(c, .8))))
		}
	})
}

func nearestVelocityNote(t *song.Track, x float32, layout []KeyLayout, min, max int, lo, hi float64, transpose int) *song.Note {
	var best *song.Note
	bestD := float32(12)
	for i := range t.Notes {
		n := &t.Notes[i]
		if n.StartSec < lo || n.StartSec > hi {
			continue
		}
		key := n.Key + transpose
		if key < min || key > max {
			continue
		}
		k := layout[key-min]
		d := absf32(x - (k.X + k.W/2))
		if d < bestD {
			bestD = d
			best = n
		}
	}
	return best
}

func absf32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
