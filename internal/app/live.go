package app

import (
	"sort"
	"time"

	shirei "go.hasen.dev/shirei"
)

// KeyDown plays a live note from the on-screen or hardware keyboard.
func (a *App) KeyDown(key int) {
	if shirei.GetInputState().Modifiers&shirei.ModShift != 0 {
		a.ToggleLockedKey(key)
		return
	}
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

// ToggleLockedKey adds or removes a key from the Lock-a-Chord set.
func (a *App) ToggleLockedKey(key int) {
	if a.LockedKeys == nil {
		a.LockedKeys = map[int]bool{}
	}
	if a.LockedKeys[key] {
		delete(a.LockedKeys, key)
	} else {
		a.LockedKeys[key] = true
	}
	if len(a.LockedKeys) >= 3 {
		a.UI.LockedChordOpen = true
	}
	shirei.RequestNextFrame()
}

// LockedChord returns the locked keys, low to high.
func (a *App) LockedChord() []int {
	keys := make([]int, 0, len(a.LockedKeys))
	for k := range a.LockedKeys {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

// PreviewSequence plays a progression of chords.
func (a *App) PreviewSequence(chords [][]int, gapMS int) {
	for i, ch := range chords {
		delay := time.Duration(i*gapMS) * time.Millisecond
		keys := append([]int(nil), ch...)
		time.AfterFunc(delay, func() {
			shirei.WithFrameLock(func() {
				a.PreviewChord(keys, gapMS-40)
			})
		})
	}
}

// PreviewChord plays a short audition on the live channel.
func (a *App) PreviewChord(keys []int, ms int) {
	if a.LiveKeys == nil {
		a.LiveKeys = map[int]bool{}
	}
	for _, k := range keys {
		a.LiveKeys[k] = true
		if a.Engine != nil {
			a.Engine.NoteOn(15, k, 100)
		}
	}
	time.AfterFunc(time.Duration(ms)*time.Millisecond, func() {
		shirei.WithFrameLock(func() {
			for _, k := range keys {
				delete(a.LiveKeys, k)
				if a.Engine != nil {
					a.Engine.NoteOff(15, k)
				}
			}
			shirei.RequestNextFrame()
		})
	})
	shirei.RequestNextFrame()
}
