// Package dspreset parses DecentSampler .dspreset presets and .dslibrary packs.
package dspreset

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/akhenakh/gmid/internal/config"
)

// Preset is a parsed DecentSampler instrument.
type Preset struct {
	Path          string
	BaseDir       string
	Name          string
	Volume        float64
	Pan           float64
	Tuning        float64
	PitchKeyTrack float64
	Groups        []*Group
	Effects       []Effect
	Controls      []Control
}

// Group is one DecentSampler group.
type Group struct {
	Index         int
	Name          string
	Enabled       bool
	Volume        float64
	Pan           float64
	AmpVelTrack   float64
	Tuning        float64
	PitchKeyTrack float64
	Amp           Envelope
	FilterCutoff  float64
	FilterQ       float64
	SeqMode       string
	SeqLength     int
	Samples       []*Sample
}

// Sample is one mapped sample zone.
type Sample struct {
	Path              string
	Root              int
	LoNote            int
	HiNote            int
	LoVel             int
	HiVel             int
	Volume            float64
	Pan               float64
	Tuning            float64
	PitchKeyTrack     float64
	Start             int
	End               int
	LoopEnabled       bool
	LoopStart         int
	LoopEnd           int
	LoopDuringRelease bool
	Trigger           string
	SeqMode           string
	SeqPosition       int
	SeqLength         int
	Amp               Envelope
	FilterCutoff      float64
	FilterQ           float64
	Tags              []string
	CCFilters         map[int][2]int
	Delay             float64
	GroupIndex        int
}

// Envelope is an amplitude envelope.
type Envelope struct {
	Enabled bool
	Delay   float64
	Attack  float64
	Hold    float64
	Decay   float64
	Sustain float64
	Release float64
}

// Effect is one global effect.
type Effect struct {
	Type   string
	Params map[string]string
}

// Control is one UI control exposed by the instrument.
type Control struct {
	Label     string
	Kind      string
	Min, Max  float64
	Value     float64
	Parameter string
	Level     string
	Position  int
}

type node struct {
	Name     string
	Attrs    map[string]string
	Children []*node
}

// Load parses a .dspreset file, or extracts and parses a .dslibrary pack.
func Load(path string) (*Preset, error) {
	presetPath := path
	if strings.EqualFold(filepath.Ext(path), ".dslibrary") {
		extracted, err := extractLibrary(path)
		if err != nil {
			return nil, err
		}
		presetPath = extracted
	}
	data, err := os.ReadFile(presetPath)
	if err != nil {
		return nil, err
	}
	root, err := parseXML(data)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(root.Name, "DecentSampler") {
		return nil, fmt.Errorf("not a DecentSampler preset")
	}
	p := &Preset{
		Path:          presetPath,
		BaseDir:       filepath.Dir(presetPath),
		Name:          strings.TrimSuffix(filepath.Base(presetPath), filepath.Ext(presetPath)),
		Volume:        1,
		PitchKeyTrack: 1,
	}
	p.Volume = attrVolume(root.Attrs, "volume", 1)
	if v, ok := root.Attrs["globalVolume"]; ok {
		p.Volume = parseVolume(v, 1)
	}
	p.Pan = attrFloat(root.Attrs, "globalPan", 0)
	p.Tuning = attrFloat(root.Attrs, "globalTuning", 0)

	groupsNode := child(root, "groups")
	if groupsNode == nil {
		return nil, fmt.Errorf("preset has no <groups>")
	}
	groupDefaults := groupsNode.Attrs
	groupVol := attrVolume(groupDefaults, "volume", 1)
	if v, ok := groupDefaults["globalVolume"]; ok {
		groupVol = parseVolume(v, 1)
	}
	globalPan := attrFloat(groupDefaults, "globalPan", 0)
	globalTuning := attrFloat(groupDefaults, "globalTuning", 0)
	groupPitchTrack := attrFloat(groupDefaults, "pitchKeyTrack", 1)
	groupEnv := envelopeFrom(groupDefaults, Envelope{Enabled: true, Sustain: 1, Release: 0.1})
	groupFilter := attrFloat(groupDefaults, "filterCutoff", 0)
	groupQ := attrFloat(groupDefaults, "filterResonance", 0)
	groupSeqMode := groupDefaults["seqMode"]
	groupSeqLen := attrInt(groupDefaults, "seqLength", 0)

	for _, g := range children(groupsNode, "group") {
		gv := groupVol * attrVolume(g.Attrs, "volume", 1)
		if v, ok := g.Attrs["groupVolume"]; ok {
			gv = groupVol * parseVolume(v, 1)
		}
		if !attrBool(g.Attrs, "enabled", true) {
			continue
		}
		grp := &Group{
			Index:         len(p.Groups),
			Name:          g.Attrs["name"],
			Enabled:       true,
			Volume:        gv,
			Pan:           attrFloat(g.Attrs, "groupPan", 0),
			AmpVelTrack:   attrFloat(g.Attrs, "ampVelTrack", 1),
			Tuning:        globalTuning + attrFloat(g.Attrs, "groupTuning", 0),
			PitchKeyTrack: attrFloat(g.Attrs, "pitchKeyTrack", groupPitchTrack),
			Amp:           envelopeFrom(g.Attrs, groupEnv),
			FilterCutoff:  attrFloat(g.Attrs, "filterCutoff", groupFilter),
			FilterQ:       attrFloat(g.Attrs, "filterResonance", groupQ),
			SeqMode:       pick(g.Attrs, "seqMode", groupSeqMode),
			SeqLength:     attrInt(g.Attrs, "seqLength", groupSeqLen),
		}
		// group pan is additive to the sample pan; keep as offset
		for _, s := range children(g, "sample") {
			sample := parseSample(s, grp, g.Attrs, groupDefaults, globalPan, p.BaseDir)
			if sample != nil {
				grp.Samples = append(grp.Samples, sample)
			}
		}
		if len(grp.Samples) > 0 {
			p.Groups = append(p.Groups, grp)
		}
	}
	if effects := child(root, "effects"); effects != nil {
		for _, e := range children(effects, "effect") {
			p.Effects = append(p.Effects, Effect{Type: e.Attrs["type"], Params: e.Attrs})
		}
	}
	if ui := child(root, "ui"); ui != nil {
		p.Controls = parseControls(ui)
	}
	if len(p.Groups) == 0 {
		return nil, fmt.Errorf("preset has no samples")
	}
	return p, nil
}

