package notation

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strconv"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"

	"github.com/akhenakh/gmid/internal/theory"
)

const (
	glyphGClef    = '\uE050'
	glyphFClef    = '\uE062'
	glyphWhole    = '\uE0A2'
	glyphHalf     = '\uE0A3'
	glyphBlack    = '\uE0A4'
	glyphRestAll  = '\uE4E3'
	glyphRestHalf = '\uE4E4'
	glyphRestQtr  = '\uE4E5'
	glyphSharp    = '\uE262'
	glyphNatural  = '\uE261'
	glyphFlat     = '\uE260'
	glyphDot      = '\uE1E7'
	glyphFlag8Up  = '\uE240'
	glyphFlag8Dn  = '\uE241'
	glyphFlag16Up = '\uE242'
	glyphFlag16Dn = '\uE243'
	glyphFlag32Up = '\uE244'
	glyphFlag32Dn = '\uE245'
	glyphTime0    = '\uE080'
)

var (
	paper = color.RGBA{0xf7, 0xf3, 0xe8, 0xff}
	ink   = color.RGBA{0x1a, 0x17, 0x12, 0xff}
	staff = color.RGBA{0x4a, 0x43, 0x38, 0xff}
)

type glyphSet struct {
	font      *sfnt.Font
	faces     map[float64]font.Face
	textPath  string
	textFaces map[float64]font.Face
}

func loadGlyphs(path, textPath string) (*glyphSet, error) {
	if path == "" {
		return nil, fmt.Errorf("sheet font path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	f, err := opentype.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse sheet font: %w", err)
	}
	return &glyphSet{font: f, faces: map[float64]font.Face{}, textPath: textPath, textFaces: map[float64]font.Face{}}, nil
}

func (g *glyphSet) textFace(size float64) font.Face {
	if f := g.textFaces[size]; f != nil {
		return f
	}
	if g.textPath == "" {
		return nil
	}
	f, err := gg.LoadFontFace(g.textPath, size)
	if err != nil {
		return nil
	}
	g.textFaces[size] = f
	return f
}

func (g *glyphSet) drawText(dc *gg.Context, s string, size, cx, cy float64) {
	f := g.textFace(size)
	if f == nil {
		return
	}
	dc.SetFontFace(f)
	bounds, _ := font.BoundString(f, s)
	minX, maxX := float64(bounds.Min.X)/64, float64(bounds.Max.X)/64
	minY, maxY := float64(bounds.Min.Y)/64, float64(bounds.Max.Y)/64
	x := cx - (minX+maxX)/2
	baseline := cy - (minY+maxY)/2
	dc.DrawString(s, x, baseline)
}

