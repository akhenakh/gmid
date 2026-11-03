// Package notation turns the tick-based song model into a small, practice-
// oriented grand staff and renders it with the Bravura music font.
package notation

import (
	"sort"

	"github.com/akhenakh/gmid/internal/song"
	"github.com/akhenakh/gmid/internal/theory"
)

// Options controls score construction and drawing.
type Options struct {
	Width        float64
	HandLH       string
	HandRH       string
	ShowHands    bool
	FontPath     string
	TextFontPath string
	Space        float64
	Scale        float64
}

// Staff geometry in logical pixels. The extra top margin leaves room for
// ledger lines and bar numbers above the treble staff.
const (
	firstSystemY = 16.0
	trebleOffset = 42.0
	bassOffset   = 118.0
	// SystemHeight is the logical height of one grand-staff system tile.
	SystemHeight = 190.0
)

// Score is the laid-out notation for one song.
type Score struct {
	PPQ      int64
	Space    float64
	Width    float64
	Height   float64
	Systems  []System
	Measures []Measure
	Bars     []Rect
}

// Rect is a measure rectangle in logical image coordinates.
type Rect struct {
	X, Y, W, H float64
}

type System struct {
	Y       float64
	X       float64
	W       float64
	MusicX  float64
	Measure []int // indices into Score.Measures
}

type Measure struct {
	Index  int
	Start  int64
	End    int64
	Num    int
	Den    int
	Fifths int
	Treble []*Event
	Bass   []*Event
	X      float64
	Y      float64
	W      float64
}

type Event struct {
	Start    int64
	Dur      int64
	Notes    []Note
	TieStart bool
	TieEnd   bool
	X        float64
	Y        float64
	Staff    int
}

type Note struct {
	Key    int
	Letter int // 0=C .. 6=B
	Octave int
	Alter  int // -2..2
	Hand   string
	Staff  int
	KeyID  int

	staffY float64
}

// Build converts the visible keyboard tracks into a grand-staff score.
func Build(s *song.Song, opt Options) (*Score, error) {
	if opt.Space <= 0 {
		opt.Space = 9
	}
	if opt.Width <= 0 {
		opt.Width = 1100
	}
	ppq := int64(s.PPQ)
	if ppq <= 0 {
		ppq = 960
	}
	score := &Score{PPQ: ppq, Space: opt.Space, Width: opt.Width}
	score.Measures = buildMeasures(s, ppq)
	score.Measures = fillMeasures(s, score.Measures, ppq, opt)
	score.layout(opt)
	return score, nil
}

func buildMeasures(s *song.Song, ppq int64) []Measure {
	bars := s.BarStarts()
	if len(bars) == 0 {
		bars = []float64{0}
	}
	starts := make([]int64, 0, len(bars))
	for _, b := range bars {
		t := s.SecToTick(b)
		if len(starts) == 0 || t > starts[len(starts)-1] {
			starts = append(starts, t)
		}
	}
	if len(starts) == 0 {
		starts = []int64{0}
	}
	measures := make([]Measure, len(starts))
	for i, start := range starts {
		var end int64
		if i+1 < len(starts) {
			end = starts[i+1]
		} else {
			end = s.SecToTick(s.DurationSec)
			if end <= start {
				num, den := meterAt(s, start)
				end = start + int64(num)*ppq*4/int64(den)
			}
		}
		if end <= start {
			end = start + ppq*4
		}
		num, den := meterAt(s, start)
		measures[i] = Measure{
			Index:  i,
			Start:  start,
			End:    end,
			Num:    num,
			Den:    den,
			Fifths: fifthsAt(s, start),
		}
	}
	return measures
}

