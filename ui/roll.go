package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// PianoRoll is the falling-note surface.
func PianoRoll(a *app.App) {
	Container(Attrs(Expand, Grow(1), Clip, NoAnimate,
		BackgroundVec(Col("#0c0c10")), Corners(10),
		BorderWidth(1), BorderColorVec(Alpha(ColBorder, .5))), func() {
		if os.Getenv("GMID_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "rollouter w=%.1f h=%.1f\n", GetContentWidth(), GetContentHeight())
		}
		if a.Song == nil {
			rollEmptyState(a)
			return
		}
		drawRoll(a)
	})
}

func rollEmptyState(a *app.App) {
	Container(Attrs(Expand, MainAlign(AlignMiddle), CrossAlign(AlignMiddle), Gap(10), Pad(24)), func() {
		Icon(TypNotes, FontSize(44), TextColorVec(Alpha(ColAmber, .9)))
		Label("No song loaded", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
		Caption("Open a MIDI, MusicXML, Guitar Pro, Capella or karaoke file to begin.")
		Container(Attrs(Row, Gap(8), Pad(4)), func() {
			for i, path := range a.DemoFiles() {
				if i >= 3 {
					break
				}
				label := trim(path, 26)
				if Button(TypMediaPlay, label) {
					if err := a.LoadFile(path); err != nil {
						a.Notify(err.Error())
					}
				}
			}
		})
		Caption("Ctrl+O — open a file    ·    Ctrl+, — settings")
	})
}

func drawRoll(a *app.App) {
	w := GetContentWidth()
	h := GetContentHeight()
	if os.Getenv("GMID_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "roll w=%.1f h=%.1f rh=%.1f now=%.2f bars=%d beats=%d tracks=%d\n",
			w, h, GetResolvedHeight(), a.Transport.Time, len(a.Song.Bars), len(a.Song.Beats), len(a.Song.Tracks))
	}
	if w <= 0 || h <= 0 {
		return
	}
	min, max := KeyRange(a.Prefs.KeyRange)
	layout := BuildKeyLayout(w, min, max)
	now := a.Transport.Time
	visible := 6.0 / a.Prefs.RollZoom
	playY := h * 0.80
	pxPerSec := playY / float32(visible)
	yAt := func(t float64) float32 {
		return playY - float32(t-now)*pxPerSec
	}
	onScreen := func(y float32) bool { return y > -80 && y < h+80 }

	// Interaction: wheel scrubs, click seeks. Process before children so
	// ModAttrs runs at the right time.
	ProcessButtonEvents(false)
	if IsHovered() {
		scroll := GetFrameInput().Scroll[1]
		if scroll != 0 {
			step := 2.0
			if GetInputState().Modifiers&ModShift != 0 {
				step = 0.15
			}
			a.Seek(now - float64(scroll)*step)
		}
		if GetFrameInput().Mouse == MouseClick {
			rect := GetScreenRect()
			pt := GetInputState().MousePoint
			pct := (pt[1] - rect.Origin[1]) / rect.Size[1]
			t := now + float64(float32(0.80)-pct)*visible
			a.Seek(t)
		}
	}

	// Black-key columns give the roll its pitch grid.
	for m := min; m <= max; m++ {
		if !IsBlackKey(m) {
			continue
		}
		k := layout[m-min]
		Element(Attrs(Float(k.X, 0), FixSize(k.W, h), NoAnimate,
			BackgroundVec(Alpha(ColBlack, .22))))
	}

	bars, beats := a.Song.BarsAndBeats()
	for i, bt := range beats {
		y := yAt(bt)
		if !onScreen(y) {
			continue
		}
		c := Alpha(Col("#1c1c30"), .55)
		if i%4 == 0 {
			c = Alpha(Col("#2b2b47"), .9)
		}
		Element(Attrs(Float(0, y), FixSize(w, 1), NoAnimate, BackgroundVec(c)))
	}
	for i, bt := range bars {
		y := yAt(bt)
		if !onScreen(y) {
			continue
		}
		Element(Attrs(Float(0, y), FixSize(w, 1), NoAnimate,
			BackgroundVec(Alpha(Col("#3a3a5e"), .95))))
		if a.Prefs.BarNumbers {
			Pill(Attrs(Float(8, y-9), NoAnimate), func() {
				Label(fmt.Sprintf("%d", i+1), FontSize(9), TextColorVec(ColTextMute))
			})
		}
	}

	// Loop region behind the notes.
	if a.Transport.LoopEnabled {
		top := yAt(a.Transport.LoopEnd)
		bottom := yAt(a.Transport.LoopStart)
		if bottom > 0 && top < h {
			Container(Attrs(Float(0, top), FixSize(w, maxf(bottom-top, 2)), NoAnimate,
				BackgroundVec(Alpha(ColAmber, .06)),
				BorderWidth(1), BorderColorVec(Alpha(ColAmber, .45))), func() {})
		}
	}

	// Notes, culled to the visible time window.
	lo := now - visible*0.8 - 1
	hi := now + visible*0.2 + 1
	for ti := range a.Song.Tracks {
		t := &a.Song.Tracks[ti]
		if !a.Song.IsShown(t) {
			continue
		}
		base := TrackColor(t.Color)
		for ni := range t.Notes {
			n := &t.Notes[ni]
			if n.EndSec < lo || n.StartSec > hi {
				continue
			}
			key := n.Key + a.Transport.Transpose
			if key < min || key > max {
				continue
			}
			k := layout[key-min]
			top := yAt(n.EndSec)
			bottom := yAt(n.StartSec)
			hh := bottom - top
			if hh < 3 {
				hh = 3
			}
			col := base
			if a.Prefs.HandAssignment && n.Hand != "" {
				col = handColor(a, n.Hand)
			}
			a4 := 0.55 + float32(n.Velocity)/127*0.4
			Container(Attrs(Float(k.X+0.5, top), FixSize(k.W-1, hh), NoAnimate,
				BackgroundVec(Alpha(col, a4)), Corners(3),
				BorderWidth(1), BorderColorVec(Alpha(Shade(col, 12), .8))), func() {
				if k.W > 26 && hh > 24 {
					Label(NoteName(key, a.Prefs.NoteNaming, a.Prefs.Accidentals),
						FontSize(9), TextColorVec(Alpha(ColWhite, .7)))
				}
			})
		}
	}

	// Playhead and the played area below it.
	Element(Attrs(Float(0, playY), FixSize(w, h-playY), NoAnimate,
		BackgroundVec(Alpha(ColBlack, .28))))
	Element(Attrs(Float(0, playY-1), FixSize(w, 2), NoAnimate,
		BackgroundVec(Alpha(ColAmber, .9))))
	Pill(Attrs(Float(w-96, playY-20), NoAnimate, BackgroundVec(Alpha(ColPanel2, .95)),
		BorderColorVec(Alpha(ColAmber, .6))), func() {
		Label(formatClock(now), Fonts("JetBrains Mono"), FontSize(10), TextColorVec(ColAmber))
	})
}

