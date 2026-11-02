package app

import (
	"path/filepath"

	shirei "go.hasen.dev/shirei"
)

// SetSoundEngine switches GM/Studio/Native and restarts output when needed.
func (a *App) SetSoundEngine(kind string) {
	if kind != "gm" && kind != "studio" && kind != "native" {
		return
	}
	a.Prefs.SoundEngine = kind
	_ = a.Prefs.Save()
	if a.Engine == nil {
		return
	}
	exclusive := kind == "native" && a.Prefs.ShareMode == "exclusive"
	if err := a.Engine.StartOutput(a.Prefs.OutputDevice, exclusive, a.Prefs.BufferFrames); err != nil {
		a.Notify("Native output unavailable, using shared audio")
		_ = a.Engine.StartOutput(a.Prefs.OutputDevice, false, a.Prefs.BufferFrames)
	}
	path := a.Prefs.SoundFont
	if kind == "gm" || path == "" {
		path = filepath.Join(a.Assets, "soundfonts", "GeneralUser-GS.sf2")
	}
	go func() {
		err := a.Engine.LoadSoundFont(path)
		shirei.WithFrameLock(func() {
			if err != nil {
				a.Notify("Soundfont: " + err.Error())
				return
			}
			if a.Song != nil {
				a.Engine.Load(a.Song)
				a.PushAudioState()
			}
		})
		shirei.RequestNextFrame()
	}()
}
