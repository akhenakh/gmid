package dspreset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExtendedAttributes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.dspreset")
	xml := `<?xml version="1.0"?>
<DecentSampler>
  <groups globalPan="-20" tuningIgnoresKeyTrack="true">
    <group groupPan="10" releaseTriggerDecay="3dB">
      <sample path="a.wav" rootNote="60" pan="5" tuning="2" loopEnabled="true"
        loopStart="100" loopEnd="900" loopCrossfade="50" loopCrossfadeMode="linear"
        trigger="legato" delay="0.25" delayUnit="beats"
        attackCurve="20" decayCurve="-30" releaseCurve="40"
        onLoCC64="90" onHiCC64="127" />
    </group>
  </groups>
</DecentSampler>`
	if err := os.WriteFile(path, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	s := p.Groups[0].Samples[0]
	if s.Pan != 5-20+10 {
		t.Fatalf("pan = %v, want -5", s.Pan)
	}
	if s.Tuning != 2 {
		t.Fatalf("tuning = %v", s.Tuning)
	}
	if !s.TuningIgnoresKeyTrack {
		t.Fatal("tuningIgnoresKeyTrack not inherited")
	}
	if s.LoopCrossfade != 50 || s.LoopCrossfadeMode != "linear" {
		t.Fatalf("crossfade = %d %q", s.LoopCrossfade, s.LoopCrossfadeMode)
	}
	if s.Trigger != "legato" || s.Delay != 0.25 || s.DelayUnit != "beats" {
		t.Fatalf("trigger/delay = %q %v %q", s.Trigger, s.Delay, s.DelayUnit)
	}
	if s.Amp.AttackCurve != 20 || s.Amp.DecayCurve != -30 || s.Amp.ReleaseCurve != 40 {
		t.Fatalf("curves = %v %v %v", s.Amp.AttackCurve, s.Amp.DecayCurve, s.Amp.ReleaseCurve)
	}
	if s.ReleaseDecayDB != 3 {
		t.Fatalf("release decay = %v dB", s.ReleaseDecayDB)
	}
	if r, ok := s.OnCCFilters[64]; !ok || r != [2]int{90, 127} {
		t.Fatalf("onCC = %v", s.OnCCFilters)
	}
}

func TestEnvelopeCurveDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "d.dspreset")
	xml := `<DecentSampler><groups><group><sample path="a.wav" rootNote="60" /></group></groups></DecentSampler>`
	if err := os.WriteFile(path, []byte(xml), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	e := p.Groups[0].Samples[0].Amp
	if e.AttackCurve != -100 || e.DecayCurve != 100 || e.ReleaseCurve != 100 {
		t.Fatalf("default curves = %v %v %v", e.AttackCurve, e.DecayCurve, e.ReleaseCurve)
	}
}
