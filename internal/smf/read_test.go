package smf

import (
	"testing"
)

const demo = "../../assets/demo/Scott Joplin - Entertainer.mid"

func TestParseDemo(t *testing.T) {
	s, err := ReadFile(demo)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	if s.NoteCount == 0 {
		t.Fatal("no notes parsed")
	}
	if len(s.Tracks) == 0 {
		t.Fatal("no tracks parsed")
	}
	if len(s.Bars) < 2 || len(s.Beats) < len(s.Bars) {
		t.Fatalf("bad grid: %d bars, %d beats", len(s.Bars), len(s.Beats))
	}
	if s.DurationSec <= 0 {
		t.Fatalf("bad duration %v", s.DurationSec)
	}
	if s.FirstNoteSec <= 0 {
		t.Fatalf("bad first note %v", s.FirstNoteSec)
	}
	for ti := range s.Tracks {
		tr := &s.Tracks[ti]
		if tr.Color == "" {
			t.Errorf("track %d has no colour", ti)
		}
		for _, n := range tr.Notes {
			if n.EndSec < n.StartSec || n.Duration() <= 0 {
				t.Fatalf("bad note %+v", n)
			}
			if n.Velocity <= 0 || n.Velocity > 127 {
				t.Fatalf("bad velocity %d", n.Velocity)
			}
		}
	}
	if s.Identity == "" {
		t.Fatal("no identity")
	}
}

func TestWriteRoundTrip(t *testing.T) {
	s, err := ReadFile(demo)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	data, err := Write(s)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	back, err := Parse(data, s.Path)
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if back.NoteCount != s.NoteCount {
		t.Fatalf("notes %d != %d", back.NoteCount, s.NoteCount)
	}
	if len(back.Tracks) != len(s.Tracks) {
		t.Fatalf("tracks %d != %d", len(back.Tracks), len(s.Tracks))
	}
	if len(back.Tempo) == 0 || len(s.Tempo) == 0 {
		t.Fatal("tempo map lost")
	}
	if back.Tempo[0].BPM != s.Tempo[0].BPM {
		t.Fatalf("tempo %v != %v", back.Tempo[0].BPM, s.Tempo[0].BPM)
	}
	for i := range s.Tracks {
		if len(back.Tracks[i].Notes) != len(s.Tracks[i].Notes) {
			t.Fatalf("track %d notes %d != %d", i, len(back.Tracks[i].Notes), len(s.Tracks[i].Notes))
		}
	}
}

func TestTickSecondsRoundTrip(t *testing.T) {
	s, err := ReadFile(demo)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	for _, tick := range []int64{0, 1, 480, 960, 1920, s.SecToTick(s.DurationSec)} {
		sec := s.TickToSec(tick)
		back := s.SecToTick(sec)
		if back < tick-2 || back > tick+2 {
			t.Fatalf("tick %d -> %v -> %d", tick, sec, back)
		}
	}
}
