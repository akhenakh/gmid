package ui

import (
	"github.com/akhenakh/gmid/internal/app"

	. "go.hasen.dev/shirei"
)

// HandleShortcuts processes global keys after the UI has been built, so a
// focused text field can claim WantsKeyboard and suppress app shortcuts.
func HandleShortcuts(a *app.App) {
	if GetHost().WantsKeyboard {
		return
	}
	c := ActiveCombo()
	primary := PrimaryMod()

	switch {
	case c.Key == KeyF11 && c.Mod&^ModShift == 0:
		a.UI.Presentation = !a.UI.Presentation
	case c.Key == KeySpace && c.Mod == ModNone:
		a.TogglePlay()
	case c.Key == KeyEscape:
		switch {
		case a.UI.Presentation:
			a.UI.Presentation = false
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
