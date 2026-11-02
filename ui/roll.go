package ui

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

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

// loopDrag holds the Alt-drag anchor between frames.
type loopDrag struct {
	anchor float64
	active bool
}

// rollWindow returns the time range the roll draws: the playhead sits at 80%
// height, so most of the surface shows upcoming notes.
func rollWindow(now, visible float64) (lo, hi float64) {
	return now - visible*0.2 - 1, now + visible*0.8 + 1
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
	min, max := EffectiveKeyRange(a)
	layout := BuildKeyLayout(w, min, max)
	now := a.Transport.Time
	visible := 6.0 / a.Prefs.RollZoom
	playY := h * 0.80
	pxPerSec := playY / float32(visible)
	yAt := func(t float64) float32 {
		return playY - float32(t-now)*pxPerSec
	}
	onScreen := func(y float32) bool { return y > -80 && y < h+80 }

	// Interaction: wheel scrubs, click seeks, Alt-drag draws a loop.
	ProcessButtonEvents(false)
	if a.Edit.Active {
		editorInteraction(a, layout, min, max, now, visible, playY, pxPerSec, IsActive())
	} else {
		ld := Use[loopDrag]("loop-drag")
		if IsHovered() {
			rect := GetScreenRect()
			pt := GetInputState().MousePoint
			pct := (pt[1] - rect.Origin[1]) / rect.Size[1]
			tAt := now + float64(float32(0.80)-pct)*visible
			scroll := GetFrameInput().Scroll[1]
			if scroll != 0 {
				step := 2.0
				if GetInputState().Modifiers&ModShift != 0 {
					step = 0.15
				}
				a.Seek(now - float64(scroll)*step)
			}
			if GetFrameInput().Mouse == MouseClick {
				if GetFrameInput().ClickCount >= 2 {
					a.Transport.LoopEnabled = false
					a.Transport.LoopActive = false
					if a.Engine != nil {
						a.Engine.SetLoop(false, 0, 0)
					}
				} else if GetInputState().Modifiers&ModAlt != 0 {
					ld.active = true
					ld.anchor = tAt
					a.Transport.LoopStart, a.Transport.LoopEnd = tAt, tAt
					a.Transport.LoopEnabled = true
				} else {
					a.Seek(tAt)
				}
			}
			if ld.active && IsActive() {
				lo, hi := ld.anchor, tAt
				if lo > hi {
					lo, hi = hi, lo
				}
				a.Transport.LoopStart, a.Transport.LoopEnd = lo, hi
				a.Transport.LoopEnabled = true
				if a.Engine != nil {
					a.Engine.SetLoop(true, lo, hi)
				}
			}
			if ld.active && !IsActive() {
				ld.active = false
				if a.Transport.LoopEnd-a.Transport.LoopStart < 0.05 {
					a.Transport.LoopEnabled = false
					if a.Engine != nil {
						a.Engine.SetLoop(false, 0, 0)
					}
				}
			}
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

	// Notes, culled to the visible time window. The playhead sits at 80% of the
	// roll height: 80% of the window is the future (above it), 20% is the past.
	lo, hi := rollWindow(now, visible)
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
			sel := a.Edit.Active && a.Edit.Selected[n.ID]
			bw := float32(1)
			bc := Alpha(Shade(col, 12), .8)
			if sel {
				bw = 2
				bc = ColWhite
			}
			Container(Attrs(Float(k.X+0.5, top), FixSize(k.W-1, hh), NoAnimate,
				BackgroundVec(Alpha(col, a4)), Corners(3),
				BorderWidth(bw), BorderColorVec(bc)), func() {
				if k.W > 26 && hh > 24 {
					Label(NoteName(key, a.Prefs.NoteNaming, a.Prefs.Accidentals),
						FontSize(9), TextColorVec(Alpha(ColWhite, .7)))
				}
			})
		}
	}

	// Note Editor marquee.
	if a.Edit.Active {
		d := Use[editDrag]("note-edit-drag")
		if d.mode == "marquee" {
			x0, x1 := d.anchorX, d.curX
			if x0 > x1 {
				x0, x1 = x1, x0
			}
			y0, y1 := d.anchorY, d.curY
			if y0 > y1 {
				y0, y1 = y1, y0
			}
			Container(Attrs(Float(x0, y0), FixSize(maxf(x1-x0, 1), maxf(y1-y0, 1)),
				NoAnimate, BackgroundVec(Alpha(ColBlue, .12)),
				BorderWidth(1), BorderColorVec(Alpha(ColBlue, .8))), func() {})
		}
	}

	// Hit effects at the playhead.
	if a.Prefs.VisualEffects {
		drawHitEffects(a, layout, min, h, playY)
	}

	// Playhead and the played area below it.
	Element(Attrs(Float(0, playY), FixSize(w, h-playY), NoAnimate,
		BackgroundVec(Alpha(ColBlack, .28))))
	Element(Attrs(Float(26, playY-1), FixSize(w-26, 2), NoAnimate,
		BackgroundVec(Alpha(ColAmber, .9))))
	Pill(Attrs(Float(w-96, playY-20), NoAnimate, BackgroundVec(Alpha(ColPanel2, .95)),
		BorderColorVec(Alpha(ColAmber, .6))), func() {
		Label(formatClock(now), Fonts("JetBrains Mono"), FontSize(10), TextColorVec(ColAmber))
	})
}

