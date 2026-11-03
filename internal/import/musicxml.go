// Package importer converts foreign score formats to the song model.
package importer

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/gomidi/midi/v2"
	gomidi "gitlab.com/gomidi/midi/v2/smf"

	"github.com/akhenakh/gmid/internal/import/gp"
	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

// Load opens any supported file and returns a parsed song.
func Load(path string) (*song.Song, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mid", ".midi", ".kar":
		return smf.ReadFile(path)
	case ".musicxml", ".xml", ".mxl":
		return loadMusicXML(path)
	case ".gp3", ".gp4", ".gp5":
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		s, err := gp.Parse(raw, path)
		if err == nil {
			return s, nil
		}
		// Fall back to MuseScore when the native reader cannot handle the file
		// (for example a version variant it does not know yet).
		if ext, extErr := externalScoreConvert(path); extErr == nil {
			return ext, nil
		}
		return nil, fmt.Errorf("guitar pro: %w", err)
	case ".gpx", ".gp", ".cap", ".capx":
		if s, err := externalScoreConvert(path); err == nil {
			return s, nil
		}
		return nil, fmt.Errorf("guitar pro and capella import need MuseScore installed (mscore command)")
	default:
		return nil, fmt.Errorf("unsupported file type: %s", filepath.Ext(path))
	}
}

// SupportedExtensions returns the exts offered by the file browser.
func SupportedExtensions() []string {
	return []string{"mid", "midi", "kar", "musicxml", "xml", "mxl", "gp", "gp3", "gp4", "gp5", "gpx", "cap", "capx"}
}

type mxScore struct {
	Work struct {
		Title string `xml:"work-title"`
	} `xml:"work"`
	MovementTitle string        `xml:"movement-title"`
	PartList      []mxScorePart `xml:"part-list>score-part"`
	Parts         []mxPart      `xml:"part"`
}

type mxScorePart struct {
	ID       string   `xml:"id,attr"`
	PartName string   `xml:"part-name"`
	MIDI     []mxMIDI `xml:"midi-instrument"`
}

type mxMIDI struct {
	Channel int `xml:"midi-channel"`
	Program int `xml:"midi-program"`
}

type mxPart struct {
	ID       string      `xml:"id,attr"`
	Measures []mxMeasure `xml:"measure"`
}

type mxMeasure struct {
	Number     string        `xml:"number,attr"`
	Attributes *mxAttributes `xml:"attributes"`
	Notes      []mxNote      `xml:"note"`
	Directions []mxDirection `xml:"direction"`
	Backup     []mxDuration  `xml:"backup"`
	Forward    []mxDuration  `xml:"forward"`
}

type mxAttributes struct {
	Divisions *int         `xml:"divisions"`
	Key       *mxKey       `xml:"key"`
	Time      *mxTime      `xml:"time"`
	Staves    int          `xml:"staves"`
	Transpose *mxTranspose `xml:"transpose"`
}

type mxKey struct {
	Fifths int    `xml:"fifths"`
	Mode   string `xml:"mode"`
}

type mxTime struct {
	Beats    string `xml:"beats"`
	BeatType string `xml:"beat-type"`
}

type mxTranspose struct {
	Chromatic int `xml:"chromatic"`
}

type mxDirection struct {
	Sound     *mxSound     `xml:"sound"`
	Metronome *mxMetronome `xml:"metronome"`
	Offset    int          `xml:"offset"`
}

type mxSound struct {
	Tempo float64 `xml:"tempo,attr"`
}

type mxMetronome struct {
	PerMinute string `xml:"per-minute"`
}

type mxDuration struct {
	Duration int `xml:"duration"`
}

type mxNote struct {
	Chord    bool      `xml:"chord"`
	Rest     *struct{} `xml:"rest"`
	Pitch    *mxPitch  `xml:"pitch"`
	Duration int       `xml:"duration"`
	Voice    string    `xml:"voice"`
	Staff    int       `xml:"staff"`
	Tie      []mxTie   `xml:"tie"`
	Grace    *struct{} `xml:"grace"`
}

type mxPitch struct {
	Step   string `xml:"step"`
	Alter  int    `xml:"alter"`
	Octave int    `xml:"octave"`
}

type mxTie struct {
	Type string `xml:"type,attr"`
}

type event struct {
	tick int64
	on   bool
	key  int
	vel  int
}

type builtTrack struct {
	name    string
	channel int
	program int
	notes   []event
}

