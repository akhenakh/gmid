package mcp

import (
	"context"

	sdk "github.com/mark3labs/mcp-go/mcp"

	"github.com/akhenakh/gmid/internal/app"
)

func (s *Server) registerDecent() {
	s.srv.AddTool(sdk.NewTool("gmid_decent",
		toolTitle("gmid DecentSampler"),
		toolDesc("Load, inspect or unload a DecentSampler .dspreset/.dslibrary instrument."),
		sdk.WithString("action", sdk.Required(), sdk.Description("status | load | unload | controls"),
			sdk.WithStringEnumItems([]string{"status", "load", "unload", "controls"})),
		sdk.WithString("path", sdk.Description("Preset path (.dspreset or .dslibrary) for load.")),
		sdk.WithNumber("gain_db", sdk.Description("Instrument gain in dB; applied and persisted.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			switch req.GetString("action", "status") {
			case "load":
				path := req.GetString("path", a.Prefs.DecentPreset)
				if path == "" {
					path = a.Assets + "/decent/demo-piano.dspreset"
				}
				if err := a.LoadDecentPreset(path); err != nil {
					return errResult(describeError("decent", err))
				}
			case "unload":
				a.UnloadDecentPreset()
			case "controls":
				return jsonResult(a.DecentControls())
			}
			if _, ok := req.GetArguments()["gain_db"]; ok {
				a.SetDecentGain(req.GetFloat("gain_db", a.Prefs.DecentGain))
			}
			info, ok := a.DecentStatus()
			return jsonResult(map[string]any{"loaded": ok, "path": a.Prefs.DecentPreset, "gainDb": a.Prefs.DecentGain, "info": info})
		})
	})
}
