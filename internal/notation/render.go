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
	glyphRest8    = '\uE4E6'
	glyphRest16   = '\uE4E7'
	glyphRest32   = '\uE4E8'
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
	num, den := m0.Num, m0.Den
	x = drawKeySignature(dc, g, fifths, top, bassTop, x, space, ink)
	drawTimeSignature(dc, g, num, den, top, bassTop, x+14, space, ink)

	for _, mi := range sys.Measure {
		m := &s.Measures[mi]
		barEnd := m.X + m.W
		dc.SetColor(staff)
		dc.SetLineWidth(0.8)
		dc.MoveTo(barEnd, top)
		dc.LineTo(barEnd, bottom)
		dc.Stroke()
		if mi != first {
			dc.SetColor(ink)
			sx := m.X + 2
			if m.Fifths != fifths {
				sx = drawKeySignature(dc, g, m.Fifths, top, bassTop, sx, space, ink)
				fifths = m.Fifths
			}
			if m.Num != num || m.Den != den {
				drawTimeSignature(dc, g, m.Num, m.Den, top, bassTop, sx+2, space, ink)
				num, den = m.Num, m.Den
			}
		}
		drawMeasure(dc, g, s, m, space, left, right)
		dc.SetColor(ink)
		g.drawText(dc, strconv.Itoa(mi+1), 9, m.X+8, sys.Y+11)
	}
	drawTies(dc, s, sys, space)
}

func drawTies(dc *gg.Context, s *Score, sys System, space float64) {
	for i, mi := range sys.Measure {
		if i+1 >= len(sys.Measure) {
			continue
		}
		next := &s.Measures[sys.Measure[i+1]]
		m := &s.Measures[mi]
		for staff := 0; staff < 2; staff++ {
			cur := m.Treble
			nxt := next.Treble
			if staff == 1 {
				cur = m.Bass
				nxt = next.Bass
			}
			for _, ev := range cur {
				if !ev.TieEnd {
					continue
				}
				for ni := range ev.Notes {
					n := &ev.Notes[ni]
					targetEv, target := findTiedNote(nxt, n)
					if target == nil {
						continue
					}
					x1 := ev.X + 0.9*space
					x2 := targetEv.X - 0.9*space
					if x2 <= x1 {
						continue
					}
					y1 := noteStepY(n.Letter, n.Octave, staff, m, space)
					y2 := noteStepY(target.Letter, target.Octave, staff, next, space)
					mid := (x1 + x2) / 2
					dc.SetColor(ink)
					dc.SetLineWidth(0.9)
					dc.MoveTo(x1, y1+space*0.35)
					dc.QuadraticTo(mid, math.Max(y1, y2)+2.2*space, x2, y2+space*0.35)
					dc.Stroke()
				}
			}
		}
	}
}

func findTiedNote(events []*Event, from *Note) (*Event, *Note) {
	for _, ev := range events {
		if !ev.TieStart {
			continue
		}
		for i := range ev.Notes {
			if ev.Notes[i].Key == from.Key && ev.Notes[i].Hand == from.Hand {
				return ev, &ev.Notes[i]
			}
		}
	}
	return nil, nil
}

func drawRest(dc *gg.Context, g *glyphSet, m *Measure, staffNo int, ppq int64, start, dur int64, space float64) {
	if dur < 1 {
		return
	}
	left := 10 + m.SigW
	right := 10.0
	if m.W-left-right < 20 {
		left = 10
	}
	frac := 0.0
	if m.End > m.Start {
		frac = float64(start-m.Start) / float64(m.End-m.Start)
	}
	x := m.X + left + frac*(m.W-left-right)
	top := m.Y + trebleOffset
	if staffNo == 1 {
		top = m.Y + bassOffset
	}
	shape := noteShape(dur, ppq)
	var glyph rune
	switch {
	case dur >= (m.End-m.Start)*95/100:
		glyph = glyphRestAll
	case shape.base >= 2:
		glyph = glyphRestHalf
	case shape.base >= 1:
		glyph = glyphRestQtr
	case shape.base >= 0.5:
		glyph = glyphRest8
	case shape.base >= 0.25:
		glyph = glyphRest16
	default:
		glyph = glyphRest32
	}
	dc.SetColor(ink)
	g.draw(dc, glyph, 4*space, x, top+2*space)
}

func drawMeasure(dc *gg.Context, g *glyphSet, s *Score, m *Measure, space float64, left, right color.RGBA) {
	drawStaff(dc, g, s, m, 0, m.Treble, space, left, right)
	drawStaff(dc, g, s, m, 1, m.Bass, space, left, right)
}