func fillMeasures(s *song.Song, measures []Measure, ppq int64, opt Options) []Measure {
	if len(measures) == 0 {
		return measures
	}
	grid := ppq / 4 // sixteenth note
	if grid < 1 {
		grid = 1
	}
	keyboard := false
	for i := range s.Tracks {
		if !s.Tracks[i].IsDrum && theory.IsKeyboardGroup(s.Tracks[i].Group) {
			keyboard = true
			break
		}
	}
	for ti := range s.Tracks {
		t := &s.Tracks[ti]
		if t.IsDrum || t.Excluded || !s.IsShown(t) {
			continue
		}
		if keyboard && !theory.IsKeyboardGroup(t.Group) {
			continue
		}
		for ni := range t.Notes {
			n := &t.Notes[ni]
			if n.DurTicks <= 0 || n.Key < 0 || n.Key > 127 {
				continue
			}
			start := quantize(n.Tick, grid)
			dur := quantize(n.DurTicks, grid)
			if dur < grid {
				dur = grid
			}
			end := start + dur
			segStart := start
			for segStart < end {
				mi := measureIndex(measures, segStart)
				if mi < 0 {
					break
				}
				m := &measures[mi]
				segEnd := end
				if segEnd > m.End {
					segEnd = m.End
				}
				if segEnd <= segStart {
					break
				}
				hand := n.Hand
				if hand != "L" && hand != "R" {
					if n.Key < 60 {
						hand = "L"
					} else {
						hand = "R"
					}
				}
				staff := 0
				if hand == "L" {
					staff = 1
				}
				sp := spell(n.Key, m.Fifths)
				note := Note{
					Key:    n.Key,
					Letter: sp.letter,
					Octave: sp.octave,
					Alter:  sp.alter,
					Hand:   hand,
					Staff:  staff,
					KeyID:  int(n.ID),
				}
				evs := &m.Treble
				if staff == 1 {
					evs = &m.Bass
				}
				ev := findEvent(*evs, segStart)
				if ev == nil {
					ev = &Event{
						Start:    segStart,
						Dur:      segEnd - segStart,
						TieStart: segStart > start,
						TieEnd:   segEnd < end,
						Staff:    staff,
					}
					*evs = append(*evs, ev)
				} else {
					if segEnd-segStart > ev.Dur {
						ev.Dur = segEnd - segStart
					}
					if segStart > start {
						ev.TieStart = true
					}
					if segEnd < end {
						ev.TieEnd = true
					}
				}
				dupe := false
				for _, ex := range ev.Notes {
					if ex.Key == n.Key {
						dupe = true
						break
					}
				}
				if !dupe {
					ev.Notes = append(ev.Notes, note)
				}
				segStart = segEnd
			}
		}
	}
	for i := range measures {
		sort.SliceStable(measures[i].Treble, func(a, b int) bool {
			return measures[i].Treble[a].Start < measures[i].Treble[b].Start
		})
		sort.SliceStable(measures[i].Bass, func(a, b int) bool {
			return measures[i].Bass[a].Start < measures[i].Bass[b].Start
		})
		for _, ev := range measures[i].Treble {
			sort.SliceStable(ev.Notes, func(a, b int) bool { return ev.Notes[a].Key < ev.Notes[b].Key })
		}
		for _, ev := range measures[i].Bass {
			sort.SliceStable(ev.Notes, func(a, b int) bool { return ev.Notes[a].Key < ev.Notes[b].Key })
		}
	}
	return measures
}

func findEvent(events []*Event, start int64) *Event {
	for _, ev := range events {
		if ev.Start == start {
			return ev
		}
	}
	return nil
}

func measureIndex(measures []Measure, tick int64) int {
	i := sort.Search(len(measures), func(i int) bool { return measures[i].End > tick })
	if i >= len(measures) || tick < measures[i].Start {
		return -1
	}
	return i
}

func quantize(v, grid int64) int64 {
	if v < 0 {
		return 0
	}
	return (v + grid/2) / grid * grid
}

func meterAt(s *song.Song, tick int64) (int, int) {
	num, den := 4, 4
	for _, sg := range s.Sigs {
		if sg.Tick > tick {
			break
		}
		if sg.Num > 0 && sg.Den > 0 {
			num, den = sg.Num, sg.Den
		}
	}
	return num, den
}

func fifthsAt(s *song.Song, tick int64) int {
	fifths := 0
	found := false
	for _, k := range s.Keys {
		if k.Tick > tick {
			break
		}
		fifths = fifthsFor(k.Semitone, k.Minor)
		found = true
	}
	if found {
		return fifths
	}
	if s.Meta.KeySemitone != nil {
		return fifthsFor(*s.Meta.KeySemitone, s.Meta.KeyMinor)
	}
	return 0
}

var majorFifths = [12]int{0, 7, 2, -3, 4, -1, 6, 1, -4, 3, -2, 5}

// minorFifths selects the common spelling for minor keys (Bb minor rather than
// A# minor, C# minor rather than Db minor, and so on).
var minorFifths = [12]int{-3, 4, -1, -6, 1, -4, 3, -2, 5, 0, -5, 2}

func fifthsFor(tonic int, minor bool) int {
	t := ((tonic % 12) + 12) % 12
	if minor {
		return minorFifths[t]
	}
	return majorFifths[t]
}

type pitchSpelling struct {
	letter int
	octave int
	alter  int
}

var naturalPC = [7]int{0, 2, 4, 5, 7, 9, 11}

