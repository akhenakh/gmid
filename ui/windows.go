package ui

import (
	"fmt"
	"strings"

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
		Container(Attrs(Row, CrossMid, Gap(8), ExpandAcross()), func() {
			Container(Attrs(FixWidth(440), Gap(2)), func() {
				NextAccessLabel("Search settings")
				TextInput(&a.UI.SettingsQuery)
				Caption("Search names; click a match to jump to its group")
			})
		})
		if results := searchSettingNames(a.UI.SettingsQuery); len(results) > 0 {
			Container(Attrs(Row, Wrap, Gap(3), ExpandAcross()), func() {
				for _, r := range results {
					if CtrlButton(NoIcon, r.name, true) {
						a.UI.SettingsGroup = r.group
					}
				}
			})
		}
		Container(Attrs(Row, Gap(3), Wrap, ExpandAcross()), func() {
			type tab struct{ id, label string }
			tabs := []tab{
				{"audio", "Audio"}, {"midi", "MIDI Keyboard"}, {"library", "Library"}, {"playback", "Playback"},
				{"practice", "Practice"}, {"editing", "Editing"}, {"notation", "Notation"},
				{"keyboard", "Keyboard"}, {"appearance", "Appearance"},
			}
			for _, t := range tabs {
				if a.UI.SettingsGroup == t.id {
					NextButtonType(ButtonPrimary)
				}
				if CtrlButton(NoIcon, t.label, true) {
					a.UI.SettingsGroup = t.id
				}
			}
		})
		Container(Attrs(Viewport, FixHeight(460)), func() {
			ScrollOnInput()
			ScrollBars()

			if a.UI.SettingsGroup == "audio" {
				SectionTitle("Audio")
				SettingRow("Sound engine", "GM is instant; Studio and Native use soundfonts.", func() {
					if CompactSegmented(&a.Prefs.SoundEngine, func() {
						SegmentedCell("GM", "gm")
						SegmentedCell("Studio", "studio")
						SegmentedCell("Native", "native")
					}) {
						a.SetSoundEngine(a.Prefs.SoundEngine)
					}
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
						if a.Engine != nil {
							a.Engine.SetMasterVolume(a.Prefs.MasterVolume)
						}
					}
				})
				SettingRow("Auto-level", "Balance loud and quiet passages on load.", func() {
					ToggleSwitch(&a.Prefs.AutoLevel)
					a.SavePrefs()
				})
				Container(Attrs(Gap(3), Pad2(6, 2)), func() {
					Label("DecentSampler instrument", FontSize(12), TextColorVec(ColText))
					Caption("A .dspreset or .dslibrary replaces the melodic synth; drums keep the soundfont.")
					Container(Attrs(FixWidth(470)), func() {
						TextInput(&a.Prefs.DecentPreset)
					})
					Container(Attrs(Row, Gap(6)), func() {
						if Button(TypFolderOpen, "Load") {
							if err := a.LoadDecentPreset(a.Prefs.DecentPreset); err != nil {
								a.Notify("DecentSampler: " + err.Error())
							}
						}
						if CtrlButton(NoIcon, "Demo piano", true) {
							_ = a.LoadDecentPreset(a.Assets + "/decent/demo-piano.dspreset")
						}
						if a.Prefs.DecentPreset != "" {
							if CtrlButton(TypTimes, "Unload", true) {
								a.UnloadDecentPreset()
							}
						}
					})
					if info, ok := a.DecentStatus(); ok {
						Caption(fmt.Sprintf("%s · %d groups · %d zones · %d buffers",
							info.Name, info.Groups, info.Zones, info.Buffers))
						if len(info.Errors) > 0 {
							Caption(fmt.Sprintf("%d sample errors", len(info.Errors)))
						}
					} else {
						Caption("No DecentSampler instrument loaded")
					}
				})

			}

			if a.UI.SettingsGroup == "midi" {
				SectionTitle("MIDI Keyboard")
				devices := a.MIDIDevices()
				SettingRow("Devices", "Every input port is opened; play on any of them.", func() {
					Label(fmt.Sprintf("%d", len(devices)), Fonts("JetBrains Mono"), FontSize(12), TextColorVec(ColText))
				})
				if len(devices) == 0 {
					Caption("No MIDI input detected. Connect a keyboard; ports are rescanned every 2 s.")
				} else {
					for _, name := range devices {
						Caption("• " + name)
					}
				}
				SettingRow("Touch response", "Velocity curve from soft to hard.", func() {
					v := float32(a.Prefs.TouchResponse)
					Slider(&v, SliderAttrs{Min: 1, Max: 100, Step: 1, Width: 130})
					if int(v) != a.Prefs.TouchResponse {
						a.Prefs.TouchResponse = int(v)
						a.SavePrefs()
					}
				})
				SettingRow("Pedals & controls", "Sustain, expression, modulation and bend.", func() {
					ToggleSwitch(&a.Prefs.PedalsControls)
					a.SavePrefs()
				})
				SettingRow("Double-note filter", "Ignore an identical note already held.", func() {
					ToggleSwitch(&a.Prefs.DuplicateFilter)
					a.SavePrefs()
				})
				SettingRow("Live response", "Balanced is lower latency; Stable is gentler.", func() {
					CompactSegmented(&a.Prefs.LiveResponse, func() {
						SegmentedCell("Stable", "stable")
						SegmentedCell("Balanced", "balanced")
					})
				})
				SettingRow("Delay check", "Output buffer adds to the key-to-speaker time.", func() {
					ms := 0.0
					if a.Engine != nil {
						ms = a.Engine.OutputDelay() * 1000
					}
					Label(fmt.Sprintf("%.1f ms", ms), Fonts("JetBrains Mono"), FontSize(12), TextColorVec(ColText))
				})
				Caption("Key → speaker: touch response and app handling are reported in the status log; the output buffer above is measured from the device.")
			}

			if a.UI.SettingsGroup == "library" {
				SectionTitle("MIDI Files & Library")
				SettingRow("Library folder", "Where songs, versions and exports live.", func() {
					if CtrlButton(TypFolderOpen, "Choose…", true) {
						a.UI.FileBrowserOpen = true
					}
				})
				Caption(a.Prefs.LibraryFolder)
				SettingRow("Show demo content", "The bundled demo songs in the Library.", func() {
					ToggleSwitch(&a.Prefs.ShowDemo)
					a.SavePrefs()
				})
				SettingRow("Chord transcription", "Show the chord-chart action for songs.", func() {
					ToggleSwitch(&a.Prefs.ChordTranscription)
					a.SavePrefs()
				})
				SettingRow("Export to audio", "Show the export action in the library.", func() {
					ToggleSwitch(&a.Prefs.ExportAudioEnabled)
					a.SavePrefs()
				})
			}

			if a.UI.SettingsGroup == "playback" {
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
					a.ReapplySongView()
				})
				SettingRow("Tap Tempo", "Show the TAP pad for fixing the beat grid.", func() {
					ToggleSwitch(&a.Prefs.TapTempo)
					a.SavePrefs()
				})
				SettingRow("Tempo warnings", "Mark songs whose bar lines miss the beat.", func() {
					ToggleSwitch(&a.Prefs.TempoWarnings)
					a.SavePrefs()
				})

			}

			if a.UI.SettingsGroup == "keyboard" {
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

			}

			if a.UI.SettingsGroup == "practice" {
				SectionTitle("Practice & Editing")
				SettingRow("Hand assignment", "Colour piano notes by the hand that plays them.", func() {
					ToggleSwitch(&a.Prefs.HandAssignment)
					a.SavePrefs()
					a.ReapplySongView()
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

			}

			if a.UI.SettingsGroup == "editing" {
				SectionTitle("Editing")
				SettingRow("Hand assignment", "Colour piano notes by the hand that plays them.", func() {
					ToggleSwitch(&a.Prefs.HandAssignment)
					a.SavePrefs()
					a.ReapplySongView()
				})
				SettingRow("Hand badges", "Small L/R letters on hand-coloured keys.", func() {
					ToggleSwitch(&a.Prefs.HandTags)
					a.SavePrefs()
				})
				SettingRow("Note editor", "Edit notes on the roll.", func() {
					ToggleSwitch(&a.Prefs.NoteEditor)
					a.SavePrefs()
				})
				SettingRow("Track VU lines", "Track colour lines pulse with the level.", func() {
					ToggleSwitch(&a.Prefs.TrackVU)
					a.SavePrefs()
				})
				if Button(TypThMenu, "Open MIDI Playback Editor") {
					a.UI.PlaybackEditorOpen = true
				}
			}

			if a.UI.SettingsGroup == "notation" {
				SectionTitle("Chords & Notation")
				SettingRow("Note naming", "English, Central European or solfège.", func() {
					CompactSegmented(&a.Prefs.NoteNaming, func() {
						SegmentedCell("C D E", "english")
						SegmentedCell("C D H", "central-european")
						SegmentedCell("Do Re", "solfege")
					})
					a.SavePrefs()
				})
				SettingRow("Accidentals", "Sharps or flats.", func() {
					v := a.Prefs.Accidentals == "sharp"
					ToggleSwitch(&v)
					if v {
						a.Prefs.Accidentals = "sharp"
					} else {
						a.Prefs.Accidentals = "flat"
					}
					a.SavePrefs()
				})
				SettingRow("Chord tracking", "Which instruments the chord display follows.", func() {
					CompactSegmented(&a.Prefs.ChordTracking, func() {
						SegmentedCell("Auto", "auto")
						SegmentedCell("Harmony", "harmony")
						SegmentedCell("Instrument", "instrument")
					})
					a.SavePrefs()
				})
				SettingRow("Chord naming", "Abbreviations or symbols.", func() {
					CompactSegmented(&a.Prefs.ChordNaming, func() {
						SegmentedCell("Abbrev", "abbreviation")
						SegmentedCell("Symbols", "symbol")
					})
					a.SavePrefs()
				})
				SettingRow("Chord reading", "How much detail to show.", func() {
					CompactSegmented(&a.Prefs.ChordReading, func() {
						SegmentedCell("Simple", "simple")
						SegmentedCell("Full", "full")
					})
					a.SavePrefs()
				})
				SettingRow("Chord Prompter", "Past, current and upcoming chords.", func() {
					ToggleSwitch(&a.Prefs.ChordPrompter)
					a.SavePrefs()
				})
			}

			if a.UI.SettingsGroup == "appearance" {
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
			}

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

// FileBrowserWindow browses a folder and opens one song.
func FileBrowserWindow(a *app.App) {
	Modal(780, func() { a.UI.FileBrowserOpen = false }, func() {
		ModAttrs(Pad(18), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Label("Open a song", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
			FlexSpacer()
			if IconButton(TypTimes, "Close", true) {
				a.UI.FileBrowserOpen = false
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
				a.UI.FileBrowserOpen = false
				if err := a.LoadFile(path); err != nil {
					a.Notify(err.Error())
				}
			}
		}
		HDivider()
		SectionTitle("DEMO SONGS")
		Container(Attrs(Row, Gap(8), Wrap), func() {
			for _, path := range a.DemoFiles() {
				if Button(TypMediaPlay, trim(path, 44)) {
					a.UI.FileBrowserOpen = false
					if err := a.LoadFile(path); err != nil {
						a.Notify(err.Error())
					}
				}
			}
		})
		Caption("Tip: choose a folder above, or start with a demo.")
	})
}

// MixerWindow is the console mixer.
func MixerWindow(a *app.App) {
	peaks := 0.0
	if a.Engine != nil {
		peaks = a.Engine.Snapshot().Peak
	}
	channels := 0
	seenCh := map[int]bool{}
	for i := range a.Song.Tracks {
		ch := a.Song.Tracks[i].Channel
		if !seenCh[ch] {
			seenCh[ch] = true
			channels++
		}
	}
	for i := range a.Song.Channels {
		ch := a.Song.Channels[i].Number
		if !seenCh[ch] {
			seenCh[ch] = true
			channels++
		}
	}
	width := 200 + channels*140
	if width < 380 {
		width = 380
	}
	if width > 1180 {
		width = 1180
	}
	Modal(float32(width), func() { a.CloseMixer() }, func() {
		ModAttrs(Pad(16), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross(), Gap(12)), func() {
			Container(Attrs(Gap(0)), func() {
				Label("Console Mixer", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
				Caption("Volume, pan, chorus and reverb save into a new version on close.")
			})
			FlexSpacer()
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.CloseMixer()
			}
		})
		Container(Attrs(Row, Gap(8), ExpandAcross(), FixHeight(360)), func() {
			// Master strip.
			Container(Attrs(FixWidth(132), Expand, Corners(8), Pad(8), Gap(6),
				BackgroundVec(ColPanel2), BorderWidth(1), BorderColorVec(Alpha(ColAmber, .55))), func() {
				Label("MASTER", FontSize(11), FontWeight(WeightBold), TextColorVec(ColAmber))
				Caption(fmt.Sprintf("Volume %.0f%%", a.Prefs.MasterVolume*100))
				v := float32(a.Prefs.MasterVolume)
				Slider(&v, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 112})
				if float64(v) != a.Prefs.MasterVolume {
					a.Prefs.MasterVolume = float64(v)
					a.SavePrefs()
					if a.Engine != nil {
						a.Engine.SetMasterVolume(a.Prefs.MasterVolume)
					}
				}
				tone := float32(a.Prefs.MasterTone)
				Caption("Tone")
				Slider(&tone, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 112})
				if float64(tone) != a.Prefs.MasterTone {
					a.Prefs.MasterTone = float64(tone)
					a.SavePrefs()
				}
				Caption("Compressor")
				ToggleSwitch(&a.Prefs.MasterCompressor)
				a.SavePrefs()
				FlexSpacer()
				LevelMeter(peaks, ColAmber, 112, 60)
			})
			Container(Attrs(Grow(1), Expand, Row, Gap(8), Clip), func() {
				seen := map[int]bool{}
				for i := range a.Song.Tracks {
					t := &a.Song.Tracks[i]
					if seen[t.Channel] {
						continue
					}
					seen[t.Channel] = true
					channelStrip(a, t.Channel, t.Name, TrackColor(t.Color), peaks)
				}
				for i := range a.Song.Channels {
					ch := &a.Song.Channels[i]
					if seen[ch.Number] {
						continue
					}
					seen[ch.Number] = true
					channelStrip(a, ch.Number, fmt.Sprintf("Channel %d", ch.Number+1), ColTextDim, peaks)
				}
			})
		})
	})
}

