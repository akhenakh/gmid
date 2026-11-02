package ui

import (
	"fmt"
	"strconv"
	"strings"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// Palette mirrors Orfeo's dark surfaces closely enough that screenshots read
// the same way: near-black canvas, charcoal panels, muted text, amber tools.
var (
	ColCanvas   = Col("#0f0f12")
	ColPanel    = Col("#171720")
	ColPanel2   = Col("#1d1d28")
	ColToolbar  = Col("#121015")
	ColRail     = Col("#131318")
	ColBorder   = Col("#2e2e48")
	ColBorder2  = Col("#3b3b58")
	ColText     = Col("#c6c8c8")
	ColTextDim  = Col("#8b8b9e")
	ColTextMute = Col("#5f5f75")
	ColAmber    = Col("#e8a027")
	ColBlue     = Col("#6270a5")
	ColRose     = Col("#cb636c")
	ColTeal     = Col("#2dd4bf")
	ColSky      = Col("#5b9bd5")
	ColSage     = Col("#7fae6f")
	ColCoral    = Col("#e0895f")
	ColGold     = Col("#c99a4a")
	ColWhite    = Col("#ffffff")
	ColBlack    = Col("#000000")
)

// TrackPalette is the user-facing track colour picker palette.
var TrackPalette = []string{
	"#e8a027", "#2dd4bf", "#6270A5", "#d1667a", "#5b9bd5",
	"#7fae6f", "#e0895f", "#CB636C", "#5b7c99", "#c99a4a",
}

// Col converts "#rrggbb" (or "#rgb") to Shirei's HSLA Vec4.
func Col(hex string) Vec4 {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return Vec4{0, 0, 100, 1}
	}
	return RGB(r, g, b)
}

// RGBA builds an HSL colour from 0..255 channels.
func RGB(r, g, b float32) Vec4 {
	rf, gf, bf := r/255, g/255, b/255
	max := max3(rf, gf, bf)
	min := min3(rf, gf, bf)
	l := (max + min) / 2
	var h, s float32
	if max != min {
		d := max - min
		if l > 0.5 {
			s = d / (2 - max - min)
		} else {
			s = d / (max + min)
		}
		switch max {
		case rf:
			h = (gf - bf) / d
			if gf < bf {
				h += 6
			}
		case gf:
			h = (bf-rf)/d + 2
		default:
			h = (rf-gf)/d + 4
		}
		h *= 60
	}
	return Vec4{h, s * 100, l * 100, 1}
}

// Alpha returns c with a new alpha.
func Alpha(c Vec4, a float32) Vec4 { c[3] = a; return c }

// Shade lightens (positive) or darkens (negative) an HSLA colour.
func Shade(c Vec4, delta float32) Vec4 {
	c[2] = clampf(c[2]+delta, 0, 100)
	return c
}

func parseHex(hex string) (float32, float32, float32, bool) {
	s := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return float32(v >> 16 & 0xff), float32(v >> 8 & 0xff), float32(v & 0xff), true
}

// Hex converts HSLA back to "#rrggbb" (opaque) for storing track colours.
func Hex(c Vec4) string {
	r, g, b := hslToRGB(c[0], c[1]/100, c[2]/100)
	return fmt.Sprintf("#%02x%02x%02x", int(r*255+0.5), int(g*255+0.5), int(b*255+0.5))
}

func hslToRGB(h, s, l float32) (float32, float32, float32) {
	h = float32((int(h)%360 + 360) % 360)
	c := (1 - absf(2*l-1)) * s
	x := c * (1 - absf(float32(int(h/60)%2)-1))
	m := l - c/2
	var r, g, b float32
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
	return r + m, g + m, b + m
}

// Install applies the gmid palette to Shirei's widgets. Call once at startup.
func Install(warm bool) {
	s := DarkColorScheme()
	if warm {
		s = WarmDarkColorScheme()
		s.Surfaces.Canvas = SurfaceColors{Background: Col("#12100e"), Text: ColText, Border: Col("#3a322c")}
		s.Surfaces.Panel = SurfaceColors{Background: Col("#1b1714"), Text: ColText, Border: Col("#403830")}
		s.Surfaces.Toolbar = SurfaceColors{Background: Col("#181411"), Text: ColText, Border: Col("#3a322c")}
	} else {
		s.Surfaces.Canvas = SurfaceColors{Background: ColCanvas, Text: ColText, Border: ColBorder}
		s.Surfaces.Panel = SurfaceColors{Background: ColPanel, Text: ColText, Border: ColBorder}
		s.Surfaces.Toolbar = SurfaceColors{Background: ColToolbar, Text: ColText, Border: ColBorder}
	}
	s.FocusRing = ColAmber
	s.Buttons.Primary = ButtonStyleWithAccent(s.Buttons.Primary, ColAmber)
	s.Slider.Track = ColAmber
	s.Slider.Handle = Col("#e6e6ee")
	s.TextInput.Background = Shade(ColCanvas, -2)
	s.Scrim = Vec4{0, 0, 0, .7}
	SetDarkColorScheme(s)
	SetDarkMode(true)
	SetLightColorScheme(s)
}

func max3(a, b, c float32) float32 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

func min3(a, b, c float32) float32 {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
