package gp

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

const ticksPerWhole = 3840 // 960 PPQ, four quarters

type version [3]int

type parser struct {
	r      *reader
	family int
	ver    version

	tempo    int
	tempos   []tempoChange
	sigs     []timeSig
	headers  []measureHeader
	tracks   []*track
	channels []int
	title    string
}

type tempoChange struct {
	tick int64
	bpm  float64
}

type timeSig struct {
	tick     int64
	num, den int
}

type measureHeader struct {
	tick     int64
	num, den int
}

type track struct {
	Name        string
	Program     int
	Channel     int
	IsDrum      bool
	Visible     bool
	Solo        bool
	Muted       bool
	Color       string
	Tuning      []int
	Notes       []Note
	lastByVoice map[[2]int]int
}

// Note is one parsed Guitar Pro note, in ticks at 960 PPQ.
type Note struct {
	Start int64
	Dur   int64
	Key   int
	Vel   int
}

// Parse parses GP3/GP4/GP5 bytes into the song model.
func Parse(data []byte, path string) (*song.Song, error) {
	r := &reader{data: data}
	versionString := r.fixedString(30)
	family, ver, err := parseVersion(versionString)
	if err != nil {
		return nil, err
	}
	p := &parser{r: r, family: family, ver: ver}
	switch family {
	case 3:
		p.readSong3()
	case 4:
		p.readSong4()
	case 5:
		p.readSong5()
	}
	if r.err != nil {
		return nil, fmt.Errorf("guitar pro %d: %w", family, r.err)
	}
	return p.build(path)
}

func parseVersion(v string) (int, version, error) {
	v = strings.TrimRight(v, "\x00 ")
	if strings.HasPrefix(v, "CLIPBOARD") {
		return 0, version{}, fmt.Errorf("guitar pro: clipboard data is not a song")
	}
	i := strings.LastIndex(v, "v")
	if i < 0 {
		return 0, version{}, fmt.Errorf("guitar pro: unknown version %q", v)
	}
	parts := strings.Split(strings.TrimSpace(v[i+1:]), ".")
	var ver version
	for j := 0; j < len(parts) && j < 3; j++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[j]))
		if err != nil {
			return 0, ver, fmt.Errorf("guitar pro: bad version %q", v)
		}
		ver[j] = n
	}
	if ver[0] < 3 || ver[0] > 5 {
		return 0, ver, fmt.Errorf("guitar pro: unsupported version %q", v)
	}
	return ver[0], ver, nil
}

func (v version) after(major, minor, patch int) bool {
	if v[0] != major {
		return v[0] > major
	}
	if v[1] != minor {
		return v[1] > minor
	}
	return v[2] > patch
}

func (p *parser) readSong3() {
	p.readInfo(8)
	p.r.bool() // triplet feel
	p.tempo = int(p.r.i32())
	p.r.i32() // key
	p.readChannels()
	measureCount := p.r.count(p.r.i32(), maxMeasures, "measure")
	trackCount := p.r.count(p.r.i32(), maxTracks, "track")
	p.readHeaders(measureCount)
	p.readTracks3(trackCount)
	p.readMeasures3()
}

func (p *parser) readSong4() {
	p.readInfo(8)
	p.r.bool() // triplet feel
	p.readLyrics()
	p.tempo = int(p.r.i32())
	p.r.i32()   // key
	p.r.skip(1) // octave
	p.readChannels()
	measureCount := p.r.count(p.r.i32(), maxMeasures, "measure")
	trackCount := p.r.count(p.r.i32(), maxTracks, "track")
	p.readHeaders(measureCount)
	p.readTracks3(trackCount)
	p.readMeasures3()
}

func (p *parser) readSong5() {
	p.readInfo(9)
	p.readLyrics()
	if p.ver.after(5, 0, 0) {
		p.r.skip(8)  // master volume, reserved
		p.r.skip(11) // 10-band equalizer + gain
	}
	p.readPageSetup()
	p.r.intByteString() // tempo name
	p.tempo = int(p.r.i32())
	if p.ver.after(5, 0, 0) {
		p.r.skip(1) // hide tempo
	}
	p.r.skip(1) // key
	p.r.skip(4) // octave
	p.readChannels()
	p.r.skip(19 * 2) // directions
	p.r.skip(4)      // master reverb
	measureCount := p.r.count(p.r.i32(), maxMeasures, "measure")
	trackCount := p.r.count(p.r.i32(), maxTracks, "track")
	if p.r.err != nil {
		return
	}
	p.readHeaders5(measureCount)
	p.readTracks5(trackCount)
	p.readMeasures5()
}

