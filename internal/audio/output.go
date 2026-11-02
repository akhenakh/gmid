package audio

import (
	"fmt"
	"log"
	"os"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// Backend names accepted by StartOutput.
const (
	BackendAuto  = "auto"
	BackendPulse = "pulse"
	BackendALSA  = "alsa"
	BackendJACK  = "jack"
	BackendNull  = "null"
)

func selectedBackends(name string) []malgo.Backend {
	switch name {
	case BackendPulse, "pulseaudio", "pipewire":
		return []malgo.Backend{malgo.BackendPulseaudio}
	case BackendALSA:
		return []malgo.Backend{malgo.BackendAlsa}
	case BackendJACK:
		return []malgo.Backend{malgo.BackendJack}
	case BackendNull:
		return []malgo.Backend{malgo.BackendNull}
	default:
		return nil
	}
}

func audioLogProc() malgo.LogProc {
	if os.Getenv("GMID_AUDIO_LOG") == "" {
		return nil
	}
	return func(msg string) { log.Printf("audio: %s", msg) }
}

type malgoOutput struct {
	ctx    *malgo.AllocatedContext
	dev    *malgo.Device
	frames int
}

// Device describes one playback device.
type Device struct {
	Name      string
	ID        string
	IsDefault bool
}

// ListDevices enumerates playback devices.
func ListDevices() ([]Device, error) {
	return ListDevicesFor(BackendAuto)
}

// ListDevicesFor enumerates playback devices for one backend (auto = all).
func ListDevicesFor(backend string) ([]Device, error) {
	ctx, err := malgo.InitContext(selectedBackends(backend), malgo.ContextConfig{}, audioLogProc())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()
	infos, err := ctx.Devices(malgo.Playback)
	if err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(infos))
	for i := range infos {
		out = append(out, Device{Name: infos[i].Name(), ID: infos[i].ID.String(), IsDefault: infos[i].IsDefault != 0})
	}
	return out, nil
}

// StartOutput opens a stereo float32 playback device. backend is auto, pulse,
// alsa, jack or null; frames is the requested period size.
func (e *Engine) StartOutput(deviceName, backend string, exclusive bool, frames int) error {
	if frames < 32 {
		frames = 512
	}
	ctx, err := malgo.InitContext(selectedBackends(backend), malgo.ContextConfig{
		ThreadPriority: malgo.ThreadPriorityRealtime,
	}, audioLogProc())
	if err != nil {
		return fmt.Errorf("audio context: %w", err)
	}
	cfg := malgo.DefaultDeviceConfig(malgo.Playback)
	cfg.Playback.Format = malgo.FormatF32
	cfg.Playback.Channels = 2
	cfg.SampleRate = uint32(e.sr)
	cfg.PeriodSizeInFrames = uint32(frames)
	cfg.PerformanceProfile = malgo.LowLatency
	cfg.Alsa.NoMMap = 1
	if exclusive {
		cfg.Playback.ShareMode = malgo.Exclusive
	} else {
		cfg.Playback.ShareMode = malgo.Shared
	}
	if deviceName != "" {
		infos, derr := ctx.Devices(malgo.Playback)
		if derr == nil {
			for i := range infos {
				if infos[i].Name() == deviceName {
					cfg.Playback.DeviceID = unsafe.Pointer(&infos[i].ID)
					break
				}
			}
		}
	}

	callbacks := malgo.DeviceCallbacks{
		Data: func(pOutput, _ []byte, framecount uint32) {
			e.onAudio(pOutput, framecount)
		},
	}
	dev, err := malgo.InitDevice(ctx.Context, cfg, callbacks)
	if err != nil {
		_ = ctx.Uninit()
		ctx.Free()
		return fmt.Errorf("audio device: %w", err)
	}
	if err := dev.Start(); err != nil {
		dev.Uninit()
		_ = ctx.Uninit()
		ctx.Free()
		return fmt.Errorf("audio start: %w", err)
	}
	e.mu.Lock()
	old := e.out
	e.out = &malgoOutput{ctx: ctx, dev: dev, frames: frames}
	e.mu.Unlock()
	if old != nil {
		old.close()
	}
	e.running.Store(true)
	return nil
}

func (o *malgoOutput) close() {
	if o == nil {
		return
	}
	if o.dev != nil {
		o.dev.Uninit()
	}
	if o.ctx != nil {
		_ = o.ctx.Uninit()
		o.ctx.Free()
	}
}

// onAudio is the realtime callback. It holds the engine mutex for the duration
// of the block; UI operations are short and do not allocate under the lock.
func (e *Engine) onAudio(pOutput []byte, framecount uint32) {
	n := int(framecount)
	if n <= 0 {
		return
	}
	out := unsafe.Slice((*float32)(unsafe.Pointer(&pOutput[0])), n*2)
	e.mu.Lock()
	if len(e.tmpL) < n {
		e.tmpL = make([]float32, n)
		e.tmpR = make([]float32, n)
	}
	if len(e.smpL) < n {
		e.smpL = make([]float32, n)
		e.smpR = make([]float32, n)
	}
	l := e.tmpL[:n]
	r := e.tmpR[:n]
	for i := range l {
		l[i], r[i] = 0, 0
	}
	e.renderLocked(l, r)
	e.mu.Unlock()
	for i := 0; i < n; i++ {
		out[i*2] = l[i]
		out[i*2+1] = r[i]
	}
}

// RenderOffline renders n frames without a device. Used by tests and export.
func (e *Engine) RenderOffline(frames int) ([]float32, []float32) {
	l := make([]float32, frames)
	r := make([]float32, frames)
	e.mu.Lock()
	if len(e.smpL) < frames {
		e.smpL = make([]float32, frames)
		e.smpR = make([]float32, frames)
	}
	e.renderLocked(l, r)
	e.mu.Unlock()
	return l, r
}

// OutputDelay approximates the device latency in seconds.
func (e *Engine) OutputDelay() float64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.out == nil {
		return 0
	}
	return float64(e.out.frames) / float64(e.sr)
}
