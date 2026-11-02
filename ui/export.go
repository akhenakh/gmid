package ui

import (
	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/export"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// ExportDialog saves the song as audio.
func ExportDialog(a *app.App) {
	Modal(620, func() { a.UI.ExportOpen = false }, func() {
		ModAttrs(Pad(18), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Container(Attrs(Gap(0)), func() {
				Label("Export to audio", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
				Caption("WAV is CD quality; MP3 is 320 kbps CBR. Files go to the library's exports folder.")
			})
			FlexSpacer()
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.UI.ExportOpen = false
			}
		})
		HDivider()
		SectionTitle("WHAT TO EXPORT")
		Container(Attrs(Row, Gap(6)), func() {
			choices := []string{"As written", "Keys + bass + drums", "As you hear it"}
			for i, label := range choices {
				if a.UI.ExportVariant == i {
					NextButtonType(ButtonPrimary)
				}
				if CtrlButton(NoIcon, label, true) {
					a.UI.ExportVariant = i
				}
			}
		})
		SectionTitle("FORMAT")
		Container(Attrs(Row, Gap(6)), func() {
			if a.UI.ExportFormat == "wav" {
				NextButtonType(ButtonPrimary)
			}
			if CtrlButton(NoIcon, "WAV (CD quality)", true) {
				a.UI.ExportFormat = "wav"
			}
			if a.UI.ExportFormat == "mp3" {
				NextButtonType(ButtonPrimary)
			}
			if CtrlButton(NoIcon, "MP3 (320 kbps)", true) {
				a.UI.ExportFormat = "mp3"
			}
		})
		HDivider()
		if a.ExportState.Running {
			ProgressBar(float32(a.ExportState.Progress))
			Caption(a.ExportState.Message)
		} else {
			Container(Attrs(Row, Gap(8)), func() {
				if Button(TypExport, "Export") {
					variant := []export.Variant{export.Original, export.Focus, export.Heard}[a.UI.ExportVariant%3]
					a.ExportAudio(variant, a.UI.ExportFormat)
				}
				FlexSpacer()
				Caption("Master volume and mixer apply to “As you hear it”.")
			})
			if a.ExportState.Message != "" {
				Caption(a.ExportState.Message)
			}
		}
	})
}
