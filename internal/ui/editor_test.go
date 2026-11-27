package ui

import (
	"fmt"
	"github.com/gogpu/ui/core/textfield"
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/theme"
)

func TestEditorTypesAndDraws(t *testing.T) {
	var edits int
	ed := NewEditor(func() { edits++ }, nil)
	ed.SetBuffer("# Hi")
	uitest.LayoutWidget(ed, 400, 200)
	ed.SetBounds(geometry.NewRect(0, 40, 400, 200))

	ctx := uitest.NewMockContext()
	if !ed.Event(ctx, uitest.Click(8, 48)) {
		t.Fatal("click was not consumed")
	}
	if !ed.IsFocused() {
		t.Fatal("editor did not take focus")
	}
	if ed.ModeName() != "View" {
		t.Fatalf("mode %s", ed.ModeName())
	}
	if !ed.Event(ctx, uitest.KeyType(event.KeyI, 'i', 0)) {
		t.Fatal("i was not consumed")
	}
	key := event.NewKeyEvent(event.KeyPress, event.KeyA, '!', event.ModNone)
	if !ed.Event(ctx, key) {
		t.Fatal("key was not consumed")
	}
	if ed.Buffer().Text() != "!# Hi" || edits < 1 {
		t.Fatalf("text %q edits %d", ed.Buffer().Text(), edits)
	}

	lines := glyphs(uitest.DrawWidget(ed))
	var sawText, sawNum bool
	for _, ln := range lines {
		if ln.text == ed.Buffer().Text() {
			sawText = true
		}
		if ln.text == "1" {
			sawNum = true
		}
	}
	if !sawText || !sawNum {
		t.Fatalf("drew %+v, buffer %q", lines, ed.Buffer().Text())
	}
	if len(uitest.DrawWidget(ed).Rects) == 0 {
		t.Fatal("no caret drawn")
	}
}

func TestEditorClickActivatesItsPane(t *testing.T) {
	n := 0
	ed := NewEditor(nil, nil)
	ed.SetActivate(func() { n++ })
	ed.SetBounds(geometry.NewRect(0, 0, 200, 40))
	if !ed.Event(uitest.NewMockContext(), uitest.Click(8, 8)) || n != 1 {
		t.Fatalf("activations %d", n)
	}
}

func TestEditorDragDeletesTheSelection(t *testing.T) {
	ed := NewEditor(nil, nil)
	ed.UseBuffer(edit.New("hello\n"))
	uitest.LayoutWidget(ed, 400, 80)
	ed.SetBounds(geometry.NewRect(0, 0, 400, 80))
	ctx := uitest.NewMockContext()
	press := &event.MouseEvent{MouseType: event.MousePress, Button: event.ButtonLeft, Position: geometry.Pt(editGutter+1, 4)}
	drag := &event.MouseEvent{MouseType: event.MouseDrag, Button: event.ButtonLeft, Position: geometry.Pt(editGutter+2*ed.charW()+0.1, 4)}
	if !ed.Event(ctx, press) || !ed.Event(ctx, drag) {
		t.Fatal("drag was not consumed")
	}
	if !ed.Event(ctx, uitest.KeyType(event.KeyX, 'x', 0)) {
		t.Fatal("x was not consumed")
	}
	if ed.Buffer().Text() != "llo\n" {
		t.Fatalf("text %q", ed.Buffer().Text())
	}
}

func TestEditorSaveChord(t *testing.T) {
	saved := false
	ed := NewEditor(nil, func() { saved = true })
	ed.SetBuffer("a")
	ed.SetBounds(geometry.NewRect(0, 0, 100, 40))
	ctx := uitest.NewMockContext()
	ed.Event(ctx, uitest.Click(8, 4))
	chord := event.NewKeyEvent(event.KeyPress, event.KeyS, 's', event.ModSuper)
	if !ed.Event(ctx, chord) {
		t.Fatal("save chord was not consumed")
	}
	if !saved {
		t.Fatal("save was not called")
	}
	if ed.Buffer().Text() != "a" {
		t.Fatalf("chord inserted text %q", ed.Buffer().Text())
	}
}

