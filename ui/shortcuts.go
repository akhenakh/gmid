package ui

import (
	"github.com/akhenakh/gmid/internal/app"

	. "go.hasen.dev/shirei"
)

// HandleShortcuts processes global keys after the UI has been built, so a
// focused text field can claim WantsKeyboard and suppress app shortcuts.
func HandleShortcuts(a *app.App) {
	c := ActiveCombo()
	// Presentation must always be escapable, even if a widget still claims the
	// keyboard from before the mode was entered.
	if c.Key == KeyF11 && c.Mod&^ModShift == 0 {
		a.UI.Presentation = !a.UI.Presentation
		return
	}
	if c.Key == KeyEscape && a.UI.Presentation {
		a.UI.Presentation = false
		return
	}
	if GetHost().WantsKeyboard {
		return
	}
	primary := PrimaryMod()

	switch {
	case a.Edit.Active && (c.Key == KeyDeleteBackward || c.Key == KeyDeleteForward):
		a.DeleteSelected()
	case a.Edit.Active && c.Key == KeyZ && c.Mod&primary != 0:
		if c.Mod&ModShift != 0 {
			a.RedoNoteEdit()
		} else {
			a.UndoNoteEdit()
		}
	case a.Edit.Active && c.Key == KeyY && c.Mod&primary != 0:
		a.RedoNoteEdit()
	case c.Key == KeySpace && c.Mod == ModNone:
		a.TogglePlay()
	case c.Key == KeyEscape:
		switch {
		case a.Edit.Active:
			a.EndNoteEditor()
		case a.UI.Confirm != nil:
			a.UI.Confirm = nil
		case a.UI.SettingsOpen:
			a.UI.SettingsOpen = false
		case a.UI.LibraryOpen:
			a.UI.LibraryOpen = false
		case a.UI.MixerOpen:
			a.CloseMixer()
		default:
			a.Stop()
		}
	case c.Key == KeyB && c.Mod&primary != 0:
		a.UI.SideOpen = !a.UI.SideOpen
	case c.Key == KeyRight && c.Mod&ModShift != 0:
		a.StepThrough(1, true)
	case c.Key == KeyLeft && c.Mod&ModShift != 0:
		a.StepThrough(-1, true)
	case c.Key == KeyRight:
		a.StepThrough(1, false)
	case c.Key == KeyLeft:
		a.StepThrough(-1, false)
	case c.Key == KeyO && c.Mod&primary != 0:
		a.UI.LibraryOpen = !a.UI.LibraryOpen
	case c.Key == KeyUp && c.Mod&primary != 0:
		a.Prefs.AppZoom = minF(a.Prefs.AppZoom+0.1, 1.5)
		a.SavePrefs()
	case c.Key == KeyDown && c.Mod&primary != 0:
		a.Prefs.AppZoom = maxF(a.Prefs.AppZoom-0.1, 0.8)
		a.SavePrefs()
	case c.Key == KeyP && c.Mod&primary != 0:
		a.UI.SettingsOpen = !a.UI.SettingsOpen
	case c.Key == KeyM && c.Mod&primary != 0 && c.Mod&ModShift != 0:
		if a.UI.MixerOpen {
			a.CloseMixer()
		} else {
			a.UI.MixerOpen = true
		}
	}
}
