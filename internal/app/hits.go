package app

import (
	"time"

	"github.com/akhenakh/gmid/internal/theory"
)

// Hit is one particle-effect spawn where a note strikes the keyboard.
type Hit struct {
	Key     int
	Born    time.Time
	Pattern string
	Color   string
	Seed    int
}

// UpdateHits scans note onsets crossed since the last frame and spawns effects.
func (a *App) UpdateHits() {
	if a.Song == nil || !a.Prefs.VisualEffects {
		a.Hits = nil
		a.lastHitScan = 0
		return
	}
	now := a.Transport.Time
	if !a.Transport.Playing {
		a.lastHitScan = now
		a.pruneHits()
		return
	}
	if now < a.lastHitScan || now-a.lastHitScan > 0.5 {
		a.lastHitScan = now
	} else {
		lo, hi := a.lastHitScan, now
		for ti := range a.Song.Tracks {
			t := &a.Song.Tracks[ti]
			if !a.Song.IsShown(t) || t.IsDrum {
				continue
			}
			for ni := range t.Notes {
				n := &t.Notes[ni]
				if n.StartSec > lo && n.StartSec <= hi {
					key := n.Key + a.Transport.Transpose
					if key < 0 || key > 127 {
						continue
					}
					a.hitSeq++
					a.Hits = append(a.Hits, Hit{
						Key: key, Born: time.Now(),
						Pattern: a.Prefs.VisualEffect,
						Color:   trackColour(t.Color),
						Seed:    a.hitSeq,
					})
				}
			}
		}
		a.lastHitScan = now
	}
	a.pruneHits()
	if len(a.Hits) > 256 {
		a.Hits = a.Hits[len(a.Hits)-256:]
	}
}

func (a *App) pruneHits() {
	if len(a.Hits) == 0 {
		return
	}
	cut := time.Now().Add(-time.Second)
	out := a.Hits[:0]
	for _, h := range a.Hits {
		if h.Born.After(cut) {
			out = append(out, h)
		}
	}
	a.Hits = out
}

// EffectColour is the configured effect colour (falls back to the track colour).
func (a *App) EffectColour(track string) string {
	if a.Prefs.EffectColor != "" {
		return a.Prefs.EffectColor
	}
	return trackColour(track)
}

func trackColour(c string) string {
	if c == "" {
		return theory.DefaultHandLH
	}
	return c
}
