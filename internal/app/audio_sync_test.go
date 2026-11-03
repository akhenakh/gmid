package app

import (
	"math"
	"testing"
	"time"
)

// TestAdvanceClockSmoothsAudioBlocks models a 120 Hz display reading a
// 512-frame/48 kHz audio clock. Every frame must move, even though the audio
// position changes only every ~10.7 ms.
func TestAdvanceClockSmoothsAudioBlocks(t *testing.T) {
	a := &App{}
	a.Transport.Speed = 1

	const frame = time.Second / 120 // 8.33 ms
	const block = 512.0 / 48000     // 10.67 ms

	start := time.Unix(0, 0)
	snapAt := func(elapsed time.Duration) float64 {
		return 10 + math.Floor(elapsed.Seconds()/block)*block
	}

	prev := a.advanceClock(snapAt(0), start)
	for i := 1; i <= 240; i++ {
		now := start.Add(time.Duration(i) * frame)
		got := a.advanceClock(snapAt(now.Sub(start)), now)
		dt := got - prev
		if dt <= 0 {
			t.Fatalf("frame %d: clock did not advance (dt=%g)", i, dt)
		}
		if lo, hi := 0.3*frame.Seconds(), 1.7*frame.Seconds(); dt < lo || dt > hi {
			t.Fatalf("frame %d: dt=%g, want within [%g,%g]", i, dt, lo, hi)
		}
		prev = got
	}

	if err := math.Abs(prev - snapAt(2*time.Second)); err > 0.03 {
		t.Fatalf("clock drifted %.3fs from the audio position", err)
	}
}

func TestAdvanceClockSnapsAfterSeek(t *testing.T) {
	a := &App{}
	a.Transport.Speed = 1
	start := time.Unix(0, 0)

	a.advanceClock(10, start)
	got := a.advanceClock(25, start.Add(time.Second/120))
	if got != 25 {
		t.Fatalf("seek clock = %g, want 25", got)
	}
}

func TestAdvanceClockRespectsSpeed(t *testing.T) {
	a := &App{}
	a.Transport.Speed = 0.5
	start := time.Unix(0, 0)

	a.advanceClock(10, start)
	got := a.advanceClock(10.005, start.Add(10*time.Millisecond))
	// Half speed: 10 ms of wall time is 5 ms of song time, matching the audio.
	if math.Abs(got-10.005) > 1e-9 {
		t.Fatalf("clock = %g, want 10.005", got)
	}
}
