package song

import "testing"

func TestApplyBeatFix(t *testing.T) {
	s := &Song{DurationSec: 10, PPQ: 480, Tempo: []TempoChange{{BPM: 120}}}
	s.Sigs = []TimeSigChange{{Tick: 0, Num: 4, Den: 4}}
	s.RebuildGrid()
	s.ApplyBeatFix(0.5, 1.0, 4)
	if !s.BeatFixed {
		t.Fatal("not fixed")
	}
	if len(s.Beats) < 4 || len(s.Bars) < 2 {
		t.Fatalf("grid %d/%d", len(s.Beats), len(s.Bars))
	}
	for i := 1; i < len(s.Beats); i++ {
		if d := s.Beats[i] - s.Beats[i-1]; d < 0.499 || d > 0.501 {
			t.Fatalf("beat spacing %v", d)
		}
	}
	for i := 1; i < len(s.Bars); i++ {
		if d := s.Bars[i] - s.Bars[i-1]; d < 1.999 || d > 2.001 {
			t.Fatalf("bar spacing %v", d)
		}
	}
	if bpm := s.EffectiveBPM(1); bpm < 119 || bpm > 121 {
		t.Fatalf("bpm %v", bpm)
	}
	s.ClearBeatFix()
	if s.BeatFixed {
		t.Fatal("fix not cleared")
	}
}
