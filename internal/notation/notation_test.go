package notation

import (
	"os"
	"testing"

	"github.com/akhenakh/gmid/internal/song"
)

func TestFifthsFor(t *testing.T) {
	cases := []struct {
		tonic int
		minor bool
		want  int
	}{
		{0, false, 0},  // C major
		{7, false, 1},  // G major
		{5, false, -1}, // F major
		{9, true, 0},   // A minor
		{4, true, 1},   // E minor
		{10, true, -5}, // Bb minor
	}
	for _, c := range cases {
		if got := fifthsFor(c.tonic, c.minor); got != c.want {
			t.Errorf("fifthsFor(%d,%v) = %d, want %d", c.tonic, c.minor, got, c.want)
		}
	}
}

func TestSpellUsesKeySignature(t *testing.T) {
	cases := []struct {
		key    int
		fifths int
		letter int
		alter  int
	}{
		{61, 2, 0, 1},   // C# in D major
		{61, -1, 1, -1}, // Db in F major
		{71, -1, 6, 0},  // B natural in F major
		{66, 2, 3, 1},   // F# in D major
	}
	for _, c := range cases {
		sp := spell(c.key, c.fifths)
		if sp.letter != c.letter || sp.alter != c.alter {
			t.Errorf("spell(%d,%d) = letter %d alter %d, want %d/%d",
				c.key, c.fifths, sp.letter, sp.alter, c.letter, c.alter)
		}
	}
}

func TestNoteValues(t *testing.T) {
	cases := []struct {
		dur   int64
		value float64
		dots  bool
	}{
		{960, 1, false},
		{480, 0.5, false},
		{240, 0.25, false},
		{1440, 1, true},
	}
	for _, c := range cases {
		v, dotted := noteValue(c.dur, 960)
		if v != c.value || dotted != c.dots {
			t.Errorf("noteValue(%d) = %v/%v, want %v/%v", c.dur, v, dotted, c.value, c.dots)
		}
	}
}

func TestBuildGrandStaff(t *testing.T) {
	s := &song.Song{
		PPQ:         960,
		DurationSec: 2,
		Tempo:       []song.TempoChange{{Tick: 0, BPM: 120}},
		Sigs:        []song.TimeSigChange{{Tick: 0, Num: 4, Den: 4}},
		Bars:        []float64{0, 2},
		Tracks: []song.Track{{
			Name: "Piano", Group: "piano", Visible: true,
			Notes: []song.Note{
				{Key: 48, Tick: 0, DurTicks: 960, Hand: "L"},
				{Key: 60, Tick: 0, DurTicks: 960, Hand: "R"},
			},
		}},
	}
	score, err := Build(s, Options{Width: 900})
	if err != nil {
		t.Fatal(err)
	}
	if len(score.Measures) == 0 {
		t.Fatal("no measures")
	}
	m := score.Measures[0]
	if len(m.Treble) != 1 || len(m.Bass) != 1 {
		t.Fatalf("staves: treble=%d bass=%d", len(m.Treble), len(m.Bass))
	}
	if m.Treble[0].Notes[0].Key != 60 || m.Bass[0].Notes[0].Key != 48 {
		t.Fatalf("notes placed on wrong staff: %+v %+v", m.Treble[0].Notes, m.Bass[0].Notes)
	}
}

func TestTripletDetection(t *testing.T) {
	cases := []struct {
		dur    int64
		base   float64
		tuplet int
	}{
		{320, 0.5, 3}, // eighth-note triplet
		{640, 1.0, 3}, // quarter-note triplet
		{960, 1.0, 0}, // plain quarter
	}
	for _, c := range cases {
		sh := noteShape(c.dur, 960)
		if sh.base != c.base || sh.tuplet != c.tuplet {
			t.Errorf("noteShape(%d) = base %v tuplet %d, want %v/%d", c.dur, sh.base, sh.tuplet, c.base, c.tuplet)
		}
	}
}

func TestRestGaps(t *testing.T) {
	m := &Measure{Start: 0, End: 3840}
	events := []*Event{
		{Start: 960, Dur: 960},
		{Start: 2880, Dur: 480},
	}
	want := []restGap{{Start: 0, Dur: 960}, {Start: 1920, Dur: 960}, {Start: 3360, Dur: 480}}
	gaps := restGaps(events, m)
	if len(gaps) != len(want) {
		t.Fatalf("gaps = %+v, want %+v", gaps, want)
	}
	for i := range want {
		if gaps[i] != want[i] {
			t.Fatalf("gap %d = %+v, want %+v", i, gaps[i], want[i])
		}
	}
	if got := restGaps(nil, m); len(got) != 1 || got[0] != (restGap{Start: 0, Dur: 3840}) {
		t.Fatalf("empty staff gaps = %+v", got)
	}
}

func TestAccidentalStateWithinBar(t *testing.T) {
	state := accidentalState{}
	fSharp := &Note{Letter: 3, Octave: 4, Alter: 1}
	if !state.altered(fSharp, 0) {
		t.Fatal("first F# should show a sharp")
	}
	if state.altered(fSharp, 0) {
		t.Fatal("repeated F# should not repeat its accidental")
	}
	fNatural := &Note{Letter: 3, Octave: 4, Alter: 0}
	if !state.altered(fNatural, 0) {
		t.Fatal("F natural after F# needs a natural sign")
	}
	if state.altered(fNatural, 0) {
		t.Fatal("repeated F natural should not repeat its accidental")
	}
	if !state.altered(fSharp, 0) {
		t.Fatal("F# after F natural needs a new sharp")
	}
	otherOctave := &Note{Letter: 3, Octave: 5, Alter: 1}
	if !state.altered(otherOctave, 0) {
		t.Fatal("F# in another octave needs its own accidental")
	}
}

