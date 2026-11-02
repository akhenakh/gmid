package song

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
)

// Song is a parsed MIDI file. Ticks are the editing truth; seconds are derived
// from the tempo map and used for playback and display.
type Song struct {
	Name string
	Path string

	PPQ     int
	TickSec float64 // SMPTE files: seconds per tick (PPQ is 0)
	Tempo   []TempoChange
	Sigs    []TimeSigChange
	Keys    []KeyChange

	Tracks   []Track
	Channels []Channel

	DurationSec  float64
	FirstNoteSec float64
	NoteCount    int

	Raw    []byte
	Lyrics []Lyric
	Meta   Meta

	Beats []float64
	Bars  []float64

	BeatFixed    bool
	BeatPeriod   float64
	BeatAnchor   float64
	BeatBpb      int
	TempoSuspect bool

	// LeadCut is the first bar worth showing; the file's own ticks are kept.
	Lead LeadCut

	Identity string
}

// TempoChange is an absolute tick and its microseconds per quarter note.
type TempoChange struct {
	Tick int64
	BPM  float64
	Sec  float64
}

// TimeSigChange is an absolute tick and its meter.
type TimeSigChange struct {
	Tick     int64
	Num, Den int
	Sec      float64
}

// KeyChange is a key signature change.
type KeyChange struct {
	Tick     int64
	Semitone int
	Minor    bool
	Sec      float64
}

// Track is one musical part.
type Track struct {
	Index      int
	Name       string
	GMName     string
	Program    int
	BankMSB    int
	BankLSB    int
	Channel    int
	IsDrum     bool
	Group      string
	Color      string
	Notes      []Note
	Muted      bool
	Solo       bool
	Visible    bool
	OnKeyboard bool

	// Reason is set for excluded tracks in the playback editor.
	Excluded bool
}

// Note is one MIDI note.
type Note struct {
	ID       int64
	Key      int
	Tick     int64
	DurTicks int64
	StartSec float64
	EndSec   float64
	Velocity int
	Hand     string // "L", "R", or ""
	Conf     float32
	Track    int
}

// CCEvent is a continuous controller move.
type CCEvent struct {
	Tick int64
	Sec  float64
	CC   int
	Val  int
}

// SustainEvent is a hold-pedal transition.
type SustainEvent struct {
	Tick int64
	Sec  float64
	Down bool
}

// ProgramEvent is a patch change.
type ProgramEvent struct {
	Tick  int64
	Sec   float64
	Bank  int
	Patch int
}

// Channel carries the per-channel automation shared by tracks on that channel.
type Channel struct {
	Number   int
	CC       []CCEvent
	Sustain  []SustainEvent
	Programs []ProgramEvent

	InitialVolume int
	InitialPan    int
	InitialReverb int
	InitialChorus int
}

// Lyric is a karaoke lyric event.
type Lyric struct {
	Tick int64
	Sec  float64
	Text string
}

// HistoryEvent is one entry from the embedded edit history.
type HistoryEvent struct {
	Timestamp int64
	Kind      string
	Summary   string
}

// Metadata is per-song state embedded in text meta events.
type Meta struct {
	KeySemitone  *int
	KeyMinor     bool
	TrackNames   map[int]string
	TrackColors  map[int]string
	TrackVisible map[int]bool
	TrackKeys    map[int]bool
	HandMap      map[int]string
	History      []HistoryEvent
}

// LeadCut marks the point where the visible song starts.
type LeadCut struct {
	Enabled bool
	Sec     float64
	Tick    int64
	Bar     int
}

// OriginSec is the playback origin (0 when no lead cut is applied).
func (s *Song) OriginSec() float64 {
	if s.Lead.Enabled {
		return s.Lead.Sec
	}
	return 0
}

// EffectiveDuration is the duration after the lead cut.
func (s *Song) EffectiveDuration() float64 {
	d := s.DurationSec - s.OriginSec()
	if d < 0 {
		return 0
	}
	return d
}

// TrackAt returns a track by index.
func (s *Song) TrackAt(i int) *Track {
	if i < 0 || i >= len(s.Tracks) {
		return nil
	}
	return &s.Tracks[i]
}

