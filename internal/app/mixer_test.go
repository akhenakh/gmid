package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/smf"
)

func TestMixerBakeSavesVersion(t *testing.T) {
	t.Setenv("GMID_CONFIG", filepath.Join(t.TempDir(), "prefs.json"))
	demo := "../../assets/demo/Scott Joplin - Entertainer.mid"
	if _, err := os.Stat(demo); err != nil {
		t.Skip("demo missing")
	}
	lib := t.TempDir()
	prefs := config.Defaults()
	prefs.LibraryFolder = lib
	a := New(prefs, "../../assets", "")
	s, err := smf.ReadFile(demo)
	if err != nil {
		t.Fatal(err)
	}
	a.InstallSong(s)
	ch := 0
	if len(s.Channels) > 0 {
		ch = s.Channels[0].Number
	}
	a.SetMix(ch, audio.ChannelMix{Volume: 0.5, Pan: 0.3, Chorus: 0.2, Reverb: 0.6})
	a.BakeMixAndSave()
	files, _ := filepath.Glob(filepath.Join(lib, "gmid", "*_GMID_v*.mid"))
	if len(files) != 1 {
		t.Fatalf("saved files: %v", files)
	}
	back, err := smf.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(back.Meta.History) == 0 {
		t.Fatal("history not embedded")
	}
	if c := back.ChannelAt(ch); c == nil || c.InitialVolume == 100 {
		t.Fatalf("mix not baked: %+v", c)
	}
}