func TestAccidentalStateFollowsKeySignature(t *testing.T) {
	state := accidentalState{}
	// F major has Bb in the signature.
	if state.altered(&Note{Letter: 6, Octave: 4, Alter: -1}, -1) {
		t.Fatal("B flat matching the key signature should not show an accidental")
	}
	if !state.altered(&Note{Letter: 6, Octave: 4, Alter: 0}, -1) {
		t.Fatal("B natural in F major needs a natural sign")
	}
}

func TestNoteSplitAcrossBarIsTied(t *testing.T) {
	s := &song.Song{
		PPQ:         960,
		DurationSec: 4,
		Tempo:       []song.TempoChange{{Tick: 0, BPM: 120}},
		Sigs:        []song.TimeSigChange{{Tick: 0, Num: 4, Den: 4}},
		Bars:        []float64{0, 2},
		Tracks: []song.Track{{
			Name: "Piano", Group: "piano", Visible: true,
			Notes: []song.Note{{Key: 60, Tick: 2880, DurTicks: 1920, Hand: "R"}},
		}},
	}
	score, err := Build(s, Options{Width: 900})
	if err != nil {
		t.Fatal(err)
	}
	if len(score.Measures) < 2 {
		t.Fatalf("measures = %d, want at least 2", len(score.Measures))
	}
	first := findEvent(score.Measures[0].Treble, 2880)
	if first == nil || !first.TieEnd {
		t.Fatalf("first half = %+v, want TieEnd", first)
	}
	second := findEvent(score.Measures[1].Treble, 3840)
	if second == nil || !second.TieStart {
		t.Fatalf("second half = %+v, want TieStart", second)
	}
}

func TestMidSystemSignatureWidth(t *testing.T) {
	s := &song.Song{
		PPQ:         960,
		DurationSec: 4,
		Tempo:       []song.TempoChange{{Tick: 0, BPM: 120}},
		Sigs: []song.TimeSigChange{
			{Tick: 0, Num: 4, Den: 4},
			{Tick: 3840, Num: 3, Den: 4},
		},
		Bars: []float64{0, 2},
	}
	score, err := Build(s, Options{Width: 1200})
	if err != nil {
		t.Fatal(err)
	}
	if len(score.Measures) < 2 {
		t.Fatalf("measures = %d", len(score.Measures))
	}
	if score.Measures[0].SigW != 0 {
		t.Errorf("first measure SigW = %v, want 0", score.Measures[0].SigW)
	}
	if score.Measures[1].SigW <= 0 {
		t.Errorf("time-signature change should reserve width, got %v", score.Measures[1].SigW)
	}
}

func TestBeamGroupingByBeat(t *testing.T) {
	m := &Measure{Start: 0, End: 3840, Num: 4, Den: 4}
	geoms := []eventGeom{
		{ev: &Event{Start: 0, Dur: 480}, minY: 50, maxY: 50, up: true},
		{ev: &Event{Start: 480, Dur: 480}, minY: 52, maxY: 52, up: true},
		{ev: &Event{Start: 960, Dur: 480}, minY: 54, maxY: 54, up: true},
	}
	layoutBeams(geoms, m, 960, 9)
	if !geoms[0].beamed || !geoms[1].beamed {
		t.Fatalf("same-beat eighths should beam: %+v", geoms[:2])
	}
	if geoms[0].beamY != geoms[1].beamY {
		t.Fatalf("beamed stems should share a beam height: %v vs %v", geoms[0].beamY, geoms[1].beamY)
	}
	if geoms[2].beamed {
		t.Fatal("eighth on the next beat should not join the previous group")
	}
}

func TestRenderSystemTile(t *testing.T) {
	font := "../../assets/fonts/Bravura.otf"
	if _, err := os.Stat(font); err != nil {
		t.Skip("Bravura font not available")
	}
	s := &song.Song{
		PPQ:         960,
		DurationSec: 2,
		Tempo:       []song.TempoChange{{Tick: 0, BPM: 120}},
		Sigs:        []song.TimeSigChange{{Tick: 0, Num: 4, Den: 4}},
		Bars:        []float64{0, 2},
		Tracks: []song.Track{{
			Name: "Piano", Group: "piano", Visible: true,
			Notes: []song.Note{
				{Key: 48, Tick: 0, DurTicks: 960, Hand: "L"},
				{Key: 60, Tick: 0, DurTicks: 960, Hand: "R"},
			},
		}},
	}
	opt := Options{
		Width:        900,
		FontPath:     font,
		TextFontPath: "../../assets/fonts/Inter-Bold.ttf",
		Scale:        2,
	}
	score, err := Build(s, opt)
	if err != nil {
		t.Fatal(err)
	}
	img, err := RenderSystem(score, 0, opt)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := img.Bounds().Dx(), int(score.Width*2); got != want {
		t.Errorf("tile width = %d, want %d", got, want)
	}
	if got, want := img.Bounds().Dy(), int(SystemHeight*2); got != want {
		t.Errorf("tile height = %d, want %d", got, want)
	}
}