func (p *parser) readInfo(fields int) {
	for i := 0; i < fields; i++ {
		v := p.r.intByteString()
		switch i {
		case 0:
			p.title = v
		}
	}
	comments := p.r.count(p.r.i32(), maxComments, "comment")
	for i := 0; i < comments; i++ {
		p.r.intByteString()
	}
}

func (p *parser) readLyrics() {
	p.r.skip(4) // track choice
	for i := 0; i < 5; i++ {
		p.r.skip(4) // starting measure
		p.r.intString()
	}
}

func (p *parser) readPageSetup() {
	p.r.skip(8)  // page size
	p.r.skip(16) // margins
	p.r.skip(4)  // score size
	p.r.skip(2)  // header/footer flags
	for i := 0; i < 10; i++ {
		p.r.intByteString()
	}
}

func (p *parser) readChannels() {
	p.channels = make([]int, 64)
	for i := range p.channels {
		p.channels[i] = int(p.r.i32())
		p.r.skip(6) // volume, balance, chorus, reverb, phaser, tremolo
		p.r.skip(2) // padding
	}
}

func (p *parser) readHeaders(n int) {
	p.headers = make([]measureHeader, 0, n)
	start := int64(960)
	num, den := 4, 4
	for i := 0; i < n; i++ {
		flags := p.r.u8()
		if flags&0x01 != 0 {
			num = int(p.r.i8())
		}
		if flags&0x02 != 0 {
			den = int(p.r.i8())
		}
		if flags&0x08 != 0 {
			p.r.i8() // repeat close
		}
		if flags&0x10 != 0 {
			p.r.u8() // alternate ending
		}
		if flags&0x20 != 0 {
			p.r.intByteString()
			p.r.color()
		}
		if flags&0x40 != 0 {
			p.r.skip(2) // key root + type
		}
		num, den = saneMeter(num, den)
		p.addHeader(start, num, den)
		start += measureTicks(num, den)
	}
}

func (p *parser) readHeaders5(n int) {
	p.headers = make([]measureHeader, 0, n)
	start := int64(960)
	num, den := 4, 4
	for i := 0; i < n; i++ {
		if i > 0 {
			p.r.skip(1) // always zero
		}
		flags := p.r.u8()
		if flags&0x01 != 0 {
			num = int(p.r.i8())
		}
		if flags&0x02 != 0 {
			den = int(p.r.i8())
		}
		if flags&0x08 != 0 {
			p.r.i8() // repeat close
		}
		if flags&0x20 != 0 {
			p.r.intByteString()
			p.r.color()
		}
		if flags&0x40 != 0 {
			p.r.skip(2) // key root + type
		}
		if flags&0x10 != 0 {
			p.r.u8() // alternate ending
		}
		if flags&0x03 != 0 {
			p.r.skip(4) // beams
		}
		if flags&0x10 == 0 {
			p.r.skip(1)
		}
		p.r.u8() // triplet feel
		num, den = saneMeter(num, den)
		p.addHeader(start, num, den)
		start += measureTicks(num, den)
	}
}

func saneMeter(num, den int) (int, int) {
	if num <= 0 || num > 32 {
		num = 4
	}
	switch den {
	case 1, 2, 4, 8, 16, 32:
	default:
		den = 4
	}
	return num, den
}

func measureTicks(num, den int) int64 {
	t := int64(num) * ticksPerWhole / int64(den)
	if t < 1 {
		return 1
	}
	return t
}

func (p *parser) addHeader(tick int64, num, den int) {
	p.headers = append(p.headers, measureHeader{tick: tick, num: num, den: den})
	if len(p.sigs) == 0 || p.sigs[len(p.sigs)-1].num != num || p.sigs[len(p.sigs)-1].den != den {
		p.sigs = append(p.sigs, timeSig{tick: tick, num: num, den: den})
	}
}

func (p *parser) readTracks3(n int) {
	p.tracks = make([]*track, 0, n)
	for i := 0; i < n; i++ {
		tr := p.readTrack3()
		tr.Name = trackName(tr.Name, i)
		p.tracks = append(p.tracks, tr)
	}
}

