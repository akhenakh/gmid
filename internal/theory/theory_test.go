package theory

import "testing"

func TestDetectChord(t *testing.T) {
	cases := []struct {
		notes []int
		want  string
	}{
		{[]int{60, 64, 67}, "C"},
		{[]int{60, 63, 67}, "Cm"},
		{[]int{60, 64, 67, 71}, "Cmaj7"},
		{[]int{60, 64, 67, 70}, "C7"},
		{[]int{64, 67, 72}, "C/E"},
		{[]int{60, 62}, "—"},
	}
	for _, c := range cases {
		if got := DetectChord(c.notes); got != c.want {
			t.Errorf("%v: got %q want %q", c.notes, got, c.want)
		}
	}
}

func TestDiatonicChords(t *testing.T) {
	chords := DiatonicChords(0, 4, ScaleByID("major"))
	if len(chords) != 7 {
		t.Fatalf("chords %d", len(chords))
	}
	if chords[0].Name != "C" || chords[1].Name != "Dm" || chords[6].Name != "Bdim" {
		t.Fatalf("unexpected names: %+v", chords)
	}
	if chords[0].Roman != "I" || chords[1].Roman != "ii" {
		t.Fatalf("unexpected romans: %+v", chords)
	}
}

func TestProgression(t *testing.T) {
	p := Progressions[0]
	name, keys := ProgressionChord(0, 4, ScaleByID("major"), p, 0)
	if name != "C" || len(keys) != 3 {
		t.Fatalf("%s %v", name, keys)
	}
	name, keys = ProgressionChord(0, 4, ScaleByID("major"), p, 1)
	if name != "G" || len(keys) != 3 {
		t.Fatalf("%s %v", name, keys)
	}
}
