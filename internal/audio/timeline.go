package audio

import (
	"sort"

	"github.com/akhenakh/gmid/internal/song"
)

// EventKind is one scheduled MIDI action.
type EventKind uint8

const (
	EvNoteOff EventKind = iota
	EvProgram
	EvCC
	EvNoteOn
)

// Event is one entry of the flattened song timeline. Track is -1 for channel
// automation (always applied).
type Event struct {
	Sec     float64
	Kind    EventKind
	Track   int
	Channel int
	A, B, C int
}

// Timeline is an immutable, time-sorted event list.
type Timeline struct {
	Events   []Event
	Duration float64
	Origin   float64
	TempoAt  func(sec float64) float64
}

// Compile flattens a song into the engine timeline.
func Compile(s *song.Song) *Timeline {
	if s == nil {
		return &Timeline{}
	}
	tl := &Timeline{Origin: s.OriginSec(), Duration: s.DurationSec, TempoAt: s.BPMAt}
	seenProgram := map[int]bool{}

	// Initial program at song start for every channel that has a track.
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if t.Excluded || seenProgram[t.Channel] {
			continue
		}
		seenProgram[t.Channel] = true
		bank := t.BankMSB*128 + t.BankLSB
		tl.Events = append(tl.Events, Event{
			Sec: 0, Kind: EvProgram, Track: -1, Channel: t.Channel,
			A: t.Program, B: bank, C: 0,
		})
	}

	for i := range s.Channels {
		c := &s.Channels[i]
		for _, cc := range c.CC {
			tl.Events = append(tl.Events, Event{Sec: cc.Sec, Kind: EvCC, Track: -1, Channel: c.Number, A: cc.CC, B: cc.Val})
		}
		for _, se := range c.Sustain {
			v := 0
			if se.Down {
				v = 127
			}
			tl.Events = append(tl.Events, Event{Sec: se.Sec, Kind: EvCC, Track: -1, Channel: c.Number, A: 64, B: v})
		}
		for _, pe := range c.Programs {
			tl.Events = append(tl.Events, Event{
				Sec: pe.Sec, Kind: EvProgram, Track: -1, Channel: c.Number,
				A: pe.Patch, B: pe.Bank, C: 0,
			})
		}
	}

	for i := range s.Tracks {
		t := &s.Tracks[i]
		if t.Excluded {
			continue
		}
		for j := range t.Notes {
			n := &t.Notes[j]
			if n.EndSec <= tl.Origin {
				continue
			}
			start := n.StartSec
			if start < tl.Origin {
				start = tl.Origin
			}
			tl.Events = append(tl.Events,
				Event{Sec: start, Kind: EvNoteOn, Track: t.Index, Channel: t.Channel, A: n.Key, B: n.Velocity},
				Event{Sec: n.EndSec, Kind: EvNoteOff, Track: t.Index, Channel: t.Channel, A: n.Key},
			)
		}
	}

	sort.SliceStable(tl.Events, func(i, j int) bool {
		a, b := tl.Events[i], tl.Events[j]
		if a.Sec != b.Sec {
			return a.Sec < b.Sec
		}
		return a.Kind < b.Kind
	})
	return tl
}

// IndexAfter returns the first event index with Sec >= t.
func (tl *Timeline) IndexAfter(t float64) int {
	return sort.Search(len(tl.Events), func(i int) bool { return tl.Events[i].Sec >= t })
}
