package theory

import (
	"sort"
	"strings"
)

// ChordType is one entry in the detector's vocabulary.
type ChordType struct {
	Suffix    string
	Intervals []int
}

// ChordTypes is a curated common-chord vocabulary, ordered by priority.
var ChordTypes = []ChordType{
	{"", []int{0, 4, 7}},
	{"m", []int{0, 3, 7}},
	{"dim", []int{0, 3, 6}},
	{"aug", []int{0, 4, 8}},
	{"5", []int{0, 7}},
	{"sus2", []int{0, 2, 7}},
	{"sus4", []int{0, 5, 7}},
	{"6", []int{0, 4, 7, 9}},
	{"m6", []int{0, 3, 7, 9}},
	{"7", []int{0, 4, 7, 10}},
	{"maj7", []int{0, 4, 7, 11}},
	{"m7", []int{0, 3, 7, 10}},
	{"m7b5", []int{0, 3, 6, 10}},
	{"dim7", []int{0, 3, 6, 9}},
	{"add9", []int{0, 2, 4, 7}},
	{"madd9", []int{0, 2, 3, 7}},
	{"9", []int{0, 2, 4, 7, 10}},
	{"maj9", []int{0, 2, 4, 7, 11}},
	{"m9", []int{0, 2, 3, 7, 10}},
	{"7sus4", []int{0, 5, 7, 10}},
	{"11", []int{0, 2, 4, 5, 7, 10}},
	{"m11", []int{0, 2, 3, 5, 7, 10}},
	{"13", []int{0, 2, 4, 7, 9, 10}},
	{"maj13", []int{0, 2, 4, 7, 9, 11}},
	{"m13", []int{0, 2, 3, 7, 9, 10}},
	{"7b9", []int{0, 1, 4, 7, 10}},
	{"7#9", []int{0, 3, 4, 7, 10}},
	{"7#11", []int{0, 4, 6, 7, 10}},
}

var chordNames = []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}

// DetectChord names the set of MIDI notes. The result is root-position with a
// slash when the bass is not the root. Returns "—" for fewer than three notes.
func DetectChord(midis []int) string {
	if len(midis) < 3 {
		return "—"
	}
	pcs := make([]int, 0, len(midis))
	seen := map[int]bool{}
	for _, m := range midis {
		pc := ((m % 12) + 12) % 12
		if !seen[pc] {
			seen[pc] = true
			pcs = append(pcs, pc)
		}
	}
	if len(pcs) < 3 {
		return "—"
	}
	sort.Ints(pcs)
	bass := ((midis[0] % 12) + 12) % 12
	for _, m := range midis {
		pc := ((m % 12) + 12) % 12
		if pc < bass {
			bass = pc
		}
	}

	bestScore := -1
	bestName := "—"
	for root := 0; root < 12; root++ {
		set := map[int]bool{}
		for _, pc := range pcs {
			set[((pc-root)%12+12)%12] = true
		}
		for _, ct := range ChordTypes {
			want := map[int]bool{}
			for _, iv := range ct.Intervals {
				want[iv] = true
			}
			if len(want) != len(set) {
				continue
			}
			ok := true
			for iv := range want {
				if !set[iv] {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
			score := 100
			if root == bass {
				score += 20
			}
			// Prefer simpler chords when several match.
			score -= len(ct.Intervals)
			if score > bestScore {
				bestScore = score
				name := chordNames[root] + ct.Suffix
				if bass != root {
					name += "/" + chordNames[bass]
				}
				bestName = name
			}
		}
	}
	if bestScore < 0 {
		return "—"
	}
	return bestName
}

// ChordTones returns the MIDI notes of a chord in root position.
func ChordTones(root, octave int, intervals []int) []int {
	base := (octave+1)*12 + root
	out := make([]int, len(intervals))
	for i, iv := range intervals {
		out[i] = base + iv
	}
	return out
}

// FindChordType returns the chord type with the given suffix.
func FindChordType(suffix string) (ChordType, bool) {
	for _, ct := range ChordTypes {
		if ct.Suffix == suffix {
			return ct, true
		}
	}
	return ChordType{}, false
}

// FormatChord spells a root and suffix in the given accidental style.
func FormatChord(root int, suffix string, accidentals string) string {
	name := NoteName(root, "english", accidentals)
	if suffix == "" {
		return name
	}
	return name + suffix
}

// ParseChordSymbol splits "C#m7/G" into root, suffix, bass.
func ParseChordSymbol(sym string) (root string, suffix string, bass string, ok bool) {
	s := strings.TrimSpace(sym)
	if s == "" {
		return "", "", "", false
	}
	if i := strings.IndexByte(s, '/'); i >= 0 {
		bass = s[i+1:]
		s = s[:i]
	}
	i := 1
	if i < len(s) && (s[i] == '#' || s[i] == 'b') {
		i++
	}
	root, suffix = s[:i], s[i:]
	return root, suffix, bass, root != ""
}

// NoteName spells a pitch class without an octave.
func NoteName(midi int, system, accidentals string) string {
	pc := ((midi % 12) + 12) % 12
	sharp := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	flat := []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	if accidentals == "sharp" {
		return sharp[pc]
	}
	return flat[pc]
}
