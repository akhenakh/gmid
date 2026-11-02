package song

import "testing"

func TestLeadCutSkipsDrumBars(t *testing.T) {
	s := &Song{PPQ: 480, DurationSec: 8}
	s.Tempo = []TempoChange{{Tick: 0, BPM: 120}}
	s.Sigs = []TimeSigChange{{Tick: 0, Num: 4, Den: 4}}
	s.Tracks = []Track{
		{Index: 0, Name: "Drums", IsDrum: true, Channel: 9, Visible: true,
			Notes: []Note{{Key: 36, Tick: 0, DurTicks: 480, StartSec: 0, EndSec: 0.5}}},
		{Index: 1, Name: "Piano", Channel: 0, Visible: true,
			Notes: []Note{{Key: 60, Tick: 1920, DurTicks: 480, StartSec: 2, EndSec: 2.5}}},
	}
	s.RebuildGrid()
	s.ApplyLeadCut(false)
	if !s.Lead.Enabled {
		t.Fatal("lead cut not applied")
	}
	if s.OriginSec() < 1.99 || s.OriginSec() > 2.01 {
		t.Fatalf("origin %v", s.OriginSec())
	}
	if len(s.EffectiveBeats()) == 0 {
		t.Fatal("no beats after cut")
	}
}
