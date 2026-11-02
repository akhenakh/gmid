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
	"github.com/akhenakh/gmid/internal/config"
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
	ui.Install(prefs.Theme == "warm")

	a := app.New(prefs, assetsDir, *scene)
	prepareScene(a, *scene)

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

	_ = noAudio
	shireiapp.SetupWindow("gmid", 1440, 900)
	shireiapp.Run(func() { ui.Root(a) })
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
	case "mixer":
		a.UI.MixerOpen = true
	case "settings":
		a.UI.SettingsOpen = true
	case "presentation":
		a.UI.Presentation = true
	case "keys-floating":
		a.UI.FloatingKeys = true
	}
}
