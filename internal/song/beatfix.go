package song

import "math"

// ApplyBeatFix replaces only the beat layer with a tapped steady grid. Note
// times, the tempo map and the file are untouched.
func (s *Song) ApplyBeatFix(period, anchor float64, bpb int) {
	if period <= 0 {
		return
	}
	if bpb < 1 || bpb > 16 {
		bpb = 4
	}
	s.BeatFixed = true
	s.BeatPeriod = period
	s.BeatAnchor = anchor
	s.BeatBpb = bpb
	s.TempoSuspect = false
	s.rebuildFixedGrid()
}

// ClearBeatFix restores the file's own grid.
func (s *Song) ClearBeatFix() {
	s.BeatFixed = false
	s.RebuildGrid()
}

// EffectivePeriod returns the beat period used by the metronome and counters.
func (s *Song) EffectivePeriod(at float64) float64 {
	if s.BeatFixed && s.BeatPeriod > 0 {
		return s.BeatPeriod
	}
	bars, beats := s.BarsAndBeats()
	_ = bars
	if len(beats) > 1 {
		i := 0
		for i+1 < len(beats) && beats[i+1] <= at {
			i++
		}
		if i+1 < len(beats) {
			return beats[i+1] - beats[i]
		}
	}
	return 60.0 / math.Max(20, s.BPMAt(at))
}

// EffectiveBPM is the beat-grid tempo shown in the top bar.
func (s *Song) EffectiveBPM(at float64) float64 {
	p := s.EffectivePeriod(at)
	if p <= 0 {
		return 120
	}
	return 60 / p
}

func (s *Song) rebuildFixedGrid() {
	s.Beats = s.Beats[:0]
	s.Bars = s.Bars[:0]
	p := s.BeatPeriod
	anchor := s.BeatAnchor
	if p <= 0 {
		return
	}
	start := anchor
	for start-p >= s.OriginSec()-1e-9 {
		start -= p
	}
	barsStart := anchor
	for barsStart-float64(s.BeatBpb)*p >= s.OriginSec()-1e-9 {
		barsStart -= float64(s.BeatBpb) * p
	}
	end := s.DurationSec + p
	for t := start; t <= end; t += p {
		s.Beats = append(s.Beats, t)
	}
	for t := barsStart; t <= end; t += float64(s.BeatBpb) * p {
		s.Bars = append(s.Bars, t)
	}
}
