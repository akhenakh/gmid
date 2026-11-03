package gp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

// bw is a tiny Guitar Pro writer used only by tests. It keeps the fixtures
// self-contained: no third-party tablature files need to ship with gmid.
type bw struct {
	bytes.Buffer
}

func (w *bw) u8(v uint8)  { w.WriteByte(v) }
func (w *bw) i8(v int8)   { w.WriteByte(uint8(v)) }
func (w *bw) i16(v int16) { _ = binary.Write(&w.Buffer, binary.LittleEndian, v) }
func (w *bw) i32(v int32) { _ = binary.Write(&w.Buffer, binary.LittleEndian, v) }

func (w *bw) version(s string) {
	w.u8(uint8(len(s)))
	pad := make([]byte, 30)
	copy(pad, s)
	w.Write(pad)
}

func (w *bw) fixed40(s string) {
	w.u8(uint8(len(s)))
	pad := make([]byte, 40)
	copy(pad, s)
	w.Write(pad)
}

func (w *bw) intByteString(s string) {
	w.i32(int32(len(s) + 1))
	w.u8(uint8(len(s)))
	w.WriteString(s)
}

func (w *bw) intString(s string) {
	w.i32(int32(len(s)))
	w.WriteString(s)
}

func (w *bw) info(fields int) {
	for i := 0; i < fields; i++ {
		switch i {
		case 0:
			w.intByteString("Test Song")
		case 2:
			w.intByteString("gmid")
		default:
			w.intByteString("")
		}
	}
	w.i32(0) // no comments
}

func (w *bw) channels() {
	for i := 0; i < 64; i++ {
		switch i {
		case 0:
			w.i32(25) // acoustic guitar
		default:
			w.i32(0)
		}
		w.Write(make([]byte, 6)) // volume..tremolo
		w.Write(make([]byte, 2))
	}
}

func (w *bw) color() { w.Write([]byte{0x40, 0x80, 0xc0, 0}) }

func (w *bw) header3(num, den int) {
	w.u8(0x03)
	w.i8(int8(num))
	w.i8(int8(den))
}

func (w *bw) header5(num, den int, first bool) {
	if !first {
		w.u8(0)
	}
	w.u8(0x03)
	w.i8(int8(num))
	w.i8(int8(den))
	w.Write(make([]byte, 4)) // beams
	w.u8(0)                  // padding when no alternate ending
	w.u8(0)                  // triplet feel
}

func (w *bw) track3() {
	w.u8(0) // flags
	w.fixed40("Guitar")
	w.i32(6) // strings
	for _, t := range []int32{64, 59, 55, 50, 45, 40, 0} {
		w.i32(t)
	}
	w.i32(0) // port
	w.i32(1) // channel (1-based)
	w.i32(1) // effect channel
	w.i32(24)
	w.i32(0) // capo
	w.color()
}

func (w *bw) track5() {
	w.u8(0) // leading byte for first track
	w.u8(8) // visible
	w.fixed40("Guitar")
	w.i32(6)
	for _, t := range []int32{64, 59, 55, 50, 45, 40, 0} {
		w.i32(t)
	}
	w.i32(0)
	w.i32(1)
	w.i32(1)
	w.i32(24)
	w.i32(0)
	w.color()
	w.i16(0) // track settings
	w.u8(0)  // auto accentuation
	w.u8(0)  // bank
	w.u8(0)  // humanize
	w.i32(0) // clef transpose
	w.i32(0) // secondary clef transpose
	w.i32(0) // unknown
	w.Write(make([]byte, 12))
	w.i32(0) // RSE instrument
	w.i32(0)
	w.i32(0)
	w.i32(0) // effect number
	w.Write(make([]byte, 4))
	w.intByteString("")
	w.intByteString("")
}

func (w *bw) beatGP3(flags uint8, dur int8, stringFlags uint8, noteType uint8, fret int8, gp5 bool) {
	w.u8(flags)
	if flags&0x40 != 0 {
		w.u8(2) // rest
	}
	w.i8(dur)
	if flags&0x02 != 0 {
		w.u8(0) // old chord format, no frets
		w.intByteString("")
		w.i32(0)
	}
	if flags&0x04 != 0 {
		w.intByteString("")
	}
	if flags&0x08 != 0 {
		w.u8(0) // beat effects flags
	}
	if flags&0x10 != 0 {
		w.i8(-1) // mix table: instrument
		for i := 0; i < 6; i++ {
			w.i8(-1) // dynamics unchanged
		}
		w.i32(-1) // tempo unchanged
	}
	w.u8(stringFlags)
	if stringFlags != 0 {
		w.u8(0x20) // note flags: type + fret
		w.u8(noteType)
		w.i8(fret)
		if gp5 {
			w.u8(0) // swap accidentals
		}
	}
}

func (w *bw) beatGP5(flags uint8, dur int8, stringFlags uint8, noteType uint8, fret int8) {
	w.beatGP3(flags, dur, stringFlags, noteType, fret, true)
	w.i16(0) // GP5 beat flags
}

