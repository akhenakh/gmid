package ui

import (
	"testing"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/song"
)

func TestRollWindowFavoursFuture(t *testing.T) {
	now, visible := 10.0, 6.0
	lo, hi := rollWindow(now, visible)
	if hi-now < visible*0.8 {
		t.Fatalf("future window %.2fs, want at least %.2fs", hi-now, visible*0.8)
	}
	if hi-now <= now-lo {
		t.Fatalf("future %.2fs should exceed past %.2fs", hi-now, now-lo)
	}
}

func TestSheetKeepsRollScale(t *testing.T) {
	a := app.New(config.Defaults(), "", "")
	a.Song = &song.Song{}
	withSheet := rollVisibleSeconds(a, 400)
	a.Prefs.Sheet = false
	withoutSheet := rollVisibleSeconds(a, 400)
	if withSheet >= withoutSheet {
		t.Fatalf("sheet window %.2fs must be shorter than full window %.2fs", withSheet, withoutSheet)
	}
	// At 400 px of roll plus 214 px of sheet, the scale is the same as a
	// 614 px roll showing 6 s.
	want := 6.0 * 400.0 / 614.0
	if withSheet != want {
		t.Fatalf("sheet window %.3fs, want %.3fs", withSheet, want)
	}
}
