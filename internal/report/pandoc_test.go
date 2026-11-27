package report

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPandocFetchInstallsFromAChecksummedZip(t *testing.T) {
	body := zipPandoc(t, "#!/bin/sh\nprintf 'pandoc 3.14\\n'\n")
	sum := sha256.Sum256(body)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write(body)
	}))
	defer srv.Close()
	dest := t.TempDir()
	got, err := fetchBinary(t.Context(), archiveSpec{
		URL: srv.URL + "/pandoc.zip", SHA: hex.EncodeToString(sum[:]),
		File: "pandoc.zip", Dest: dest, Label: "pandoc", Version: pandocVersion,
		Names: []string{"pandoc", "pandoc.exe"}, Bin: pandocName(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dest, pandocName()) {
		t.Fatalf("installed %s", got)
	}
	out, err := exec.Command(got, "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(out), pandocVersion) {
		t.Fatalf("version %q err %v", out, err)
	}
	if _, err := fetchBinary(t.Context(), archiveSpec{
		URL: srv.URL + "/pandoc.zip", SHA: hex.EncodeToString(sum[:]),
		File: "pandoc.zip", Dest: dest, Label: "pandoc", Version: pandocVersion,
		Names: []string{"pandoc"}, Bin: pandocName(),
	}); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("downloads %d", hits)
	}
}

func TestPandocFetchRejectsABadChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not a zip"))
	}))
	defer srv.Close()
	dest := t.TempDir()
	_, err := fetchBinary(t.Context(), archiveSpec{
		URL: srv.URL, SHA: strings.Repeat("0", 64),
		File: "pandoc.zip", Dest: dest, Label: "pandoc", Version: pandocVersion,
		Names: []string{"pandoc"}, Bin: pandocName(),
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dest, pandocName())); !os.IsNotExist(statErr) {
		t.Fatal("a bad download was kept")
	}
}

func zipPandoc(t *testing.T, script string) []byte {
	t.Helper()
	buf := filepath.Join(t.TempDir(), "pandoc.zip")
	f, err := os.Create(buf)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	entry, err := w.Create("pandoc-3.14/bin/pandoc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(script)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(buf)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
