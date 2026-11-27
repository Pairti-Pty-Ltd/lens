package mermaid

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"
)

func TestRenderReturnsAPNG(t *testing.T) {
	img, err := New().Render(context.Background(), "graph TD\nA-->B\n")
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() < 10 || img.Bounds().Dy() < 10 {
		t.Fatalf("size %v", img.Bounds())
	}
	if !hasInk(img) {
		t.Fatal("diagram has no painted pixels")
	}
}

func TestPaperPNGIsOpaque(t *testing.T) {
	data, err := PaperPNG(context.Background(), "graph TD\nA-->B\n")
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, a := img.At(img.Bounds().Min.X, img.Bounds().Min.Y).RGBA()
	if a < 0xffff {
		t.Fatal("paper diagram is transparent")
	}
}

func TestRenderKeepsASubgraphFlowchart(t *testing.T) {
	const src = `flowchart TD
    subgraph Readers["LIBRARY VISITORS"]
        direction LR
        Desk["Reading Room\n• Browse shelves\n• Request a book"]
        Mobile["Mobile Catalogue\n• Search by title"]
    end
    subgraph Catalogue["CATALOGUE SERVICES"]
        Index["Search Index\n(library.example.org)"]
    end
    Desk -->|Lookup| Index
    Mobile -.->|"Title search"| Index
`
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	img, err := New().Render(ctx, src)
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() < 200 || b.Dy() < 200 {
		t.Fatalf("size %v", b)
	}
	if !hasInk(img) {
		t.Fatal("diagram has no painted pixels")
	}
}

func TestEmptyAndOversizedDiagrams(t *testing.T) {
	if _, err := New().Render(context.Background(), "  \n"); err == nil {
		t.Fatal("empty diagram rendered")
	}
	huge := "graph TD\nA-->B\n" + strings.Repeat("%% pad\n", 40_000)
	if _, err := New().Render(context.Background(), huge); err == nil {
		t.Fatal("oversized diagram rendered")
	}
}

func TestBadSyntaxReturnsAnError(t *testing.T) {
	_, err := New().Render(context.Background(), "this is not a diagram")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestCancelledRenderStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New().Render(ctx, "graph TD\nA-->B\n")
	if err == nil {
		t.Fatal("expected an error")
	}
}

func hasInk(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 3 {
		for x := b.Min.X; x < b.Max.X; x += 3 {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				return true
			}
		}
	}
	return false
}
