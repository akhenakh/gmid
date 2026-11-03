package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

const prefsVersion = 2

// Prefs is the durable user configuration. Soundfont/mixer choices that belong
// to one song live in the file, not here.
type Prefs struct {
	Version int `json:"version"`

	LibraryFolder      string `json:"libraryFolder"`
	LastFile           string `json:"lastFile"`
	ShowDemo           bool   `json:"showDemo"`
	ExportAudioEnabled bool   `json:"exportAudioEnabled"`

	SoundEngine  string  `json:"soundEngine"`  // gm | studio | native
	SoundFont    string  `json:"soundFont"`    // absolute path or bundled name
	DecentPreset string  `json:"decentPreset"` // .dspreset or .dslibrary
	DecentGain   float64 `json:"decentGain"`   // user gain in dB on top of level matching
	OutputDevice string  `json:"outputDevice"`
	AudioBackend string  `json:"audioBackend"` // auto | pulse | alsa | jack | null
	ShareMode    string  `json:"shareMode"`    // shared | exclusive
	BufferFrames int     `json:"bufferFrames"`
	MasterVolume float64 `json:"masterVolume"`
	AutoLevel    bool    `json:"autoLevel"`

	Metronome       bool    `json:"metronome"`
	MetronomeVolume float64 `json:"metronomeVolume"`
	CountIn         bool    `json:"countIn"`
	CountInBars     int     `json:"countInBars"`
	SkipEmptyBars   bool    `json:"skipEmptyBars"`
	KeepDrumCountIn bool    `json:"keepDrumCountIn"`

	TapTempo           bool `json:"tapTempo"`
	TempoWarnings      bool `json:"tempoWarnings"`
	LoopRegion         bool `json:"loopRegion"`
	FocusMode          bool `json:"focusMode"`
	StepThrough        bool `json:"stepThrough"`
	ChordPrompter      bool `json:"chordPrompter"`
	ChordTranscription bool `json:"chordTranscription"`
	ClosePanelsOnPlay  bool `json:"closePanelsOnPlay"`
	ExplorerPause      bool `json:"explorerPause"`

	HandAssignment bool `json:"handAssignment"`
	HandTags       bool `json:"handTags"`
	TrackVU        bool `json:"trackVU"`
	NoteEditor     bool `json:"noteEditor"`

	HandLH   string `json:"handLH"`
	HandRH   string `json:"handRH"`
	Theme    string `json:"theme"` // dark | warm
	Tooltips bool   `json:"tooltips"`
	Starmap  bool   `json:"starmap"`

	NoteNaming       string  `json:"noteNaming"`    // english | central-european | solfege | hidden
	Accidentals      string  `json:"accidentals"`   // sharp | flat
	ChordNaming      string  `json:"chordNaming"`   // abbreviation | symbol
	ChordTracking    string  `json:"chordTracking"` // auto | harmony | instrument
	ChordReading     string  `json:"chordReading"`  // simple | full
	KeyRange         int     `json:"keyRange"`      // 61 | 73 | 88
	FitSongRange     bool    `json:"fitSongRange"`  // trim the keyboard to the song's pitches
	ShowOctaves      bool    `json:"showOctaves"`
	ShowNoteNames    bool    `json:"showNoteNames"`
	ReflectPianoRoll bool    `json:"reflectPianoRoll"`
	PreciseScrubTime bool    `json:"preciseScrubTime"`
	RollZoom         float64 `json:"rollZoom"`
	BarNumbers       bool    `json:"barNumbers"`
	Playbar          bool    `json:"playbar"`
	VisualEffects    bool    `json:"visualEffects"`
	VisualEffect     string  `json:"visualEffect"`
	EffectColor      string  `json:"effectColor"`
	AppZoom          float64 `json:"appZoom"`

	LiveResponse    string   `json:"liveResponse"` // stable | balanced
	LatencyOffsetMS float64  `json:"latencyOffsetMS"`
	TouchResponse   int      `json:"touchResponse"` // 0..100 curve
	DuplicateFilter bool     `json:"duplicateFilter"`
	PedalsControls  bool     `json:"pedalsControls"`
	MidiReconnect   bool     `json:"midiReconnect"`
	SeenDevices     []string `json:"seenDevices"`

	MasterTone       float64 `json:"masterTone"`
	MasterCompressor bool    `json:"masterCompressor"`

	Stars []string `json:"stars"`

	TempoFixes map[string]TempoFix `json:"tempoFixes,omitempty"`
}

// TempoFix is a tap-tempo correction stored by song identity.
type TempoFix struct {
	PeriodSec   float64 `json:"periodSec"`
	AnchorSec   float64 `json:"anchorSec"`
	BeatsPerBar int     `json:"beatsPerBar"`
}

