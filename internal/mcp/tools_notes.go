package mcp

import (
	"context"
	"sort"

	sdk "github.com/mark3labs/mcp-go/mcp"

	"github.com/akhenakh/gmid/internal/app"
)

func (s *Server) registerNotes() {
	s.srv.AddTool(sdk.NewTool("gmid_notes",
		append(meta("gmid notes", "Query notes in a time, track and pitch range, ordered by onset."),
			sdk.WithInteger("track", sdk.Description("Track index; omit or −1 for every track.")),
			sdk.WithNumber("from", sdk.Description("Start time in song seconds (default 0).")),
			sdk.WithNumber("to", sdk.Description("End time in song seconds (default song end).")),
			sdk.WithInteger("min_key", sdk.Description("Lowest MIDI key (default 0).")),
			sdk.WithInteger("max_key", sdk.Description("Highest MIDI key (default 127).")),
			sdk.WithBoolean("include_drums", sdk.Description("Include drum tracks.")),
			sdk.WithInteger("limit", sdk.Description("Maximum rows (default 200, max 2000).")),
			sdk.WithInteger("offset", sdk.Description("Rows to skip.")),
		)...,
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			track := intArg(req, "track", -1)
			from := floatArg(req, "from", 0)
			to := floatArg(req, "to", a.Song.DurationSec)
			minKey := intArg(req, "min_key", 0)
			maxKey := intArg(req, "max_key", 127)
			limit := intArg(req, "limit", 200)
			if limit <= 0 || limit > 2000 {
				limit = 200
			}
			offset := intArg(req, "offset", 0)
			includeDrums := boolArg(req, "include_drums", true)
			var rows []noteView
			for i := range a.Song.Tracks {
				if track >= 0 && i != track {
					continue
				}
				t := &a.Song.Tracks[i]
				if t.IsDrum && !includeDrums {
					continue
				}
				for j := range t.Notes {
					n := &t.Notes[j]
					if n.StartSec < from || n.StartSec > to || n.Key < minKey || n.Key > maxKey {
						continue
					}
					rows = append(rows, viewNote(n))
				}
			}
			sort.SliceStable(rows, func(i, j int) bool {
				if rows[i].Start != rows[j].Start {
					return rows[i].Start < rows[j].Start
				}
				return rows[i].Key < rows[j].Key
			})
			total := len(rows)
			if offset > len(rows) {
				offset = len(rows)
			}
			rows = rows[offset:]
			if len(rows) > limit {
				rows = rows[:limit]
			}
			return jsonResult(map[string]any{"total": total, "offset": offset, "notes": rows})
		})
	})

	s.srv.AddTool(sdk.NewTool("gmid_note_edit",
		toolTitle("gmid note editor"),
		toolDesc("Drive the Note Editor: enter/exit, select, add, move, resize, set velocity, delete, undo/redo or save."),
		sdk.WithString("action", sdk.Required(), sdk.Description("enter | exit | status | select | clear_selection | add | move | resize | velocity | delete | undo | redo | save"),
			sdk.WithStringEnumItems([]string{"enter", "exit", "status", "select", "clear_selection", "add", "move", "resize", "velocity", "delete", "undo", "redo", "save"})),
		sdk.WithInteger("track", sdk.Description("Track to edit.")),
		sdk.WithString("tool", sdk.Description("select | marquee | pen")),
		sdk.WithArray("ids", sdk.Description("Note IDs."), sdk.WithIntegerItems()),
		sdk.WithBoolean("add", sdk.Description("Add to the current selection.")),
		sdk.WithInteger("key", sdk.Description("MIDI key for a new note.")),
		sdk.WithInteger("tick", sdk.Description("Start tick for a new note.")),
		sdk.WithInteger("duration_ticks", sdk.Description("Duration in ticks.")),
		sdk.WithInteger("velocity", sdk.Description("Velocity 1–127.")),
		sdk.WithInteger("delta_ticks", sdk.Description("Move/resize amount in ticks.")),
		sdk.WithInteger("delta_key", sdk.Description("Move amount in semitones.")),
	), func(ctx context.Context, req sdk.CallToolRequest) (*sdk.CallToolResult, error) {
		return s.withApp(func(a *app.App) (*sdk.CallToolResult, error) {
			if a.Song == nil {
				return errResult("no song loaded")
			}
			action := req.GetString("action", "status")
			if action == "enter" {
				a.BeginNoteEditor()
				if i := intArg(req, "track", -1); i >= 0 && i < len(a.Song.Tracks) {
					a.Edit.Track = i
				}
				if tool := req.GetString("tool", ""); tool != "" {
					a.Edit.Tool = tool
				}
			}
			if action == "exit" {
				a.EndNoteEditor()
			}
			if action == "save" {
				a.SaveNoteEdits()
				return jsonResult(map[string]any{"saved": true, "path": a.Song.Path})
			}
			if action != "status" && action != "enter" && action != "exit" && action != "save" && !a.Edit.Active {
				return errResult("note editor is not active; call action=enter first")
			}
			if ids := req.GetIntSlice("ids", nil); len(ids) > 0 {
				add := boolArg(req, "add", false)
				for _, id := range ids {
					a.SelectNote(int64(id), add)
					add = true
				}
			}
			switch action {
			case "select":
			case "clear_selection":
				a.Edit.Selected = map[int64]bool{}
			case "add":
				a.AddNote(intArg(req, "key", 60), int64(intArg(req, "tick", 0)),
					int64(intArg(req, "duration_ticks", 480)), intArg(req, "velocity", 90))
				last := lastNoteID(a)
				if last != 0 {
					a.Edit.Selected = map[int64]bool{last: true}
				}
			case "move":
				a.MoveSelected(int64(intArg(req, "delta_ticks", 0)), intArg(req, "delta_key", 0))
			case "resize":
				a.ResizeSelected(int64(intArg(req, "delta_ticks", 0)))
			case "velocity":
				a.SetSelectedVelocity(intArg(req, "velocity", 90))
			case "delete":
				a.DeleteSelected()
			case "undo":
				a.UndoNoteEdit()
			case "redo":
				a.RedoNoteEdit()
			}
			return jsonResult(editorState(a))
		})
	})
}

func lastNoteID(a *app.App) int64 {
	t := a.EditedTrack()
	if t == nil {
		return 0
	}
	var max int64
	for i := range t.Notes {
		if t.Notes[i].ID > max {
			max = t.Notes[i].ID
		}
	}
	return max
}

func editorState(a *app.App) map[string]any {
	ids := make([]int64, 0, len(a.Edit.Selected))
	for id := range a.Edit.Selected {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	track := -1
	if t := a.EditedTrack(); t != nil {
		track = t.Index
	}
	return map[string]any{
		"active": a.Edit.Active, "track": track, "tool": a.Edit.Tool,
		"selected": ids, "undo": len(a.Edit.Undo), "redo": len(a.Edit.Redo),
	}
}
