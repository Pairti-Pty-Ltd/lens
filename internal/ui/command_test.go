package ui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
)

func TestCommandLineIsOneRow(t *testing.T) {
	line := NewCommandLine()
	size := line.Layout(nil, geometry.Constraints{MaxWidth: 400, MaxHeight: 80})
	if size.Height != 0 {
		t.Fatalf("idle height %v", size.Height)
	}
	line.Set(true, "w")
	size = line.Layout(nil, geometry.Constraints{MaxWidth: 400, MaxHeight: 80})
	if size.Height != 26 {
		t.Fatalf("height %v", size.Height)
	}
	line.SetBounds(geometry.NewRect(0, 0, 400, 26))
	rows := glyphs(uitest.DrawWidget(line))
	if !hasGlyph(rows, ":w") {
		t.Fatalf("rows %+v", rows)
	}
}

func TestColonWRunsTheCommand(t *testing.T) {
	var got string
	p := NewPage()
	buf := edit.New("## Required checks\n")
	p.UseBuffer(buf)
	p.SetBlocks(doc.Blocks([]byte(buf.Text())))
	p.SetInline(nil, nil, func(cmd string) { got = cmd })
	uitest.LayoutWidget(p, 600, 200)
	p.SetBounds(geometry.NewRect(0, 0, 600, 200))
	ctx := uitest.NewMockContext()
	if !p.Event(ctx, uitest.Click(p.gutterWidth()+4, p.LineTop(1)+4)) {
		t.Fatal("click was not consumed")
	}
	if !p.Event(ctx, uitest.KeyType(event.KeySemicolon, ':', 0)) {
		t.Fatal("colon was not consumed")
	}
	if p.ModeName() != "Command" {
		t.Fatalf("mode %s", p.ModeName())
	}
	p.Event(ctx, uitest.KeyType(event.KeyW, 'w', 0))
	if p.Command() != "w" {
		t.Fatalf("command %q", p.Command())
	}
	if !p.Event(ctx, uitest.KeyPress(event.KeyEnter, 0)) {
		t.Fatal("enter was not consumed")
	}
	if got != "w" || p.ModeName() != "View" {
		t.Fatalf("ran %q mode %s", got, p.ModeName())
	}
}

func containsGlyph(rows []glyph, part string) bool {
	for _, g := range rows {
		if g.text == part {
			return true
		}
	}
	return false
}
