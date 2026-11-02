package ui

import "strings"

type settingRef struct {
	name  string
	group string
}

var settingIndex = []settingRef{
	{"Sound engine", "audio"}, {"Sound fonts", "audio"}, {"Master volume", "audio"},
	{"Auto-level", "audio"}, {"Metronome", "playback"}, {"Count-in", "playback"},
	{"Count-in bars", "playback"}, {"Skip empty bars", "playback"}, {"Tap Tempo", "playback"},
	{"Tempo warnings", "playback"}, {"Loop region", "playback"}, {"Key range", "keyboard"},
	{"Note names", "keyboard"}, {"Octaves", "keyboard"}, {"Note naming", "keyboard"},
	{"Accidentals", "keyboard"}, {"Roll zoom", "keyboard"}, {"Bar numbers", "keyboard"},
	{"Visual effects", "keyboard"}, {"Hand assignment", "practice"}, {"Hand badges", "practice"},
	{"Note editor", "practice"}, {"Step Through", "practice"}, {"Chord Prompter", "practice"},
	{"Focus mode", "practice"}, {"Warm theme", "appearance"}, {"Tooltips", "appearance"},
	{"Library folder", "appearance"}, {"MIDI devices", "midi"}, {"Touch response", "midi"},
	{"Pedals & controls", "midi"}, {"Double-note filter", "midi"}, {"Live response", "midi"},
	{"Delay check", "midi"},
}

func searchSettingNames(query string) []settingRef {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}
	var out []settingRef
	for _, s := range settingIndex {
		if strings.Contains(strings.ToLower(s.name), q) {
			out = append(out, s)
		}
	}
	return out
}
