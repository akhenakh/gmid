package audio

import (
	"math"
	"testing"

	"github.com/akhenakh/gmid/internal/smf"
)

func TestDecentSamplerPlayback(t *testing.T) {
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	e := New(44100)
	if err := e.LoadDecentPreset("../../assets/decent/demo-piano.dspreset"); err != nil {
		t.Fatalf("decent: %v", err)
	}
	info, ok := e.DecentInfo()
	if !ok || info.Zones == 0 {
		t.Fatalf("info: %+v ok=%v", info, ok)
	}
	e.Load(s)
	e.SetMasterVolume(0.8)
	e.Play(s.OriginSec())
	peak := 0.0
	for i := 0; i < 4*44100/2048; i++ {
		l, r := e.RenderOffline(2048)
		for j := range l {
			v := math.Abs(float64(l[j])) + math.Abs(float64(r[j]))
			if v > peak {
				peak = v
			}
		}
	}
	if peak < 1e-3 {
		t.Fatalf("silent decent render (peak %g)", peak)
	}
	t.Logf("zones=%d peak=%g", info.Zones, peak)
}
