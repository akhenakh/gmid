package app

import (
	"time"

	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/library"
	"github.com/akhenakh/gmid/internal/midiio"
	"github.com/akhenakh/gmid/internal/song"
)

// App is the root application state. Views read and mutate it directly on the
// UI thread; background goroutines publish under Shirei's frame lock.
type App struct {
	Prefs  config.Prefs
	Assets string

	Song *song.Song

	Engine      Engine
	LiveKeys    map[int]bool
	LockedKeys  map[int]bool
	Mixes       map[int]audio.ChannelMix
	MixesDirty  bool
	MIDI        *midiio.Manager
	ExportState ExportState
	Lib         *library.Library

	focusSnapshot []focusState
	Hits          []Hit
	lastHitScan   float64
	hitSeq        int
	tapWall       []float64
	tapSong       []float64
	tapLast       time.Time
	Edit          EditorState

	Transport Transport

	Status string
	Scene  string

	UI UIState
}

// Transport is the current playback state.
type Transport struct {
	Playing     bool
	Time        float64 // song seconds
	Speed       float64 // 1 = file tempo
	Transpose   int
	LoopEnabled bool
	LoopStart   float64
	LoopEnd     float64
	LoopActive  bool
	CountIn     bool
}

type UIState struct {
	SetupTab   string // setup | tracks
	SideOpen   bool
	PanelsOpen bool

	SettingsOpen  bool
	SettingsGroup string
	SettingsQuery string

	LibraryOpen         bool
	FileBrowserOpen     bool
	LibQuery            string
	LibScanning         bool
	LibraryPath         string
	LibraryFilter       string
	LibrarySelected     int
	LibrarySelection    string
	MixerOpen           bool
	PlaybackEditorOpen  bool
	NoteEditorActive    bool
	ChordExplorerOpen   bool
	ScaleExplorerOpen   bool
	LockedChordOpen     bool
	ExportOpen          bool
	ExportVariant       int
	ExportFormat        string
	MetronomeMenu       bool
	ExplorerKey         int
	ExplorerChord       string
	ExplorerInversion   int
	ExplorerProgression int
	ExplorerOctave      int
	ScaleType           string
	ScaleRoot           int
	ScaleStarmap        bool
	Confirm             *Confirm
	MenuAnchor          string

	Presentation bool
	FloatingKeys bool
	KeysDocked   bool
	FloatX       float32
	FloatY       float32
	FloatW       float32
	FloatH       float32

	Toast     string
	ToastTill time.Time

	AudioDevices []audio.Device
}

type Confirm struct {
	Title   string
	Message string
	Action  func()
}

// New creates the root state from persisted preferences.
func New(prefs config.Prefs, assets, scene string) *App {
	return &App{
		Prefs:  prefs,
		Assets: assets,
		Scene:  scene,
		Lib:    library.New(prefs.LibraryFolder),
		Transport: Transport{
			Speed: 1,
		},
		UI: UIState{
			SetupTab:      "setup",
			SideOpen:      true,
			PanelsOpen:    true,
			KeysDocked:    true,
			SettingsGroup: "audio",
			ExplorerKey:   0,
			ExplorerChord: "",
			ScaleType:     "major",
			ExportFormat:  "wav",
			FloatW:        760,
			FloatH:        230,
			FloatX:        620,
			FloatY:        500,
		},
	}
}

// UpdatePref mutates preferences and persists them. Preference changes are small
// and infrequent; writing synchronously keeps state coherent without a queue.
func (a *App) UpdatePref(fn func(*config.Prefs)) {
	fn(&a.Prefs)
	_ = a.Prefs.Save()
}

// SavePrefs persists the current preferences.
func (a *App) SavePrefs() { _ = a.Prefs.Save() }

// Notify shows a transient status message.
func (a *App) Notify(msg string) {
	a.Status = msg
	a.UI.Toast = msg
	a.UI.ToastTill = time.Now().Add(4 * time.Second)
}

// Confirm asks the user before a destructive action.
func (a *App) Confirm(title, message string, action func()) {
	a.UI.Confirm = &Confirm{Title: title, Message: message, Action: action}
}

// CloseConfirm dismisses the pending confirmation.
func (a *App) CloseConfirm() { a.UI.Confirm = nil }