// eventGeom is the per-chord geometry needed for stems, beams and tuplets.
type eventGeom struct {
	ev     *Event
	x      float64
	up     bool
	minY   float64
	maxY   float64
	stemX  float64
	beamed bool
	beamY  float64
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
	// Rests fill the leading, internal and trailing gaps of the staff.
	for _, gap := range restGaps(events, m) {
		drawRest(dc, g, m, staffNo, s.PPQ, gap.Start, gap.Dur, space)
	}

	geoms := make([]eventGeom, len(events))
	for i, ev := range events {
		x := ev.X
		if x < m.X+6 {
			x = m.X + 6
		}
		gm := eventGeom{ev: ev, x: x, minY: math.MaxFloat64, maxY: -math.MaxFloat64}
		offsets := make([]float64, 0, len(ev.Notes))
		for ni := range ev.Notes {
			n := &ev.Notes[ni]
			y := noteStepY(n.Letter, n.Octave, staffNo, m, space)
			n.staffY = y
			offsets = append(offsets, float64(n.Octave*7+n.Letter-base))
			gm.minY = math.Min(gm.minY, y)
			gm.maxY = math.Max(gm.maxY, y)
		}
		gm.up = average(offsets) < 4
		gm.stemX = x - 1.25*space*0.48
		if gm.up {
			gm.stemX = x + 1.25*space*0.48
		}
		geoms[i] = gm
	}
	layoutBeams(geoms, m, s.PPQ, space)

	accState := accidentalState{}
	for i := range geoms {
		gm := &geoms[i]
		ev := gm.ev
		stemTop, stemBottom := gm.minY, gm.maxY
		switch {
		case gm.beamed && gm.up:
			stemTop = gm.beamY
		case gm.beamed:
			stemBottom = gm.beamY
		case gm.up:
			stemTop = gm.minY - 3.4*space
		default:
			stemBottom = gm.maxY + 3.4*space
		}
		dc.SetColor(staff)
		dc.SetLineWidth(0.8)
		if !isWhole(ev.Dur, s.PPQ) {
			dc.MoveTo(gm.stemX, stemTop)
			dc.LineTo(gm.stemX, stemBottom)
			dc.Stroke()
		}
		accIdx := 0
		for ni := range ev.Notes {
			n := &ev.Notes[ni]
			if accState.altered(n, m.Fifths) {
				drawAccidental(dc, g, n.Alter, gm.x-(1.9+float64(accIdx)*0.8)*space, n.staffY, space)
				accIdx++
			}
			drawLedger(dc, g, gm.x, n.staffY, top, space)
			col := ink
			switch n.Hand {
			case "L":
				col = left
			case "R":
				col = right
			}
			dc.SetColor(col)
			drawNotehead(dc, g, ev.Dur, s.PPQ, gm.x, n.staffY, space)
		}
		if !gm.beamed {
			if flags := flagCount(ev.Dur, s.PPQ); flags > 0 {
				glyph := flagGlyph(flags, gm.up)
				if glyph != 0 {
					dc.SetColor(ink)
					y := stemTop
					if !gm.up {
						y = stemBottom
					}
					g.draw(dc, glyph, 3.4*space, gm.stemX, y)
				}
			}
		}
	}
	drawBeams(dc, geoms, s.PPQ, space)
	drawTuplets(dc, g, geoms, s.PPQ, space)
}

// layoutBeams groups consecutive eighth-or-shorter chords inside one beat and
// assigns the shared stem/beam height.
func layoutBeams(geoms []eventGeom, m *Measure, ppq int64, space float64) {
	beat := ppq * 4 / int64(m.Den)
	if beat <= 0 {
		beat = ppq
	}
	group := make([]int, 0, 4)
	flush := func() {
		if len(group) < 2 {
			group = group[:0]
			return
		}
		up := geoms[group[0]].up
		minY, maxY := math.MaxFloat64, -math.MaxFloat64
		for _, gi := range group {
			geoms[gi].up = up
			minY = math.Min(minY, geoms[gi].minY)
			maxY = math.Max(maxY, geoms[gi].maxY)
		}
		beamY := maxY + 3.4*space
		if up {
			beamY = minY - 3.4*space
		}
		for _, gi := range group {
			geoms[gi].beamed = true
			geoms[gi].beamY = beamY
		}
		group = group[:0]
	}
	curBeat := int64(-1)
	var prevEnd int64
	for i := range geoms {
		ev := geoms[i].ev
		if isWhole(ev.Dur, ppq) || flagCount(ev.Dur, ppq) == 0 {
			flush()
			curBeat = -1
			continue
		}
		b := (ev.Start - m.Start) / beat
		if len(group) > 0 && (b != curBeat || ev.Start != prevEnd) {
			flush()
		}
		group = append(group, i)
		curBeat = b
		prevEnd = ev.Start + ev.Dur
	}
	flush()
}

func drawBeams(dc *gg.Context, geoms []eventGeom, ppq int64, space float64) {
	for i := 0; i < len(geoms); {
		if !geoms[i].beamed {
			i++
			continue
		}
		j := i
		for j+1 < len(geoms) && geoms[j+1].beamed && geoms[j+1].beamY == geoms[i].beamY {
			j++
		}
		drawBeamGroup(dc, geoms[i:j+1], ppq, space)
		i = j + 1
	}
}

