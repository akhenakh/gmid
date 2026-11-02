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
