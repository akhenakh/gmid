package ui

import (
	"fmt"
	"math"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// ScaleExplorer is the interactive Circle of Fifths and diatonic chord view.
func ScaleExplorer(a *app.App) {
	Modal(1000, func() { a.UI.ScaleExplorerOpen = false }, func() {
		ModAttrs(Pad(16), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Container(Attrs(Gap(0)), func() {
				Label("Scale Explorer", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
				Caption("Pick a key on the circle; the scale's diatonic chords follow.")
			})
			FlexSpacer()
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.UI.ScaleExplorerOpen = false
			}
		})
		HDivider()
		Container(Attrs(Row, Gap(16), ExpandAcross(), FixHeight(500)), func() {
			Container(Attrs(FixSize(320, 340), NoAnimate), func() {
				var img ImageId
				if a.UI.ScaleStarmap {
					img = starmap(a)
				} else {
					img = circleOfFifths(a)
				}
				ProcessButtonEvents(false)
				if !a.UI.ScaleStarmap && IsHovered() && GetFrameInput().Mouse == MouseClick {
					rect := GetScreenRect()
					pt := GetInputState().MousePoint
					cx := rect.Origin[0] + rect.Size[0]/2
					cy := rect.Origin[1] + rect.Size[1]/2
					dx := float64(pt[0] - cx)
					dy := float64(pt[1] - cy)
					if dx*dx+dy*dy > 22*22 {
						ang := math.Atan2(dy, dx)
						idx := int(math.Round((ang + math.Pi/2) / (math.Pi / 6)))
						idx = ((idx % 12) + 12) % 12
						a.UI.ScaleRoot = circleOrder[idx]
					}
				}
				ImageViewAt(img, Vec2{300, 300})
			})
			Container(Attrs(Grow(1), Expand, Gap(8)), func() {
				Caption("SCALE")
				Container(Attrs(Row, Wrap, Gap(4)), func() {
					for _, sc := range theory.Scales {
						if a.UI.ScaleType == sc.ID {
							NextButtonType(ButtonPrimary)
						}
						if CtrlButton(NoIcon, sc.Name, true) {
							a.UI.ScaleType = sc.ID
						}
					}
				})
				HDivider()
				scale := theory.ScaleByID(a.UI.ScaleType)
				chords := theory.DiatonicChords(a.UI.ScaleRoot, 4, scale)
				Caption("CHORDS IN THE SCALE")
				Container(Attrs(Row, Wrap, Gap(6), MaxWidth(600)), func() {
					for _, c := range chords {
						card := c
						ContainerWithKey(card.Degree, Attrs(FixWidth(118), Corners(8), Pad(8), Gap(2),
							BackgroundVec(ColPanel), BorderWidth(1),
							BorderColorVec(Alpha(ColBorder, .7))), func() {
							ProcessButtonEvents(false)
							if IsHovered() {
								ModAttrs(BorderColorVec(Alpha(ColAmber, .8)))
							}
							if IsClicked() {
								a.PreviewChord(card.Keys, 600)
							}
							Label(card.Roman, FontSize(12), FontWeight(WeightBold), TextColorVec(ColAmber))
							Label(card.Name, FontSize(14), TextColorVec(ColText))
							Caption(fmt.Sprintf("%d notes", len(card.Keys)))
						})
					}
				})
				HDivider()
				Caption("Click a chord card to hear it. The outline marks the selected key.")
				FlexSpacer()
				Container(Attrs(Row, CrossMid, Gap(8)), func() {
					Caption("Starmap")
					ToggleSwitch(&a.UI.ScaleStarmap)
					if a.UI.ScaleStarmap {
						Caption("(chord-star view is being built)")
					}
				})
			})
		})
	})
}