func (p *parser) readTrack3() *track {
	flags := p.r.u8()
	tr := &track{
		Name:        p.r.fixedString(40),
		Visible:     true,
		Color:       "",
		lastByVoice: map[[2]int]int{},
	}
	tuning := make([]int, 7)
	stringCount := p.r.count(p.r.i32(), 7, "string")
	for i := range tuning {
		tuning[i] = int(p.r.i32())
	}
	if stringCount == 0 {
		stringCount = 6
	}
	tr.Tuning = append([]int(nil), tuning[:stringCount]...)
	p.r.skip(4) // port
	channel := int(p.r.i32()) - 1
	p.r.skip(4) // effect channel
	p.r.skip(4) // fret count
	p.r.skip(4) // capo
	tr.Color = p.r.color()
	if channel < 0 || channel >= len(p.channels) {
		channel = 0
	}
	tr.Channel = channel
	tr.Program = p.program(channel)
	tr.IsDrum = flags&0x01 != 0 || channel == 9
	return tr
}

func (p *parser) readTracks5(n int) {
	p.tracks = make([]*track, 0, n)
	for i := 0; i < n; i++ {
		tr := p.readTrack5(i == 0)
		tr.Name = trackName(tr.Name, i)
		p.tracks = append(p.tracks, tr)
	}
	if p.ver == (version{5, 0, 0}) {
		p.r.skip(2)
	} else {
		p.r.skip(1)
	}
}

func (p *parser) readTrack5(first bool) *track {
	if first || p.ver == (version{5, 0, 0}) {
		p.r.skip(1)
	}
	flags := p.r.u8()
	tr := &track{
		Name:        p.r.fixedString(40),
		Visible:     flags&0x08 != 0,
		Solo:        flags&0x10 != 0,
		Muted:       flags&0x20 != 0,
		lastByVoice: map[[2]int]int{},
	}
	tuning := make([]int, 7)
	stringCount := p.r.count(p.r.i32(), 7, "string")
	for i := range tuning {
		tuning[i] = int(p.r.i32())
	}
	if stringCount == 0 {
		stringCount = 6
	}
	tr.Tuning = append([]int(nil), tuning[:stringCount]...)
	p.r.skip(4) // port
	channel := int(p.r.i32()) - 1
	p.r.skip(4) // effect channel
	p.r.skip(4) // fret count
	p.r.skip(4) // capo
	tr.Color = p.r.color()
	p.r.skip(2)  // track settings flags
	p.r.skip(1)  // auto accentuation
	p.r.skip(1)  // bank
	p.r.skip(1)  // humanize
	p.r.skip(4)  // clef transpose
	p.r.skip(4)  // secondary clef transpose
	p.r.skip(4)  // unknown
	p.r.skip(12) // unknown
	p.readRSEInstrument()
	if p.ver.after(5, 0, 0) {
		p.r.skip(4) // 3-band equalizer + gain
		p.r.intByteString()
		p.r.intByteString()
	}
	if channel < 0 || channel >= len(p.channels) {
		channel = 0
	}
	tr.Channel = channel
	tr.Program = p.program(channel)
	tr.IsDrum = flags&0x01 != 0 || channel == 9
	return tr
}

func (p *parser) readRSEInstrument() {
	p.r.skip(4) // instrument
	p.r.skip(4) // unknown
	p.r.skip(4) // sound bank
	if p.ver == (version{5, 0, 0}) {
		p.r.skip(3) // effect number (short) + padding
	} else {
		p.r.skip(4) // effect number
	}
}

func (p *parser) program(channel int) int {
	if channel < 0 || channel >= len(p.channels) {
		return 0
	}
	prog := p.channels[channel]
	if prog < 0 {
		prog = 0
	}
	if prog > 127 {
		prog = 127
	}
	return prog
}

func trackName(name string, index int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Sprintf("Track %d", index+1)
	}
	return name
}

func (p *parser) readMeasures3() {
	for _, h := range p.headers {
		for _, tr := range p.tracks {
			p.readVoice(tr, h.tick, 0)
		}
	}
}

func (p *parser) readMeasures5() {
	for _, h := range p.headers {
		for _, tr := range p.tracks {
			p.readVoice(tr, h.tick, 0)
			p.readVoice(tr, h.tick, 1)
			p.r.skipOptional(1) // line break; absent in the last measure of some files
		}
	}
}

