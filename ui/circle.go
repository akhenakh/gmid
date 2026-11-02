package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"

	"github.com/fogleman/gg"

	"github.com/akhenakh/gmid/internal/app"

	. "go.hasen.dev/shirei"
)

var circleOrder = []int{0, 7, 2, 9, 4, 11, 6, 1, 8, 3, 10, 5}

// circleOfFifths renders the interactive circle. The image is cached by key
// until the selected root or theme changes.
func circleOfFifths(a *app.App) ImageId {
	const size = 300
	scale := 2.0
	px := int(size * scale)
	key := fmt.Sprintf("cof/%d/%d", a.UI.ScaleRoot, px)
	dc := gg.NewContext(px, px)
	if face, err := gg.LoadFontFace(filepath.Join(a.Assets, "fonts", "Inter-Bold.ttf"), 17*scale); err == nil {
		dc.SetFontFace(face)
	}
	cx, cy := float64(px)/2, float64(px)/2
	r := float64(px)*0.46 - 4
	dc.SetColor(color.RGBA{0x17, 0x17, 0x20, 0xff})
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
	for i, pc := range circleOrder {
		start := -math.Pi/2 + float64(i)*math.Pi/6
		end := start + math.Pi/6
		col := hslColor(pc, 0.42, 0.38)
		dc.SetColor(col)
		dc.MoveTo(cx, cy)
		dc.LineTo(cx+r*math.Cos(start), cy+r*math.Sin(start))
		dc.DrawArc(cx, cy, r, start, end)
		dc.ClosePath()
		dc.Fill()
		mid := (start + end) / 2
		name := noteNameShort(pc)
		dc.SetColor(color.RGBA{0xf0, 0xf0, 0xf6, 0xff})
		dc.DrawStringAnchored(name, cx+math.Cos(mid)*r*0.80, cy+math.Sin(mid)*r*0.80, 0.5, 0.5)
	}
	// Highlight the selected root.
	for i, pc := range circleOrder {
		if pc != a.UI.ScaleRoot {
			continue
		}
		start := -math.Pi/2 + float64(i)*math.Pi/6
		end := start + math.Pi/6
		dc.SetColor(color.RGBA{0xe8, 0xa0, 0x27, 0xff})
		dc.SetLineWidth(4)
		dc.MoveTo(cx, cy)
		dc.LineTo(cx+r*math.Cos(start), cy+r*math.Sin(start))
		dc.DrawArc(cx, cy, r, start, end)
		dc.ClosePath()
		dc.Stroke()
	}
	// Centre label.
	dc.SetColor(color.RGBA{0x0f, 0x0f, 0x12, 0xff})
	dc.DrawCircle(cx, cy, r*0.30)
	dc.Fill()
	dc.SetColor(color.RGBA{0xe8, 0xa0, 0x27, 0xff})
	dc.DrawStringAnchored(noteNameShort(a.UI.ScaleRoot), cx, cy, 0.5, 0.5)
	img := dc.Image().(*image.RGBA)
	return UseImage(key, img)
}

func noteNameShort(pc int) string {
	names := []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	return names[((pc%12)+12)%12]
}

func hslColor(pc int, s, l float64) color.RGBA {
	h := float64(pc) * 30
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 0xff}
}
