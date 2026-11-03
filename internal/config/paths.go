package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DataDir is the per-user directory for preferences, soundfonts, exports and
// other mutated state. A directory named "gmid-data" next to the executable (or
// in the working directory) switches the app into portable mode.
func DataDir() string {
	if dir := portableDir(); dir != "" {
		return dir
	}
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "gmid")
}

func portableDir() string {
	candidates := []string{filepath.Join(".", "gmid-data")}
	if exe, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(exe), "gmid-data")}, candidates...)
	}
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir
		}
	}
	return ""
}

// AssetsDir finds the bundled assets (fonts, demo songs, soundfonts). It checks
// GMID_ASSETS, the working directory, the executable's directory and their
// parents, so `go run`, a Linux install and a macOS `.app` bundle all work. In
// a bundle the assets live in Contents/Resources/assets, hence the extra
// Resources probe at each level.
func AssetsDir() string {
	if dir := os.Getenv("GMID_ASSETS"); dir != "" {
		if isDir(dir) {
			return dir
		}
	}
	starts := []string{"."}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	for _, start := range starts {
		dir, err := filepath.Abs(start)
		if err != nil {
			continue
		}
		for i := 0; i < 8; i++ {
			for _, candidate := range []string{
				filepath.Join(dir, "assets"),
				filepath.Join(dir, "Resources", "assets"),
			} {
				if isDir(candidate) {
					return candidate
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return "assets"
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

// FontPaths returns the bundled font files that exist.
func FontPaths() []string {
	dir := filepath.Join(AssetsDir(), "fonts")
	names := []string{"Inter-Regular.ttf", "Inter-Bold.ttf", "JetBrainsMono-Regular.ttf"}
	var out []string
	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			out = append(out, path)
		}
	}
	return out
}

// SoundfontsDir is where user and bundled soundfonts live.
func SoundfontsDir() string {
	dir := filepath.Join(DataDir(), "soundfonts")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// GmidDir is the versioned-save/import/cache folder inside a library folder.
func GmidDir(library string) string {
	if library == "" {
		library = filepath.Join(DataDir(), "library")
	}
	return filepath.Join(library, "gmid")
}

// ExportsDir is where rendered audio and chord charts are written.
func ExportsDir(library string) string {
	dir := filepath.Join(GmidDir(library), "exports")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// DownloadsDir returns the user's Downloads folder when it exists. It honours
// XDG_DOWNLOAD_DIR on Linux and otherwise falls back to ~/Downloads, which is
// easier to find than the library's nested exports folder.
func DownloadsDir() string {
	home, _ := os.UserHomeDir()
	if dir := xdgDownloadDir(home); dir != "" && isDir(dir) {
		return dir
	}
	if home != "" {
		if dir := filepath.Join(home, "Downloads"); isDir(dir) {
			return dir
		}
	}
	return ""
}

func xdgDownloadDir(home string) string {
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		if home == "" {
			return ""
		}
		cfg = filepath.Join(home, ".config")
	}
	data, err := os.ReadFile(filepath.Join(cfg, "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "XDG_DOWNLOAD_DIR=") {
			continue
		}
		dir := strings.TrimPrefix(line, "XDG_DOWNLOAD_DIR=")
		dir = strings.Trim(dir, `"`)
		dir = strings.ReplaceAll(dir, "${HOME}", home)
		dir = strings.ReplaceAll(dir, "$HOME", home)
		return dir
	}
	return ""
}

// DefaultLibrary is the library folder used before the user picks one. The
// bundled demo songs are copied there once.
func DefaultLibrary() string {
	dir := filepath.Join(DataDir(), "library")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

// IsWindows reports whether the build targets Windows, where exclusive audio
// and endpoint volume have extra meaning.
func IsWindows() bool { return runtime.GOOS == "windows" }
