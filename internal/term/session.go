// Package term runs the user's shell in a pseudoterminal.
package term

import (
	"os"
	"os/exec"
	"sync"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/vt"
	"github.com/creack/pty"
)

// Session is one interactive shell. The screen is a cell grid, not a log.
type Session struct {
	mu             sync.Mutex
	f              *os.File
	cmd            *exec.Cmd
	term           *vt.Emulator
	cols           int
	rows           int
	gen            uint64
	cwd            string
	mark           string
	wake           func()
	writeMu        sync.Mutex
	bracketedPaste bool
	toPty          chan []byte
}

// Start launches $SHELL, or /bin/sh, with its working directory at dir.
func Start(dir string, wake func()) (*Session, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-i")
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 80, Rows: 24})
	if err != nil {
		return nil, err
	}
	s := &Session{
		f: f, cmd: cmd, wake: wake, cols: 80, rows: 24,
		toPty: make(chan []byte, 64),
	}
	s.attach(newScreen(80, 24))
	// Replies are an unbuffered pipe. The reader only queues them. A second
	// loop writes the pty, so a slow write cannot stall the parser while the
	// screen lock is held. That stall is the beach ball.
	go s.pumpPty()
	go s.reply()
	go s.read()
	return s, nil
}

// Write sends keystrokes to the shell.
func (s *Session) Write(p []byte) {
	s.writePty(p)
}

// Paste writes one PTY message, respecting the mode requested by the foreground program.
func (s *Session) Paste(text string) {
	s.mu.Lock()
	bracketed := s.bracketedPaste
	s.mu.Unlock()
	if bracketed {
		text = ansi.BracketedPasteStart + text + ansi.BracketedPasteEnd
	}
	s.writePty([]byte(text))
}

func (s *Session) writePty(p []byte) {
	if s == nil || s.f == nil || len(p) == 0 {
		return
	}
	s.writeMu.Lock()
	_, _ = s.f.Write(p)
	s.writeMu.Unlock()
}

// Resize tells the emulator and the process the grid size.
func (s *Session) Resize(cols, rows int) {
	if s == nil || s.term == nil {
		return
	}
	if cols < 2 {
		cols = 2
	}
	if rows < 2 {
		rows = 2
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cols == s.cols && rows == s.rows {
		return
	}
	s.cols, s.rows = cols, rows
	s.term.Resize(cols, rows)
	if s.f != nil {
		_ = pty.Setsize(s.f, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}
}

// Close stops the shell.
func (s *Session) Close() {
	if s == nil {
		return
	}
	if s.term != nil {
		_ = s.term.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	if s.f != nil {
		_ = s.f.Close()
	}
}

func (s *Session) read() {
	tmp := make([]byte, 4096)
	for {
		n, err := s.f.Read(tmp)
		if n > 0 {
			s.mu.Lock()
			_, _ = s.term.Write(tmp[:n])
			s.gen++
			s.mu.Unlock()
			if s.wake != nil {
				s.wake()
			}
		}
		if err != nil {
			return
		}
	}
}

// reply drains terminal answers off the unbuffered pipe. It must not write
// the pty itself: that write can block, and the parser is waiting on this read.
func (s *Session) reply() {
	buf := make([]byte, 4096)
	for {
		n, err := s.term.Read(buf)
		if n > 0 {
			b := append([]byte(nil), buf[:n]...)
			s.toPty <- b
		}
		if err != nil {
			close(s.toPty)
			return
		}
	}
}

func (s *Session) pumpPty() {
	for b := range s.toPty {
		s.writePty(b)
	}
}