// Defaults returns the fresh-install configuration.
func Defaults() Prefs {
	return Prefs{
		Version:            prefsVersion,
		ShowDemo:           true,
		ExportAudioEnabled: true,
		SoundEngine:        "gm",
		ShareMode:          "shared",
		AudioBackend:       "auto",
		BufferFrames:       512,
		MasterVolume:       0.85,
		AutoLevel:          true,
		Metronome:          false,
		MetronomeVolume:    0.6,
		CountIn:            false,
		CountInBars:        1,
		SkipEmptyBars:      false,
		KeepDrumCountIn:    true,
		TapTempo:           true,
		TempoWarnings:      true,
		LoopRegion:         true,
		FocusMode:          true,
		StepThrough:        true,
		ChordPrompter:      false,
		ChordTranscription: false,
		ClosePanelsOnPlay:  false,
		ExplorerPause:      true,
		HandAssignment:     true,
		HandTags:           false,
		TrackVU:            true,
		NoteEditor:         true,
		HandLH:             "#6270A5",
		HandRH:             "#CB636C",
		Theme:              "dark",
		Tooltips:           true,
		Starmap:            false,
		NoteNaming:         "english",
		Accidentals:        "flat",
		ChordNaming:        "abbreviation",
		ChordTracking:      "auto",
		ChordReading:       "simple",
		KeyRange:           88,
		FitSongRange:       true,
		ShowOctaves:        true,
		ShowNoteNames:      true,
		ReflectPianoRoll:   true,
		PreciseScrubTime:   false,
		RollZoom:           1,
		BarNumbers:         true,
		Playbar:            true,
		VisualEffects:      true,
		VisualEffect:       "glow",
		EffectColor:        "#e8a027",
		AppZoom:            1,
		LiveResponse:       "balanced",
		LatencyOffsetMS:    0,
		TouchResponse:      50,
		DuplicateFilter:    false,
		PedalsControls:     true,
		MidiReconnect:      true,
		MasterTone:         0.5,
		MasterCompressor:   true,
	}
}

// Path returns the preferences file. GMID_CONFIG overrides it for tests and
// portable runtimes.
func Path() string {
	if p := os.Getenv("GMID_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(DataDir(), "prefs.json")
}

// Load reads preferences, returning defaults for missing or invalid files.
func Load() (Prefs, error) {
	prefs := Defaults()
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return prefs, nil
		}
		return prefs, err
	}
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Defaults(), err
	}
	prefs.fixup()
	return prefs, nil
}

// Save writes preferences atomically.
func (p *Prefs) Save() error {
	p.fixup()
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (p *Prefs) fixup() {
	d := Defaults()
	if p.Version < 2 {
		// v2 raises the old 256-frame default: it underran on PipeWire systems.
		if p.BufferFrames == 256 {
			p.BufferFrames = 512
		}
		p.Version = prefsVersion
	}
	if p.Version == 0 {
		p.Version = prefsVersion
	}
	if p.SoundEngine != "gm" && p.SoundEngine != "studio" && p.SoundEngine != "native" {
		p.SoundEngine = d.SoundEngine
	}
	if p.ShareMode != "shared" && p.ShareMode != "exclusive" {
		p.ShareMode = d.ShareMode
	}
	switch p.AudioBackend {
	case "auto", "pulse", "alsa", "jack", "null":
	default:
		p.AudioBackend = d.AudioBackend
	}
	if p.Theme != "warm" {
		p.Theme = "dark"
	}
	if p.KeyRange != 61 && p.KeyRange != 73 {
		p.KeyRange = 88
	}
	if p.CountInBars < 1 || p.CountInBars > 4 {
		p.CountInBars = 1
	}
	if p.BufferFrames < 32 {
		p.BufferFrames = d.BufferFrames
	}
	if p.BufferFrames > 8192 {
		p.BufferFrames = 8192
	}
	if p.MasterVolume < 0 || p.MasterVolume > 1.5 {
		p.MasterVolume = d.MasterVolume
	}
	if p.MetronomeVolume < 0 || p.MetronomeVolume > 1 {
		p.MetronomeVolume = d.MetronomeVolume
	}
	if p.RollZoom < 0.25 || p.RollZoom > 4 {
		p.RollZoom = 1
	}
	if p.AppZoom < 0.5 || p.AppZoom > 2.5 {
		p.AppZoom = 1
	}
	if p.LibraryFolder == "" {
		p.LibraryFolder = DefaultLibrary()
	}
	if !validHexColor(p.HandLH) || !validHexColor(p.HandRH) {
		// Older/zeroed preferences predate hand colours. Restore the stock
		// pair and turn colouring on so the feature is visible without a
		// settings trip.
		if !validHexColor(p.HandLH) {
			p.HandLH = d.HandLH
		}
		if !validHexColor(p.HandRH) {
			p.HandRH = d.HandRH
		}
		p.HandAssignment = true
	}
}

func validHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 24)
	return err == nil
}
