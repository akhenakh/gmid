package export

import (
	"os"
	"strings"
	"testing"

	"github.com/akhenakh/gmid/internal/smf"
)

func TestChordChart(t *testing.T) {
	s, err := smf.ReadFile(demo)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	path, err := ChordChart(s, t.TempDir())
	if err != nil {
		t.Fatalf("chart: %v", err)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
