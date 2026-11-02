package app

import (
	"fmt"
	"path/filepath"

	shirei "go.hasen.dev/shirei"

	"github.com/akhenakh/gmid/internal/audio"
)

// OpenOutput starts audio with the configured backend and falls back to auto
// when that backend is unavailable, so a bad choice never leaves the app mute.
func (a *App) OpenOutput(exclusive bool) error {
	if a.Engine == nil {
		return fmt.Errorf("audio engine is not running")
	}
	err := a.Engine.StartOutput(a.Prefs.OutputDevice, a.Prefs.AudioBackend, exclusive, a.Prefs.BufferFrames)
	if err == nil {
		return nil
	}
	if a.Prefs.AudioBackend != "auto" {
		if err2 := a.Engine.StartOutput(a.Prefs.OutputDevice, "auto", false, a.Prefs.BufferFrames); err2 == nil {
			a.Prefs.AudioBackend = "auto"
			_ = a.Prefs.Save()
			a.Notify("Audio backend unavailable; using auto")
			return nil
		}
	}
	return err
}

// SetSoundEngine switches GM/Studio/Native and restarts output when needed.
func (a *App) SetSoundEngine(kind string) {
	if kind != "gm" && kind != "studio" && kind != "native" {
		return
	}
	a.Prefs.SoundEngine = kind
	_ = a.Prefs.Save()
	if a.Prefs.DecentPreset != "" {
		a.Notify("DecentSampler is loaded and overrides melodic tracks — unload it to hear this engine")
	} else if (kind == "studio" || kind == "native") && a.Prefs.SoundFont == "" {
		a.Notify("Studio and Native use the GM bank until a Studio soundfont is set")
	}
	if a.Engine == nil {
		return
	}
	exclusive := kind == "native" && a.Prefs.ShareMode == "exclusive"
	if err := a.OpenOutput(exclusive); err != nil {
		a.Notify("Native output unavailable, using shared audio")
		_ = a.Engine.StartOutput(a.Prefs.OutputDevice, "auto", false, a.Prefs.BufferFrames)
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

// RestartAudio reopens the output device with the current preferences.
func (a *App) RestartAudio() {
	if a.Engine == nil {
		return
	}
	exclusive := a.Prefs.SoundEngine == "native" && a.Prefs.ShareMode == "exclusive"
	if err := a.OpenOutput(exclusive); err != nil {
		a.Notify("Audio output: " + err.Error())
		return
	}
	a.Notify("Audio output restarted")
}

// SetAudioBackend switches the output backend and reopens the device.
func (a *App) SetAudioBackend(name string) {
	switch name {
	case "auto", "pulse", "alsa", "jack", "null":
	default:
		return
	}
	a.Prefs.AudioBackend = name
	_ = a.Prefs.Save()
	a.UI.AudioDevices = nil
	a.RefreshAudioDevices()
	a.RestartAudio()
}

// SetBufferFrames changes the requested period size and reopens the device.
func (a *App) SetBufferFrames(frames int) {
	if frames < 32 {
		frames = 32
	}
	if frames > 8192 {
		frames = 8192
	}
	a.Prefs.BufferFrames = frames
	_ = a.Prefs.Save()
	a.RestartAudio()
}

// RefreshAudioDevices repopulates the output device list for the current
// backend. It is called from the UI thread (frame lock held).
func (a *App) RefreshAudioDevices() {
	devices, err := audio.ListDevicesFor(a.Prefs.AudioBackend)
	if err != nil {
		a.Notify("Audio devices: " + err.Error())
		a.UI.AudioDevices = nil
		return
	}
	a.UI.AudioDevices = devices
}

// SetAudioDevice selects an output device by name (empty = system default).
func (a *App) SetAudioDevice(name string) {
	a.Prefs.OutputDevice = name
	_ = a.Prefs.Save()
	a.RestartAudio()
}
