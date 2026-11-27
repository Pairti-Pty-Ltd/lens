package ui

import (
	"github.com/gogpu/ui/geometry"
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
)

// The desktop delivers a physical press separately from its printable character.
func TestDesktopModalEditing(t *testing.T) {
	for _, formatted := range []bool{false, true} {
		name := "Raw"
		if formatted {
			name = "Formatted"
		}
		t.Run(name, func(t *testing.T) {
			const source = "first second third\nfourth"
			b := edit.New(source)
			var w widget.Widget
			if formatted {
				p := NewPage()
				p.UseBuffer(b)
				p.SetBlocks(doc.Blocks([]byte(source)))
				p.SetFocused(true)
				w = p
			} else {
				e := NewEditor(nil, nil)
				e.UseBuffer(b)
				e.SetFocused(true)
				w = e
			}
			ctx := uitest.NewMockContext()
			typeKeys := func(keys string) {
				t.Helper()
				for _, r := range keys {
					key := event.KeyUnknown
					if r >= 'a' && r <= 'z' {
						key = event.KeyA + event.Key(r-'a')
					}
					w.Event(ctx, event.NewKeyEvent(event.KeyPress, key, 0, 0))
					if !w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, 0)) {
						t.Fatalf("character %q was ignored", r)
					}
					w.Event(ctx, event.NewKeyEvent(event.KeyRelease, key, 0, 0))
				}
			}
			escape := func() { w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, 0)) }
			typeKeys("wcw")
			if b.Modal.Mode != edit.ModeEdit {
				t.Fatal("cw did not enter Edit")
			}
			typeKeys("new")
			escape()
			escape()
			if b.Text() != "first new third\nfourth" || b.Modal.Mode != edit.ModeView || b.CaretCol() != 8 {
				t.Fatalf("after Escape got %q at %d in mode %v", b.Text(), b.CaretCol(), b.Modal.Mode)
			}
			typeKeys("w.")
			if b.Text() != "first new new\nfourth" {
				t.Fatalf("repeat got %q", b.Text())
			}
			typeKeys("uu")
			if b.Text() != source {
				t.Fatalf("undo got %q", b.Text())
			}
			typeKeys("gg0ftciw")
			typeKeys("one")
			escape()
			if b.Text() != "one second third\nfourth" {
				t.Fatalf("find and change inner word got %q", b.Text())
			}
		})
	}
}

func TestSwitchPresentationDuringInsert(t *testing.T) {
	buf := edit.New("first second")
	raw := NewEditor(nil, nil)
	raw.UseBuffer(buf)
	raw.SetFocused(true)
	ctx := uitest.NewMockContext()
	send := func(w widget.Widget, r rune) { w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, 0)) }
	for _, r := range "wcwne" {
		send(raw, r)
	}
	page := NewPage()
	page.UseBuffer(buf)
	page.SetSource("", doc.Blocks([]byte(buf.Text())))
	page.ResumeFormatted(Passage{Line: 1})
	page.SetFocused(true)
	if !page.Open() || page.ModeName() != "Edit" {
		t.Fatalf("formatted mode %q, open %v", page.ModeName(), page.Open())
	}
	send(page, 'w')
	page.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, 0))
	if buf.Text() != "first new" || raw.ModeName() != "View" {
		t.Fatalf("text %q, raw mode %s", buf.Text(), raw.ModeName())
	}
	send(raw, 'u')
	if buf.Text() != "first second" {
		t.Fatalf("undo across modes got %q", buf.Text())
	}
}

func TestNavigationAndRangeEditingInBothPresentations(t *testing.T) {
	for _, name := range []string{"Raw", "Formatted"} {
		t.Run(name, func(t *testing.T) {
			src := strings.Repeat("spare line\n", 41) + "alpha beta gamma\nlast"
			b := edit.New(src)
			var w widget.Widget
			if name == "Raw" {
				e := NewEditor(nil, nil)
				e.UseBuffer(b)
				e.SetFocused(true)
				w = e
			} else {
				p := NewPage()
				p.UseBuffer(b)
				p.SetBlocks(doc.Blocks([]byte(src)))
				p.SetFocused(true)
				w = p
			}
			ctx := uitest.NewMockContext()
			key := func(k event.Key, r rune, mods event.Modifiers) {
				w.Event(ctx, event.NewKeyEvent(event.KeyPress, k, r, mods))
			}
			typeKeys := func(s string) {
				for _, r := range s {
					key(event.KeyUnknown, r, 0)
				}
			}
			typeKeys(":42")
			key(event.KeyEnter, 0, 0)
			if b.CaretLine() != 41 {
				t.Fatalf(":42 landed on line %d", b.CaretLine()+1)
			}
			key(event.KeyW, 0, event.ModCtrl)
			typeKeys("ecnew")
			key(event.KeyEscape, 0, 0)
			if b.LineText(42) != "new beta gamma" {
				t.Fatal(b.LineText(42))
			}
			typeKeys("u/beta")
			key(event.KeyEnter, 0, 0)
			if b.CaretLine() != 41 || b.CaretCol() != 6 {
				t.Fatalf("search at %d:%d", b.CaretLine(), b.CaretCol())
			}
			typeKeys(":42,43d")
			key(event.KeyEnter, 0, 0)
			if strings.Contains(b.Text(), "gamma") {
				t.Fatal("addressed delete left the selected lines")
			}
		})
	}
}

