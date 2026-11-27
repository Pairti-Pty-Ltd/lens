package ui

import (
	"testing"

	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

func TestFieldPaintsDark(t *testing.T) {
	th := theme.Pairti()
	f := textfield.New(
		textfield.Placeholder("Filter"),
		textfield.PainterOpt(FieldPainter(th)),
	)
	uitest.LayoutWidget(f, 220, 32)
	f.SetBounds(geometry.NewRect(0, 0, 220, 32))
	canvas := uitest.DrawWidget(f)
	if len(canvas.RoundRects) == 0 {
		t.Fatal("field drew no background")
	}
	if canvas.RoundRects[0].Color != widget.Hex(th.CodeFill) {
		t.Fatalf("field fill %v, want %v", canvas.RoundRects[0].Color, widget.Hex(th.CodeFill))
	}
	if canvas.RoundRects[0].Color == widget.ColorWhite {
		t.Fatal("field is white")
	}
}

func TestQuietButtonHasNoPill(t *testing.T) {
	b := button.New(
		button.Text("Open"),
		button.PainterOpt(QuietButton(theme.Pairti(), false)),
		button.VariantOpt(button.TextOnly),
	)
	uitest.LayoutWidget(b, 80, 28)
	b.SetBounds(geometry.NewRect(0, 0, 80, 28))
	canvas := uitest.DrawWidget(b)
	if len(canvas.RoundRects) != 0 {
		t.Fatalf("idle button drew a fill: %+v", canvas.RoundRects)
	}
	var saw bool
	for _, tx := range canvas.Texts {
		if tx.Text == "Open" {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("button text %+v", canvas.Texts)
	}
}
