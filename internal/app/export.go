package app

import (
	"path/filepath"

	shirei "go.hasen.dev/shirei"

	"github.com/akhenakh/gmid/internal/export"
)

// ExportState tracks a background audio export.
type ExportState struct {
	Running  bool
	Progress float64
	Message  string
}

// SoundFontPath returns the configured or bundled soundfont path.
func (a *App) SoundFontPath() string {
	if a.Prefs.SoundFont != "" {
		return a.Prefs.SoundFont
	}
	return filepath.Join(a.Assets, "soundfonts", "GeneralUser-GS.sf2")
}

// TranscribeChordChart writes a chord chart PDF in the background.
func (a *App) TranscribeChordChart() {
	if a.Song == nil {
		return
	}
	s := a.Song
	lib := a.Prefs.LibraryFolder
	go func() {
		path, err := export.ChordChart(s, lib)
		shirei.WithFrameLock(func() {
			if err != nil {
				a.Notify("Chart failed: " + err.Error())
			} else {
				a.Notify("Chart saved " + path)
			}
		})
		shirei.RequestNextFrame()
	}()
}

// ExportAudio renders the current song in the background.
func (a *App) ExportAudio(variant export.Variant, format string) {
	if a.Song == nil || a.ExportState.Running {
		return
	}
	s := a.Song
	a.ExportState = ExportState{Running: true, Message: "Rendering…"}
	opts := export.Options{
		Song:         s,
		Variant:      variant,
		Format:       format,
		SoundFont:    a.SoundFontPath(),
		DecentPreset: a.Prefs.DecentPreset,
		Library:      a.Prefs.LibraryFolder,
		Master:       a.Prefs.MasterVolume,
		Tone:         a.Prefs.MasterTone,
		Compressor:   a.Prefs.MasterCompressor,
		Speed:        a.Transport.Speed,
		Transpose:    a.Transport.Transpose,
		LoopOn:       a.Transport.LoopEnabled,
		LoopStart:    a.Transport.LoopStart,
		LoopEnd:      a.Transport.LoopEnd,
		Mixes:        a.Mixes,
	}
	opts.Progress = func(p float64) {
		shirei.WithFrameLock(func() { a.ExportState.Progress = p })
	}
	go func() {
		path, err := export.Run(opts)
		shirei.WithFrameLock(func() {
			a.ExportState.Running = false
			if err != nil {
				a.ExportState.Message = err.Error()
				a.Notify("Export failed: " + err.Error())
			} else {
				a.ExportState.Message = path
				a.Notify("Exported " + path)
			}
		})
		shirei.RequestNextFrame()
	}()
}
