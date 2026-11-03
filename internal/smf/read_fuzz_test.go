package smf

import (
	"os"
	"testing"
)

// A minimal valid SMF: one track with only an end-of-track meta event.
var minimalSMF = []byte{
	'M', 'T', 'h', 'd', 0, 0, 0, 6, 0, 0, 0, 1, 0x01, 0xE0,
	'M', 'T', 'r', 'k', 0, 0, 0, 4, 0x00, 0xFF, 0x2F, 0x00,
}

func FuzzParse(f *testing.F) {
	f.Add(minimalSMF)
	if raw, err := os.ReadFile(demo); err == nil {
		f.Add(raw)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		s, err := Parse(raw, "fuzz.mid")
		if err != nil {
			return
		}
		if s == nil {
			t.Fatal("nil song without an error")
		}
	})
}

// TestMalformedFiles pins the contract that corrupt input is rejected or
// tolerated without panicking.
func TestMalformedFiles(t *testing.T) {
	cases := map[string][]byte{
		"empty":            {},
		"no header":        []byte("not a midi file"),
		"truncated header": minimalSMF[:10],
		"bad magic":        append([]byte("XXXX"), minimalSMF[4:]...),
		"zero tracks":      {0x4D, 0x54, 0x68, 0x64, 0, 0, 0, 6, 0, 0, 0, 0, 0x01, 0xE0},
		"zero division":    {0x4D, 0x54, 0x68, 0x64, 0, 0, 0, 6, 0, 0, 0, 1, 0, 0},
		"truncated track":  minimalSMF[:len(minimalSMF)-2],
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, _ = Parse(raw, name+".mid")
		})
	}
}
