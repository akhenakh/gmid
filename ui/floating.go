package ui

import (
	"github.com/akhenakh/gmid/internal/app"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// FloatingKeyboard is a draggable, resizable keyboard panel.
func FloatingKeyboard(a *app.App) {
	win := GetHost().WindowSize
	w := minf(a.UI.FloatW, win[0]-20)
	h := minf(a.UI.FloatH, win[1]-20)
	x := clampf2(a.UI.FloatX, 0, maxf(win[0]-w, 0))
	y := clampf2(a.UI.FloatY, 0, maxf(win[1]-h, 0))
	Container(Attrs(Float(x, y), Z(60), FixSize(w, h), NoAnimate, Clip,
		BackgroundVec(ColToolbar), Corners(12), BorderWidth(1),
		BorderColorVec(Alpha(ColAmber, .55)), BoxShadow(18)), func() {

		// Header drag handle.
		Container(Attrs(Row, CrossMid, FixHeight(30), ExpandAcross(), Pad2(0, 8), Gap(6),
			BackgroundVec(ColPanel)), func() {
			ProcessButtonEvents(false)
			if IsActive() {
				m := GetFrameInput().Motion
				a.UI.FloatX += m[0]
				a.UI.FloatY += m[1]
			}
			Icon(TypKeyboard, FontSize(14), TextColorVec(ColAmber))
			Label("Keyboard", FontSize(12), FontWeight(WeightBold), TextColorVec(ColText))
			FlexSpacer()
			if CtrlButton(TypArrowMinimise, "Dock", true) {
				a.UI.FloatingKeys = false
			}
			if CtrlButton(TypTimes, "Hide", true) {
				a.UI.FloatingKeys = false
			}
		})
		Container(Attrs(Expand, Clip, Pad2(0, 2), BackgroundVec(Col("#08080b"))), func() {
			min, max := EffectiveKeyRange(a)
			PianoKeyboard(min, max, maxf(h-32, 80), soundingKeys(a),
				a.Prefs.ShowNoteNames, a.Prefs.ShowOctaves, a.KeyDown, a.KeyUp)
			// Resize handle.
			Container(Attrs(Float(GetContentWidth()-22, GetContentHeight()-22), Z(5),
				FixSize(20, 20), NoAnimate, Corners(5),
				BackgroundVec(ColAmber), BorderWidth(2),
				BorderColorVec(Col("#181005"))), func() {
				ProcessButtonEvents(false)
				if IsActive() {
					m := GetFrameInput().Motion
					a.UI.FloatW = maxf(a.UI.FloatW+m[0], 320)
					a.UI.FloatH = maxf(a.UI.FloatH+m[1], 140)
				}
			})
		})
	})
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func clampf2(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
