package mcp

import (
	"context"
	"encoding/json"

	sdk "github.com/mark3labs/mcp-go/mcp"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"
)

func (s *Server) status(limit int, verbose bool) (*sdk.CallToolResult, error) {
	a := s.app
	out := map[string]any{
		"app":       "gmid",
		"version":   Version,
		"playing":   a.Transport.Playing,
		"position":  a.Transport.Time,
		"speed":     a.Transport.Speed,
		"transpose": a.Transport.Transpose,
		"loop": map[string]any{
			"enabled": a.Transport.LoopEnabled,
			"start":   a.Transport.LoopStart,
			"end":     a.Transport.LoopEnd,
		},
		"metronome": map[string]any{
			"enabled": a.Prefs.Metronome,
			"volume":  a.Prefs.MetronomeVolume,
			"countIn": a.Prefs.CountIn,
			"bars":    a.Prefs.CountInBars,
		},
		"master": map[string]any{
			"volume": a.Prefs.MasterVolume, "tone": a.Prefs.MasterTone,
			"compressor": a.Prefs.MasterCompressor,
		},
		"keyboard": map[string]any{
			"range":  a.Prefs.KeyRange,
			"live":   len(a.LiveKeys),
			"locked": len(a.LockedKeys),
		},
		"library":      a.Prefs.LibraryFolder,
		"midiDevices":  a.MIDIDevices(),
		"export":       a.ExportState,
		"noteEditor":   a.Edit.Active,
		"focusMode":    a.FocusActive(),
		"presentation": a.UI.Presentation,
	}
	if a.Engine != nil {
		snap := a.Engine.Snapshot()
		out["audio"] = map[string]any{
			"ready": snap.Ready, "error": snap.Error, "peak": snap.Peak,
			"outputDelayMs": a.Engine.OutputDelay() * 1000,
			"soundfont":     a.Engine.SoundFontPath(),
		}
	} else {
		out["audio"] = map[string]any{"ready": false, "error": "engine not started"}
	}
	if a.Song != nil {
		v := viewSong(a.Song)
		if !verbose && limit <= 0 && len(v.Tracks) > 32 {
			v.Tracks = v.Tracks[:32]
		}
		if limit > 0 && len(v.Tracks) > limit {
			v.Tracks = v.Tracks[:limit]
		}
		out["song"] = v
	}
	return jsonResult(out)
}

