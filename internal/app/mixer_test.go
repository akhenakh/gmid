package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

func TestChannelMetersFollowNotesAndMutes(t *testing.T) {
	a := &App{
		Song: &song.Song{Tracks: []song.Track{
			{Channel: 1, Visible: true, Notes: []song.Note{{StartSec: 0, EndSec: 4, Key: 60, Velocity: 127}}},
			{Channel: 2, Visible: true, Notes: []song.Note{{StartSec: 0, EndSec: 4, Key: 64, Velocity: 64}}},
		}},
		Transport: Transport{Playing: true, Time: 1},
	}
	levels := a.UpdateChannelMeters()
	if levels[1] <= levels[2] {
		t.Fatalf("loud channel %v should exceed quiet channel %v", levels[1], levels[2])
	}
	if levels[1] == 0 || levels[2] == 0 {
		t.Fatalf("active channels have no level: %v", levels)
	}

	a.Song.Tracks[0].Muted = true
	for i := 0; i < 30; i++ {
		levels = a.UpdateChannelMeters()
	}
	if levels[1] > 0.05 {
		t.Fatalf("muted channel still at %v", levels[1])
	}
	if levels[2] == 0 {
		t.Fatalf("unmuted channel lost its level: %v", levels[2])
	}

	a.LiveKeys = map[int]bool{60: true}
	levels = a.UpdateChannelMeters()
	if levels[15] == 0 {
		t.Fatalf("live key did not light channel 15: %v", levels)
	}
}

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
