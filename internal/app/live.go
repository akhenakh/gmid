package app

import shirei "go.hasen.dev/shirei"

// KeyDown plays a live note from the on-screen or hardware keyboard.
func (a *App) KeyDown(key int) {
	if a.LiveKeys == nil {
		a.LiveKeys = map[int]bool{}
	}
	if a.LiveKeys[key] {
		return
	}
	a.LiveKeys[key] = true
	if a.Engine != nil {
		a.Engine.NoteOn(15, key, 100)
	}
	shirei.RequestNextFrame()
}

// KeyUp releases a live note.
func (a *App) KeyUp(key int) {
	if a.LiveKeys == nil {
		return
	}
	if !a.LiveKeys[key] {
		return
	}
	delete(a.LiveKeys, key)
	if a.Engine != nil {
		a.Engine.NoteOff(15, key)
	}
	shirei.RequestNextFrame()
}

// ReleaseAllKeys silences live notes (window focus loss, stop).
func (a *App) ReleaseAllKeys() {
	for k := range a.LiveKeys {
		if a.Engine != nil {
			a.Engine.NoteOff(15, k)
		}
	}
	a.LiveKeys = nil
}
