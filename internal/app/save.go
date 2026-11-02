package app

import (
	"github.com/akhenakh/gmid/internal/smf"
)

// SaveVersion writes a new versioned file and reports the path.
func (a *App) SaveVersion(kind, summary string) (string, error) {
	path, err := smf.SaveVersion(a.Song, a.Prefs.LibraryFolder, kind, summary)
	if err != nil {
		a.Notify("Save failed: " + err.Error())
		return "", err
	}
	a.Notify("Saved " + path)
	return path, nil
}
