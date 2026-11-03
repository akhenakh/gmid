package export

import (
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	gowk "github.com/chinmay-sawant/gowkhtmltopdf"

	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/song"
	"github.com/akhenakh/gmid/internal/theory"
)

// ChordChart writes a printable chord chart PDF for a song.
func ChordChart(s *song.Song, library string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("no song")
	}
	bars := s.BarStarts()
	var rows strings.Builder
	for i, b := range bars {
		if i >= 200 {
			break
		}
		name := chordAt(s, b+0.02)
		fmt.Fprintf(&rows, `<div class="bar"><span class="n">%d</span><span class="c">%s</span></div>`,
			i+1, html.EscapeString(name))
	}
	title := html.EscapeString(s.Name)
	key := "—"
	if len(s.Keys) > 0 {
		key = theory.NoteName(s.Keys[0].Semitone, "english", "flat")
		if s.Keys[0].Minor {
			key += "m"
		}
	}
	htmlDoc := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8">
<style>
@page { size: A4; margin: 18mm; }
body { font-family: sans-serif; color:#111; }
h1 { font-size: 22px; margin: 0 0 4px; }
.sub { color:#555; font-size: 12px; margin-bottom: 16px; }
.grid { display: flex; flex-wrap: wrap; }
.bar { width: 23%%; border: 1px solid #bbb; padding: 8px; margin: 0 1%% 8px 0; box-sizing: border-box; }
.n { color:#888; font-size: 10px; }
.c { font-size: 18px; font-weight: bold; }
</style></head><body>
<h1>%s</h1>
<div class="sub">Chord chart · key %s · %d bars</div>
<div class="grid">%s</div>
</body></html>`, title, html.EscapeString(key), len(bars), rows.String())

	doc := gowk.Document{
		Pages:       []gowk.Page{{Source: gowk.Content{HTML: []byte(htmlDoc)}}},
		PageSize:    "A4",
		Orientation: "portrait",
		Margin:      gowk.Margin{Top: 14, Right: 12, Bottom: 14, Left: 12},
	}
	pdf, err := doc.PDF(context.Background())
	if err != nil {
		return "", err
	}
	base := strings.TrimSuffix(filepath.Base(s.Path), filepath.Ext(s.Path))
	dir := config.DownloadsDir()
	if dir != "" {
		if path, err := writeChartPDF(dir, base, pdf); err == nil {
			return path, nil
		}
	}
	return writeChartPDF(config.ExportsDir(library), base, pdf)
}

func writeChartPDF(dir, base string, pdf []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := nextFree(filepath.Join(dir, base+" - chords.pdf"))
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func chordAt(s *song.Song, t float64) string {
	set := map[int]bool{}
	for i := range s.Tracks {
		tr := &s.Tracks[i]
		if tr.IsDrum || tr.Excluded || tr.Muted {
			continue
		}
		for j := range tr.Notes {
			n := &tr.Notes[j]
			if n.StartSec <= t && t < n.EndSec {
				set[n.Key%12] = true
			}
		}
	}
	if len(set) < 3 {
		return "—"
	}
	keys := make([]int, 0, len(set))
	for pc := range set {
		keys = append(keys, pc)
	}
	return theory.DetectChord(keys)
}
