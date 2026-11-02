package sampler

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewkiz/flac"
)

// Buffer is a loaded stereo sample at its native rate.
type Buffer struct {
	L, R      []float32
	Rate      int
	Loop      bool
	LoopStart int
	LoopEnd   int
}

// LoadBuffer reads WAV, AIFF or FLAC into float32 stereo.
func LoadBuffer(path string) (*Buffer, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".wav":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return loadWAV(data)
	case ".aif", ".aiff", ".aifc":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return loadAIFF(data)
	case ".flac":
		return loadFLAC(path)
	default:
		return nil, fmt.Errorf("unsupported sample format %s", filepath.Ext(path))
	}
}

func loadWAV(data []byte) (*Buffer, error) {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a WAV file")
	}
	var (
		channels, bits, rate int
		format               = 1
		pcm                  []byte
		loopStart, loopEnd   int
		hasLoop              bool
	)
	for off := 12; off+8 <= len(data); {
		id := string(data[off : off+4])
		size := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		body := off + 8
		end := body + size
		if end > len(data) {
			end = len(data)
		}
		switch id {
		case "fmt ":
			if end-body >= 16 {
				format = int(binary.LittleEndian.Uint16(data[body : body+2]))
				channels = int(binary.LittleEndian.Uint16(data[body+2 : body+4]))
				rate = int(binary.LittleEndian.Uint32(data[body+4 : body+8]))
				bits = int(binary.LittleEndian.Uint16(data[body+14 : body+16]))
				if format == 0xFFFE && end-body >= 40 {
					format = int(binary.LittleEndian.Uint16(data[body+24 : body+26]))
				}
			}
		case "data":
			pcm = data[body:end]
		case "smpl":
			if end-body >= 36 {
				nLoops := int(binary.LittleEndian.Uint32(data[body+28 : body+32]))
				if nLoops > 0 && end-body >= 60 {
					loopStart = int(binary.LittleEndian.Uint32(data[body+44 : body+48]))
					loopEnd = int(binary.LittleEndian.Uint32(data[body+48 : body+52]))
					hasLoop = loopEnd > loopStart
				}
			}
		}
		off = end
		if size%2 == 1 {
			off++
		}
	}
	if channels <= 0 || bits <= 0 || rate <= 0 || len(pcm) == 0 {
		return nil, fmt.Errorf("invalid WAV header")
	}
	frames := len(pcm) / (channels * (bits / 8))
	buf := &Buffer{Rate: rate, L: make([]float32, frames), R: make([]float32, frames)}
	step := channels * (bits / 8)
	for i := 0; i < frames; i++ {
		base := i * step
		l := decodePCM(pcm[base:], bits, format == 3)
		var r float32
		if channels > 1 {
			r = decodePCM(pcm[base+bits/8:], bits, format == 3)
		} else {
			r = l
		}
		buf.L[i], buf.R[i] = l, r
	}
	if hasLoop {
		buf.Loop = true
		buf.LoopStart = loopStart
		buf.LoopEnd = loopEnd
	}
	return buf, nil
}

func decodePCM(b []byte, bits int, isFloat bool) float32 {
	switch bits {
	case 8:
		if len(b) < 1 {
			return 0
		}
		return (float32(b[0]) - 128) / 128
	case 16:
		if len(b) < 2 {
			return 0
		}
		return float32(int16(binary.LittleEndian.Uint16(b))) / 32768
	case 24:
		if len(b) < 3 {
			return 0
		}
		v := int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16
		if v&0x800000 != 0 {
			v |= ^0xFFFFFF
		}
		return float32(v) / 8388608
	case 32:
		if len(b) < 4 {
			return 0
		}
		u := binary.LittleEndian.Uint32(b)
		if isFloat {
			return math.Float32frombits(u)
		}
		return float32(int32(u)) / 2147483648
	}
	return 0
}

