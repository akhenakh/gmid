package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

// DemoFiles lists the bundled demo songs.
func (a *App) DemoFiles() []string {
	entries, err := os.ReadDir(filepath.Join(a.Assets, "demo"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".mid" || ext == ".midi" || ext == ".kar" {
			out = append(out, filepath.Join(a.Assets, "demo", e.Name()))
		}
	}
	sort.Strings(out)
	return out
}

// LoadFile opens a supported file and installs it as the current song.
func (a *App) LoadFile(path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mid", ".midi", ".kar":
		return a.loadSMF(path)
	case ".musicxml", ".xml", ".mxl", ".gp", ".gp3", ".gp4", ".gp5", ".gpx", ".cap", ".capx":
		return fmt.Errorf("%s import arrives in a later milestone", ext)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}
}

func (a *App) loadSMF(path string) error {
	s, err := smf.ReadFile(path)
	if err != nil {
		return err
	}
	a.InstallSong(s)
	return nil
}

// InstallSong makes a parsed song current, applying user preferences that do not
// belong in the parser.
func (a *App) InstallSong(s *song.Song) {
	if s == nil {
		a.Song = nil
		a.Notify("Closed song")
		return
	}
	if a.Prefs.SkipEmptyBars {
		s.ApplyLeadCut(a.Prefs.KeepDrumCountIn)
	}
	smf.ApplyHandColors(s, a.Prefs.HandLH, a.Prefs.HandRH)
	a.Song = s
	a.Prefs.LastFile = s.Path
	_ = a.Prefs.Save()
	a.UI.LibraryOpen = false
	a.Notify("Loaded " + s.Name)
}
