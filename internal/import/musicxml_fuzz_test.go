package importer

import (
	"archive/zip"
	"bytes"
	"testing"
)

func FuzzParseMusicXML(f *testing.F) {
	f.Add([]byte(sample))
	f.Add([]byte(`<score-partwise version="3.1"></score-partwise>`))
	f.Add([]byte(`<score-partwise><part-list></part-list><part id="P1"></part></score-partwise>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		s, err := parseMusicXML(data, "fuzz.musicxml")
		if err == nil && s == nil {
			t.Fatal("nil song without an error")
		}
	})
}

func FuzzUnzipMusicXML(f *testing.F) {
	f.Add(mxlSeed(f))
	f.Add([]byte("not a zip"))
	f.Fuzz(func(t *testing.T, data []byte) {
		out, err := unzipMusicXML(data)
		if err == nil && len(out) == 0 {
			t.Fatal("empty score without an error")
		}
	})
}

// mxlSeed builds a minimal compressed MusicXML container.
func mxlSeed(tb testing.TB) []byte {
	tb.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	container, err := zw.Create("META-INF/container.xml")
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := container.Write([]byte(`<?xml version="1.0"?>
<container><rootfiles><rootfile full-path="score.xml"/></rootfiles></container>`)); err != nil {
		tb.Fatal(err)
	}
	score, err := zw.Create("score.xml")
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := score.Write([]byte(sample)); err != nil {
		tb.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		tb.Fatal(err)
	}
	return buf.Bytes()
}

// TestMalformedMusicXML covers the failure paths the importer must survive.
func TestMalformedMusicXML(t *testing.T) {
	cases := map[string][]byte{
		"empty":      {},
		"not xml":    []byte("{\"json\": true}"),
		"no parts":   []byte(`<score-partwise version="3.1"></score-partwise>`),
		"unclosed":   []byte(`<score-partwise><part-list>`),
		"bad octave": []byte(`<score-partwise><part-list><score-part id="P1"><part-name>P</part-name></score-part></part-list><part id="P1"><measure number="1"><attributes><divisions>4</divisions></attributes><note><pitch><step>H</step><octave>99</octave></pitch><duration>-8</duration></note></measure></part></score-partwise>`),
		"zero div":   []byte(`<score-partwise><part-list><score-part id="P1"><part-name>P</part-name></score-part></part-list><part id="P1"><measure number="1"><attributes><divisions>0</divisions></attributes><note><pitch><step>C</step><octave>4</octave></pitch><duration>0</duration></note></measure></part></score-partwise>`),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, _ = parseMusicXML(raw, name+".musicxml")
		})
	}
}

func TestMalformedMXL(t *testing.T) {
	cases := map[string][]byte{
		"empty":   {},
		"not zip": []byte("not a zip"),
		"empty zip": func() []byte {
			var buf bytes.Buffer
			if err := zip.NewWriter(&buf).Close(); err != nil {
				t.Fatal(err)
			}
			return buf.Bytes()
		}(),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, _ = unzipMusicXML(raw)
		})
	}
}
