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
	mu       sync.Mutex
	preset   *dspreset.Preset
	zones    []*zone
	groups   []*groupState
	sr       int
	voices   []*voice
	ch       [16]channelState
	lastNote int
	errors   []string
}

type zone struct {
	group  int
	sample *dspreset.Sample
	buf    *Buffer
	gain   float64
	pan    float64
	rate   float64
	cutoff float64
	q      float64
}

type groupState struct {
	rr  int
	seq string
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
	inst := &Instrument{preset: p, sr: sampleRate}
	buffers := map[string]*Buffer{}
	for gi, g := range p.Groups {
		if !g.Enabled {
			continue
		}
		gs := &groupState{seq: g.SeqMode}
		inst.groups = append(inst.groups, gs)
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
				group:  gi,
				sample: s,
				buf:    buf,
				gain:   p.Volume * g.Volume * s.Volume,
				pan:    s.Pan,
				rate:   float64(buf.Rate) / float64(sampleRate),
				cutoff: s.FilterCutoff,
				q:      s.FilterQ,
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
}

// Info returns a summary.
func (i *Instrument) Info() Stats {
	i.mu.Lock()
	defer i.mu.Unlock()
	st := Stats{Path: i.preset.Path, Name: i.preset.Name, Groups: len(i.groups), Zones: len(i.zones)}
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
	return st
}

// Controls exposes parsed instrument UI controls.
func (i *Instrument) Controls() []dspreset.Control { return i.preset.Controls }

// NoteOn triggers matching zones.
func (i *Instrument) NoteOn(key, vel, ch int) {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.preset == nil {
		return
	}
	i.lastNote = key
	for gi, g := range i.preset.Groups {
		if !g.Enabled {
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
			if s.Trigger == "release" {
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
		// Round robin within a group.
		if gi < len(i.groups) && (g.SeqMode == "round_robin" || g.SeqMode == "random" || g.SeqMode == "true_random") {
			n := i.groups[gi].rr
			i.groups[gi].rr++
			if g.SeqMode == "random" || g.SeqMode == "true_random" {
				matches = []*zone{matches[rand.Intn(len(matches))]}
			} else {
				matches = []*zone{matches[n%len(matches)]}
			}
		}
		for _, z := range matches {
			i.startVoice(z, key, vel, ch)
		}
	}
}

func (i *Instrument) startVoice(z *zone, key, vel, ch int) {
	s := z.sample
	semis := (float64(key-s.Root))*s.PitchKeyTrack + s.Tuning
	step := math.Pow(2, semis/12) * z.rate
	gain := z.gain
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
		env: 0,
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
		}
	}
	v.pos = float64(v.start)
	if s.Amp.Enabled {
		v.stage = 0
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
	for _, z := range i.zones {
		s := z.sample
		if s.Trigger != "release" || key < s.LoNote || key > s.HiNote {
			continue
		}
		i.startVoice(z, key, 100, ch)
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
			v.pos = float64(v.loopStart) + math.Mod(v.pos-float64(v.loopStart), float64(v.loopEnd-v.loopStart))
		}
		idx := int(v.pos)
		if idx < 0 || idx+1 >= len(buf.L) {
			v.finished = true
			return
		}
		frac := float32(v.pos - float64(idx))
		l0 := buf.L[idx]*(1-frac) + buf.L[idx+1]*frac
		r0 := buf.R[idx]*(1-frac) + buf.R[idx+1]*frac
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
		g := float32(v.gain * chGain * v.env)
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
// the voice has finished releasing.
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
			return false
		}
		v.env += 1 / (e.Attack * float64(i.sr))
		if v.env >= 1 {
			v.env = 1
			v.stage = 2
			v.stageT = 0
		}
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
		v.env -= (1 - e.Sustain) / (e.Decay * float64(i.sr))
		if v.env <= e.Sustain {
			v.env = e.Sustain
			v.stage = 4
		}
	case 5: // release
		if e.Release <= 0 {
			v.env = 0
			v.finished = true
			return true
		}
		v.env -= v.releaseLevel / (e.Release * float64(i.sr))
		if v.env <= 0 {
			v.env = 0
			v.finished = true
			return true
		}
	}
	return false
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
