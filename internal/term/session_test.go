package term

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestQueryIsAnswered(t *testing.T) {
	s := &Session{}
	s.attach(newScreen(40, 12))
	got := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 256)
		n, err := s.term.Read(buf)
		if err != nil && n == 0 {
			got <- nil
			return
		}
		got <- append([]byte(nil), buf[:n]...)
	}()
	_, _ = s.term.Write([]byte("\x1b[c"))
	select {
	case b := <-got:
		if len(b) == 0 || b[0] != 0x1b {
			t.Fatalf("reply %q", b)
		}
	case <-time.After(time.Second):
		t.Fatal("device attributes query was not answered")
	}
}

func TestCursorAddressLandsOnTheCell(t *testing.T) {
	s := &Session{}
	s.attach(newScreen(40, 12))
	_, _ = s.term.Write([]byte("\x1b[2J\x1b[H\x1b[5;10HX"))
	v := s.View()
	cell := v.Cells[4*v.Cols+9]
	if cell.Text != "X" {
		t.Fatalf("cell %q view %q", cell.Text, s.Text())
	}
}

func TestColourAndShellMarksStayOffTheGrid(t *testing.T) {
	s := &Session{}
	s.attach(newScreen(40, 8))
	_, _ = s.term.Write([]byte("\x1b[31mred\x1b[0m\x1b]133;A\x07\x1b]7;file://host/tmp\x07\r\n"))
	if strings.Contains(s.Text(), "133") || strings.Contains(s.Text(), "[31") {
		t.Fatalf("escapes painted %q", s.Text())
	}
	if !strings.Contains(s.Text(), "red") {
		t.Fatalf("text %q", s.Text())
	}
	if !strings.Contains(s.cwd, "/tmp") {
		t.Fatalf("cwd %q", s.cwd)
	}
	v := s.View()
	var saw bool
	for _, c := range v.Cells {
		if c.Text == "r" && c.Fg != nil {
			saw = true
		}
	}
	if !saw {
		t.Fatal("red had no colour")
	}
}

func TestScrollbackKeepsTheLines(t *testing.T) {
	s := &Session{}
	s.attach(newScreen(10, 3))
	for i := 0; i < 6; i++ {
		_, _ = s.term.Write([]byte("L" + string(rune('0'+i)) + "\r\n"))
	}
	live := s.ViewAt(0)
	older := s.ViewAt(live.Hist)
	if older.Hist < 1 || older.Back != live.Hist {
		t.Fatalf("hist %d back %d", older.Hist, older.Back)
	}
	if lineOf(older, 0) == lineOf(live, 0) || !strings.Contains(lineOf(older, 0), "L0") {
		t.Fatalf("scrolled %q live %q", lineOf(older, 0), lineOf(live, 0))
	}
}

func lineOf(v View, y int) string {
	if y < 0 || y >= v.Rows || v.Cols == 0 {
		return ""
	}
	var b strings.Builder
	for x := 0; x < v.Cols; x++ {
		b.WriteString(v.Cells[y*v.Cols+x].Text)
	}
	return strings.TrimRight(b.String(), " ")
}

func TestResizeChangesTheGrid(t *testing.T) {
	s := &Session{}
	s.attach(newScreen(80, 24))
	s.cols, s.rows = 80, 24
	s.Resize(100, 30)
	if s.term.Width() != 100 || s.term.Height() != 30 {
		t.Fatalf("grid %dx%d", s.term.Width(), s.term.Height())
	}
}

func TestSessionPrintsALine(t *testing.T) {
	s, err := Start(t.TempDir(), nil)
	if err != nil {
		t.Skip(err)
	}
	defer s.Close()
	s.Write([]byte("printf '%s\\n' pairti-lens-shell\r"))
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if hasLine(s.Text(), "pairti-lens-shell") {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("transcript %q", s.Text())
}

func hasLine(text, want string) bool {
	for _, line := range strings.Split(text, "\n") {
		if line == want {
			return true
		}
	}
	return false
}

func TestTerminalPasteRespectsRequestedMode(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	s := &Session{f: w}
	s.attach(newScreen(20, 5))
	defer s.term.Close()
	for _, tc := range []struct{ mode, want string }{{"", "café\n"}, {"\x1b[?2004h", "\x1b[200~café\n\x1b[201~"}, {"\x1b[?2004l", "café\n"}} {
		_, _ = s.term.Write([]byte(tc.mode))
		s.Paste("café\n")
		buf := make([]byte, len(tc.want))
		_, err := io.ReadFull(r, buf)
		if err != nil || string(buf) != tc.want {
			t.Fatalf("mode %q: %q, %v", tc.mode, buf, err)
		}
	}
}
