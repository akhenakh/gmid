package audio

import (
	"math"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/sinshu/go-meltysynth/meltysynth"

	"github.com/akhenakh/gmid/internal/dspreset"
	"github.com/akhenakh/gmid/internal/sampler"
	"github.com/akhenakh/gmid/internal/song"
)

// ChannelMix is the console mixer state for one MIDI channel.
type ChannelMix struct {
	Volume float64 // 0..1.5, 1 = song's own level
	Pan    float64 // 0..1, 0.5 = centre
	Chorus float64 // 0..1
	Reverb float64 // 0..1
}

func defaultMix(pan, reverb, chorus int) ChannelMix {
	return ChannelMix{
		Volume: 1,
		Pan:    clamp01(float64(pan) / 127),
		Reverb: clamp01(float64(reverb) / 127),
		Chorus: clamp01(float64(chorus) / 127),
	}
}

// Snapshot is the audio-thread state the UI reads once per frame.
type Snapshot struct {
	Time    float64
	Playing bool
	Peak    float64
	Ready   bool
	CountIn bool
	Error   string
}

// Engine is the realtime playback engine. The synth is owned by the audio
// thread; UI changes take the mutex for short critical sections.
type Engine struct {
	mu        sync.Mutex
	synth     *meltysynth.Synthesizer
	soundfont *meltysynth.SoundFont
	decent    *sampler.Instrument
	sfPath    string
	sfErr     string

	timeline *Timeline
	song     *song.Song

	playing   bool
	pos       float64
	speed     float64
	transpose int

	loopOn    bool
	loopStart float64
	loopEnd   float64

	metronomeOn  bool
	metronomeVol float64
	masterVol    float64
	masterTone   float64
	compressor   bool
	toneStateL   float32
	toneStateR   float32
	compEnv      float32

	countInBars    int
	countInActive  bool
	countInSong    float64
	countInElapsed float64
	countInTotal   float64
	countInClicks  []countClick

	audible [256]bool
	chMix   [16]ChannelMix

	decentGainDB float64

	idx     int
	beatIdx int
	beats   []float64
	bars    []float64
	peak    float64
	ready   bool

	sr      int
	tmpL    []float32
	tmpR    []float32
	smpL    []float32
	smpR    []float32
	out     *malgoOutput
	running atomic.Bool
}

// New creates an engine. A soundfont must be installed before it produces sound.
func New(sampleRate int) *Engine {
	if sampleRate <= 0 {
		sampleRate = 48000
	}
	e := &Engine{sr: sampleRate, speed: 1, masterVol: 0.85, metronomeVol: 0.6}
	for i := range e.audible {
		e.audible[i] = true
	}
	for i := range e.chMix {
		e.chMix[i] = ChannelMix{Volume: 1, Pan: 0.5, Reverb: 0.4}
	}
	e.tmpL = make([]float32, 4096)
	e.tmpR = make([]float32, 4096)
	e.smpL = make([]float32, 4096)
	e.smpR = make([]float32, 4096)
	return e
}

// SampleRate returns the engine's output rate.
func (e *Engine) SampleRate() int { return e.sr }

// LoadSoundFont reads an SF2 file and installs it. It is safe to call while
// audio is running; the swap happens under the engine lock.
func (e *Engine) LoadSoundFont(path string) error {
	f, err := os.Open(path)
	if err != nil {
		e.mu.Lock()
		e.sfErr = err.Error()
		e.mu.Unlock()
		return err
	}
	defer func() { _ = f.Close() }()
	sf, err := meltysynth.NewSoundFont(f)
	if err != nil {
		e.mu.Lock()
		e.sfErr = err.Error()
		e.mu.Unlock()
		return err
	}
	settings := meltysynth.NewSynthesizerSettings(int32(e.sr))
	settings.BlockSize = 64
	settings.MaximumPolyphony = 256
	synth, err := meltysynth.NewSynthesizer(sf, settings)
	if err != nil {
		e.mu.Lock()
		e.sfErr = err.Error()
		e.mu.Unlock()
		return err
	}
	e.mu.Lock()
	e.soundfont, e.synth, e.sfPath = sf, synth, path
	e.sfErr = ""
	e.ready = true
	e.applyMixLocked()
	e.applyStateLocked(e.pos)
	e.mu.Unlock()
	return nil
}

// SoundFontPath returns the installed soundfont path.
func (e *Engine) SoundFontPath() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.sfPath
}

