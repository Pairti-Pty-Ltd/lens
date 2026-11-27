package edit

import (
	"fmt"
	"strings"
	"testing"
)

func TestViewMotionsAndModes(t *testing.T) {
	b := New("alpha beta\ngamma\n")
	var m Modal
	m.Handle(b, Key{Rune: 'w'})
	if b.CaretLine() != 0 || b.CaretCol() != 6 {
		t.Fatalf("word %d:%d", b.CaretLine(), b.CaretCol())
	}
	m.Handle(b, Key{Rune: '0'})
	if b.CaretCol() != 0 {
		t.Fatalf("start %d", b.CaretCol())
	}
	m.Handle(b, Key{Rune: '$'})
	if got := string(b.r[b.pos]); got != "a" || b.CaretCol() != 9 {
		t.Fatalf("end %q col %d", got, b.CaretCol())
	}
	m.Handle(b, Key{Rune: 'j'})
	if b.CaretLine() != 1 {
		t.Fatalf("line %d", b.CaretLine())
	}
	m.Handle(b, Key{Rune: '0'})
	m.Handle(b, Key{Rune: 'x'})
	if b.Text() != "alpha beta\namma\n" {
		t.Fatalf("x %q", b.Text())
	}
	m.Handle(b, Key{Rune: 'g'})
	m.Handle(b, Key{Rune: 'g'})
	if b.CaretLine() != 0 || b.CaretCol() != 0 {
		t.Fatalf("gg %d:%d", b.CaretLine(), b.CaretCol())
	}
	m.Handle(b, Key{Rune: 'd'})
	if _, ok := m.Handle(b, Key{Rune: 'd'}); !ok {
		t.Fatal("dd was not used")
	}
	if b.Text() != "amma\n" {
		t.Fatalf("dd %q", b.Text())
	}
	m.Handle(b, Key{Rune: 'i'})
	if m.Name() != "Edit" {
		t.Fatalf("mode %s", m.Name())
	}
	m.Handle(b, Key{Rune: 'Z'})
	if b.Text() != "Zamma\n" {
		t.Fatalf("insert %q", b.Text())
	}
	m.Handle(b, Key{Code: CodeEsc})
	if m.Name() != "View" {
		t.Fatalf("after esc %s", m.Name())
	}
	res, _ := m.Handle(b, Key{Rune: ':'})
	if m.Name() != "Command" || res.Cmd != "" {
		t.Fatalf("command mode %s %q", m.Name(), res.Cmd)
	}
	m.Handle(b, Key{Rune: 'w'})
	res, _ = m.Handle(b, Key{Code: CodeEnter})
	if res.Cmd != "w" || m.Name() != "View" {
		t.Fatalf("ran %q mode %s", res.Cmd, m.Name())
	}
	res, _ = m.Handle(b, Key{Code: CodeEsc})
	if m.Mode != ModeView || res.Changed {
		t.Fatal("view escape should stay in navigation")
	}
}

func TestPageAndWindowKeys(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	buf := New(b.String())
	var m Modal
	res, ok := m.Handle(buf, Key{Rune: 'f', Ctrl: true, Rows: 10})
	if !ok || res.Scroll != 8 || buf.CaretLine() != 8 {
		t.Fatalf("ctrl-f scroll %d line %d", res.Scroll, buf.CaretLine())
	}
	res, ok = m.Handle(buf, Key{Rune: 'u', Ctrl: true, Rows: 10})
	if !ok || res.Scroll != -5 || buf.CaretLine() != 3 {
		t.Fatalf("ctrl-u scroll %d line %d", res.Scroll, buf.CaretLine())
	}
	m.Handle(buf, Key{Rune: 'w', Ctrl: true})
	res, ok = m.Handle(buf, Key{Rune: 'l', Ctrl: true})
	if !ok || res.Win != 'l' {
		t.Fatalf("ctrl-w l %+v", res)
	}
	res, _ = m.Handle(buf, Key{Rune: 'z'})
	res, _ = m.Handle(buf, Key{Rune: 't'})
	if res.Place != 't' {
		t.Fatalf("zt %q", res.Place)
	}
}

