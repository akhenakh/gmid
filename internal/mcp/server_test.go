package mcp

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	sdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/akhenakh/gmid/internal/app"
	"github.com/akhenakh/gmid/internal/audio"
	"github.com/akhenakh/gmid/internal/config"
	"github.com/akhenakh/gmid/internal/smf"
)

func testServer(t *testing.T) (*client.Client, *app.App, func()) {
	t.Helper()
	t.Setenv("GMID_CONFIG", filepath.Join(t.TempDir(), "prefs.json"))
	demo := "../../assets/demo/Scott Joplin - Entertainer.mid"
	if _, err := os.Stat(demo); err != nil {
		t.Skip("demo missing")
	}
	prefs := config.Defaults()
	prefs.LibraryFolder = t.TempDir()
	a := app.New(prefs, "../../assets", "")
	a.Engine = audio.New(48000)
	s, err := smf.ReadFile(demo)
	if err != nil {
		t.Fatal(err)
	}
	a.InstallSong(s)

	srv := New(a)
	hs := server.NewStreamableHTTPServer(srv.SDK(), server.WithEndpointPath("/mcp"))
	httpSrv := httptest.NewServer(hs)
	tr, err := transport.NewStreamableHTTP(httpSrv.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	c := client.NewClient(tr)
	ctx := context.Background()
	if err := c.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Initialize(ctx, sdk.InitializeRequest{Params: sdk.InitializeParams{
		ProtocolVersion: sdk.LATEST_PROTOCOL_VERSION,
		ClientInfo:      sdk.Implementation{Name: "test", Version: "0"},
	}}); err != nil {
		t.Fatal(err)
	}
	return c, a, func() {
		_ = tr.Close()
		httpSrv.Close()
	}
}

func call(t *testing.T, c *client.Client, name string, args map[string]any) string {
	t.Helper()
	res, err := c.CallTool(context.Background(), sdk.CallToolRequest{
		Params: sdk.CallToolParams{Name: name, Arguments: args},
	})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("%s: %s", name, firstText(res))
	}
	return firstText(res)
}

func firstText(res *sdk.CallToolResult) string {
	for _, c := range res.Content {
		if t, ok := c.(sdk.TextContent); ok {
			return t.Text
		}
	}
	return ""
}

func TestMCPStatusAndSong(t *testing.T) {
	c, _, closeFn := testServer(t)
	defer closeFn()
	status := call(t, c, "gmid_status", map[string]any{})
	if !contains(status, "gmid") {
		t.Fatalf("status: %s", status)
	}
	song := call(t, c, "gmid_song", map[string]any{})
	if !contains(song, "Entertainer") {
		t.Fatalf("song: %s", song)
	}
}

func TestMCPTransportAndTracks(t *testing.T) {
	c, a, closeFn := testServer(t)
	defer closeFn()
	call(t, c, "gmid_transport", map[string]any{"action": "seek", "time": 3.0})
	if a.Transport.Time < 2.99 || a.Transport.Time > 3.01 {
		t.Fatalf("seek: %v", a.Transport.Time)
	}
	call(t, c, "gmid_transport", map[string]any{"action": "speed", "speed": 0.75})
	if a.Transport.Speed != 0.75 {
		t.Fatalf("speed: %v", a.Transport.Speed)
	}
	call(t, c, "gmid_set_track", map[string]any{"index": 0, "muted": true})
	if !a.Song.Tracks[0].Muted {
		t.Fatal("track not muted")
	}
	call(t, c, "gmid_set_track", map[string]any{"index": 0, "muted": false, "color": "#ffffff"})
	if a.Song.Tracks[0].Color != "#ffffff" {
		t.Fatal("colour not applied")
	}
}

func TestMCPNoteEditor(t *testing.T) {
	c, a, closeFn := testServer(t)
	defer closeFn()
	call(t, c, "gmid_note_edit", map[string]any{"action": "enter", "track": 0, "tool": "select"})
	if !a.Edit.Active {
		t.Fatal("editor not active")
	}
	notes := call(t, c, "gmid_notes", map[string]any{"track": 0, "limit": 5})
	if !contains(notes, `"id"`) {
		t.Fatalf("notes: %s", notes)
	}
	before := len(a.Song.Tracks[0].Notes)
	call(t, c, "gmid_note_edit", map[string]any{
		"action": "add", "key": 64, "tick": 960, "duration_ticks": 480, "velocity": 100,
	})
	if len(a.Song.Tracks[0].Notes) != before+1 {
		t.Fatalf("add: %d -> %d", before, len(a.Song.Tracks[0].Notes))
	}
	call(t, c, "gmid_note_edit", map[string]any{"action": "undo"})
	if len(a.Song.Tracks[0].Notes) != before {
		t.Fatal("undo failed")
	}
	call(t, c, "gmid_note_edit", map[string]any{"action": "exit"})
}

func TestMCPDecentSampler(t *testing.T) {
	c, a, closeFn := testServer(t)
	defer closeFn()
	res := call(t, c, "gmid_decent", map[string]any{
		"action": "load", "path": "../../assets/decent/demo-piano.dspreset",
	})
	if !contains(res, `"loaded": true`) {
		t.Fatalf("decent load: %s", res)
	}
	if info, ok := a.DecentStatus(); !ok || info.Zones < 5 {
		t.Fatalf("status: %+v ok=%v", info, ok)
	}
	call(t, c, "gmid_decent", map[string]any{"action": "unload"})
	if _, ok := a.DecentStatus(); ok {
		t.Fatal("decent not unloaded")
	}
}

func TestMCPHarmony(t *testing.T) {
	c, _, closeFn := testServer(t)
	defer closeFn()
	chord := call(t, c, "gmid_chord", map[string]any{"root": 0, "suffix": "maj7", "octave": 4})
	if !contains(chord, "Cmaj7") {
		t.Fatalf("chord: %s", chord)
	}
	scale := call(t, c, "gmid_scale", map[string]any{"action": "chords", "root": 0, "scale": "major"})
	if !contains(scale, "C") || !contains(scale, "Dm") {
		t.Fatalf("scale: %s", scale)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
