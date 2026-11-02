package ui

import (
	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"

	"github.com/akhenakh/gmid/internal/app"
)

// EffectiveKeyRange returns the keyboard/roll range for the loaded song. With
// FitSongRange on it trims to the pitches the song uses (plus context), never
// beyond the configured 61/73/88-key span, so notes use the available width
// instead of clustering around the middle of a full 88-key keyboard.
func EffectiveKeyRange(a *app.App) (int, int) {
	baseMin, baseMax := KeyRange(a.Prefs.KeyRange)
	if !a.Prefs.FitSongRange || a.Song == nil || a.SongMaxKey <= 0 {
		return baseMin, baseMax
	}
	lo, hi := a.SongMinKey+a.Transport.Transpose-2, a.SongMaxKey+a.Transport.Transpose+2
	if hi-lo < 24 {
		mid := (lo + hi) / 2
		lo, hi = mid-12, mid+12
	}
	if lo < baseMin {
		lo = baseMin
	}
	if hi > baseMax {
		hi = baseMax
	}
	if hi <= lo {
		return baseMin, baseMax
	}
	return lo, hi
}

// KeyRange returns the MIDI range for a 61/73/88-key keyboard.
func KeyRange(size int) (min, max int) {
	switch size {
	case 61:
		return 36, 96
	case 73:
		return 28, 103
	default:
		return 21, 108
	}
}

// IsBlackKey reports whether a MIDI note is a black key.
func IsBlackKey(midi int) bool {
	switch ((midi % 12) + 12) % 12 {
	case 1, 3, 6, 8, 10:
		return true
	}
	return false
}

// KeyLayout is the horizontal geometry of one key in pixels.
type KeyLayout struct {
	X, W float32
}

// BuildKeyLayout returns pixel geometry aligned across the roll and keyboard.
func BuildKeyLayout(width float32, min, max int) []KeyLayout {
	n := 0
	for m := min; m <= max; m++ {
		if !IsBlackKey(m) {
			n++
		}
	}
	out := make([]KeyLayout, max-min+1)
	if n == 0 || width <= 0 {
		return out
	}
	ww := width / float32(n)
	wi := 0
	for m := min; m <= max; m++ {
		i := m - min
		if !IsBlackKey(m) {
			out[i] = KeyLayout{X: float32(wi) * ww, W: ww}
			wi++
		} else {
			out[i] = KeyLayout{X: (float32(wi) - 0.30) * ww, W: 0.60 * ww}
		}
	}
	return out
}

// KeyAt returns the MIDI note under x, preferring the black key when the point
// falls inside both.
func KeyAt(x float32, layout []KeyLayout, min, max int) int {
	for m := max; m >= min; m-- {
		k := layout[m-min]
		if !IsBlackKey(m) {
			continue
		}
		if x >= k.X && x <= k.X+k.W {
			return m
		}
	}
	if len(layout) == 0 {
		return min
	}
	w := layout[0].W
	idx := int(x / w)
	if idx < 0 {
		return min
	}
	// Skip black-key slots while walking white keys.
	seen := 0
	for m := min; m <= max; m++ {
		if IsBlackKey(m) {
			continue
		}
		if seen == idx {
			return m
		}
		seen++
	}
	return max
}

// KeyState is one sounding key: the colour it was last lit with.
type KeyState struct {
	Color  Vec4
	Labels []string
}

