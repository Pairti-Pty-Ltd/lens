package report

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTransferLineNamesTheDownload(t *testing.T) {
	if got := transferLine("Typst 0.15.1", 0, 0); got != "Downloading Typst 0.15.1" {
		t.Fatalf("idle %q", got)
	}
	if got := transferLine("Typst 0.15.1", 6<<20, 14<<20); !strings.Contains(got, "6.0 MB / 14.0 MB") {
		t.Fatalf("progress %q", got)
	}
}

func TestFetchPublishesDownloadStatus(t *testing.T) {
	var got []string
	WatchTransfer(func() {
		if msg, ok := Transfer(); ok {
			got = append(got, msg)
		}
	})
	t.Cleanup(func() {
		WatchTransfer(nil)
		SetTransfer("")
		Transfer()
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not an archive"))
	}))
	defer srv.Close()
	_, err := fetchTypst(t.Context(), typstFetch{
		URL: srv.URL, SHA: strings.Repeat("ab", 32), File: "typst.tar.xz", Dest: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if len(got) < 2 || got[0] != "Downloading Typst "+typstVersion {
		t.Fatalf("status %q", got)
	}
	if !strings.Contains(got[len(got)-1], "checksum mismatch") {
		t.Fatalf("error status %q", got)
	}
}