// loadMusicXML parses MusicXML (plain or compressed .mxl) and renders it as an
// in-memory SMF, then reuses the main parser for timing and metadata.
func loadMusicXML(path string) (*song.Song, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(filepath.Ext(path), ".mxl") {
		data, err = unzipMusicXML(data)
		if err != nil {
			return nil, err
		}
	}
	return parseMusicXML(data, path)
}

// parseMusicXML parses an uncompressed MusicXML document. Kept separate from
// loadMusicXML so fuzzing can feed it bytes without touching the filesystem.
func parseMusicXML(data []byte, path string) (*song.Song, error) {
	var score mxScore
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	if err := dec.Decode(&score); err != nil {
		return nil, fmt.Errorf("musicxml: %w", err)
	}
	if len(score.Parts) == 0 {
		return nil, fmt.Errorf("musicxml: no parts")
	}
	title := score.MovementTitle
	if title == "" {
		title = score.Work.Title
	}
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	names := map[string]string{}
	programs := map[string]int{}
	channels := map[string]int{}
	for _, p := range score.PartList {
		names[p.ID] = p.PartName
		if len(p.MIDI) > 0 {
			programs[p.ID] = p.MIDI[0].Program - 1
			channels[p.ID] = p.MIDI[0].Channel - 1
		}
	}

	const ppq = 960
	f := gomidi.NewSMF1()
	f.TimeFormat = gomidi.MetricTicks(ppq)
	conductor := gomidi.Track{}
	type metaEvent struct {
		tick int64
		msg  gomidi.Message
	}
	var metas []metaEvent
	metas = append(metas, metaEvent{0, gomidi.MetaTrackSequenceName(title)})

	tracks := map[string]*builtTrack{}
	order := []string{}
	divisions := 1
	currentKey := 0
	minor := false
	transpose := 0
	for _, part := range score.Parts {
		partName := names[part.ID]
		if partName == "" {
			partName = part.ID
		}
		tick := int64(0)
		lastStart := int64(0)
		for _, measure := range part.Measures {
			if measure.Attributes != nil {
				a := measure.Attributes
				if a.Divisions != nil && *a.Divisions > 0 {
					divisions = *a.Divisions
				}
				if a.Key != nil {
					currentKey, minor = keyFromFifths(a.Key.Fifths, a.Key.Mode)
					metas = append(metas, metaEvent{tick, gomidi.MetaText(fmt.Sprintf("GMID_KEY:%d:%s", currentKey, minorWord(minor)))})
				}
				if a.Time != nil {
					num := atoiSafe(a.Time.Beats)
					den := atoiSafe(a.Time.BeatType)
					if num > 0 && den > 0 {
						metas = append(metas, metaEvent{tick, gomidi.MetaTimeSig(uint8(num), uint8(den), 24, 8)})
					}
				}
				transpose = 0
				if a.Transpose != nil {
					transpose = a.Transpose.Chromatic
				}
			}
			for _, dir := range measure.Directions {
				if dir.Sound != nil && dir.Sound.Tempo > 0 {
					metas = append(metas, metaEvent{tick, gomidi.MetaTempo(dir.Sound.Tempo)})
				} else if dir.Metronome != nil {
					if bpm := atoiSafe(dir.Metronome.PerMinute); bpm > 0 {
						metas = append(metas, metaEvent{tick, gomidi.MetaTempo(float64(bpm))})
					}
				}
			}
			for _, n := range measure.Notes {
				staff := n.Staff
				if staff == 0 {
					staff = 1
				}
				tkey := fmt.Sprintf("%s/%d", part.ID, staff)
				tr := tracks[tkey]
				if tr == nil {
					ch := channels[part.ID]
					if ch == 0 {
						ch = len(order) % 15
					}
					tr = &builtTrack{name: partName, channel: ch, program: programs[part.ID]}
					if staff > 1 {
						tr.name = partName + " · staff " + strconv.Itoa(staff)
					}
					tracks[tkey] = tr
					order = append(order, tkey)
				}
				dur := int64(n.Duration)
				if n.Grace != nil || dur <= 0 {
					dur = maxI64(1, int64(ppq/16))
				}
				dur = dur * ppq / int64(maxIntXML(divisions, 1))
				start := tick
				if n.Chord {
					start = lastStart
				} else {
					lastStart = tick
				}
				if n.Pitch != nil && n.Rest == nil {
					key := midiFromPitch(*n.Pitch) + transpose
					tr.notes = append(tr.notes,
						event{start, true, key, 80},
						event{start + dur, false, key, 0},
					)
				}
				if !n.Chord {
					tick += dur
				}
			}
			for _, b := range measure.Backup {
				tick -= int64(b.Duration) * ppq / int64(maxIntXML(divisions, 1))
				if tick < 0 {
					tick = 0
				}
			}
			for _, fw := range measure.Forward {
				tick += int64(fw.Duration) * ppq / int64(maxIntXML(divisions, 1))
			}
		}
	}
	sort.SliceStable(metas, func(i, j int) bool { return metas[i].tick < metas[j].tick })
	var last int64
	for _, m := range metas {
		d := m.tick - last
		if d < 0 {
			d = 0
		}
		conductor.Add(uint32(d), []byte(m.msg))
		last = m.tick
	}
	conductor.Close(0)
	if err := f.Add(conductor); err != nil {
		return nil, err
	}
	for _, key := range order {
		tr := tracks[key]
		sort.SliceStable(tr.notes, func(i, j int) bool {
			if tr.notes[i].tick != tr.notes[j].tick {
				return tr.notes[i].tick < tr.notes[j].tick
			}
			if tr.notes[i].on != tr.notes[j].on {
				return !tr.notes[i].on
			}
			return tr.notes[i].key < tr.notes[j].key
		})
		gt := gomidi.Track{}
		gt.Add(0, gomidi.MetaTrackSequenceName(tr.name))
		gt.Add(0, []byte(midi.ProgramChange(uint8(clamp(tr.channel, 0, 15)), uint8(clamp(tr.program, 0, 127)))))
		var lt int64
		for _, e := range tr.notes {
			d := e.tick - lt
			if d < 0 {
				d = 0
			}
			var msg midi.Message
			if e.on {
				msg = midi.NoteOn(uint8(clamp(tr.channel, 0, 15)), uint8(clamp(e.key, 0, 127)), uint8(clamp(e.vel, 1, 127)))
			} else {
				msg = midi.NoteOff(uint8(clamp(tr.channel, 0, 15)), uint8(clamp(e.key, 0, 127)))
			}
			gt.Add(uint32(d), []byte(msg))
			lt = e.tick
		}
		gt.Close(0)
		if err := f.Add(gt); err != nil {
			return nil, err
		}
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, err
	}
	s, err := smf.Parse(buf.Bytes(), path)
	if err != nil {
		return nil, err
	}
	s.Name = title
	return s, nil
}

