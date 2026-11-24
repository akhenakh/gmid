package ui

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/config"

	shirei "go.hasen.dev/shirei"
)

var addFontsOnce sync.Once

// addFrame builds LibraryAddWindow headlessly with a synthetic key, mirroring
// the shirei widget tests. There is no pointer, so only keyboard paths run.
func addFrame(scope any, key shirei.KeyCode, fn func()) {
	shirei.GetHost().WindowSize = shirei.Vec2{1200, 800}
	shirei.GetInputState().MousePoint = shirei.Vec2{-1e5, -1e5}
	shirei.GetInputState().Modifiers = 0
	shirei.GetFrameInput().Mouse = 0
	shirei.GetFrameInput().Scroll = shirei.Vec2{}
	shirei.GetFrameInput().Motion = shirei.Vec2{}
	shirei.GetFrameInput().Key = key
	shirei.GetFrameInput().Text = ""
	if key == 0 {
		shirei.GetInputState().DownKeys = nil
	} else {
		shirei.GetInputState().DownKeys = []shirei.KeyCode{key}
	}
	shirei.RunFrameFn(func() {
		shirei.ModAttrs(shirei.NoAnimate)
		shirei.ContainerWithKey(scope, shirei.Attrs(shirei.Viewport), fn)
	})
}

func newAddApp(t *testing.T) *app.App {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	addFontsOnce.Do(shirei.InitFontSubsystem)
	shirei.ResetInputSession()
	a := app.New(config.Defaults(), "", "")
	a.Prefs.LibraryFolder = t.TempDir()
	return a
}

// TestLibraryAddCopiesChosenScore drives the in-app picker with the keyboard:
// two Downs select the only file after "..", Enter accepts it, and the copy
// lands in the library while the library reopens.
func TestLibraryAddCopiesChosenScore(t *testing.T) {
	a := newAddApp(t)

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "song.mid")
	if err := os.WriteFile(src, []byte("MThd"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.UI.LibAddOpen = true
	a.UI.LibAddStart = srcDir

	scope := new(int)
	view := func() { LibraryAddWindow(a) }

	addFrame(scope, 0, view)
	addFrame(scope, shirei.KeyDown, view)
	addFrame(scope, shirei.KeyDown, view)
	addFrame(scope, shirei.KeyEnter, view)
	addFrame(scope, 0, view)

	if a.UI.LibAddOpen {
		t.Fatal("picker should close after accepting a file")
	}
	if !a.UI.LibraryOpen {
		t.Fatal("library should reopen after accepting a file")
	}
	data, err := os.ReadFile(filepath.Join(a.Prefs.LibraryFolder, "song.mid"))
	if err != nil {
		t.Fatalf("copied file: %v", err)
	}
	if string(data) != "MThd" {
		t.Fatalf("copied content = %q", data)
	}
}

// TestLibraryAddEscapeReturnsToLibrary checks that cancelling the picker puts
// the user back in the library instead of leaving the main window.
func TestLibraryAddEscapeReturnsToLibrary(t *testing.T) {
	a := newAddApp(t)
	a.UI.LibAddOpen = true
	a.UI.LibAddStart = t.TempDir()

	scope := new(int)
	view := func() { LibraryAddWindow(a) }

	addFrame(scope, 0, view)
	// The filter field starts focused: the first Escape blurs it, the second
	// reaches the modal and dismisses.
	addFrame(scope, shirei.KeyEscape, view)
	addFrame(scope, 0, view)
	addFrame(scope, shirei.KeyEscape, view)
	addFrame(scope, 0, view)

	if a.UI.LibAddOpen {
		t.Fatal("Escape should close the picker")
	}
	if !a.UI.LibraryOpen {
		t.Fatal("library should reopen after cancelling the picker")
	}
}
