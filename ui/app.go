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
	trackPanelWidth = 290
	keyboardHeight  = 150
)

// Root builds the whole application surface.
func Root(a *app.App) {
	hoverTip = ""
	a.SyncEngine()
	a.UpdateHits()
	GetHost().ComfortScale = clampf(float32(a.Prefs.AppZoom), 0.8, 1.5)
	ModAttrs(UseSurface(SurfaceCanvas),
		AmendTextStyle(Fonts("Inter"), FontSize(13)))

	if a.UI.Presentation {
		Presentation(a)
		HandleShortcuts(a)
		DrawTooltip(a)
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
	DrawTooltip(a)
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

		// Panel toggle: the Setup pane can be closed from its own header, so
		// keep a recall button in the chrome.
		panelIcon := TypChevronLeft
		if !a.UI.SideOpen {
			panelIcon = TypChevronRight
		}
		if IconButton(panelIcon, "Toggle setup panel (Ctrl+B)", true) {
			a.UI.SideOpen = !a.UI.SideOpen
		}

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

		tempoControl(a)
		keyControl(a)
		if a.Prefs.TempoWarnings && a.Song != nil && a.Song.TempoSuspect {
			Badge(ColAmber, 7)
			Caption("beat may not match")
		}

		if IconButton(TypArrowRepeat, "Loop region", a.Song != nil) {
			a.ToggleLoop()
		}
		metID := metronomeButton(a)
		PopupPanel(&a.UI.MetronomeMenu, metID, Attrs(Pad(14), Gap(8), FixWidth(240), Corners(10)), func() {
			ModAttrs(UseSurface(SurfacePanel))
			Label("Metronome", FontSize(14), FontWeight(WeightBold), TextColorVec(ColText))
			SettingRow("Click", "Follows the file's tempo map.", func() {
				ToggleSwitch(&a.Prefs.Metronome)
				a.SavePrefs()
				a.PushAudioState()
			})
			SettingRow("Volume", "Independent of the master fader.", func() {
				v := float32(a.Prefs.MetronomeVolume)
				Slider(&v, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 110})
				if float64(v) != a.Prefs.MetronomeVolume {
					a.Prefs.MetronomeVolume = float64(v)
					a.SavePrefs()
					a.PushAudioState()
				}
			})
			HDivider()
			SettingRow("Count-in", "Clicks before the music starts.", func() {
				ToggleSwitch(&a.Prefs.CountIn)
				a.SavePrefs()
				a.PushAudioState()
			})
			SettingRow("Bars", "1–4 bars.", func() {
				n := a.Prefs.CountInBars
				if CtrlButton(TypMinus, "", n > 1) {
					a.Prefs.CountInBars--
					a.SavePrefs()
					a.PushAudioState()
				}
				Label(fmt.Sprintf("%d", n), Fonts("JetBrains Mono"), FontSize(12), TextColorVec(ColText))
				if CtrlButton(TypPlus, "", n < 4) {
					a.Prefs.CountInBars++
					a.SavePrefs()
					a.PushAudioState()
				}
			})
		})
		if IconButton(TypCog, "Settings (Ctrl+,)", true) {
			a.UI.SettingsOpen = !a.UI.SettingsOpen
		}
		if IconButton(TypArrowMaximise, "Presentation mode (F11, Esc exits)", true) {
			a.UI.Presentation = true
		}
		if IconButton(TypKeyboard, "Floating keyboard", true) {
			a.UI.FloatingKeys = !a.UI.FloatingKeys
		}
	})
}

func metronomeButton(a *app.App) ContainerId {
	return Container(Attrs(FixSize(30, 28), Corners(6)), func() {
		ProcessButtonEvents(false)
		if a.Prefs.Metronome {
			ModAttrs(BackgroundVec(Alpha(ColAmber, .18)))
		}
		if IsHovered() {
			if GetFrameInput().Mouse == MouseClick {
				if GetInputState().MouseButton == MouseSecondary {
					a.UI.MetronomeMenu = true
				} else {
					a.ToggleMetronome()
				}
			}
		}
		c := ColTextDim
		if a.Prefs.Metronome {
			c = ColAmber
		}
		Icon(TypStopwatch, FontSize(15), TextColorVec(c))
	})
}

