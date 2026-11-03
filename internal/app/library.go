package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ncruces/zenity"
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

// AddFileToLibrary opens the operating system's file picker and copies the
// selected score into the library folder. The dialog runs off the UI thread;
// its result is published under the frame lock.
func (a *App) AddFileToLibrary() {
	if a.UI.LibAdding {
		return
	}
	root := a.Prefs.LibraryFolder
	if root == "" {
		root = config.DefaultLibrary()
		a.Prefs.LibraryFolder = root
	}
	a.UI.LibAdding = true
	go func() {
		path, err := pickScoreFile()
		if errors.Is(err, zenity.ErrCanceled) {
			shirei.WithFrameLock(func() { a.UI.LibAdding = false })
			shirei.RequestNextFrame()
			return
		}
		var copied bool
		if err == nil {
			path, copied, err = library.CopyInto(path, root)
		}
		shirei.WithFrameLock(func() {
			a.UI.LibAdding = false
			switch {
			case err != nil:
				a.Notify("Add to library: " + err.Error())
				return
			case path == "":
				return
			case !copied:
				a.Notify(filepath.Base(path) + " is already in the library")
			default:
				a.Notify("Added " + filepath.Base(path))
			}
			a.SavePrefs()
			a.RefreshLibrary()
		})
		shirei.RequestNextFrame()
	}()
}

var scoreFilePatterns = []string{
	"*.mid", "*.midi", "*.kar", "*.musicxml", "*.xml", "*.mxl",
	"*.gp", "*.gp3", "*.gp4", "*.gp5", "*.gpx", "*.cap", "*.capx",
}

// pickScoreFile opens the platform's file picker. The zenity package uses
// osascript on macOS and zenity/matedialog/qarma on Linux; KDE systems without
// those get a kdialog fallback.
func pickScoreFile() (string, error) {
	if zenity.IsAvailable() {
		opts := []zenity.Option{
			zenity.Title("Add a song to the gmid library"),
			zenity.FileFilters{{
				Name:     "Music scores",
				Patterns: scoreFilePatterns,
				CaseFold: true,
			}},
		}
		if dir := config.DownloadsDir(); dir != "" {
			opts = append(opts, zenity.Filename(dir))
		}
		return zenity.SelectFile(opts...)
	}
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("no native file picker available")
	}
	kdialog, err := exec.LookPath("kdialog")
	if err != nil {
		return "", fmt.Errorf("no native file picker found (install zenity or kdialog)")
	}
	start := config.DownloadsDir()
	if start == "" {
		start, _ = os.UserHomeDir()
	}
	filter := "Music scores (" + strings.Join(scoreFilePatterns, " ") + ")"
	out, err := exec.Command(kdialog, "--getopenfilename", start, filter).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return "", zenity.ErrCanceled
		}
		return "", err
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", zenity.ErrCanceled
	}
	return path, nil
}
