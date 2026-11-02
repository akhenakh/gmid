package ui

import (
	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/song"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

type editDrag struct {
	mode    string // move | resize | marquee
	noteID  int64
	lastX   float32
	lastY   float32
	anchorX float32
	anchorY float32
	curX    float32
	curY    float32
}

type editRect struct {
	note      *song.Note
	x, top, w float32
	h         float32
}

// editorInteraction runs before the notes are drawn so ModAttrs happens at the
// right time. active is the roll's ProcessButtonEvents state.
func editorInteraction(a *app.App, layout []KeyLayout, min, max int, now, visible float64, playY, pxPerSec float32, active bool) {
	if !a.Edit.Active || a.Song == nil {
		return
	}
	drag := Use[editDrag]("note-edit-drag")
	rect := GetScreenRect()
	mx := GetInputState().MousePoint[0] - rect.Origin[0]
	my := GetInputState().MousePoint[1] - rect.Origin[1]
	key := KeyAt(mx, layout, min, max)
	yToTime := func(y float32) float64 {
		return float64(now) + float64(playY-y)/float64(pxPerSec)
	}
	timeToTick := func(y float32) int64 { return a.Song.SecToTick(yToTime(y)) }
	hit := hitNoteAt(a, layout, min, mx, my, now, visible, playY, pxPerSec)

	if GetFrameInput().Mouse == MouseClick {
		shift := GetInputState().Modifiers&ModShift != 0
		switch a.Edit.Tool {
		case "pen":
			if hit == nil {
				beat := a.Song.TicksPerBeat()
				a.AddNote(key, timeToTick(my), beat, 90)
				a.SelectNote(lastNoteID(a), false)
			}
		case "marquee":
			drag.mode = "marquee"
			drag.anchorX, drag.anchorY = mx, my
			drag.curX, drag.curY = mx, my
			if !shift {
				a.Edit.Selected = map[int64]bool{}
			}
		default:
			if hit != nil {
				a.SelectNote(hit.note.ID, shift)
				if a.Edit.Selected[hit.note.ID] {
					a.SnapshotNotes()
					drag.noteID = hit.note.ID
					drag.lastX, drag.lastY = mx, my
					if mx > hit.x+hit.w-6 {
						drag.mode = "resize"
					} else {
						drag.mode = "move"
					}
				}
			} else {
				if !shift {
					a.Edit.Selected = map[int64]bool{}
				}
				drag.mode = "marquee"
				drag.anchorX, drag.anchorY = mx, my
				drag.curX, drag.curY = mx, my
			}
		}
	}

	if active && drag.mode != "" {
		switch drag.mode {
		case "move":
			dTicks := timeToTick(my) - timeToTick(drag.lastY)
			prevKey := KeyAt(drag.lastX, layout, min, max)
			a.MoveSelected(dTicks, key-prevKey)
			drag.lastX, drag.lastY = mx, my
		case "resize":
			dTicks := timeToTick(my) - timeToTick(drag.lastY)
			a.ResizeSelected(dTicks)
			drag.lastY = my
		case "marquee":
			drag.curX, drag.curY = mx, my
		}
	}
	if !active && drag.mode != "" {
		if drag.mode == "marquee" {
			selectInRect(a, layout, min, drag, now, visible, playY, pxPerSec)
		}
		drag.mode = ""
	}
}

func hitNoteAt(a *app.App, layout []KeyLayout, min int, mx, my float32, now, visible float64, playY, pxPerSec float32) *editRect {
	t := a.EditedTrack()
	if t == nil {
		return nil
	}
	for i := len(t.Notes) - 1; i >= 0; i-- {
		n := &t.Notes[i]
		key := n.Key + a.Transport.Transpose
		if key < min || key-min >= len(layout) {
			continue
		}
		k := layout[key-min]
		top := playY - float32(n.EndSec-float64(now))*pxPerSec
		bottom := playY - float32(n.StartSec-float64(now))*pxPerSec
		h := bottom - top
		if h < 3 {
			h = 3
		}
		if mx >= k.X && mx <= k.X+k.W && my >= top && my <= top+h {
			return &editRect{note: n, x: k.X, top: top, w: k.W, h: h}
		}
	}
	return nil
}

func selectInRect(a *app.App, layout []KeyLayout, min int, drag *editDrag, now, visible float64, playY, pxPerSec float32) {
	x0, x1 := drag.anchorX, drag.curX
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	y0, y1 := drag.anchorY, drag.curY
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	t := a.EditedTrack()
	if t == nil {
		return
	}
	for i := range t.Notes {
		n := &t.Notes[i]
		key := n.Key + a.Transport.Transpose
		if key < min || key-min >= len(layout) {
			continue
		}
		k := layout[key-min]
		top := playY - float32(n.EndSec-float64(now))*pxPerSec
		bottom := playY - float32(n.StartSec-float64(now))*pxPerSec
		h := bottom - top
		if h < 3 {
			h = 3
		}
		if k.X+k.W >= x0 && k.X <= x1 && top+h >= y0 && top <= y1 {
			a.Edit.Selected[n.ID] = true
		}
	}
}

func lastNoteID(a *app.App) int64 {
	t := a.EditedTrack()
	if t == nil || len(t.Notes) == 0 {
		return 0
	}
	return t.Notes[len(t.Notes)-1].ID
}

// EditToolbar is the small overlay shown while the Note Editor is active.
func EditToolbar(a *app.App) {
	Container(Attrs(Row, CrossMid, Gap(4), Pad2(4, 6), Corners(8),
		BackgroundVec(Alpha(ColPanel2, .95)), BorderWidth(1),
		BorderColorVec(Alpha(ColBorder, .8))), func() {
		Caption("TOOL")
		for _, tool := range []string{"select", "marquee", "pen"} {
			label := map[string]string{"select": "Select", "marquee": "Marquee", "pen": "Pen"}[tool]
			if a.Edit.Tool == tool {
				NextButtonType(ButtonPrimary)
			}
			if CtrlButton(NoIcon, label, true) {
				a.Edit.Tool = tool
			}
		}
		Divider()
		if CtrlButton(TypArrowBackOutline, "Undo (Ctrl+Z)", len(a.Edit.Undo) > 0) {
			a.UndoNoteEdit()
		}
		if CtrlButton(TypArrowForwardOutline, "Redo (Ctrl+Y)", len(a.Edit.Redo) > 0) {
			a.RedoNoteEdit()
		}
		if CtrlButton(TypTrash, "Delete selection", len(a.Edit.Selected) > 0) {
			a.DeleteSelected()
		}
		Divider()
		if CtrlButton(TypArchive, "Save a new version", true) {
			a.SaveNoteEdits()
		}
		if CtrlButton(TypTimes, "Leave the editor", true) {
			a.EndNoteEditor()
		}
	})
}