func (g *glyphSet) face(size float64) (font.Face, error) {
	if f := g.faces[size]; f != nil {
		return f, nil
	}
	f, err := opentype.NewFace(g.font, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	g.faces[size] = f
	return f, nil
}

func (g *glyphSet) draw(dc *gg.Context, r rune, size, cx, cy float64) {
	f, err := g.face(size)
	if err != nil {
		return
	}
	dc.SetFontFace(f)
	s := string(r)
	bounds, _ := font.BoundString(f, s)
	minX, maxX := float64(bounds.Min.X)/64, float64(bounds.Max.X)/64
	minY, maxY := float64(bounds.Min.Y)/64, float64(bounds.Max.Y)/64
	x := cx - (minX+maxX)/2
	baseline := cy - (minY+maxY)/2
	dc.DrawString(s, x, baseline)
}

// RenderSystem draws one system tile at the requested scale, keeping 2x as the
// default. Coordinates are translated so the tile starts at its system top.
func RenderSystem(s *Score, index int, opt Options) (*image.RGBA, error) {
	if index < 0 || index >= len(s.Systems) {
		return nil, fmt.Errorf("system %d out of range", index)
	}
	space := s.Space
	if space <= 0 {
		space = 9
	}
	g, err := loadGlyphs(opt.FontPath, opt.TextFontPath)
	if err != nil {
		return nil, err
	}
	scale := opt.Scale
	if scale <= 0 {
		scale = 2
	}
	leftColor, rightColor := ink, ink
	if opt.ShowHands {
		leftColor = parseHex(opt.HandLH, theory.DefaultHandLH)
		rightColor = parseHex(opt.HandRH, theory.DefaultHandRH)
	}
	dc := gg.NewContext(int(s.Width*scale), int(SystemHeight*scale))
	dc.Scale(scale, scale)
	dc.SetColor(paper)
	dc.Clear()
	dc.SetLineCap(gg.LineCapRound)
	dc.Translate(0, -s.Systems[index].Y)
	drawSystem(dc, g, s, s.Systems[index], space, leftColor, rightColor)
	return dc.Image().(*image.RGBA), nil
}

// Render draws the complete score into one tall 2x image.
func Render(s *Score, opt Options) (*image.RGBA, error) {
	space := s.Space
	if space <= 0 {
		space = 9
	}
	if s.Width < 200 {
		s.Width = 1000
	}
	g, err := loadGlyphs(opt.FontPath, opt.TextFontPath)
	if err != nil {
		return nil, err
	}
	const defaultScale = 2.0
	scale := opt.Scale
	if scale <= 0 {
		scale = defaultScale
	}
	w, h := s.Width, s.Height
	dc := gg.NewContext(int(w*scale), int(h*scale))
	dc.Scale(scale, scale)
	dc.SetColor(paper)
	dc.Clear()
	dc.SetLineCap(gg.LineCapRound)
	dc.SetColor(staff)
	dc.SetLineWidth(0.7)

	leftColor, rightColor := ink, ink
	if opt.ShowHands {
		leftColor = parseHex(opt.HandLH, theory.DefaultHandLH)
		rightColor = parseHex(opt.HandRH, theory.DefaultHandRH)
	}

	for _, sys := range s.Systems {
		drawSystem(dc, g, s, sys, space, leftColor, rightColor)
	}
	return dc.Image().(*image.RGBA), nil
}

func drawSystem(dc *gg.Context, g *glyphSet, s *Score, sys System, space float64, left, right color.RGBA) {
	top := sys.Y + trebleOffset
	bassTop := sys.Y + bassOffset
	bottom := bassTop + 4*space
	x0, x1 := sys.X, sys.X+sys.W
	dc.SetColor(staff)
	dc.SetLineWidth(0.7)
	for i := 0; i < 5; i++ {
		y := top + float64(i)*space
		dc.MoveTo(x0, y)
		dc.LineTo(x1, y)
		dc.Stroke()
		y = bassTop + float64(i)*space
		dc.MoveTo(x0, y)
		dc.LineTo(x1, y)
		dc.Stroke()
	}
	// Simple brace/bracket at the system start.
	dc.SetLineWidth(0.9)
	dc.MoveTo(x0-8, top)
	dc.LineTo(x0-8, bottom)
	dc.Stroke()
	dc.MoveTo(x0-8, top)
	dc.LineTo(x0-4, top)
	dc.MoveTo(x0-8, bottom)
	dc.LineTo(x0-4, bottom)
	dc.Stroke()

	first := sys.Measure[0]
	m0 := s.Measures[first]
	dc.SetColor(ink)
	g.draw(dc, glyphGClef, 7*space, x0+14, top+1.0*space)
	g.draw(dc, glyphFClef, 7*space, x0+14, bassTop+2*space)

	x := x0 + 31
	fifths := m0.Fifths
	x = drawKeySignature(dc, g, fifths, top, bassTop, x, space, ink)
	drawTimeSignature(dc, g, m0.Num, m0.Den, top, bassTop, x+14, space, ink)

	for _, mi := range sys.Measure {
		m := &s.Measures[mi]
		barEnd := m.X + m.W
		dc.SetColor(staff)
		dc.SetLineWidth(0.8)
		dc.MoveTo(barEnd, top)
		dc.LineTo(barEnd, bottom)
		dc.Stroke()
		// System-start signature changes are rare; draw them at the measure.
		if mi != first {
			if m.Fifths != fifths {
				dc.SetColor(ink)
				drawKeySignature(dc, g, m.Fifths, top, bassTop, m.X+4, space, ink)
				fifths = m.Fifths
			}
		}
		drawMeasure(dc, g, s, m, space, left, right)
		dc.SetColor(ink)
		g.drawText(dc, strconv.Itoa(mi+1), 9, m.X+8, sys.Y+11)
	}
}

func drawMeasure(dc *gg.Context, g *glyphSet, s *Score, m *Measure, space float64, left, right color.RGBA) {
	drawStaff(dc, g, s, m, 0, m.Treble, space, left, right)
	drawStaff(dc, g, s, m, 1, m.Bass, space, left, right)
}

func drawStaff(dc *gg.Context, g *glyphSet, s *Score, m *Measure, staffNo int, events []*Event, space float64, left, right color.RGBA) {
	top := m.Y + trebleOffset
	base := 30
	if staffNo == 1 {
		top = m.Y + bassOffset
		base = 18
	}
	if len(events) == 0 {
		dc.SetColor(ink)
		g.draw(dc, glyphRestAll, 4*space, m.X+m.W/2, top+1.0*space)
		return
	}
	for _, ev := range events {
		x := ev.X
		if x < m.X+6 {
			x = m.X + 6
		}
		// Place the chord notes and work out the stem extent.
		minY, maxY := math.MaxFloat64, -math.MaxFloat64
		offsets := make([]float64, 0, len(ev.Notes))
		for i := range ev.Notes {
			n := &ev.Notes[i]
			y := noteStepY(n.Letter, n.Octave, staffNo, m, space)
			n.staffY = y
			offsets = append(offsets, float64(n.Octave*7+n.Letter-base))
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
		up := average(offsets) < 4
		noteW := 1.25 * space
		stemX := x - noteW*0.48
		if up {
			stemX = x + noteW*0.48
		}
		stemTop, stemBottom := minY, maxY
		if up {
			stemTop = minY - 3.4*space
		} else {
			stemBottom = maxY + 3.4*space
		}
		dc.SetColor(staff)
		dc.SetLineWidth(0.8)
		if !isWhole(ev.Dur, s.PPQ) {
			dc.MoveTo(stemX, stemTop)
			dc.LineTo(stemX, stemBottom)
			dc.Stroke()
		}
		accIdx := 0
		for i := range ev.Notes {
			n := &ev.Notes[i]
			if n.Alter != signatureAlter(n.Letter, m.Fifths) {
				drawAccidental(dc, g, n.Alter, x-(1.9+float64(accIdx)*0.8)*space, n.staffY, space)
				accIdx++
			}
			drawLedger(dc, g, x, n.staffY, top, space)
			col := ink
			switch n.Hand {
			case "L":
				col = left
			case "R":
				col = right
			}
			dc.SetColor(col)
			drawNotehead(dc, g, ev.Dur, s.PPQ, x, n.staffY, space)
		}
		if flags := flagCount(ev.Dur, s.PPQ); flags > 0 {
			glyph := flagGlyph(flags, up)
			if glyph != 0 {
				dc.SetColor(ink)
				y := stemTop
				if !up {
					y = stemBottom
				}
				g.draw(dc, glyph, 3.4*space, stemX, y)
			}
		}
	}
}

func drawNotehead(dc *gg.Context, g *glyphSet, dur, ppq int64, x, y, space float64) {
	value, dotted := noteValue(dur, ppq)
	glyph := glyphBlack
	if value >= 4 {
		glyph = glyphWhole
	} else if value >= 2 {
		glyph = glyphHalf
	}
	g.draw(dc, glyph, 3.6*space, x, y)
	if dotted {
		g.draw(dc, glyphDot, 2.2*space, x+1.1*space, y-space*0.75)
	}
}

func drawAccidental(dc *gg.Context, g *glyphSet, alter int, x, y, space float64) {
	glyph := glyphNatural
	switch alter {
	case 1, 2:
		glyph = glyphSharp
	case -1, -2:
		glyph = glyphFlat
	}
	dc.SetColor(ink)
	g.draw(dc, glyph, 3.2*space, x, y)
	if alter == 2 {
		g.draw(dc, glyphSharp, 3.2*space, x+0.9*space, y)
	}
	if alter == -2 {
		g.draw(dc, glyphFlat, 3.2*space, x+0.8*space, y)
	}
}

func drawLedger(dc *gg.Context, g *glyphSet, x, y, top, space float64) {
	bottom := top + 4*space
	offset := (bottom - y) / (space / 2)
	if offset < 0 {
		for o := -2; o >= int(offset)-1; o -= 2 {
			yy := bottom - float64(o)*space/2
			dc.SetColor(staff)
			dc.SetLineWidth(0.7)
			dc.MoveTo(x-1.55*space, yy)
			dc.LineTo(x+1.55*space, yy)
			dc.Stroke()
		}
	} else if offset > 8 {
		for o := 10; o <= int(offset)+1; o += 2 {
			yy := bottom - float64(o)*space/2
			dc.SetColor(staff)
			dc.SetLineWidth(0.7)
			dc.MoveTo(x-1.55*space, yy)
			dc.LineTo(x+1.55*space, yy)
			dc.Stroke()
		}
	}
}

func drawKeySignature(dc *gg.Context, g *glyphSet, fifths int, trebleTop, bassTop, x, space float64, col color.RGBA) float64 {
	if fifths == 0 {
		return x
	}
	sharp := fifths > 0
	order := []int{3, 0, 4, 1, 5, 2, 6}
	stepsT := []int{8, 5, 9, 6, 3, 7, 4}
	stepsB := []int{6, 3, 7, 4, 1, 5, 2}
	if !sharp {
		order = []int{6, 2, 5, 1, 4, 0, 3}
		stepsT = []int{4, 7, 3, 6, 2, 5, 1}
		stepsB = []int{2, 5, 1, 4, 0, 3, -1}
	}
	n := fifths
	if n < 0 {
		n = -n
	}
	if n > len(order) {
		n = len(order)
	}
	dc.SetColor(col)
	for i := 0; i < n; i++ {
		glyph := glyphSharp
		if !sharp {
			glyph = glyphFlat
		}
		g.draw(dc, glyph, 3.1*space, x+float64(i)*0.9*space, trebleTop+4*space-float64(stepsT[i])*space/2)
		g.draw(dc, glyph, 3.1*space, x+float64(i)*0.9*space, bassTop+4*space-float64(stepsB[i])*space/2)
	}
	return x + float64(n)*0.9*space + 0.8*space
}

func drawTimeSignature(dc *gg.Context, g *glyphSet, num, den int, trebleTop, bassTop, x, space float64, col color.RGBA) float64 {
	if num <= 0 || den <= 0 {
		return x
	}
	dc.SetColor(col)
	drawTimeDigits(dc, g, num, x+space*0.7, trebleTop+1.6*space, space)
	drawTimeDigits(dc, g, den, x+space*0.7, trebleTop+3.6*space, space)
	drawTimeDigits(dc, g, num, x+space*0.7, bassTop+1.6*space, space)
	drawTimeDigits(dc, g, den, x+space*0.7, bassTop+3.6*space, space)
	return x + 2.1*space
}

func drawTimeDigits(dc *gg.Context, g *glyphSet, n int, x, y, space float64) {
	s := strconv.Itoa(n)
	g.drawText(dc, s, 2.4*space, x, y)
}

// noteValue returns the notation value in quarter-note units: whole=4,
// half=2, quarter=1, eighth=0.5, sixteenth=0.25, thirty-second=0.125.
func noteValue(dur, ppq int64) (float64, bool) {
	if ppq <= 0 {
		ppq = 960
	}
	if dur <= 0 {
		return 1, false
	}
	ratio := float64(dur) / float64(ppq)
	base := 4.0
	for _, v := range []float64{4, 2, 1, 0.5, 0.25, 0.125} {
		if ratio >= v*0.9 {
			base = v
			break
		}
	}
	dotted := ratio >= base*1.4 && ratio < base*1.7
	return base, dotted
}

func isWhole(dur, ppq int64) bool {
	v, _ := noteValue(dur, ppq)
	return v >= 4
}

func flagCount(dur, ppq int64) int {
	v, _ := noteValue(dur, ppq)
	switch {
	case v >= 1:
		return 0
	case v >= 0.5:
		return 1
	case v >= 0.25:
		return 2
	default:
		return 3
	}
}

func flagGlyph(n int, up bool) rune {
	upGlyphs := []rune{0, glyphFlag8Up, glyphFlag16Up, glyphFlag32Up}
	dnGlyphs := []rune{0, glyphFlag8Dn, glyphFlag16Dn, glyphFlag32Dn}
	if n < 1 || n > 3 {
		return 0
	}
	if up {
		return upGlyphs[n]
	}
	return dnGlyphs[n]
}

func average(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}

func parseHex(s, fallback string) color.RGBA {
	if len(s) != 7 || s[0] != '#' {
		s = fallback
	}
	if len(s) != 7 || s[0] != '#' {
		return ink
	}
	v, err := strconv.ParseUint(s[1:], 16, 24)
	if err != nil {
		return ink
	}
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}
