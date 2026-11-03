package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go.hasen.dev/generic"
	. "go.hasen.dev/shirei"
	shireiapp "go.hasen.dev/shirei/app"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/config"
	gmidmcp "github.com/akhenakh/gmid/internal/mcp"
	"github.com/akhenakh/gmid/internal/midiio"
	"github.com/akhenakh/gmid/ui"
)

// version is stamped by GoReleaser at release time.
var version = "dev"

func main() {
	var (
		showVersion = flag.Bool("version", false, "print the version and exit")
		pngPath     = flag.String("png", "", "render one settled frame to PATH and exit")
		scene       = flag.String("scene", "shell", "scene preset for headless renders")
		width       = flag.Int("width", 1440, "headless render width")
		height      = flag.Int("height", 900, "headless render height")
		assets      = flag.String("assets", "", "assets directory override")
		noAudio     = flag.Bool("noaudio", false, "do not open the audio device")
		mcpOn       = flag.Bool("mcp", true, "start the gmid MCP server")
		mcpAddr     = flag.String("mcp-addr", "127.0.0.1:8765", "MCP listen address")
		mcpToken    = flag.String("mcp-token", "", "optional bearer token for MCP")
	)
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	assetsDir := *assets
	if assetsDir == "" {
		assetsDir = config.AssetsDir()
	}
	if _, err := os.Stat(assetsDir); err != nil {
		log.Printf("assets directory %q not found; run `make assets`", assetsDir)
	}
	if icon := filepath.Join(assetsDir, "icons", "gmid.png"); fileExists(icon) {
		shireiapp.SetupIcon(icon)
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
		if err := a.OpenOutput(prefs.ShareMode == "exclusive"); err != nil {
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
	if *pngPath == "" && *mcpOn && *mcpAddr != "" {
		mcpServer := gmidmcp.New(a)
		go func() {
			log.Printf("MCP: http://%s/mcp (SSE /sse)", *mcpAddr)
			if err := mcpServer.Serve(*mcpAddr, *mcpToken); err != nil {
				log.Printf("MCP server: %v", err)
			}
		}()
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

	shireiapp.SetupWindow("gmid", *width, *height)
	shireiapp.SetupDrive()
	generic.AddExitCleanup(func() {
		if engine != nil {
			engine.Close()
		}
		if a.MIDI != nil {
			a.MIDI.Close()
		}
	})
	shireiapp.Run(func() { ui.Root(a) })
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
		if a.Prefs.DecentPreset != "" {
			if err := engine.LoadDecentPreset(a.Prefs.DecentPreset); err != nil {
				a.Notify("DecentSampler: " + err.Error())
			} else {
				engine.SetDecentGain(a.Prefs.DecentGain)
			}
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
	case "starmap":
		a.UI.ScaleExplorerOpen = true
		a.UI.ScaleStarmap = true
	case "export":
		a.UI.ExportOpen = true
	}
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
