package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/akhenakh/gmid/internal/app"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

const (
	topBarHeight    = 54
	sidePanelWidth  = 276
	trackPanelWidth = 268
	keyboardHeight  = 150
)

// Root builds the whole application surface.
func Root(a *app.App) {
	ModAttrs(UseSurface(SurfaceCanvas),
		AmendTextStyle(Fonts("Inter"), FontSize(13)))

	if a.UI.Presentation {
		Presentation(a)
		return
	}

	TopBar(a)
	Container(Attrs(Row, Grow(1), Expand, Gap(0)), func() {
		if a.UI.SideOpen {
			LeftPanel(a)
		}
		MainColumn(a)
		if a.UI.PanelsOpen && a.Song != nil {
			TrackPanel(a)
		}
	})

	Overlays(a)
	HandleShortcuts(a)
}

func TopBar(a *app.App) {
	Container(Attrs(Row, CrossMid, FixHeight(topBarHeight), ExpandAcross(),
		Pad2(0, 12), Gap(10), UseSurface(SurfaceToolbar),
		BorderWidth(0)), func() {

		// Logo
		Container(Attrs(FixSize(28, 28), Corners(8), BackgroundVec(ColAmber),
			MainAlign(AlignMiddle), CrossAlign(AlignMiddle)), func() {
			Label("g", FontSize(17), FontWeight(WeightBold), TextColorVec(Col("#181005")))
		})

		Container(Attrs(Gap(0), MainAlign(AlignMiddle)), func() {
			title := "gmid"
			sub := "No song loaded"
			if a.Song != nil {
				title = a.Song.Name
				sub = fmt.Sprintf("%d tracks · %d notes · %s", len(a.Song.Tracks), a.Song.NoteCount, formatDuration(a.Song.DurationSec))
			}
			Label(title, FontSize(13), FontWeight(WeightBold), TextColorVec(ColText))
			if time.Now().Before(a.UI.ToastTill) {
				Label(a.UI.Toast, FontSize(11), TextColorVec(ColAmber))
			} else {
				Caption(sub)
			}
		})

		FlexSpacer()

		// Transport cluster.
		Container(Attrs(Row, CrossMid, Gap(4), Pad2(3, 6), Corners(9),
			BackgroundVec(Alpha(ColPanel, .85)), BorderWidth(1),
			BorderColorVec(Alpha(ColBorder, .7))), func() {
			if IconButton(TypMediaRewind, "Back to start", a.Song != nil) {
				a.Seek(0)
			}
			playIcon := TypMediaPlay
			tip := "Play (Space)"
			if a.Transport.Playing {
				playIcon = TypMediaPause
				tip = "Pause (Space)"
			}
			if PrimaryIconButton(playIcon, tip, a.Song != nil) {
				a.TogglePlay()
			}
			if IconButton(TypMediaStop, "Stop (Escape)", a.Song != nil) {
				a.Stop()
			}
			Label(formatClock(a.Transport.Time), Fonts("JetBrains Mono"), FontSize(12),
				TextColorVec(ColText))
		})

		FlexSpacer()

		bpm := 120.0
		if a.Song != nil {
			bpm = a.Song.BPMAt(a.Transport.Time) * a.Transport.Speed
		}
		ValuePill("BPM", fmt.Sprintf("%.0f", bpm), false)
		ValuePill("KEY", keyLabel(a), true)

		if IconButton(TypArrowRepeat, "Loop region", a.Song != nil) {
			a.ToggleLoop()
		}
		if IconButton(TypStopwatch, "Metronome (right-click for count-in)", true) {
			a.ToggleMetronome()
		}
		if IconButton(TypCog, "Settings (Ctrl+,)", true) {
			a.UI.SettingsOpen = !a.UI.SettingsOpen
		}
		if IconButton(TypArrowMaximise, "Presentation mode (F11)", true) {
			a.UI.Presentation = true
		}
	})
}

func MainColumn(a *app.App) {
	Container(Attrs(Grow(1), Expand, Pad(10), Gap(8)), func() {
		if os.Getenv("GMID_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "main w=%.1f h=%.1f\n", GetContentWidth(), GetContentHeight())
		}
		RollHeader(a)
		PianoRoll(a)
		if a.Prefs.LoopRegion {
			LoopStrip(a)
		}
		KeyboardBar(a)
		bed := Container(Attrs(ExpandAcross(), Corners(8), Clip,
			BackgroundVec(Col("#08080b")), BorderWidth(1),
			BorderColorVec(Alpha(ColBorder, .6))), func() {
			min, max := KeyRange(a.Prefs.KeyRange)
			PianoKeyboard(min, max, keyboardHeight, soundingKeys(a),
				a.Prefs.ShowNoteNames, a.Prefs.ShowOctaves, nil, nil)
		})
		_ = bed
	})
}

