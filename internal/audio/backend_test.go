package audio

import (
	"testing"

	"github.com/gen2brain/malgo"
)

func TestSelectedBackends(t *testing.T) {
	cases := map[string][]malgo.Backend{
		"":         nil,
		"auto":     nil,
		"pulse":    {malgo.BackendPulseaudio},
		"alsa":     {malgo.BackendAlsa},
		"jack":     {malgo.BackendJack},
		"null":     {malgo.BackendNull},
		"bogus":    nil,
		"pipewire": {malgo.BackendPulseaudio},
	}
	for name, want := range cases {
		got := selectedBackends(name)
		if len(got) != len(want) {
			t.Fatalf("%q: %v, want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%q: %v, want %v", name, got, want)
			}
		}
	}
}
