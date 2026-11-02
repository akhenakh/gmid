package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/library"

	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// LibraryWindow is the browsable song library.
func LibraryWindow(a *app.App) {
	if a.Lib != nil && !a.Lib.Scanned() && !a.UI.LibScanning {
		a.RefreshLibrary()
	}
	Modal(900, func() { a.UI.LibraryOpen = false }, func() {
		ModAttrs(Pad(18), Gap(10))
		Container(Attrs(Row, CrossMid, ExpandAcross()), func() {
			Container(Attrs(Gap(0)), func() {
				Label("Library", FontSize(20), FontWeight(WeightBold), TextColorVec(ColText))
				Caption(a.Prefs.LibraryFolder)
			})
			FlexSpacer()
			if a.UI.LibScanning {
				Caption("Scanning…")
			}
			if CtrlButton(TypRefresh, "Refresh", true) {
				a.RefreshLibrary()
			}
			if CtrlButton(TypFolderOpen, "Open a file…", true) {
				a.UI.FileBrowserOpen = true
			}
			if IconButton(TypTimes, "Close (Esc)", true) {
				a.UI.LibraryOpen = false
			}
		})
		Container(Attrs(Row, CrossMid, Gap(8), ExpandAcross()), func() {
			NextAccessLabel("Search library")
			TextInput(&a.UI.LibQuery)
			Caption("fuzzy search")
		})
		HDivider()
		Container(Attrs(Viewport, FixHeight(440)), func() {
			ScrollOnInput()
			ScrollBars()
			entries := a.SearchLibrary(a.UI.LibQuery)
			stars := a.FavouriteEntries()
			if a.UI.LibQuery == "" && len(stars) > 0 {
				SectionTitle("STARRED")
				for _, e := range stars {
					libraryRow(a, e)
				}
				HDivider()
				SectionTitle("ALL SONGS")
			}
			if len(entries) == 0 {
				Caption("No songs found in this folder.")
			}
			for _, e := range entries {
				libraryRow(a, e)
			}
		})
		HDivider()
		Container(Attrs(Row, Gap(8), CrossMid), func() {
			Caption(fmt.Sprintf("%d songs", len(a.SearchLibrary(a.UI.LibQuery))))
			FlexSpacer()
			SectionTitle("DEMO")
			for _, path := range a.DemoFiles() {
				if CtrlButton(TypMediaPlay, demoLabel(path), true) {
					a.UI.LibraryOpen = false
					if err := a.LoadFile(path); err != nil {
						a.Notify(err.Error())
					}
				}
			}
		})
	})
}

func demoLabel(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if i := strings.LastIndex(base, " - "); i >= 0 {
		return base[i+3:]
	}
	return base
}

func libraryRow(a *app.App, e library.Entry) {
	ContainerWithKey(e.Path, Attrs(Row, CrossMid, ExpandAcross(), Gap(8),
		Pad2(5, 6), Corners(6), BackgroundVec(ColPanel), BorderWidth(1),
		BorderColorVec(Alpha(ColBorder, .55))), func() {
		star := TypStarOutline
		if a.IsStarred(e.Path) {
			star = TypStar
		}
		if CtrlButton(star, "", true) {
			a.ToggleStar(e.Path)
		}
		Container(Attrs(Grow(1), Gap(0)), func() {
			ProcessButtonEvents(false)
			if IsHovered() {
				ModAttrs(BackgroundVec(ColPanel2))
			}
			if IsClicked() {
				a.UI.LibraryOpen = false
				if err := a.LoadFile(e.Path); err != nil {
					a.Notify(err.Error())
				}
			}
			Container(Attrs(Row, CrossMid, Gap(6)), func() {
				Label(e.Name, FontSize(12), TextColorVec(ColText))
				if e.Suspect && a.Prefs.TempoWarnings {
					Badge(ColAmber, 6)
					Caption("tempo")
				}
			})
			meta := fmt.Sprintf("%.0f BPM · %d tracks · %d notes", e.BPM, e.Tracks, e.Notes)
			if e.Error != "" {
				meta = e.Error
			}
			Caption(meta)
		})
		Caption(formatDuration(e.Duration))
	})
}
