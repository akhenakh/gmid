package app

import (
	"fmt"

	"github.com/akhenakh/gmid/internal/dspreset"
	"github.com/akhenakh/gmid/internal/sampler"
)

// LoadDecentPreset installs a DecentSampler instrument and persists the path.
func (a *App) LoadDecentPreset(path string) error {
	if a.Engine == nil {
		return fmt.Errorf("audio engine is not running")
	}
	if err := a.Engine.LoadDecentPreset(path); err != nil {
		return err
	}
	a.Engine.SetDecentGain(a.Prefs.DecentGain)
	a.Prefs.DecentPreset = path
	_ = a.Prefs.Save()
	a.Notify("Loaded DecentSampler: " + path)
	return nil
}

// SetDecentGain applies and persists the user's DecentSampler gain in dB.
func (a *App) SetDecentGain(db float64) {
	a.Prefs.DecentGain = db
	if a.Engine != nil {
		a.Engine.SetDecentGain(db)
	}
	_ = a.Prefs.Save()
}

// UnloadDecentPreset removes the DecentSampler instrument.
func (a *App) UnloadDecentPreset() {
	if a.Engine != nil {
		a.Engine.UnloadDecent()
	}
	a.Prefs.DecentPreset = ""
	_ = a.Prefs.Save()
	a.Notify("DecentSampler unloaded")
}

// DecentStatus reports the loaded instrument.
func (a *App) DecentStatus() (sampler.Stats, bool) {
	if a.Engine == nil {
		return sampler.Stats{}, false
	}
	return a.Engine.DecentInfo()
}

// DecentControls returns the instrument's parsed UI controls.
func (a *App) DecentControls() []dspreset.Control {
	if a.Engine == nil {
		return nil
	}
	return a.Engine.DecentControls()
}
