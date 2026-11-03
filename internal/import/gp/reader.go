// Package gp parses Guitar Pro 3, 4 and 5 files natively.
//
// The binary layouts follow the documented Guitar Pro format as implemented by
// PyGuitarPro (LGPL-3.0). gmid reads only what it needs to build its song model
// — tempo, time signatures, tracks, tunings, notes and velocities — while
// consuming every field so the stream stays aligned.
package gp

import (
	"fmt"
	"math"
	"strings"
)

const (
	maxString     = 1 << 20
	maxTracks     = 256
	maxMeasures   = 10000
	maxBeats      = 4096
	maxChordNotes = 16
	maxPoints     = 1024
	maxComments   = 1000
)

// reader is a bounds-checked little-endian stream. Every read records the first
// error; callers can continue linearly and check err once at the end.
type reader struct {
	data []byte
	pos  int
	err  error
}

func (r *reader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
}

func (r *reader) bytes(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.data)-r.pos {
		r.fail("unexpected end of file at offset %d", r.pos)
		return nil
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) u8() uint8 {
	b := r.bytes(1)
	if b == nil {
		return 0
	}
	return b[0]
}

func (r *reader) i8() int8 { return int8(r.u8()) }

func (r *reader) bool() bool { return r.u8() != 0 }

func (r *reader) i16() int16 {
	b := r.bytes(2)
	if b == nil {
		return 0
	}
	return int16(uint16(b[0]) | uint16(b[1])<<8)
}

func (r *reader) i32() int32 {
	b := r.bytes(4)
	if b == nil {
		return 0
	}
	return int32(uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24)
}

func (r *reader) f64() float64 {
	b := r.bytes(8)
	if b == nil {
		return 0
	}
	var bits uint64
	for i := 7; i >= 0; i-- {
		bits = bits<<8 | uint64(b[i])
	}
	return math.Float64frombits(bits)
}

func (r *reader) skip(n int) { r.bytes(n) }

// skipOptional consumes up to n bytes and does not fail at end of file. Guitar
// Pro files omit the trailing optional line-break byte in the final measure.
func (r *reader) skipOptional(n int) {
	if r.err != nil {
		return
	}
	if n > len(r.data)-r.pos {
		n = len(r.data) - r.pos
	}
	r.pos += n
}

// fixedString reads a length byte followed by n fixed-width bytes.
func (r *reader) fixedString(n int) string {
	size := int(r.u8())
	b := r.bytes(n)
	if b == nil {
		return ""
	}
	if size > n {
		size = n
	}
	return latin1(b[:size])
}

// intString reads a 32-bit length followed by that many bytes.
func (r *reader) intString() string {
	n := int(r.i32())
	if r.err != nil {
		return ""
	}
	if n < 0 || n > maxString {
		r.fail("string length %d out of range", n)
		return ""
	}
	return latin1(r.bytes(n))
}

// intByteString reads a 32-bit byte count, then a length byte and count-1 bytes.
func (r *reader) intByteString() string {
	count := int(r.i32())
	if r.err != nil {
		return ""
	}
	if count <= 0 {
		return ""
	}
	if count > maxString {
		r.fail("string length %d out of range", count)
		return ""
	}
	size := int(r.u8())
	b := r.bytes(count - 1)
	if b == nil {
		return ""
	}
	if size > len(b) {
		size = len(b)
	}
	return latin1(b[:size])
}

func (r *reader) color() string {
	b := r.bytes(4)
	if b == nil {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", b[0], b[1], b[2])
}

func (r *reader) count(v int32, max int, what string) int {
	if r.err != nil {
		return 0
	}
	if v < 0 || int(v) > max {
		r.fail("%s count %d out of range", what, v)
		return 0
	}
	return int(v)
}

func latin1(b []byte) string {
	ascii := true
	for _, c := range b {
		if c >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return string(b)
	}
	var sb strings.Builder
	sb.Grow(len(b))
	for _, c := range b {
		sb.WriteRune(rune(c))
	}
	return sb.String()
}
