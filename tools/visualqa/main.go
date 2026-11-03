// Command visualqa sends a rendered gmid screenshot to the local Qwen vision
// model and prints its textual verdict. Images never enter the coding model's
// context; only this output does.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultURL = "http://127.0.0.1:11434/v1/chat/completions"

type request struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type message struct {
	Role    string    `json:"role"`
	Content []content `json:"content"`
}

type content struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *imageURL `json:"image_url,omitempty"`
}

type imageURL struct {
	URL string `json:"url"`
}

type response struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func main() {
	var (
		imagePath = flag.String("image", "", "PNG screenshot to review")
		scene     = flag.String("scene", "shell", "scene name for the checklist")
		model     = flag.String("model", "Qwen3.8-27B-VISION", "vision model")
		url       = flag.String("url", defaultURL, "chat completions endpoint")
		extra     = flag.String("prompt", "", "extra instructions")
	)
	flag.Parse()
	if *imagePath == "" {
		fmt.Fprintln(os.Stderr, "visualqa: -image is required")
		os.Exit(2)
	}
	data, err := os.ReadFile(*imagePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "visualqa:", err)
		os.Exit(1)
	}
	prompt := buildPrompt(*scene, *extra)
	reqBody := request{
		Model:       *model,
		Temperature: 0.1,
		MaxTokens:   4096,
		Messages: []message{{
			Role: "user",
			Content: []content{
				{Type: "text", Text: prompt},
				{Type: "image_url", ImageURL: &imageURL{URL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)}},
			},
		}},
	}
	body, _ := json.Marshal(reqBody)

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Post(*url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(os.Stderr, "visualqa:", err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "visualqa: %s: %s\n", resp.Status, strings.TrimSpace(string(raw)))
		os.Exit(1)
	}
	var out response
	if err := json.Unmarshal(raw, &out); err != nil {
		fmt.Fprintln(os.Stderr, "visualqa: decode:", err)
		os.Exit(1)
	}
	if out.Error != nil {
		fmt.Fprintln(os.Stderr, "visualqa:", out.Error.Message)
		os.Exit(1)
	}
	if len(out.Choices) == 0 {
		fmt.Fprintln(os.Stderr, "visualqa: empty response")
		os.Exit(1)
	}
	msg := out.Choices[0].Message.Content
	if strings.TrimSpace(msg) == "" {
		fmt.Fprintf(os.Stderr, "visualqa: empty content; raw=%s\n", string(raw))
		os.Exit(1)
	}
	fmt.Printf("scene: %s\nimage: %s\n\n%s\n", *scene, filepath.Base(*imagePath), msg)
}

func buildPrompt(scene, extra string) string {
	common := `You are reviewing a screenshot of "gmid", a Go desktop MIDI learning app (a piano roll with a keyboard below). Judge only what is visible. Check:
- layout hierarchy, spacing and alignment;
- piano-roll notes aligned to keyboard key columns;
- labels legible, contrast sufficient, nothing clipped or overlapping;
- empty states and panel headers clear.
Return a short JSON object on the last line: {"pass": true|false, "issues": ["..."]}. List only concrete visual defects, not feature requests.`
	specific := map[string]string{
		"shell":          "This is the empty shell: expect a top bar, left setup panel, empty piano roll and keyboard.",
		"roll":           "This is a loaded song: the piano roll is a waterfall, so the playhead is the horizontal amber line near the bottom; note rectangles fall downward toward the keyboard. Check bar lines, notes, and keyboard alignment.",
		"mixer":          "This is the console mixer: check channel strips, sliders, meters and header spacing.",
		"settings":       "This is the settings window: check group headings, row alignment, control widths and scrolling.",
		"library":        "This is the file browser: check the path field, file list and demo buttons.",
		"starmap":        "This is the Scale Explorer chord-star view: a twelve-node circle with a highlighted scale ring, a highlighted triad star and note labels. Check node/ring alignment, legible labels, sufficient contrast, and that nothing is clipped.",
		"icon":           "This is the gmid app icon: an amber/orange rounded square with a large dark lowercase Inter Bold 'g' centered on it. Check that the glyph is optically centred, not clipped, consistent in weight, and that the corner radius and flat background look clean at icon scale.",
		"scale-explorer": "This is the Scale Explorer with the circle of fifths: check the circle geometry, selected-key outline, chord cards and toggle row.",
		"keys-floating":  "This is a floating keyboard panel. It intentionally overlays the roll; check that the panel is contained in the window, its header and keys are readable, and its resize handle is visible — overlap with the background is expected.",
		"presentation":   "This is presentation mode: only the roll and keyboard should be visible.",
	}[scene]
	if specific != "" {
		common += "\nScene: " + specific
	}
	if extra != "" {
		common += "\nExtra: " + extra
	}
	return common
}
