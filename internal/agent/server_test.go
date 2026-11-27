package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"pairti-lense/internal/edit"
)

func TestEndpointFileKeepsTheTokenPrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairti-lense", "mcp.json")
	if err := SaveEndpoint(path, Endpoint{URL: "http://127.0.0.1:9/mcp", Token: "secret"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	ep, err := LoadEndpoint(path)
	if err != nil || ep.Token != "secret" {
		t.Fatalf("ep %+v err %v", ep, err)
	}
	if PreferredAddr(ep.URL) != "127.0.0.1:9" {
		t.Fatalf("addr %s", PreferredAddr(ep.URL))
	}
	if PreferredAddr("http://0.0.0.0:9/mcp") != DefaultAddr {
		t.Fatal("non-loopback address was kept")
	}
	if PreferredAddr("") != DefaultAddr {
		t.Fatal("an empty address picked a fresh port")
	}
}

func TestServeRequiresTheBearerToken(t *testing.T) {
	h := NewHub(nil)
	buf := edit.New("# Title\n\nhello\n")
	h = NewHub(func() { answer(h, buf) })
	srv, rawURL, err := Serve(h, "tok", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })

	req, err := http.NewRequest(http.MethodPost, rawURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil).Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:             rawURL,
		DisableStandaloneSSE: true,
		HTTPClient:           &http.Client{Transport: bearer("tok")},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "document",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("tool error %#v", res.Content)
	}
	var snap Snapshot
	b, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Text != buf.Text() {
		t.Fatalf("text %q", snap.Text)
	}
	at := strings.Index(buf.Text(), "hello")
	edited, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "apply_edit",
		Arguments: map[string]any{
			"span": true, "from": at, "to": at + len("hello"), "text": "there", "expect": "hello",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if edited.IsError {
		t.Fatalf("edit error %#v", edited.Content)
	}
	if buf.Text() != "# Title\n\nthere\n" || !buf.Dirty() {
		t.Fatalf("text %q dirty %v", buf.Text(), buf.Dirty())
	}
	got, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "pairti://document"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Contents) != 1 || got.Contents[0].Text != buf.Text() {
		t.Fatalf("resource %#v", got.Contents)
	}
	if _, _, err := Serve(NewHub(nil), "tok", "0.0.0.0:0"); err == nil {
		t.Fatal("bound a public address")
	}
}

func answer(h *Hub, b *edit.Buffer) {
	for {
		req, ok := h.Take()
		if !ok {
			return
		}
		switch req.Kind {
		case KindRead:
			text := b.Text()
			req.Reply(Response{Snap: Snapshot{
				Text:    text,
				Outline: []Heading{},
				Tabs:    []Tab{},
				MIME:    "text/markdown",
			}})
		case KindEdit:
			_, err := Apply(b, req.Edit)
			req.Reply(Response{Err: err, Applied: ApplyOut{Dirty: b.Dirty()}})
		default:
			req.Reply(Response{Err: errUnknown})
		}
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (f roundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func bearer(token string) http.RoundTripper {
	return roundTripper(func(r *http.Request) (*http.Response, error) {
		r = r.Clone(r.Context())
		r.Header.Set("Authorization", "Bearer "+token)
		return http.DefaultTransport.RoundTrip(r)
	})
}

var errUnknown = errString("unknown request")

type errString string

func (e errString) Error() string { return string(e) }
