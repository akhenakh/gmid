package theory

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// TrackPalette is the colour picker palette.
var TrackPalette = []string{
	"#e8a027", "#2dd4bf", "#6270A5", "#d1667a", "#5b9bd5",
	"#7fae6f", "#e0895f", "#CB636C", "#5b7c99", "#c99a4a",
}

// PianoFamilyColors are the deterministic first three piano-family colours.
var PianoFamilyColors = []string{TrackPalette[2], TrackPalette[7], TrackPalette[0]}

var autoTrackColors = []string{
	"#2dd4bf", "#5b9bd5", "#7fae6f", "#a78bfa", "#b5c45a",
	"#c47bb8", "#8f8f6a", "#e8d77a", "#d6b8e8",
}

// DefaultHands are Orfeo's stock hand pair.
const (
	DefaultHandLH = "#6270A5"
	DefaultHandRH = "#CB636C"
)

// PianoFamilyColor returns the starting colour for the nth piano-family track.
func PianoFamilyColor(index int, lh, rh string) string {
	if index == 0 {
		return lh
	}
	if index == 1 {
		return rh
	}
	if index < len(PianoFamilyColors) {
		return PianoFamilyColors[index]
	}
	deck := autoDeck(lh, rh)
	return deck[(index-len(PianoFamilyColors))%len(deck)]
}

// DistinctTrackColor picks the colour farthest from the hands and from colours
// already assigned, deterministically per track order.
func DistinctTrackColor(used []string, lh, rh string) string {
	deck := autoDeck(lh, rh)
	fewest := math.MaxInt
	uses := make(map[string]int, len(deck))
	for _, c := range deck {
		n := 0
		for _, u := range used {
			if sameColor(u, c) {
				n++
			}
		}
		uses[c] = n
		if n < fewest {
			fewest = n
		}
	}
	avoid := append([]string{lh, rh}, used...)
	best := deck[0]
	bestD := -1.0
	for _, c := range deck {
		if uses[c] != fewest {
			continue
		}
		var d float64
		if fewest == 0 {
			d = math.MaxFloat64
			for _, u := range avoid {
				d = math.Min(d, colorDistance(c, u))
			}
		} else if len(used) > 0 {
			d = colorDistance(c, used[len(used)-1])
		}
		if d > bestD {
			best, bestD = c, d
		}
	}
	return best
}

// FollowedHandColor keeps a piano/organ track on the hand pair when the user
// changes hand colours, matching Orfeo's behaviour.
func FollowedHandColor(color, lh, rh string) string {
	switch {
	case sameColor(color, DefaultHandLH):
		return lh
	case sameColor(color, DefaultHandRH):
		return rh
	}
	return color
}

func autoDeck(lh, rh string) []string {
	out := make([]string, 0, len(autoTrackColors))
	for _, c := range autoTrackColors {
		if colorDistance(c, lh) >= 0.1 && colorDistance(c, rh) >= 0.1 {
			out = append(out, c)
		}
	}
	if len(out) < 4 {
		return append([]string(nil), autoTrackColors...)
	}
	return out
}

func sameColor(a, b string) bool { return strings.EqualFold(a, b) }

// colorDistance approximates perceptual distance in OKLab.
func colorDistance(a, b string) float64 {
	r1, g1, b1 := hexRGB(a)
	r2, g2, b2 := hexRGB(b)
	l1, a1, bb1 := oklab(r1, g1, b1)
	l2, a2, bb2 := oklab(r2, g2, b2)
	return math.Sqrt((l1-l2)*(l1-l2) + (a1-a2)*(a1-a2) + (bb1-bb2)*(bb1-bb2))
}

func hexRGB(hex string) (float64, float64, float64) {
	s := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return 0, 0, 0
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0
	}
	return float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255
}

func oklab(r, g, b float64) (float64, float64, float64) {
	r = srgbToLinear(r)
	g = srgbToLinear(g)
	b = srgbToLinear(b)
	l := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b
	l = math.Cbrt(l)
	m = math.Cbrt(m)
	s = math.Cbrt(s)
	return 0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

// HSLToHex converts HSL (h degrees, s/l 0..1) to a hex colour.
func HSLToHex(h, s, l float64) string {
	h = math.Mod(h, 360)
	if h < 0 {
		h += 360
	}
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
	return fmt.Sprintf("#%02x%02x%02x", int((r+m)*255+0.5), int((g+m)*255+0.5), int((b+m)*255+0.5))
}
