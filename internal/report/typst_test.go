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

func TestLocatePrefersTheBundledBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "typst")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	looked := false
	got, err := locateTypst([]string{dir, t.TempDir()}, func(string) (string, error) {
		looked = true
		return "/usr/bin/typst", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != bin {
		t.Fatalf("bin %s", got)
	}
	if looked {
		t.Fatal("PATH was consulted")
	}
}

func TestLocateUsesPathWhenNothingIsBundled(t *testing.T) {
	got, err := locateTypst([]string{t.TempDir()}, func(name string) (string, error) {
		if name != "typst" && name != "typst.exe" {
			t.Fatalf("lookup %s", name)
		}
		return "/opt/typst", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/opt/typst" {
		t.Fatalf("bin %s", got)
	}
}

func TestFetchInstallsFromAChecksummedArchive(t *testing.T) {
	src := t.TempDir()
	script := "#!/bin/sh\nprintf 'typst " + typstVersion + "\\n'\n"
	if err := os.WriteFile(filepath.Join(src, "typst"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(t.TempDir(), "typst.tar.xz")
	cmd := exec.Command("tar", "-cJf", archive, "-C", src, "typst")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("tar xz: %v %s", err, out)
	}
	body, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write(body)
	}))
	defer srv.Close()
	dest := t.TempDir()
	spec := typstFetch{URL: srv.URL + "/typst.tar.xz", SHA: hex.EncodeToString(sum[:]), File: "typst.tar.xz", Dest: dest}
	got, err := fetchTypst(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(dest, typstName()) {
		t.Fatalf("installed %s", got)
	}
	out, err := exec.Command(got, "--version").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), typstVersion) {
		t.Fatalf("version %q", out)
	}
	if _, err := fetchTypst(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	if hits != 1 {
		t.Fatalf("downloads %d", hits)
	}
}

func TestFetchRejectsABadChecksum(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not an archive"))
	}))
	defer srv.Close()
	_, err := fetchTypst(t.Context(), typstFetch{
		URL: srv.URL, SHA: strings.Repeat("ab", 32), File: "typst.tar.xz", Dest: t.TempDir(),
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err %v", err)
	}
}

func TestFetchInstallsFromAZip(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "typst.zip")
	zf, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(zf)
	entry, err := w.Create("typst-windows/typst")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("#!/bin/sh\nprintf 'typst " + typstVersion + "\\n'\n")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zf.Close(); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(body)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()
	got, err := fetchTypst(t.Context(), typstFetch{
		URL: srv.URL + "/typst.zip", SHA: hex.EncodeToString(sum[:]), File: "typst.zip", Dest: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(got, "--version").CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if !strings.Contains(string(out), typstVersion) {
		t.Fatalf("version %q", out)
	}
}

func TestPinnedTypstInstalls(t *testing.T) {
	if os.Getenv("PAIRTI_INSTALL_TYPST") == "" {
		t.Skip("set PAIRTI_INSTALL_TYPST=1 to download the pinned compiler")
	}
	bin, err := Bin()
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(bin, "--version").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), typstVersion) {
		t.Fatalf("version %q from %s", out, bin)
	}
	dir, err := installDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(bin, dir) && filepath.Base(filepath.Dir(bin)) != "MacOS" && filepath.Base(filepath.Dir(bin)) != "Resources" {
		t.Fatalf("bin %s is not the app copy", bin)
	}
}
