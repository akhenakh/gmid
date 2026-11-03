package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akhenakh/gmid/internal/smf"
)

const (
	demo = "../../assets/demo/Scott Joplin - Entertainer.mid"
	sf2  = "../../assets/soundfonts/GeneralUser-GS.sf2"
)

func TestExportWAVAndMP3(t *testing.T) {
	if _, err := os.Stat(sf2); err != nil {
		t.Skip("soundfont missing")
	}
	s, err := smf.ReadFile(demo)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	dir := os.Getenv("GMID_EXPORT_DIR")
	if dir == "" {
		dir = t.TempDir()
	} else {
		_ = os.MkdirAll(dir, 0o755)
	}
	home := t.TempDir()
	downloads := filepath.Join(home, "Downloads")
	if err := os.Mkdir(downloads, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	for _, format := range []string{"wav", "mp3"} {
		path, err := Run(Options{
			Song: s, Variant: Heard, Format: format, SoundFont: sf2,
			Library: dir, Master: 0.8, Speed: 1,
			LoopOn: true, LoopStart: s.OriginSec(), LoopEnd: s.OriginSec() + 2,
		})
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if filepath.Dir(path) != downloads {
			t.Fatalf("%s path %q, want it in %q", format, path, downloads)
		}
		st, err := os.Stat(path)
		if err != nil {
			t.Fatalf("%s: %v", format, err)
		}
		if st.Size() < 1000 {
			t.Fatalf("%s too small: %d", format, st.Size())
		}
		if filepath.Dir(path) == "" {
			t.Fatal("bad path")
		}
		t.Logf("%s: %s (%d bytes)", format, filepath.Base(path), st.Size())
	}
}