func spell(key, fifths int) pitchSpelling {
	best := pitchSpelling{letter: 0, octave: key/12 - 1, alter: 0}
	bestScore := 1 << 30
	for oct := key/12 - 1; oct <= key/12+1; oct++ {
		for letter := 0; letter < 7; letter++ {
			natural := 12*(oct+1) + naturalPC[letter]
			alter := key - natural
			if alter < -2 || alter > 2 {
				continue
			}
			score := abs(alter-signatureAlter(letter, fifths))*4 + abs(alter)*2
			if alter == 0 {
				score--
			}
			if fifths >= 0 && alter < 0 {
				score++
			}
			if fifths < 0 && alter > 0 {
				score++
			}
			if score < bestScore {
				bestScore = score
				best = pitchSpelling{letter: letter, octave: oct, alter: alter}
			}
		}
	}
	return best
}

func signatureAlter(letter, fifths int) int {
	if fifths > 0 {
		order := []int{3, 0, 4, 1, 5, 2, 6}
		n := fifths
		if n > len(order) {
			n = len(order)
		}
		for i := 0; i < n; i++ {
			if order[i] == letter {
				return 1
			}
		}
	}
	if fifths < 0 {
		order := []int{6, 2, 5, 1, 4, 0, 3}
		n := -fifths
		if n > len(order) {
			n = len(order)
		}
		for i := 0; i < n; i++ {
			if order[i] == letter {
				return -1
			}
		}
	}
	return 0
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// layout assigns systems and measure rectangles, then positions events.
func (s *Score) layout(opt Options) {
	if len(s.Measures) == 0 {
		return
	}
	space := s.Space
	marginX := 54.0
	content := s.Width - marginX*2
	if content < 240 {
		content = 240
	}
	systems := make([]System, 0, 8)
	for i := 0; i < len(s.Measures); {
		sys := System{X: marginX, W: content}
		for i < len(s.Measures) {
			w := measureWidth(&s.Measures[i], space)
			used := 0.0
			for _, mi := range sys.Measure {
				used += measureWidth(&s.Measures[mi], space)
			}
			if len(sys.Measure) > 0 && used+w > content {
				break
			}
			sys.Measure = append(sys.Measure, i)
			i++
			if len(sys.Measure) >= 8 {
				break
			}
		}
		systems = append(systems, sys)
	}

	for si := range systems {
		systems[si].Y = firstSystemY + float64(si)*SystemHeight
	}
	for si := range systems {
		sys := &systems[si]
		m0 := &s.Measures[sys.Measure[0]]
		sigW := 31 + keySigWidth(m0.Fifths, space) + 14 + 2.1*space
		sys.MusicX = sys.X + sigW
		avail := sys.W - sigW
		weights := make([]float64, len(sys.Measure))
		sum := 0.0
		for i, mi := range sys.Measure {
			weights[i] = measureWidth(&s.Measures[mi], space)
			sum += weights[i]
		}
		scale := 1.0
		if sum > 0 {
			scale = avail / sum
		}
		x := sys.MusicX
		for i, mi := range sys.Measure {
			m := &s.Measures[mi]
			m.X = x
			m.Y = sys.Y
			m.W = weights[i] * scale
			x += m.W
		}
	}
	s.Systems = systems
	s.Height = systems[len(systems)-1].Y + SystemHeight
	s.Bars = make([]Rect, len(s.Measures))
	for i := range s.Measures {
		m := &s.Measures[i]
		s.Bars[i] = Rect{X: m.X, Y: m.Y, W: m.W, H: 150}
		layoutEvents(m.Treble, m, 0, space)
		layoutEvents(m.Bass, m, 1, space)
	}
}

func keySigWidth(fifths int, space float64) float64 {
	if fifths == 0 {
		return 0
	}
	n := fifths
	if n < 0 {
		n = -n
	}
	if n > 7 {
		n = 7
	}
	return float64(n)*0.9*space + 0.8*space
}

func measureWidth(m *Measure, space float64) float64 {
	n := len(m.Treble)
	if len(m.Bass) > n {
		n = len(m.Bass)
	}
	w := 78 + float64(n)*18
	if w < 96 {
		w = 96
	}
	if w > 250 {
		w = 250
	}
	return w
}

func layoutEvents(events []*Event, m *Measure, staff int, space float64) {
	top := m.Y + trebleOffset
	if staff == 1 {
		top = m.Y + bassOffset
	}
	_ = top
	for _, ev := range events {
		if m.End <= m.Start {
			ev.X = m.X + 10
			continue
		}
		frac := float64(ev.Start-m.Start) / float64(m.End-m.Start)
		ev.X = m.X + 10 + frac*(m.W-20)
	}
}

func noteStepY(letter, octave, staff int, m *Measure, space float64) float64 {
	top := m.Y + trebleOffset
	base := 30 // E4 at the bottom line of the treble staff
	if staff == 1 {
		top = m.Y + bassOffset
		base = 18 // G2 at the bottom line of the bass staff
	}
	step := octave*7 + letter
	return top + 4*space - float64(step-base)*space/2
}
