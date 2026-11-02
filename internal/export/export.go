// Package export renders songs offline and writes WAV/MP3 files.
package export

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/config"
	shine "github.com/akhenakh/gmid/internal/export/shine"
	"github.com/akhenakh/gmid/internal/song"
)

// Variant selects what the export contains.
type Variant string

const (
	Original Variant = "original"
	Focus    Variant = "focus"
	Heard    Variant = "heard"
)

// Options configures one export.
type Options struct {
	Song         *song.Song
	Variant      Variant
	Format       string // wav | mp3
	SoundFont    string
	DecentPreset string
	Library      string
	Master       float64
	Tone         float64
	Compressor   bool
	Speed        float64
	Transpose    int
	LoopOn       bool
	LoopStart    float64
	LoopEnd      float64
	Mixes        map[int]audio.ChannelMix
	Progress     func(float64)
}

// Run renders and writes the file, returning its path.
func Run(o Options) (string, error) {
	if o.Song == nil {
		return "", fmt.Errorf("no song")
	}
	if o.SoundFont == "" {
		return "", fmt.Errorf("no soundfont")
	}
	if o.Speed <= 0 {
		o.Speed = 1
	}
	const sr = 44100
	e := audio.New(sr)
	if err := e.LoadSoundFont(o.SoundFont); err != nil {
		return "", err
	}
	if o.DecentPreset != "" {
		if err := e.LoadDecentPreset(o.DecentPreset); err != nil {
			return "", err
		}
	}
	e.SetCountIn(0)
	e.SetMetronome(false, 0)
	e.SetMasterVolume(o.Master)
	e.SetMasterTone(o.Tone)
	e.SetCompressor(o.Compressor)
	e.Load(o.Song)
	if o.Variant == Heard {
		e.SetSpeed(o.Speed)
		e.SetTranspose(o.Transpose)
	}
	if o.Variant == Heard && o.LoopOn && o.LoopEnd > o.LoopStart {
		e.SetLoop(true, o.LoopStart, o.LoopEnd)
	}
	switch o.Variant {
	case Focus:
		for i := range o.Song.Tracks {
			t := &o.Song.Tracks[i]
			audible := t.Group == "piano" || t.Group == "organ" || t.Group == "bass" || t.IsDrum
			e.SetTrackAudible(i, audible)
		}
	case Original:
		for i := range o.Song.Tracks {
			e.SetTrackAudible(i, true)
		}
	}
	if o.Variant == Heard && o.Mixes != nil {
		for ch, mix := range o.Mixes {
			e.SetChannelMix(ch, mix.Volume, mix.Pan, mix.Chorus, mix.Reverb)
		}
	}
	start := o.Song.OriginSec()
	end := o.Song.DurationSec
	if o.Variant == Heard && o.LoopOn && o.LoopEnd > o.LoopStart {
		start, end = o.LoopStart, o.LoopEnd
	}
	if end <= start {
		return "", fmt.Errorf("nothing to export")
	}
	e.Play(start)
	total := int(math.Round((end - start) / o.Speed * sr))
	if total <= 0 {
		return "", fmt.Errorf("nothing to export")
	}
	total += sr // one second of tail; trimmed later by peak scan
	left := make([]float32, 0, total)
	right := make([]float32, 0, total)
	const block = 4096
	done := 0
	for done < total {
		n := block
		if total-done < n {
			n = total - done
		}
		l, r := e.RenderOffline(n)
		left = append(left, l...)
		right = append(right, r...)
		done += n
		if o.Progress != nil && done%(block*10) == 0 {
			o.Progress(float64(done) / float64(total))
		}
	}
	if o.Progress != nil {
		o.Progress(1)
	}
	normalize(left, right, 0.891) // -1 dBFS
	dir := config.ExportsDir(o.Library)
	base := strings.TrimSuffix(filepath.Base(o.Song.Path), filepath.Ext(o.Song.Path))
	base = base + " - " + string(o.Variant)
	ext := ".wav"
	if strings.EqualFold(o.Format, "mp3") {
		ext = ".mp3"
	}
	path := nextFree(filepath.Join(dir, base+ext))
	if ext == ".wav" {
		if err := writeWAV(path, sr, left, right); err != nil {
			return "", err
		}
	} else {
		if err := writeMP3(path, sr, left, right); err != nil {
			return "", err
		}
	}
	return path, nil
}

func nextFree(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func normalize(l, r []float32, target float64) {
	peak := 0.0
	for i := range l {
		if v := math.Abs(float64(l[i])); v > peak {
			peak = v
		}
		if v := math.Abs(float64(r[i])); v > peak {
			peak = v
		}
	}
	if peak <= 1e-9 {
		return
	}
	g := float32(target / peak)
	for i := range l {
		l[i] *= g
	}
	for i := range r {
		r[i] *= g
	}
}

func writeWAV(path string, sr int, l, r []float32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	w := bufio.NewWriterSize(f, 1<<16)
	n := len(l)
	dataLen := uint32(n * 2 * 2)
	// RIFF header
	_, _ = w.WriteString("RIFF")
	_ = binary.Write(w, binary.LittleEndian, uint32(36+dataLen))
	_, _ = w.WriteString("WAVEfmt ")
	_ = binary.Write(w, binary.LittleEndian, uint32(16))
	_ = binary.Write(w, binary.LittleEndian, uint16(1))
	_ = binary.Write(w, binary.LittleEndian, uint16(2))
	_ = binary.Write(w, binary.LittleEndian, uint32(sr))
	_ = binary.Write(w, binary.LittleEndian, uint32(sr*2*2))
	_ = binary.Write(w, binary.LittleEndian, uint16(4))
	_ = binary.Write(w, binary.LittleEndian, uint16(16))
	_, _ = w.WriteString("data")
	_ = binary.Write(w, binary.LittleEndian, dataLen)
	for i := 0; i < n; i++ {
		_ = binary.Write(w, binary.LittleEndian, clamp16(l[i]))
		_ = binary.Write(w, binary.LittleEndian, clamp16(r[i]))
	}
	return w.Flush()
}

func writeMP3(path string, sr int, l, r []float32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	enc := shine.NewEncoder(sr, 2)
	inter := make([]int16, 0, len(l)*2)
	for i := range l {
		inter = append(inter, clamp16(l[i]), clamp16(r[i]))
	}
	return enc.Write(f, inter)
}

func clamp16(v float32) int16 {
	x := v * 32767
	if x > 32767 {
		x = 32767
	}
	if x < -32768 {
		x = -32768
	}
	return int16(x)
}
