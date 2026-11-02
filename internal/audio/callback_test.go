package audio

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/akhenakh/gmid/internal/smf"
)

// TestCallbackPathMatchesOffline guards the device callback's buffer use: the
// callback used to pass its scratch buffers as the mix destination, so the
// synth overwrote samples already mixed for the block and the output crackled.
// Offline renders use fresh buffers and therefore stayed clean.
func TestCallbackPathMatchesOffline(t *testing.T) {
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	build := func(decent bool) *Engine {
		e := New(44100)
		if err := e.LoadSoundFont("../../assets/soundfonts/GeneralUser-GS.sf2"); err != nil {
			t.Skip(err)
		}
		if decent {
			if err := e.LoadDecentPreset("../../assets/decent/demo-piano.dspreset"); err != nil {
				t.Fatal(err)
			}
		}
		e.Load(s)
		e.Play(s.OriginSec())
		return e
	}
	for _, decent := range []bool{false, true} {
		a, b := build(decent), build(decent)
		const n = 1024
		buf := make([]byte, n*2*4)
		worst := 0.0
		for blk := 0; blk < 60; blk++ {
			lo, ro := a.RenderOffline(n)
			b.onAudio(buf, n)
			for i := 0; i < n; i++ {
				lc := math.Float32frombits(binary.LittleEndian.Uint32(buf[i*8:]))
				rc := math.Float32frombits(binary.LittleEndian.Uint32(buf[i*8+4:]))
				worst = math.Max(worst, math.Abs(float64(lc-lo[i])))
				worst = math.Max(worst, math.Abs(float64(rc-ro[i])))
			}
		}
		if worst > 1e-5 {
			t.Fatalf("decent=%v: callback differs from offline render by %g", decent, worst)
		}
	}
}