func TestOperatorsCoverLinesAndMotions(t *testing.T) {
	b := New("alpha beta\ngamma delta\n")
	var m Modal
	m.Handle(b, Key{Rune: 'd'})
	m.Handle(b, Key{Rune: 'd'})
	if b.Text() != "gamma delta\n" {
		t.Fatalf("dd %q", b.Text())
	}
	m.Handle(b, Key{Rune: '0'})
	m.Handle(b, Key{Rune: 'd'})
	m.Handle(b, Key{Rune: 'w'})
	if b.Text() != "delta\n" {
		t.Fatalf("dw %q", b.Text())
	}
	m.Handle(b, Key{Rune: 'u'})
	if b.Text() != "gamma delta\n" {
		t.Fatalf("undo dw %q", b.Text())
	}
	m.Handle(b, Key{Rune: 'D'})
	if b.Text() != "\n" && b.Text() != "" {
		t.Fatalf("D %q", b.Text())
	}
	b = New("one\ntwo\n")
	m = Modal{}
	m.Handle(b, Key{Rune: 'y'})
	m.Handle(b, Key{Rune: 'y'})
	m.Handle(b, Key{Rune: 'p'})
	if b.Text() != "one\none\ntwo\n" {
		t.Fatalf("yyp %q", b.Text())
	}
	b = New("word")
	m = Modal{}
	m.Handle(b, Key{Rune: 'c'})
	m.Handle(b, Key{Rune: 'w'})
	if m.Name() != "Edit" || b.Text() != "" {
		t.Fatalf("cw mode %s text %q", m.Name(), b.Text())
	}
}

func TestUndoAndRedoATypingSession(t *testing.T) {
	b := New("go")
	var m Modal
	m.Handle(b, Key{Rune: 'A'})
	m.Handle(b, Key{Rune: 'o'})
	m.Handle(b, Key{Rune: 'd'})
	m.Handle(b, Key{Code: CodeEsc})
	if b.Text() != "good" {
		t.Fatalf("typed %q", b.Text())
	}
	m.Handle(b, Key{Rune: 'u'})
	if b.Text() != "go" {
		t.Fatalf("undo %q", b.Text())
	}
	m.Handle(b, Key{Ctrl: true, Rune: 'r'})
	if b.Text() != "good" {
		t.Fatalf("ctrl-r %q", b.Text())
	}
	m.Handle(b, Key{Super: true, Rune: 'z'})
	if b.Text() != "go" {
		t.Fatalf("cmd-z %q", b.Text())
	}
	m.Handle(b, Key{Super: true, Shift: true, Rune: 'z'})
	if b.Text() != "good" || b.Cursor() != 3 {
		t.Fatalf("cmd-shift-z %q caret %d", b.Text(), b.Cursor())
	}
	m.Handle(b, Key{Rune: 'x'})
	if b.Redo() {
		t.Fatal("buffer redo survived x")
	}
}

func TestModalOperatorRanges(t *testing.T) {
	for _, tc := range []struct{ source, keys, want string }{
		{"one two three four five", "2d2w", "five"},
		{"one two three", "wldaw", "one three"},
		{"one two", "wevbhcA\x1b", "oneA"},
		{"one\ntwo", "Gddp", "one\ntwo"},
		{"one\ntwo", "GyyP", "one\ntwo\ntwo"},
		{"one\ntwo\nthree", "Vjcnew\x1b", "new\nthree"},
		{"one\ntwo\nthree", "jVkd", "three"},
		{"one two", "3rx", "xxx two"},
		{"\nnext", "Ctext\x1b", "text\nnext"},
	} {
		t.Run(tc.keys, func(t *testing.T) {
			b := New(tc.source)
			for _, r := range tc.keys {
				k := Key{Rune: r}
				if r == 27 {
					k = Key{Code: CodeEsc}
				}
				b.Modal.Handle(b, k)
			}
			if got := b.Text(); got != tc.want || b.Modal.Mode != ModeView {
				t.Fatalf("got %q in %s, want %q in View", got, b.Modal.Name(), tc.want)
			}
		})
	}
}

func TestNavigationTargets(t *testing.T) {
	src := "one [a(b)c] two\n\nrepeat\n\nrepeat"
	for _, tc := range []struct {
		keys      string
		line, col int
	}{
		{"%", 0, 10}, {"f(%;", 0, 8}, {"f];", 0, 10},
		{"}", 1, 0}, {"2}", 3, 0}, {"G{", 3, 0},
		{"/repeat\nn", 4, 0}, {"/repeat\nnN", 2, 0}, {"?repeat\n", 4, 0},
		{":999\n", 4, 0}, {":2\n", 1, 0},
	} {
		b := New(src)
		for _, r := range tc.keys {
			k := Key{Rune: r}
			if r == '\n' {
				k = Key{Code: CodeEnter}
			}
			b.Modal.Handle(b, k)
		}
		if b.CaretLine() != tc.line || b.CaretCol() != tc.col {
			t.Errorf("%q got %d:%d want %d:%d", tc.keys, b.CaretLine(), b.CaretCol(), tc.line, tc.col)
		}
	}
	b := New("a,a,a,a")
	for _, r := range "f,;;," {
		b.Modal.Handle(b, Key{Rune: r})
	}
	if b.CaretCol() != 3 {
		t.Errorf("repeat find landed at %d", b.CaretCol())
	}
}
