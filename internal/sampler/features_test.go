package sampler

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/akhenakh/gmid/internal/dspreset"
)

// writeTestWAV writes a mono 16-bit WAV at 48 kHz.
func writeTestWAV(t *testing.T, path string, samples []float32) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	const rate = 48000
	dataLen := uint32(len(samples) * 2)
	write := func(v any) { _ = binary.Write(f, binary.LittleEndian, v) }
	_, _ = f.WriteString("RIFF")
	write(uint32(36 + dataLen))
	_, _ = f.WriteString("WAVEfmt ")
	write(uint32(16))
	write(uint16(1))
	write(uint16(1))
	write(uint32(rate))
	write(uint32(rate * 2))
	write(uint16(2))
	write(uint16(16))
	_, _ = f.WriteString("data")
	write(dataLen)
	for _, s := range samples {
		write(int16(math.Max(-1, math.Min(1, float64(s))) * 32767))
	}
}

func testInstrument(t *testing.T, xml string) (*Instrument, string) {
	t.Helper()
	dir := t.TempDir()
	sample := filepath.Join(dir, "tone.wav")
	tone := make([]float32, 24000)
	for i := range tone {
		tone[i] = 0.4 * float32(math.Sin(2*math.Pi*440*float64(i)/48000))
	}
	writeTestWAV(t, sample, tone)
	preset := filepath.Join(dir, "test.dspreset")
	body := fmt.Sprintf(`<?xml version="1.0"?><DecentSampler>%s</DecentSampler>`, xml)
	body = replaceAll(body, "SAMPLE", sample)
	if err := os.WriteFile(preset, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := dspreset.Load(preset)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := Load(p, 48000)
	if err != nil {
		t.Fatal(err)
	}
	return inst, sample
}

func replaceAll(s, old, new string) string {
	for {
		i := indexOf(s, old)
		if i < 0 {
			return s
		}
		s = s[:i] + new + s[i+len(old):]
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestEnvelopeCurveShapes(t *testing.T) {
	if envelopeCurve(0, 100) != 0 || envelopeCurve(1, 100) != 1 {
		t.Fatal("curve endpoints")
	}
	midExp := envelopeCurve(0.5, 100)
	midLog := envelopeCurve(0.5, -100)
	if !(midExp < 0.5 && midLog > 0.5) {
		t.Fatalf("exp/log midpoints: %g %g", midExp, midLog)
	}
	if envelopeCurve(0.25, 0) != 0.25 {
		t.Fatal("linear curve")
	}
}

func TestSampleDelay(t *testing.T) {
	inst, _ := testInstrument(t, `<groups ampEnvEnabled="false"><group><sample path="SAMPLE" rootNote="60" delay="0.01" /></group></groups>`)
	inst.NoteOn(60, 100, 0)
	l := make([]float32, 700)
	inst.Render(l, make([]float32, 700))
	for i := 0; i < 470; i++ {
		if math.Abs(float64(l[i])) > 1e-6 {
			t.Fatalf("delay leaked at %d: %g", i, l[i])
		}
	}
	peak := 0.0
	for i := 500; i < len(l); i++ {
		peak = math.Max(peak, math.Abs(float64(l[i])))
	}
	if peak < 1e-3 {
		t.Fatal("delayed voice never sounded")
	}
}

func TestLoopCrossfadeSmoothsWrap(t *testing.T) {
	dir := t.TempDir()
	ramp := make([]float32, 3000)
	for i := range ramp {
		ramp[i] = float32(i) / float32(len(ramp)-1)
	}
	sample := filepath.Join(dir, "ramp.wav")
	writeTestWAV(t, sample, ramp)
	preset := filepath.Join(dir, "loop.dspreset")
	xml := fmt.Sprintf(`<DecentSampler><groups ampEnvEnabled="false"><group><sample path=%q rootNote="60" loopEnabled="true" loopStart="500" loopEnd="1500" loopCrossfade="100" /></group></groups></DecentSampler>`, sample)
	if err := os.WriteFile(preset, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := dspreset.Load(preset)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := Load(p, 48000)
	if err != nil {
		t.Fatal(err)
	}
	inst.NoteOn(60, 100, 0)
	l := make([]float32, 4000)
	inst.Render(l, make([]float32, 4000))
	worst := 0.0
	for i := 1; i < len(l); i++ {
		worst = math.Max(worst, math.Abs(float64(l[i]-l[i-1])))
	}
	if worst > 0.05 {
		t.Fatalf("loop splice jumps by %g", worst)
	}
}

func TestTuningRespectsKeyTrack(t *testing.T) {
	plain, _ := testInstrument(t, `<groups><group pitchKeyTrack="0"><sample path="SAMPLE" rootNote="60" tuning="12" /></group></groups>`)
	plain.NoteOn(60, 100, 0)
	if got := plain.voices[0].step; math.Abs(got-1) > 1e-6 {
		t.Fatalf("normal tuning step = %g, want 1 (tuning follows key track)", got)
	}
	full, _ := testInstrument(t, `<groups><group pitchKeyTrack="0"><sample path="SAMPLE" rootNote="60" tuning="12" tuningIgnoresKeyTrack="true" /></group></groups>`)
	full.NoteOn(60, 100, 0)
	if got := full.voices[0].step; math.Abs(got-2) > 1e-6 {
		t.Fatalf("ignoring key track step = %g, want 2", got)
	}
}

func TestTriggerModes(t *testing.T) {
	inst, _ := testInstrument(t, `<groups><group>
		<sample path="SAMPLE" rootNote="60" loNote="60" hiNote="60" trigger="first" />
		<sample path="SAMPLE" rootNote="62" loNote="62" hiNote="62" trigger="legato" />
		<sample path="SAMPLE" rootNote="64" loNote="64" hiNote="64" trigger="continuous" />
	</group></groups>`)
	inst.NoteOn(60, 100, 0)
	if inst.VoiceCount() != 1 {
		t.Fatalf("first trigger: %d voices", inst.VoiceCount())
	}
	inst.NoteOn(64, 100, 0)
	if inst.VoiceCount() != 2 {
		t.Fatalf("continuous trigger: %d voices", inst.VoiceCount())
	}
	inst.Reset()
	inst.NoteOn(62, 100, 0)
	if inst.VoiceCount() != 0 {
		t.Fatalf("legato trigger without a held note: %d voices", inst.VoiceCount())
	}
}

func TestCCTriggeredSample(t *testing.T) {
	inst, _ := testInstrument(t, `<groups><group>
		<sample path="SAMPLE" rootNote="60" onLoCC64="100" onHiCC64="127" />
	</group></groups>`)
	inst.NoteOn(60, 100, 0)
	if inst.VoiceCount() != 0 {
		t.Fatalf("CC-only zone played on note-on: %d voices", inst.VoiceCount())
	}
	inst.Control(0, 64, 127)
	if inst.VoiceCount() != 1 {
		t.Fatalf("CC trigger did not start: %d voices", inst.VoiceCount())
	}
	inst.Reset()
	inst.Control(0, 64, 0)
	if inst.VoiceCount() != 0 {
		t.Fatalf("CC trigger fired below its range: %d voices", inst.VoiceCount())
	}
}

func TestReleaseTriggerDecay(t *testing.T) {
	inst, _ := testInstrument(t, `<groups><group>
		<sample path="SAMPLE" rootNote="60" loNote="60" hiNote="60" release="0.01" />
		<sample path="SAMPLE" rootNote="60" loNote="60" hiNote="60" trigger="release" ampEnvEnabled="false" releaseTriggerDecay="12dB" />
	</group></groups>`)
	inst.NoteOn(60, 100, 0)
	inst.Render(make([]float32, 48000), make([]float32, 48000))
	inst.NoteOff(60, 0)
	var release *voice
	for _, v := range inst.voices {
		if v.z.sample.Trigger == "release" {
			release = v
		}
	}
	if release == nil {
		t.Fatal("release trigger did not fire")
	}
	want := math.Pow(10, -12.0/20)
	got := release.gain / release.z.gain
	if math.Abs(got-want)/want > 0.05 {
		t.Fatalf("release gain %.3f, want %.3f", got, want)
	}
}

func renderPeak(t *testing.T, xml string, frames int) float64 {
	t.Helper()
	inst, _ := testInstrument(t, xml)
	inst.NoteOn(60, 100, 0)
	l := make([]float32, frames)
	inst.Render(l, make([]float32, frames))
	peak := 0.0
	for _, v := range l {
		peak = math.Max(peak, math.Abs(float64(v)))
	}
	return peak
}

func TestGainEffect(t *testing.T) {
	body := `<groups ampEnvEnabled="false"><group><sample path="SAMPLE" rootNote="60" /></group></groups>`
	plain := renderPeak(t, body, 4800)
	cut := renderPeak(t, body+`<effects><effect type="gain" level="-6" /></effects>`, 4800)
	if ratio := cut / plain; math.Abs(ratio-0.5) > 0.02 {
		t.Fatalf("gain ratio = %.3f, want 0.5", ratio)
	}
}

func TestFilterEffects(t *testing.T) {
	body := `<groups ampEnvEnabled="false"><group><sample path="SAMPLE" rootNote="60" /></group></groups>`
	plain := renderPeak(t, body, 4800)
	lp := renderPeak(t, body+`<effects><effect type="lowpass" frequency="8000" resonance="0.7" /></effects>`, 4800)
	hp := renderPeak(t, body+`<effects><effect type="highpass" frequency="8000" resonance="0.7" /></effects>`, 4800)
	if plain-lp > 0.05 {
		t.Fatalf("lowpass changed a 440 Hz tone: %.3f vs %.3f", lp, plain)
	}
	if hp > plain*0.2 {
		t.Fatalf("highpass did not attenuate: %.3f vs %.3f", hp, plain)
	}
}

func TestReverbTail(t *testing.T) {
	inst, _ := testInstrument(t, `<groups ampEnvEnabled="false"><group><sample path="SAMPLE" rootNote="60" end="2400" /></group></groups>
		<effects><effect type="reverb" roomSize="0.9" damping="0.2" wetLevel="1" /></effects>`)
	inst.NoteOn(60, 100, 0)
	frames := 48000
	l := make([]float32, frames)
	inst.Render(l, make([]float32, frames))
	var sum float64
	start := frames - 12000
	for i := start; i < frames; i++ {
		sum += float64(l[i]) * float64(l[i])
	}
	rms := math.Sqrt(sum / float64(frames-start))
	if rms < 1e-4 {
		t.Fatalf("no reverb tail after the sample ended (rms %g)", rms)
	}
}
