package gp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealSamples(t *testing.T) {
	dir := os.Getenv("GP_SAMPLES")
	if dir == "" {
		t.Skip("set GP_SAMPLES")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			s, err := Parse(raw, e.Name())
			if err != nil {
				t.Fatal(err)
			}
			notes := 0
			for _, tr := range s.Tracks {
				notes += len(tr.Notes)
			}
			t.Logf("tempo=%v tracks=%d notes=%d bars=%d duration=%.1fs name=%q",
				s.Tempo, len(s.Tracks), notes, len(s.Bars), s.DurationSec, s.Name)
			if notes > 0 && s.DurationSec <= 0 {
				t.Errorf("duration %v", s.DurationSec)
			}
		})
	}
}
