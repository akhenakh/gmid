package sampler

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"

	"github.com/akhenakh/gmid/internal/dspreset"
)

const maxVoices = 256

// Instrument is a DecentSampler-compatible sample player.
type Instrument struct {
	mu         sync.Mutex
	preset     *dspreset.Preset
	zones      []*zone
	groups     []*groupState
	sr         int
	voices     []*voice
	ch         [16]channelState
	lastNote   int
	errors     []string
	enabled    int
	rrIndex    int
	gain       float64
	gainAdjust float64
	refPeak    float64
	now        float64
	onAt       [128]float64
}

type zone struct {
	group       int
	sample      *dspreset.Sample
	buf         *Buffer
	gain        float64
	pan         float64
	rate        float64
	cutoff      float64
	q           float64
	delayFrames int
	loopXF      int
	loopXFMode  string
	releaseDB   float64
	releaseLin  float64
	onCC        map[int][2]int
}

type groupState struct {
	rr  int
	seq string
	pos int
}

type channelState struct {
	volume     float64
	expression float64
	pan        float64
	sustain    bool
	mixVolume  float64
	cc         [128]int
}

type voice struct {
	z            *zone
	pos          float64
	step         float64
	gain         float64
	panL         float32
	panR         float32
	key          int
	ch           int
	released     bool
	held         bool
	finished     bool
	loop         bool
	loopStart    int
	loopEnd      int
	start        int
	end          int
	age          float64
	release      float64
	releaseLevel float64
	env          float64
	stage        int // 0 delay, 1 attack, 2 hold, 3 decay, 4 sustain, 5 release
	stageT       float64
	delay        int
	xf           int
	xfMode       string
	// state-variable filter
	lowL, bandL, lowR, bandR float32
	f, q                     float32
}

// Load builds an instrument from a parsed preset.
func Load(p *dspreset.Preset, sampleRate int) (*Instrument, error) {
	if p == nil {
		return nil, fmt.Errorf("no preset")
	}
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	inst := &Instrument{preset: p, sr: sampleRate, gain: 1, gainAdjust: 1}
	inst.voices = make([]*voice, 0, 64)
	buffers := map[string]*Buffer{}
	inst.groups = make([]*groupState, len(p.Groups))
	for gi, g := range p.Groups {
		if !g.Enabled {
			continue
		}
		inst.groups[gi] = &groupState{seq: g.SeqMode, pos: g.SeqPosition}
		inst.enabled++
		for _, s := range g.Samples {
			buf := buffers[s.Path]
			if buf == nil {
				b, err := LoadBuffer(s.Path)
				if err != nil {
					inst.errors = append(inst.errors, fmt.Sprintf("%s: %v", s.Path, err))
					continue
				}
				buf = b
				buffers[s.Path] = buf
			}
			z := &zone{
				group:      gi,
				sample:     s,
				buf:        buf,
				gain:       p.Volume * g.Volume * s.Volume,
				pan:        s.Pan,
				rate:       float64(buf.Rate) / float64(sampleRate),
				cutoff:     s.FilterCutoff,
				q:          s.FilterQ,
				loopXF:     s.LoopCrossfade,
				loopXFMode: s.LoopCrossfadeMode,
				releaseDB:  s.ReleaseDecayDB,
				releaseLin: s.ReleaseDecayLinear,
				onCC:       s.OnCCFilters,
			}
			switch s.DelayUnit {
			case "samples":
				z.delayFrames = int(s.Delay)
			case "beats":
				// No host tempo reaches the sampler; assume 120 BPM.
				z.delayFrames = int(s.Delay * 0.5 * float64(sampleRate))
			default:
				z.delayFrames = int(s.Delay * float64(sampleRate))
			}
			if z.delayFrames < 0 {
				z.delayFrames = 0
			}
			if z.cutoff <= 0 || z.cutoff > float64(sampleRate)/2 {
				z.cutoff = 0
			}
			inst.zones = append(inst.zones, z)
		}
	}
	if len(inst.zones) == 0 {
		return nil, fmt.Errorf("no playable samples (%d errors)", len(inst.errors))
	}
	inst.gain, inst.refPeak = makeupGain(buffers)
	if inst.gain != 1 {
		for _, z := range inst.zones {
			z.gain *= inst.gain
		}
	}
	for i := range inst.ch {
		inst.ch[i].volume = 1
		inst.ch[i].expression = 1
		inst.ch[i].pan = 0.5
		inst.ch[i].mixVolume = 1
		for j := range inst.ch[i].cc {
			inst.ch[i].cc[j] = -1
		}
	}
	return inst, nil
}

