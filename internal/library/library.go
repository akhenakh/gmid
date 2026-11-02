// Package library scans a folder of songs and provides fuzzy search.
package library

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/akhenakh/gmid/internal/smf"
)

// Entry is one song in the library.
type Entry struct {
	Path     string
	Name     string
	Size     int64
	ModTime  time.Time
	BPM      float64
	Duration float64
	Tracks   int
	Notes    int
	Suspect  bool
	Error    string
}

// Library is a scanned folder.
type Library struct {
	mu      sync.RWMutex
	root    string
	entries []Entry
	scanned time.Time
}

// New creates a library rooted at a folder.
func New(root string) *Library { return &Library{root: root} }

// Root is the current folder.
func (l *Library) Root() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.root
}

// SetRoot changes the folder and clears the cache.
func (l *Library) SetRoot(root string) {
	l.mu.Lock()
	l.root = root
	l.entries = nil
	l.scanned = time.Time{}
	l.mu.Unlock()
}

// Scan reads the folder (recursively) in the caller's goroutine.
func (l *Library) Scan() error {
	l.mu.RLock()
	root := l.root
	l.mu.RUnlock()
	var entries []Entry
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mid" && ext != ".midi" && ext != ".kar" &&
			ext != ".musicxml" && ext != ".mxl" && ext != ".xml" &&
			ext != ".gp" && ext != ".gp3" && ext != ".gp4" && ext != ".gp5" && ext != ".gpx" {
			return nil
		}
		st, serr := d.Info()
		if serr != nil {
			return nil
		}
		e := Entry{Path: path, Name: strings.TrimSuffix(d.Name(), ext), Size: st.Size(), ModTime: st.ModTime()}
		if s, perr := smf.ReadFile(path); perr == nil {
			e.BPM = s.BPMAt(0)
			e.Duration = s.DurationSec
			e.Tracks = len(s.Tracks)
			e.Notes = s.NoteCount
			e.Suspect = s.TempoSuspect
		} else if ext == ".mid" || ext == ".midi" || ext == ".kar" {
			e.Error = perr.Error()
		}
		entries = append(entries, e)
		return nil
	})
	if err != nil {
		return err
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	l.mu.Lock()
	l.entries = entries
	l.scanned = time.Now()
	l.mu.Unlock()
	return nil
}

// Entries returns a copy of the scanned list.
func (l *Library) Entries() []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Entry, len(l.entries))
	copy(out, l.entries)
	return out
}

// Empty reports whether a scan found nothing.
func (l *Library) Empty() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.entries) == 0
}

// Scanned reports whether a scan has completed.
func (l *Library) Scanned() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return !l.scanned.IsZero()
}

// Search returns entries matching a fuzzy query.
func (l *Library) Search(query string) []Entry {
	entries := l.Entries()
	if strings.TrimSpace(query) == "" {
		return entries
	}
	type scored struct {
		e Entry
		s int
	}
	var hits []scored
	for _, e := range entries {
		if score, ok := fuzzyScore(query, e.Name); ok {
			hits = append(hits, scored{e, score})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].s > hits[j].s })
	out := make([]Entry, len(hits))
	for i := range hits {
		out[i] = hits[i].e
	}
	return out
}

// fuzzyScore returns a match score and whether the query is a subsequence.
func fuzzyScore(query, name string) (int, bool) {
	q := strings.ToLower(query)
	n := strings.ToLower(name)
	if strings.Contains(n, q) {
		return 1000 - len(n), true
	}
	qi, score, streak := 0, 0, 0
	for i := 0; i < len(n) && qi < len(q); i++ {
		if n[i] == q[qi] {
			qi++
			streak++
			score += 10 + streak
		} else {
			streak = 0
		}
	}
	return score, qi == len(q)
}
