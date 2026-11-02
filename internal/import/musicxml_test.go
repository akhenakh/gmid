package importer

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `<?xml version="1.0" encoding="UTF-8"?>
<score-partwise version="3.1">
  <work><work-title>Test Piece</work-title></work>
  <part-list>
    <score-part id="P1"><part-name>Piano</part-name>
      <midi-instrument><midi-channel>1</midi-channel><midi-program>2</midi-program></midi-instrument>
    </score-part>
  </part-list>
  <part id="P1">
    <measure number="1">
      <attributes>
        <divisions>2</divisions>
        <key><fifths>0</fifths><mode>major</mode></key>
        <time><beats>4</beats><beat-type>4</beat-type></time>
        <staves>1</staves>
      </attributes>
      <direction><sound tempo="90"/></direction>
      <note><pitch><step>C</step><octave>4</octave></pitch><duration>2</duration><type>quarter</type></note>
      <note><pitch><step>E</step><octave>4</octave></pitch><duration>2</duration><type>quarter</type></note>
      <note><pitch><step>G</step><octave>4</octave></pitch><duration>2</duration><type>quarter</type></note>
    </measure>
    <measure number="2">
      <note><pitch><step>C</step><octave>5</octave></pitch><duration>4</duration><type>half</type></note>
      <backup><duration>4</duration></backup>
      <note><pitch><step>C</step><octave>3</octave></pitch><duration>4</duration><type>half</type></note>
    </measure>
  </part>
</score-partwise>`

func TestMusicXML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.musicxml")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := loadMusicXML(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Test Piece" {
		t.Fatalf("name %q", s.Name)
	}
	if s.NoteCount != 5 {
		t.Fatalf("notes %d", s.NoteCount)
	}
	if len(s.Tracks) != 1 {
		t.Fatalf("tracks %d", len(s.Tracks))
	}
	if s.Tracks[0].Program != 1 {
		t.Fatalf("program %d", s.Tracks[0].Program)
	}
	if bpm := s.BPMAt(0); bpm < 89.9 || bpm > 90.1 {
		t.Fatalf("tempo %v", bpm)
	}
	if len(s.Bars) < 2 {
		t.Fatalf("bars %d", len(s.Bars))
	}
}
