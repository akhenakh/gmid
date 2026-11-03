package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"
	"time"

	"github.com/fogleman/gg"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"

	. "go.hasen.dev/shirei"
)

// starmap renders the chord-star view used by the Scale Explorer. Root changes
// rotate the scale ring and crossfade the chord star; scale-type changes
// crossfade the ring. Headless renders snap to the final frame so scene
// screenshots stay deterministic.
func starmap(a *app.App) ImageId {
	const size = 300
	const scale = 2.0
	px := int(size * scale)

	ease, fromRoot, fromType, animating := starAnimation(a)

	key := fmt.Sprintf("star/%d/%s/%d", a.UI.ScaleRoot, a.UI.ScaleType, px)
	if animating {
		key = "star/anim"
	} else if starCacheKey == key && starCacheImg != nil {
		return UseImage(key, starCacheImg)
	}
	dc := gg.NewContext(px, px)
	if face, err := gg.LoadFontFace(filepath.Join(a.Assets, "fonts", "Inter-Bold.ttf"), 15*scale); err == nil {
		dc.SetFontFace(face)
	}
	dc.SetColor(color.RGBA{0x0f, 0x0f, 0x12, 0xff})
	dc.Clear()
	cx, cy := float64(px)/2, float64(px)/2
	r := float64(px) * 0.38

	fromScale := theory.ScaleByID(fromType)
	toScale := theory.ScaleByID(a.UI.ScaleType)
	rootF := float64(fromRoot) + shortestRootDelta(fromRoot, a.UI.ScaleRoot)*ease

	// Scale rings: rotate to the new root while crossfading between types.
	if fromType == a.UI.ScaleType || ease >= 1 {
		drawScaleRing(dc, toScale, rootF, cx, cy, r, 1)
	} else {
		drawScaleRing(dc, fromScale, rootF, cx, cy, r, 1-ease)
		drawScaleRing(dc, toScale, rootF, cx, cy, r, ease)
	}

	// Chord stars: crossfade the old and new root chords.
	fromChord := firstChord(fromRoot, fromType)
	toChord := firstChord(a.UI.ScaleRoot, a.UI.ScaleType)
	switch {
	case fromChord == nil && toChord != nil:
		drawChordStar(dc, toChord, cx, cy, r, ease)
	case fromChord != nil && toChord == nil:
		drawChordStar(dc, fromChord, cx, cy, r, 1-ease)
	case fromChord != nil && toChord != nil:
		if sameKeys(fromChord.Keys, toChord.Keys) || ease >= 1 {
			drawChordStar(dc, toChord, cx, cy, r, 1)
		} else {
			drawChordStar(dc, fromChord, cx, cy, r, 1-ease)
			drawChordStar(dc, toChord, cx, cy, r, ease)
		}
	}

	// Nodes and labels. Labels stay fixed; the root ring rotates underneath.
	inactive := color.RGBA{0x4a, 0x4a, 0x62, 0xff}
	active := color.RGBA{0x7a, 0xe2, 0xc0, 0xff}
	for pc := 0; pc < 12; pc++ {
		wFrom := scaleWeight(pc, fromRoot, fromScale)
		wTo := scaleWeight(pc, a.UI.ScaleRoot, toScale)
		w := wFrom + (wTo-wFrom)*ease
		col := blendColor(inactive, active, w)
		ang := -math.Pi/2 + float64(pc)*math.Pi/6
		x := cx + r*math.Cos(ang)
		y := cy + r*math.Sin(ang)
		dc.SetColor(col)
		dc.DrawCircle(x, y, (5.5+1.8*w)*scale)
		dc.Fill()
		dc.SetColor(color.RGBA{0xd0, 0xd0, 0xdc, 0xff})
		dc.DrawStringAnchored(noteNameShort(pc), cx+(r+20*scale)*math.Cos(ang), cy+(r+20*scale)*math.Sin(ang), 0.5, 0.5)
	}
	img := dc.Image().(*image.RGBA)
	if !animating {
		starCacheKey, starCacheImg = key, img
	}
	return UseImage(key, img)
}

// starCache keeps the settled starmap so it is rasterized once, not every
// frame while the Scale Explorer stays open.
var (
	starCacheKey string
	starCacheImg *image.RGBA
)

// starAnimation advances the starmap transition and reports its eased progress
// plus the previous root and scale type. The first call initializes the state
// at the current selection; headless frames always return the settled state.
func starAnimation(a *app.App) (ease float64, fromRoot int, fromType string, animating bool) {
	u := &a.UI
	if u.StarToType == "" {
		u.StarFromRoot, u.StarFromType = u.ScaleRoot, u.ScaleType
		u.StarToRoot, u.StarToType = u.ScaleRoot, u.ScaleType
		u.StarAnimStart = time.Time{}
	}
	if u.StarToRoot != u.ScaleRoot || u.StarToType != u.ScaleType {
		u.StarFromRoot, u.StarFromType = u.StarToRoot, u.StarToType
		u.StarToRoot, u.StarToType = u.ScaleRoot, u.ScaleType
		u.StarAnimStart = time.Now()
	}
	ease = 1
	if !GetHost().HeadlessRender && !u.StarAnimStart.IsZero() {
		t := time.Since(u.StarAnimStart).Seconds() / 0.28
		if t < 1 {
			ease = easeInOut(t)
			animating = true
			RequestNextFrame()
		}
	}
	return ease, u.StarFromRoot, u.StarFromType, animating
}

func drawScaleRing(dc *gg.Context, sc theory.ScaleDef, rootF, cx, cy, r, alpha float64) {
	if sc.ID == "" || alpha <= 0 {
		return
	}
	dc.SetColor(color.RGBA{0x3b, 0x3b, 0x58, byte(0xff * alpha)})
	dc.SetLineWidth(2)
	for i, iv := range sc.Intervals {
		ang := -math.Pi/2 + (rootF+float64(iv))*math.Pi/6
		x := cx + r*math.Cos(ang)
		y := cy + r*math.Sin(ang)
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.ClosePath()
	dc.Stroke()
}

func drawChordStar(dc *gg.Context, chord *theory.DiatonicChord, cx, cy, r, alpha float64) {
	if len(chord.Keys) < 2 || alpha <= 0 {
		return
	}
	dc.SetColor(color.RGBA{0xe8, 0xa0, 0x27, byte(0xff * alpha)})
	dc.SetLineWidth(3)
	for i, k := range chord.Keys {
		ang := -math.Pi/2 + float64(((k%12)+12)%12)*math.Pi/6
		x := cx + r*math.Cos(ang)
		y := cy + r*math.Sin(ang)
		if i == 0 {
			dc.MoveTo(x, y)
		} else {
			dc.LineTo(x, y)
		}
	}
	dc.ClosePath()
	dc.Stroke()
}

func firstChord(root int, typeID string) *theory.DiatonicChord {
	sc := theory.ScaleByID(typeID)
	if sc.ID == "" {
		return nil
	}
	chords := theory.DiatonicChords(root, 4, sc)
	if len(chords) == 0 {
		return nil
	}
	return &chords[0]
}

func scaleWeight(pc, root int, sc theory.ScaleDef) float64 {
	for _, iv := range sc.Intervals {
		if ((root+iv)%12+12)%12 == pc {
			return 1
		}
	}
	return 0
}

func shortestRootDelta(from, to int) float64 {
	d := ((to-from)%12 + 12) % 12
	if d > 6 {
		d -= 12
	}
	return float64(d)
}

func easeInOut(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

func blendColor(a, b color.RGBA, t float64) color.RGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	lerp := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*t)
	}
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), lerp(a.A, b.A)}
}

func sameKeys(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
