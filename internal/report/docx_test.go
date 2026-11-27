package report

import (
	"archive/zip"
	"bytes"
	"image"
	"image/png"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocxExportRecordsTheBrand(t *testing.T) {
	installExample(t, emptyBrands(t))
	dir := t.TempDir()
	bin := filepath.Join(dir, "pandoc")
	script := `#!/bin/sh
out=
filter=
for a in "$@"; do
  case "$a" in
    -o) out=next ;;
    --lua-filter=*) filter=${a#--lua-filter=} ;;
    *) if [ "$out" = next ]; then out=$a; fi ;;
  esac
done
{
  printf '%s\n' "$*"
  printf 'brand=%s\n' "$PAIRTI_BRAND"
  echo '---note---'
  cat note.md
  echo '---filter---'
  cat "$filter"
} > "$out"
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Export(ExportRequest{
		Src:    []byte("---\ntitle: Coverage\nclient: West\n---\n\n# Assay\n\nA note.\n"),
		Name:   "orchard-survey-coverage",
		Brand:  "cedar.docx",
		Dir:    dir,
		Pandoc: bin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.PDF != filepath.Join(dir, "orchard-survey-coverage.docx") {
		t.Fatalf("path %s", res.PDF)
	}
	b, err := os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"--reference-doc=", "--lua-filter=", "brand=cedar", "title=Coverage", "client=West", "logo.png"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s\n%s", want, got)
		}
	}
	if strings.Contains(got, "title: Coverage") {
		t.Fatalf("front matter was left in the note\n%s", got)
	}
}

func TestPandocIsFoundOutsideTheAppPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "pandoc")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findPandoc(func(string) (string, error) {
		return "", os.ErrNotExist
	}, []string{bin})
	if err != nil || got != bin {
		t.Fatalf("got %q err %v", got, err)
	}
	_, err = findPandoc(func(string) (string, error) {
		return "", os.ErrNotExist
	}, nil)
	if _, ok := err.(*NotInstalledError); !ok {
		t.Fatalf("err %v", err)
	}
}

func TestDocxStagesAMermaidDiagram(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "pandoc")
	script := `#!/bin/sh
out=
for a in "$@"; do
  case "$a" in
    -o) out=next ;;
    *) if [ "$out" = next ]; then out=$a; fi ;;
  esac
done
cat note.md > "$out"
`
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Export(ExportRequest{
		Src:    []byte("# Note\n\n```mermaid\ngraph TD\nA-->B\n```\n\nAfter.\n"),
		Name:   "coverage",
		Brand:  "default.docx",
		Dir:    dir,
		Pandoc: bin,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if strings.Contains(text, "A-->B") || strings.Contains(text, "```mermaid") {
		t.Fatalf("fence kept:\n%s", text)
	}
	if !strings.Contains(text, "![](diag-1.png)") || !strings.Contains(text, "After.") {
		t.Fatalf("diagram missing:\n%s", text)
	}
}

func TestDocxEmbedsALocalImage(t *testing.T) {
	bin, err := exec.LookPath("pandoc")
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	fig := filepath.Join(dir, "figures")
	if err := os.Mkdir(fig, 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(fig, "shot.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewNRGBA(image.Rect(0, 0, 8, 4))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	out := t.TempDir()
	res, err := Export(ExportRequest{
		Src:    []byte("# Note\n\n![plate](figures/shot.png)\n"),
		Name:   "coverage",
		Brand:  "default.docx",
		Dir:    out,
		Base:   dir,
		Pandoc: bin,
	})
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var media bool
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "word/media/") && strings.HasSuffix(f.Name, ".png") {
			media = true
		}
	}
	if !media {
		t.Fatal("docx has no embedded image")
	}
}

func TestPandocWritesABrandedDocx(t *testing.T) {
	installExample(t, emptyBrands(t))
	bin, err := exec.LookPath("pandoc")
	if err != nil {
		t.Skip(err)
	}
	dir := t.TempDir()
	res, err := Export(ExportRequest{
		Src:    []byte("---\ntitle: Coverage\nclient: West\n---\n\n# Assay\n\nA note.\n"),
		Name:   "coverage",
		Brand:  "cedar.docx",
		Dir:    dir,
		Pandoc: bin,
	})
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.OpenReader(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	logo, err := os.ReadFile("../../testdata/brands/cedar/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	var text string
	var header, hasLogo bool
	for _, f := range z.File {
		if strings.HasPrefix(f.Name, "word/media/") || strings.HasPrefix(f.Name, "word/header") {
			r, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(r)
			r.Close()
			if err != nil {
				t.Fatal(err)
			}
			header = header || strings.Contains(string(data), "Cedar field notes")
			hasLogo = hasLogo || bytes.Equal(logo, data)
		}
		if f.Name != "word/document.xml" {
			continue
		}
		r, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 1<<20)
		n, _ := r.Read(buf)
		r.Close()
		text = string(buf[:n])
	}
	if !header || !hasLogo {
		t.Fatalf("header=%v logo=%v", header, hasLogo)
	}
	for _, want := range []string{"Coverage", "Client: West", "Licence and Disclosure", "CedarCoverTitle", "Prepared from fictional observations."} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s\n%s", want, text)
		}
	}
	if strings.Count(text, `w:type="page"`) < 3 {
		t.Fatalf("contents is not on its own page\n%s", text)
	}
}
