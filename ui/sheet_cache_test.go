package ui

import (
	"path/filepath"
	"testing"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/config"

	. "go.hasen.dev/shirei"
)

func TestSheetCacheDoesNotRebuildPerFrame(t *testing.T) {
	t.Setenv("GMID_CONFIG", filepath.Join(t.TempDir(), "prefs.json"))
	UseFontFiles(config.FontPaths()...)
	Install(false)
	prefs := config.Defaults()
	prefs.Sheet = true
	a := app.New(prefs, "../assets", "")
	if err := a.LoadFile("../assets/demo/Scott Joplin - Entertainer.mid"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		RenderToImage(1000, 800, func() { Root(a) })
	}
	before := sheetBuilds
	for i := 0; i < 5; i++ {
		RenderToImage(1000, 800, func() { Root(a) })
	}
	if sheetBuilds != before {
		t.Fatalf("sheet rebuilt %d times across stable frames", sheetBuilds-before)
	}
}
