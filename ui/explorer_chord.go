package ui

import (
	"fmt"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// ChordExplorer browses and auditions chords in any key.
func ChordExplorer(a *app.App) {
	Modal(980, func() { a.UI.ChordExplorerOpen = false }, func() {
		ModAttrs(Pad(16), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Container(Attrs(Gap(0)), func() {
				Label("Chord Explorer", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
				Caption("Browse any chord in any key; cycle inversions and audition progressions.")
			})
			FlexSpacer()
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.UI.ChordExplorerOpen = false
			}
		})
		HDivider()
		rootSelector(a, false)
		Container(Attrs(Row, Gap(14), ExpandAcross(), FixHeight(330)), func() {
			Container(Attrs(Grow(1), Expand, Viewport, MaxWidth(560)), func() {
				ScrollOnInput()
				ScrollBars()
				Container(Attrs(Row, Wrap, Gap(5), MaxWidth(540)), func() {
					for _, ct := range theory.ChordTypes {
						label := ct.Suffix
						if label == "" {
							label = "maj"
						}
						if a.UI.ExplorerChord == ct.Suffix {
							NextButtonType(ButtonPrimary)
						}
						if CtrlButton(NoIcon, label, true) {
							a.UI.ExplorerChord = ct.Suffix
							a.UI.ExplorerInversion = 0
							a.PreviewChord(chordKeys(a.UI.ExplorerKey, 4+a.UI.ExplorerOctave, ct.Suffix, 0), 600)
						}
					}
				})
			})
			Container(Attrs(FixWidth(320), Expand, BackgroundVec(ColPanel), Corners(10),
				BorderWidth(1), BorderColorVec(Alpha(ColBorder, .7)), Pad(12), Gap(8)), func() {
				Label(currentChordName(a), FontSize(28), FontWeight(WeightBold), TextColorVec(ColAmber))
				Caption("Inversion")
				Container(Attrs(Row, CrossMid, Gap(6)), func() {
					if CtrlButton(TypArrowLeft, "", true) {
						a.UI.ExplorerInversion = (a.UI.ExplorerInversion + 5) % 6
						a.PreviewChord(chordKeys(a.UI.ExplorerKey, 4+a.UI.ExplorerOctave, a.UI.ExplorerChord, a.UI.ExplorerInversion), 600)
					}
					Container(Attrs(MinWidth(30), MainAlign(AlignMiddle)), func() {
						Label(fmt.Sprintf("%d", a.UI.ExplorerInversion+1), Fonts("JetBrains Mono"), FontSize(14), TextColorVec(ColText))
					})
					if CtrlButton(TypArrowRight, "", true) {
						a.UI.ExplorerInversion = (a.UI.ExplorerInversion + 1) % 6
						a.PreviewChord(chordKeys(a.UI.ExplorerKey, 4+a.UI.ExplorerOctave, a.UI.ExplorerChord, a.UI.ExplorerInversion), 600)
					}
					if CtrlButton(TypRefresh, "Root position", true) {
						a.UI.ExplorerInversion = 0
					}
				})
				Caption("Octave")
				Container(Attrs(Row, CrossMid, Gap(6)), func() {
					if CtrlButton(TypMinus, "", a.UI.ExplorerOctave > -1) {
						a.UI.ExplorerOctave--
					}
					Label(fmt.Sprintf("%d", 4+a.UI.ExplorerOctave), Fonts("JetBrains Mono"), FontSize(13), TextColorVec(ColText))
					if CtrlButton(TypPlus, "", a.UI.ExplorerOctave < 1) {
						a.UI.ExplorerOctave++
					}
				})
				HDivider()
				Caption("Progressions")
				Container(Attrs(Row, Wrap, Gap(4)), func() {
					for i, p := range theory.Progressions {
						if a.UI.ExplorerProgression == i {
							NextButtonType(ButtonPrimary)
						}
						if CtrlButton(NoIcon, p.Name, true) {
							a.UI.ExplorerProgression = i
						}
					}
				})
				if Button(TypMediaPlay, "Play progression") {
					scale := theory.ScaleByID("major")
					var chords [][]int
					p := theory.Progressions[a.UI.ExplorerProgression]
					for step := 0; step < len(p.Chords); step++ {
						_, keys := theory.ProgressionChord(a.UI.ExplorerKey, 4+a.UI.ExplorerOctave, scale, p, step)
						chords = append(chords, keys)
					}
					a.PreviewSequence(chords, 620)
				}
				Caption("Click a chord above to audition it.")
			})
		})
	})
}

func rootSelector(a *app.App, scale bool) {
	Container(Attrs(Row, CrossMid, Gap(3), Wrap), func() {
		SectionTitle("KEY")
		for pc := 0; pc < 12; pc++ {
			selected := a.UI.ExplorerKey == pc
			if scale {
				selected = a.UI.ScaleRoot == pc
			}
			if selected {
				NextButtonType(ButtonPrimary)
			}
			if CtrlButton(NoIcon, theory.NoteName(pc, "english", "flat"), true) {
				if scale {
					a.UI.ScaleRoot = pc
				} else {
					a.UI.ExplorerKey = pc
				}
			}
		}
	})
}

func currentChordName(a *app.App) string {
	return theory.FormatChord(a.UI.ExplorerKey, a.UI.ExplorerChord, "flat")
}

func chordKeys(root, octave int, suffix string, inversion int) []int {
	ct, ok := theory.FindChordType(suffix)
	if !ok {
		ct = theory.ChordType{Intervals: []int{0, 4, 7}}
	}
	ivs := ct.Intervals
	if len(ivs) == 0 {
		ivs = []int{0, 4, 7}
	}
	base := (octave+1)*12 + root
	keys := make([]int, len(ivs))
	for i, iv := range ivs {
		keys[i] = base + iv
	}
	for n := 0; n < inversion%len(keys); n++ {
		keys[0] += 12
		for i := 0; i < len(keys)-1; i++ {
			if keys[i] > keys[i+1] {
				keys[i], keys[i+1] = keys[i+1], keys[i]
			}
		}
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
