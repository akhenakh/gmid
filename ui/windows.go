package ui

import (
	"fmt"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/config"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// SettingsWindow is the full settings surface. It grows group by group; the
// Setup panel and this window read the same preferences.
func SettingsWindow(a *app.App) {
	Modal(760, func() { a.UI.SettingsOpen = false }, func() {
		ModAttrs(Pad(18), Gap(12))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Label("Settings", FontSize(22), FontWeight(WeightBold), TextColorVec(ColText))
			FlexSpacer()
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.UI.SettingsOpen = false
			}
		})
		Caption("Every setting is described in one line. Ctrl+, toggles this window.")
		HDivider()
		Container(Attrs(Viewport, Grow(1)), func() {
			ScrollOnInput()
			ScrollBars()

			SectionTitle("Audio")
			SettingRow("Sound engine", "GM is instant; Studio and Native use soundfonts.", func() {
				CompactSegmented(&a.Prefs.SoundEngine, func() {
					SegmentedCell("GM", "gm")
					SegmentedCell("Studio", "studio")
					SegmentedCell("Native", "native")
				})
			})
			SettingRow("Native output", "Exclusive is shortest; Shared lets other apps play.", func() {
				v := a.Prefs.ShareMode == "exclusive"
				ToggleSwitch(&v)
				if v {
					a.Prefs.ShareMode = "exclusive"
				} else {
					a.Prefs.ShareMode = "shared"
				}
				a.SavePrefs()
			})
			SettingRow("Master volume", "Persists across sessions.", func() {
				v := float32(a.Prefs.MasterVolume)
				Slider(&v, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 130})
				if float64(v) != a.Prefs.MasterVolume {
					a.Prefs.MasterVolume = float64(v)
					a.SavePrefs()
				}
			})
			SettingRow("Auto-level", "Balance loud and quiet passages on load.", func() {
				ToggleSwitch(&a.Prefs.AutoLevel)
				a.SavePrefs()
			})

			SectionTitle("Playback")
			SettingRow("Metronome", "Follows the file's tempo map.", func() {
				ToggleSwitch(&a.Prefs.Metronome)
				a.SavePrefs()
			})
			SettingRow("Count-in", "1–4 bars before the music starts.", func() {
				ToggleSwitch(&a.Prefs.CountIn)
				a.SavePrefs()
			})
			SettingRow("Count-in bars", "Bars of clicks.", func() {
				n := a.Prefs.CountInBars
				if CtrlButton(TypMinus, "", true) && n > 1 {
					a.Prefs.CountInBars--
					a.SavePrefs()
				}
				Label(fmt.Sprintf("%d", n), Fonts("JetBrains Mono"), FontSize(12), TextColorVec(ColText))
				if CtrlButton(TypPlus, "", true) && n < 4 {
					a.Prefs.CountInBars++
					a.SavePrefs()
				}
			})
			SettingRow("Skip empty bars", "Start where the music starts; the file is untouched.", func() {
				ToggleSwitch(&a.Prefs.SkipEmptyBars)
				a.SavePrefs()
			})
			SettingRow("Tap Tempo", "Show the TAP pad for fixing the beat grid.", func() {
				ToggleSwitch(&a.Prefs.TapTempo)
				a.SavePrefs()
			})
			SettingRow("Tempo warnings", "Mark songs whose bar lines miss the beat.", func() {
				ToggleSwitch(&a.Prefs.TempoWarnings)
				a.SavePrefs()
			})

			SectionTitle("Keyboard & Roll")
			SettingRow("Key range", "61, 73 or 88 keys.", func() {
				CompactSegmented(&a.Prefs.KeyRange, func() {
					SegmentedCell("61", 61)
					SegmentedCell("73", 73)
					SegmentedCell("88", 88)
				})
			})
			SettingRow("Note names", "Names on the keyboard keys.", func() {
				ToggleSwitch(&a.Prefs.ShowNoteNames)
				a.SavePrefs()
			})
			SettingRow("Octaves", "Octave numbers on C keys.", func() {
				ToggleSwitch(&a.Prefs.ShowOctaves)
				a.SavePrefs()
			})
			SettingRow("Note naming", "English, Central European or solfège.", func() {
				CompactSegmented(&a.Prefs.NoteNaming, func() {
					SegmentedCell("C D E", "english")
					SegmentedCell("C D H", "central-european")
					SegmentedCell("Do Re", "solfege")
				})
			})
			SettingRow("Accidentals", "Spell black keys as sharps or flats.", func() {
				v := a.Prefs.Accidentals == "sharp"
				ToggleSwitch(&v)
				if v {
					a.Prefs.Accidentals = "sharp"
				} else {
					a.Prefs.Accidentals = "flat"
				}
				a.SavePrefs()
			})
			SettingRow("Roll zoom", "How many seconds are visible at once.", func() {
				v := float32(a.Prefs.RollZoom)
				Slider(&v, SliderAttrs{Min: 0.5, Max: 3, Step: 0.05, Width: 130})
				if float64(v) != a.Prefs.RollZoom {
					a.Prefs.RollZoom = float64(v)
					a.SavePrefs()
				}
			})
			SettingRow("Bar numbers", "Number every bar on the roll.", func() {
				ToggleSwitch(&a.Prefs.BarNumbers)
				a.SavePrefs()
			})
			SettingRow("Visual effects", "Particles where notes hit the keys.", func() {
				ToggleSwitch(&a.Prefs.VisualEffects)
				a.SavePrefs()
			})

			SectionTitle("Practice & Editing")
			SettingRow("Hand assignment", "Colour piano notes by the hand that plays them.", func() {
				ToggleSwitch(&a.Prefs.HandAssignment)
				a.SavePrefs()
			})
			SettingRow("Hand badges", "Small L/R letters on hand-coloured keys.", func() {
				ToggleSwitch(&a.Prefs.HandTags)
				a.SavePrefs()
			})
			SettingRow("Note editor", "Edit notes on the roll.", func() {
				ToggleSwitch(&a.Prefs.NoteEditor)
				a.SavePrefs()
			})
			SettingRow("Step Through", "Walk the song one note or chord at a time.", func() {
				ToggleSwitch(&a.Prefs.StepThrough)
				a.SavePrefs()
			})
			SettingRow("Chord Prompter", "Past, current and upcoming chords.", func() {
				ToggleSwitch(&a.Prefs.ChordPrompter)
				a.SavePrefs()
			})
			SettingRow("Focus mode", "Hear keys, bass, drums; see only the keys.", func() {
				ToggleSwitch(&a.Prefs.FocusMode)
				a.SavePrefs()
			})

			SectionTitle("Appearance")
			SettingRow("Warm theme", "Cream and green instead of cool charcoal.", func() {
				v := a.Prefs.Theme == "warm"
				ToggleSwitch(&v)
				if v {
					a.Prefs.Theme = "warm"
				} else {
					a.Prefs.Theme = "dark"
				}
				a.SavePrefs()
				Install(a.Prefs.Theme == "warm")
				RequestNextFrame()
			})
			SettingRow("Tooltips", "Show hints on hover.", func() {
				ToggleSwitch(&a.Prefs.Tooltips)
				a.SavePrefs()
			})
			SettingRow("Library folder", "Where songs, versions and exports live.", func() {
				if CtrlButton(TypFolderOpen, "Choose…", true) {
					a.UI.LibraryOpen = true
				}
			})
			Caption("Library: " + a.Prefs.LibraryFolder)

			HDivider()
			Container(Attrs(Row, Gap(8)), func() {
				if Button(TypRefresh, "Reload song") && a.Song != nil {
					if err := a.LoadFile(a.Song.Path); err != nil {
						a.Notify(err.Error())
					}
				}
				FlexSpacer()
				if Button(TypTimes, "Close") {
					a.UI.SettingsOpen = false
				}
			})
		})
	})
}