func drawBeamGroup(dc *gg.Context, group []eventGeom, ppq int64, space float64) {
	if len(group) < 2 {
		return
	}
	up := group[0].up
	beamY := group[0].beamY
	dir := 1.0
	if !up {
		dir = -1.0
	}
	maxFlags := 1
	for _, gm := range group {
		if f := flagCount(gm.ev.Dur, ppq); f > maxFlags {
			maxFlags = f
		}
	}
	dc.SetColor(ink)
	dc.SetLineWidth(0.55 * space)
	beam := func(a, b, y float64) {
		dc.MoveTo(a, y)
		dc.LineTo(b, y)
		dc.Stroke()
	}
	x1, x2 := group[0].stemX, group[len(group)-1].stemX
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	beam(x1, x2, beamY)
	for k := 1; k < maxFlags; k++ {
		y := beamY + float64(k)*0.62*space*dir
		for i := 0; i < len(group); {
			if flagCount(group[i].ev.Dur, ppq) < k+1 {
				i++
				continue
			}
			j := i
			for j+1 < len(group) && flagCount(group[j+1].ev.Dur, ppq) >= k+1 {
				j++
			}
			a, b := group[i].stemX, group[j].stemX
			if a > b {
				a, b = b, a
			}
			if j == i {
				if i > 0 {
					a = b - 0.85*space
				} else {
					b = a + 0.85*space
				}
			}
			beam(a, b, y)
			i = j + 1
		}
	}
}

// restGap is a silent span that needs a rest symbol.
type restGap struct{ Start, Dur int64 }

// restGaps returns the leading, internal and trailing silent spans of a staff.
func restGaps(events []*Event, m *Measure) []restGap {
	var gaps []restGap
	cursor := m.Start
	for _, ev := range events {
		if ev.Start > cursor {
			gaps = append(gaps, restGap{Start: cursor, Dur: ev.Start - cursor})
		}
		if ev.Start+ev.Dur > cursor {
			cursor = ev.Start + ev.Dur
		}
	}
	if cursor < m.End {
		gaps = append(gaps, restGap{Start: cursor, Dur: m.End - cursor})
	}
	return gaps
}

// accidentalState tracks the accidentals already applied in one bar so repeated
// notes stay unmarked and naturals cancel earlier sharps/flats.
type accidentalState map[[2]int]int

// altered reports whether the note needs an accidental and records its new
// state for the rest of the bar.
func (a accidentalState) altered(n *Note, fifths int) bool {
	pos := [2]int{n.Letter, n.Octave}
	current, seen := a[pos]
	if !seen {
		current = signatureAlter(n.Letter, fifths)
	}
	if n.Alter == current {
		return false
	}
	a[pos] = n.Alter
	return true
}

// drawTuplets labels complete groups of three triplet chords with a single "3".
func drawTuplets(dc *gg.Context, g *glyphSet, geoms []eventGeom, ppq int64, space float64) {
	for i := 0; i < len(geoms); {
		if noteShape(geoms[i].ev.Dur, ppq).tuplet != 3 {
			i++
			continue
		}
		j := i
		for j+1 < len(geoms) &&
			noteShape(geoms[j+1].ev.Dur, ppq).tuplet == 3 &&
			geoms[j+1].ev.Start == geoms[j].ev.Start+geoms[j].ev.Dur {
			j++
		}
		if j >= i+2 {
			cx := (geoms[i].x + geoms[j].x) / 2
			y := math.MaxFloat64
			for k := i; k <= j; k++ {
				y = math.Min(y, geoms[k].minY)
			}
			dc.SetColor(ink)
			g.drawText(dc, "3", 2.2*space, cx, y-2.1*space)
		}
		i = j + 1
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

// noteValue returns the notated base value in quarter-note units and whether it
// is dotted. Triplet detection lives in noteShape.
func noteValue(dur, ppq int64) (float64, bool) {
	sh := noteShape(dur, ppq)
	return sh.base, sh.dotted
}

type noteShapeInfo struct {
	base   float64
	dotted bool
	tuplet int
}

// noteShape fits a tick duration to a standard note value, a dotted value or a
// 3:2 triplet. The closest candidate wins, with a mild tolerance for humanized
// MIDI timing.
func noteShape(dur, ppq int64) noteShapeInfo {
	if ppq <= 0 {
		ppq = 960
	}
	if dur <= 0 {
		return noteShapeInfo{base: 1}
	}
	ratio := float64(dur) / float64(ppq)
	best := noteShapeInfo{base: 1}
	bestDiff := math.MaxFloat64
	for _, base := range []float64{4, 2, 1, 0.5, 0.25, 0.125} {
		candidates := []noteShapeInfo{
			{base: base},
			{base: base, dotted: true},
			{base: base, tuplet: 3},
		}
		for _, c := range candidates {
			val := c.base
			if c.dotted {
				val *= 1.5
			}
			if c.tuplet == 3 {
				val *= 2.0 / 3.0
			}
			diff := math.Abs(ratio - val)
			if diff < bestDiff {
				bestDiff = diff
				best = c
			}
		}
	}
	return best
}

func isWhole(dur, ppq int64) bool {
	sh := noteShape(dur, ppq)
	return sh.base >= 4
}

func flagCount(dur, ppq int64) int {
	sh := noteShape(dur, ppq)
	switch {
	case sh.base >= 1:
		return 0
	case sh.base >= 0.5:
		return 1
	case sh.base >= 0.25:
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
