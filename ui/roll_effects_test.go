package ui

import (
	"testing"

	. "go.hasen.dev/shirei"
)

// TestGlowSizeIsQuantized keeps animated glows from invalidating the shadow
// cache on every frame: the size must land on a small, bounded set of values.
func TestGlowSizeIsQuantized(t *testing.T) {
	seen := map[float32]bool{}
	for i := 0; i <= 100; i++ {
		seen[glowSize(26+70*float32(i)/100)] = true
	}
	if len(seen) > 12 {
		t.Fatalf("glow grew through %d distinct sizes, want at most 12", len(seen))
	}
	for sz := range seen {
		if sz/glowQuantum != float32(int(sz/glowQuantum)) {
			t.Fatalf("size %v is not on the %v-point grid", sz, glowQuantum)
		}
	}
}

// TestGlowBallPaintsHalo guards the split effect: the halo is a shadow-only
// container behind a separately drawn core, so it must still paint.
func TestGlowBallPaintsHalo(t *testing.T) {
	img := RenderToImage(100, 100, func() {
		glowBall(50, 50, 60, Vec4{1, 1, 1, 1}, 14)
	})
	s := int(HeadlessScale)
	// Device pixel at logical (50, 10): outside the 60pt core, inside the blur.
	if a := img.RGBAAt(50*s, 10*s).A; a == 0 {
		t.Fatalf("halo pixel is transparent: %d", a)
	}
	// Logical centre is inside the opaque core.
	if a := img.RGBAAt(50*s, 50*s).A; a != 255 {
		t.Fatalf("core pixel alpha = %d, want 255", a)
	}
}
