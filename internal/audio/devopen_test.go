package audio

import (
	"os"
	"testing"
)

// TestOutputDeviceByID guards the cgo device-ID path: the ID must be pinned
// before it is stored in the malgo config, or cgo panics.
func TestOutputDeviceByID(t *testing.T) {
	if os.Getenv("GMID_AUDIO_TEST") == "" {
		t.Skip("set GMID_AUDIO_TEST=1 to open the real device")
	}
	ds, err := ListDevicesFor("pulse")
	if err != nil || len(ds) == 0 {
		t.Skip(err)
	}
	e := New(48000)
	if err := e.StartOutput(ds[0].Name, "pulse", false, 512); err != nil {
		t.Fatal(err)
	}
	e.Close()
	t.Logf("opened %q", ds[0].Name)
}
