// Command mkdecent generates the tiny DecentSampler demo instrument shipped
// with gmid. It is a development tool, not part of the app.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

func main() {
	dir := "assets/decent"
	sampleDir := filepath.Join(dir, "samples")
	if err := os.MkdirAll(sampleDir, 0o755); err != nil {
		panic(err)
	}
	roots := []int{36, 48, 60, 72, 84}
	for _, root := range roots {
		path := filepath.Join(sampleDir, fmt.Sprintf("piano-%d.wav", root))
		if err := writeWAV(path, renderNote(root)); err != nil {
			panic(err)
		}
		fmt.Println(path)
	}
	preset := `<?xml version="1.0" encoding="UTF-8"?>
<DecentSampler minVersion="1.0.0">
<ui width="640" height="240" layoutMode="relative" bgMode="top_left">
  <tab name="main">
    <labeled-knob x="40" y="60" width="90" label="Attack" valueType="float" minValue="0" maxValue="0.2" value="0.004">
      <binding type="amp" level="instrument" position="0" parameter="ENV_ATTACK" />
    </labeled-knob>
    <labeled-knob x="150" y="60" width="90" label="Release" valueType="float" minValue="0" maxValue="2" value="0.3">
      <binding type="amp" level="instrument" position="0" parameter="ENV_RELEASE" />
    </labeled-knob>
    <labeled-knob x="260" y="60" width="90" label="Tone" valueType="float" minValue="0" maxValue="1" value="0.85">
      <binding type="effect" level="instrument" position="0" parameter="FX_FILTER_FREQUENCY" />
    </labeled-knob>
  </tab>
</ui>
<groups volume="-4dB" attack="0.004" decay="2.4" sustain="0.0" release="0.35">
  <group ampVelTrack="0.85">
    <sample path="samples/piano-36.wav" rootNote="36" loNote="21" hiNote="41" />
    <sample path="samples/piano-48.wav" rootNote="48" loNote="42" hiNote="53" />
    <sample path="samples/piano-60.wav" rootNote="60" loNote="54" hiNote="65" />
    <sample path="samples/piano-72.wav" rootNote="72" loNote="66" hiNote="77" />
    <sample path="samples/piano-84.wav" rootNote="84" loNote="78" hiNote="108" />
  </group>
</groups>
<effects>
  <effect type="lowpass" frequency="12000" />
  <effect type="reverb" wetLevel="0.12" />
</effects>
</DecentSampler>
`
	if err := os.WriteFile(filepath.Join(dir, "demo-piano.dspreset"), []byte(preset), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("assets/decent/demo-piano.dspreset")
}

func renderNote(root int) []float32 {
	const rate = 48000
	const seconds = 2.6
	n := int(rate * seconds)
	out := make([]float32, n)
	freq := 440 * math.Pow(2, float64(root-69)/12)
	// Additive piano-like partials with inharmonicity and per-partial decay.
	partials := []struct {
		ratio, amp, decay, detune float64
	}{
		{1, 0.65, 2.2, 0},
		{2.002, 0.28, 1.5, 0.4},
		{3.008, 0.15, 1.0, -0.3},
		{4.018, 0.09, 0.7, 0.6},
		{5.03, 0.05, 0.5, -0.5},
		{6.05, 0.03, 0.35, 0.2},
		{8.1, 0.02, 0.25, -0.2},
	}
	for i := 0; i < n; i++ {
		t := float64(i) / rate
		attack := math.Min(1, t/0.004)
		var s float64
		for _, p := range partials {
			env := math.Exp(-t / p.decay)
			s += p.amp * env * math.Sin(2*math.Pi*freq*p.ratio*t+p.detune)
		}
		s *= attack * 0.45
		// Gentle stereo width.
		left := s * (1 + 0.04*math.Sin(2*math.Pi*0.6*t))
		right := s * (1 - 0.04*math.Sin(2*math.Pi*0.6*t))
		out[i] = float32((left + right) * 0.5)
	}
	return out
}

func writeWAV(path string, samples []float32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	const channels = 1
	const rate = 48000
	dataLen := uint32(len(samples) * 2)
	write := func(v any) { _ = binary.Write(f, binary.LittleEndian, v) }
	_, _ = f.WriteString("RIFF")
	write(uint32(36 + dataLen))
	_, _ = f.WriteString("WAVEfmt ")
	write(uint32(16))
	write(uint16(1))
	write(uint16(channels))
	write(uint32(rate))
	write(uint32(rate * channels * 2))
	write(uint16(channels * 2))
	write(uint16(16))
	_, _ = f.WriteString("data")
	write(dataLen)
	for _, s := range samples {
		v := int16(s * 32767)
		write(v)
	}
	return nil
}