func RollHeader(a *app.App) {
	Container(Attrs(Row, CrossMid, Gap(8), ExpandAcross()), func() {
		if a.Song == nil {
			Caption("Open a file to begin")
		} else {
			Caption(fmt.Sprintf("Bar %d · %s / %d", a.Song.BarNumberAt(a.Transport.Time),
				fmt.Sprintf("%.0f", a.Song.BPMAt(a.Transport.Time)), a.Song.PPQ))
			if a.Transport.Speed != 1 {
				Badge(ColAmber, 6)
				Caption(fmt.Sprintf("%.0f%% speed", a.Transport.Speed*100))
			}
			if a.Transport.Transpose != 0 {
				Badge(ColBlue, 6)
				Caption(fmt.Sprintf("%+d st", a.Transport.Transpose))
			}
		}
		FlexSpacer()
		if a.Prefs.TapTempo && a.Song != nil {
			if CtrlButton(TypTime, "TAP", true) {
				a.ToggleMetronome()
			}
		}
		if IconButton(TypFolderOpen, "Library", true) {
			a.UI.LibraryOpen = !a.UI.LibraryOpen
		}
		if IconButton(TypThLarge, "Console Mixer (Ctrl+Shift+M)", a.Song != nil) {
			a.UI.MixerOpen = !a.UI.MixerOpen
		}
	})
}

func KeyboardBar(a *app.App) {
	Container(Attrs(Row, CrossMid, FixHeight(28), ExpandAcross(), Pad2(0, 4), Gap(8)), func() {
		if a.Prefs.StepThrough && a.Song != nil {
			if CtrlButton(TypArrowLeft, "Step back (←)", true) {
				a.StepThrough(-1, false)
			}
			if CtrlButton(TypArrowRight, "Step forward (→)", true) {
				a.StepThrough(1, false)
			}
		}
		FlexSpacer()
		Label(chordNameAt(a), FontSize(15), FontWeight(WeightBold), TextColorVec(ColText))
		FlexSpacer()
		if a.Song != nil {
			Caption(fmt.Sprintf("%d keys", a.Prefs.KeyRange))
		}
	})
}

func LeftPanel(a *app.App) {
	Container(Attrs(FixWidth(sidePanelWidth), Expand, UseSurface(SurfacePanel),
		Pad(12), Gap(10)), func() {
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			TitleText("Setup")
			FlexSpacer()
			if IconButton(TypTimes, "Close panel", true) {
				a.UI.SideOpen = false
			}
		})
		Divider()
		Container(Attrs(Viewport), func() {
			ScrollOnInput()
			ScrollBars()

			SectionTitle("Sound")
			SettingRow("Engine", "GM is instant; Studio and Native use soundfonts.", func() {
				CompactSegmented(&a.Prefs.SoundEngine, func() {
					SegmentedCell("GM", "gm")
					SegmentedCell("Studio", "studio")
					SegmentedCell("Native", "native")
				})
			})
			SettingRow("Master volume", "Persists across sessions.", func() {
				v := float32(a.Prefs.MasterVolume)
				Slider(&v, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 110})
				if float64(v) != a.Prefs.MasterVolume {
					a.Prefs.MasterVolume = float64(v)
					a.SavePrefs()
				}
			})

			SectionTitle("Keyboard")
			SettingRow("Key range", "61, 73 or 88 keys.", func() {
				CompactSegmented(&a.Prefs.KeyRange, func() {
					SegmentedCell("61", 61)
					SegmentedCell("73", 73)
					SegmentedCell("88", 88)
				})
			})
			SettingRow("Note names", "Show names on the keys.", func() {
				ToggleSwitch(&a.Prefs.ShowNoteNames)
				a.SavePrefs()
			})
			SettingRow("Octaves", "Show C3, C4 … on the keys.", func() {
				ToggleSwitch(&a.Prefs.ShowOctaves)
				a.SavePrefs()
			})

			SectionTitle("Practice")
			SettingRow("Focus mode", "Hear keys, bass and drums; see only the keys.", func() {
				ToggleSwitch(&a.Prefs.FocusMode)
				a.SavePrefs()
			})
			SettingRow("Step Through", "Walk the song with ← and →.", func() {
				ToggleSwitch(&a.Prefs.StepThrough)
				a.SavePrefs()
			})
			SettingRow("Tap Tempo", "Add a TAP pad to the roll header.", func() {
				ToggleSwitch(&a.Prefs.TapTempo)
				a.SavePrefs()
			})
			SettingRow("Tooltips", "Hints on hover.", func() {
				ToggleSwitch(&a.Prefs.Tooltips)
				a.SavePrefs()
			})

			HDivider()
			if Button(TypCog, "All settings…") {
				a.UI.SettingsOpen = true
			}
			if Button(TypFolderOpen, "Open file…") {
				a.UI.LibraryOpen = true
			}
			FlexSpacer()
			if a.Status != "" {
				Caption(a.Status)
			}
		})
	})
}

