package ui

import (
	"image/color"
	"math"
	"testing"
)

func TestShortestRootDelta(t *testing.T) {
	cases := []struct {
		from, to int
		want     float64
	}{
		{0, 0, 0},
		{0, 5, 5},
		{0, 7, -5},
		{11, 0, 1},
		{0, 11, -1},
		{5, 11, 6},
		{11, 5, 6},
	}
	for _, c := range cases {
		if got := shortestRootDelta(c.from, c.to); got != c.want {
			t.Errorf("shortestRootDelta(%d,%d) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestEaseInOutEndpoints(t *testing.T) {
	if got := easeInOut(-1); got != 0 {
		t.Fatalf("easeInOut(-1) = %v", got)
	}
	if got := easeInOut(0); got != 0 {
		t.Fatalf("easeInOut(0) = %v", got)
	}
	if got := easeInOut(1); got != 1 {
		t.Fatalf("easeInOut(1) = %v", got)
	}
	if got := easeInOut(2); got != 1 {
		t.Fatalf("easeInOut(2) = %v", got)
	}
	if got := easeInOut(0.5); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("easeInOut(0.5) = %v", got)
	}
}

func TestBlendColor(t *testing.T) {
	a := color.RGBA{0, 0, 0, 0}
	b := color.RGBA{255, 100, 50, 255}
	if got := blendColor(a, b, 0); got != a {
		t.Fatalf("blend at 0 = %v", got)
	}
	if got := blendColor(a, b, 1); got != b {
		t.Fatalf("blend at 1 = %v", got)
	}
	got := blendColor(a, b, 0.5)
	if got.R != 127 || got.G != 50 || got.B != 25 || got.A != 127 {
		t.Fatalf("blend at 0.5 = %v", got)
	}
}
