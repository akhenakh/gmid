package mcp

import (
	"context"

	sdk "github.com/mark3labs/mcp-go/mcp"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/export"
	"github.com/akhenakh/gmid/internal/library"
)

func (s *Server) registerLibrary() {
	s.srv.AddTool(sdk.NewTool("gmid_open",
		toolTitle("gmid open song"),
		toolDesc("Open a MIDI, MusicXML, KAR, Guitar Pro or Capella file from disk."),
		sdk.WithString("path", sdk.Required(), sdk.Description("Absolute file path.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		path, errRes := requireText("gmid_open", req, "path")
		if errRes != nil {
			return errRes, nil
		}
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if err := a.LoadFile(path); err != nil {
				return errResult(describeError("open", err))
			}
			return jsonResult(viewSong(a.Song))
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_library",
		toolTitle("gmid library"),
		toolDesc("Scan, search, star or load songs from the configured library folder."),
		sdk.WithString("action", sdk.Required(), sdk.Description("scan | list | search | star | unstar | load"),
			sdk.WithStringEnumItems([]string{"scan", "list", "search", "star", "unstar", "load"})),
		sdk.WithString("query", sdk.Description("Fuzzy search query.")),
		sdk.WithString("path", sdk.Description("Song path for star/unstar/load.")),
		sdk.WithInteger("limit", sdk.Description("Maximum rows (default 100).")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			action := req.GetString("action", "list")
			if action == "scan" || a.Lib == nil || !a.Lib.Scanned() {
				l := library.New(a.Prefs.LibraryFolder)
				if err := l.Scan(); err != nil {
					return errResult(describeError("scan", err))
				}
				a.Lib = l
			}
			switch action {
			case "star":
				a.ToggleStar(req.GetString("path", ""))
			case "unstar":
				p := req.GetString("path", "")
				if a.IsStarred(p) {
					a.ToggleStar(p)
				}
			case "load":
				p := req.GetString("path", "")
				if err := a.LoadFile(p); err != nil {
					return errResult(describeError("load", err))
				}
				return jsonResult(viewSong(a.Song))
			}
			entries := a.SearchLibrary(req.GetString("query", ""))
			limit := intArg(req, "limit", 100)
			if limit <= 0 || limit > 1000 {
				limit = 100
			}
			if len(entries) > limit {
				entries = entries[:limit]
			}
			rows := make([]map[string]any, 0, len(entries))
			for _, e := range entries {
				rows = append(rows, map[string]any{
					"path": e.Path, "name": e.Name, "bpm": e.BPM,
					"duration": e.Duration, "tracks": e.Tracks, "notes": e.Notes,
					"starred": a.IsStarred(e.Path), "tempoSuspect": e.Suspect,
					"error": e.Error,
				})
			}
			return jsonResult(map[string]any{
				"root": a.Prefs.LibraryFolder, "count": len(entries), "songs": rows,
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_export_audio",
		toolTitle("gmid export audio"),
		toolDesc("Render the current song to WAV or 320 kbps MP3 in the background."),
		sdk.WithString("variant", sdk.Description("original | focus | heard"), sdk.WithStringEnumItems([]string{"original", "focus", "heard"})),
		sdk.WithString("format", sdk.Description("wav | mp3"), sdk.WithStringEnumItems([]string{"wav", "mp3"})),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			if a.ExportState.Running {
				return errResult("an export is already running")
			}
			variant := export.Original
			switch req.GetString("variant", "heard") {
			case "focus":
				variant = export.Focus
			case "heard":
				variant = export.Heard
			}
			format := req.GetString("format", "wav")
			if format != "mp3" {
				format = "wav"
			}
			a.ExportAudio(variant, format)
			return jsonResult(map[string]any{"started": true, "variant": variant, "format": format})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_export_status",
		meta("gmid export status", "Progress and result of the running/last audio export.")...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			return jsonResult(a.ExportState)
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_chord_chart",
		toolTitle("gmid chord chart"),
		toolDesc("Generate a printable chord-chart PDF for the open song."),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			a.TranscribeChordChart()
			return jsonResult(map[string]any{"started": true})
		})
	})
}
