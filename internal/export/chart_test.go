package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/smf"
)

func TestChordChart(t *testing.T) {
	s, err := smf.ReadFile(demo)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	home := t.TempDir()
	downloads := filepath.Join(home, "Downloads")
	if err := os.Mkdir(downloads, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	path, err := ChordChart(s, t.TempDir())
	if err != nil {
		t.Fatalf("chart: %v", err)
	}
	if filepath.Dir(path) != downloads {
		t.Fatalf("chart path %q, want it in %q", path, downloads)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "%PDF") {
		t.Fatalf("not a PDF: %q", string(data[:min(8, len(data))]))
	}
	if len(data) < 2000 {
		t.Fatalf("PDF too small: %d", len(data))
	}
	t.Logf("%s (%d bytes)", path, len(data))
}

func TestChordChartFallsBackToLibraryExports(t *testing.T) {
	s, err := smf.ReadFile(demo)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	home := t.TempDir() // no Downloads folder
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	lib := t.TempDir()
	path, err := ChordChart(s, lib)
	if err != nil {
		t.Fatalf("chart: %v", err)
	}
	if filepath.Dir(path) != config.ExportsDir(lib) {
		t.Fatalf("chart path %q, want %q", path, config.ExportsDir(lib))
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
