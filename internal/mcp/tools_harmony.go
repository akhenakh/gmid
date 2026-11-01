package mcp

import (
	"context"
	"sort"

	sdk "github.com/mark3labs/mcp-go/mcp"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"
)

func (s *Server) registerHarmony() {
	s.srv.AddTool(sdk.NewTool("gmid_chords",
		toolTitle("gmid chords"),
		toolDesc("Read the chord under the playhead, the previous/current/next chords, or a bar-by-bar sequence."),
		sdk.WithString("action", sdk.Description("current | around | sequence"), sdk.WithStringEnumItems([]string{"current", "around", "sequence"})),
		sdk.WithNumber("time", sdk.Description("Song time in seconds.")),
		sdk.WithInteger("limit", sdk.Description("Bars for sequence (default 64).")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			t := floatArg(req, "time", a.Transport.Time)
			switch req.GetString("action", "current") {
			case "around":
				prev, cur, next := a.ChordsAround(t)
				return jsonResult(map[string]string{"previous": prev, "current": cur, "next": next})
			case "sequence":
				limit := intArg(req, "limit", 64)
				if limit <= 0 || limit > 500 {
					limit = 64
				}
				bars := a.Song.BarStarts()
				type barChord struct {
					Bar   int     `json:"bar"`
					Time  float64 `json:"time"`
					Chord string  `json:"chord"`
				}
				out := make([]barChord, 0, limit)
				for i, b := range bars {
					if i >= limit {
						break
					}
					out = append(out, barChord{Bar: i + 1, Time: b, Chord: a.ChordAt(b + 0.02)})
				}
				return jsonResult(out)
			default:
				return jsonResult(map[string]string{"chord": a.ChordAt(t)})
			}
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_scale",
		append(meta("gmid scale", "List scale types or the diatonic chords of one scale."),
			sdk.WithString("action", sdk.Description("types | chords"), sdk.WithStringEnumItems([]string{"types", "chords"})),
			sdk.WithInteger("root", sdk.Description("Pitch class 0–11 (0=C).")),
			sdk.WithString("scale", sdk.Description("Scale id, e.g. major, minor, dorian, blues.")),
			sdk.WithInteger("octave", sdk.Description("Register octave (default 4).")),
		)...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if req.GetString("action", "types") == "types" {
				return jsonResult(theory.Scales)
			}
			if req.GetString("scale", "") == "" {
				return jsonResult(theory.Scales)
			}
			sc := theory.ScaleByID(req.GetString("scale", "major"))
			root := intArg(req, "root", 0)
			oct := intArg(req, "octave", 4)
			chords := theory.DiatonicChords(root, oct, sc)
			return jsonResult(map[string]any{
				"root":  theory.NoteName(root, "english", "flat"),
				"scale": sc.Name, "intervals": sc.Intervals, "chords": chords,
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_chord",
		append(meta("gmid chord", "Build and inspect any chord, including inversions."),
			sdk.WithInteger("root", sdk.Description("Pitch class 0–11 (0=C).")),
			sdk.WithString("suffix", sdk.Description("Chord suffix: '' (major), m, 7, maj7, m7, dim, sus4, …")),
			sdk.WithInteger("octave", sdk.Description("Register octave (default 4).")),
			sdk.WithInteger("inversion", sdk.Description("Inversion 0…n−1.")),
		)...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			root := intArg(req, "root", 0)
			suffix := req.GetString("suffix", "")
			oct := intArg(req, "octave", 4)
			inv := intArg(req, "inversion", 0)
			keys := buildChordKeys(root, oct, suffix, inv)
			return jsonResult(map[string]any{
				"name":  theory.FormatChord(root, suffix, "flat"),
				"keys":  keys,
				"notes": keyNames(keys),
			})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_progressions",
		append(meta("gmid progressions", "List the nine chord-explorer progressions, or resolve one in a key."),
			sdk.WithInteger("index", sdk.Description("Progression index; omit to list all.")),
			sdk.WithInteger("root", sdk.Description("Pitch class 0–11.")),
			sdk.WithInteger("octave", sdk.Description("Register octave (default 4).")),
		)...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			idx := intArg(req, "index", -1)
			if idx < 0 {
				return jsonResult(theory.Progressions)
			}
			if idx >= len(theory.Progressions) {
				return errResult("progression index out of range")
			}
			p := theory.Progressions[idx]
			sc := theory.ScaleByID("major")
			root := intArg(req, "root", 0)
			oct := intArg(req, "octave", 4)
			var rows []map[string]any
			for step := range p.Chords {
				name, keys := theory.ProgressionChord(root, oct, sc, p, step)
				rows = append(rows, map[string]any{
					"step": step + 1, "chord": name, "keys": keys, "notes": keyNames(keys),
				})
			}
			return jsonResult(map[string]any{"name": p.Name, "steps": rows})
		})
	})
}

func buildChordKeys(root, octave int, suffix string, inversion int) []int {
	ct, ok := theory.FindChordType(suffix)
	if !ok {
		ct = theory.ChordType{Intervals: []int{0, 4, 7}}
	}
	ivs := ct.Intervals
	if len(ivs) == 0 {
		ivs = []int{0, 4, 7}
	}
	base := (octave+1)*12 + root
	keys := make([]int, len(ivs))
	for i, iv := range ivs {
		keys[i] = base + iv
	}
	for n := 0; n < inversion%len(keys); n++ {
		keys[0] += 12
		sort.Ints(keys)
	}
	return keys
}

func keyNames(keys []int) []string {
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = theory.NoteName(k, "english", "flat")
	}
	return out
}