// Stats describes the loaded instrument.
type Stats struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Groups   int      `json:"groups"`
	Zones    int      `json:"zones"`
	Buffers  int      `json:"buffers"`
	Samples  []string `json:"samples,omitempty"`
	Effects  []string `json:"effects,omitempty"`
	Controls int      `json:"controls"`
	Errors   []string `json:"errors,omitempty"`
	Voices   int      `json:"voices"`
	GainDB   float64  `json:"gainDb"`
	RefPeak  float64  `json:"refPeak"`
}

// Info returns a summary.
func (i *Instrument) Info() Stats {
	i.mu.Lock()
	defer i.mu.Unlock()
	st := Stats{Path: i.preset.Path, Name: i.preset.Name, Groups: i.enabled, Zones: len(i.zones)}
	seen := map[string]bool{}
	for _, z := range i.zones {
		if !seen[z.sample.Path] {
			st.Samples = append(st.Samples, z.sample.Path)
			seen[z.sample.Path] = true
		}
	}
	for _, e := range i.preset.Effects {
		st.Effects = append(st.Effects, e.Type)
	}
	st.Controls = len(i.preset.Controls)
	st.Errors = append(st.Errors, i.errors...)
	st.Voices = len(i.voices)
	st.Buffers = len(st.Samples)
	st.GainDB = 20 * math.Log10(i.gain)
	st.RefPeak = i.refPeak
	return st
}

// Controls exposes parsed instrument UI controls.
func (i *Instrument) Controls() []dspreset.Control { return i.preset.Controls }

// SetGainAdjust applies a user gain in decibels on top of the automatic
// loudness matching. It takes effect on the next rendered block, including
// sustained voices.
func (i *Instrument) SetGainAdjust(db float64) {
	i.mu.Lock()
	i.gainAdjust = math.Pow(10, db/20)
	i.mu.Unlock()
}

