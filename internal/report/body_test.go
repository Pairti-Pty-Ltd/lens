package report

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBodySkipsFrontMatter(t *testing.T) {
	got := Body([]byte("---\ntitle: Assay\n---\n\n# Assay\n\nNote.\n"))
	if strings.Contains(got, "title:") {
		t.Fatalf("front matter was exported:\n%s", got)
	}
	if !strings.Contains(got, "= Assay") {
		t.Fatalf("heading missing:\n%s", got)
	}
}

func TestBodyCopiesALocalImage(t *testing.T) {
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
	work := t.TempDir()
	got := writeBody(work, dir, []byte("# Note\n\n![plate](figures/shot.png)\n"))
	if !strings.Contains(got, `#image("img-1.png"`) {
		t.Fatalf("image missing:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(work, "img-1.png")); err != nil {
		t.Fatal(err)
	}
	missing := writeBody(work, dir, []byte("![gone](missing.png)\n"))
	if strings.Contains(missing, "#image") || !strings.Contains(missing, "gone") {
		t.Fatalf("missing image:\n%s", missing)
	}
}

func TestBodyRendersAMermaidFence(t *testing.T) {
	work := t.TempDir()
	src := []byte("# Note\n\n```mermaid\ngraph TD\nA-->B\n```\n\nAfter.\n\n```mermaid\nthis is not a diagram\n```\n\n```go\nfmt.Println()\n```\n")
	got := writeBody(work, t.TempDir(), src)
	if strings.Contains(got, "A-->B") || strings.Count(got, "```mermaid") != 1 {
		t.Fatalf("fence kept:\n%s", got)
	}
	if !strings.Contains(got, `#image("diag-1.png"`) {
		t.Fatalf("diagram missing:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(work, "diag-1.png")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "After.") || !strings.Contains(got, "this is not a diagram") || !strings.Contains(got, "fmt.Println") {
		t.Fatalf("surrounding text lost:\n%s", got)
	}
}

func TestBodySplitsATallDiagram(t *testing.T) {
	var b strings.Builder
	b.WriteString("```mermaid\ngraph TD\n")
	for i := 0; i < 36; i++ {
		fmt.Fprintf(&b, "N%d[\"Step %d of the long referral walk\"]\n", i, i)
		if i > 0 {
			fmt.Fprintf(&b, "N%d-->N%d\n", i-1, i)
		}
	}
	b.WriteString("```\n")
	work := t.TempDir()
	got := writeBody(work, t.TempDir(), []byte(b.String()))
	if strings.Contains(got, "Step 0") {
		t.Fatalf("source kept:\n%s", got)
	}
	if !strings.Contains(got, `#image("diag-1.png"`) || !strings.Contains(got, `#image("diag-1-2.png"`) {
		t.Fatalf("diagram was not split:\n%s", got)
	}
	for _, name := range []string{"diag-1.png", "diag-1-2.png"} {
		_, h, ok := imagePixels(filepath.Join(work, name))
		if !ok {
			t.Fatal(name)
		}
		hm := float64(h) / diagramDPI * 25.4
		if hm > diagramSliceMM*1.25 {
			t.Fatalf("%s is %.1fmm tall", name, hm)
		}
	}
}

func TestBodyKeepsAHeadingAndEscapesMarkup(t *testing.T) {
	src := []byte("# Assay\n\nCost is #1 and uses *stars*.\n\n- one\n- two\n")
	got := Body(src)
	if !strings.Contains(got, "= Assay") {
		t.Fatalf("heading missing:\n%s", got)
	}
	if !strings.Contains(got, `\#1`) {
		t.Fatalf("hash was not escaped:\n%s", got)
	}
	if !strings.Contains(got, "- one") || !strings.Contains(got, "- two") {
		t.Fatalf("list missing:\n%s", got)
	}
}
