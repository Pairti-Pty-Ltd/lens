package ui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/term"
)

type fakeShell struct {
	view term.View
	got  []byte
}

func (f *fakeShell) View() term.View      { return f.view }
func (f *fakeShell) ViewAt(int) term.View { return f.view }
func (f *fakeShell) Resize(int, int)      {}
func (f *fakeShell) Write(p []byte)       { f.got = append(f.got, p...) }
func (f *fakeShell) Paste(text string)    { f.got = append(f.got, text...) }
func (f *fakeShell) Close()               {}

func cells(text string) term.View {
	v := term.View{Cols: len(text), Rows: 1, Gen: 1, CurOn: true}
	v.Cells = make([]term.Cell, len(text))
	for i, r := range text {
		v.Cells[i] = term.Cell{Text: string(r), Width: 1}
	}
	return v
}

func TestConsolePaintsCommandOutput(t *testing.T) {
	c := NewConsole()
	sh := &fakeShell{view: cells("PAIRTI_OUT")}
	c.sess = sh
	c.open = true
	if !c.Pump() || !c.NeedsRedraw() {
		t.Fatal("new text did not ask for a redraw")
	}
	if c.Pump() {
		t.Fatal("unchanged text redrawn")
	}
	c.SetBounds(geometry.NewRect(0, 40, 400, consoleH))
	canvas := uitest.DrawWidget(c)
	var saw bool
	for _, g := range glyphs(canvas) {
		if g.text == "PAIRTI_OUT" {
			saw = true
		}
	}
	if !saw {
		t.Fatal("output line was not drawn")
	}
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "PAIRTI_OUT" && tx.Style.FontFamily != ShellFace() {
			t.Fatalf("shell face %q", tx.Style.FontFamily)
		}
	}
	sh.view.Gen = 2
	if !c.Pump() {
		t.Fatal("later output was ignored")
	}
}

func TestEnterSendsAReturn(t *testing.T) {
	c := NewConsole()
	c.open = true
	c.SetFocused(true)
	sh := &fakeShell{}
	c.sess = sh
	ev := event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, 0)
	if !c.Event(nil, ev) || string(sh.got) != "\r" {
		t.Fatalf("enter %q", sh.got)
	}
}

func TestConsoleWheelScrollsBack(t *testing.T) {
	c := NewConsole()
	c.open = true
	c.sess = &fakeShell{view: term.View{Cols: 4, Rows: 2, Hist: 20, Gen: 1}}
	c.SetBounds(geometry.NewRect(0, 0, 200, consoleH))
	up := event.NewWheelEvent(geometry.Pt(0, -3), geometry.Pt(20, 40), geometry.Pt(20, 40), 0)
	if !c.Event(nil, up) || c.back < 1 {
		t.Fatalf("back %d", c.back)
	}
	down := event.NewWheelEvent(geometry.Pt(0, 30), geometry.Pt(20, 40), geometry.Pt(20, 40), 0)
	if !c.Event(nil, down) || c.back != 0 {
		t.Fatalf("back after down %d", c.back)
	}
}

func TestFocusedShellKeepsSlash(t *testing.T) {
	c := NewConsole()
	c.open = true
	c.SetFocused(true)
	sh := &fakeShell{}
	c.sess = sh
	slash := event.NewKeyEvent(event.KeyPress, event.KeySlash, 0, 0)
	if !c.Event(nil, slash) || len(sh.got) != 0 {
		t.Fatalf("slash key wrote %q", sh.got)
	}
	text := event.NewKeyEvent(event.KeyPress, event.KeyUnknown, '/', 0)
	if !c.Event(nil, text) || string(sh.got) != "/" {
		t.Fatalf("slash text %q", sh.got)
	}
	if !c.GrabsKey(event.KeyTab) {
		t.Fatal("focused shell does not keep tab")
	}
	c.SetFocused(false)
	if c.Event(nil, slash) {
		t.Fatal("unfocused shell kept the slash")
	}
	if c.GrabsKey(event.KeyTab) {
		t.Fatal("unfocused shell kept tab")
	}
	c.SetFocused(true)
	ctrl := event.NewKeyEvent(event.KeyPress, event.KeyC, 0, event.ModCtrl)
	sh.got = nil
	if !c.Event(nil, ctrl) || string(sh.got) != "\x03" {
		t.Fatalf("ctrl-c %q", sh.got)
	}
}

func TestConsoleDragGrowsThePane(t *testing.T) {
	c := NewConsole()
	c.open = true
	c.sess = &fakeShell{}
	c.SetBounds(geometry.NewRect(0, 300, 400, consoleH))
	ctx := uitest.NewMockContext()
	press := event.NewMouseEvent(event.MousePress, event.ButtonLeft, event.ButtonStateLeft,
		geometry.Pt(20, 302), geometry.Pt(20, 302), 0)
	if !c.Event(ctx, press) {
		t.Fatal("edge press was not consumed")
	}
	if ctx.Cursor() != widget.CursorResizeNS {
		t.Fatalf("cursor %v", ctx.Cursor())
	}
	drag := event.NewMouseEvent(event.MouseDrag, event.ButtonLeft, event.ButtonStateLeft,
		geometry.Pt(20, 202), geometry.Pt(20, 202), 0)
	if !c.Event(ctx, drag) {
		t.Fatal("drag was not consumed")
	}
	size := c.Layout(nil, geometry.Constraints{MaxWidth: 400, MaxHeight: 800})
	if size.Height < 240 {
		t.Fatalf("height %v", size.Height)
	}
}

func TestConsolePasteWritesClipboardToTerminal(t *testing.T) {
	c := NewConsole()
	sh := &fakeShell{}
	c.sess = sh
	c.SetOpen(true)
	c.Paste("café\nsecond line")
	if string(sh.got) != "café\nsecond line" {
		t.Fatalf("shell input %q", sh.got)
	}
}