func (p *parser) readVoice(tr *track, start int64, voice int) {
	beats := p.r.count(p.r.i32(), maxBeats, "beat")
	tick := start
	for i := 0; i < beats; i++ {
		flags := p.r.u8()
		status := 1 // normal
		if flags&0x40 != 0 {
			status = int(p.r.u8())
		}
		dur := p.readDuration(flags)
		if flags&0x02 != 0 {
			p.readChord(len(tr.Tuning))
		}
		if flags&0x04 != 0 {
			p.r.intByteString()
		}
		if flags&0x08 != 0 {
			p.readBeatEffects()
		}
		if flags&0x10 != 0 {
			p.readMixTable(tick)
		}
		p.readNotes(tr, voice, tick, dur)
		if p.family == 5 {
			flags2 := p.r.i16()
			if flags2&0x0800 != 0 {
				p.r.u8() // break secondary beams
			}
		}
		if status != 0 { // empty beats do not advance time
			tick += dur
		}
	}
}

func (p *parser) readDuration(flags uint8) int64 {
	b := int(p.r.i8())
	if b < -2 || b > 5 {
		b = 0
	}
	value := int64(1) << (b + 2)
	ticks := ticksPerWhole / value
	if flags&0x01 != 0 { // dotted
		ticks += ticks / 2
	}
	if flags&0x20 != 0 { // tuplet
		switch int(p.r.i32()) {
		case 3:
			ticks = ticks * 2 / 3
		case 5:
			ticks = ticks * 4 / 5
		case 6:
			ticks = ticks * 4 / 6
		case 7:
			ticks = ticks * 4 / 7
		case 9:
			ticks = ticks * 8 / 9
		case 10:
			ticks = ticks * 8 / 10
		case 11:
			ticks = ticks * 8 / 11
		case 12:
			ticks = ticks * 8 / 12
		case 13:
			ticks = ticks * 8 / 13
		}
	}
	if ticks < 1 {
		ticks = 1
	}
	return ticks
}

func (p *parser) readChord(stringCount int) {
	newFormat := p.r.bool()
	if !newFormat {
		p.r.intByteString() // name
		firstFret := p.r.i32()
		if firstFret != 0 {
			p.r.skip(6 * 4)
		}
		return
	}
	p.r.bool() // sharp
	p.r.skip(3)
	if p.family == 3 {
		p.r.skip(5 * 4) // root, type, extension, bass, tonality
		p.r.skip(1)     // add
		p.r.fixedString(22)
		p.r.skip(3 * 4) // fifth, ninth, eleventh
		p.r.skip(4)     // first fret
		p.r.skip(6 * 4)
		p.r.count(p.r.i32(), 5, "barre")
		p.r.skip(24) // two barres: fret, start, end
		p.r.skip(7)  // omissions
		p.r.skip(1)
		return
	}
	// GP4/GP5 new chord format.
	p.r.skip(2) // root, type
	p.r.skip(1) // extension
	p.r.skip(4) // bass
	p.r.skip(4) // tonality
	p.r.bool()  // add
	p.r.fixedString(22)
	p.r.skip(3) // fifth, ninth, eleventh
	p.r.skip(4) // first fret
	p.r.skip(7 * 4)
	p.r.count(int32(p.r.u8()), 5, "barre")
	p.r.skip(15) // five barres: fret, start, end
	p.r.skip(7)  // omissions
	p.r.skip(1)
	p.r.skip(7) // fingerings
	p.r.bool()  // show
	_ = stringCount
}

func (p *parser) readBeatEffects() {
	if p.family == 3 {
		flags := p.r.u8()
		if flags&0x20 != 0 {
			p.r.skip(1) // slap effect
			p.r.skip(4) // tremolo bar value or tap value
		}
		if flags&0x40 != 0 {
			p.r.skip(2) // stroke down/up
		}
		return
	}
	flags1 := p.r.i8()
	flags2 := p.r.i8()
	if flags1&0x20 != 0 {
		p.r.skip(1) // slap
	}
	if flags2&0x04 != 0 {
		p.readBend() // tremolo bar
	}
	if flags1&0x40 != 0 {
		p.r.skip(2) // stroke
	}
	if flags2&0x02 != 0 {
		p.r.skip(1) // pick stroke
	}
}

func (p *parser) readBend() {
	p.r.skip(1) // type
	p.r.skip(4) // value
	points := p.r.count(p.r.i32(), maxPoints, "bend point")
	for i := 0; i < points; i++ {
		p.r.skip(4) // position
		p.r.skip(4) // value
		p.r.skip(1) // vibrato
	}
}