const (
	maxMusicXMLContainer = 1 << 20  // META-INF/container.xml
	maxMusicXMLScore     = 64 << 20 // score document
)

// readLimited reads r but rejects anything larger than limit, so a small
// compressed .mxl cannot expand without bound.
func readLimited(r io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("mxl: entry exceeds %d bytes", limit)
	}
	return body, nil
}

func unzipMusicXML(data []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	var root string
	for _, f := range zr.File {
		if strings.EqualFold(f.Name, "META-INF/container.xml") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			body, err := readLimited(rc, maxMusicXMLContainer)
			_ = rc.Close()
			if err != nil {
				return nil, err
			}
			var c struct {
				Rootfiles []struct {
					FullPath string `xml:"full-path,attr"`
				} `xml:"rootfiles>rootfile"`
			}
			if xml.Unmarshal(body, &c) == nil && len(c.Rootfiles) > 0 {
				root = c.Rootfiles[0].FullPath
			}
		}
	}
	if root == "" {
		for _, f := range zr.File {
			if strings.HasSuffix(strings.ToLower(f.Name), ".xml") && !strings.HasPrefix(f.Name, "META-INF/") {
				root = f.Name
				break
			}
		}
	}
	if root == "" {
		return nil, fmt.Errorf("mxl: no score xml")
	}
	for _, f := range zr.File {
		if f.Name != root {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return readLimited(rc, maxMusicXMLScore)
	}
	return nil, fmt.Errorf("mxl: rootfile missing")
}

func midiFromPitch(p mxPitch) int {
	step := 0
	switch strings.ToUpper(p.Step) {
	case "C":
		step = 0
	case "D":
		step = 2
	case "E":
		step = 4
	case "F":
		step = 5
	case "G":
		step = 7
	case "A":
		step = 9
	case "B":
		step = 11
	}
	return (p.Octave+1)*12 + step + p.Alter
}

func keyFromFifths(fifths int, mode string) (int, bool) {
	sem := fifths * 7
	sem = ((sem % 12) + 12) % 12
	minor := strings.EqualFold(mode, "minor")
	if minor {
		sem = ((sem-3)%12 + 12) % 12
	}
	return sem, minor
}

func minorWord(minor bool) string {
	if minor {
		return "minor"
	}
	return "major"
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxIntXML(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxI64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
