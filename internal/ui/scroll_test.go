package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/doc"
)

func TestScrollPainterLeavesTheTrackClear(t *testing.T) {
	canvas := &uitest.MockCanvas{}
	track := geometry.NewRect(888, 0, 12, 400)
	thumb := geometry.NewRect(890, 40, 8, 90)
	ScrollPainter{}.PaintScrollbar(canvas, scrollview.PaintState{
		Bounds:         geometry.NewRect(0, 0, 900, 400),
		VScrollVisible: true,
		VTrackRect:     track,
		VThumbRect:     thumb,
	})
	if len(canvas.RoundRects) != 1 {
		t.Fatalf("draws %d marks, want the pill only: %+v", len(canvas.RoundRects), canvas.RoundRects)
	}
	got := canvas.RoundRects[0]
	if got.Bounds != thumb {
		t.Fatalf("drew %v, want the pill %v", got.Bounds, thumb)
	}
	if got.Color.A < 0.5 {
		t.Fatalf("pill alpha %v", got.Color.A)
	}
}

func TestScrollGutterEndsBeforeThePill(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "line %d of the assay note\n\n", i)
	}
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte(b.String())))
	sv := scrollview.New(ScrollInset(p), scrollview.DirectionOpt(scrollview.Vertical))
	uitest.LayoutWidget(sv, 900, 400)
	if p.Bounds().Width() > 900-ScrollGutter+0.5 {
		t.Fatalf("page width %v, gutter %v", p.Bounds().Width(), ScrollGutter)
	}
	if p.Bounds().Max.X > 900-ScrollGutter+0.5 {
		t.Fatalf("text ends at %v", p.Bounds().Max.X)
	}
}

func TestDocumentScrollHasNoTrack(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "line %d of the assay note\n\n", i)
	}
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte(b.String())))
	sv := scrollview.New(p,
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.PainterOpt(ScrollPainter{}),
	)
	uitest.LayoutWidget(sv, 900, 400)
	sv.SetBounds(geometry.NewRect(0, 0, 900, 400))
	canvas := uitest.DrawWidget(sv)
	if len(canvas.RoundRects) != 1 {
		t.Fatalf("round rects %+v", canvas.RoundRects)
	}
	pill := canvas.RoundRects[0]
	if pill.Bounds.Height() > 200 {
		t.Fatalf("track still painted, height %v", pill.Bounds.Height())
	}
	if pill.Color.A < 0.5 {
		t.Fatalf("pill alpha %v", pill.Color.A)
	}
}