func TestCtrlFScrollsTheEditor(t *testing.T) {
	ed := NewEditor(nil, nil)
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	ed.SetBuffer(b.String())
	ed.SetFocused(true)
	lh := ed.lineH()
	ed.SetBounds(geometry.NewRect(0, 0, 400, lh*10))
	key := event.NewKeyEvent(event.KeyPress, event.KeyF, 0, event.ModCtrl)
	if !ed.Event(nil, key) {
		t.Fatal("ctrl-f was not used")
	}
	if ed.Buffer().CaretLine() != 8 {
		t.Fatalf("caret %d", ed.Buffer().CaretLine())
	}
	if ed.scroll < lh*7 {
		t.Fatalf("scroll %v", ed.scroll)
	}
	ed.scroll = lh * 20
	ed.Buffer().GoLine(1)
	if !ed.Event(nil, key) {
		t.Fatal("ctrl-f from a scrolled view was not used")
	}
	if ed.scroll < lh*27 {
		t.Fatalf("ctrl-f snapped back to %v", ed.scroll)
	}
	if ed.Buffer().CaretLine() < 20 {
		t.Fatalf("caret jumped to %d", ed.Buffer().CaretLine())
	}
}

func TestCtrlEAndYScrollOneLine(t *testing.T) {
	ed := NewEditor(nil, nil)
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	ed.SetBuffer(b.String())
	ed.SetFocused(true)
	ed.Buffer().GoLine(8)
	lh := ed.lineH()
	ed.SetBounds(geometry.NewRect(0, 0, 400, lh*10))
	ed.scroll = lh * 2
	down := event.NewKeyEvent(event.KeyPress, event.KeyE, 0x05, event.ModCtrl)
	if !ed.Event(nil, down) {
		t.Fatal("ctrl-e was not used")
	}
	if ed.scroll != lh*3 || ed.Buffer().CaretLine() != 7 {
		t.Fatalf("after ctrl-e scroll %v caret %d", ed.scroll, ed.Buffer().CaretLine())
	}
	up := event.NewKeyEvent(event.KeyPress, event.KeyY, 'Y', event.ModCtrl)
	if !ed.Event(nil, up) {
		t.Fatal("ctrl-y was not used")
	}
	if ed.scroll != lh*2 || ed.Buffer().CaretLine() != 7 {
		t.Fatalf("after ctrl-y scroll %v caret %d", ed.scroll, ed.Buffer().CaretLine())
	}
}

func TestEditorColoursMarkdown(t *testing.T) {
	ed := NewEditor(nil, nil)
	ed.SetFile("note.md")
	ed.SetBuffer("# Title\n\nA **bold** word.\n")
	uitest.LayoutWidget(ed, 480, 200)
	ed.SetBounds(geometry.NewRect(0, 0, 480, 200))
	canvas := uitest.DrawWidget(ed)
	var heading, strong bool
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "# Title" && tx.Style.Bold && tx.Style.Color == rgb(theme.Pairti().Accent) {
			heading = true
		}
		if tx.Text == "**bold**" && tx.Style.Bold {
			strong = true
		}
	}
	if !heading || !strong {
		t.Fatalf("heading %v strong %v", heading, strong)
	}
}

func TestShowLineScrollsTheRawSource(t *testing.T) {
	ed := NewEditor(nil, nil)
	var b strings.Builder
	for i := 0; i < 80; i++ {
		b.WriteString("line\n")
	}
	b.WriteString("## Later\n")
	ed.SetBuffer(b.String())
	uitest.LayoutWidget(ed, 400, 100)
	ed.SetBounds(geometry.NewRect(0, 0, 400, 100))
	ed.ShowLine(81)
	if ed.Buffer().CaretLine() != 80 {
		t.Fatalf("caret line %d", ed.Buffer().CaretLine())
	}
	if ed.scroll < 70*ed.lineH() {
		t.Fatalf("scroll %v", ed.scroll)
	}
}

