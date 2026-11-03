package notation

import (
	"testing"

	"github.com/akhenakh/gmid/internal/song"
)

func TestFifthsFor(t *testing.T) {
	cases := []struct {
		tonic int
		minor bool
		want  int
	}{
		{0, false, 0},  // C major
		{7, false, 1},  // G major
		{5, false, -1}, // F major
		{9, true, 0},   // A minor
		{4, true, 1},   // E minor
		{10, true, -5}, // Bb minor
	}
	for _, c := range cases {
		if got := fifthsFor(c.tonic, c.minor); got != c.want {
			t.Errorf("fifthsFor(%d,%v) = %d, want %d", c.tonic, c.minor, got, c.want)
		}
	}
}

func TestSpellUsesKeySignature(t *testing.T) {
	cases := []struct {
		key    int
		fifths int
		letter int
		alter  int
	}{
		{61, 2, 0, 1},   // C# in D major
		{61, -1, 1, -1}, // Db in F major
		{71, -1, 6, 0},  // B natural in F major
		{66, 2, 3, 1},   // F# in D major
	}
	for _, c := range cases {
		sp := spell(c.key, c.fifths)
		if sp.letter != c.letter || sp.alter != c.alter {
			t.Errorf("spell(%d,%d) = letter %d alter %d, want %d/%d",
				c.key, c.fifths, sp.letter, sp.alter, c.letter, c.alter)
		}
	}
}

func TestNoteValues(t *testing.T) {
	cases := []struct {
		dur   int64
		value float64
		dots  bool
	}{
		{960, 1, false},
		{480, 0.5, false},
		{240, 0.25, false},
		{1440, 1, true},
	}
	for _, c := range cases {
		v, dotted := noteValue(c.dur, 960)
		if v != c.value || dotted != c.dots {
			t.Errorf("noteValue(%d) = %v/%v, want %v/%v", c.dur, v, dotted, c.value, c.dots)
		}
	}
}

func TestBuildGrandStaff(t *testing.T) {
	s := &song.Song{
		PPQ:         960,
		DurationSec: 2,
		Tempo:       []song.TempoChange{{Tick: 0, BPM: 120}},
		Sigs:        []song.TimeSigChange{{Tick: 0, Num: 4, Den: 4}},
		Bars:        []float64{0, 2},
		Tracks: []song.Track{{
			Name: "Piano", Group: "piano", Visible: true,
			Notes: []song.Note{
				{Key: 48, Tick: 0, DurTicks: 960, Hand: "L"},
				{Key: 60, Tick: 0, DurTicks: 960, Hand: "R"},
			},
		}},
	}
	score, err := Build(s, Options{Width: 900})
	if err != nil {
		t.Fatal(err)
	}
	if len(score.Measures) == 0 {
		t.Fatal("no measures")
	}
	m := score.Measures[0]
	if len(m.Treble) != 1 || len(m.Bass) != 1 {
		t.Fatalf("staves: treble=%d bass=%d", len(m.Treble), len(m.Bass))
	}
	if m.Treble[0].Notes[0].Key != 60 || m.Bass[0].Notes[0].Key != 48 {
		t.Fatalf("notes placed on wrong staff: %+v %+v", m.Treble[0].Notes, m.Bass[0].Notes)
	}
}