func (s *Server) registerMeta() {
	s.srv.AddTool(sdk.NewTool("gmid_status",
		append(meta("gmid status", "Overall app, song, transport, audio engine and MIDI input status."),
			sdk.WithBoolean("verbose", sdk.Description("Include the full track list and device details.")),
		)...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			return s.status(intArg(req, "limit", 0), boolArg(req, "verbose", false))
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_song",
		meta("gmid song", "Full metadata for the open song: key, tempo, bars and every track.")...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			return jsonResult(viewSong(a.Song))
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_set_track",
		toolTitle("gmid set track"),
		toolDesc("Update one track: name, colour, mute/solo/eye, keyboard lighting, instrument program or exclusion."),
		sdk.WithInteger("index", sdk.Required(), sdk.Description("Track index from gmid_song.")),
		sdk.WithString("name", sdk.Description("New track name.")),
		sdk.WithString("color", sdk.Description("Hex colour, e.g. #e8a027.")),
		sdk.WithInteger("program", sdk.Description("General MIDI program 0–127.")),
		sdk.WithBoolean("muted", sdk.Description("Mute the track.")),
		sdk.WithBoolean("solo", sdk.Description("Solo the track.")),
		sdk.WithBoolean("visible", sdk.Description("Show the track on the roll and keyboard.")),
		sdk.WithBoolean("on_keyboard", sdk.Description("Light the keys for this track.")),
		sdk.WithBoolean("excluded", sdk.Description("Exclude the track from playback and saves.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			i, err := req.RequireInt("index")
			if err != nil || i < 0 || i >= len(a.Song.Tracks) {
				return errResult("index out of range")
			}
			t := &a.Song.Tracks[i]
			args := req.GetArguments()
			reload := false
			if _, ok := args["name"]; ok {
				t.Name = req.GetString("name", t.Name)
			}
			if _, ok := args["color"]; ok {
				t.Color = req.GetString("color", t.Color)
			}
			if _, ok := args["program"]; ok {
				t.Program = intArg(req, "program", t.Program)
				t.GMName = theory.GMName(t.Program)
				t.Group = theory.GMGroupOf(t.Program, t.IsDrum)
				reload = true
			}
			if _, ok := args["muted"]; ok {
				t.Muted = req.GetBool("muted", t.Muted)
			}
			if _, ok := args["solo"]; ok {
				t.Solo = req.GetBool("solo", t.Solo)
			}
			if _, ok := args["visible"]; ok {
				t.Visible = req.GetBool("visible", t.Visible)
			}
			if _, ok := args["on_keyboard"]; ok {
				t.OnKeyboard = req.GetBool("on_keyboard", t.OnKeyboard)
			}
			if _, ok := args["excluded"]; ok {
				t.Excluded = req.GetBool("excluded", t.Excluded)
				reload = true
			}
			a.PushAudioState()
			if reload {
				a.ReloadEngine()
			}
			return jsonResult(viewTrack(a.Song, i))
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_save_version",
		toolTitle("gmid save version"),
		toolDesc("Write the current model as a new <name>_GMID_vN.mid in the library and embed a history entry."),
		sdk.WithString("kind", sdk.Description("Change kind, e.g. ai-edit or mixer.")),
		sdk.WithString("summary", sdk.Description("Human-readable summary stored in the file.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			kind := stringArg(req, "kind", "ai-edit")
			summary := stringArg(req, "summary", "Edited through the gmid MCP server")
			path, err := a.SaveVersion(kind, summary)
			if err != nil {
				return errResult(describeError("save", err))
			}
			return jsonResult(map[string]string{"path": path})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_track_edit",
		toolTitle("gmid track edit"),
		toolDesc("Playback-editor transforms: merge a track into the previous one, or split a keyboard track by hand."),
		sdk.WithString("action", sdk.Required(), sdk.Description("merge | split_hands"), sdk.WithStringEnumItems([]string{"merge", "split_hands"})),
		sdk.WithInteger("index", sdk.Required(), sdk.Description("Track index.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			i, err := req.RequireInt("index")
			if err != nil || i < 0 || i >= len(a.Song.Tracks) {
				return errResult("index out of range")
			}
			switch req.GetString("action", "") {
			case "merge":
				a.MergeTrack(i)
			case "split_hands":
				a.SplitHands(i)
			default:
				return errResult("unknown action")
			}
			return jsonResult(viewSong(a.Song))
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_midi_devices",
		meta("gmid MIDI devices", "List connected hardware MIDI input ports and live event count.")...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			return jsonResult(map[string]any{
				"devices": a.MIDIDevices(),
				"events":  midiEvents(a),
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_settings",
		meta("gmid settings", "Read the full preference object.")...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			return jsonResult(a.Prefs)
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_settings_set",
		toolTitle("gmid settings set"),
		toolDesc("Merge a JSON object into preferences (partial update) and persist it."),
		sdk.WithString("json", sdk.Required(), sdk.Description(`Partial preferences JSON, e.g. {"keyRange":61,"metronome":true}`)),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		raw, errRes := requireText("gmid_settings_set", req, "json")
		if errRes != nil {
			return errRes, nil
		}
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			before := a.Prefs
			if err := json.Unmarshal([]byte(raw), &a.Prefs); err != nil {
				a.Prefs = before
				return errResult(err.Error())
			}
			a.SavePrefs()
			a.PushAudioState()
			a.ReapplySongView()
			if a.Lib != nil {
				a.Lib.SetRoot(a.Prefs.LibraryFolder)
			}
			return jsonResult(a.Prefs)
		})
	})
}

func midiEvents(a *app.App) int64 {
	if a.MIDI == nil {
		return 0
	}
	return a.MIDI.Events()
}
