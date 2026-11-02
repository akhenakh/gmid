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

func TestMakeupGainAndUserGain(t *testing.T) {
	p, err := dspreset.Load("../../assets/decent/demo-piano.dspreset")
	if err != nil {
		t.Skipf("demo preset missing: %v", err)
	}
	base, err := Load(p, 44100)
	if err != nil {
		t.Fatal(err)
	}
	if info := base.Info(); info.GainDB <= 0 || info.RefPeak <= 0 {
		t.Fatalf("no level matching: %+v", info)
	}
	base.NoteOn(60, 100, 0)
	l := make([]float32, 22050)
	base.Render(l, make([]float32, 22050))
	basePeak := 0.0
	for _, v := range l {
		if a := math.Abs(float64(v)); a > basePeak {
			basePeak = a
		}
	}
	if basePeak < 0.1 || basePeak > 0.4 {
		t.Fatalf("level-matched peak = %g", basePeak)
	}

	boosted, err := Load(p, 44100)
	if err != nil {
		t.Fatal(err)
	}
	boosted.SetGainAdjust(6)
	boosted.NoteOn(60, 100, 0)
	l2 := make([]float32, 22050)
	boosted.Render(l2, make([]float32, 22050))
	boostPeak := 0.0
	for _, v := range l2 {
		if a := math.Abs(float64(v)); a > boostPeak {
			boostPeak = a
		}
	}
	ratio := boostPeak / basePeak
	if ratio < 1.8 || ratio > 2.2 {
		t.Fatalf("+6 dB ratio = %.2f", ratio)
	}
}