// SettingRow places a control to the right of a title and one-line hint.
func SettingRow(title, hint string, control func()) {
	Container(Attrs(Row, CrossMid, ExpandAcross(), Gap(10), Pad2(6, 2)), func() {
		Container(Attrs(Grow(1), Gap(0)), func() {
			Label(title, FontSize(12), TextColorVec(ColText))
			Caption(hint)
		})
		Container(Attrs(Row, CrossMid, Gap(4)), control)
	})
}

func TrackPanel(a *app.App) {
	Container(Attrs(FixWidth(trackPanelWidth), Expand, UseSurface(SurfacePanel),
		Pad(12), Gap(8)), func() {
		PanelHeader("Tracks", func() {
			if IconButton(TypCog, "Track settings", true) {
				a.UI.SettingsOpen = true
			}
		})
		Caption("Click a name to select; icons toggle sound and visibility.")
		Divider()
		Container(Attrs(Viewport), func() {
			ScrollOnInput()
			ScrollBars()
			if a.Song == nil {
				Caption("No tracks")
				return
			}
			for i := range a.Song.Tracks {
				trackRow(a, i)
			}
		})
	})
}

func trackRow(a *app.App, i int) {
	t := &a.Song.Tracks[i]
	ContainerWithKey(t, Attrs(Row, CrossMid, ExpandAcross(), Gap(6),
		Pad2(5, 4), Corners(6),
		BackgroundVec(func() Vec4 {
			if t.Muted || t.Excluded {
				return Alpha(ColPanel2, .5)
			}
			return ColPanel
		}())), func() {
		Badge(TrackColor(t.Color), 9)
		Container(Attrs(Grow(1), Gap(0)), func() {
			Label(t.Name, FontSize(11), TextColorVec(func() Vec4 {
				if t.Muted || t.Excluded {
					return ColTextMute
				}
				return ColText
			}()))
			Caption(fmt.Sprintf("%s · ch %d", t.GMName, t.Channel+1))
		})
		if IconButton(iconBool(t.Muted, TypVolumeMute, TypVolume), "Mute", true) {
			t.Muted = !t.Muted
		}
		if IconButton(iconBool(t.Solo, TypStar, TypStarOutline), "Solo", true) {
			t.Solo = !t.Solo
		}
		if IconButton(iconBool(t.Visible, TypEye, TypEyeOutline), "Show on roll", true) {
			t.Visible = !t.Visible
		}
	})
}

func iconBool(on bool, yes, no IconGlyph) IconGlyph {
	if on {
		return yes
	}
	return no
}

func Overlays(a *app.App) {
	if a.UI.SettingsOpen {
		SettingsWindow(a)
	}
	if a.UI.LibraryOpen {
		LibraryWindow(a)
	}
	if a.UI.MixerOpen && a.Song != nil {
		MixerWindow(a)
	}
	if a.UI.Confirm != nil {
		ConfirmDialog(a)
	}
}

func Presentation(a *app.App) {
	Container(Attrs(Expand, BackgroundVec(Col("#000000")), Pad(0)), func() {
		if IconButton(TypArrowMinimise, "Leave presentation (F11)", true) {
			a.UI.Presentation = false
		}
		PianoRoll(a)
		min, max := KeyRange(a.Prefs.KeyRange)
		PianoKeyboard(min, max, 190, soundingKeys(a), a.Prefs.ShowNoteNames, a.Prefs.ShowOctaves, nil, nil)
	})
}

func keyLabel(a *app.App) string {
	if a.Song == nil {
		return "—"
	}
	sem, minor, ok := a.Song.KeyAt(a.Transport.Time)
	if !ok {
		return "—"
	}
	name := NoteName(sem, "english", "flat")
	if minor {
		return name + "m"
	}
	return name
}

func formatDuration(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	m := int(sec) / 60
	s := int(sec) % 60
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatClock(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	m := int(sec) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%02d:%02d.%02d", m, s, cs)
}
