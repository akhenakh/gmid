package app

import (
	"path/filepath"

	shirei "go.hasen.dev/shirei"

	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/library"
)

// RefreshLibrary scans the library folder in the background.
func (a *App) RefreshLibrary() {
	if a.UI.LibScanning {
		a.UI.LibRescan = true
		return
	}
	root := a.Prefs.LibraryFolder
	if root == "" {
		return
	}
	a.UI.LibScanning = true
	go func() {
		l := library.New(root)
		err := l.Scan()
		shirei.WithFrameLock(func() {
			a.UI.LibScanning = false
			if err == nil {
				a.Lib = l
			} else {
				a.Notify("Library: " + err.Error())
			}
			if a.UI.LibRescan {
				a.UI.LibRescan = false
				a.RefreshLibrary()
			}
		})
		shirei.RequestNextFrame()
	}()
}

// SearchLibrary returns fuzzy matches from the last scan.
func (a *App) SearchLibrary(query string) []library.Entry {
	if a.Lib == nil {
		return nil
	}
	return a.Lib.Search(query)
}

// ToggleStar marks a song as a favourite.
func (a *App) ToggleStar(path string) {
	for i, p := range a.Prefs.Stars {
		if p == path {
			a.Prefs.Stars = append(a.Prefs.Stars[:i], a.Prefs.Stars[i+1:]...)
			_ = a.Prefs.Save()
			return
		}
	}
	a.Prefs.Stars = append(a.Prefs.Stars, path)
	_ = a.Prefs.Save()
}

// IsStarred reports whether a path is a favourite.
func (a *App) IsStarred(path string) bool {
	for _, p := range a.Prefs.Stars {
		if p == path {
			return true
		}
	}
	return false
}

// FavouriteEntries returns starred entries that still exist in the library.
func (a *App) FavouriteEntries() []library.Entry {
	if a.Lib == nil {
		return nil
	}
	var out []library.Entry
	for _, e := range a.Lib.Entries() {
		if a.IsStarred(e.Path) {
			out = append(out, e)
		}
	}
	return out
}

// OpenLibraryAdd dismisses the library and opens the in-app file browser that
// picks a score to copy into it. It starts in Downloads when that folder is
// available; the browser keeps its own last directory after that.
func (a *App) OpenLibraryAdd() {
	a.UI.LibraryOpen = false
	a.UI.LibAddOpen = true
	a.UI.LibAddStart = config.DownloadsDir()
}

// AddFileToLibrary copies an already picked score into the library folder and
// queues a rescan. Name collisions become " (2)" rather than overwriting, and a
// file already inside the library is a no-op.
func (a *App) AddFileToLibrary(path string) {
	if path == "" {
		return
	}
	root := a.Prefs.LibraryFolder
	if root == "" {
		root = config.DefaultLibrary()
		a.Prefs.LibraryFolder = root
	}
	dst, copied, err := library.CopyInto(path, root)
	switch {
	case err != nil:
		a.Notify("Add to library: " + err.Error())
		return
	case !copied:
		a.Notify(filepath.Base(dst) + " is already in the library")
	default:
		a.Notify("Added " + filepath.Base(dst))
	}
	a.SavePrefs()
	a.RefreshLibrary()
}
