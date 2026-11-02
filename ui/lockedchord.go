package ui

import (
	"strings"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// LockedChordModal shows the shift-clicked chord and cycles its inversions.
func LockedChordModal(a *app.App) {
	keys := a.LockedChord()
	Modal(460, func() { a.UI.LockedChordOpen = false }, func() {
		ModAttrs(Pad(18), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Label("Locked chord", FontSize(18), FontWeight(WeightBold), TextColorVec(ColText))
			FlexSpacer()
			if IconButton(TypTimes, "Close", true) {
				a.UI.LockedChordOpen = false
			}
		})
		name := theory.DetectChord(keys)
		Label(name, FontSize(30), FontWeight(WeightBold), TextColorVec(ColAmber))
		Caption("Keys: " + keyList(keys, a.Prefs.NoteNaming, a.Prefs.Accidentals))
		Container(Attrs(Row, Gap(6)), func() {
			if CtrlButton(TypArrowLeft, "Inversion down", true) {
				a.UI.ExplorerInversion = (a.UI.ExplorerInversion + 5) % 6
			}
			Label("Inversion "+itoa(a.UI.ExplorerInversion+1), FontSize(12), TextColorVec(ColText))
			if CtrlButton(TypArrowRight, "Inversion up", true) {
				a.UI.ExplorerInversion = (a.UI.ExplorerInversion + 1) % 6
			}
		})
		Container(Attrs(Row, Gap(8)), func() {
			if Button(TypMediaPlay, "Play") {
				a.PreviewChord(invertKeys(keys, a.UI.ExplorerInversion), 700)
			}
			if Button(TypRefresh, "Clear lock") {
				a.LockedKeys = map[int]bool{}
				a.UI.LockedChordOpen = false
			}
			FlexSpacer()
			if Button(NoIcon, "Close") {
				a.UI.LockedChordOpen = false
			}
		})
		Caption("Shift+click three or more keys to build another chord.")
	})
}

func keyList(keys []int, naming, accidentals string) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = NoteName(k, naming, accidentals)
	}
	return strings.Join(parts, " · ")
}

func invertKeys(keys []int, inv int) []int {
	out := append([]int(nil), keys...)
	if len(out) == 0 {
		return out
	}
	for n := 0; n < inv%len(out); n++ {
		out[0] += 12
		for i := 0; i < len(out)-1; i++ {
			if out[i] > out[i+1] {
				out[i], out[i+1] = out[i+1], out[i]
			}
		}
	}
	return out
}
