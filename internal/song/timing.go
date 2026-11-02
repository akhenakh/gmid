package song

import "sort"

// TickToSec converts an absolute tick to song seconds.
func (s *Song) TickToSec(tick int64) float64 {
	if s.PPQ <= 0 {
		if s.TickSec > 0 {
			return float64(tick) * s.TickSec
		}
		return 0
	}
	if len(s.Tempo) == 0 {
		return float64(tick) / float64(s.PPQ) * 0.5
	}
	i := sort.Search(len(s.Tempo), func(i int) bool { return s.Tempo[i].Tick > tick }) - 1
	if i < 0 {
		i = 0
	}
	t := s.Tempo[i]
	if t.BPM <= 0 {
		return t.Sec
	}
	return t.Sec + float64(tick-t.Tick)/float64(s.PPQ)*(60.0/t.BPM)
}

// SecToTick converts song seconds to the nearest tick.
func (s *Song) SecToTick(sec float64) int64 {
	if s.PPQ <= 0 {
		if s.TickSec > 0 {
			return int64(sec / s.TickSec)
		}
		return 0
	}
	if len(s.Tempo) == 0 {
		return int64(sec * 2 * float64(s.PPQ))
	}
	i := sort.Search(len(s.Tempo), func(i int) bool { return s.Tempo[i].Sec > sec }) - 1
	if i < 0 {
		i = 0
	}
	t := s.Tempo[i]
	if t.BPM <= 0 {
		return t.Tick
	}
	return t.Tick + int64((sec-t.Sec)*t.BPM/60.0*float64(s.PPQ))
}

// TempoAt returns the tempo change in force at a tick.
func (s *Song) TempoAt(tick int64) TempoChange {
	if len(s.Tempo) == 0 {
		return TempoChange{BPM: 120}
	}
	i := sort.Search(len(s.Tempo), func(i int) bool { return s.Tempo[i].Tick > tick }) - 1
	if i < 0 {
		i = 0
	}
	return s.Tempo[i]
}

// TicksPerBeat returns the PPQ (or a safe default).
func (s *Song) TicksPerBeat() int64 {
	if s.PPQ > 0 {
		return int64(s.PPQ)
	}
	return 480
}
