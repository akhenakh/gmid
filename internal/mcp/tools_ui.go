package mcp

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"

	sdk "github.com/mark3labs/mcp-go/mcp"

	shirei "go.hasen.dev/shirei"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/ui"
)

func (s *Server) registerUI() {
	s.srv.AddTool(sdk.NewTool("gmid_ui",
		toolTitle("gmid UI"),
		toolDesc("Open or close panels and modes: settings, library, mixer, editors, explorers, presentation, floating keyboard."),
		sdk.WithBoolean("settings", sdk.Description("Settings window.")),
		sdk.WithBoolean("library", sdk.Description("Library window.")),
		sdk.WithBoolean("mixer", sdk.Description("Console Mixer.")),
		sdk.WithBoolean("playback_editor", sdk.Description("MIDI Playback Editor.")),
		sdk.WithBoolean("note_editor", sdk.Description("Note Editor.")),
		sdk.WithBoolean("chord_explorer", sdk.Description("Chord Explorer.")),
		sdk.WithBoolean("scale_explorer", sdk.Description("Scale Explorer.")),
		sdk.WithBoolean("export", sdk.Description("Export dialog.")),
		sdk.WithBoolean("presentation", sdk.Description("Presentation mode.")),
		sdk.WithBoolean("floating_keys", sdk.Description("Floating keyboard panel.")),
		sdk.WithBoolean("side_panel", sdk.Description("Left Setup panel.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			args := req.GetArguments()
			if _, ok := args["settings"]; ok {
				a.UI.SettingsOpen = req.GetBool("settings", a.UI.SettingsOpen)
			}
			if _, ok := args["library"]; ok {
				a.UI.LibraryOpen = req.GetBool("library", a.UI.LibraryOpen)
			}
			if _, ok := args["mixer"]; ok {
				want := req.GetBool("mixer", a.UI.MixerOpen)
				if a.UI.MixerOpen && !want {
					a.CloseMixer()
				} else {
					a.UI.MixerOpen = want
				}
			}
			if _, ok := args["playback_editor"]; ok {
				a.UI.PlaybackEditorOpen = req.GetBool("playback_editor", a.UI.PlaybackEditorOpen)
			}
			if _, ok := args["note_editor"]; ok {
				want := req.GetBool("note_editor", a.Edit.Active)
				if want && !a.Edit.Active {
					a.BeginNoteEditor()
				}
				if !want && a.Edit.Active {
					a.EndNoteEditor()
				}
			}
			if _, ok := args["chord_explorer"]; ok {
				a.UI.ChordExplorerOpen = req.GetBool("chord_explorer", a.UI.ChordExplorerOpen)
			}
			if _, ok := args["scale_explorer"]; ok {
				a.UI.ScaleExplorerOpen = req.GetBool("scale_explorer", a.UI.ScaleExplorerOpen)
			}
			if _, ok := args["export"]; ok {
				a.UI.ExportOpen = req.GetBool("export", a.UI.ExportOpen)
			}
			if _, ok := args["presentation"]; ok {
				a.UI.Presentation = req.GetBool("presentation", a.UI.Presentation)
			}
			if _, ok := args["floating_keys"]; ok {
				a.UI.FloatingKeys = req.GetBool("floating_keys", a.UI.FloatingKeys)
			}
			if _, ok := args["side_panel"]; ok {
				a.UI.SideOpen = req.GetBool("side_panel", a.UI.SideOpen)
			}
			shirei.RequestNextFrame()
			return jsonResult(uiState(a))
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_keyboard",
		toolTitle("gmid keyboard"),
		toolDesc("Play the virtual keyboard: press/release a key, audition a chord, clear live notes, or lock a chord."),
		sdk.WithString("action", sdk.Required(), sdk.Description("press | release | chord | clear | lock | unlock"),
			sdk.WithStringEnumItems([]string{"press", "release", "chord", "clear", "lock", "unlock"})),
		sdk.WithInteger("key", sdk.Description("MIDI key for press/release/lock.")),
		sdk.WithArray("keys", sdk.Description("MIDI keys for a chord."), sdk.WithIntegerItems()),
		sdk.WithInteger("duration_ms", sdk.Description("Chord audition length (default 700 ms).")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			switch req.GetString("action", "") {
			case "press":
				a.KeyDown(intArg(req, "key", 60))
			case "release":
				a.KeyUp(intArg(req, "key", 60))
			case "chord":
				keys := req.GetIntSlice("keys", nil)
				if len(keys) == 0 {
					keys = []int{intArg(req, "key", 60)}
				}
				a.PreviewChord(keys, intArg(req, "duration_ms", 700))
			case "clear":
				a.ReleaseAllKeys()
			case "lock":
				a.ToggleLockedKey(intArg(req, "key", 60))
			case "unlock":
				a.LockedKeys = map[int]bool{}
				a.UI.LockedChordOpen = false
			}
			return jsonResult(map[string]any{
				"live": a.LiveKeys, "locked": a.LockedChord(),
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_screenshot",
		toolTitle("gmid screenshot"),
		toolDesc("Render the current UI to a PNG and return it as an image for visual inspection."),
		sdk.WithInteger("width", sdk.Description("Logical width (default current window or 1440).")),
		sdk.WithInteger("height", sdk.Description("Logical height (default current window or 900).")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		host := shirei.GetHost()
		oldSize := host.WindowSize
		oldScale := host.WindowScale
		w := intArg(req, "width", int(oldSize[0]))
		h := intArg(req, "height", int(oldSize[1]))
		if w <= 0 {
			w = 1440
		}
		if h <= 0 {
			h = 900
		}
		if w < 1280 {
			w = 1280
		}
		if h < 800 {
			h = 800
		}
		tmp, err := os.CreateTemp("", "gmid-shot-*.png")
		if err != nil {
			return errResult(err.Error())
		}
		name := tmp.Name()
		_ = tmp.Close()
		defer func() { _ = os.Remove(name) }()
		if err := shirei.RenderToPNG(name, w, h, func() { ui.Root(s.app) }); err != nil {
			return errResult(describeError("render", err))
		}
		host.WindowSize = oldSize
		host.WindowScale = oldScale
		data, err := os.ReadFile(name)
		if err != nil {
			return errResult(err.Error())
		}
		return sdk.NewToolResultImage(
			"gmid UI at "+filepath.Base(name),
			base64.StdEncoding.EncodeToString(data), "image/png",
		), nil
	})
}

func uiState(a *app.App) map[string]any {
	return map[string]any{
		"settings": a.UI.SettingsOpen, "library": a.UI.LibraryOpen,
		"mixer": a.UI.MixerOpen, "playbackEditor": a.UI.PlaybackEditorOpen,
		"noteEditor": a.Edit.Active, "chordExplorer": a.UI.ChordExplorerOpen,
		"scaleExplorer": a.UI.ScaleExplorerOpen, "export": a.UI.ExportOpen,
		"presentation": a.UI.Presentation, "floatingKeys": a.UI.FloatingKeys,
		"sidePanel": a.UI.SideOpen,
	}
}
