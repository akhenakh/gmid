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

func TestDownloadsDirHonoursXDG(t *testing.T) {
	home := t.TempDir()
	downloads := filepath.Join(home, "Téléchargements")
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".config")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	entry := `XDG_DOWNLOAD_DIR="$HOME/Téléchargements"` + "\n"
	if err := os.WriteFile(filepath.Join(cfg, "user-dirs.dirs"), []byte(entry), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", cfg)

	if got := DownloadsDir(); got != downloads {
		t.Fatalf("DownloadsDir() = %q, want %q", got, downloads)
	}
}

func TestDownloadsDirFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	downloads := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(downloads, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	if got := DownloadsDir(); got != downloads {
		t.Fatalf("DownloadsDir() = %q, want %q", got, downloads)
	}
}

func TestDownloadsDirMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	if got := DownloadsDir(); got != "" {
		t.Fatalf("DownloadsDir() = %q, want empty", got)
	}
}