func MainColumn(a *app.App) {
	Container(Attrs(Grow(1), Expand, Pad(10), Gap(8)), func() {
		if os.Getenv("GMID_DEBUG") != "" {
			fmt.Fprintf(os.Stderr, "main w=%.1f h=%.1f\n", GetContentWidth(), GetContentHeight())
		}
		RollHeader(a)
		PianoRoll(a)
		if a.Edit.Active {
			VelocityLane(a)
		}
		if a.Prefs.LoopRegion {
			LoopStrip(a)
		}
		KeyboardBar(a)
		if !a.UI.FloatingKeys {
			bed := Container(Attrs(ExpandAcross(), Corners(8), Clip, Pad2(0, 1),
				BackgroundVec(Col("#08080b")), BorderWidth(1),
				BorderColorVec(Alpha(ColBorder, .6))), func() {
				min, max := EffectiveKeyRange(a)
				PianoKeyboard(min, max, keyboardHeight, soundingKeys(a),
					a.Prefs.ShowNoteNames, a.Prefs.ShowOctaves, a.KeyDown, a.KeyUp)
			})
			_ = bed
		}
	})
}

func RollHeader(a *app.App) {
	Container(Attrs(Row, CrossMid, Gap(8), ExpandAcross()), func() {
		if a.Song == nil {
			Caption("Open a file to begin")
		} else {
			Caption(fmt.Sprintf("Bar %d · %s / %d", a.Song.BarNumberAt(a.Transport.Time),
				fmt.Sprintf("%.0f", a.Song.EffectiveBPM(a.Transport.Time)), a.Song.PPQ))
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
		if a.Edit.Active {
			EditToolbar(a)
			FlexSpacer()
		}
		if a.Prefs.TapTempo && a.Song != nil {
			label := "TAP"
			if n := a.TapCount(); n > 0 {
				label = fmt.Sprintf("TAP %d", n)
			}
			if CtrlButton(TypTime, label, true) {
				a.TapTempo()
			}
			if a.Song.BeatFixed {
				if CtrlButton(TypRefresh, "Clear tap tempo", true) {
					a.ClearTempoFix()
				}
			}
		}
		if IconButton(TypFolderOpen, "Library", true) {
			a.UI.LibraryOpen = !a.UI.LibraryOpen
		}
		if IconButton(TypThLarge, "Console Mixer (Ctrl+Shift+M)", a.Song != nil) {
			if a.UI.MixerOpen {
				a.CloseMixer()
			} else {
				a.UI.MixerOpen = true
			}
		}
		if CtrlButton(NoIcon, "Chords", true) {
			a.UI.ChordExplorerOpen = !a.UI.ChordExplorerOpen
		}
		if CtrlButton(NoIcon, "Scales", true) {
			a.UI.ScaleExplorerOpen = !a.UI.ScaleExplorerOpen
		}
		if IconButton(TypExport, "Export to audio", a.Song != nil) {
			a.UI.ExportOpen = true
		}
		if IconButton(TypPrinter, "Chord chart PDF", a.Song != nil) {
			a.TranscribeChordChart()
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
		if a.Prefs.ChordPrompter && a.Song != nil {
			prev, cur, next := a.ChordsAround(a.Transport.Time)
			Container(Attrs(Row, CrossMid, Gap(14)), func() {
				ChordCell("PREV", prev, ColTextMute)
				ChordCell("NOW", cur, ColAmber)
				ChordCell("NEXT", next, ColTextDim)
			})
		} else {
			Label(chordNameAt(a), FontSize(15), FontWeight(WeightBold), TextColorVec(ColText))
		}
		FlexSpacer()
		if a.Song != nil {
			if a.Prefs.FitSongRange {
				Caption("fitted to song")
			} else {
				Caption(fmt.Sprintf("%d keys", a.Prefs.KeyRange))
			}
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
			Container(Attrs(Gap(3), Pad2(6, 2)), func() {
				Label("Engine", FontSize(12), TextColorVec(ColText))
				hint := "GM is instant; Studio and Native use soundfonts."
				if info, ok := a.DecentStatus(); ok {
					hint = "DecentSampler (" + info.Name + ") overrides melodic tracks."
				}
				Caption(hint)
				if CompactSegmented(&a.Prefs.SoundEngine, func() {
					SegmentedCell("GM", "gm")
					SegmentedCell("Studio", "studio")
					SegmentedCell("Native", "native")
				}) {
					a.SetSoundEngine(a.Prefs.SoundEngine)
				}
			})
			SettingRow("Master volume", "Persists across sessions.", func() {
				v := float32(a.Prefs.MasterVolume)
				Slider(&v, SliderAttrs{Min: 0, Max: 1, Step: 0.01, Width: 110})
				if float64(v) != a.Prefs.MasterVolume {
					a.Prefs.MasterVolume = float64(v)
					a.SavePrefs()
					if a.Engine != nil {
						a.Engine.SetMasterVolume(a.Prefs.MasterVolume)
					}
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
			SettingRow("Fit to song", "Show only the pitches this song uses.", func() {
				ToggleSwitch(&a.Prefs.FitSongRange)
				a.SavePrefs()
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
				Label(a.Status, FontSize(11), TextColorVec(ColTextDim))
			}
		})
	})
}

// SettingRow places a control to the right of a title and one-line hint.
func SettingRow(title, hint string, control func()) {
	Container(Attrs(Row, CrossMid, ExpandAcross(), Gap(10), Pad2(6, 2)), func() {
		w := GetResolvedWidth()
		textMax := float32(0)
		if w > 150 {
			textMax = w - 110
		}
		Container(Attrs(Grow(1), Gap(0), MaxWidth(textMax)), func() {
			Label(title, FontSize(12), TextColorVec(ColText))
			Caption(hint)
		})
		Container(Attrs(Row, CrossMid, Gap(4), MainAlign(AlignEnd), MinWidth(50)), control)
	})
}

func TrackPanel(a *app.App) {
	Container(Attrs(FixWidth(trackPanelWidth), Expand, UseSurface(SurfacePanel),
		Pad(12), Gap(8)), func() {
		PanelHeader("Tracks", func() {
			if a.Prefs.FocusMode && a.Song != nil {
				if a.FocusActive() {
					NextButtonType(ButtonPrimary)
					if CtrlButton(TypAdjustContrast, "Focus on", true) {
						a.ToggleFocus()
					}
				} else if CtrlButton(TypAdjustContrast, "Focus", true) {
					a.ToggleFocus()
				}
			}
			if IconButton(TypCog, "Track settings", true) {
				a.UI.SettingsOpen = true
			}
			if IconButton(TypThMenu, "MIDI Playback Editor", a.Song != nil) {
				a.UI.PlaybackEditorOpen = true
			}
			if IconButton(TypPen, "Note Editor", a.Prefs.NoteEditor && a.Song != nil) {
				if a.Edit.Active {
					a.EndNoteEditor()
				} else {
					a.BeginNoteEditor()
				}
			}
		})
		Caption("Mute · Solo · Eye")
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
		Container(Attrs(Grow(1), Gap(0), MaxWidth(150)), func() {
			Label(stripName(t.Name), FontSize(11), TextColorVec(func() Vec4 {
				if t.Muted || t.Excluded {
					return ColTextMute
				}
				return ColText
			}()))
			Caption(stripName(t.GMName))
		})
		if IconButton(iconBool(t.Muted, TypVolumeMute, TypVolume), "Mute", true) {
			t.Muted = !t.Muted
			a.PushAudioState()
		}
		if IconButton(iconBool(t.Solo, TypStar, TypStarOutline), "Solo", true) {
			t.Solo = !t.Solo
			a.PushAudioState()
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
	if a.UI.FileBrowserOpen {
		FileBrowserWindow(a)
	}
	if a.UI.MixerOpen && a.Song != nil {
		MixerWindow(a)
	}
	if a.UI.PlaybackEditorOpen && a.Song != nil {
		PlaybackEditorModal(a)
	}
	if a.UI.ChordExplorerOpen {
		ChordExplorer(a)
	}
	if a.UI.ScaleExplorerOpen {
		ScaleExplorer(a)
	}
	if a.UI.LockedChordOpen {
		LockedChordModal(a)
	}
	if a.UI.ExportOpen {
		ExportDialog(a)
	}
	if a.UI.FloatingKeys {
		FloatingKeyboard(a)
	}
	if a.UI.Confirm != nil {
		ConfirmDialog(a)
	}
}

func Presentation(a *app.App) {
	Container(Attrs(FixSizeVec(GetHost().WindowSize), BackgroundVec(Col("#000000")), Pad(0)), func() {
		PianoRoll(a)
		min, max := EffectiveKeyRange(a)
		PianoKeyboard(min, max, 190, soundingKeys(a), a.Prefs.ShowNoteNames, a.Prefs.ShowOctaves, a.KeyDown, a.KeyUp)
		win := GetHost().WindowSize
		Container(Attrs(Float(maxf(win[0]-180, 8), 12), Z(120), NoAnimate,
			Row, CrossMid, Gap(6), Pad2(5, 9), Corners(9),
			BackgroundVec(Alpha(ColToolbar, .94)), BorderWidth(1),
			BorderColorVec(Alpha(ColAmber, .45)), BoxShadow(12)), func() {
			if CtrlButton(TypArrowMinimise, "Exit fullscreen (Esc)", true) {
				a.UI.Presentation = false
			}
		})
	})
}

func ChordCell(label, chord string, c Vec4) {
	Container(Attrs(Row, CrossMid, Gap(5)), func() {
		Label(label, FontSize(8), TextColorVec(ColTextMute))
		Label(chord, FontSize(14), FontWeight(WeightBold), TextColorVec(c))
	})
}

type tempoHold struct {
	ticks int
}

func tempoControl(a *app.App) {
	bpm := 120.0
	if a.Song != nil {
		bpm = a.Song.EffectiveBPM(a.Transport.Time) * a.Transport.Speed
	}
	Pill(Attrs(Gap(4)), func() {
		Caption("BPM")
		Label(fmt.Sprintf("%.0f", bpm), Fonts("JetBrains Mono"), FontSize(12),
			FontWeight(WeightBold), TextColorVec(ColText))
		if a.Transport.Speed != 1 {
			Caption(fmt.Sprintf("%.0f%%", a.Transport.Speed*100))
		}
		tempoStep(a, "tempo-down", -1, TypArrowDown)
		tempoStep(a, "tempo-up", 1, TypArrowUp)
		if a.Transport.Speed != 1 {
			if CtrlButton(TypRefresh, "Reset tempo", true) {
				a.SetSpeed(1)
			}
		}
	})
}

func tempoStep(a *app.App, key string, dir int, icon IconGlyph) {
	h := Use[tempoHold](key)
	Container(Attrs(FixSize(20, 24), Corners(4)), func() {
		ProcessButtonEvents(false)
		if IsActive() {
			if h.ticks == 0 || h.ticks > 18 && h.ticks%4 == 0 {
				a.SetSpeed(a.Transport.Speed + float64(dir)*0.05)
			}
			h.ticks++
			ModAttrs(BackgroundVec(Alpha(ColAmber, .16)))
		} else {
			h.ticks = 0
		}
		Icon(icon, FontSize(13), TextColorVec(ColTextDim))
	})
}

func keyControl(a *app.App) {
	Pill(Attrs(Gap(4)), func() {
		Caption("KEY")
		Label(keyLabel(a), FontSize(12), FontWeight(WeightBold), TextColorVec(ColAmber))
		if a.Transport.Transpose != 0 {
			Caption(fmt.Sprintf("%+d", a.Transport.Transpose))
		}
		if CtrlButton(TypArrowDown, "", true) {
			a.SetTranspose(a.Transport.Transpose - 1)
		}
		if CtrlButton(TypArrowUp, "", true) {
			a.SetTranspose(a.Transport.Transpose + 1)
		}
		if a.Transport.Transpose != 0 {
			if CtrlButton(TypRefresh, "Reset key", true) {
				a.SetTranspose(0)
			}
		}
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
	sem = ((sem+a.Transport.Transpose)%12 + 12) % 12
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
