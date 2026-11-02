package audio

import (
	"math"
)

// renderLocked fills the stereo block. The engine mutex is held by the caller.
func (e *Engine) renderLocked(l, r []float32) {
	if !e.playing || e.timeline == nil {
		e.renderLiveLocked(l, r)
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
	if e.compressor || e.masterTone != 0.5 {
		e.processMasterLocked(l, r)
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

// renderLiveLocked renders held live notes while the transport is stopped or
// paused. The song timeline is not scheduled, so only voices started by the
// keyboard (or MIDI input) sound.
func (e *Engine) renderLiveLocked(l, r []float32) {
	n := len(l)
	if len(e.tmpL) < n {
		e.tmpL = make([]float32, n)
		e.tmpR = make([]float32, n)
	}
	if len(e.smpL) < n {
		e.smpL = make([]float32, n)
		e.smpR = make([]float32, n)
	}
	tl := e.tmpL[:n]
	tr := e.tmpR[:n]
	for i := range tl {
		tl[i], tr[i] = 0, 0
	}
	if e.synth != nil {
		e.synth.Render(tl, tr)
	}
	if e.decent != nil {
		sl := e.smpL[:n]
		sr := e.smpR[:n]
		for i := range sl {
			sl[i], sr[i] = 0, 0
		}
		e.decent.Render(sl, sr)
		for i := range tl {
			tl[i] += sl[i]
			tr[i] += sr[i]
		}
	}
	gain := float32(e.masterVol)
	for i := range l {
		l[i] = tl[i] * gain
		r[i] = tr[i] * gain
	}
	if e.compressor || e.masterTone != 0.5 {
		e.processMasterLocked(l, r)
	}
	peak := 0.0
	for i := range l {
		if a := math.Abs(float64(l[i])); a > peak {
			peak = a
		}
		if a := math.Abs(float64(r[i])); a > peak {
			peak = a
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
		if e.decent != nil {
			sl := e.smpL[:frames]
			sr := e.smpR[:frames]
			for i := range sl {
				sl[i], sr[i] = 0, 0
			}
			e.decent.Render(sl, sr)
			gain := float32(e.masterVol)
			for i := 0; i < frames; i++ {
				outL[off+i] += sl[i] * gain
				outR[off+i] += sr[i] * gain
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

func (e *Engine) processMasterLocked(l, r []float32) {
	for i := range l {
		xL, xR := l[i], r[i]
		if e.masterTone != 0.5 {
			a := float32(0.05 + e.masterTone*0.9)
			e.toneStateL += a * (xL - e.toneStateL)
			e.toneStateR += a * (xR - e.toneStateR)
			if e.masterTone < 0.5 {
				xL, xR = e.toneStateL, e.toneStateR
			} else {
				amount := float32((e.masterTone - 0.5) * 2)
				xL += (xL - e.toneStateL) * amount
				xR += (xR - e.toneStateR) * amount
			}
		}
		if e.compressor {
			a := float32(math.Abs(float64(xL)))
			if b := float32(math.Abs(float64(xR))); b > a {
				a = b
			}
			if a > e.compEnv {
				e.compEnv += (a - e.compEnv) * 0.4
			} else {
				e.compEnv += (a - e.compEnv) * 0.002
			}
			if e.compEnv > 0.7 {
				gain := 0.7 + (e.compEnv-0.7)/4
				gain /= e.compEnv
				xL *= gain
				xR *= gain
			}
		}
		l[i], r[i] = xL, xR
	}
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
