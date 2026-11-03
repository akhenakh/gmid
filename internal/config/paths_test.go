package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAssetsDirFindsResourcesBundle guards the macOS .app layout: the executable
// lives in Contents/MacOS and the assets in Contents/Resources/assets, so the
// upward walk has to probe Resources as well as a sibling assets directory.
func TestAssetsDirFindsResourcesBundle(t *testing.T) {
	root := t.TempDir()
	want := filepath.Join(root, "Resources", "assets")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	t.Setenv("GMID_ASSETS", "")

	if got := AssetsDir(); got != want {
		t.Fatalf("AssetsDir() = %q, want %q", got, want)
	}
}

func TestAssetsDirPrefersEnvOverride(t *testing.T) {
	override := t.TempDir()
	t.Setenv("GMID_ASSETS", override)

	if got := AssetsDir(); got != override {
		t.Fatalf("AssetsDir() = %q, want %q", got, override)
	}
}
