package mcp

import (
	"context"

	sdk "github.com/mark3labs/mcp-go/mcp"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/audio"
)

func (s *Server) registerTransport() {
	s.srv.AddTool(sdk.NewTool("gmid_transport",
		toolTitle("gmid transport"),
		toolDesc("Control playback: play, pause, toggle, stop, seek, speed, transpose, or read the clock."),
		sdk.WithString("action", sdk.Required(), sdk.Description("play | pause | toggle | stop | seek | speed | transpose | status"),
			sdk.WithStringEnumItems([]string{"play", "pause", "toggle", "stop", "seek", "speed", "transpose", "status"})),
		sdk.WithNumber("time", sdk.Description("Seek target in song seconds.")),
		sdk.WithNumber("speed", sdk.Description("Tempo ratio 0.25–2.0; pitch is unchanged.")),
		sdk.WithInteger("transpose", sdk.Description("Semitones −24…24.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil && req.GetString("action", "") != "status" {
				return errResult("no song loaded")
			}
			switch req.GetString("action", "status") {
			case "play":
				if !a.Transport.Playing {
					a.TogglePlay()
				}
			case "pause":
				if a.Transport.Playing {
					a.TogglePlay()
				}
			case "toggle":
				a.TogglePlay()
			case "stop":
				a.Stop()
			case "seek":
				a.Seek(floatArg(req, "time", a.Transport.Time))
			case "speed":
				a.SetSpeed(floatArg(req, "speed", a.Transport.Speed))
			case "transpose":
				a.SetTranspose(intArg(req, "transpose", a.Transport.Transpose))
			}
			return jsonResult(map[string]any{
				"playing": a.Transport.Playing, "position": a.Transport.Time,
				"speed": a.Transport.Speed, "transpose": a.Transport.Transpose,
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_loop",
		toolTitle("gmid loop"),
		toolDesc("Set, clear or inspect the loop region."),
		sdk.WithBoolean("enabled", sdk.Description("Enable the loop region.")),
		sdk.WithNumber("start", sdk.Description("Loop start in song seconds.")),
		sdk.WithNumber("end", sdk.Description("Loop end in song seconds.")),
		sdk.WithBoolean("clear", sdk.Description("Disable and clear the loop.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if boolArg(req, "clear", false) {
				a.Transport.LoopEnabled = false
				a.Transport.LoopStart, a.Transport.LoopEnd = 0, 0
				if a.Engine != nil {
					a.Engine.SetLoop(false, 0, 0)
				}
			} else {
				if _, ok := req.GetArguments()["start"]; ok {
					a.Transport.LoopStart = floatArg(req, "start", 0)
				}
				if _, ok := req.GetArguments()["end"]; ok {
					a.Transport.LoopEnd = floatArg(req, "end", 0)
				}
				if _, ok := req.GetArguments()["enabled"]; ok {
					a.Transport.LoopEnabled = boolArg(req, "enabled", true)
				} else {
					a.Transport.LoopEnabled = true
				}
				if a.Engine != nil {
					a.Engine.SetLoop(a.Transport.LoopEnabled, a.Transport.LoopStart, a.Transport.LoopEnd)
				}
			}
			return jsonResult(map[string]any{
				"enabled": a.Transport.LoopEnabled,
				"start":   a.Transport.LoopStart,
				"end":     a.Transport.LoopEnd,
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_metronome",
		toolTitle("gmid metronome"),
		toolDesc("Enable the click, set its volume, and configure the count-in."),
		sdk.WithBoolean("enabled", sdk.Description("Metronome on/off.")),
		sdk.WithNumber("volume", sdk.Description("Click level 0–1.")),
		sdk.WithBoolean("count_in", sdk.Description("Enable the count-in.")),
		sdk.WithInteger("bars", sdk.Description("Count-in bars 1–4.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			args := req.GetArguments()
			if _, ok := args["enabled"]; ok {
				a.Prefs.Metronome = req.GetBool("enabled", a.Prefs.Metronome)
			}
			if _, ok := args["volume"]; ok {
				a.Prefs.MetronomeVolume = floatArg(req, "volume", a.Prefs.MetronomeVolume)
			}
			if _, ok := args["count_in"]; ok {
				a.Prefs.CountIn = req.GetBool("count_in", a.Prefs.CountIn)
			}
			if _, ok := args["bars"]; ok {
				a.Prefs.CountInBars = intArg(req, "bars", a.Prefs.CountInBars)
			}
			a.SavePrefs()
			a.PushAudioState()
			return jsonResult(map[string]any{
				"enabled": a.Prefs.Metronome, "volume": a.Prefs.MetronomeVolume,
				"countIn": a.Prefs.CountIn, "bars": a.Prefs.CountInBars,
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_mixer",
		toolTitle("gmid mixer"),
		toolDesc("Read or change the console mixer and the master strip."),
		sdk.WithString("action", sdk.Description("get | set"), sdk.WithStringEnumItems([]string{"get", "set"})),
		sdk.WithInteger("channel", sdk.Description("MIDI channel 0–15 for a channel strip change.")),
		sdk.WithNumber("volume", sdk.Description("Channel fader 0–1.5.")),
		sdk.WithNumber("pan", sdk.Description("Channel pan 0–1 (0.5 centre).")),
		sdk.WithNumber("chorus", sdk.Description("Channel chorus send 0–1.")),
		sdk.WithNumber("reverb", sdk.Description("Channel reverb send 0–1.")),
		sdk.WithNumber("master_volume", sdk.Description("Master volume 0–1.5.")),
		sdk.WithNumber("master_tone", sdk.Description("Master tone 0 dark … 1 bright.")),
		sdk.WithBoolean("compressor", sdk.Description("Master compressor on/off.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			args := req.GetArguments()
			if req.GetString("action", "get") == "set" {
				if _, ok := args["channel"]; ok {
					ch := intArg(req, "channel", 0)
					m := a.MixFor(ch)
					if _, ok := args["volume"]; ok {
						m.Volume = floatArg(req, "volume", m.Volume)
					}
					if _, ok := args["pan"]; ok {
						m.Pan = floatArg(req, "pan", m.Pan)
					}
					if _, ok := args["chorus"]; ok {
						m.Chorus = floatArg(req, "chorus", m.Chorus)
					}
					if _, ok := args["reverb"]; ok {
						m.Reverb = floatArg(req, "reverb", m.Reverb)
					}
					a.SetMix(ch, m)
				}
				if _, ok := args["master_volume"]; ok {
					a.Prefs.MasterVolume = floatArg(req, "master_volume", a.Prefs.MasterVolume)
				}
				if _, ok := args["master_tone"]; ok {
					a.Prefs.MasterTone = floatArg(req, "master_tone", a.Prefs.MasterTone)
				}
				if _, ok := args["compressor"]; ok {
					a.Prefs.MasterCompressor = req.GetBool("compressor", a.Prefs.MasterCompressor)
				}
				a.SavePrefs()
				a.PushAudioState()
			}
			mixes := map[int]audio.ChannelMix{}
			if a.Song != nil {
				for i := range a.Song.Channels {
					ch := a.Song.Channels[i].Number
					mixes[ch] = a.MixFor(ch)
				}
			}
			return jsonResult(map[string]any{
				"master": map[string]any{
					"volume": a.Prefs.MasterVolume, "tone": a.Prefs.MasterTone,
					"compressor": a.Prefs.MasterCompressor,
				},
				"channels": mixes,
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_focus",
		toolTitle("gmid focus mode"),
		toolDesc("Turn Focus mode on/off (hear keys, bass and drums; show only keys)."),
		sdk.WithString("action", sdk.Description("on | off | toggle | status"), sdk.WithStringEnumItems([]string{"on", "off", "toggle", "status"})),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			on := a.FocusActive()
			switch req.GetString("action", "toggle") {
			case "on":
				if !on {
					a.ToggleFocus()
				}
			case "off":
				if on {
					a.ToggleFocus()
				}
			case "toggle":
				a.ToggleFocus()
			}
			return jsonResult(map[string]any{"active": a.FocusActive()})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_tap",
		toolTitle("gmid tap tempo"),
		toolDesc("Tap the beat (call repeatedly), inspect or clear the song's tapped grid."),
		sdk.WithString("action", sdk.Description("tap | state | clear"), sdk.WithStringEnumItems([]string{"tap", "state", "clear"})),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			switch req.GetString("action", "tap") {
			case "tap":
				a.TapTempo()
			case "clear":
				a.ClearTempoFix()
			}
			return jsonResult(map[string]any{
				"taps": a.TapCount(), "fixed": a.Song.BeatFixed,
				"bpm": a.Song.EffectiveBPM(a.Transport.Time),
			})
		})
	})
}