func parseSample(s *node, grp *Group, groupAttrs, groupsAttrs map[string]string, globalPan float64, baseDir string) *Sample {
	path := s.Attrs["path"]
	if path == "" {
		return nil
	}
	path = filepath.FromSlash(strings.TrimSpace(path))
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	smp := &Sample{
		Path:              path,
		Root:              attrInt(s.Attrs, "rootNote", 60),
		LoNote:            attrInt(s.Attrs, "loNote", 0),
		HiNote:            attrInt(s.Attrs, "hiNote", 127),
		LoVel:             attrInt(s.Attrs, "loVel", 0),
		HiVel:             attrInt(s.Attrs, "hiVel", 127),
		Volume:            attrVolume(s.Attrs, "volume", 1),
		Pan:               attrFloat(s.Attrs, "pan", attrFloat(groupAttrs, "pan", attrFloat(groupsAttrs, "pan", 0))) + globalPan + grp.Pan,
		Tuning:            grp.Tuning + attrFloat(s.Attrs, "tuning", 0),
		PitchKeyTrack:     attrFloat(s.Attrs, "pitchKeyTrack", grp.PitchKeyTrack),
		Start:             attrInt(s.Attrs, "start", 0),
		End:               attrInt(s.Attrs, "end", -1),
		LoopStart:         attrInt(s.Attrs, "loopStart", 0),
		LoopEnd:           attrInt(s.Attrs, "loopEnd", -1),
		LoopEnabled:       attrBool(s.Attrs, "loopEnabled", s.Attrs["loopStart"] != "" || s.Attrs["loopEnd"] != ""),
		LoopDuringRelease: attrBool(s.Attrs, "loopDuringRelease", true),
		Trigger:           pick(s.Attrs, "trigger", pick(groupAttrs, "trigger", "attack")),
		SeqMode:           pick(s.Attrs, "seqMode", grp.SeqMode),
		SeqPosition:       attrInt(s.Attrs, "seqPosition", 1),
		SeqLength:         attrInt(s.Attrs, "seqLength", grp.SeqLength),
		Amp:               envelopeFrom(s.Attrs, grp.Amp),
		FilterCutoff:      attrFloat(s.Attrs, "filterCutoff", grp.FilterCutoff),
		FilterQ:           attrFloat(s.Attrs, "filterResonance", grp.FilterQ),
		Delay:             attrFloat(s.Attrs, "delay", 0),
		GroupIndex:        grp.Index,
	}
	if tags := s.Attrs["tags"]; tags != "" {
		smp.Tags = strings.Split(tags, ",")
	}
	smp.CCFilters = ccFilters(s.Attrs, groupAttrs, groupsAttrs)
	return smp
}

