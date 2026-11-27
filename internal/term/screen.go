package term

import (
	"image/color"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
)

// Cell is one column of the shell screen.
type Cell struct {
	Text  string
	Fg    color.Color
	Bg    color.Color
	Bold  bool
	Width int
}

// View is a copy of the screen the panel can draw without holding the pty.
// Back is how many lines above the live bottom the copy starts. Hist is how
// many lines are stored above the live screen.
type View struct {
	Cols, Rows int
	Cells      []Cell
	CurX, CurY int
	CurOn      bool
	Alt        bool
	Cwd        string
	Gen        uint64
	Hist       int
	Back       int
}

func newScreen(cols, rows int) *vt.Emulator {
	if cols < 2 {
		cols = 2
	}
	if rows < 2 {
		rows = 2
	}
	return vt.NewEmulator(cols, rows)
}

func (s *Session) attach(term *vt.Emulator) {
	s.term = term
	term.SetCallbacks(vt.Callbacks{
		WorkingDirectory: func(path string) { s.cwd = path },
		EnableMode: func(mode ansi.Mode) {
			if mode == ansi.ModeBracketedPaste {
				s.bracketedPaste = true
			}
		},
		DisableMode: func(mode ansi.Mode) {
			if mode == ansi.ModeBracketedPaste {
				s.bracketedPaste = false
			}
		},
	})
	// Shell integration marks (prompt, command, exit). They are not cells.
	term.RegisterOscHandler(133, func(data []byte) bool {
		s.mark = string(data)
		return true
	})
	term.RegisterOscHandler(633, func([]byte) bool { return true })
}

// View copies the live screen.
func (s *Session) View() View { return s.ViewAt(0) }

// ViewAt copies rows lines of the grid, starting back lines above the live
// bottom. Zero follows the cursor. The copy is one frame.
func (s *Session) ViewAt(back int) View {
	if s == nil || s.term == nil {
		return View{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cols, rows := s.term.Width(), s.term.Height()
	hist := s.term.ScrollbackLen()
	if back < 0 {
		back = 0
	}
	if back > hist {
		back = hist
	}
	out := View{
		Cols: cols, Rows: rows,
		Cells: make([]Cell, cols*rows),
		CurOn: !s.term.CursorHidden(),
		Alt:   s.term.IsAltScreen(),
		Cwd:   s.cwd,
		Gen:   s.gen,
		Hist:  hist,
		Back:  back,
	}
	pos := s.term.CursorPosition()
	out.CurX, out.CurY = pos.X, pos.Y+back
	start := hist - back
	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			out.Cells[y*cols+x] = copyCell(s.cellAt(start+y, x, hist, cols))
		}
	}
	if out.CurY < 0 || out.CurY >= rows {
		out.CurOn = false
	}
	return out
}

// cellAt reads one cell. index is into history then the live screen.
func (s *Session) cellAt(index, x, hist, cols int) *uv.Cell {
	if index < hist {
		return s.term.ScrollbackCellAt(x, index)
	}
	return s.term.CellAt(x, index-hist)
}

func copyCell(c *uv.Cell) Cell {
	if c == nil || c.Content == "" {
		return Cell{Text: " ", Width: 1}
	}
	w := c.Width
	if w < 1 {
		w = 1
	}
	return Cell{
		Text:  c.Content,
		Fg:    c.Style.Fg,
		Bg:    c.Style.Bg,
		Bold:  c.Style.Attrs&uv.AttrBold != 0,
		Width: w,
	}
}

// Text is the screen as lines, for tests. Trailing spaces are dropped.
func (s *Session) Text() string {
	v := s.View()
	if v.Rows == 0 || v.Cols == 0 {
		return ""
	}
	var b strings.Builder
	for y := 0; y < v.Rows; y++ {
		var line strings.Builder
		for x := 0; x < v.Cols; {
			c := v.Cells[y*v.Cols+x]
			if c.Width < 1 {
				c.Width = 1
			}
			line.WriteString(c.Text)
			x += c.Width
		}
		b.WriteString(strings.TrimRight(line.String(), " "))
		if y+1 < v.Rows {
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}