func buildGP(family int) []byte {
	var w bw
	switch family {
	case 3:
		w.version("FICHIER GUITAR PRO v3.00")
		w.info(8)
		w.u8(0) // triplet
		w.i32(90)
		w.i32(0) // key
		w.channels()
		w.i32(2) // measures
		w.i32(1) // tracks
		w.header3(4, 4)
		w.header3(3, 4)
		w.track3()
	case 4:
		w.version("FICHIER GUITAR PRO v4.06")
		w.info(8)
		w.u8(0)
		w.i32(0) // lyric track
		for i := 0; i < 5; i++ {
			w.i32(0)
			w.intString("")
		}
		w.i32(90)
		w.i32(0)
		w.i8(0) // octave
		w.channels()
		w.i32(2)
		w.i32(1)
		w.header3(4, 4)
		w.header3(3, 4)
		w.track3()
	case 5:
		w.version("FICHIER GUITAR PRO v5.10")
		w.info(9)
		w.i32(0)
		for i := 0; i < 5; i++ {
			w.i32(0)
			w.intString("")
		}
		w.i32(100) // master volume
		w.i32(0)
		w.Write(make([]byte, 11)) // equalizer
		// Page setup.
		w.Write(make([]byte, 8+16+4+2))
		for i := 0; i < 10; i++ {
			w.intByteString("")
		}
		w.intByteString("") // tempo name
		w.i32(90)
		w.u8(0) // hide tempo
		w.i8(0) // key
		w.i32(0)
		w.channels()
		for i := 0; i < 19; i++ {
			w.i16(-1)
		}
		w.i32(0) // master reverb
		w.i32(2)
		w.i32(1)
		w.header5(4, 4, true)
		w.header5(3, 4, false)
		w.track5()
		w.u8(0) // trailing track padding
	}
	// Measures.
	w.i32(4) // beats, measure 1
	if family == 5 {
		w.beatGP5(0, 0, 0x40, 1, 0)
	} else {
		w.beatGP3(0, 0, 0x40, 1, 0, false)
	}
	if family == 5 {
		w.beatGP5(0, 0, 0x40, 2, 0) // tie extends the previous note
	} else {
		w.beatGP3(0, 0, 0x40, 2, 0, false)
	}
	if family == 5 {
		w.beatGP5(0, 0, 0x20, 1, 1)
		w.beatGP5(0x40, 0, 0, 0, 0)
		w.i32(0) // voice 2 has no beats
		w.u8(0)  // line break
	} else {
		w.beatGP3(0, 0, 0x20, 1, 1, false)
		w.beatGP3(0x40, 0, 0, 0, 0, false)
	}
	w.i32(3) // beats, measure 2 (3/4)
	for _, fret := range []int8{0, 2, 3} {
		if family == 5 {
			w.beatGP5(0, 1, 0x40, 1, fret)
		} else {
			w.beatGP3(0, 1, 0x40, 1, fret, false)
		}
	}
	if family == 5 {
		w.i32(0)
		w.u8(0)
	}
	return w.Bytes()
}

func TestParseFamilies(t *testing.T) {
	for _, family := range []int{3, 4, 5} {
		t.Run(fmt.Sprintf("gp%d", family), func(t *testing.T) {
			raw := buildGP(family)
			s, err := Parse(raw, "test.gp"+string(rune('0'+family)))
			if err != nil {
				t.Fatal(err)
			}
			if s.Name != "Test Song" {
				t.Errorf("name %q", s.Name)
			}
			if len(s.Tracks) != 1 {
				t.Fatalf("tracks %d", len(s.Tracks))
			}
			tr := s.Tracks[0]
			if tr.Name != "Guitar" || tr.Program != 25 || tr.IsDrum {
				t.Errorf("track %+v", tr)
			}
			if len(tr.Notes) != 5 {
				t.Fatalf("notes %d: %+v", len(tr.Notes), tr.Notes)
			}
			// The tie on beat 2 extends the first note from 960 to 1920 ticks.
			if tr.Notes[0].Key != 64 || tr.Notes[0].DurTicks != 1920 || tr.Notes[0].Tick != 960 {
				t.Errorf("tied note %+v", tr.Notes[0])
			}
			if tr.Notes[1].Key != 60 || tr.Notes[1].Tick != 2880 {
				t.Errorf("second note %+v", tr.Notes[1])
			}
			if len(s.Sigs) < 2 || s.Sigs[len(s.Sigs)-1].Num != 3 {
				t.Errorf("sigs %+v", s.Sigs)
			}
			if len(s.Sigs) > 0 && s.Sigs[len(s.Sigs)-1].Tick != 4800 {
				t.Errorf("3/4 starts at %d, want 4800", s.Sigs[len(s.Sigs)-1].Tick)
			}
			if s.Tempo[0].BPM < 89.9 || s.Tempo[0].BPM > 90.1 {
				t.Errorf("tempo %+v", s.Tempo)
			}
			if len(s.Bars) < 2 {
				t.Errorf("bars %d", len(s.Bars))
			}
		})
	}
}

func FuzzParse(f *testing.F) {
	for _, family := range []int{3, 4, 5} {
		f.Add(buildGP(family))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		_, _ = Parse(raw, "fuzz.gp5")
	})
}

func TestMalformed(t *testing.T) {
	valid := buildGP(5)
	cases := map[string][]byte{
		"empty":     {},
		"version":   []byte("FICHIER GUITAR PRO v9.00"),
		"truncated": valid[:len(valid)/2],
		"garbage":   bytes.Repeat([]byte{0xff}, 64),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, _ = Parse(raw, name+".gp5")
		})
	}
}