func handColor(a *app.App, hand string) Vec4 {
	if hand == "L" {
		return Col(a.Prefs.HandLH)
	}
	if hand == "R" {
		return Col(a.Prefs.HandRH)
	}
	return ColAmber
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// LoopStrip shows the selected loop and its times.
func LoopStrip(a *app.App) {
	Container(Attrs(FixHeight(24), ExpandAcross(), Row, CrossMid, Gap(8),
		Pad2(0, 4), BackgroundVec(ColToolbar), Corners(6)), func() {
		SectionTitle("LOOP")
		if a.Transport.LoopEnabled {
			Badge(ColAmber, 7)
			Label(fmt.Sprintf("%s → %s", formatClock(a.Transport.LoopStart), formatClock(a.Transport.LoopEnd)),
				Fonts("JetBrains Mono"), FontSize(11), TextColorVec(ColText))
			Caption("Alt-drag the roll to refine")
			FlexSpacer()
			if CtrlButton(TypTimes, "Clear loop", true) {
				a.Transport.LoopEnabled = false
				a.Transport.LoopActive = false
				if a.Engine != nil {
					a.Engine.SetLoop(false, 0, 0)
				}
			}
		} else {
			Caption("No loop — Alt-drag the roll or press the loop button")
			FlexSpacer()
			if CtrlButton(TypPlus, "Loop 4 seconds from here", true) {
				a.ToggleLoop()
			}
		}
	})
}

func soundingKeys(a *app.App) map[int]KeyState {
	out := map[int]KeyState{}
	if a.Song == nil {
		return out
	}
	for key, tr := range a.SoundingNotes(a.Transport.Time) {
		out[key] = KeyState{Color: TrackColor(tr.Color)}
	}
	return out
}

func chordNameAt(a *app.App) string {
	if a.Song == nil {
		return "—"
	}
	notes := a.SoundingNotes(a.Transport.Time)
	keys := make([]int, 0, len(notes))
	for k := range notes {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return "—"
	}
	// Sort low to high so the detector sees the real bass.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return theory.DetectChord(keys)
}

func trim(s string, n int) string {
	b := []rune(filepath.Base(s))
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n-1]) + "…"
}
