package app

import (
	"sort"

	"github.com/akhenakh/gmid/internal/song"
)

// EditorState is the Note Editor's working state.
type EditorState struct {
	Active   bool
	Track    int
	Tool     string // select | marquee | pen
	Selected map[int64]bool
	Undo     [][]song.Note
	Redo     [][]song.Note
}

// BeginNoteEditor enters edit mode on the first melodic track.
func (a *App) BeginNoteEditor() {
	if a.Song == nil {
		return
	}
	track := -1
	for i := range a.Song.Tracks {
		if !a.Song.Tracks[i].IsDrum && !a.Song.Tracks[i].Excluded {
			track = i
			break
		}
	}
	if track < 0 {
		a.Notify("No editable track")
		return
	}
	a.Edit = EditorState{Active: true, Track: track, Tool: "select", Selected: map[int64]bool{}}
	a.UI.NoteEditorActive = true
}

// EndNoteEditor leaves edit mode.
func (a *App) EndNoteEditor() {
	a.Edit.Active = false
	a.UI.NoteEditorActive = false
}

// EditedTrack returns the track being edited.
func (a *App) EditedTrack() *song.Track {
	if a.Song == nil || a.Edit.Track < 0 || a.Edit.Track >= len(a.Song.Tracks) {
		return nil
	}
	return &a.Song.Tracks[a.Edit.Track]
}

// snapshot pushes an undo point.
func (a *App) snapshot() {
	t := a.EditedTrack()
	if t == nil {
		return
	}
	cp := make([]song.Note, len(t.Notes))
	copy(cp, t.Notes)
	a.Edit.Undo = append(a.Edit.Undo, cp)
	if len(a.Edit.Undo) > 100 {
		a.Edit.Undo = a.Edit.Undo[1:]
	}
	a.Edit.Redo = nil
}

// UndoNoteEdit restores the previous note state.
func (a *App) UndoNoteEdit() {
	t := a.EditedTrack()
	if t == nil || len(a.Edit.Undo) == 0 {
		return
	}
	cp := make([]song.Note, len(t.Notes))
	copy(cp, t.Notes)
	a.Edit.Redo = append(a.Edit.Redo, cp)
	t.Notes = a.Edit.Undo[len(a.Edit.Undo)-1]
	a.Edit.Undo = a.Edit.Undo[:len(a.Edit.Undo)-1]
	a.RecomputeTrack(t)
	a.ReloadEngine()
}

// RedoNoteEdit reapplies the last undone edit.
func (a *App) RedoNoteEdit() {
	t := a.EditedTrack()
	if t == nil || len(a.Edit.Redo) == 0 {
		return
	}
	cp := make([]song.Note, len(t.Notes))
	copy(cp, t.Notes)
	a.Edit.Undo = append(a.Edit.Undo, cp)
	t.Notes = a.Edit.Redo[len(a.Edit.Redo)-1]
	a.Edit.Redo = a.Edit.Redo[:len(a.Edit.Redo)-1]
	a.RecomputeTrack(t)
	a.ReloadEngine()
}

// DeleteSelected removes selected notes.
func (a *App) DeleteSelected() {
	t := a.EditedTrack()
	if t == nil || len(a.Edit.Selected) == 0 {
		return
	}
	a.snapshot()
	out := t.Notes[:0]
	for _, n := range t.Notes {
		if !a.Edit.Selected[n.ID] {
			out = append(out, n)
		}
	}
	t.Notes = out
	a.Edit.Selected = map[int64]bool{}
	a.NormalizeTracks()
	a.ReloadEngine()
}

// AddNote inserts a note at a tick.
func (a *App) AddNote(key int, tick, durTicks int64, vel int) {
	t := a.EditedTrack()
	if t == nil {
		return
	}
	a.snapshot()
	n := song.Note{Key: key, Tick: tick, DurTicks: durTicks, Velocity: vel, Track: a.Edit.Track}
	a.RecomputeNote(&n)
	t.Notes = append(t.Notes, n)
	sort.SliceStable(t.Notes, func(i, j int) bool { return t.Notes[i].Tick < t.Notes[j].Tick })
	a.NormalizeTracks()
	a.ReloadEngine()
}

// MoveSelected shifts the selection in ticks and semitones.
func (a *App) MoveSelected(dTicks int64, dKey int) {
	t := a.EditedTrack()
	if t == nil || len(a.Edit.Selected) == 0 {
		return
	}
	for i := range t.Notes {
		n := &t.Notes[i]
		if !a.Edit.Selected[n.ID] {
			continue
		}
		n.Tick += dTicks
		if n.Tick < 0 {
			n.Tick = 0
		}
		n.Key += dKey
		if n.Key < 0 {
			n.Key = 0
		}
		if n.Key > 127 {
			n.Key = 127
		}
		a.RecomputeNote(n)
	}
	a.ReloadEngine()
}

// ResizeSelected changes the selection's duration.
func (a *App) ResizeSelected(dTicks int64) {
	t := a.EditedTrack()
	if t == nil || len(a.Edit.Selected) == 0 {
		return
	}
	for i := range t.Notes {
		n := &t.Notes[i]
		if !a.Edit.Selected[n.ID] {
			continue
		}
		n.DurTicks += dTicks
		if n.DurTicks < 1 {
			n.DurTicks = 1
		}
		a.RecomputeNote(n)
	}
	a.ReloadEngine()
}

// SetSelectedVelocity sets the velocity of selected notes.
func (a *App) SetSelectedVelocity(vel int) {
	t := a.EditedTrack()
	if t == nil {
		return
	}
	for i := range t.Notes {
		if a.Edit.Selected[t.Notes[i].ID] {
			t.Notes[i].Velocity = clampInt(vel, 1, 127)
		}
	}
	a.ReloadEngine()
}

// SelectNote toggles or replaces the selection.
func (a *App) SelectNote(id int64, add bool) {
	if a.Edit.Selected == nil {
		a.Edit.Selected = map[int64]bool{}
	}
	if !add {
		a.Edit.Selected = map[int64]bool{}
	}
	if a.Edit.Selected[id] && add {
		delete(a.Edit.Selected, id)
	} else {
		a.Edit.Selected[id] = true
	}
}

// SnapshotNotes records an undo point (for drag gestures).
func (a *App) SnapshotNotes() { a.snapshot() }

// SaveNoteEdits writes a versioned file and reloads it.
func (a *App) SaveNoteEdits() {
	path, err := a.SaveVersion("note-editor", "Edited notes on the piano roll")
	if err != nil || path == "" {
		return
	}
	a.EndNoteEditor()
	if err := a.LoadFile(path); err != nil {
		a.Notify(err.Error())
	}
}

func (a *App) RecomputeNote(n *song.Note) {
	if a.Song == nil {
		return
	}
	n.StartSec = a.Song.TickToSec(n.Tick)
	n.EndSec = a.Song.TickToSec(n.Tick + n.DurTicks)
}

func (a *App) RecomputeTrack(t *song.Track) {
	for i := range t.Notes {
		a.RecomputeNote(&t.Notes[i])
	}
}
