package diagram

import (
	"context"
	"image"
	"testing"
)

type fake struct {
	langs []string
	n     int
}

func (f *fake) Languages() []string { return f.langs }

func (f *fake) Render(context.Context, string) (image.Image, error) {
	f.n++
	return image.NewNRGBA(image.Rect(0, 0, 2, 2)), nil
}

func TestRegisterLookupIsCaseInsensitive(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	f := &fake{langs: []string{"Mermaid"}}
	Register(f)
	if !Known("mermaid") || !Known("MERMAID") {
		t.Fatal("mermaid not known")
	}
	if Known("go") {
		t.Fatal("go was registered")
	}
	img, err := Render(context.Background(), "mermaid", "graph TD")
	if err != nil || img == nil || img.Bounds().Dx() != 2 {
		t.Fatalf("img %v err %v", img, err)
	}
	if f.n != 1 {
		t.Fatalf("calls %d", f.n)
	}
}