func channelStrip(a *app.App, ch int, name string, accent Vec4, peak float64) {
	mix := a.MixFor(ch)
	Container(Attrs(FixWidth(148), Expand, Corners(8), Pad(8), Gap(5),
		BackgroundVec(ColPanel), BorderWidth(1), BorderColorVec(Alpha(accent, .5))), func() {
		Label(stripName(name), FontSize(10), FontWeight(WeightBold), TextColorVec(accent))
		Caption(fmt.Sprintf("channel %d", ch+1))
		Caption("Volume")
		v := float32(mix.Volume)
		Slider(&v, SliderAttrs{Min: 0, Max: 1.5, Step: 0.01, Width: 98})
		if float64(v) != mix.Volume {
			mix.Volume = float64(v)
			a.SetMix(ch, mix)
		}
		Caption("Pan")
		p := float32(mix.Pan)
		Slider(&p, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 98})
		if float64(p) != mix.Pan {
			mix.Pan = float64(p)
			a.SetMix(ch, mix)
		}
		c := float32(mix.Chorus)
		Caption("Chorus")
		Slider(&c, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 98})
		if float64(c) != mix.Chorus {
			mix.Chorus = float64(c)
			a.SetMix(ch, mix)
		}
		r := float32(mix.Reverb)
		Caption("Reverb")
		Slider(&r, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 98})
		if float64(r) != mix.Reverb {
			mix.Reverb = float64(r)
			a.SetMix(ch, mix)
		}
		// Meter (master peak; per-channel buses are not exposed by the synth).
		FlexSpacer()
		LevelMeter(peak, accent, 98, 60)
	})
}

// LevelMeter draws a labelled vertical peak meter.
func LevelMeter(peak float64, accent Vec4, width, height float32) {
	Caption("Level")
	Container(Attrs(FixSize(width, height), Corners(4),
		BackgroundVec(Alpha(ColBlack, .6)), BorderWidth(1),
		BorderColorVec(Alpha(ColBorder, .6))), func() {
		h := float32(peak) * height
		if h < 2 {
			h = 2
		}
		if h > height {
			h = height
		}
		Element(Attrs(Float(0, height-h), FixSize(width-2, h), Corners(3),
			BackgroundVec(Alpha(accent, .8))))
		for i := 1; i < 4; i++ {
			Element(Attrs(Float(0, height*float32(i)/4), FixSize(width-2, 1),
				BackgroundVec(Alpha(ColBorder, .55))))
		}
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

func stripName(s string) string {
	if i := strings.Index(s, " ("); i > 0 {
		return s[:i]
	}
	return s
}
