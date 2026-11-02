package sampler

import (
	"math"
	"testing"

	"github.com/akhenakh/gmid/internal/dspreset"
)

func TestDemoInstrument(t *testing.T) {
	p, err := dspreset.Load("../../assets/decent/demo-piano.dspreset")
	if err != nil {
		t.Skipf("demo preset missing: %v", err)
	}
	inst, err := Load(p, 48000)
	if err != nil {
		t.Fatal(err)
	}
	info := inst.Info()
	if info.Groups != 1 || info.Zones < 5 {
		t.Fatalf("info: %+v", info)
	}
	inst.NoteOn(60, 100, 0)
	l := make([]float32, 4800)
	r := make([]float32, 4800)
	inst.Render(l, r)
	peak := 0.0
	for i := range l {
		if v := math.Abs(float64(l[i])); v > peak {
			peak = v
		}
	}
	if peak < 1e-3 {
		t.Fatalf("silent sampler (peak %g)", peak)
	}
	inst.NoteOff(60, 0)
	for i := 0; i < 30; i++ {
		inst.Render(make([]float32, 4800), make([]float32, 4800))
	}
	if inst.VoiceCount() != 0 {
		t.Fatalf("voice not released: %d", inst.VoiceCount())
	}
}

func TestPitchTracking(t *testing.T) {
	p, err := dspreset.Load("../../assets/decent/demo-piano.dspreset")
	if err != nil {
		t.Skipf("demo preset missing: %v", err)
	}
	inst, err := Load(p, 48000)
	if err != nil {
		t.Fatal(err)
	}
	// A note one octave above the root should consume the sample in half the time.
	inst.NoteOn(72, 100, 0)
	buf := make([]float32, 48000)
	inst.Render(buf, make([]float32, 48000))
	if inst.VoiceCount() == 0 {
		t.Fatal("voice ended immediately")
	}
}
