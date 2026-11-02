package sampler

import (
	"math"
	"strconv"

	"github.com/akhenakh/gmid/internal/dspreset"
)

// fxChain is an instrument-wide post effect chain. Only global effects are
// supported; group effects are created per voice and are not implemented.
type fxChain struct {
	effects []fxEffect
}

type fxEffect interface {
	process(l, r []float32)
	reset()
}

func newFXChain(specs []dspreset.Effect, sr int) *fxChain {
	if len(specs) == 0 {
		return nil
	}
	chain := &fxChain{}
	for _, e := range specs {
		if !attrOn(e.Params["enabled"]) {
			continue
		}
		switch e.Type {
		case "gain":
			if e.Params["levelUnit"] == "linear" {
				chain.effects = append(chain.effects, &gainFX{g: float32(attrNum(e.Params["level"], 0.5))})
			} else {
				db := attrNum(e.Params["level"], -6)
				chain.effects = append(chain.effects, &gainFX{g: float32(math.Pow(10, db/20))})
			}
		case "lowpass", "lowpass_4pl", "lowpass_1pl":
			chain.effects = append(chain.effects, newBiquadFX(lowpass, e.Params, sr))
		case "highpass":
			chain.effects = append(chain.effects, newBiquadFX(highpass, e.Params, sr))
		case "bandpass":
			chain.effects = append(chain.effects, newBiquadFX(bandpass, e.Params, sr))
		case "reverb":
			chain.effects = append(chain.effects, newReverbFX(e.Params, sr))
		}
	}
	if len(chain.effects) == 0 {
		return nil
	}
	return chain
}

func (c *fxChain) process(l, r []float32) {
	if c == nil {
		return
	}
	for _, e := range c.effects {
		e.process(l, r)
	}
}

func (c *fxChain) reset() {
	if c == nil {
		return
	}
	for _, e := range c.effects {
		e.reset()
	}
}

