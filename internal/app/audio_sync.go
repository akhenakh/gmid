package app

import shirei "go.hasen.dev/shirei"

// SyncEngine copies the audio clock into the transport. Called once per frame
// before the UI is built.
func (a *App) SyncEngine() {
	if a.Engine == nil {
		return
	}
	snap := a.Engine.Snapshot()
	if snap.Time != a.Transport.Time {
		a.Transport.Time = snap.Time
	}
	if snap.Playing != a.Transport.Playing {
		a.Transport.Playing = snap.Playing
	}
	if snap.Playing {
		shirei.RequestNextFrame()
	}
}

// PushAudioState re-sends mute/solo, master level and metronome state after a
// UI change.
func (a *App) PushAudioState() {
	if a.Engine == nil || a.Song == nil {
		return
	}
	anySolo := a.Song.AnySolo()
	for i := range a.Song.Tracks {
		t := &a.Song.Tracks[i]
		audible := !t.Muted && !t.Excluded && (!anySolo || t.Solo)
		a.Engine.SetTrackAudible(i, audible)
	}
	a.Engine.SetMasterVolume(a.Prefs.MasterVolume)
	a.Engine.SetMetronome(a.Prefs.Metronome, a.Prefs.MetronomeVolume)
	if a.Prefs.CountIn {
		a.Engine.SetCountIn(a.Prefs.CountInBars)
	} else {
		a.Engine.SetCountIn(0)
	}
	a.Engine.SetSpeed(a.Transport.Speed)
	a.Engine.SetTranspose(a.Transport.Transpose)
	if a.Transport.LoopEnabled {
		a.Engine.SetLoop(true, a.Transport.LoopStart, a.Transport.LoopEnd)
	}
}