// PianoKeyboard draws the on-screen keyboard. nolint:gocognit
func PianoKeyboard(min, max int, height float32, active map[int]KeyState,
	showNames, showOctaves bool, onDown, onUp func(int)) {

	Container(Attrs(FixHeight(height), ExpandAcross(), NoAnimate, Clip,
		BackgroundVec(Col("#08080b"))), func() {
		w := GetContentWidth()
		h := GetContentHeight()

		// Interaction is handled at the bed level so dragging across keys
		// glissandos without per-key containers fighting for the pointer.
		lastStored := Use[int]("kb-last")
		st := ProcessButtonEvents(false)
		if onDown != nil || onUp != nil {
			rect := GetScreenRect()
			x := GetInputState().MousePoint[0] - rect.Origin[0]
			layout := BuildKeyLayout(rect.Size[0], min, max)
			pressedKey := KeyAt(x, layout, min, max)
			if st.Active && x >= 0 && x <= rect.Size[0] {
				key := pressedKey
				if *lastStored-1 != key {
					if *lastStored > 0 && onUp != nil {
						onUp(*lastStored - 1)
					}
					if onDown != nil {
						onDown(key)
					}
					*lastStored = key + 1
				}
			} else if *lastStored > 0 {
				if onUp != nil {
					onUp(*lastStored - 1)
				}
				*lastStored = 0
			}
		}

		layout := BuildKeyLayout(w, min, max)
		blackH := h * 0.62

		// White keys first.
		for m := min; m <= max; m++ {
			if IsBlackKey(m) {
				continue
			}
			k := layout[m-min]
			activeState, lit := active[m]
			activeCol := activeState.Color
			bg := Col("#e3e3ea")
			border := Col("#8a8a9c")
			if lit {
				bg = activeCol
				border = Shade(activeCol, -18)
			}
			Container(Attrs(Float(k.X, 0), FixSize(k.W-1, h), NoAnimate,
				BackgroundVec(bg), Corners4(0, 0, 3, 3),
				BorderWidth(1), BorderColorVec(border)), func() {
				if lit && len(activeState.Labels) > 0 {
					BadgeLabel(activeState.Labels[0], k.W)
				}
				if showNames || showOctaves {
					labelCol := Col("#55556a")
					if lit {
						if bg[2] > 60 {
							labelCol = Col("#1a1a24")
						} else {
							labelCol = Alpha(ColWhite, .88)
						}
					}
					Container(Attrs(FixSize(k.W-1, 20), Float(0, h-19),
						MainAlign(AlignMiddle), CrossAlign(AlignEnd), Pad2(0, 1)), func() {
						name, fs := fitKeyLabel(m, k.W, showNames, showOctaves)
						if name != "" {
							Label(name, FontSize(fs), TextColorVec(labelCol))
						}
					})
				}
			})
		}

		// Black keys on top.
		for m := min; m <= max; m++ {
			if !IsBlackKey(m) {
				continue
			}
			k := layout[m-min]
			activeState, lit := active[m]
			activeCol := activeState.Color
			bg := Col("#14141c")
			border := Col("#000000")
			if lit {
				bg = Shade(activeCol, -12)
				border = Shade(activeCol, -25)
			}
			Container(Attrs(Float(k.X, 0), Z(2), FixSize(k.W, blackH), NoAnimate,
				BackgroundVec(bg), Corners4(0, 0, 2, 2),
				BorderWidth(1), BorderColorVec(border),
				BoxShadow(3)), func() {
				if lit && len(activeState.Labels) > 0 {
					BadgeLabel(activeState.Labels[0], k.W)
				}
			})
		}
	})
}

// BadgeLabel draws the L/R hand badge at the top of a lit key.
func BadgeLabel(text string, keyWidth float32) {
	size := float32(14)
	if keyWidth < 18 {
		size = 10
	}
	Container(Attrs(Float((keyWidth-size)/2, 4), FixSize(size, size),
		Corners(size/2), BackgroundVec(Alpha(ColBlack, .35)),
		MainAlign(AlignMiddle), CrossAlign(AlignMiddle)), func() {
		Label(text, FontSize(9), FontWeight(WeightBold), TextColorVec(ColWhite))
	})
}

// KeyLabel returns the label drawn at the foot of a key.
func KeyLabel(midi int, names, octaves bool) string {
	pc := ((midi % 12) + 12) % 12
	if !names && !octaves {
		return ""
	}
	if !names {
		if pc != 0 {
			return ""
		}
		return itoa(midi/12 - 1)
	}
	name := NoteName(midi, "english", "flat")
	if octaves {
		return name + itoa(midi/12-1)
	}
	return name
}

// fitKeyLabel returns the key label and a font size that keeps the whole token
// on one line. Shirei soft-wraps text at the container width, which split
// "E6" into "E" and "6" on narrow keys. If even a minimum font cannot fit a
// name, fall back to the octave digit on C keys (and hide it on tiny keys).
func fitKeyLabel(midi int, keyWidth float32, names, octaves bool) (string, float32) {
	name := KeyLabel(midi, names, octaves)
	fs := float32(9)
	if name != "" && keyWidth > 2 {
		// Inter is wider for some letters (C, G) than others; use a slightly
		// different per-rune budget for accidentals so nothing soft-wraps.
		per := float32(0.75)
		if len(name) >= 3 {
			per = 0.66
		}
		fit := (keyWidth - 3) / (per * float32(len(name)))
		if fit < fs {
			fs = fit
		}
	}
	if fs < 6 {
		name = KeyLabel(midi, false, octaves)
		fs = 9
		if keyWidth < 11 {
			name = ""
		}
	}
	return name, fs
}

func itoa(v int) string {
	if v < 0 {
		return "-" + itoa(-v)
	}
	if v < 10 {
		return string(rune('0' + v))
	}
	return itoa(v/10) + string(rune('0'+v%10))
}

// NoteName spells a MIDI note in the selected naming system.
func NoteName(midi int, system, accidentals string) string {
	pc := ((midi % 12) + 12) % 12
	sharp := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	flat := []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	ceSharp := []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "B", "H"}
	ceFlat := []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "B", "H"}
	solSharp := []string{"Do", "Do#", "Re", "Re#", "Mi", "Fa", "Fa#", "Sol", "Sol#", "La", "La#", "Si"}
	solFlat := []string{"Do", "Reb", "Re", "Mib", "Mi", "Fa", "Solb", "Sol", "Lab", "La", "Sib", "Si"}
	if system == "hidden" {
		return ""
	}
	useSharp := accidentals == "sharp"
	switch system {
	case "central-european":
		if useSharp {
			return ceSharp[pc]
		}
		return ceFlat[pc]
	case "solfege":
		if useSharp {
			return solSharp[pc]
		}
		return solFlat[pc]
	default:
		if useSharp {
			return sharp[pc]
		}
		return flat[pc]
	}
}
