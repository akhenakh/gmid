package importer

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/akhenakh/gmid/internal/smf"
	"github.com/akhenakh/gmid/internal/song"
)

// externalScoreConvert asks MuseScore to render a Guitar Pro or Capella file
// to MIDI. MuseScore is not bundled; the importer degrades cleanly when absent.
func externalScoreConvert(path string) (*song.Song, error) {
	var bin string
	for _, name := range []string{"mscore", "musescore", "musescore4", "mscore4"} {
		if p, err := exec.LookPath(name); err == nil {
			bin = p
			break
		}
	}
	if bin == "" {
		return nil, fmt.Errorf("no MuseScore command")
	}
	tmp, err := os.CreateTemp("", "gmid-import-*.mid")
	if err != nil {
		return nil, err
	}
	name := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(name)
	defer func() { _ = os.Remove(name) }()
	out, err := exec.Command(bin, "-o", name, path).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("musescore: %v: %s", err, string(out))
	}
	return smf.ReadFile(name)
}
