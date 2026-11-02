package audio

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/akhenakh/gmid/internal/smf"
)

const (
	testSong      = "../../assets/demo/Scott Joplin - Entertainer.mid"
	testSoundfont = "../../assets/soundfonts/GeneralUser-GS.sf2"
)

func TestCompileTimeline(t *testing.T) {
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	tl := Compile(s)
	if len(tl.Events) == 0 {
		t.Fatal("empty timeline")
	}
	on, off := 0, 0
	lastSec := -1.0
	for i, ev := range tl.Events {
		if ev.Sec < lastSec {
			t.Fatalf("events out of order at %d", i)
		}
		lastSec = ev.Sec
		switch ev.Kind {
		case EvNoteOn:
			on++
		case EvNoteOff:
			off++
		}
	}
	if on != s.NoteCount {
		t.Fatalf("note-ons %d, song notes %d", on, s.NoteCount)
	}
	if off < on {
		t.Fatalf("note-offs %d < note-ons %d", off, on)
	}
}

func TestRenderAudible(t *testing.T) {
	if _, err := os.Stat(testSoundfont); err != nil {
		t.Skip("soundfont missing")
	}
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	e := New(44100)
	if err := e.LoadSoundFont(testSoundfont); err != nil {
		t.Fatalf("soundfont: %v", err)
	}
	e.Load(s)
	e.SetMasterVolume(0.8)
	e.Play(s.OriginSec())

	var peak, sum float64
	var n int
	for i := 0; i < 4*44100/512; i++ {
		l, r := e.RenderOffline(512)
		for j := range l {
			v := math.Abs(float64(l[j])) + math.Abs(float64(r[j]))
			if v > peak {
				peak = v
			}
			sum += v
			n++
		}
	}
	if peak < 1e-4 {
		t.Fatalf("silent render (peak %g)", peak)
	}
	if peak > 4 {
		t.Fatalf("render clips hard (peak %g)", peak)
	}
	t.Logf("peak=%g mean=%g", peak, sum/float64(n))
}

func TestOutputDevice(t *testing.T) {
	if os.Getenv("GMID_AUDIO_TEST") == "" {
		t.Skip("set GMID_AUDIO_TEST=1 to open the real device")
	}
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	e := New(48000)
	if err := e.LoadSoundFont(testSoundfont); err != nil {
		t.Fatalf("soundfont: %v", err)
	}
	if err := e.StartOutput("", false, 256); err != nil {
		t.Fatalf("device: %v", err)
	}
	defer e.Close()
	e.Load(s)
	e.Play(s.OriginSec())
	time.Sleep(400 * time.Millisecond)
	snap := e.Snapshot()
	if !snap.Playing || snap.Time <= s.OriginSec() {
		t.Fatalf("clock did not advance: %+v", snap)
	}
	t.Logf("device ok, time=%.2fs peak=%.3f", snap.Time, snap.Peak)
}

func TestCountInDelaysSong(t *testing.T) {
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	e := New(44100)
	e.Load(s)
	e.SetCountIn(1)
	e.Play(s.OriginSec())
	if !e.Snapshot().CountIn {
		t.Fatal("count-in did not arm")
	}
	for i := 0; i < 60; i++ {
		e.RenderOffline(4410)
		if !e.Snapshot().CountIn {
			break
		}
	}
	snap := e.Snapshot()
	if snap.CountIn {
		t.Fatal("count-in never finished")
	}
	if snap.Time < s.FirstNoteSec-0.01 {
		t.Fatalf("song started before first note: %v (first %v)", snap.Time, s.FirstNoteSec)
	}
}

func TestLoopWraps(t *testing.T) {
	s, err := smf.ReadFile(testSong)
	if err != nil {
		t.Skipf("demo missing: %v", err)
	}
	e := New(44100)
	e.Load(s)
	e.SetLoop(true, 2, 3)
	e.Play(2)
	for i := 0; i < 100; i++ {
		e.RenderOffline(512)
	}
	snap := e.Snapshot()
	if snap.Time < 2 || snap.Time >= 3.01 {
		t.Fatalf("loop position out of range: %v", snap.Time)
	}
}
