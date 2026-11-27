package agent

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultAddr is the loopback port Lens keeps across launches.
const DefaultAddr = "127.0.0.1:47328"

// Endpoint is the local MCP address and bearer token.
type Endpoint struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// LoadEndpoint reads the endpoint file. A missing file is an empty endpoint.
func LoadEndpoint(path string) (Endpoint, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Endpoint{}, nil
		}
		return Endpoint{}, err
	}
	var ep Endpoint
	if err := json.Unmarshal(b, &ep); err != nil {
		return Endpoint{}, err
	}
	return ep, nil
}

// SaveEndpoint writes the endpoint so only the user can read the token.
func SaveEndpoint(path string, ep Endpoint) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ep, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}

// NewToken is a random bearer token.
func NewToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// PreferredAddr is the host:port from a previously saved loopback URL.
// A missing or non-loopback URL uses DefaultAddr, not a fresh port.
func PreferredAddr(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return DefaultAddr
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil || port == "" || (host != "127.0.0.1" && host != "localhost") {
		return DefaultAddr
	}
	return "127.0.0.1:" + port
}

// Serve listens on addr, which must be 127.0.0.1, and serves the document tools.
// The returned URL ends in /mcp. Requests without the bearer token are refused.
func Serve(h *Hub, token, addr string) (*http.Server, string, error) {
	if h == nil {
		return nil, "", errors.New("mcp hub is nil")
	}
	if token == "" {
		return nil, "", errors.New("mcp token is empty")
	}
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil || host != "127.0.0.1" {
		return nil, "", fmt.Errorf("mcp listens on 127.0.0.1 only, not %s", addr)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", Handler(h))
	srv := &http.Server{Handler: auth(token, mux)}
	go func() {
		err := srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "mcp: %v\n", err)
		}
	}()
	return srv, "http://" + ln.Addr().String() + "/mcp", nil
}

// Handler is the streamable HTTP MCP endpoint, without authentication.
func Handler(h *Hub) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pairti-lense",
		Title:   "Pairti Lens",
		Version: "0.1.0",
	}, nil)
	ro := true
	closed := false
	mcp.AddTool(server, &mcp.Tool{
		Name:        "document",
		Description: "Read the focused Pairti Lens document: path, title, dirty state, formatted or raw mode, caret, selection or current line, outline, open tabs, a window around the caret, and the full text.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: ro, OpenWorldHint: &closed, Title: "Read document"},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ReadIn) (*mcp.CallToolResult, Snapshot, error) {
		snap, err := h.Read(ctx, in)
		if err != nil {
			return nil, Snapshot{}, err
		}
		return nil, snap, nil
	})
	mcp.AddTool(server, &mcp.Tool{
		Name:        "apply_edit",
		Description: "Replace the focused document or a rune span in it. The change is one undo step and stays unsaved. This does not write the file. Pass expect as the text you read so a newer edit is refused. To update from disk, write the file yourself; a clean buffer reloads and a dirty buffer asks.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, OpenWorldHint: &closed, Title: "Edit document"},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in ApplyIn) (*mcp.CallToolResult, ApplyOut, error) {
		out, err := h.Edit(ctx, in)
		if err != nil {
			return nil, ApplyOut{}, err
		}
		return nil, out, nil
	})
	server.AddResource(&mcp.Resource{
		URI:         "pairti://document",
		Name:        "document",
		Title:       "Focused document",
		Description: "Text of the document open in Pairti Lens.",
		MIMEType:    "text/plain",
	}, func(ctx context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		snap, err := h.Read(ctx, ReadIn{})
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI:      "pairti://document",
			MIMEType: snap.MIME,
			Text:     snap.Text,
		}}}, nil
	})
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
}

func auth(token string, next http.Handler) http.Handler {
	want := "Bearer " + token
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if len(got) != len(want) || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