// LoadDecentPreset loads a DecentSampler .dspreset or .dslibrary and routes
// melodic notes through it. Drums keep using the soundfont.
func (e *Engine) LoadDecentPreset(path string) error {
	p, err := dspreset.Load(path)
	if err != nil {
		return err
	}
	inst, err := sampler.Load(p, e.sr)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.decent = inst
	e.decent.SetGainAdjust(e.decentGainDB)
	for ch := 0; ch < 16; ch++ {
		m := e.chMix[ch]
		e.decent.SetChannelMix(ch, m.Volume, m.Pan)
	}
	e.mu.Unlock()
	return nil
}

// SetDecentGain applies the user's instrument gain in decibels and remembers it
// for instruments loaded later.
func (e *Engine) SetDecentGain(db float64) {
	if db < -48 {
		db = -48
	}
	if db > 24 {
		db = 24
	}
	e.mu.Lock()
	e.decentGainDB = db
	if e.decent != nil {
		e.decent.SetGainAdjust(db)
	}
	e.mu.Unlock()
}

// UnloadDecent removes the DecentSampler instrument.
func (e *Engine) UnloadDecent() {
	e.mu.Lock()
	e.decent = nil
	e.mu.Unlock()
}

// DecentInfo reports the loaded instrument.
func (e *Engine) DecentInfo() (sampler.Stats, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.decent == nil {
		return sampler.Stats{}, false
	}
	return e.decent.Info(), true
}

// DecentControls returns the loaded instrument's UI controls.
func (e *Engine) DecentControls() []dspreset.Control {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.decent == nil {
		return nil
	}
	return e.decent.Controls()
}

// Load installs a song and resets transport state.
func (e *Engine) Load(s *song.Song) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.song = s
	if s == nil {
		e.timeline = nil
		e.playing = false
		e.pos = 0
		return
	}
	e.timeline = Compile(s)
	e.beats = s.EffectiveBeats()
	e.bars = s.BarStarts()
	e.playing = false
	e.pos = s.OriginSec()
	e.ready = e.synth != nil
	for i := range s.Tracks {
		e.audible[i] = !s.Tracks[i].Muted && !s.Tracks[i].Excluded
	}
	if anySolo := s.AnySolo(); anySolo {
		for i := range s.Tracks {
			if !s.Tracks[i].Solo {
				e.audible[i] = false
			}
		}
	}
	for i := range s.Channels {
		c := &s.Channels[i]
		e.chMix[c.Number] = defaultMix(c.InitialPan, c.InitialReverb, c.InitialChorus)
	}
	e.resetLocked(e.pos)
}

// Play starts or resumes from a song time.
func (e *Engine) Play(from float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.timeline == nil {
		return
	}
	if from < e.timeline.Origin || from >= e.timeline.Duration {
		from = e.timeline.Origin
	}
	e.pos = from
	e.resetLocked(from)
	e.prepareCountInLocked(from)
	e.playing = true
}

// Pause stops playback but keeps the playhead.
func (e *Engine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.playing = false
	e.countInActive = false
	if e.synth != nil {
		e.synth.NoteOffAll(false)
	}
	if e.decent != nil {
		e.decent.Reset()
	}
}

// Stop stops and rewinds to the visible start.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.playing = false
	e.countInActive = false
	if e.timeline != nil {
		e.pos = e.timeline.Origin
		e.resetLocked(e.pos)
	}
}

// Seek moves the playhead, preserving the playing state.
func (e *Engine) Seek(t float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.timeline == nil {
		return
	}
	if t < e.timeline.Origin {
		t = e.timeline.Origin
	}
	if t > e.timeline.Duration {
		t = e.timeline.Duration
	}
	e.pos = t
	e.countInActive = false
	e.resetLocked(t)
}

// SetSpeed sets the tempo ratio.
func (e *Engine) SetSpeed(v float64) {
	if v <= 0 {
		v = 1
	}
	e.mu.Lock()
	e.speed = v
	e.mu.Unlock()
}

// SetTranspose changes the playback key.
func (e *Engine) SetTranspose(n int) {
	e.mu.Lock()
	e.transpose = n
	e.mu.Unlock()
}

// SetLoop enables or configures the loop region.
func (e *Engine) SetLoop(enabled bool, start, end float64) {
	e.mu.Lock()
	e.loopOn = enabled
	e.loopStart, e.loopEnd = start, end
	e.mu.Unlock()
}

