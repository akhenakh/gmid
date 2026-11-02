package hands

import (
	"math"
	"sort"

	"github.com/akhenakh/gmid/internal/song"
	"github.com/akhenakh/gmid/internal/theory"
)

// Options controls the assignment.
type Options struct {
	MaxSpan      int // comfortable one-hand stretch in semitones
	ClusterSec   float64
	RHFingers    int
	LHFingers    int
	CrossingMode string
}

// Defaults are the same starting values as the reference engine.
func Defaults() Options {
	return Options{MaxSpan: 16, ClusterSec: 0.02, RHFingers: 4, LHFingers: 4, CrossingMode: "strict"}
}

type clusterNote struct {
	key   int
	start float64
	idx   int // index into the flat note slice
}

type trackNotes struct {
	track int
	notes []int
}

// AssignSong tags every piano/organ note with a hand.
func AssignSong(s *song.Song, opt Options) {
	if s == nil {
		return
	}
	if opt.MaxSpan == 0 {
		opt = Defaults()
	}
	var tracks []trackNotes
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if t.IsDrum || !theory.IsKeyboardGroup(t.Group) {
			continue
		}
		var ids []int
		for j := range t.Notes {
			t.Notes[j].Hand = ""
			ids = append(ids, j)
		}
		if len(ids) > 0 {
			tracks = append(tracks, trackNotes{track: i, notes: ids})
		}
	}
	if len(tracks) == 0 {
		return
	}

	// Fast path: a clean two-track split with a clear pitch gap.
	if len(tracks) == 2 {
		a, b := tracks[0], tracks[1]
		avgA := averagePitch(s, a.track, a.notes)
		avgB := averagePitch(s, b.track, b.notes)
		if avgA > avgB {
			a, b = b, a
			avgA, avgB = avgB, avgA
		}
		if avgB-avgA >= 3 && !collides(s, a, b) {
			for _, j := range a.notes {
				s.Tracks[a.track].Notes[j].Hand = "L"
				s.Tracks[a.track].Notes[j].Conf = 1
			}
			for _, j := range b.notes {
				s.Tracks[b.track].Notes[j].Hand = "R"
				s.Tracks[b.track].Notes[j].Conf = 1
			}
			return
		}
	}

	var flat []clusterNote
	for _, tn := range tracks {
		for _, j := range tn.notes {
			n := &s.Tracks[tn.track].Notes[j]
			flat = append(flat, clusterNote{key: n.Key, start: n.StartSec, idx: len(flat)})
		}
	}
	sort.SliceStable(flat, func(i, j int) bool { return flat[i].start < flat[j].start })

	// Group by onset.
	var clusters [][]clusterNote
	for _, n := range flat {
		if len(clusters) == 0 || n.start-clusters[len(clusters)-1][0].start > opt.ClusterSec {
			clusters = append(clusters, []clusterNote{n})
		} else {
			clusters[len(clusters)-1] = append(clusters[len(clusters)-1], n)
		}
	}

	lastL, lastR := 48.0, 72.0
	for _, cl := range clusters {
		sort.SliceStable(cl, func(i, j int) bool { return cl[i].key < cl[j].key })
		bestSplit, bestCost, second := 0, math.MaxFloat64, math.MaxFloat64
		for split := 0; split <= len(cl); split++ {
			cost := 0.0
			if split > 0 {
				var sum float64
				minK, maxK := cl[0].key, cl[split-1].key
				for _, n := range cl[:split] {
					sum += float64(n.key)
				}
				avg := sum / float64(split)
				cost += math.Abs(avg-lastL) * 0.6
				if maxK-minK > opt.MaxSpan {
					cost += float64(maxK-minK-opt.MaxSpan) * 3
				}
			}
			if split < len(cl) {
				var sum float64
				minK, maxK := cl[split].key, cl[len(cl)-1].key
				for _, n := range cl[split:] {
					sum += float64(n.key)
				}
				avg := sum / float64(len(cl)-split)
				cost += math.Abs(avg-lastR) * 0.6
				if maxK-minK > opt.MaxSpan {
					cost += float64(maxK-minK-opt.MaxSpan) * 3
				}
			}
			if split > 0 && split < len(cl) && cl[split-1].key > cl[split].key-2 {
				cost += 12
			}
			if cost < bestCost {
				second = bestCost
				bestCost, bestSplit = cost, split
			} else if cost < second {
				second = cost
			}
		}
		conf := float32(0.5)
		if second < math.MaxFloat64 {
			conf = float32(math.Min(1, (second-bestCost)/math.Max(second, 1)))
		}
		var lSum, rSum float64
		var lN, rN int
		for i, cn := range cl {
			hand := "R"
			if i < bestSplit {
				hand = "L"
				lSum += float64(cn.key)
				lN++
			} else {
				rSum += float64(cn.key)
				rN++
			}
			assign(s, tracks, cn, hand, conf)
		}
		if lN > 0 {
			lastL = lSum / float64(lN)
		}
		if rN > 0 {
			lastR = rSum / float64(rN)
		}
	}
}

func assign(s *song.Song, tracks []trackNotes, cn clusterNote, hand string, conf float32) {
	// clusterNote.idx is the flat index; map back via a linear scan is fine for
	// the hand engine's use (only called once per note).
	for _, tn := range tracks {
		for _, j := range tn.notes {
			n := &s.Tracks[tn.track].Notes[j]
			if n.Key == cn.key && math.Abs(n.StartSec-cn.start) < 1e-9 {
				n.Hand = hand
				n.Conf = conf
				return
			}
		}
	}
}

func averagePitch(s *song.Song, track int, ids []int) float64 {
	if len(ids) == 0 {
		return 0
	}
	sum := 0
	for _, j := range ids {
		sum += s.Tracks[track].Notes[j].Key
	}
	return float64(sum) / float64(len(ids))
}

func collides(s *song.Song, a, b trackNotes) bool {
	type noteKey struct {
		tick int64
		key  int
	}
	seen := map[noteKey]bool{}
	for _, j := range a.notes {
		n := s.Tracks[a.track].Notes[j]
		seen[noteKey{n.Tick, n.Key}] = true
	}
	for _, j := range b.notes {
		n := s.Tracks[b.track].Notes[j]
		if seen[noteKey{n.Tick, n.Key}] {
			return true
		}
	}
	return false
}
