package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"pairti-lense/internal/agent"
)

func TestMCPReadsTheSelectionAndEditsTheBuffer(t *testing.T) {
	s := testShell()
	body := "# Title\n\nhello world\n"
	path := openNote(t, s, body)
	at := strings.Index(s.buf.Text(), "hello")
	s.buf.Select(at, at+len("hello"))
	s.agent = agent.NewHub(func() { s.drainAgent() })
	srv, rawURL, err := agent.Serve(s.agent, "tok", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             rawURL,
		DisableStandaloneSSE: true,
		HTTPClient: &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
			r = r.Clone(r.Context())
			r.Header.Set("Authorization", "Bearer tok")
			return http.DefaultTransport.RoundTrip(r)
		})},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "document",
		Arguments: map[string]any{"radius": 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	snap := decodeSnap(t, res)
	if snap.Path != path || snap.Title != "Title" || snap.Selection != "hello" || snap.SelectedKind != "selection" {
		t.Fatalf("snap path %q title %q sel %q kind %q", snap.Path, snap.Title, snap.Selection, snap.SelectedKind)
	}
	if snap.Text != body || !strings.Contains(snap.Around, "2|") || snap.Mode != "Formatted" {
		t.Fatalf("text %q around %q mode %q", snap.Text, snap.Around, snap.Mode)
	}
	found := false
	for _, h := range snap.Outline {
		if h.Text == "Title" && h.Line == 1 {
			found = true
		}
	}
	if !found || len(snap.Tabs) != 1 || !snap.Tabs[0].Focused || snap.Tabs[0].Path != path {
		t.Fatalf("outline %v tabs %v", snap.Outline, snap.Tabs)
	}

	edited, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "apply_edit",
		Arguments: map[string]any{
			"path": path, "span": true, "from": at, "to": at + len("hello"),
			"text": "goodbye", "expect": "hello",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if edited.IsError {
		t.Fatalf("edit %#v", edited.Content)
	}
	if s.buf.Text() != "# Title\n\ngoodbye world\n" || !s.buf.Dirty() {
		t.Fatalf("text %q dirty %v", s.buf.Text(), s.buf.Dirty())
	}
	disk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(disk) != body {
		t.Fatalf("file changed %q", disk)
	}
	if s.line3.Get() != "Unsaved" {
		t.Fatalf("line3 %q", s.line3.Get())
	}
	if !s.buf.Undo() || s.buf.Text() != body || s.buf.Dirty() {
		t.Fatalf("undo %q dirty %v", s.buf.Text(), s.buf.Dirty())
	}

	if err := os.WriteFile(path, []byte("# Disk\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !s.onDisk("# Disk\n") || s.buf.Text() != "# Disk\n" || s.buf.Dirty() {
		t.Fatalf("disk text %q dirty %v", s.buf.Text(), s.buf.Dirty())
	}
}

func TestMCPApplyRefusesADifferentFile(t *testing.T) {
	s := testShell()
	openNote(t, s, "# Title\n")
	s.agent = agent.NewHub(func() { s.drainAgent() })
	_, err := s.agent.Edit(context.Background(), agent.ApplyIn{
		Path: "/tmp/other.md", ReplaceAll: true, Text: "nope",
	})
	if err == nil || !strings.Contains(err.Error(), "focused document") {
		t.Fatalf("err %v", err)
	}
	if s.buf.Text() != "# Title\n" || s.buf.Dirty() {
		t.Fatalf("text %q dirty %v", s.buf.Text(), s.buf.Dirty())
	}
}

func decodeSnap(t *testing.T, res *mcp.CallToolResult) agent.Snapshot {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool error %#v", res.Content)
	}
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var snap agent.Snapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		t.Fatal(err)
	}
	return snap
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