func TestFormattedDragKeepsWrapAndSourceColumns(t *testing.T) {
	p := NewPage()
	src := "Wide letters mix with iii and Wide letters mix with iii and café again."
	b := edit.New(src)
	p.UseBuffer(b)
	p.SetBlocks(doc.Blocks([]byte(src)))
	p.measure = func(s string, size float32, _ bool, _ string) float32 {
		var w float32
		for _, r := range s {
			switch r {
			case 'i', ' ':
				w += size * .25
			case 'W':
				w += size
			default:
				w += size * .55
			}
		}
		return w
	}
	const width = float32(188)
	layout := func() { uitest.LayoutWidget(p, width, 400); p.SetBounds(geometry.NewRect(0, 0, width, 400)) }
	layout()
	original := append([]rendered(nil), p.lines...)
	if len(original) < 3 {
		t.Fatal("fixture did not wrap")
	}
	x := p.gutterWidth()
	y := float32(0)
	for i := 0; i < 2; i++ {
		y += original[i].before + p.rowHeight(original[i])
	}
	y += original[2].before + 2
	ctx := uitest.NewMockContext()
	p.Event(ctx, uitest.Click(x+p.prefixWidth(original[2], 1), y))
	if b.CaretCol() != original[2].col+1 {
		t.Fatalf("click on third row got col %d, row starts %d", b.CaretCol(), original[2].col)
	}
	layout()
	p.Event(ctx, uitest.MouseDrag(x+p.prefixWidth(original[2], 4), y))
	from, to, ok := b.Selection()
	if !ok || from != original[2].col+1 || to != original[2].col+4 {
		t.Fatalf("range [%d,%d), row starts %d", from, to, original[2].col)
	}
	layout()
	if len(original) != len(p.lines) {
		t.Fatal("selection changed number of rows")
	}
	for i, row := range p.lines {
		if row.col != original[i].col || p.piecesWidth(row.pieces) > width-x+.1 {
			t.Fatalf("row %d column %d width %f", i, row.col, p.piecesWidth(row.pieces))
		}
	}
	canvas := uitest.DrawWidget(p)
	var highlights int
	for _, r := range canvas.Rects {
		if r.Color == rgba(p.theme.Selection) {
			highlights++
			if r.Bounds.Width() > p.prefixWidth(original[2], 4)-p.prefixWidth(original[2], 1)+.1 {
				t.Fatalf("selection painted %v", r.Bounds)
			}
		}
	}
	if highlights != 1 {
		t.Fatalf("got %d selection rectangles", highlights)
	}
}

func TestWrappedMotionStaysOnSourceLine(t *testing.T) {
	for _, name := range []string{"Raw", "Formatted"} {
		t.Run(name, func(t *testing.T) {
			source := strings.Repeat("words with spaces ", 20) + "\nnext"
			b := edit.New(source)
			var w widget.Widget
			if name == "Raw" {
				e := NewEditor(nil, nil)
				e.UseBuffer(b)
				e.SetFocused(true)
				w = e
			} else {
				p := NewPage()
				p.UseBuffer(b)
				p.SetBlocks(doc.Blocks([]byte(source)))
				p.SetFocused(true)
				w = p
			}
			uitest.LayoutWidget(w, 220, 180)
			w.(interface{ SetBounds(geometry.Rect) }).SetBounds(geometry.NewRect(0, 0, 220, 180))
			uitest.DrawWidget(w)
			ctx := uitest.NewMockContext()
			keys := func(s string) {
				for _, r := range s {
					w.Event(ctx, event.NewKeyEvent(event.KeyPress, event.KeyUnknown, r, 0))
				}
			}
			keys("gj")
			if b.CaretLine() != 0 || b.CaretCol() < 1 {
				t.Fatalf("gj at %d:%d", b.CaretLine(), b.CaretCol())
			}
			keys("gk")
			if b.Cursor() != 0 {
				t.Fatalf("gk at %d", b.Cursor())
			}
			keys("j")
			if b.CaretLine() != 1 {
				t.Fatal("j failed to cross the source newline")
			}
		})
	}
}
