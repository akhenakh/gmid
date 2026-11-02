package app

import shirei "go.hasen.dev/shirei"

// focusState remembers one track's mute/visibility before Focus mode.
type focusState struct {
	muted   bool
	visible bool
}

// FocusActive reports whether Focus mode is on.
func (a *App) FocusActive() bool { return len(a.focusSnapshot) > 0 }

// ToggleFocus turns Focus mode on or off. On: keys are heard and shown, bass
// and drums are heard but hidden, everything else is muted and hidden. Off
// restores each track exactly.
func (a *App) ToggleFocus() {
	if a.Song == nil || !a.Prefs.FocusMode {
		return
	}
	if a.FocusActive() {
		for i, st := range a.focusSnapshot {
			if i < len(a.Song.Tracks) {
				a.Song.Tracks[i].Muted = st.muted
				a.Song.Tracks[i].Visible = st.visible
			}
		}
		a.focusSnapshot = nil
	} else {
		a.focusSnapshot = make([]focusState, len(a.Song.Tracks))
		for i := range a.Song.Tracks {
			t := &a.Song.Tracks[i]
			a.focusSnapshot[i] = focusState{muted: t.Muted, visible: t.Visible}
			switch {
			case isKeysGroup(t.Group):
				t.Muted = false
				t.Visible = true
			case t.Group == "bass" || t.IsDrum:
				t.Muted = false
				t.Visible = false
			default:
				t.Muted = true
				t.Visible = false
			}
		}
	}
	a.PushAudioState()
	shirei.RequestNextFrame()
}

func isKeysGroup(group string) bool {
	return group == "piano" || group == "organ"
}
