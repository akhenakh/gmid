package song

import (
	"math"
	"sort"
)

// RebuildGrid computes the beat and bar times from the tempo and meter maps.
// It is the single source of truth for the roll, metronome and counters.
func (s *Song) RebuildGrid() {
	s.Beats = s.Beats[:0]
	s.Bars = s.Bars[:0]
	if s.PPQ <= 0 {
		return
	}
	if len(s.Sigs) == 0 {
		s.Sigs = []TimeSigChange{{Tick: 0, Num: 4, Den: 4, Sec: 0}}
	}
	sort.SliceStable(s.Sigs, func(i, j int) bool { return s.Sigs[i].Tick < s.Sigs[j].Tick })

	endTick := s.SecToTick(s.DurationSec) + int64(s.PPQ*4)
	if endTick < int64(s.PPQ*4) {
		endTick = int64(s.PPQ * 4)
	}
	tick := int64(0)
	guard := 0
	for tick < endTick && guard < 200000 {
		guard++
		num, den := s.sigAtTick(tick)
		if num <= 0 || den <= 0 {
			break
		}
		s.Bars = append(s.Bars, s.TickToSec(tick))
		beatTick := float64(s.PPQ*4) / float64(den)
		for b := 0; b < num; b++ {
			s.Beats = append(s.Beats, s.TickToSec(tick+int64(float64(b)*beatTick)))
		}
		step := int64(beatTick * float64(num))
		if step <= 0 {
			break
		}
		tick += step
	}
}

// CheckTempoTrust flags songs whose notes do not line up with the file's beat
// grid — the ones Tap Tempo can fix. Returns the fraction that aligned.
func (s *Song) CheckTempoTrust() float64 {
	if len(s.Beats) == 0 {
		return 1
	}
	total, aligned := 0, 0
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if t.IsDrum || t.Excluded {
			continue
		}
		for j := range t.Notes {
			total++
			if total > 600 {
				break
			}
			at := t.Notes[j].StartSec
			// nearest beat
			lo, hi := 0, len(s.Beats)-1
			for lo < hi {
				mid := (lo + hi) / 2
				if s.Beats[mid] < at {
					lo = mid + 1
				} else {
					hi = mid
				}
			}
			best := math.Abs(s.Beats[lo] - at)
			if lo > 0 {
				if d := math.Abs(s.Beats[lo-1] - at); d < best {
					best = d
				}
			}
			if best < 0.06 {
				aligned++
			}
		}
	}
	if total == 0 {
		return 1
	}
	return float64(aligned) / float64(total)
}

func (s *Song) sigAtTick(tick int64) (int, int) {
	if len(s.Sigs) == 0 {
		return 4, 4
	}
	i := sort.Search(len(s.Sigs), func(i int) bool { return s.Sigs[i].Tick > tick }) - 1
	if i < 0 {
		i = 0
	}
	num, den := s.Sigs[i].Num, s.Sigs[i].Den
	if num <= 0 {
		num = 4
	}
	if den <= 0 {
		den = 4
	}
	return num, den
}

// ApplyLeadCut selects the first bar worth showing. With keepDrumCountIn, up to
// two preceding bars of plain drum clicks stay. The file's ticks are untouched.
func (s *Song) ApplyLeadCut(keepDrumCountIn bool) {
	s.Lead = LeadCut{}
	if len(s.Bars) < 2 {
		return
	}
	first := -1
	for i := 0; i < len(s.Bars); i++ {
		start := s.Bars[i]
		end := math.MaxFloat64
		if i+1 < len(s.Bars) {
			end = s.Bars[i+1]
		}
		if s.barHasMusic(start, end) {
			first = i
			break
		}
	}
	if first <= 0 {
		return
	}
	if keepDrumCountIn {
		for back := 1; back <= 2; back++ {
			i := first - back
			if i < 0 || i+1 >= len(s.Bars) {
				break
			}
			if s.barHasMusic(s.Bars[i], s.Bars[i+1]) {
				break
			}
			if s.barHasDrums(s.Bars[i], s.Bars[i+1]) {
				first = i
			}
		}
	}
	if first <= 0 {
		return
	}
	s.Lead = LeadCut{
		Enabled: true,
		Sec:     s.Bars[first],
		Tick:    s.SecToTick(s.Bars[first]),
		Bar:     first,
	}
}

func (s *Song) barHasMusic(start, end float64) bool {
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if t.IsDrum || t.Excluded {
			continue
		}
		for j := range t.Notes {
			if t.Notes[j].StartSec >= start-1e-6 && t.Notes[j].StartSec < end-1e-6 {
				return true
			}
		}
	}
	return false
}

func (s *Song) barHasDrums(start, end float64) bool {
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if !t.IsDrum {
			continue
		}
		for j := range t.Notes {
			if t.Notes[j].StartSec >= start-1e-6 && t.Notes[j].StartSec < end-1e-6 {
				return true
			}
		}
	}
	return false
}

// BarNumberAt returns the displayed bar number (1-based from the visible start)
// for a song time.
func (s *Song) BarNumberAt(sec float64) int {
	bars, _ := s.BarsAndBeats()
	i := sort.Search(len(bars), func(i int) bool { return bars[i] > sec }) - 1
	if i < 0 {
		return 1
	}
	return i + 1
}

// IsShown implements the one visibility rule used by the roll, keys and
// step-through: the eye is on, the track is not muted, and if anything is
// soloed it must be a soloed track.
func (s *Song) IsShown(t *Track) bool {
	if t == nil || !t.Visible || t.Muted || t.Excluded {
		return false
	}
	anySolo := false
	for i := range s.Tracks {
		if s.Tracks[i].Solo {
			anySolo = true
			break
		}
	}
	if !anySolo {
		return true
	}
	return t.Solo
}

// AnySolo reports whether any track is soloed.
func (s *Song) AnySolo() bool {
	for i := range s.Tracks {
		if s.Tracks[i].Solo {
			return true
		}
	}
	return false
}
