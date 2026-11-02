package theory

// GMNames is the General MIDI program name table.
var GMNames = []string{
	"Acoustic Grand Piano", "Bright Acoustic Piano", "Electric Grand Piano", "Honky-tonk Piano",
	"Electric Piano 1", "Electric Piano 2", "Harpsichord", "Clavinet",
	"Celesta", "Glockenspiel", "Music Box", "Vibraphone",
	"Marimba", "Xylophone", "Tubular Bells", "Dulcimer",
	"Drawbar Organ", "Percussive Organ", "Rock Organ", "Church Organ",
	"Reed Organ", "Accordion", "Harmonica", "Tango Accordion",
	"Nylon Guitar", "Steel Guitar", "Jazz Guitar", "Clean Guitar",
	"Muted Guitar", "Overdriven Guitar", "Distortion Guitar", "Guitar Harmonics",
	"Acoustic Bass", "Finger Bass", "Pick Bass", "Fretless Bass",
	"Slap Bass 1", "Slap Bass 2", "Synth Bass 1", "Synth Bass 2",
	"Violin", "Viola", "Cello", "Contrabass",
	"Tremolo Strings", "Pizzicato Strings", "Orchestral Harp", "Timpani",
	"String Ensemble 1", "String Ensemble 2", "Synth Strings 1", "Synth Strings 2",
	"Choir Aahs", "Voice Oohs", "Synth Voice", "Orchestra Hit",
	"Trumpet", "Trombone", "Tuba", "Muted Trumpet",
	"French Horn", "Brass Section", "Synth Brass 1", "Synth Brass 2",
	"Soprano Sax", "Alto Sax", "Tenor Sax", "Baritone Sax",
	"Oboe", "English Horn", "Bassoon", "Clarinet",
	"Piccolo", "Flute", "Recorder", "Pan Flute",
	"Blown Bottle", "Shakuhachi", "Whistle", "Ocarina",
	"Lead 1 Square", "Lead 2 Sawtooth", "Lead 3 Calliope", "Lead 4 Chiff",
	"Lead 5 Charang", "Lead 6 Voice", "Lead 7 Fifths", "Lead 8 Bass+Lead",
	"Pad 1 New Age", "Pad 2 Warm", "Pad 3 Polysynth", "Pad 4 Choir",
	"Pad 5 Bowed", "Pad 6 Metallic", "Pad 7 Halo", "Pad 8 Sweep",
	"FX 1 Rain", "FX 2 Soundtrack", "FX 3 Crystal", "FX 4 Atmosphere",
	"FX 5 Brightness", "FX 6 Goblins", "FX 7 Echoes", "FX 8 Sci-fi",
	"Sitar", "Banjo", "Shamisen", "Koto",
	"Kalimba", "Bag Pipe", "Fiddle", "Shanai",
	"Tinkle Bell", "Agogo", "Steel Drums", "Woodblock",
	"Taiko Drum", "Melodic Tom", "Synth Drum", "Reverse Cymbal",
	"Guitar Fret Noise", "Breath Noise", "Seashore", "Bird Tweet",
	"Telephone Ring", "Helicopter", "Applause", "Gunshot",
}

// GMGroup describes one General MIDI family.
type GMGroup struct {
	ID       string
	Label    string
	Programs []int
}

// GMGroups is ordered by program block.
var GMGroups = []GMGroup{
	{"piano", "Piano", rangeTo(0, 7)},
	{"chromatic", "Chromatic Perc", rangeTo(8, 15)},
	{"organ", "Organ", rangeTo(16, 23)},
	{"guitar", "Guitar", rangeTo(24, 31)},
	{"bass", "Bass", rangeTo(32, 39)},
	{"strings", "Strings", rangeTo(40, 47)},
	{"ensemble", "Ensemble", rangeTo(48, 55)},
	{"brass", "Brass", rangeTo(56, 63)},
	{"reed", "Reed", rangeTo(64, 71)},
	{"pipe", "Pipe", rangeTo(72, 79)},
	{"synth_lead", "Synth Lead", rangeTo(80, 87)},
	{"synth_pad", "Synth Pad", rangeTo(88, 95)},
	{"synth_fx", "Synth FX", rangeTo(96, 103)},
	{"ethnic", "Ethnic", rangeTo(104, 111)},
	{"percussive", "Percussive", rangeTo(112, 119)},
	{"sfx", "Sound FX", rangeTo(120, 127)},
	{"drums", "Drums", []int{-1}},
}

func rangeTo(from, to int) []int {
	out := make([]int, 0, to-from+1)
	for i := from; i <= to; i++ {
		out = append(out, i)
	}
	return out
}

// GMName returns a program's display name.
func GMName(program int) string {
	if program < 0 || program >= len(GMNames) {
		return "Unknown"
	}
	return GMNames[program]
}

// GMGroupOf returns the family for a program (-1 means drums).
func GMGroupOf(program int, isDrum bool) string {
	if isDrum {
		return "drums"
	}
	if program < 0 {
		return "sfx"
	}
	return GMGroups[program/8].ID
}

// IsKeyboardGroup reports whether the group lights the piano keys and receives
// automatic hand assignment (pianos and organs).
func IsKeyboardGroup(group string) bool {
	return group == "piano" || group == "organ"
}

// IsKeyInstrument includes chromatic percussion for keyboard display, without
// consuming a piano-family colour slot.
func IsKeyInstrument(program int, isDrum bool) bool {
	if isDrum {
		return false
	}
	return program >= 0 && program <= 23
}

// IsRhythmGroup is the Focus-mode rhythm family.
func IsRhythmGroup(group string) bool {
	return group == "bass" || group == "drums"
}
