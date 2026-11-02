// Package midiio owns hardware MIDI input through rtmidi.
package midiio

import (
	"sort"
	"sync"
	"time"

	"gitlab.com/gomidi/midi/v2"
	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

// Handler receives live MIDI events. It is called from the rtmidi callback
// goroutine; implementations must be quick and must not block on the UI.
type Handler interface {
	MIDINoteOn(key, velocity int)
	MIDINoteOff(key int)
	MIDIControl(cc, value int)
	MIDIPitchBend(value int) // -8192..8191
}

// Device is one discovered input port.
type Device struct {
	Name string
	Port drivers.In
}

// Manager keeps one listener per input port and polls for hotplug.
type Manager struct {
	driver  *rtmididrv.Driver
	handler Handler

	mu      sync.Mutex
	open    map[string]drivers.In
	stop    map[string]func()
	devices []Device
	events  int64

	pollStop chan struct{}
}

// New creates the manager and performs a first scan.
func New(handler Handler) *Manager {
	m := &Manager{
		handler:  handler,
		open:     map[string]drivers.In{},
		stop:     map[string]func(){},
		pollStop: make(chan struct{}),
	}
	if d, err := rtmididrv.New(); err == nil {
		m.driver = d
	}
	m.Refresh()
	go m.poll()
	return m
}

// Devices returns the ports found by the last scan.
func (m *Manager) Devices() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.devices))
	for _, d := range m.devices {
		out = append(out, d.Name)
	}
	sort.Strings(out)
	return out
}

// Events reports how many live events have arrived.
func (m *Manager) Events() int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.events
}

// Refresh re-scans ports and opens any new input.
func (m *Manager) Refresh() {
	if m.driver == nil {
		return
	}
	ins, err := m.driver.Ins()
	if err != nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[string]bool{}
	m.devices = m.devices[:0]
	for _, in := range ins {
		name := in.String()
		seen[name] = true
		m.devices = append(m.devices, Device{Name: name, Port: in})
		if _, ok := m.open[name]; ok {
			continue
		}
		if err := in.Open(); err != nil {
			continue
		}
		port := in
		stop, lerr := port.Listen(func(msg []byte, _ int32) {
			m.dispatch(msg)
		}, drivers.ListenConfig{})
		if lerr != nil {
			_ = port.Close()
			continue
		}
		m.open[name] = port
		m.stop[name] = stop
	}
	for name, port := range m.open {
		if seen[name] {
			continue
		}
		if stop := m.stop[name]; stop != nil {
			stop()
		}
		_ = port.Close()
		delete(m.open, name)
		delete(m.stop, name)
	}
}

// Close releases every port.
func (m *Manager) Close() {
	close(m.pollStop)
	m.mu.Lock()
	defer m.mu.Unlock()
	for name, stop := range m.stop {
		if stop != nil {
			stop()
		}
		if p := m.open[name]; p != nil {
			_ = p.Close()
		}
	}
	m.open = map[string]drivers.In{}
	m.stop = map[string]func(){}
	if m.driver != nil {
		_ = m.driver.Close()
	}
}

func (m *Manager) poll() {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-m.pollStop:
			return
		case <-t.C:
			m.Refresh()
		}
	}
}

func (m *Manager) dispatch(raw []byte) {
	msg := midi.Message(raw)
	var ch, key, vel uint8
	if msg.GetNoteOn(&ch, &key, &vel) {
		m.mu.Lock()
		m.events++
		m.mu.Unlock()
		if vel == 0 {
			if m.handler != nil {
				m.handler.MIDINoteOff(int(key))
			}
			return
		}
		if m.handler != nil {
			m.handler.MIDINoteOn(int(key), int(vel))
		}
		return
	}
	if msg.GetNoteOff(&ch, &key, &vel) {
		if m.handler != nil {
			m.handler.MIDINoteOff(int(key))
		}
		return
	}
	var cc, value uint8
	if msg.GetControlChange(&ch, &cc, &value) {
		if m.handler != nil {
			m.handler.MIDIControl(int(cc), int(value))
		}
		return
	}
	var rel int16
	var abs uint16
	if msg.GetPitchBend(&ch, &rel, &abs) {
		if m.handler != nil {
			m.handler.MIDIPitchBend(int(rel))
		}
	}
}
