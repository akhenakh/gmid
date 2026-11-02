package audio

import (
	"math"
	"testing"

	"github.com/akhenakh/gmid/internal/smf"
)

func peakOf(l, r []float32) float64 {
	peak := 0.0
	for i := range l {
		if a := math.Abs(float64(l[i])); a > peak {
			peak = a
		}
		if a := math.Abs(float64(r[i])); a > peak {
			peak = a
		}
	}
	return peak
}

func TestLiveNotesWhilePaused(t *testing.T) {
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	e := New(44100)
	if err := e.LoadSoundFont("../../assets/soundfonts/GeneralUser-GS.sf2"); err != nil {
		t.Skip(err)
	}
	e.Load(s)
	e.Pause()
	e.NoteOn(15, 60, 110)
	peak := 0.0
	for i := 0; i < 4; i++ {
		l, r := e.RenderOffline(1024)
		peak = math.Max(peak, peakOf(l, r))
	}
	if peak < 1e-3 {
		t.Fatalf("live notes are silent while paused (peak %g)", peak)
	}
	e.NoteOff(15, 60)
	for i := 0; i < 40; i++ {
		e.RenderOffline(1024)
	}
	l, r := e.RenderOffline(1024)
	if p := peakOf(l, r); p > 1e-3 {
		t.Fatalf("release tail still sounding after 40 blocks (peak %g)", p)
	}
}

func TestDecentLiveNotesWhilePaused(t *testing.T) {
	e := New(44100)
	if err := e.LoadSoundFont("../../assets/soundfonts/GeneralUser-GS.sf2"); err != nil {
		t.Skip(err)
	}
	if err := e.LoadDecentPreset("../../assets/decent/demo-piano.dspreset"); err != nil {
		t.Skip(err)
	}
	e.Pause()
	e.NoteOn(15, 60, 110)
	peak := 0.0
	for i := 0; i < 4; i++ {
		l, r := e.RenderOffline(1024)
		peak = math.Max(peak, peakOf(l, r))
	}
	if peak < 1e-3 {
		t.Fatalf("decent live notes are silent while paused (peak %g)", peak)
	}
}