// LibraryWindow browses a folder and opens a song.
func LibraryWindow(a *app.App) {
	Modal(780, func() { a.UI.LibraryOpen = false }, func() {
		ModAttrs(Pad(18), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Label("Open a song", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
			FlexSpacer()
			if IconButton(TypTimes, "Close", true) {
				a.UI.LibraryOpen = false
			}
		})
		if a.UI.LibraryPath == "" {
			a.UI.LibraryPath = a.Prefs.LibraryFolder
			if a.UI.LibraryPath == "" {
				a.UI.LibraryPath = config.DefaultLibrary()
			}
		}
		a.UI.LibrarySelected = -1
		attrs := FileBrowserAttrs{
			Title: "Choose a MIDI or score file",
			Width: 720,
			Files: true,
			Dirs:  true,
			Exts:  []string{"mid", "midi", "kar", "musicxml", "xml", "mxl", "gp", "gp3", "gp4", "gp5", "gpx", "cap", "capx"},
		}
		if FileBrowserPanel(&a.UI.LibraryPath, &a.UI.LibraryFilter, &a.UI.LibrarySelected,
			&a.UI.LibrarySelection, attrs) {
			if a.UI.LibrarySelection != "" {
				path := a.UI.LibrarySelection
				a.UI.LibraryOpen = false
				if err := a.LoadFile(path); err != nil {
					a.Notify(err.Error())
				}
			}
		}
		HDivider()
		SectionTitle("DEMO SONGS")
		Container(Attrs(Row, Gap(8), Wrap), func() {
			for _, path := range a.DemoFiles() {
				if Button(TypMediaPlay, trim(path, 30)) {
					a.UI.LibraryOpen = false
					if err := a.LoadFile(path); err != nil {
						a.Notify(err.Error())
					}
				}
			}
		})
	})
}

