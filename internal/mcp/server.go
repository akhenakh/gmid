// Package mcp exposes gmid to LLM clients over the Model Context Protocol.
package mcp

import (
	"encoding/json"
	"net/http"
	"strings"

	sdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	shirei "go.hasen.dev/shirei"

	"github.com/akhenakh/gmid/internal/app"
)

// Version is the MCP server version.
const Version = "0.1.0"

// Server wraps the MCP SDK and the live application.
type Server struct {
	app *app.App
	srv *server.MCPServer
}

// New creates the MCP server and registers every gmid tool.
func New(a *app.App) *Server {
	s := &Server{app: a}
	s.srv = server.NewMCPServer(
		"gmid",
		Version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithInstructions(`gmid is a MIDI piano-learning app. Use the tools to inspect the open song,
control transport, edit tracks/notes, browse the library, audition chords and export audio.
All state changes are applied to the running app; there is one open song at a time.
Times are song seconds unless a tool says otherwise; ticks are MIDI ticks at the song's PPQ.`),
	)
	s.registerMeta()
	s.registerTransport()
	s.registerNotes()
	s.registerLibrary()
	s.registerHarmony()
	s.registerUI()
	return s
}

// SDK exposes the underlying MCP server (used by tests and embedders).
func (s *Server) SDK() *server.MCPServer { return s.srv }

// Serve starts Streamable HTTP at /mcp and SSE at /sse.
func (s *Server) Serve(addr, token string) error {
	streamable := server.NewStreamableHTTPServer(s.srv,
		server.WithEndpointPath("/mcp"),
	)
	sse := server.NewSSEServer(s.srv,
		server.WithBaseURL("http://"+addr),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", streamable)
	mux.Handle("/", sse)
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "gmid", "version": Version,
			"streamable_http": "/mcp", "sse": "/sse", "message": "/message",
		})
	})

	handler := http.Handler(mux)
	if token != "" {
		handler = authMiddleware(token, handler)
	}
	srv := &http.Server{Addr: addr, Handler: handler}
	return srv.ListenAndServe()
}

func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer "+token && r.URL.Query().Get("token") != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// withApp runs fn while the UI frame lock is held, so MCP reads/writes cannot
// race the render thread.
func (s *Server) withApp(fn func(*app.App) (*sdk.CallToolResult, error)) (*sdk.CallToolResult, error) {
	var res *sdk.CallToolResult
	var err error
	shirei.WithFrameLock(func() { res, err = fn(s.app) })
	return res, err
}

func jsonResult(v any) (*sdk.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return sdk.NewToolResultError(err.Error()), nil
	}
	return sdk.NewToolResultText(string(b)), nil
}

func errResult(msg string) (*sdk.CallToolResult, error) {
	return sdk.NewToolResultError(msg), nil
}

func readOnly(title, desc string) sdk.ToolOption {
	return func(t *sdk.Tool) {
		t.Annotations.Title = title
		t.Annotations.ReadOnlyHint = sdk.ToBoolPtr(true)
		t.Annotations.DestructiveHint = sdk.ToBoolPtr(false)
		t.Annotations.OpenWorldHint = sdk.ToBoolPtr(false)
		_ = desc
	}
}

func toolDesc(desc string) sdk.ToolOption {
	return sdk.WithDescription(desc)
}

func toolTitle(title string) sdk.ToolOption {
	return sdk.WithTitleAnnotation(title)
}

// description combines a title and a description.
func meta(title, desc string) []sdk.ToolOption {
	return []sdk.ToolOption{toolTitle(title), toolDesc(desc), readOnly(title, desc)}
}

// stringArg fetches a string argument or returns the default.
func stringArg(r sdk.CallToolRequest, key, def string) string { return r.GetString(key, def) }

// intArg fetches an integer argument or returns the default.
func intArg(r sdk.CallToolRequest, key string, def int) int { return r.GetInt(key, def) }

// floatArg fetches a numeric argument or returns the default.
func floatArg(r sdk.CallToolRequest, key string, def float64) float64 { return r.GetFloat(key, def) }

// boolArg fetches a boolean argument or returns the default.
func boolArg(r sdk.CallToolRequest, key string, def bool) bool { return r.GetBool(key, def) }

// requireText is a convenience for required string arguments.
func requireText(tool string, r sdk.CallToolRequest, key string) (string, *sdk.CallToolResult) {
	v, err := r.RequireString(key)
	if err != nil {
		return "", sdk.NewToolResultErrorf("%s: %s is required", tool, key)
	}
	return v, nil
}

// describeError formats a Go error for a tool result.
func describeError(prefix string, err error) string {
	return strings.TrimSpace(prefix + ": " + err.Error())
}
