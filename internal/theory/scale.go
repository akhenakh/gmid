package theory

import "strings"

// ScaleDef is one scale type in the Scale Explorer.
type ScaleDef struct {
	ID        string
	Name      string
	Intervals []int
	BlueNote  int // interval or -1
}

// Scales are the ten scale types offered by the explorer.
var Scales = []ScaleDef{
	{"major", "Major (Ionian)", []int{0, 2, 4, 5, 7, 9, 11}, -1},
	{"minor", "Natural Minor", []int{0, 2, 3, 5, 7, 8, 10}, -1},
	{"harmonic-minor", "Harmonic Minor", []int{0, 2, 3, 5, 7, 8, 11}, -1},
	{"melodic-minor", "Melodic Minor", []int{0, 2, 3, 5, 7, 9, 11}, -1},
	{"dorian", "Dorian", []int{0, 2, 3, 5, 7, 9, 10}, -1},
	{"phrygian", "Phrygian", []int{0, 1, 3, 5, 7, 8, 10}, -1},
	{"lydian", "Lydian", []int{0, 2, 4, 6, 7, 9, 11}, -1},
	{"mixolydian", "Mixolydian", []int{0, 2, 4, 5, 7, 9, 10}, -1},
	{"locrian", "Locrian", []int{0, 1, 3, 5, 6, 8, 10}, -1},
	{"blues", "Blues", []int{0, 3, 5, 6, 7, 10}, 3},
}

// ScaleByID finds a scale definition.
func ScaleByID(id string) ScaleDef {
	for _, s := range Scales {
		if s.ID == id {
			return s
		}
	}
	return Scales[0]
}

// DiatonicChord is one chord in a scale, with its Roman numeral.
type DiatonicChord struct {
	Degree int
	Roman  string
	Name   string
	Keys   []int // MIDI notes at the given register
}

// DiatonicChords stacks thirds through the scale for seven degrees.
func DiatonicChords(root, octave int, scale ScaleDef) []DiatonicChord {
	out := make([]DiatonicChord, 0, 7)
	n := len(scale.Intervals)
	steps := []int{0, 2, 4}
	for degree := 0; degree < 7; degree++ {
		tones := make([]int, 0, 3)
		for _, st := range steps {
			idx := (degree + st) % n
			oct := (degree + st) / n
			tones = append(tones, root+scale.Intervals[idx]+oct*12)
		}
		suffix, _ := qualityOf(tones)
		roman := RomanFor(degree, suffix)
		name := FormatChord(((root+scale.Intervals[degree])%12+12)%12, suffix, "flat")
		keys := make([]int, 3)
		for i, t := range tones {
			keys[i] = (octave+1)*12 + ((t%12)+12)%12
		}
		out = append(out, DiatonicChord{Degree: degree, Roman: roman, Name: name, Keys: keys})
	}
	return out
}

func qualityOf(tones []int) (suffix, roman string) {
	if len(tones) < 3 {
		return "", ""
	}
	third := ((tones[1]-tones[0])%12 + 12) % 12
	fifth := ((tones[2]-tones[0])%12 + 12) % 12
	switch {
	case third == 4 && fifth == 7:
		return "", "I"
	case third == 3 && fifth == 7:
		return "m", "i"
	case third == 3 && fifth == 6:
		return "dim", "°"
	case third == 4 && fifth == 8:
		return "aug", "+"
	case third == 2 && fifth == 7:
		return "sus2", "sus2"
	case third == 5 && fifth == 7:
		return "sus4", "sus4"
	}
	return "", ""
}

// RomanFor renders a degree number with the scale's chord case.
func RomanFor(degree int, suffix string) string {
	base := []string{"I", "II", "III", "IV", "V", "VI", "VII"}[degree%7]
	if suffix == "dim" {
		return strings.ToLower(base) + "°"
	}
	if suffix == "aug" {
		return base + "+"
	}
	if suffix != "" && suffix[0] == 'm' && !strings.HasPrefix(suffix, "maj") {
		return strings.ToLower(base)
	}
	return base
}

// Progression is one of the nine styles in the Chord Explorer.
type Progression struct {
	Name   string
	Chords []int // scale degrees, -1 for a chromatic chord
	Suffix []string
}

// Progressions are played over the selected key.
var Progressions = []Progression{
	{"Pop", []int{0, 4, 5, 3}, []string{"", "", "m", ""}},
	{"Sad", []int{5, 3, 0, 4}, []string{"m", "", "", ""}},
	{"Jazz", []int{1, 4, 0, 0}, []string{"m", "", "maj7", "maj7"}},
	{"Blues", []int{0, 3, 0, 4}, []string{"", "", "", "7"}},
	{"Canon", []int{0, 4, 5, 2}, []string{"", "", "", "m"}},
	{"Andalusian", []int{0, 6, 5, 4}, []string{"m", "", "", ""}},
	{"Doo-wop", []int{0, 5, 3, 4}, []string{"", "m", "", ""}},
	{"Modal", []int{0, 1, 0, 6}, []string{"", "", "", ""}},
	{"Turnaround", []int{0, 5, 1, 4}, []string{"", "m", "m", "7"}},
}

// ProgressionChord returns the chord for a step of a progression.
func ProgressionChord(root, octave int, scale ScaleDef, p Progression, step int) (string, []int) {
	deg := p.Chords[step%len(p.Chords)]
	suffix := ""
	if len(p.Suffix) > 0 {
		suffix = p.Suffix[step%len(p.Suffix)]
	}
	sem := scale.Intervals[deg%len(scale.Intervals)]
	name := FormatChord(((root+sem)%12+12)%12, suffix, "flat")
	ct, ok := FindChordType(suffix)
	if !ok {
		ct = ChordType{Suffix: suffix, Intervals: []int{0, 4, 7}}
	}
	ivs := ct.Intervals
	if len(ivs) < 3 {
		ivs = []int{0, 4, 7}
	}
	keys := make([]int, 3)
	for i, iv := range ivs[:3] {
		keys[i] = (octave+1)*12 + ((root+sem+iv)%12+12)%12
	}
	return name, keys
}