// MixerWindow is the console mixer. The strips control track audibility now;
// per-channel volume/pan/chorus/reverb are wired in the next milestone.
func MixerWindow(a *app.App) {
	Modal(900, func() { a.UI.MixerOpen = false }, func() {
		ModAttrs(Pad(16), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Label("Console Mixer", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
			FlexSpacer()
			Caption(fmt.Sprintf("Master %.0f%%", a.Prefs.MasterVolume*100))
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.UI.MixerOpen = false
			}
		})
		Container(Attrs(Row, Gap(8), ExpandAcross(), FixHeight(320)), func() {
			for i := range a.Song.Tracks {
				t := &a.Song.Tracks[i]
				Container(Attrs(FixWidth(104), Expand, Corners(8), Pad(8), Gap(6),
					BackgroundVec(ColPanel), BorderWidth(1),
					BorderColorVec(Alpha(TrackColor(t.Color), .5))), func() {
					Label(trimText(t.Name, 12), FontSize(11), FontWeight(WeightBold), TextColorVec(TrackColor(t.Color)))
					Caption(fmt.Sprintf("ch %d", t.Channel+1))
					v := float32(1)
					Slider(&v, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 86})
					Container(Attrs(Row, Gap(4)), func() {
						if CtrlButton(iconBool(t.Muted, TypVolumeMute, TypVolume), "", true) {
							t.Muted = !t.Muted
						}
						if CtrlButton(iconBool(t.Solo, TypStar, TypStarOutline), "", true) {
							t.Solo = !t.Solo
						}
					})
					Element(Attrs(ExpandAcross(), FixHeight(40), Corners(4),
						BackgroundVec(Alpha(TrackColor(t.Color), .22))))
				})
			}
		})
		a.UI.MixerOpen = true
	})
}

// ConfirmDialog is the shared destructive-action prompt.
func ConfirmDialog(a *app.App) {
	c := a.UI.Confirm
	Modal(440, func() { a.UI.Confirm = nil }, func() {
		ModAttrs(Pad(18), Gap(12))
		Label(c.Title, FontSize(18), FontWeight(WeightBold), TextColorVec(ColText))
		BodyText(c.Message)
		Container(Attrs(Row, Gap(8), ExpandAcross(), MainAlign(AlignEnd)), func() {
			if Button(NoIcon, "Cancel") {
				a.UI.Confirm = nil
			}
			NextButtonType(ButtonDestructive)
			if Button(NoIcon, "Confirm") {
				action := c.Action
				a.UI.Confirm = nil
				if action != nil {
					action()
				}
			}
		})
	})
}

func trimText(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
