package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	. "go.hasen.dev/shirei"
	shireiapp "go.hasen.dev/shirei/app"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/midiio"
	"github.com/akhenakh/gmid/ui"
)

func main() {
	var (
		pngPath = flag.String("png", "", "render one settled frame to PATH and exit")
		scene   = flag.String("scene", "shell", "scene preset for headless renders")
		width   = flag.Int("width", 1440, "headless render width")
		height  = flag.Int("height", 900, "headless render height")
		assets  = flag.String("assets", "", "assets directory override")
		noAudio = flag.Bool("noaudio", false, "do not open the audio device")
	)
	flag.Parse()

	assetsDir := *assets
	if assetsDir == "" {
		assetsDir = config.AssetsDir()
	}
	if _, err := os.Stat(assetsDir); err != nil {
		log.Printf("assets directory %q not found; run `make assets`", assetsDir)
	}

	if fonts := config.FontPaths(); len(fonts) > 0 {
		UseFontFiles(fonts...)
	}

	prefs, err := config.Load()
	if err != nil {
		log.Printf("preferences: %v (using defaults)", err)
	}
	if prefs.ShowDemo {
		config.SeedDemo(assetsDir, prefs.LibraryFolder)
	}
	ui.Install(prefs.Theme == "warm")

	a := app.New(prefs, assetsDir, *scene)

	var engine *audio.Engine
	if *pngPath == "" && !*noAudio {
		engine = audio.New(48000)
		a.Engine = engine
		if err := engine.StartOutput(prefs.OutputDevice, prefs.ShareMode == "exclusive", prefs.BufferFrames); err != nil {
			a.Notify("Audio device: " + err.Error())
		}
	}
	prepareScene(a, *scene)
	if engine != nil {
		go loadSoundFont(a, engine, prefs)
	}
	if *pngPath == "" && !*noAudio {
		a.MIDI = midiio.New(a)
	}

	if *pngPath != "" {
		if err := os.MkdirAll(filepath.Dir(*pngPath), 0o755); err != nil {
			log.Fatal(err)
		}
		if err := RenderToPNG(*pngPath, *width, *height, func() { ui.Root(a) }); err != nil {
			log.Fatalf("render: %v", err)
		}
		fmt.Println(*pngPath)
		return
	}

	shireiapp.SetupWindow("gmid", 1440, 900)
	shireiapp.SetupDrive()
	shireiapp.Run(func() { ui.Root(a) })
	if engine != nil {
		engine.Close()
	}
	if a.MIDI != nil {
		a.MIDI.Close()
	}
}

// loadSoundFont loads the GM bank off the UI thread, then re-installs the open
// song so playback can start immediately.
func loadSoundFont(a *app.App, engine *audio.Engine, prefs config.Prefs) {
	path := prefs.SoundFont
	if path == "" {
		path = filepath.Join(a.Assets, "soundfonts", "GeneralUser-GS.sf2")
	}
	if _, err := os.Stat(path); err != nil {
		publishStatus(a, "Soundfont missing: "+path)
		return
	}
	if err := engine.LoadSoundFont(path); err != nil {
		publishStatus(a, "Soundfont: "+err.Error())
		return
	}
	WithFrameLock(func() {
		if a.Song != nil {
			snap := engine.Snapshot()
			engine.Load(a.Song)
			engine.Seek(snap.Time)
			if snap.Playing {
				engine.Play(snap.Time)
			}
			a.PushAudioState()
		}
	})
	RequestNextFrame()
}

func publishStatus(a *app.App, msg string) {
	WithFrameLock(func() { a.Notify(msg) })
	RequestNextFrame()
}

func prepareScene(a *app.App, scene string) {
	switch scene {
	case "", "shell":
		return
	case "roll", "mixer", "settings", "presentation", "keys-floating":
	default:
		// Unknown scene names behave like the roll so new scenes can be added
		// before their dedicated layout exists.
	}
	if demos := a.DemoFiles(); len(demos) > 0 {
		if err := a.LoadFile(demos[0]); err != nil {
			a.Notify(err.Error())
		}
	}
	switch scene {
	case "roll", "mixer", "presentation":
		a.Seek(8)
	}
	switch scene {
	case "mixer":
		a.UI.MixerOpen = true
	case "library":
		a.UI.LibraryOpen = true
	case "settings":
		a.UI.SettingsOpen = true
	case "presentation":
		a.UI.Presentation = true
	case "keys-floating":
		a.UI.FloatingKeys = true
	case "note-editor":
		a.BeginNoteEditor()
	case "chord-explorer":
		a.UI.ChordExplorerOpen = true
	case "scale-explorer":
		a.UI.ScaleExplorerOpen = true
	case "export":
		a.UI.ExportOpen = true
	}
}
