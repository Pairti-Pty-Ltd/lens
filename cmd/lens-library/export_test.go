package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"pairti-lense/internal/project"
)

func exportLibrary(t *testing.T) (*library, string) {
	t.Helper()
	dir := t.TempDir()
	if out, e := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); e != nil {
		t.Fatalf("%s: %v", out, e)
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { root.Close() })
	return &library{root: root}, dir
}

func stageReportFile(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", dir, "add", "--", name).CombinedOutput(); err != nil {
		t.Fatalf("%s: %v", out, err)
	}
}
func TestLibraryReportSelection(t *testing.T) {
	l, dir := exportLibrary(t)
	stageReportFile(t, dir, ".pairti-project.json", `{"files":["a.md"],"template":"cedar.docx"}`)
	stageReportFile(t, dir, "a.md", "# A\n")
	stageReportFile(t, dir, "notes/b.md", "# B\n")
	s, err := l.reportSnapshot(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	choice, src, err := l.reportSource(s, "report-1")
	if err != nil || choice.Template != "cedar.docx" || string(src) != "# A\n" {
		t.Fatal(choice, string(src), err)
	}
	stageReportFile(t, dir, ".pairti-reports.json", `{"open":"release","reports":[{"id":"release","name":"Book","files":["notes/b.md","a.md"]}]}`)
	stageReportFile(t, dir, "lens.toml", "author = \"Cedar\"")
	s, err = l.reportSnapshot(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, src, err = l.reportSource(s, "release")
	if err != nil || string(src) != "# B\n\n\n# A\n" {
		t.Fatalf("%q %v", src, err)
	}
	if p, err := os.ReadFile(filepath.Join(s.dir, "lens.toml")); err != nil || !strings.Contains(string(p), "Cedar") {
		t.Fatal(string(p), err)
	}
	for _, name := range []string{"../a.md", "hidden.md", "lens.toml"} {
		s.book.Reports[0].Files = []string{name}
		if _, _, err = l.reportSource(s, "release"); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	for _, target := range []string{"unlisted.png", "../outside.png", "https://example.org/plot.png"} {
		stageReportFile(t, dir, "notes/b.md", "![figure]("+target+")")
		s.book.Reports[0].Files = []string{"notes/b.md"}
		if _, _, err = l.reportSource(s, "release"); err == nil {
			t.Fatalf("accepted unavailable image %q", target)
		}
	}
	if err = os.Remove(filepath.Join(dir, "a.md")); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("notes/b.md", filepath.Join(dir, "a.md")); err != nil {
		t.Fatal(err)
	}
	s.book.Reports[0].Files = []string{"a.md"}
	if _, _, err = l.reportSource(s, "release"); err == nil {
		t.Fatal("accepted linked report member")
	}
}
func TestLibraryReportExport(t *testing.T) {
	l, dir := exportLibrary(t)
	realTypst := os.Getenv("LENS_LIBRARY_TEST_TYPST")
	if realTypst == "" {
		realTypst, _ = exec.LookPath("typst")
	}
	realPandoc, _ := exec.LookPath("pandoc")
	conf := t.TempDir()
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "AppData"} {
		t.Setenv(key, conf)
	}
	cfg, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.CopyFS(filepath.Join(cfg, "pairti-lense", "brands", "cedar"), os.DirFS("../../testdata/brands/cedar")); err != nil {
		t.Fatal(err)
	}
	var imageBytes bytes.Buffer
	if err := png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 13, 13))); err != nil {
		t.Fatal(err)
	}
	stageReportFile(t, dir, "chap/plot.png", imageBytes.String())
	stageReportFile(t, dir, "chap/r.md", "# Novel report\n\nA line of field notes.\n\n![Plot](plot.png)\n")
	stageReportFile(t, dir, ".pairti-reports.json", `{"open":"r","reports":[{"id":"r","name":"Field trip","template":"cedar.pdf","files":["chap/r.md"]}]}`)
	run := func(method, path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost:8081"+path, nil)
		w := httptest.NewRecorder()
		l.handler(t.TempDir()).ServeHTTP(w, r)
		return w
	}
	path := "/api/report/export?report=r"
	out := run("GET", "/api/reports")
	var data struct{ Book project.Book }
	if err = json.Unmarshal(out.Body.Bytes(), &data); err != nil || data.Book.Open != "r" || !strings.Contains(out.Body.String(), "cedar.docx") {
		t.Fatal(out.Code, err, out.Body.String())
	}
	for _, id := range []string{"unknown.pdf", "../cedar.pdf"} {
		if r := run("POST", path+"&output="+url.QueryEscape(id)); r.Code != 422 {
			t.Fatal(id, r.Code)
		}
	}
	if r := run("POST", "/api/report/export?report=no"); r.Code != 422 {
		t.Fatal(r.Code)
	}
	t.Run("missing binary", func(t *testing.T) {
		// Existing Git remains usable while conversion tools are unavailable.
		git, err := exec.LookPath("git")
		if err != nil {
			t.Fatal(err)
		}
		bin := t.TempDir()
		if err = os.Symlink(git, filepath.Join(bin, "git")); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", bin)
		for _, target := range []string{"cedar.pdf", "cedar.docx"} {
			w := run("POST", path+"&output="+target)
			if w.Code != 503 {
				t.Fatalf("%s: %d %s", target, w.Code, w.Body.String())
			}
		}
	})
	t.Run("branded PDF", func(t *testing.T) {
		if realTypst == "" {
			t.Skip("typst missing")
		}
		t.Setenv("PATH", filepath.Dir(realTypst)+string(os.PathListSeparator)+os.Getenv("PATH"))
		w := run("POST", path)
		if w.Code != 200 || !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
			t.Fatalf("PDF %d: %.500s", w.Code, w.Body.String())
		}
		f := filepath.Join(t.TempDir(), "out.pdf")
		if err := os.WriteFile(f, w.Body.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
		bin, err := exec.LookPath("pdftotext")
		if err != nil {
			t.Skip("pdftotext missing")
		}
		text, err := exec.CommandContext(context.Background(), bin, f, "-").Output()
		if err != nil || !bytes.Contains(text, []byte("Cedar field notes")) {
			t.Fatalf("error %v text %s", err, text)
		}
	})
	t.Run("branded Word", func(t *testing.T) {
		if realPandoc == "" {
			t.Skip("pandoc missing")
		}
		w := run("POST", path+"&output=cedar.docx")
		if w.Code != 200 {
			t.Fatalf("Word %d: %s", w.Code, w.Body.String())
		}
		z, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
		if err != nil {
			t.Fatal(err)
		}
		logo, err := os.ReadFile("../../testdata/brands/cedar/logo.png")
		if err != nil {
			t.Fatal(err)
		}
		var foundLogo, foundNote, foundPlot bool
		for _, part := range z.File {
			f, err := part.Open()
			if err != nil {
				t.Fatal(err)
			}
			b, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			foundLogo = foundLogo || bytes.Equal(b, logo)
			foundPlot = foundPlot || bytes.Equal(b, imageBytes.Bytes())
			foundNote = foundNote || (part.Name == "word/document.xml" && bytes.Contains(b, []byte("field notes")))
		}
		if !foundLogo || !foundNote || !foundPlot {
			t.Fatalf("logo %v content %v plot %v", foundLogo, foundNote, foundPlot)
		}
	})
}
