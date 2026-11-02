package ui

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"path/filepath"

	"github.com/fogleman/gg"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/theory"

	. "go.hasen.dev/shirei"
)

// starmap renders a small chord-star view used by the Scale Explorer.
func starmap(a *app.App) ImageId {
	const size = 300
	scale := 2.0
	px := int(size * scale)
	key := fmt.Sprintf("star/%d/%s/%d", a.UI.ScaleRoot, a.UI.ScaleType, px)
	dc := gg.NewContext(px, px)
	if face, err := gg.LoadFontFace(filepath.Join(a.Assets, "fonts", "Inter-Bold.ttf"), 15*scale); err == nil {
		dc.SetFontFace(face)
	}
	dc.SetColor(color.RGBA{0x0f, 0x0f, 0x12, 0xff})
	dc.Clear()
	cx, cy := float64(px)/2, float64(px)/2
	r := float64(px) * 0.38
	// Scale ring.
	sc := theory.ScaleByID(a.UI.ScaleType)
	dc.SetColor(color.RGBA{0x3b, 0x3b, 0x58, 0xff})
	dc.SetLineWidth(2)
	for i, iv := range sc.Intervals {
		ang := -math.Pi/2 + float64(a.UI.ScaleRoot+iv)*math.Pi/6
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
	// Chord star (root triad from the scale).
	chord := theory.DiatonicChords(a.UI.ScaleRoot, 4, sc)
	if len(chord) > 0 {
		dc.SetColor(color.RGBA{0xe8, 0xa0, 0x27, 0xff})
		dc.SetLineWidth(3)
		for i, k := range chord[0].Keys {
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
	// Nodes and labels.
	for pc := 0; pc < 12; pc++ {
		ang := -math.Pi/2 + float64(pc)*math.Pi/6
		x := cx + r*math.Cos(ang)
		y := cy + r*math.Sin(ang)
		inScale := false
		for _, iv := range sc.Intervals {
			if ((a.UI.ScaleRoot+iv)%12+12)%12 == pc {
				inScale = true
				break
			}
		}
		if inScale {
			dc.SetColor(color.RGBA{0x7a, 0xe2, 0xc0, 0xff})
		} else {
			dc.SetColor(color.RGBA{0x4a, 0x4a, 0x62, 0xff})
		}
		dc.DrawCircle(x, y, 6*scale)
		dc.Fill()
		dc.SetColor(color.RGBA{0xd0, 0xd0, 0xdc, 0xff})
		dc.DrawStringAnchored(noteNameShort(pc), cx+(r+20*scale)*math.Cos(ang), cy+(r+20*scale)*math.Sin(ang), 0.5, 0.5)
	}
	img := dc.Image().(*image.RGBA)
	return UseImage(key, img)
}