func (p *parser) readMixTable(tick int64) {
	var tempo int32
	if p.family == 5 {
		p.r.skip(1) // instrument
		p.readRSEInstrument()
		if p.ver == (version{5, 0, 0}) {
			p.r.skip(1)
		}
		var dyn [6]int8
		for i := range dyn {
			dyn[i] = p.r.i8()
		}
		p.r.intByteString() // tempo name
		tempo = p.r.i32()
		for _, v := range dyn {
			if v >= 0 {
				p.r.skip(1) // duration
			}
		}
		if tempo >= 0 {
			p.r.skip(1) // tempo duration
			if p.ver.after(5, 0, 0) {
				p.r.skip(1) // hide tempo
			}
		}
		p.r.skip(1) // flags
		p.r.skip(1) // wah
		if p.ver.after(5, 0, 0) {
			p.r.intByteString() // RSE effect name
			p.r.intByteString() // RSE effect category
		}
	} else {
		p.r.skip(1) // instrument
		var dyn [6]int8
		for i := range dyn {
			dyn[i] = p.r.i8()
		}
		tempo = p.r.i32()
		for _, v := range dyn {
			if v >= 0 {
				p.r.skip(1) // duration
			}
		}
		if tempo >= 0 {
			p.r.skip(1) // tempo duration
		}
		if p.family == 4 {
			p.r.skip(1) // flags
		}
	}
	if tempo > 0 {
		p.addTempo(tick, float64(tempo))
	}
}

func (p *parser) addTempo(tick int64, bpm float64) {
	if bpm < 1 || bpm > 1000 {
		return
	}
	if tick == 0 && p.tempo > 0 {
		p.tempo = int(bpm)
		return
	}
	if len(p.tempos) > 0 && p.tempos[len(p.tempos)-1].tick == tick {
		p.tempos[len(p.tempos)-1].bpm = bpm
		return
	}
	p.tempos = append(p.tempos, tempoChange{tick: tick, bpm: bpm})
}

func (p *parser) readNotes(tr *track, voice int, start, dur int64) {
	flags := p.r.u8()
	for i := 0; i < len(tr.Tuning); i++ {
		if flags&(1<<uint(7-(i+1))) == 0 {
			continue
		}
		p.readNote(tr, voice, i+1, start, dur)
	}
}

func (p *parser) readNote(tr *track, voice, stringNo int, start, dur int64) {
	flags := p.r.u8()
	noteType := 1
	fret := 0
	dyn := -1
	if p.family == 5 {
		if flags&0x20 != 0 {
			noteType = int(p.r.u8())
		}
		if flags&0x10 != 0 {
			dyn = int(p.r.i8())
		}
		if flags&0x20 != 0 {
			fret = int(p.r.i8())
		}
		if flags&0x80 != 0 {
			p.r.skip(2)
		}
		if flags&0x01 != 0 {
			p.r.f64() // duration percent
		}
		p.r.u8() // swap accidentals
		if flags&0x08 != 0 {
			p.readNoteEffects()
		}
	} else {
		if flags&0x20 != 0 {
			noteType = int(p.r.u8())
		}
		if flags&0x01 != 0 {
			p.r.skip(2) // per-note duration/tuplet
		}
		if flags&0x10 != 0 {
			dyn = int(p.r.i8())
		}
		if flags&0x20 != 0 {
			fret = int(p.r.i8())
		}
		if flags&0x80 != 0 {
			p.r.skip(2)
		}
		if flags&0x08 != 0 {
			p.readNoteEffects()
		}
	}
	if p.r.err != nil {
		return
	}
	key := tr.Tuning[stringNo-1] + fret
	if tr.IsDrum && tr.Tuning[stringNo-1] == 0 {
		key = fret
	}
	if key < 0 || key > 127 {
		return
	}
	vel := 95
	if dyn >= 0 {
		vel = 15 + 16*(dyn-1)
	}
	if vel < 1 {
		vel = 1
	}
	if vel > 127 {
		vel = 127
	}
	pk := [2]int{stringNo, voice}
	if noteType == 2 { // tie: extend the previous note on this string
		if idx, ok := tr.lastByVoice[pk]; ok && idx < len(tr.Notes) && tr.Notes[idx].Start+tr.Notes[idx].Dur == start {
			tr.Notes[idx].Dur += dur
		}
		return
	}
	if noteType == 3 { // dead note
		return
	}
	tr.Notes = append(tr.Notes, Note{Start: start, Dur: dur, Key: key, Vel: vel})
	tr.lastByVoice[pk] = len(tr.Notes) - 1
}

