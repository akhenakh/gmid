package smf

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/v2/smf"

	"github.com/akhenakh/gmid/internal/song"
	"github.com/akhenakh/gmid/internal/theory"
)

type pendingNote struct {
	tick int64
	vel  int
}

type trackBuild struct {
	name     string
	channel  int
	hasChan  bool
	program  int
	bankMSB  int
	bankLSB  int
	programs []song.ProgramEvent
	notes    []song.Note
	pending  map[[2]int][]pendingNote
	lastTick int64
}

type rawTempo struct {
	tick int64
	bpm  float64
}

type rawSig struct {
	tick     int64
	num, den int
}

type rawKey struct {
	tick     int64
	semitone int
	minor    bool
}

// ReadFile parses a Standard MIDI File into the song model.
func ReadFile(path string) (*song.Song, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s, err := Parse(raw, path)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// readSMF isolates the third-party parser. gomidi panics on some malformed
// SMPTE headers (it assumes MetricTicks while finishing tempo changes), so turn
// any panic into an error: a corrupt file must never crash the app.
func readSMF(raw []byte) (f *smf.SMF, err error) {
	defer func() {
		if r := recover(); r != nil {
			f = nil
			err = fmt.Errorf("malformed smf: %v", r)
		}
	}()
	return smf.ReadFrom(bytes.NewReader(raw))
}

// Parse parses raw SMF bytes. Name is derived from path when empty.
func Parse(raw []byte, path string) (*song.Song, error) {
	f, err := readSMF(raw)
	if err != nil {
		return nil, fmt.Errorf("read midi: %w", err)
	}
	name := path
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	s := &song.Song{
		Name:     name,
		Path:     path,
		Raw:      raw,
		Identity: song.ComputeIdentity(raw),
		Tracks:   nil,
	}

	switch tf := f.TimeFormat.(type) {
	case smf.MetricTicks:
		s.PPQ = int(tf)
		if s.PPQ == 0 {
			s.PPQ = 960
		}
	case smf.TimeCode:
		// SMPTE: ticks are frames/subframes, tempo has no effect.
		s.PPQ = 0
		fps := float64(tf.FramesPerSecond)
		if fps == 0 {
			fps = 25
		}
		sub := float64(tf.SubFrames)
		if sub == 0 {
			sub = 4
		}
		s.TickSec = 1 / (fps * sub)
	default:
		s.PPQ = 960
	}

	var tempos []rawTempo
	var sigs []rawSig
	var keys []rawKey
	var lyrics []song.Lyric

	builds := make([]trackBuild, 0, len(f.Tracks))
	channelCC := map[int][]song.CCEvent{}
	channelSustain := map[int][]song.SustainEvent{}
	channelPrograms := map[int][]song.ProgramEvent{}
	channelBank := map[int][2]int{}

	maxSec := 0.0

	for _, tr := range f.Tracks {
		b := trackBuild{channel: 0, program: 0, pending: map[[2]int][]pendingNote{}}
		abs := int64(0)
		for _, ev := range tr {
			abs += int64(ev.Delta)
			b.lastTick = abs
			m := ev.Message

			var txt string
			if m.GetMetaTrackName(&txt) {
				if b.name == "" {
					b.name = txt
				}
				continue
			}
			var bpm float64
			if m.GetMetaTempo(&bpm) {
				if bpm > 0 {
					tempos = append(tempos, rawTempo{tick: abs, bpm: bpm})
				}
				continue
			}
			var num, den uint8
			if m.GetMetaTimeSig(&num, &den, new(uint8), new(uint8)) {
				if num > 0 && den > 0 {
					sigs = append(sigs, rawSig{tick: abs, num: int(num), den: int(den)})
				}
				continue
			}
			var sf uint8
			var major, flat bool
			if m.GetMetaKeySig(&sf, new(uint8), &major, &flat) {
				_ = flat
				sem := int(int8(sf)) * 7
				sem = ((sem % 12) + 12) % 12
				minor := !major
				if minor {
					sem = ((sem-3)%12 + 12) % 12
				}
				keys = append(keys, rawKey{tick: abs, semitone: sem, minor: minor})
				continue
			}
			if m.GetMetaLyric(&txt) {
				lyrics = append(lyrics, song.Lyric{Tick: abs, Text: txt})
				continue
			}
			if m.GetMetaText(&txt) {
				parseMetaText(s, txt)
				continue
			}

			var ch, key, vel uint8
			if m.GetNoteOn(&ch, &key, &vel) {
				b.hasChan = true
				b.channel = int(ch)
				if vel == 0 {
					closeNote(&b, abs, int(ch), int(key))
					continue
				}
				k := [2]int{int(ch), int(key)}
				b.pending[k] = append(b.pending[k], pendingNote{tick: abs, vel: int(vel)})
				continue
			}
			if m.GetNoteOff(&ch, &key, &vel) {
				b.hasChan = true
				b.channel = int(ch)
				closeNote(&b, abs, int(ch), int(key))
				continue
			}
			var prog uint8
			if m.GetProgramChange(&ch, &prog) {
				b.hasChan = true
				b.channel = int(ch)
				bank := channelBank[int(ch)]
				pe := song.ProgramEvent{Tick: abs, Bank: bank[0]*128 + bank[1], Patch: int(prog)}
				channelPrograms[int(ch)] = append(channelPrograms[int(ch)], pe)
				b.programs = append(b.programs, pe)
				b.program = int(prog)
				continue
			}
			var cc, val uint8
			if m.GetControlChange(&ch, &cc, &val) {
				b.hasChan = true
				b.channel = int(ch)
				switch int(cc) {
				case 0, 32:
					bank := channelBank[int(ch)]
					if cc == 0 {
						bank[0] = int(val)
						b.bankMSB = int(val)
					} else {
						bank[1] = int(val)
						b.bankLSB = int(val)
					}
					channelBank[int(ch)] = bank
				case 64:
					channelSustain[int(ch)] = append(channelSustain[int(ch)], song.SustainEvent{Tick: abs, Down: val >= 64})
				default:
					if cc == 7 || cc == 10 || cc == 11 || cc == 91 || cc == 93 {
						channelCC[int(ch)] = append(channelCC[int(ch)], song.CCEvent{Tick: abs, CC: int(cc), Val: int(val)})
					}
				}
				continue
			}
		}
		// Close dangling notes at the last tick.
		for k, stack := range b.pending {
			for _, pn := range stack {
				dur := b.lastTick - pn.tick
				if dur <= 0 {
					dur = 1
				}
				n := song.Note{Key: k[1], Tick: pn.tick, DurTicks: dur, Velocity: pn.vel, Track: len(builds)}
				b.notes = append(b.notes, n)
			}
			delete(b.pending, k)
		}
		builds = append(builds, b)
	}

	// Tempo map with a sane default and a tick-0 anchor. SMPTE files have a
	// fixed tick duration and ignore tempo meta events.
	if s.PPQ > 0 {
		if len(tempos) == 0 {
			tempos = append(tempos, rawTempo{tick: 0, bpm: 120})
		}
		sort.SliceStable(tempos, func(i, j int) bool { return tempos[i].tick < tempos[j].tick })
		dedup := tempos[:0]
		for _, t := range tempos {
			if len(dedup) > 0 && dedup[len(dedup)-1].tick == t.tick {
				dedup[len(dedup)-1] = t
				continue
			}
			dedup = append(dedup, t)
		}
		tempos = dedup
		if tempos[0].tick > 0 {
			tempos = append([]rawTempo{{tick: 0, bpm: 120}}, tempos...)
		}
		prevTick := int64(0)
		prevSec := 0.0
		for i, t := range tempos {
			if i > 0 {
				prevSec += float64(t.tick-prevTick) / float64(s.PPQ) * (60.0 / tempos[i-1].bpm)
				prevTick = t.tick
			}
			s.Tempo = append(s.Tempo, song.TempoChange{Tick: t.tick, BPM: t.bpm, Sec: prevSec})
		}
	} else {
		s.Tempo = []song.TempoChange{{Tick: 0, BPM: 120}}
	}

	// Time signatures, anchored at tick 0.
	if len(sigs) == 0 {
		sigs = append(sigs, rawSig{tick: 0, num: 4, den: 4})
	}
	sort.SliceStable(sigs, func(i, j int) bool { return sigs[i].tick < sigs[j].tick })
	seenSig := map[int64]bool{}
	for _, sg := range sigs {
		if seenSig[sg.tick] {
			// last change at a tick wins
			for i := len(s.Sigs) - 1; i >= 0; i-- {
				if s.Sigs[i].Tick == sg.tick {
					s.Sigs[i].Num, s.Sigs[i].Den = sg.num, sg.den
					break
				}
			}
			continue
		}
		seenSig[sg.tick] = true
		s.Sigs = append(s.Sigs, song.TimeSigChange{Tick: sg.tick, Num: sg.num, Den: sg.den, Sec: s.TickToSec(sg.tick)})
	}
	if s.Sigs[0].Tick > 0 {
		s.Sigs = append([]song.TimeSigChange{{Tick: 0, Num: 4, Den: 4}}, s.Sigs...)
	}

	sort.SliceStable(keys, func(i, j int) bool { return keys[i].tick < keys[j].tick })
	seenKey := map[int64]bool{}
	for _, k := range keys {
		if seenKey[k.tick] {
			continue
		}
		seenKey[k.tick] = true
		s.Keys = append(s.Keys, song.KeyChange{Tick: k.tick, Semitone: k.semitone, Minor: k.minor, Sec: s.TickToSec(k.tick)})
	}

	// Build tracks with notes.
	for i := range builds {
		b := &builds[i]
		if len(b.notes) == 0 {
			continue
		}
		isDrum := b.channel == 9
		firstTick := b.notes[0].Tick
		for j := range b.notes {
			if b.notes[j].Tick < firstTick {
				firstTick = b.notes[j].Tick
			}
		}
		sort.SliceStable(b.programs, func(i, j int) bool { return b.programs[i].Tick < b.programs[j].Tick })
		program, bankMSB, bankLSB := 0, b.bankMSB, b.bankLSB
		for _, pe := range b.programs {
			if pe.Tick <= firstTick {
				program = pe.Patch
				bankMSB = pe.Bank / 128
				bankLSB = pe.Bank % 128
			}
		}
		if isDrum {
			program = -1
			bankMSB, bankLSB = 0, 0
		}
		t := song.Track{
			Index:      len(s.Tracks),
			Name:       b.name,
			GMName:     theory.GMName(program),
			Program:    program,
			BankMSB:    bankMSB,
			BankLSB:    bankLSB,
			Channel:    b.channel,
			IsDrum:     isDrum,
			Group:      theory.GMGroupOf(program, isDrum),
			Visible:    true,
			OnKeyboard: true,
			Notes:      b.notes,
		}
		if t.Name == "" {
			t.Name = t.GMName
		}
		for j := range t.Notes {
			n := &t.Notes[j]
			n.Track = t.Index
			n.StartSec = s.TickToSec(n.Tick)
			n.EndSec = s.TickToSec(n.Tick + n.DurTicks)
			if n.EndSec < n.StartSec {
				n.EndSec = n.StartSec
			}
			if n.EndSec > maxSec {
				maxSec = n.EndSec
			}
		}
		s.NoteCount += len(t.Notes)
		s.Tracks = append(s.Tracks, t)
	}

	// Stable note IDs for the editor.
	id := int64(1)
	for i := range s.Tracks {
		for j := range s.Tracks[i].Notes {
			s.Tracks[i].Notes[j].ID = id
			id++
		}
	}

	// Channel automation.
	chans := map[int]*song.Channel{}
	for ch, evs := range channelCC {
		c := ensureChannel(chans, ch)
		sort.SliceStable(evs, func(i, j int) bool { return evs[i].Tick < evs[j].Tick })
		for i := range evs {
			evs[i].Sec = s.TickToSec(evs[i].Tick)
		}
		c.CC = evs
	}
	for ch, evs := range channelSustain {
		c := ensureChannel(chans, ch)
		sort.SliceStable(evs, func(i, j int) bool { return evs[i].Tick < evs[j].Tick })
		// Collapse repeated states.
		out := evs[:0]
		for i := range evs {
			evs[i].Sec = s.TickToSec(evs[i].Tick)
			if len(out) > 0 && out[len(out)-1].Down == evs[i].Down {
				continue
			}
			out = append(out, evs[i])
		}
		c.Sustain = out
	}
	for ch, evs := range channelPrograms {
		c := ensureChannel(chans, ch)
		sort.SliceStable(evs, func(i, j int) bool { return evs[i].Tick < evs[j].Tick })
		for i := range evs {
			evs[i].Sec = s.TickToSec(evs[i].Tick)
		}
		c.Programs = evs
	}
	for _, c := range chans {
		c.InitialVolume = 100
		c.InitialPan = 64
		c.InitialReverb = 40
		c.InitialChorus = 0
		for _, e := range c.CC {
			switch e.CC {
			case 7:
				c.InitialVolume = e.Val
			case 10:
				c.InitialPan = e.Val
			case 91:
				c.InitialReverb = e.Val
			case 93:
				c.InitialChorus = e.Val
			}
		}
		s.Channels = append(s.Channels, *c)
	}
	sort.SliceStable(s.Channels, func(i, j int) bool { return s.Channels[i].Number < s.Channels[j].Number })

	if maxSec <= 0 {
		maxSec = s.TickToSec(s.SecToTick(0))
	}
	s.DurationSec = maxSec
	s.FirstNoteSec = s.FirstNonDrumNote()
	for i := range lyrics {
		lyrics[i].Sec = s.TickToSec(lyrics[i].Tick)
	}
	s.Lyrics = lyrics

	s.SortNotes()
	s.RebuildGrid()
	s.TempoSuspect = s.CheckTempoTrust() < 0.35
	applyMetaTracks(s)
	ApplyDefaultColors(s, theory.DefaultHandLH, theory.DefaultHandRH)
	if s.Meta.KeySemitone != nil {
		s.Keys = append(s.Keys, song.KeyChange{Tick: 0, Semitone: *s.Meta.KeySemitone, Minor: s.Meta.KeyMinor})
		sort.SliceStable(s.Keys, func(i, j int) bool { return s.Keys[i].Tick < s.Keys[j].Tick })
		// A metadata key overrides the file's own key at the same tick.
		out := s.Keys[:0]
		for _, k := range s.Keys {
			if len(out) > 0 && out[len(out)-1].Tick == k.Tick {
				out[len(out)-1] = k
				continue
			}
			out = append(out, k)
		}
		s.Keys = out
	}
	return s, nil
}

func ensureChannel(m map[int]*song.Channel, ch int) *song.Channel {
	if c := m[ch]; c != nil {
		return c
	}
	c := &song.Channel{Number: ch}
	m[ch] = c
	return c
}

func closeNote(b *trackBuild, tick int64, ch, key int) {
	k := [2]int{ch, key}
	stack := b.pending[k]
	if len(stack) == 0 {
		return
	}
	pn := stack[len(stack)-1]
	b.pending[k] = stack[:len(stack)-1]
	dur := tick - pn.tick
	if dur <= 0 {
		dur = 1
	}
	n := song.Note{ID: 0, Key: key, Tick: pn.tick, DurTicks: dur, Velocity: pn.vel, Track: 0}
	b.notes = append(b.notes, n)
}

func parseMetaText(s *song.Song, text string) {
	if s.Meta.TrackNames == nil {
		s.Meta.TrackNames = map[int]string{}
		s.Meta.TrackColors = map[int]string{}
		s.Meta.TrackVisible = map[int]bool{}
		s.Meta.TrackKeys = map[int]bool{}
		s.Meta.HandMap = map[int]string{}
	}
	prefix := func(p string) (string, bool) {
		if strings.HasPrefix(text, p) {
			return strings.TrimPrefix(text, p), true
		}
		return "", false
	}
	if rest, ok := prefix("GMID_KEY:"); ok {
		setKeyMeta(s, rest)
		return
	}
	if rest, ok := prefix("ORFEO_KEY:"); ok {
		setKeyMeta(s, rest)
		return
	}
	for _, p := range []string{"GMID_", "ORFEO_"} {
		if rest, ok := prefix(p + "TRACK_NAME:"); ok {
			idx, val, ok := splitIndex(rest)
			if ok {
				s.Meta.TrackNames[idx] = val
			}
			return
		}
		if rest, ok := prefix(p + "TRACK_COLOR:"); ok {
			idx, val, ok := splitIndex(rest)
			if ok && len(val) == 7 && val[0] == '#' {
				s.Meta.TrackColors[idx] = val
			}
			return
		}
		if rest, ok := prefix(p + "TRACK_VISIBLE:"); ok {
			idx, val, ok := splitIndex(rest)
			if ok {
				s.Meta.TrackVisible[idx] = val == "1"
			}
			return
		}
		if rest, ok := prefix(p + "TRACK_KEYBOARD:"); ok {
			idx, val, ok := splitIndex(rest)
			if ok {
				s.Meta.TrackKeys[idx] = val == "1"
			}
			return
		}
		if rest, ok := prefix(p + "HISTORY:"); ok {
			parts := strings.SplitN(rest, ":", 3)
			if len(parts) == 3 {
				ts, _ := strconv.ParseInt(parts[0], 10, 64)
				s.Meta.History = append(s.Meta.History, song.HistoryEvent{Timestamp: ts, Kind: parts[1], Summary: parts[2]})
			}
			return
		}
	}
}

func setKeyMeta(s *song.Song, rest string) {
	parts := strings.Split(rest, ":")
	if len(parts) < 2 {
		return
	}
	sem, err := strconv.Atoi(parts[0])
	if err != nil || sem < 0 || sem > 11 {
		return
	}
	s.Meta.KeySemitone = &sem
	s.Meta.KeyMinor = parts[1] == "minor"
}

func splitIndex(rest string) (int, string, bool) {
	i := strings.Index(rest, ":")
	if i < 0 {
		return 0, "", false
	}
	idx, err := strconv.Atoi(rest[:i])
	if err != nil {
		return 0, "", false
	}
	return idx, rest[i+1:], true
}

func applyMetaTracks(s *song.Song) {
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if n, ok := s.Meta.TrackNames[i]; ok && n != "" {
			t.Name = n
		}
		if c, ok := s.Meta.TrackColors[i]; ok {
			t.Color = c
		}
		if v, ok := s.Meta.TrackVisible[i]; ok {
			t.Visible = v
		}
		if v, ok := s.Meta.TrackKeys[i]; ok {
			t.OnKeyboard = v
		}
	}
}

// ApplyDefaultColors assigns deterministic track colours. Piano-family tracks
// take the hand pair; everything else picks the most distinct palette entry.
func ApplyDefaultColors(s *song.Song, lh, rh string) {
	pianoIndex := 0
	used := []string{}
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if t.Color != "" {
			used = append(used, t.Color)
			continue
		}
		var c string
		if theory.IsKeyboardGroup(t.Group) {
			c = theory.PianoFamilyColor(pianoIndex, lh, rh)
			pianoIndex++
		} else {
			c = theory.DistinctTrackColor(used, lh, rh)
		}
		t.Color = c
		used = append(used, c)
	}
}

// ApplyHandColors recolours an open song when the user changes the hand pair.
func ApplyHandColors(s *song.Song, lh, rh string) {
	if lh == "" {
		lh = theory.DefaultHandLH
	}
	if rh == "" {
		rh = theory.DefaultHandRH
	}
	pianoIndex := 0
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if !theory.IsKeyboardGroup(t.Group) {
			continue
		}
		old := t.Color
		t.Color = theory.FollowedHandColor(old, lh, rh)
		switch pianoIndex {
		case 0:
			t.Color = lh
		case 1:
			t.Color = rh
		}
		pianoIndex++
	}
}
