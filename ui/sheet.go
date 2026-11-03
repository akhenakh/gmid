package ui

import (
	"fmt"
	"image"
	"path/filepath"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/notation"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

const sheetHeight = 214

type sheetView struct {
	key   string
	score *notation.Score
	opt   notation.Options
	tiles map[int]*image.RGBA
}

var (
	sheetCache     = map[string]*sheetView{}
	sheetCacheKeys []string
	sheetBuilds    int
)

var (
	sheetPaper = Col("#f7f3e8")
	sheetInk   = Col("#4a4135")
)

// SheetPanel shows an optional grand-staff reduction above the falling notes.
func SheetPanel(a *app.App) {
	Container(Attrs(FixHeight(sheetHeight), ExpandAcross(), Corners(9), Clip,
		BackgroundVec(sheetPaper), BorderWidth(1), BorderColorVec(ColBorder2)), func() {
		Container(Attrs(Row, CrossMid, FixHeight(26), ExpandAcross(), Pad2(0, 8), Gap(8),
			BackgroundVec(Col("#e9e2d3"))), func() {
			Label("SHEET", FontSize(10), FontWeight(WeightBold), TextColorVec(sheetInk))
			Label(a.Song.Name, FontSize(10), TextColorVec(sheetInk))
			FlexSpacer()
			Label("Hand colours", FontSize(10), TextColorVec(sheetInk))
			hand := a.Prefs.SheetHandColors
			ToggleSwitch(&a.Prefs.SheetHandColors)
			if hand != a.Prefs.SheetHandColors {
				a.SavePrefs()
			}
			Label("Follow", FontSize(10), TextColorVec(sheetInk))
			follow := a.Prefs.SheetFollow
			ToggleSwitch(&a.Prefs.SheetFollow)
			if follow != a.Prefs.SheetFollow {
				a.SavePrefs()
			}
			if IconButton(TypTimes, "Hide sheet", true) {
				a.Prefs.Sheet = false
				a.SavePrefs()
			}
		})
		Container(Attrs(Viewport, Grow(1)), func() {
			ScrollOnInput()
			ScrollBars()
			width := int(GetContentWidth() - 18)
			if width < 240 {
				width = 240
			}
			// Quantize so a scrollbar appearing/disappearing by a few pixels
			// does not rebuild a large notation bitmap on every frame.
			width = width / 64 * 64
			view, err := sheetFor(a, width)
			if err != nil {
				Label(err.Error(), FontSize(12), TextColorVec(sheetInk))
				return
			}
			bar := a.Song.BarNumberAt(a.Transport.Time) - 1
			if bar < 0 {
				bar = 0
			}
			if bar >= len(view.score.Bars) {
				bar = len(view.score.Bars) - 1
			}
			if a.UI.SheetSong != a.Song.Identity {
				a.UI.SheetSong = a.Song.Identity
				a.UI.SheetBar = -1
			}
			if a.Prefs.SheetFollow && bar != a.UI.SheetBar && bar >= 0 {
				r := view.score.Bars[bar]
				y := r.Y - 12
				if y < 0 {
					y = 0
				}
				SetScrollOffset(Vec2{0, float32(y)})
				a.UI.SheetBar = bar
			}
			h := float32(view.score.Height)
			viewH := float64(GetContentHeight())
			if viewH <= 0 {
				viewH = sheetHeight - 26
			}
			scrollY := float64(GetScrollOffset()[1])
			Container(Attrs(FixSize(float32(width), h)), func() {
				for si := range view.score.Systems {
					sys := view.score.Systems[si]
					if sys.Y+notation.SystemHeight < scrollY || sys.Y > scrollY+viewH {
						continue
					}
					tile := sheetTile(view, si)
					if tile == nil {
						continue
					}
					Container(Attrs(Float(0, float32(sys.Y)),
						FixSize(float32(width), float32(notation.SystemHeight))), func() {
						ImageViewAt(UseImage(fmt.Sprintf("%s/%d", view.key, si), tile),
							Vec2{float32(width), float32(notation.SystemHeight)})
					})
				}
				if bar >= 0 && bar < len(view.score.Bars) {
					r := view.score.Bars[bar]
					Element(Attrs(Float(float32(r.X), float32(r.Y)),
						FixSize(float32(r.W), float32(r.H)),
						BackgroundVec(Alpha(ColAmber, .16)), NoAnimate))
				}
			})
		})
	})
}

func sheetFor(a *app.App, width int) (*sheetView, error) {
	key := fmt.Sprintf("%p|%s|%d|%d|%t|%t|%s|%s",
		a.Song, a.Song.Identity, a.Song.NoteCount, width,
		a.Prefs.SheetHandColors, a.Prefs.HandAssignment, a.Prefs.HandLH, a.Prefs.HandRH)
	if v := sheetCache[key]; v != nil {
		return v, nil
	}
	opt := notation.Options{
		Width:        float64(width),
		HandLH:       a.Prefs.HandLH,
		HandRH:       a.Prefs.HandRH,
		ShowHands:    a.Prefs.SheetHandColors,
		FontPath:     filepath.Join(a.Assets, "fonts", "Bravura.otf"),
		TextFontPath: filepath.Join(a.Assets, "fonts", "Inter-Bold.ttf"),
		Scale:        2,
	}
	score, err := notation.Build(a.Song, opt)
	if err != nil {
		return nil, err
	}
	v := &sheetView{key: key, score: score, opt: opt, tiles: map[int]*image.RGBA{}}
	sheetBuilds++
	sheetCache[key] = v
	sheetCacheKeys = append(sheetCacheKeys, key)
	for len(sheetCacheKeys) > 1 {
		delete(sheetCache, sheetCacheKeys[0])
		sheetCacheKeys = sheetCacheKeys[1:]
	}
	return v, nil
}

func sheetTile(v *sheetView, index int) *image.RGBA {
	if img := v.tiles[index]; img != nil {
		return img
	}
	img, err := notation.RenderSystem(v.score, index, v.opt)
	if err != nil {
		return nil
	}
	v.tiles[index] = img
	return img
}
