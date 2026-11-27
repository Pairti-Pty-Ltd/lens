package ui

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/doc"
)

func writePNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 200, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func drawPage(p *Page, w, h float32) *uitest.MockCanvas {
	uitest.LayoutWidget(p, w, h)
	p.SetBounds(geometry.NewRect(0, 0, w, h))
	return uitest.DrawWidget(p)
}

func TestPageDrawsInlineImage(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "figures", "plate.png"), 8, 8)
	p := NewPage()
	p.SetSource(dir, doc.Blocks([]byte("See the ![plate](figures/plate.png) mark.\n")))
	canvas := drawPage(p, 640, 300)
	rows := glyphs(canvas)
	if !hasGlyph(rows, "See") || !hasGlyph(rows, "mark.") {
		t.Fatalf("text = %+v", rows)
	}
	for _, tx := range rows {
		if tx.text == "plate" || strings.Contains(tx.text, "plate.png") || strings.Contains(tx.text, "figures") {
			t.Fatalf("image label drawn beside the pixels: %q", tx.text)
		}
	}
	if len(canvas.Images) != 1 {
		t.Fatalf("images = %d", len(canvas.Images))
	}
	b := canvas.Images[0].Image.Bounds()
	if b.Dx() != 8 || b.Dy() != 8 {
		t.Fatalf("drawn size %dx%d", b.Dx(), b.Dy())
	}
	seeY := glyphY(canvas, "See")
	imgY := canvas.Images[0].At.Y
	line := p.theme.Body * p.theme.Line
	if imgY < seeY || imgY > seeY+line {
		t.Fatalf("image y %v is outside the See line at %v", imgY, seeY)
	}
}

func glyphY(c *uitest.MockCanvas, text string) float32 {
	for _, t := range c.StyledTexts {
		if t.Text == text {
			return t.Bounds.Min.Y
		}
	}
	for _, t := range c.Texts {
		if t.Text == text {
			return t.Bounds.Min.Y
		}
	}
	return -1
}

func TestPageKeepsAPictureInTheSentence(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "mid.png"), 200, 40)
	p := NewPage()
	p.SetSource(dir, doc.Blocks([]byte("Before ![shot](mid.png) after.\n")))
	canvas := drawPage(p, 640, 300)
	if len(canvas.Images) != 1 {
		t.Fatalf("images = %d glyphs %+v", len(canvas.Images), glyphs(canvas))
	}
	b := canvas.Images[0].Image.Bounds()
	if b.Dx() != 200 || b.Dy() != 40 {
		t.Fatalf("drawn %dx%d, want the natural 200x40", b.Dx(), b.Dy())
	}
	before := glyphBox(canvas, "Before")
	after := glyphBox(canvas, "after.")
	if before.Min.Y < 0 || after.Min.Y != before.Min.Y {
		t.Fatalf("before %v after %v", before, after)
	}
	img := canvas.Images[0].At
	if img.X < before.Max.X || after.Min.X < img.X+float32(b.Dx()) {
		t.Fatalf("picture at %v is not between before %v and after %v", img, before, after)
	}
	line := p.theme.Body * p.theme.Line
	if img.Y < before.Min.Y || img.Y > before.Min.Y+line+40 {
		t.Fatalf("picture y %v is off the text line %v", img.Y, before.Min.Y)
	}
}

func glyphBox(c *uitest.MockCanvas, text string) geometry.Rect {
	for _, t := range c.StyledTexts {
		if t.Text == text {
			return t.Bounds
		}
	}
	for _, t := range c.Texts {
		if t.Text == text {
			return t.Bounds
		}
	}
	return geometry.Rect{Min: geometry.Pt(-1, -1)}
}

func TestPageFitsWideImageToTheMeasure(t *testing.T) {
	dir := t.TempDir()
	writePNG(t, filepath.Join(dir, "wide.png"), 800, 40)
	p := NewPage()
	p.SetSource(dir, doc.Blocks([]byte("![shot](wide.png)\n")))
	canvas := drawPage(p, 400, 400)
	if len(canvas.Images) != 1 {
		t.Fatalf("images = %d glyphs %+v", len(canvas.Images), glyphs(canvas))
	}
	got := canvas.Images[0].Image.Bounds().Dx()
	content := 400 - int(p.gutterWidth())
	if got != content {
		t.Fatalf("width %d, want %d", got, content)
	}
	for _, tx := range glyphs(canvas) {
		if tx.text == "shot" || strings.Contains(tx.text, "wide.png") {
			t.Fatalf("label drawn: %q", tx.text)
		}
	}
}

func TestPageMissingImageIsTheAlt(t *testing.T) {
	dir := t.TempDir()
	p := NewPage()
	p.SetSource(dir, doc.Blocks([]byte("![plate](figures/missing.png)\n\n![](figures/other.png)\n")))
	rows := glyphs(drawPage(p, 640, 300))
	if !hasGlyph(rows, "plate") || !hasGlyph(rows, "other.png") {
		t.Fatalf("rows = %+v", rows)
	}
	for _, tx := range rows {
		if strings.Contains(tx.text, "figures") || strings.Contains(tx.text, "missing") {
			t.Fatalf("path drawn: %q", tx.text)
		}
	}
}

func TestPictureResolvesEscapedAndParent(t *testing.T) {
	root := t.TempDir()
	docDir := filepath.Join(root, "notes")
	if err := os.Mkdir(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writePNG(t, filepath.Join(root, "my pic.png"), 3, 3)
	p := NewPage()
	p.SetSource(docDir, doc.Blocks([]byte("![a](../my%20pic.png)\n")))
	canvas := drawPage(p, 480, 200)
	if len(canvas.Images) != 1 || canvas.Images[0].Image.Bounds().Dx() != 3 {
		t.Fatalf("images %d glyphs %+v", len(canvas.Images), glyphs(canvas))
	}
}

func TestDataURLImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	raw := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	p := NewPage()
	p.SetSource(t.TempDir(), doc.Blocks([]byte("![]("+raw+")\n")))
	canvas := drawPage(p, 480, 200)
	if len(canvas.Images) != 1 || canvas.Images[0].Image.Bounds().Dx() != 4 {
		t.Fatalf("images %+v glyphs %+v", canvas.Images, glyphs(canvas))
	}
}

func TestRemoteImage(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 3))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()
	p := NewPage()
	p.SetSource(t.TempDir(), doc.Blocks([]byte("![badge]("+srv.URL+"/badge.png)\n")))
	canvas := drawPage(p, 480, 200)
	if len(canvas.Images) != 1 || canvas.Images[0].Image.Bounds().Dx() != 6 {
		t.Fatalf("images %d glyphs %+v", len(canvas.Images), glyphs(canvas))
	}
}