// ChannelAt returns a channel by MIDI number.
func (s *Song) ChannelAt(n int) *Channel {
	for i := range s.Channels {
		if s.Channels[i].Number == n {
			return &s.Channels[i]
		}
	}
	return nil
}

// TrackByName finds the first track with a matching name.
func (s *Song) TrackByName(name string) *Track {
	for i := range s.Tracks {
		if s.Tracks[i].Name == name {
			return &s.Tracks[i]
		}
	}
	return nil
}

// FirstNonDrumNote returns the earliest non-drum note start, or the earliest
// note when every track is drums.
func (s *Song) FirstNonDrumNote() float64 {
	best := math.MaxFloat64
	for i := range s.Tracks {
		t := &s.Tracks[i]
		if t.IsDrum || t.Excluded {
			continue
		}
		for j := range t.Notes {
			if t.Notes[j].StartSec < best {
				best = t.Notes[j].StartSec
			}
		}
	}
	if best == math.MaxFloat64 {
		for i := range s.Tracks {
			for j := range s.Tracks[i].Notes {
				if s.Tracks[i].Notes[j].StartSec < best {
					best = s.Tracks[i].Notes[j].StartSec
				}
			}
		}
	}
	if best == math.MaxFloat64 {
		return 0
	}
	return best
}

// SortNotes ensures each track is ordered by onset.
func (s *Song) SortNotes() {
	for i := range s.Tracks {
		notes := s.Tracks[i].Notes
		sort.SliceStable(notes, func(a, b int) bool {
			if notes[a].Tick != notes[b].Tick {
				return notes[a].Tick < notes[b].Tick
			}
			return notes[a].Key < notes[b].Key
		})
	}
}

// ComputeIdentity hashes the file bytes, so per-song state follows renames.
func ComputeIdentity(raw []byte) string {
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%d-%s", len(raw), hex.EncodeToString(sum[:12]))
}

// BPMAt returns the tempo in BPM at a song time.
func (s *Song) BPMAt(sec float64) float64 {
	if len(s.Tempo) == 0 {
		return 120
	}
	i := sort.Search(len(s.Tempo), func(i int) bool { return s.Tempo[i].Sec > sec }) - 1
	if i < 0 {
		i = 0
	}
	return s.Tempo[i].BPM
}

// SigAt returns the meter at a song time.
func (s *Song) SigAt(sec float64) (num, den int) {
	if len(s.Sigs) == 0 {
		return 4, 4
	}
	i := sort.Search(len(s.Sigs), func(i int) bool { return s.Sigs[i].Sec > sec }) - 1
	if i < 0 {
		i = 0
	}
	return s.Sigs[i].Num, s.Sigs[i].Den
}

// KeyAt returns the key signature at a song time.
func (s *Song) KeyAt(sec float64) (semitone int, minor bool, ok bool) {
	if len(s.Keys) == 0 {
		return 0, false, false
	}
	i := sort.Search(len(s.Keys), func(i int) bool { return s.Keys[i].Sec > sec }) - 1
	if i < 0 {
		i = 0
	}
	return s.Keys[i].Semitone, s.Keys[i].Minor, true
}

// NoteTick resolves a note's end tick.
func (n Note) EndTick() int64 { return n.Tick + n.DurTicks }

// Duration is the sounding length in seconds.
func (n Note) Duration() float64 { return n.EndSec - n.StartSec }

// BarsAndBeats returns the bar and beat times in the visible window.
func (s *Song) BarsAndBeats() (bars, beats []float64) {
	origin := s.OriginSec()
	for _, b := range s.Bars {
		if b >= origin-1e-6 {
			bars = append(bars, b)
		}
	}
	for _, b := range s.Beats {
		if b >= origin-1e-6 {
			beats = append(beats, b)
		}
	}
	return bars, beats
}

// BarStarts is the effective (lead-cut) bar list.
func (s *Song) BarStarts() []float64 {
	bars, _ := s.BarsAndBeats()
	return bars
}

// EffectiveBeats is the effective beat list.
func (s *Song) EffectiveBeats() []float64 {
	_, beats := s.BarsAndBeats()
	return beats
}