func loadAIFF(data []byte) (*Buffer, error) {
	if len(data) < 12 || string(data[:4]) != "FORM" {
		return nil, fmt.Errorf("not an AIFF file")
	}
	var (
		channels, bits int
		rate           int
		pcm            []byte
		offset         int
		little         bool
		isFloat        bool
	)
	for off := 12; off+8 <= len(data); {
		id := string(data[off : off+4])
		size := int(binary.BigEndian.Uint32(data[off+4 : off+8]))
		body := off + 8
		end := body + size
		if end > len(data) {
			end = len(data)
		}
		switch id {
		case "COMM":
			if end-body >= 18 {
				channels = int(binary.BigEndian.Uint16(data[body : body+2]))
				bits = int(binary.BigEndian.Uint16(data[body+6 : body+8]))
				rate = int(extended80(data[body+8 : body+18]))
				if end-body >= 22 {
					comp := string(data[body+18 : body+22])
					switch comp {
					case "sowt":
						little = true
					case "fl32", "FL32":
						isFloat = true
					}
				}
			}
		case "SSND":
			if end-body >= 8 {
				offset = int(binary.BigEndian.Uint32(data[body : body+4]))
				start := body + 8 + offset
				if start < end {
					pcm = data[start:end]
				}
			}
		}
		off = end
		if size%2 == 1 {
			off++
		}
	}
	if channels <= 0 || bits <= 0 || rate <= 0 || len(pcm) == 0 {
		return nil, fmt.Errorf("invalid AIFF header")
	}
	frames := len(pcm) / (channels * (bits / 8))
	buf := &Buffer{Rate: rate, L: make([]float32, frames), R: make([]float32, frames)}
	step := channels * (bits / 8)
	for i := 0; i < frames; i++ {
		base := i * step
		l := decodeAIFPCM(pcm[base:], bits, little, isFloat)
		r := l
		if channels > 1 {
			r = decodeAIFPCM(pcm[base+bits/8:], bits, little, isFloat)
		}
		buf.L[i], buf.R[i] = l, r
	}
	return buf, nil
}

func decodeAIFPCM(b []byte, bits int, little, isFloat bool) float32 {
	var order binary.ByteOrder = binary.BigEndian
	if little {
		order = binary.LittleEndian
	}
	switch bits {
	case 8:
		return (float32(b[0]) - 128) / 128
	case 16:
		return float32(int16(order.Uint16(b))) / 32768
	case 24:
		if little {
			v := int32(b[0]) | int32(b[1])<<8 | int32(b[2])<<16
			if v&0x800000 != 0 {
				v |= ^0xFFFFFF
			}
			return float32(v) / 8388608
		}
		v := int32(b[2]) | int32(b[1])<<8 | int32(b[0])<<16
		if v&0x800000 != 0 {
			v |= ^0xFFFFFF
		}
		return float32(v) / 8388608
	case 32:
		if isFloat {
			return math.Float32frombits(order.Uint32(b))
		}
		return float32(int32(order.Uint32(b))) / 2147483648
	}
	return 0
}

func extended80(b []byte) float64 {
	if len(b) < 10 {
		return 0
	}
	exp := int(binary.BigEndian.Uint16(b[:2]) & 0x7FFF)
	mant := binary.BigEndian.Uint64(b[2:])
	if exp == 0 && mant == 0 {
		return 0
	}
	return float64(mant) * math.Pow(2, float64(exp-16383-63))
}

func loadFLAC(path string) (*Buffer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	stream, err := flac.New(f)
	if err != nil {
		return nil, err
	}
	info := stream.Info
	buf := &Buffer{Rate: int(info.SampleRate)}
	scale := float32(1) / float32(uint64(1)<<(info.BitsPerSample-1))
	for {
		frame, err := stream.ParseNext()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		n := len(frame.Subframes[0].Samples)
		for i := 0; i < n; i++ {
			l := float32(frame.Subframes[0].Samples[i]) * scale
			r := l
			if len(frame.Subframes) > 1 {
				r = float32(frame.Subframes[1].Samples[i]) * scale
			}
			buf.L = append(buf.L, l)
			buf.R = append(buf.R, r)
		}
	}
	return buf, nil
}

// DecodeBytes is used by tests.
func DecodeBytes(data []byte, ext string) (*Buffer, error) {
	switch strings.ToLower(ext) {
	case ".wav":
		return loadWAV(data)
	case ".aif", ".aiff":
		return loadAIFF(data)
	default:
		return nil, fmt.Errorf("unsupported")
	}
}