func (p *parser) readNoteEffects() {
	if p.family == 3 {
		flags := p.r.u8()
		if flags&0x01 != 0 {
			p.readBend()
		}
		if flags&0x10 != 0 {
			p.readGrace()
		}
		return
	}
	flags1 := p.r.i8()
	flags2 := p.r.i8()
	if flags1&0x01 != 0 {
		p.readBend()
	}
	if flags1&0x10 != 0 {
		p.readGrace()
	}
	if flags2&0x04 != 0 {
		p.r.skip(1) // tremolo picking
	}
	if flags2&0x08 != 0 {
		p.readSlides()
	}
	if flags2&0x10 != 0 {
		p.readHarmonic()
	}
	if flags2&0x20 != 0 {
		p.r.skip(2) // trill fret + period
	}
}

func (p *parser) readGrace() {
	if p.family == 5 {
		p.r.skip(5) // fret, velocity, transition, duration, flags
		return
	}
	p.r.skip(4) // fret, velocity, duration, transition
}

func (p *parser) readSlides() {
	if p.family == 5 {
		p.r.skip(1) // bitmask
		return
	}
	p.r.skip(1) // slide type
}

func (p *parser) readHarmonic() {
	kind := p.r.i8()
	if p.family == 5 {
		switch kind {
		case 2:
			p.r.skip(3) // semitone, accidental, octave
		case 3:
			p.r.skip(1) // fret
		}
	}
}

func (p *parser) build(path string) (*song.Song, error) {
	if len(p.tracks) == 0 {
		return nil, fmt.Errorf("guitar pro: no tracks")
	}
	name := strings.TrimSpace(p.title)
	if name == "" {
		name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	if name == "" {
		name = "Guitar Pro"
	}
	s := &song.Song{Name: name, Path: path, PPQ: 960}
	if p.tempo > 0 {
		s.Tempo = append(s.Tempo, song.TempoChange{Tick: 0, BPM: float64(p.tempo)})
	}
	for _, tc := range p.tempos {
		if tc.tick == 0 {
			if len(s.Tempo) > 0 {
				s.Tempo[0] = song.TempoChange{Tick: 0, BPM: tc.bpm}
			} else {
				s.Tempo = append(s.Tempo, song.TempoChange{Tick: 0, BPM: tc.bpm})
			}
			continue
		}
		s.Tempo = append(s.Tempo, song.TempoChange{Tick: tc.tick, BPM: tc.bpm})
	}
	if len(s.Tempo) == 0 {
		s.Tempo = []song.TempoChange{{Tick: 0, BPM: 120}}
	}
	if len(p.sigs) == 0 {
		p.sigs = []timeSig{{tick: 0, num: 4, den: 4}}
	}
	for _, ts := range p.sigs {
		s.Sigs = append(s.Sigs, song.TimeSigChange{Tick: ts.tick, Num: ts.num, Den: ts.den})
	}
	id := int64(1)
	for ti, tr := range p.tracks {
		st := song.Track{
			Index:   ti,
			Name:    tr.Name,
			Program: tr.Program,
			Channel: tr.Channel,
			IsDrum:  tr.IsDrum,
			Visible: tr.Visible,
			Muted:   tr.Muted,
			Solo:    tr.Solo,
			Color:   tr.Color,
			Notes:   make([]song.Note, 0, len(tr.Notes)),
		}
		if st.Color == "" {
			st.Color = "#7a8aa0"
		}
		for _, n := range tr.Notes {
			st.Notes = append(st.Notes, song.Note{
				ID:       id,
				Key:      n.Key,
				Tick:     n.Start,
				DurTicks: n.Dur,
				Velocity: n.Vel,
				Track:    ti,
			})
			id++
		}
		s.Tracks = append(s.Tracks, st)
	}
	raw, err := smf.Write(s)
	if err != nil {
		return nil, fmt.Errorf("guitar pro: encode: %w", err)
	}
	parsed, err := smf.Parse(raw, path)
	if err != nil {
		return nil, fmt.Errorf("guitar pro: normalize: %w", err)
	}
	parsed.Name = name
	return parsed, nil
}