// NoteOn triggers matching zones.
func (i *Instrument) NoteOn(key, vel, ch int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.preset == nil {
		return
	}
	i.lastNote = key
	i.onAt[key] = i.now

	busy := false
	for _, v := range i.voices {
		if !v.finished {
			busy = true
			break
		}
	}

	// Group-level round robins: presets like the Steinway Grand split their
	// round robins into separate groups (seqPosition 1..N). Only the group at
	// the current position plays, otherwise every take is layered and the
	// resulting comb filtering makes the instrument sound thin and quiet.
	var rrPositions []int
	seenPos := map[int]bool{}
	rrMode, rrSeqLen := "", 0
	for _, g := range i.preset.Groups {
		if !g.Enabled || !isRRMode(g.SeqMode) {
			continue
		}
		if rrMode == "" {
			rrMode = g.SeqMode
		}
		if !seenPos[g.SeqPosition] {
			seenPos[g.SeqPosition] = true
			rrPositions = append(rrPositions, g.SeqPosition)
		}
		if rrSeqLen == 0 {
			rrSeqLen = g.SeqLength
		}
	}
	sort.Ints(rrPositions)
	rrPos := 0
	if len(rrPositions) > 1 {
		n := len(rrPositions)
		if rrSeqLen > 0 && rrSeqLen < n {
			n = rrSeqLen
		}
		if rrMode == "random" || rrMode == "true_random" {
			rrPos = rrPositions[rand.Intn(n)]
		} else {
			rrPos = rrPositions[i.rrIndex%n]
			i.rrIndex++
		}
	}

	for gi, g := range i.preset.Groups {
		if !g.Enabled {
			continue
		}
		if rrPos > 0 && isRRMode(g.SeqMode) && g.SeqPosition != rrPos {
			continue
		}
		var matches []*zone
		for _, z := range i.zones {
			if z.group != gi {
				continue
			}
			s := z.sample
			if key < s.LoNote || key > s.HiNote || vel < s.LoVel || vel > s.HiVel {
				continue
			}
			switch s.Trigger {
			case "release":
				continue
			case "first":
				if busy {
					continue
				}
			case "legato":
				if !busy {
					continue
				}
			}
			// onLoCC/onHiCC zones are triggered by controller moves, not notes.
			if len(s.OnCCFilters) > 0 {
				continue
			}
			ok := true
			for cc, rng := range s.CCFilters {
				v := i.ch[ch].cc[cc]
				if v < 0 {
					v = 0
				}
				if v < rng[0] || v > rng[1] {
					ok = false
					break
				}
			}
			if ok {
				matches = append(matches, z)
			}
		}
		if len(matches) == 0 {
			continue
		}
		// Sample-level round robins: only choose between zones that actually
		// declare distinct seqPosition values, and cycle through the declared
		// positions in order.
		if i.groups[gi] != nil && isRRMode(g.SeqMode) && hasDistinctSeqPositions(matches) {
			gs := i.groups[gi]
			if g.SeqMode == "random" || g.SeqMode == "true_random" {
				matches = []*zone{matches[rand.Intn(len(matches))]}
			} else {
				positions := uniqueSeqPositions(matches)
				n := len(positions)
				if sl := matches[0].sample.SeqLength; sl > 0 && sl < n {
					n = sl
				}
				pos := positions[gs.rr%n]
				gs.rr++
				filtered := matches[:0]
				for _, z := range matches {
					if z.sample.SeqPosition == pos {
						filtered = append(filtered, z)
					}
				}
				matches = filtered
			}
		}
		for _, z := range matches {
			i.startVoice(z, key, vel, ch, 1)
		}
	}
}

// ccMatches reports whether a controller value falls in the zone's range.
func ccMatches(filters map[int][2]int, cc, val int) bool {
	rng, ok := filters[cc]
	if !ok {
		return false
	}
	return val >= rng[0] && val <= rng[1]
}

func uniqueSeqPositions(zones []*zone) []int {
	seen := map[int]bool{}
	var out []int
	for _, z := range zones {
		if !seen[z.sample.SeqPosition] {
			seen[z.sample.SeqPosition] = true
			out = append(out, z.sample.SeqPosition)
		}
	}
	sort.Ints(out)
	return out
}

func isRRMode(mode string) bool {
	return mode == "round_robin" || mode == "random" || mode == "true_random"
}

// makeupGain returns a whole-instrument gain so libraries mastered at very
// different sample levels play at a comparable loudness. The reference is the
// 90th percentile of sample peaks, so one hot sample cannot hold the rest down.
// The gain is clamped to +3.8 dB of attenuation and +15.6 dB of boost.
func makeupGain(buffers map[string]*Buffer) (gain, ref float64) {
	if len(buffers) == 0 {
		return 1, 0
	}
	peaks := make([]float64, 0, len(buffers))
	for _, b := range buffers {
		peaks = append(peaks, float64(bufferPeak(b)))
	}
	sort.Float64s(peaks)
	ref = peaks[int(float64(len(peaks)-1)*0.9)]
	if ref <= 0.0001 {
		return 1, ref
	}
	gain = 0.55 / ref
	if gain < 0.5 {
		gain = 0.5
	}
	if gain > 6 {
		gain = 6
	}
	return gain, ref
}

func bufferPeak(b *Buffer) float32 {
	var peak float32
	for i := range b.L {
		if v := float32(math.Abs(float64(b.L[i]))); v > peak {
			peak = v
		}
		if v := float32(math.Abs(float64(b.R[i]))); v > peak {
			peak = v
		}
	}
	return peak
}

func hasDistinctSeqPositions(zones []*zone) bool {
	for _, z := range zones[1:] {
		if z.sample.SeqPosition != zones[0].sample.SeqPosition {
			return true
		}
	}
	return false
}

