package audio

import (
	"math"
)

// renderLocked fills the stereo block. The engine mutex is held by the caller.
func (e *Engine) renderLocked(l, r []float32) {
	if !e.playing || e.timeline == nil {
		e.peak *= 0.85
		return
	}
	if e.countInActive {
		e.renderCountInLocked(l, r)
		return
	}
	frames := len(l)
	off := 0
	for off < frames {
		remaining := frames - off
		dt := float64(remaining) / float64(e.sr) * e.speed
		end := e.pos + dt
		if e.loopOn && e.loopEnd > e.loopStart && e.pos < e.loopEnd && end > e.loopEnd {
			n := int(math.Round((e.loopEnd - e.pos) / e.speed * float64(e.sr)))
			if n <= 0 {
				n = 1
			}
			if n > remaining {
				n = remaining
			}
			e.renderSpanLocked(l[off:off+n], r[off:off+n], e.pos, e.loopEnd)
			off += n
			e.pos = e.loopEnd
			e.resetLocked(e.loopStart)
			e.pos = e.loopStart
			continue
		}
		e.renderSpanLocked(l[off:], r[off:], e.pos, end)
		e.pos = end
		off = frames
	}
	peak := 0.0
	for i := range l {
		a := math.Abs(float64(l[i]))
		b := math.Abs(float64(r[i]))
		if a > peak {
			peak = a
		}
		if b > peak {
			peak = b
		}
	}
	if peak > e.peak {
		e.peak = peak
	} else {
		e.peak = e.peak*0.9 + peak*0.1
	}
}

// renderSpanLocked renders [from,to) with events dispatched at their exact
// sample offsets.
func (e *Engine) renderSpanLocked(outL, outR []float32, from, to float64) {
	pos := from
	off := 0
	n := len(outL)

	renderTo := func(t float64) bool {
		if t <= pos || off >= n {
			return true
		}
		frames := int(math.Round((t - pos) / e.speed * float64(e.sr)))
		if frames <= 0 {
			return true
		}
		if off+frames > n {
			frames = n - off
		}
		if frames <= 0 {
			return false
		}
		if e.synth != nil {
			e.synth.Render(e.tmpL[:frames], e.tmpR[:frames])
			gain := float32(e.masterVol)
			for i := 0; i < frames; i++ {
				outL[off+i] += e.tmpL[i] * gain
				outR[off+i] += e.tmpR[i] * gain
			}
		}
		// Metronome clicks are on their own bus and ignore the master fader.
		if e.metronomeOn {
			for e.beatIdx < len(e.beats) && e.beats[e.beatIdx] < t {
				bt := e.beats[e.beatIdx]
				if bt >= pos {
					boff := int(math.Round((bt - pos) / e.speed * float64(e.sr)))
					if boff >= 0 && boff < frames {
						e.addClick(outL[off:off+frames], outR[off:off+frames], boff, e.isBar(bt))
					}
				}
				e.beatIdx++
			}
		} else {
			for e.beatIdx < len(e.beats) && e.beats[e.beatIdx] < t {
				e.beatIdx++
			}
		}
		off += frames
		pos = t
		return true
	}

	for e.idx < len(e.timeline.Events) {
		ev := e.timeline.Events[e.idx]
		if ev.Sec >= to {
			break
		}
		if !renderTo(ev.Sec) {
			return
		}
		if ev.Sec > pos {
			pos = ev.Sec
		}
		e.dispatchLocked(ev)
		e.idx++
	}
	renderTo(to)
}

func (e *Engine) isBar(t float64) bool {
	for _, b := range e.bars {
		if math.Abs(b-t) < 1e-6 {
			return true
		}
		if b > t {
			break
		}
	}
	return false
}

// addClick adds a short metronome click at a sample offset.
func (e *Engine) addClick(l, r []float32, offset int, accent bool) {
	length := int(float64(e.sr) * 0.018)
	if length < 1 {
		return
	}
	amp := float32(0.20 * e.metronomeVol)
	freq := 1760.0
	if accent {
		amp *= 1.35
		freq = 2093.0
	}
	decay := float64(e.sr) * 0.004
	for i := 0; i < length; i++ {
		idx := offset + i
		if idx < 0 || idx >= len(l) {
			continue
		}
		env := math.Exp(-float64(i) / decay)
		v := float32(math.Sin(2*math.Pi*freq*float64(i)/float64(e.sr))) * amp * float32(env)
		l[idx] += v
		r[idx] += v
	}
}