// SetMetronome enables the click and sets its level.
func (e *Engine) SetMetronome(on bool, volume float64) {
	e.mu.Lock()
	e.metronomeOn = on
	e.metronomeVol = clamp01(volume)
	e.mu.Unlock()
}

// SetCountIn sets the number of bars of clicks before a start (0 = off).
func (e *Engine) SetCountIn(bars int) {
	if bars < 0 {
		bars = 0
	}
	if bars > 4 {
		bars = 4
	}
	e.mu.Lock()
	e.countInBars = bars
	e.mu.Unlock()
}

// SetMasterVolume sets the song-bus gain.
func (e *Engine) SetMasterVolume(v float64) {
	e.mu.Lock()
	e.masterVol = v
	e.mu.Unlock()
}

// SetMasterTone sets the master tone control (0 dark … 1 bright).
func (e *Engine) SetMasterTone(v float64) {
	e.mu.Lock()
	e.masterTone = clamp01(v)
	e.mu.Unlock()
}

// SetCompressor enables the master compressor.
func (e *Engine) SetCompressor(on bool) {
	e.mu.Lock()
	e.compressor = on
	e.mu.Unlock()
}

// SetTrackAudible applies mute/solo for a track.
func (e *Engine) SetTrackAudible(track int, audible bool) {
	if track < 0 || track >= len(e.audible) {
		return
	}
	e.mu.Lock()
	e.audible[track] = audible
	e.mu.Unlock()
}

// SetChannelMix updates one channel's console strip.
func (e *Engine) SetChannelMix(ch int, volume, pan, chorus, reverb float64) {
	if ch < 0 || ch >= 16 {
		return
	}
	e.mu.Lock()
	e.chMix[ch] = ChannelMix{
		Volume: volume, Pan: clamp01(pan), Chorus: clamp01(chorus), Reverb: clamp01(reverb),
	}
	if e.synth != nil {
		e.sendMixLocked(ch)
	}
	e.mu.Unlock()
}

// NoteOn plays a live note on the reserved live channel.
func (e *Engine) NoteOn(ch, key, vel int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.decent != nil {
		e.decent.NoteOn(clampKey(key), clampVel(vel), 15)
		return
	}
	if e.synth == nil {
		return
	}
	e.synth.ProcessMidiMessage(15, 0xC0, 0, 0)
	e.synth.NoteOn(15, int32(clampKey(key)), int32(clampVel(vel)))
}

// NoteOff releases a live note.
func (e *Engine) NoteOff(ch, key int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.decent != nil {
		e.decent.NoteOff(clampKey(key), 15)
		return
	}
	if e.synth == nil {
		return
	}
	e.synth.NoteOff(15, int32(clampKey(key)))
}

// Control sends a live controller move (pedals, expression, …).
func (e *Engine) Control(ch, cc, value int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.decent != nil {
		e.decent.Control(15, cc, clampCC(value))
	}
	if e.synth == nil {
		return
	}
	e.synth.ProcessMidiMessage(15, 0xB0, int32(clampCC(cc)), int32(clampCC(value)))
}

// PitchBend sends a live pitch bend in -8192..8191.
func (e *Engine) PitchBend(ch, value int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.synth == nil {
		return
	}
	if value < -8192 {
		value = -8192
	}
	if value > 8191 {
		value = 8191
	}
	v := value + 8192
	e.synth.ProcessMidiMessage(15, 0xE0, int32(v&0x7f), int32(v>>7))
}

// Snapshot returns the UI-facing state.
func (e *Engine) Snapshot() Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	return Snapshot{Time: e.pos, Playing: e.playing, Peak: e.peak, Ready: e.ready, CountIn: e.countInActive, Error: e.sfErr}
}

// Close stops the device.
func (e *Engine) Close() {
	e.mu.Lock()
	e.playing = false
	dev := e.out
	e.out = nil
	e.mu.Unlock()
	if dev != nil {
		dev.close()
	}
}

func (e *Engine) resetLocked(pos float64) {
	if (e.synth == nil && e.decent == nil) || e.timeline == nil {
		return
	}
	if e.synth != nil {
		e.synth.Reset()
	}
	if e.decent != nil {
		e.decent.Reset()
	}
	e.applyMixLocked()
	e.applyStateLocked(pos)
}

