package ui

import (
	"fmt"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// PlaybackEditorModal edits tracks without touching the original file.
func PlaybackEditorModal(a *app.App) {
	Modal(900, func() { a.UI.PlaybackEditorOpen = false }, func() {
		ModAttrs(Pad(16), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Container(Attrs(Gap(0)), func() {
				Label("MIDI Playback Editor", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
				Caption("Exclude, rename, recolour, reassign, merge or split tracks. Saves a new version.")
			})
			FlexSpacer()
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.UI.PlaybackEditorOpen = false
			}
		})
		HDivider()
		Container(Attrs(Viewport, FixHeight(430)), func() {
			ScrollOnInput()
			ScrollBars()
			if a.Song == nil {
				Caption("No song loaded")
				return
			}
			for i := range a.Song.Tracks {
				playbackEditorRow(a, i)
			}
		})
		HDivider()
		Container(Attrs(Row, Gap(8), ExpandAcross(), CrossMid), func() {
			Caption(fmt.Sprintf("%d tracks · %d notes", len(a.Song.Tracks), a.Song.NoteCount))
			FlexSpacer()
			if Button(TypArchive, "Save as new version") {
				path, err := a.SaveVersion("playback-editor", "Edited tracks in the Playback Editor")
				if err == nil && path != "" {
					a.UI.PlaybackEditorOpen = false
					if err := a.LoadFile(path); err != nil {
						a.Notify(err.Error())
					}
				}
			}
			if Button(NoIcon, "Done") {
				a.UI.PlaybackEditorOpen = false
			}
		})
	})
}

func playbackEditorRow(a *app.App, i int) {
	t := &a.Song.Tracks[i]
	ContainerWithKey(t, Attrs(Row, CrossMid, ExpandAcross(), Gap(8), Pad2(6, 4),
		Corners(6), BackgroundVec(ColPanel), BorderWidth(1),
		BorderColorVec(Alpha(TrackColor(t.Color), .4))), func() {
		Badge(TrackColor(t.Color), 10)
		Container(Attrs(FixWidth(210), Gap(0)), func() {
			TextInput(&t.Name)
			Caption(fmt.Sprintf("%s · ch %d", t.GMName, t.Channel+1))
		})
		Container(Attrs(Row, CrossMid, Gap(3)), func() {
			if CtrlButton(TypMinus, "", t.Program > 0) {
				t.Program--
				t.GMName = theory.GMName(t.Program)
				t.Group = theory.GMGroupOf(t.Program, t.IsDrum)
			}
			Container(Attrs(FixWidth(150), MainAlign(AlignMiddle)), func() {
				Label(trimText(t.GMName, 22), FontSize(11), TextColorVec(ColTextDim))
			})
			if CtrlButton(TypPlus, "", t.Program < 127) {
				t.Program++
				t.GMName = theory.GMName(t.Program)
				t.Group = theory.GMGroupOf(t.Program, t.IsDrum)
			}
		})
		if CtrlButton(TypPipette, "Colour", true) {
			for j, c := range theory.TrackPalette {
				if c == t.Color {
					t.Color = theory.TrackPalette[(j+1)%len(theory.TrackPalette)]
					break
				}
			}
		}
		Caption("Exclude")
		ToggleSwitch(&t.Excluded)
		if i > 0 {
			if CtrlButton(TypFlowMerge, "Merge into previous", true) {
				a.MergeTrack(i)
			}
		}
		if theory.IsKeyboardGroup(t.Group) {
			if CtrlButton(TypScissorsOutline, "Split hands", true) {
				a.SplitHands(i)
			}
		}
	})
}