func (i *Instrument) startVoice(z *zone, key, vel, ch int, gainScale float64) {
	s := z.sample
	var semis float64
	if s.TuningIgnoresKeyTrack {
		semis = float64(key-s.Root)*s.PitchKeyTrack + s.Tuning
	} else {
		semis = (float64(key-s.Root) + s.Tuning) * s.PitchKeyTrack
	}
	step := math.Pow(2, semis/12) * z.rate
	gain := z.gain * gainScale
	if s.Amp.Enabled {
		ampTrack := 1.0
		if gi := z.group; gi < len(i.preset.Groups) {
			ampTrack = i.preset.Groups[gi].AmpVelTrack
		}
		gain *= 1 - ampTrack + ampTrack*float64(vel)/127
	}
	pan := (z.pan + 100) / 200
	pan = math.Max(0, math.Min(1, pan))
	theta := pan * math.Pi / 2
	v := &voice{
		z: z, step: step, gain: gain,
		panL: float32(math.Cos(theta)), panR: float32(math.Sin(theta)),
		key: key, ch: ch, start: s.Start, end: s.End,
		loop: s.LoopEnabled, loopStart: s.LoopStart, loopEnd: s.LoopEnd,
		env: 0, delay: z.delayFrames, xfMode: z.loopXFMode,
	}
	if v.end <= 0 || v.end > len(z.buf.L) {
		v.end = len(z.buf.L)
	}
	if v.start < 0 {
		v.start = 0
	}
	if v.loop {
		if v.loopStart <= 0 {
			v.loopStart = z.buf.LoopStart
		}
		if v.loopEnd <= 0 {
			v.loopEnd = z.buf.LoopEnd
		}
		if v.loopEnd <= v.loopStart || v.loopEnd > len(z.buf.L) {
			v.loop = false
		} else {
			v.xf = z.loopXF
			if v.xf > v.loopEnd-v.loopStart {
				v.xf = v.loopEnd - v.loopStart
			}
			if v.xf < 0 {
				v.xf = 0
			}
		}
	}
	v.pos = float64(v.start)
	if s.Amp.Enabled {
		if s.Amp.Delay > 0 {
			v.stage = 0
		} else {
			v.stage = 1
		}
	} else {
		v.env = 1
		v.stage = 4
	}
	if z.cutoff > 0 {
		f := 2 * math.Sin(math.Pi*z.cutoff/float64(i.sr))
		v.f = float32(math.Max(0.001, math.Min(1.5, f)))
		v.q = float32(math.Max(0.5, math.Min(2.0, z.q/10+0.7)))
	}
	i.voices = append(i.voices, v)
	if len(i.voices) > maxVoices {
		i.voices = i.voices[len(i.voices)-maxVoices:]
	}
}

// NoteOff releases matching voices.
func (i *Instrument) NoteOff(key, ch int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for _, v := range i.voices {
		if v.key != key || v.ch != ch || v.released {
			continue
		}
		if i.ch[ch].sustain {
			v.held = true
			continue
		}
		v.released = true
		v.release = v.z.sample.Amp.Release
		v.releaseLevel = v.env
		v.stage = 5
		v.stageT = 0
		if !v.z.sample.LoopDuringRelease {
			v.loop = false
		}
	}
	// Release-trigger samples.
	held := i.now - i.onAt[key]
	if held < 0 {
		held = 0
	}
	for _, z := range i.zones {
		s := z.sample
		if s.Trigger != "release" || key < s.LoNote || key > s.HiNote {
			continue
		}
		scale := 1.0
		switch {
		case z.releaseDB > 0:
			scale = math.Pow(10, -z.releaseDB*held/20)
		case z.releaseLin > 0:
			scale = math.Pow(z.releaseLin, held)
		}
		i.startVoice(z, key, 100, ch, scale)
	}
}

