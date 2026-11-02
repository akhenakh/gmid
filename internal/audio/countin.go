package audio

import "math"

type countClick struct {
	rel    float64
	accent bool
}

// prepareCountInLocked builds the click plan for a start at `from`.
func (e *Engine) prepareCountInLocked(from float64) {
	e.countInActive = false
	e.countInClicks = e.countInClicks[:0]
	if e.countInBars <= 0 || e.song == nil || e.timeline == nil {
		return
	}
	first := e.song.FirstNoteSec
	if first < e.timeline.Origin {
		first = e.timeline.Origin
	}
	if from > first+0.05 {
		return
	}
	d := first
	for _, b := range e.bars {
		if b >= first-1e-6 {
			d = b
			break
		}
	}
	num, _ := e.song.SigAt(d)
	if num <= 0 {
		num = 4
	}
	period := e.beatPeriodLocked(d)
	if period <= 0 {
		period = 60.0 / math.Max(20, e.song.BPMAt(d))
	}
	count := e.countInBars * num
	start := d - float64(count)*period
	e.countInTotal = (d - start) / math.Max(e.speed, 0.01)
	e.countInElapsed = 0
	e.countInSong = first
	for k := count; k >= 1; k-- {
		t := d - float64(k)*period
		if t >= first-1e-6 {
			continue
		}
		n := ((count - k) % num) + 1
		e.countInClicks = append(e.countInClicks, countClick{
			rel:    (t - start) / math.Max(e.speed, 0.01),
			accent: n == 1,
		})
	}
	e.countInActive = true
	e.pos = from
}

func (e *Engine) beatPeriodLocked(at float64) float64 {
	i := firstAtOrAfter(e.beats, at)
	if i >= 0 && i+1 < len(e.beats) {
		return e.beats[i+1] - e.beats[i]
	}
	if i > 0 && i < len(e.beats) {
		return e.beats[i] - e.beats[i-1]
	}
	if i == len(e.beats) && i > 0 {
		return e.beats[i-1] - e.beats[i-2]
	}
	return 0
}

func (e *Engine) renderCountInLocked(l, r []float32) {
	frames := len(l)
	dt := float64(frames) / float64(e.sr)
	start := e.countInElapsed
	end := start + dt
	for _, c := range e.countInClicks {
		if c.rel >= start && c.rel < end {
			off := 0
			if dt > 0 {
				off = int((c.rel - start) / dt * float64(frames))
			}
			e.addClick(l, r, off, c.accent)
		}
	}
	e.countInElapsed = end
	peak := 0.0
	for i := range l {
		v := math.Abs(float64(l[i]))
		if v > peak {
			peak = v
		}
	}
	e.peak = peak
	if e.countInElapsed >= e.countInTotal {
		e.countInActive = false
		e.pos = e.countInSong
		e.resetLocked(e.pos)
	}
}