func attrOn(v string) bool {
	switch v {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

func attrNum(v string, def float64) float64 {
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

type filterKind int

const (
	lowpass filterKind = iota
	highpass
	bandpass
)

// gainFX applies a linear gain.
type gainFX struct{ g float32 }

func (g *gainFX) process(l, r []float32) {
	for i := range l {
		l[i] *= g.g
		r[i] *= g.g
	}
}
func (g *gainFX) reset() {}

// biquadFX is a stereo 2-pole RBJ filter in transposed direct form II.
type biquadFX struct {
	b0, b1, b2, a1, a2 float64
	z1L, z2L, z1R, z2R float64
	onePole            bool
	alpha, beta        float64
	yL, yR             float64
}

func newBiquadFX(kind filterKind, p map[string]string, sr int) *biquadFX {
	if p["type"] == "lowpass_1pl" {
		f := attrNum(p["frequency"], 22000)
		if f <= 0 || f > float64(sr)/2 {
			f = float64(sr) / 2
		}
		a := 1 - math.Exp(-2*math.Pi*f/float64(sr))
		return &biquadFX{onePole: true, alpha: a, beta: 1 - a}
	}
	freq := attrNum(p["frequency"], 22000)
	if freq < 20 {
		freq = 20
	}
	if freq > float64(sr)/2-1 {
		freq = float64(sr)/2 - 1
	}
	q := attrNum(p["resonance"], 0.7)
	if q < 0.001 {
		q = 0.001
	}
	if q > 5 {
		q = 5
	}
	w0 := 2 * math.Pi * freq / float64(sr)
	cs, sn := math.Cos(w0), math.Sin(w0)
	alpha := sn / (2 * q)
	var b0, b1, b2 float64
	switch kind {
	case highpass:
		b0 = (1 + cs) / 2
		b1 = -(1 + cs)
		b2 = b0
	case bandpass:
		b0 = alpha
		b1 = 0
		b2 = -alpha
	default:
		b0 = (1 - cs) / 2
		b1 = 1 - cs
		b2 = b0
	}
	a0 := 1 + alpha
	return &biquadFX{
		b0: b0 / a0, b1: b1 / a0, b2: b2 / a0,
		a1: -2 * cs / a0, a2: (1 - alpha) / a0,
	}
}

func (b *biquadFX) process(l, r []float32) {
	if b.onePole {
		for i := range l {
			b.yL = b.alpha*float64(l[i]) + b.beta*b.yL
			b.yR = b.alpha*float64(r[i]) + b.beta*b.yR
			l[i], r[i] = float32(b.yL), float32(b.yR)
		}
		return
	}
	for i := range l {
		xl, xr := float64(l[i]), float64(r[i])
		yl := b.b0*xl + b.z1L
		b.z1L = b.b1*xl - b.a1*yl + b.z2L
		b.z2L = b.b2*xl - b.a2*yl

		yr := b.b0*xr + b.z1R
		b.z1R = b.b1*xr - b.a1*yr + b.z2R
		b.z2R = b.b2*xr - b.a2*yr

		l[i], r[i] = float32(yl), float32(yr)
	}
}

func (b *biquadFX) reset() {
	b.z1L, b.z2L, b.z1R, b.z2R, b.yL, b.yR = 0, 0, 0, 0, 0, 0
}

// reverbFX is a stereo Freeverb-style reverb.
type reverbFX struct {
	combsL, combsR []*comb
	apL, apR       []*allpass
	wet            float32
}

type comb struct {
	buf   []float32
	idx   int
	store float32
	fb    float32
	damp1 float32
	damp2 float32
}

func (c *comb) process(x float32) float32 {
	out := c.buf[c.idx]
	c.store = out*c.damp2 + c.store*c.damp1
	c.buf[c.idx] = x + c.store*c.fb
	c.idx++
	if c.idx >= len(c.buf) {
		c.idx = 0
	}
	return out
}

type allpass struct {
	buf []float32
	idx int
}

func (a *allpass) process(x float32) float32 {
	bo := a.buf[a.idx]
	out := -x + bo
	a.buf[a.idx] = x + bo*0.5
	a.idx++
	if a.idx >= len(a.buf) {
		a.idx = 0
	}
	return out
}

func newReverbFX(p map[string]string, sr int) *reverbFX {
	scale := float64(sr) / 44100
	room := attrNum(p["roomSize"], 0.7)
	damp := attrNum(p["damping"], 0.3)
	wet := attrNum(p["wetLevel"], 0)
	fb := float32(room*0.28 + 0.7)
	d1 := float32(damp * 0.4)
	d2 := 1 - d1
	combTune := []float64{1116, 1188, 1277, 1356, 1422, 1491, 1557, 1617}
	apTune := []float64{556, 441, 341, 225}
	r := &reverbFX{wet: float32(wet)}
	for _, t := range combTune {
		r.combsL = append(r.combsL, &comb{buf: make([]float32, int(t*scale)+1), fb: fb, damp1: d1, damp2: d2})
		r.combsR = append(r.combsR, &comb{buf: make([]float32, int((t+23)*scale)+1), fb: fb, damp1: d1, damp2: d2})
	}
	for _, t := range apTune {
		r.apL = append(r.apL, &allpass{buf: make([]float32, int(t*scale)+1)})
		r.apR = append(r.apR, &allpass{buf: make([]float32, int((t+23)*scale)+1)})
	}
	return r
}

func (rv *reverbFX) process(l, r []float32) {
	if rv.wet <= 0 {
		return
	}
	for i := range l {
		x := (l[i] + r[i]) * 0.015
		var wl, wr float32
		for _, c := range rv.combsL {
			wl += c.process(x)
		}
		for _, c := range rv.combsR {
			wr += c.process(x)
		}
		wl /= float32(len(rv.combsL))
		wr /= float32(len(rv.combsR))
		for _, a := range rv.apL {
			wl = a.process(wl)
		}
		for _, a := range rv.apR {
			wr = a.process(wr)
		}
		l[i] += wl * rv.wet
		r[i] += wr * rv.wet
	}
}

func (rv *reverbFX) reset() {
	for _, c := range rv.combsL {
		c.reset()
	}
	for _, c := range rv.combsR {
		c.reset()
	}
	for _, a := range rv.apL {
		a.reset()
	}
	for _, a := range rv.apR {
		a.reset()
	}
}

func (c *comb) reset() {
	for i := range c.buf {
		c.buf[i] = 0
	}
	c.idx, c.store = 0, 0
}

func (a *allpass) reset() {
	for i := range a.buf {
		a.buf[i] = 0
	}
	a.idx = 0
}