func parseControls(ui *node) []Control {
	var out []Control
	var walk func(n *node)
	walk = func(n *node) {
		switch n.Name {
		case "labeled-knob", "control":
			c := Control{
				Label: n.Attrs["label"], Kind: n.Name,
				Min: attrFloat(n.Attrs, "minValue", 0), Max: attrFloat(n.Attrs, "maxValue", 1),
				Value: attrFloat(n.Attrs, "value", 0),
			}
			if b := child(n, "binding"); b != nil {
				c.Parameter = b.Attrs["parameter"]
				c.Level = b.Attrs["level"]
				c.Position = attrInt(b.Attrs, "position", 0)
			}
			out = append(out, c)
		}
		for _, ch := range n.Children {
			walk(ch)
		}
	}
	walk(ui)
	return out
}

func envelopeFrom(attrs map[string]string, base Envelope) Envelope {
	e := base
	if v, ok := attrs["ampEnvEnabled"]; ok {
		e.Enabled = parseBool(v, e.Enabled)
	}
	e.Delay = attrFloat(attrs, "ampEnvDelay", e.Delay)
	e.Attack = attrFloat(attrs, "attack", attrFloat(attrs, "ampEnvAttack", e.Attack))
	e.Hold = attrFloat(attrs, "ampEnvHold", e.Hold)
	e.Decay = attrFloat(attrs, "decay", attrFloat(attrs, "ampEnvDecay", e.Decay))
	e.Sustain = attrFloat(attrs, "sustain", attrFloat(attrs, "ampEnvSustain", e.Sustain))
	e.Release = attrFloat(attrs, "release", attrFloat(attrs, "ampEnvRelease", e.Release))
	return e
}

func ccFilters(attrs ...map[string]string) map[int][2]int {
	out := map[int][2]int{}
	for _, a := range attrs {
		for k, v := range a {
			if !strings.HasPrefix(k, "loCC") {
				continue
			}
			num, err := strconv.Atoi(strings.TrimPrefix(k, "loCC"))
			if err != nil {
				continue
			}
			hiKey := "hiCC" + strconv.Itoa(num)
			hi := 127
			if h, ok := a[hiKey]; ok {
				hi = int(parseFloat(h, 127))
			}
			out[num] = [2]int{int(parseFloat(v, 0)), hi}
		}
	}
	return out
}

func parseXML(data []byte) (*node, error) {
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	dec.Strict = false
	var stack []*node
	var root *node
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			n := &node{Name: t.Name.Local, Attrs: map[string]string{}}
			for _, a := range t.Attr {
				n.Attrs[a.Name.Local] = a.Value
			}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, n)
			} else if root == nil {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		return nil, fmt.Errorf("empty XML")
	}
	return root, nil
}

func child(n *node, name string) *node {
	for _, c := range n.Children {
		if strings.EqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}

func children(n *node, name string) []*node {
	var out []*node
	for _, c := range n.Children {
		if strings.EqualFold(c.Name, name) {
			out = append(out, c)
		}
	}
	return out
}

func pick(attrs map[string]string, key, def string) string {
	if v, ok := attrs[key]; ok && v != "" {
		return v
	}
	return def
}

func attrFloat(attrs map[string]string, key string, def float64) float64 {
	if v, ok := attrs[key]; ok {
		return parseFloat(v, def)
	}
	return def
}

func attrInt(attrs map[string]string, key string, def int) int {
	if v, ok := attrs[key]; ok {
		return int(parseFloat(v, float64(def)))
	}
	return def
}

func attrBool(attrs map[string]string, key string, def bool) bool {
	if v, ok := attrs[key]; ok {
		return parseBool(v, def)
	}
	return def
}

func attrVolume(attrs map[string]string, key string, def float64) float64 {
	if v, ok := attrs[key]; ok {
		return parseVolume(v, def)
	}
	return def
}

func parseVolume(v string, def float64) float64 {
	v = strings.TrimSpace(v)
	if strings.HasSuffix(strings.ToLower(v), "db") {
		db := parseFloat(strings.TrimSpace(v[:len(v)-2]), 0)
		return math.Pow(10, db/20)
	}
	return parseFloat(v, def)
}

func parseFloat(v string, def float64) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return def
	}
	return f
}

func parseBool(v string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	}
	return def
}

func extractLibrary(path string) (string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = zr.Close() }()
	root := filepath.Join(config.DataDir(), "decent", strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	if err := os.RemoveAll(root); err != nil {
		return "", err
	}
	var presetPath string
	for _, f := range zr.File {
		name := filepath.Clean(f.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}
		dest := filepath.Join(root, name)
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(dest, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", err
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out, err := os.Create(dest)
		if err != nil {
			_ = rc.Close()
			return "", err
		}
		_, err = io.Copy(out, rc)
		_ = out.Close()
		_ = rc.Close()
		if err != nil {
			return "", err
		}
		if strings.EqualFold(filepath.Ext(dest), ".dspreset") && presetPath == "" {
			presetPath = dest
		}
	}
	if presetPath == "" {
		return "", fmt.Errorf("no .dspreset inside library")
	}
	return presetPath, nil
}