func drawHitEffects(a *app.App, layout []KeyLayout, min int, h, playY float32) {
	now := time.Now()
	for _, hit := range a.Hits {
		age := now.Sub(hit.Born).Seconds()
		if age < 0 || age > 0.65 {
			continue
		}
		if hit.Key < min || hit.Key-min >= len(layout) {
			continue
		}
		k := layout[hit.Key-min]
		cx := k.X + k.W/2
		col := Col(hit.Color)
		t := float32(age / 0.65)
		fade := (1 - t) * (1 - t)
		pattern := hit.Pattern
		if pattern == "" {
			pattern = "glow"
		}
		switch pattern {
		case "ripple":
			size := 10 + 90*t
			Element(Attrs(Float(cx-size/2, playY-size/2), FixSize(size, size),
				Corners(size/2), BorderWidth(2), BorderColorVec(Alpha(col, fade)), NoAnimate))
		case "burst":
			for i := 0; i < 8; i++ {
				ang := float64(i)*math.Pi/4 + float64(hit.Seed%7)*0.2
				r := float64(8 + 42*t)
				px := cx + float32(math.Cos(ang)*r)
				py := playY + float32(math.Sin(ang)*r)
				sz := 8 * (1 - t)
				Element(Attrs(Float(px-sz/2, py-sz/2), FixSize(sz, sz), Corners(sz/2),
					BackgroundVec(Alpha(col, fade)), NoAnimate))
			}
		case "smoke":
			for i := 0; i < 3; i++ {
				sz := (14 + float32(i)*8) * (1 + t)
				px := cx + float32(i-1)*10*t
				py := playY - (18+float32(i)*10)*t
				Element(Attrs(Float(px-sz/2, py-sz/2), FixSize(sz, sz), Corners(sz/2),
					BackgroundVec(Alpha(Col("#9a9aa8"), .25*fade)), NoAnimate, Glow(10)))
			}
		case "aura":
			sz := k.W + 30*t
			Element(Attrs(Float(cx-sz/2, playY-sz/2), FixSize(sz, sz), Corners(sz/2),
				BackgroundVec(Alpha(col, .20*fade)), NoAnimate, Glow(16)))
		case "nova":
			for i := 0; i < 8; i++ {
				ang := float64(i) * math.Pi / 4
				len := float64(16 + 56*t)
				px := cx + float32(math.Cos(ang)*len/2)
				py := playY + float32(math.Sin(ang)*len/2)
				w := float32(2)
				hh := float32(len)
				if i%2 == 1 {
					w, hh = hh, w
				}
				Element(Attrs(Float(px-w/2, py-hh/2), FixSize(w, hh), NoAnimate,
					BackgroundVec(Alpha(col, .7*fade))))
			}
		case "comet":
			for i := 0; i < 6; i++ {
				sz := (10 - float32(i)) * (1 - t*0.4)
				px := cx
				py := playY - float32(i)*8 - 30*t
				Element(Attrs(Float(px-sz/2, py-sz/2), FixSize(sz, sz), Corners(sz/2),
					BackgroundVec(Alpha(col, fade*float32(1-float32(i)/6))), NoAnimate))
			}
		default: // glow
			sz := 26 + 70*t
			Element(Attrs(Float(cx-sz/2, playY-sz/2), FixSize(sz, sz), Corners(sz/2),
				BackgroundVec(Alpha(col, .35*fade)), NoAnimate, Glow(14)))
		}
	}
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
	for key, sounding := range a.SoundingNotes(a.Transport.Time) {
		c := TrackColor(sounding.Track.Color)
		if a.Prefs.HandAssignment && sounding.Note.Hand != "" {
			c = handColor(a, sounding.Note.Hand)
		}
		out[key] = KeyState{Color: c}
		if a.Prefs.HandTags && sounding.Note.Hand != "" {
			out[key] = KeyState{Color: c, Labels: []string{sounding.Note.Hand}}
		}
	}
	for key := range a.LiveKeys {
		out[key] = KeyState{Color: ColAmber}
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
