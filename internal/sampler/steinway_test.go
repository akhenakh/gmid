package sampler

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akhenakh/gmid/internal/dspreset"
)

// TestExternalSteinway probes a real DecentSampler library when it is installed
// locally. It is skipped on machines without it. The library splits its round
// robins into separate groups, so this guards against layering every take.
func TestExternalSteinway(t *testing.T) {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config/DecentSampler/Sample Libraries/Steinway Grand  (DS)/Steinway Grand.dspreset")
	if _, err := os.Stat(path); err != nil {
		t.Skip("Steinway library not installed")
	}
	p, err := dspreset.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := Load(p, 44100)
	if err != nil {
		t.Fatal(err)
	}
	info := inst.Info()
	t.Logf("groups=%d zones=%d buffers=%d gain=%.2fdB ref=%.3f", info.Groups, info.Zones, info.Buffers, info.GainDB, info.RefPeak)
	inst.NoteOn(60, 100, 0)
	if got := inst.VoiceCount(); got != 1 {
		t.Fatalf("round robin layered %d voices, want 1", got)
	}
	l := make([]float32, 44100)
	inst.Render(l, make([]float32, 44100))
	peak := 0.0
	for _, v := range l {
		if a := math.Abs(float64(v)); a > peak {
			peak = a
		}
	}
	if peak < 0.08 {
		t.Fatalf("quiet after level matching: peak %g", peak)
	}
}
