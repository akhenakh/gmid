package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/hands"
	gimport "github.com/akhenakh/gmid/internal/import"
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
		s, err := gimport.Load(path)
		if err != nil {
			return err
		}
		a.InstallSong(s)
		return nil
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

// computeSongRange caches the pitch extent of the loaded song.
func (a *App) computeSongRange() {
	a.SongMinKey, a.SongMaxKey = 0, 0
	if a.Song == nil {
		return
	}
	lo, hi := 128, -1
	for _, t := range a.Song.Tracks {
		if t.IsDrum {
			continue
		}
		for _, n := range t.Notes {
			if n.Key < lo {
				lo = n.Key
			}
			if n.Key > hi {
				hi = n.Key
			}
		}
	}
	if hi >= lo {
		a.SongMinKey, a.SongMaxKey = lo, hi
	}
}

// InstallSong makes a parsed song current, applying user preferences that do not
// belong in the parser.
func (a *App) InstallSong(s *song.Song) {
	if s == nil {
		a.Song = nil
		a.SongMinKey, a.SongMaxKey = 0, 0
		a.Notify("Closed song")
		return
	}
	a.Song = s
	a.computeSongRange()
	if a.Prefs.SkipEmptyBars {
		s.ApplyLeadCut(a.Prefs.KeepDrumCountIn)
	}
	a.ApplyStoredTempoFix()
	smf.ApplyHandColors(s, a.Prefs.HandLH, a.Prefs.HandRH)
	a.focusSnapshot = nil
	a.Mixes = map[int]audio.ChannelMix{}
	if a.Prefs.HandAssignment {
		hands.AssignSong(s, hands.Defaults())
	}
	a.Transport.Time = s.OriginSec()
	a.Transport.Playing = false
	if a.Engine != nil {
		a.Engine.Load(s)
		a.PushAudioState()
	}
	a.Prefs.LastFile = s.Path
	_ = a.Prefs.Save()
	a.UI.LibraryOpen = false
	a.Notify("Loaded " + s.Name)
}
