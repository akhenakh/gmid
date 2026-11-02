package smf

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/smf"

	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/song"
)

type writeEvent struct {
	tick    int64
	kind    int // 0 = note off, 1 = meta/ctrl, 2 = note on
	message smf.Message
}

// Write rebuilds a Standard MIDI File (format 1) from the song model.
func Write(s *song.Song) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("nil song")
	}
	ppq := s.PPQ
	if ppq <= 0 {
		ppq = 960
	}
	f := smf.NewSMF1()
	f.TimeFormat = smf.MetricTicks(ppq)

	conductor := smf.Track{}
	var metaEvents []writeEvent
	metaEvents = append(metaEvents, writeEvent{0, 1, smf.MetaTrackSequenceName(s.Name)})
	for _, t := range s.Tempo {
		metaEvents = append(metaEvents, writeEvent{t.Tick, 1, smf.MetaTempo(t.BPM)})
	}
	for _, ts := range s.Sigs {
		metaEvents = append(metaEvents, writeEvent{ts.Tick, 1, smf.MetaTimeSig(uint8(ts.Num), uint8(ts.Den), 24, 8)})
	}
	for _, k := range s.Keys {
		metaEvents = append(metaEvents, writeEvent{k.Tick, 1, smf.MetaText(fmt.Sprintf("GMID_KEY:%d:%s", k.Semitone, minorWord(k.Minor)))})
	}
	for _, l := range s.Lyrics {
		metaEvents = append(metaEvents, writeEvent{l.Tick, 1, smf.MetaLyric(l.Text)})
	}
	outIndex := 0
	for i := range s.Tracks {
		if s.Tracks[i].Excluded {
			continue
		}
		t := &s.Tracks[i]
		metaEvents = append(metaEvents, writeEvent{0, 1, smf.MetaText(fmt.Sprintf("GMID_TRACK_NAME:%d:%s", outIndex, t.Name))})
		metaEvents = append(metaEvents, writeEvent{0, 1, smf.MetaText(fmt.Sprintf("GMID_TRACK_COLOR:%d:%s", outIndex, t.Color))})
		visible := 0
		if t.Visible {
			visible = 1
		}
		keyboard := 0
		if t.OnKeyboard {
			keyboard = 1
		}
		metaEvents = append(metaEvents, writeEvent{0, 1, smf.MetaText(fmt.Sprintf("GMID_TRACK_VISIBLE:%d:%d", outIndex, visible))})
		metaEvents = append(metaEvents, writeEvent{0, 1, smf.MetaText(fmt.Sprintf("GMID_TRACK_KEYBOARD:%d:%d", outIndex, keyboard))})
		outIndex++
	}
	for _, h := range s.Meta.History {
		metaEvents = append(metaEvents, writeEvent{0, 1, smf.MetaText(fmt.Sprintf("GMID_HISTORY:%d:%s:%s", h.Timestamp, h.Kind, h.Summary))})
	}
	sort.SliceStable(metaEvents, func(i, j int) bool {
		if metaEvents[i].tick != metaEvents[j].tick {
			return metaEvents[i].tick < metaEvents[j].tick
		}
		return metaEvents[i].kind < metaEvents[j].kind
	})
	var last int64
	for _, ev := range metaEvents {
		d := ev.tick - last
		if d < 0 {
			d = 0
		}
		conductor.Add(uint32(d), []byte(ev.message))
		last = ev.tick
	}
	conductor.Close(0)
	if err := f.Add(conductor); err != nil {
		return nil, err
	}

	// Channel automation is written once, on the first track using the channel.
	writtenChannel := map[int]bool{}
	for i := range s.Tracks {
		if s.Tracks[i].Excluded {
			continue
		}
		tr := staticTrack(s, i, writtenChannel)
		if err := f.Add(tr); err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func staticTrack(s *song.Song, index int, writtenChannel map[int]bool) smf.Track {
	t := &s.Tracks[index]
	var evs []writeEvent
	evs = append(evs, writeEvent{0, 1, smf.MetaTrackSequenceName(t.Name)})
	if !t.IsDrum {
		bank := t.BankMSB*128 + t.BankLSB
		evs = append(evs,
			writeEvent{0, 1, smf.Message(midi.ControlChange(uint8(t.Channel), 0, uint8(bank/128)))},
			writeEvent{0, 1, smf.Message(midi.ControlChange(uint8(t.Channel), 32, uint8(bank%128)))},
			writeEvent{0, 1, smf.Message(midi.ProgramChange(uint8(t.Channel), uint8(clampProgram(t.Program))))},
		)
	}
	if !writtenChannel[t.Channel] {
		writtenChannel[t.Channel] = true
		if c := s.ChannelAt(t.Channel); c != nil {
			for _, cc := range c.CC {
				evs = append(evs, writeEvent{cc.Tick, 1, smf.Message(midi.ControlChange(uint8(t.Channel), uint8(cc.CC), uint8(cc.Val)))})
			}
			for _, se := range c.Sustain {
				v := uint8(0)
				if se.Down {
					v = 127
				}
				evs = append(evs, writeEvent{se.Tick, 1, smf.Message(midi.ControlChange(uint8(t.Channel), 64, v))})
			}
			for _, pe := range c.Programs {
				bank := pe.Bank
				evs = append(evs,
					writeEvent{pe.Tick, 1, smf.Message(midi.ControlChange(uint8(t.Channel), 0, uint8(bank/128)))},
					writeEvent{pe.Tick, 1, smf.Message(midi.ControlChange(uint8(t.Channel), 32, uint8(bank%128)))},
					writeEvent{pe.Tick, 1, smf.Message(midi.ProgramChange(uint8(t.Channel), uint8(clampProgram(pe.Patch))))},
				)
			}
		}
	}
	for i := range t.Notes {
		n := &t.Notes[i]
		key := uint8(clampProgram(n.Key))
		evs = append(evs,
			writeEvent{n.Tick, 2, smf.Message(midi.NoteOn(uint8(t.Channel), key, uint8(clampVelocity(n.Velocity))))},
			writeEvent{n.Tick + n.DurTicks, 0, smf.Message(midi.NoteOff(uint8(t.Channel), key))},
		)
	}
	sort.SliceStable(evs, func(i, j int) bool {
		if evs[i].tick != evs[j].tick {
			return evs[i].tick < evs[j].tick
		}
		return evs[i].kind < evs[j].kind
	})
	tr := smf.Track{}
	var last int64
	for _, ev := range evs {
		d := ev.tick - last
		if d < 0 {
			d = 0
		}
		tr.Add(uint32(d), []byte(ev.message))
		last = ev.tick
	}
	tr.Close(0)
	return tr
}

// SaveVersion writes a new `<name>_GMID_vN.mid` next to the library's gmid
// folder and records a history entry. The original file is never touched.
func SaveVersion(s *song.Song, library, kind, summary string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("nil song")
	}
	now := time.Now().Unix()
	s.Meta.History = append(s.Meta.History, song.HistoryEvent{
		Timestamp: now, Kind: kind, Summary: summary,
	})
	dir := config.GmidDir(library)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	base := strings.TrimSuffix(filepath.Base(s.Path), filepath.Ext(s.Path))
	var name string
	for v := 1; ; v++ {
		name = fmt.Sprintf("%s_GMID_v%d.mid", base, v)
		if _, err := os.Stat(filepath.Join(dir, name)); os.IsNotExist(err) {
			break
		}
	}
	data, err := Write(s)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, name)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	return path, nil
}

func clampProgram(v int) int {
	if v < 0 {
		return 0
	}
	if v > 127 {
		return 127
	}
	return v
}

func clampVelocity(v int) int {
	if v < 1 {
		return 1
	}
	if v > 127 {
		return 127
	}
	return v
}

func minorWord(minor bool) string {
	if minor {
		return "minor"
	}
	return "major"
}

// ParseVersionNumber extracts N from a `_GMID_vN.mid` filename.
func ParseVersionNumber(name string) (int, bool) {
	lower := strings.ToLower(name)
	i := strings.LastIndex(lower, "_gmid_v")
	if i < 0 {
		return 0, false
	}
	rest := lower[i+len("_gmid_v"):]
	rest = strings.TrimSuffix(rest, ".mid")
	n, err := strconv.Atoi(rest)
	return n, err == nil
}
