package ui

import (
	"fmt"
	"testing"

	. "go.hasen.dev/shirei"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

// BenchmarkRoll2k measures one software-rendered roll frame for a 2000-note
// song. The M9 budget is 16.7 ms per frame (60 fps).
func BenchmarkRoll2k(b *testing.B) {
	Install(false)
	a := app.New(config.Prefs{}, "../../assets", "roll")
	a.Song = benchmarkRollSong(b, 2000)
	a.Transport.Time = 30

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		img := RenderToImage(1440, 900, func() { PianoRoll(a) })
		if img == nil {
			b.Fatal("no frame rendered")
		}
	}
}

// BenchmarkRoll2kFrame measures the frame tree/layout cost without the
// software rasterizer, which is the closest headless proxy for the 60 fps
// budget.
func BenchmarkRoll2kFrame(b *testing.B) {
	Install(false)
	a := app.New(config.Prefs{}, "../../assets", "roll")
	a.Song = benchmarkRollSong(b, 2000)
	a.Transport.Time = 30

	RenderToImage(1440, 900, func() { PianoRoll(a) }) // init glyph cache
	host := GetHost()
	host.HeadlessRender = true
	host.WindowSize = Vec2{1440, 900}
	host.WindowScale = 1
	defer func() { host.HeadlessRender = false }()
	for i := 0; i < 3; i++ {
		RunFrameFn(func() { PianoRoll(a) })
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out := RunFrameFn(func() { PianoRoll(a) })
		if out.Surfaces == nil {
			b.Fatal("no frame built")
		}
	}
}

func benchmarkRollSong(tb testing.TB, notes int) *song.Song {
	tb.Helper()
	s := &song.Song{
		Name: "bench",
		PPQ:  480,
		Tempo: []song.TempoChange{
			{Tick: 0, BPM: 120},
		},
		Sigs: []song.TimeSigChange{
			{Tick: 0, Num: 4, Den: 4},
		},
		Tracks: []song.Track{{
			Name:    "Piano",
			Channel: 0,
			Program: 0,
			Visible: true,
			Color:   "#e0a030",
		}},
	}
	for i := 0; i < notes; i++ {
		s.Tracks[0].Notes = append(s.Tracks[0].Notes, song.Note{
			ID:       int64(i + 1),
			Key:      48 + i%25,
			Tick:     int64(i * 60),
			DurTicks: 55,
			Velocity: 90,
		})
	}
	raw, err := smf.Write(s)
	if err != nil {
		tb.Fatalf("write: %v", err)
	}
	back, err := smf.Parse(raw, fmt.Sprintf("bench-%d.mid", notes))
	if err != nil {
		tb.Fatalf("parse: %v", err)
	}
	back.Name = "bench"
	return back
}
