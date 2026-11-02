package app

import shirei "go.hasen.dev/shirei"

// MIDINoteOn plays a hardware note.
func (a *App) MIDINoteOn(key, velocity int) {
	// Touch response is a curve around the default velocity.
	v := velocity
	if a.Prefs.TouchResponse != 50 {
		shape := float64(a.Prefs.TouchResponse) / 50
		v = int(float64(velocity) * shape)
		if v < 1 {
			v = 1
		}
		if v > 127 {
			v = 127
		}
	}
	shirei.WithFrameLock(func() {
		if a.LiveKeys == nil {
			a.LiveKeys = map[int]bool{}
		}
		if a.LiveKeys[key] && a.Prefs.DuplicateFilter {
			return
		}
		a.LiveKeys[key] = true
		if a.Engine != nil {
			a.Engine.NoteOn(15, key, v)
		}
	})
	shirei.RequestNextFrame()
}

// MIDINoteOff releases a hardware note.
func (a *App) MIDINoteOff(key int) {
	shirei.WithFrameLock(func() {
		if a.LiveKeys == nil {
			return
		}
		delete(a.LiveKeys, key)
		if a.Engine != nil {
			a.Engine.NoteOff(15, key)
		}
	})
	shirei.RequestNextFrame()
}

// MIDIControl forwards pedals and controllers to the live channel.
func (a *App) MIDIControl(cc, value int) {
	if !a.Prefs.PedalsControls {
		return
	}
	if a.Engine != nil {
		a.Engine.Control(15, cc, value)
	}
}

// MIDIPitchBend forwards pitch bend.
func (a *App) MIDIPitchBend(value int) {
	if a.Engine != nil {
		a.Engine.PitchBend(15, value)
	}
}

// MIDIDevices lists the connected input ports.
func (a *App) MIDIDevices() []string {
	if a.MIDI == nil {
		return nil
	}
	return a.MIDI.Devices()
}
