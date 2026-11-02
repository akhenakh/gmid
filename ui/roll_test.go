package ui

import "testing"

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