// Control handles MIDI controllers.
func (i *Instrument) Control(ch, cc, val int) {
	if ch < 0 || ch >= 16 {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if cc >= 0 && cc < 128 {
		i.ch[ch].cc[cc] = val
	}
	for _, z := range i.zones {
		if len(z.onCC) == 0 || !ccMatches(z.onCC, cc, val) {
			continue
		}
		i.startVoice(z, z.sample.Root, 100, ch, 1)
	}
	switch cc {
	case 7:
		i.ch[ch].volume = float64(val) / 127
	case 11:
		i.ch[ch].expression = float64(val) / 127
	case 10:
		i.ch[ch].pan = float64(val) / 127
	case 64:
		up := val < 64
		i.ch[ch].sustain = !up
		if up {
			for _, v := range i.voices {
				if v.ch == ch && v.held {
					v.held = false
					v.released = true
					v.release = v.z.sample.Amp.Release
					v.releaseLevel = v.env
					v.stage = 5
					v.stageT = 0
					if !v.z.sample.LoopDuringRelease {
						v.loop = false
					}
				}
			}
		}
	}
}

// SetChannelMix applies the console fader and pan to a channel.
func (i *Instrument) SetChannelMix(ch int, volume, pan float64) {
	if ch < 0 || ch >= 16 {
		return
	}
	i.mu.Lock()
	i.ch[ch].mixVolume = volume
	i.ch[ch].pan = pan
	i.mu.Unlock()
}

// Reset stops all voices.
func (i *Instrument) Reset() {
	i.mu.Lock()
	i.voices = nil
	i.mu.Unlock()
}

// Render adds the instrument output into the stereo buffers.
func (i *Instrument) Render(l, r []float32) {
	i.mu.Lock()
	defer i.mu.Unlock()
	out := i.voices[:0]
	for _, v := range i.voices {
		i.renderVoice(v, l, r)
		if !v.finished {
			out = append(out, v)
		}
	}
	i.voices = out
	i.now += float64(len(l)) / float64(i.sr)
}

func (i *Instrument) renderVoice(v *voice, l, r []float32) {
	s := v.z.sample
	buf := v.z.buf
	n := len(l)
	ch := i.ch[v.ch]
	chGain := ch.volume * ch.expression * ch.mixVolume
	if chGain <= 0 {
		return
	}
	for j := 0; j < n; j++ {
		if v.delay > 0 {
			v.delay--
			continue
		}
		if v.pos >= float64(v.end) {
			v.finished = true
			return
		}
		// envelope
		if s.Amp.Enabled {
			if i.advanceEnvelope(v) {
				return
			}
		}
		// loop
		if v.loop && v.pos >= float64(v.loopEnd) {
			if v.xf > 0 && v.loopEnd-v.loopStart > v.xf {
				span := float64(v.loopEnd - v.loopStart - v.xf)
				v.pos = float64(v.loopStart+v.xf) + math.Mod(v.pos-float64(v.loopStart+v.xf), span)
			} else {
				v.pos = float64(v.loopStart) + math.Mod(v.pos-float64(v.loopStart), float64(v.loopEnd-v.loopStart))
			}
		}
		idx := int(v.pos)
		if idx < 0 || idx+1 >= len(buf.L) {
			v.finished = true
			return
		}
		frac := float32(v.pos - float64(idx))
		l0 := buf.L[idx]*(1-frac) + buf.L[idx+1]*frac
		r0 := buf.R[idx]*(1-frac) + buf.R[idx+1]*frac
		// Loop crossfade: during the last xf frames before loopEnd, fade in the
		// loop-start region while the tail fades out; the wrap then lands at
		// loopStart+xf, so the splice is continuous.
		if v.loop && v.xf > 0 && v.pos >= float64(v.loopEnd-v.xf) && v.pos < float64(v.loopEnd) {
			d := v.pos - float64(v.loopEnd-v.xf)
			fi := v.loopStart + int(d)
			if fi >= v.loopStart && fi+1 < len(buf.L) {
				fl := buf.L[fi]*(1-frac) + buf.L[fi+1]*frac
				fr := buf.R[fi]*(1-frac) + buf.R[fi+1]*frac
				w := float32(d / float64(v.xf))
				var a, b float32
				if v.xfMode == "linear" {
					a, b = 1-w, w
				} else {
					a, b = float32(math.Cos(float64(w)*math.Pi/2)), float32(math.Sin(float64(w)*math.Pi/2))
				}
				l0 = l0*a + fl*b
				r0 = r0*a + fr*b
			}
		}
		// state-variable low-pass
		if v.f > 0 {
			v.lowL += v.f * v.bandL
			hiL := l0 - v.lowL - v.q*v.bandL
			v.bandL += v.f * hiL
			l0 = v.lowL
			v.lowR += v.f * v.bandR
			hiR := r0 - v.lowR - v.q*v.bandR
			v.bandR += v.f * hiR
			r0 = v.lowR
		}
		g := float32(v.gain * chGain * v.env * i.gainAdjust)
		panPos := (v.z.pan+100)/200 + (ch.pan - 0.5)
		if panPos < 0 {
			panPos = 0
		}
		if panPos > 1 {
			panPos = 1
		}
		lp := float32(math.Cos(panPos * math.Pi / 2))
		rp := float32(math.Sin(panPos * math.Pi / 2))
		l[j] += l0 * g * lp
		r[j] += r0 * g * rp
		v.pos += v.step
		v.age += 1 / float64(i.sr)
	}
}

// advanceEnvelope steps the amp envelope for one sample. It reports true when
// the voice has finished releasing. Stage progress is shaped by the preset's
// attack/decay/release curves (-100 logarithmic … 0 linear … 100 exponential).
func (i *Instrument) advanceEnvelope(v *voice) bool {
	e := &v.z.sample.Amp
	dt := 1 / float64(i.sr)
	switch v.stage {
	case 0: // delay
		v.stageT += dt
		if v.stageT >= e.Delay {
			v.stage = 1
			v.stageT = 0
		}
	case 1: // attack
		if e.Attack <= 0 {
			v.env = 1
			v.stage = 2
			v.stageT = 0
			return false
		}
		v.stageT += dt
		p := v.stageT / e.Attack
		if p >= 1 {
			v.env = 1
			v.stage = 2
			v.stageT = 0
			return false
		}
		v.env = envelopeCurve(p, e.AttackCurve)
	case 2: // hold
		v.stageT += dt
		if v.stageT >= e.Hold {
			v.stage = 3
			v.stageT = 0
		}
	case 3: // decay
		if e.Decay <= 0 {
			v.env = e.Sustain
			v.stage = 4
			return false
		}
		v.stageT += dt
		p := v.stageT / e.Decay
		if p >= 1 {
			v.env = e.Sustain
			v.stage = 4
			return false
		}
		v.env = 1 - (1-e.Sustain)*envelopeCurve(p, e.DecayCurve)
	case 5: // release
		if e.Release <= 0 {
			v.env = 0
			v.finished = true
			return true
		}
		v.stageT += dt
		p := v.stageT / e.Release
		if p >= 1 {
			v.env = 0
			v.finished = true
			return true
		}
		v.env = v.releaseLevel * (1 - envelopeCurve(p, e.ReleaseCurve))
	}
	return false
}

// envelopeCurve maps linear stage progress to the DecentSampler curve shape.
func envelopeCurve(p, curve float64) float64 {
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return 1
	}
	if curve == 0 {
		return p
	}
	k := math.Abs(curve) / 20
	if k < 1e-4 {
		return p
	}
	den := math.Exp(k) - 1
	if curve > 0 {
		return (math.Exp(k*p) - 1) / den
	}
	return 1 - (math.Exp(k*(1-p))-1)/den
}

// VoiceCount is the current polyphony.
func (i *Instrument) VoiceCount() int {
	i.mu.Lock()
	defer i.mu.Unlock()
	return len(i.voices)
}

// SortZonesForTest is a small helper for deterministic tests.
func (i *Instrument) SortZonesForTest() {
	i.mu.Lock()
	defer i.mu.Unlock()
	sort.SliceStable(i.zones, func(a, b int) bool { return i.zones[a].sample.Path < i.zones[b].sample.Path })
}