func TestLineNumberToggleReclaimsGutter(t *testing.T) {
	page := NewPage()
	page.SetBlocks(doc.Blocks([]byte("# Heading\n")))
	formatted := func() *uitest.MockCanvas { return drawPage(page, 500, 150) }
	off := formatted()
	left := glyphBox(off, "Heading").Min.X
	if hasGlyph(glyphs(off), "h1") || hasGlyph(glyphs(off), "1") {
		t.Fatal("formatted gutter is visible by default")
	}
	page.SetLineNumbers(true)
	on := formatted()
	if !hasGlyph(glyphs(on), "h1") || !hasGlyph(glyphs(on), "1") || glyphBox(on, "Heading").Min.X <= left {
		t.Fatal("formatted gutter did not appear")
	}
	editor := NewEditor(nil, nil)
	editor.SetBuffer("sample")
	editor.SetBounds(geometry.NewRect(0, 0, 400, 100))
	if !hasGlyph(glyphs(uitest.DrawWidget(editor)), "1") {
		t.Fatal("raw gutter is hidden by default")
	}
	editor.SetLineNumbers(false)
	without := uitest.DrawWidget(editor)
	if hasGlyph(glyphs(without), "1") || glyphBox(without, "sample").Min.X >= editGutter {
		t.Fatal("raw gutter was not reclaimed")
	}
}

func TestRawWrapKeepsSyntaxAndSourceCoordinates(t *testing.T) {
	ed := NewEditor(nil, nil)
	text := "å **several highlighted words** after"
	ed.SetBuffer(text)
	ed.SetFile("notes.md")
	width := ed.gutterWidth() + ed.charW()*14
	ed.SetBounds(geometry.NewRect(0, 0, width, 200))
	rows := ed.visualRows()
	if len(rows) < 3 {
		t.Fatalf("rows %v", rows)
	}
	var joined strings.Builder
	var strong strings.Builder
	for i, row := range rows {
		if i > 0 && row.col != rows[i-1].col+rows[i-1].length {
			t.Fatalf("row gap %v", row)
		}
		joined.WriteString(row.text)
		for _, sp := range row.spans {
			if sp.Bold {
				strong.WriteString(sp.Text)
			}
		}
	}
	if joined.String() != text || strong.String() != "**several highlighted words**" {
		t.Fatalf("source %q, emphasis %q", joined.String(), strong.String())
	}
	ctx := uitest.NewMockContext()
	x := ed.gutterWidth() + ed.charW()*2.2
	ed.Event(ctx, uitest.Click(x, ed.lineH()+2))
	if ed.buf.CaretCol() != rows[1].col+2 {
		t.Fatalf("wrapped click at %d", ed.buf.CaretCol())
	}
	ed.Event(ctx, uitest.KeyType(event.KeyI, 'i', 0))
	ed.Event(ctx, uitest.KeyType(event.KeyDown, 0, 0))
	if ed.buf.CaretCol() != rows[2].col+2 {
		t.Fatalf("wrapped arrow at %d", ed.buf.CaretCol())
	}
	at := ed.buf.Cursor()
	ed.Event(ctx, uitest.KeyType(event.KeyX, 'x', 0))
	rs := []rune(text)
	if ed.buf.Text() != string(rs[:at])+"x"+string(rs[at:]) {
		t.Fatalf("edit moved source: %q", ed.buf.Text())
	}
	ed.SetFile("code.go")
	if n := len(ed.visualRows()); n != 1 {
		t.Fatalf("code file used %d rows", n)
	}
}

func TestRawPassageAfterViewportReflow(t *testing.T) {
	ed := NewEditor(nil, nil)
	ed.SetBuffer(strings.Repeat("å word ", 120) + "\ntarget\n" + strings.Repeat("rest\n", 10))
	ed.buf.Place(1, 2)
	ed.RestorePassage(Passage{Line: 2, Offset: 0})
	uitest.LayoutWidget(ed, 180, 80)
	ed.SetBounds(geometry.NewRect(0, 0, 180, 80))
	if got := ed.Passage(); got.Line != 2 || got.Offset != 0 {
		t.Fatalf("passage %+v, scroll %v", got, ed.scroll)
	}
	if ed.buf.CaretCol() != 2 {
		t.Fatal("restoring passage moved the caret")
	}
}

func TestSidebarFocusKeepsKeysOutOfSplitEditors(t *testing.T) {
	ctx := uitest.NewMockContext()
	editor, page := NewEditor(nil, nil), NewPage()
	page.UseBuffer(edit.New("original"))
	editor.SetBuffer("original")
	ctx.RequestFocus(textfield.New())
	editor.SetFocused(true)
	page.SetFocused(true)
	key := uitest.KeyType(event.KeyI, 'i', 0)
	if editor.Event(ctx, key) || page.editKey(ctx, key) {
		t.Fatal("inactive document consumed a search keystroke")
	}
}
