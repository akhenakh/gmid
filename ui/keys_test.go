package ui

import (
	"testing"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/song"
)

func fitApp(minKey, maxKey int) *app.App {
	a := &app.App{Song: &song.Song{}}
	a.Prefs.KeyRange = 88
	a.Prefs.FitSongRange = true
	a.SongMinKey, a.SongMaxKey = minKey, maxKey
	return a
}

func TestEffectiveKeyRangeFitsSong(t *testing.T) {
	a := fitApp(47, 84)
	lo, hi := EffectiveKeyRange(a)
	if lo != 45 || hi != 86 {
		t.Fatalf("range = %d..%d, want 45..86", lo, hi)
	}

	a.Transport.Transpose = 12
	lo, hi = EffectiveKeyRange(a)
	if lo != 57 || hi != 98 {
		t.Fatalf("transposed range = %d..%d, want 57..98", lo, hi)
	}

	a.Prefs.FitSongRange = false
	lo, hi = EffectiveKeyRange(a)
	if lo != 21 || hi != 108 {
		t.Fatalf("unfitted range = %d..%d, want 21..108", lo, hi)
	}
}

func TestEffectiveKeyRangeMinimumSpan(t *testing.T) {
	a := fitApp(60, 64)
	lo, hi := EffectiveKeyRange(a)
	if hi-lo < 24 {
		t.Fatalf("narrow song range = %d..%d, want at least two octaves", lo, hi)
	}
	if lo > 60 || hi < 64 {
		t.Fatalf("range %d..%d does not contain the song", lo, hi)
	}
}

func TestEffectiveKeyRangeWithoutSong(t *testing.T) {
	a := &app.App{}
	a.Prefs.KeyRange = 61
	a.Prefs.FitSongRange = true
	lo, hi := EffectiveKeyRange(a)
	if lo != 36 || hi != 96 {
		t.Fatalf("range = %d..%d, want 36..96", lo, hi)
	}
}

func TestFitKeyLabelKeepsTokenOnOneLine(t *testing.T) {
	name, fs := fitKeyLabel(88, 26, true, true) // E6
	if name != "E6" || fs != 9 {
		t.Fatalf("wide key: %q %.2f", name, fs)
	}
	name, fs = fitKeyLabel(88, 14, true, true)
	if name != "E6" || fs < 6 || fs >= 9 {
		t.Fatalf("narrow key: %q %.2f", name, fs)
	}
	name, _ = fitKeyLabel(88, 9, true, true)
	if name != "" {
		t.Fatalf("tiny non-C key should hide, got %q", name)
	}
	// 15.1px is the settled white-key width at 700px window width; C2 used to
	// wrap there at font size 9.
	name, fs = fitKeyLabel(36, 15.1, true, true)
	if name != "C2" || fs >= 9 || fs < 6 {
		t.Fatalf("C2 at 15.1px: %q %.2f", name, fs)
	}
	name, fs = fitKeyLabel(87, 15.1, true, true) // Eb6, three runes
	if name == "" || fs < 6 {
		t.Fatalf("Eb6 at 15.1px: %q %.2f", name, fs)
	}
}

// TestFitKeyLabelNeverOverflows guards the wrapping bug: a returned label must
// fit in the key width at the returned font size, otherwise Shirei soft-wraps
// "E6" into "E" and "6".
func TestFitKeyLabelNeverOverflows(t *testing.T) {
	for width := 8; width <= 30; width++ {
		for _, midi := range []int{84, 87, 88, 90} {
			name, fs := fitKeyLabel(midi, float32(width), true, true)
			if name == "" {
				continue
			}
			estimate := 0.62*fs*float32(len(name)) + 2
			if estimate > float32(width) {
				t.Fatalf("%q at %.1fpx: estimated %.1fpx in %dpx key", name, fs, estimate, width)
			}
		}
	}
}