// applyStateLocked rebuilds controller/program state as it was at pos.
func (e *Engine) applyStateLocked(pos float64) {
	if (e.synth == nil && e.decent == nil) || e.timeline == nil {
		return
	}
	n := e.timeline.IndexAfter(pos)
	for i := 0; i < n; i++ {
		ev := e.timeline.Events[i]
		if ev.Kind == EvProgram || ev.Kind == EvCC {
			e.dispatchLocked(ev)
		}
	}
	e.idx = n
	e.beatIdx = firstAtOrAfter(e.beats, pos)
}

func firstAtOrAfter(a []float64, t float64) int {
	return sort.Search(len(a), func(i int) bool { return a[i] >= t })
}

func (e *Engine) applyMixLocked() {
	if e.synth == nil && e.decent == nil {
		return
	}
	for ch := 0; ch < 16; ch++ {
		e.sendMixLocked(ch)
	}
}

func (e *Engine) sendMixLocked(ch int) {
	m := e.chMix[ch]
	if e.decent != nil {
		e.decent.SetChannelMix(ch, m.Volume, m.Pan)
	}
	if e.synth == nil {
		return
	}
	e.synth.ProcessMidiMessage(int32(ch), 0xB0, 10, int32(math.Round(clamp01(m.Pan)*127)))
	e.synth.ProcessMidiMessage(int32(ch), 0xB0, 91, int32(math.Round(clamp01(m.Reverb)*127)))
	e.synth.ProcessMidiMessage(int32(ch), 0xB0, 93, int32(math.Round(clamp01(m.Chorus)*127)))
	vol := 100.0
	if e.song != nil {
		if c := e.song.ChannelAt(ch); c != nil && c.InitialVolume > 0 {
			vol = float64(c.InitialVolume)
		}
	}
	e.synth.ProcessMidiMessage(int32(ch), 0xB0, 7, int32(math.Round(clamp01(m.Volume*vol/100)*127)))
}

func (e *Engine) dispatchLocked(ev Event) {
	if e.synth == nil && e.decent == nil {
		return
	}
	switch ev.Kind {
	case EvProgram:
		if e.synth == nil {
			return
		}
		bank := ev.B
		if bank >= 0 {
			e.synth.ProcessMidiMessage(int32(ev.Channel), 0xB0, 0, int32(bank/128))
			e.synth.ProcessMidiMessage(int32(ev.Channel), 0xB0, 32, int32(bank%128))
		}
		e.synth.ProcessMidiMessage(int32(ev.Channel), 0xC0, int32(ev.A), 0)
	case EvCC:
		val := ev.B
		if ev.A == 7 {
			m := e.chMix[ev.Channel]
			val = int(math.Round(clamp01(float64(val)*m.Volume/127) * 127))
		}
		if e.decent != nil {
			e.decent.Control(ev.Channel, ev.A, clampCC(ev.B))
		}
		if e.synth != nil {
			e.synth.ProcessMidiMessage(int32(ev.Channel), 0xB0, int32(ev.A), int32(clampCC(val)))
		}
	case EvNoteOn:
		if ev.Track >= 0 && !e.audible[ev.Track] {
			return
		}
		key := ev.A
		drum := e.isDrum(ev.Track)
		if !drum {
			key += e.transpose
		}
		if e.decent != nil && !drum {
			e.decent.NoteOn(clampKey(key), clampVel(ev.B), ev.Channel)
			return
		}
		if e.synth != nil {
			e.synth.NoteOn(int32(ev.Channel), int32(clampKey(key)), int32(clampVel(ev.B)))
		}
	case EvNoteOff:
		key := ev.A
		drum := e.isDrum(ev.Track)
		if !drum {
			key += e.transpose
		}
		if e.decent != nil && !drum {
			e.decent.NoteOff(clampKey(key), ev.Channel)
			return
		}
		if e.synth != nil {
			e.synth.NoteOff(int32(ev.Channel), int32(clampKey(key)))
		}
	}
}

func (e *Engine) isDrum(track int) bool {
	if e.song == nil || track < 0 || track >= len(e.song.Tracks) {
		return false
	}
	return e.song.Tracks[track].IsDrum
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clampKey(k int) int {
	if k < 0 {
		return 0
	}
	if k > 127 {
		return 127
	}
	return k
}

func clampVel(v int) int {
	if v < 1 {
		return 1
	}
	if v > 127 {
		return 127
	}
	return v
}

func clampCC(v int) int {
	if v < 0 {
		return 0
	}
	if v > 127 {
		return 127
	}
	return v
}
