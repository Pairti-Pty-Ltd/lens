package report

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStagingRejectsMissingOrCorruptAsset(t *testing.T) {
	body := zipPandoc(t, "pandoc fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer server.Close()
	for _, tc := range []struct{ name, sum, want string }{
		{"checksum", strings.Repeat("0", 64), "checksum mismatch"},
		{"missing tool", fmt.Sprintf("%x", sha256.Sum256(body)), "archive had no compiler"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := archiveSpec{URL: server.URL, SHA: tc.sum, File: "tools.zip", Label: "typst", Bin: "typst", Names: []string{"typst"}}
			err := stageMacTool(t.Context(), t.TempDir(), spec)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want %q", err, tc.want)
			}
		})
	}
}
