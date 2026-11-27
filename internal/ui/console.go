//go:build !js

package ui

import (
	"image/color"
	"runtime"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/term"
	"pairti-lense/internal/theme"
)

const (
	consoleH    = float32(150)
	consoleMin  = float32(80)
	consoleMax  = float32(720)
	consoleDrag = float32(6)
)

// Console is the project shell. It takes no space until it is open.
// The top edge drags to change the height.
type Console struct {
	widget.WidgetBase
	theme    theme.Theme
	open     bool
	sess     screen
	seen     uint64
	height   float32
	dragging bool
	startY   float32
	startH   float32
	cols     int
	rows     int
	back     int
	hist     int
}

// screen is the cell grid the panel draws and types into.
type screen interface {
	View() term.View
	ViewAt(back int) term.View
	Resize(cols, rows int)
	Write([]byte)
	Paste(string)
	Close()
}

// NewConsole returns a hidden shell panel.
func NewConsole() *Console {
	c := &Console{theme: theme.Pairti()}
	c.SetVisible(true)
	c.SetEnabled(true)
	return c
}

// SetTheme replaces the semantic theme.
func (c *Console) SetTheme(t theme.Theme) {
	c.theme = t
	c.SetNeedsRedraw(true)
}

// Open reports whether the panel is showing.
func (c *Console) Open() bool { return c.open }

// Start runs a shell in dir if one is not already running, and shows it.
func (c *Console) Start(dir string, wake func()) error {
	if c.sess == nil {
		sess, err := term.Start(dir, wake)
		if err != nil {
			return err
		}
		c.sess = sess
	}
	c.open = true
	c.MarkNeedsLayout()
	c.SetNeedsRedraw(true)
	return nil
}

// SetOpen shows or hides the panel without stopping the shell.
func (c *Console) SetOpen(on bool) {
	if c.open == on {
		return
	}
	c.open = on
	c.MarkNeedsLayout()
	c.SetNeedsRedraw(true)
}

// Paste sends text as a terminal paste, honoring bracketed-paste mode.
func (c *Console) Paste(text string) {
	if c.open && c.sess != nil && text != "" {
		c.back = 0
		c.sess.Paste(text)
		c.SetNeedsRedraw(true)
	}
}

// Close stops the shell.
func (c *Console) Close() {
	if c.sess != nil {
		c.sess.Close()
		c.sess = nil
	}
	c.open = false
}

// IsFocusable reports that an open shell takes the keyboard.
func (c *Console) IsFocusable() bool { return c.open && c.IsVisible() && c.IsEnabled() }

// GrabsKey keeps framework keys for the shell. Tab is completion, not focus travel.
func (c *Console) GrabsKey(event.Key) bool { return c.IsFocusable() && c.IsFocused() }

// Layout is the dragged height, or nothing while closed.
func (c *Console) Layout(_ widget.Context, cons geometry.Constraints) geometry.Size {
	w := cons.MaxWidth
	if w <= 0 || w >= geometry.Infinity {
		w = 400
	}
	h := float32(0)
	if c.open {
		h = c.pixelH()
		if cons.MaxHeight > consoleMin && cons.MaxHeight < geometry.Infinity {
			limit := cons.MaxHeight - 80
			if limit < consoleMin {
				limit = consoleMin
			}
			if h > limit {
				h = limit
			}
		}
	}
	return cons.Constrain(geometry.Sz(w, h))
}

func (c *Console) pixelH() float32 {
	h := c.height
	if h < consoleMin {
		h = consoleH
	}
	if h > consoleMax {
		h = consoleMax
	}
	return h
}

// Draw paints the cell grid in the mono face.
func (c *Console) Draw(_ widget.Context, canvas widget.Canvas) {
	if !c.open || c.sess == nil {
		return
	}
	b := c.Bounds()
	if b.Height() < 1 {
		return
	}
	canvas.DrawRect(b, rgb(c.theme.CodeFill))
	canvas.DrawLine(geometry.Pt(b.Min.X, b.Min.Y), geometry.Pt(b.Max.X, b.Min.Y), rgb(c.theme.Rule), 1)
	canvas.PushClip(b)
	defer canvas.PopClip()
	cw, ch := c.cellSize(canvas)
	c.fit(b.Width(), b.Height(), cw, ch)
	v := c.sess.ViewAt(c.back)
	c.stick(v)
	if v.Back != c.back {
		v = c.sess.ViewAt(c.back)
	}
	face := ShellFace()
	ink := rgb(c.theme.Ink)
	for y := 0; y < v.Rows; y++ {
		c.drawRow(canvas, v, y, b.Min.X, b.Min.Y+float32(y)*ch, cw, ch, face, ink)
	}
	if v.CurOn && v.CurX >= 0 && v.CurY >= 0 {
		canvas.DrawRect(geometry.NewRect(
			b.Min.X+float32(v.CurX)*cw, b.Min.Y+float32(v.CurY)*ch+2,
			2, ch-4,
		), rgb(c.theme.Accent))
	}
}

func (c *Console) drawRow(canvas widget.Canvas, v term.View, y int, x0, y0, cw, ch float32, face string, ink widget.Color) {
	type run struct {
		text   string
		fg, bg widget.Color
		bold   bool
		hasBg  bool
		x, w   float32
	}
	var cur *run
	flush := func() {
		if cur == nil || cur.text == "" {
			cur = nil
			return
		}
		if cur.hasBg {
			canvas.DrawRect(geometry.NewRect(x0+cur.x, y0, cur.w, ch), cur.bg)
		}
		drawFace(canvas, cur.text, geometry.NewRect(x0+cur.x, y0, cur.w+1, ch), c.theme.Code, cur.fg, cur.bold, face)
		cur = nil
	}
	x := 0
	for x < v.Cols {
		cell := v.Cells[y*v.Cols+x]
		span := cell.Width
		if span < 1 {
			span = 1
		}
		if cell.Text == "" || cell.Text == " " {
			flush()
			if cell.Bg != nil {
				canvas.DrawRect(geometry.NewRect(x0+float32(x)*cw, y0, float32(span)*cw, ch), cellColor(cell.Bg, ink))
			}
			x += span
			continue
		}
		fg := cellColor(cell.Fg, ink)
		bg := widget.Color{}
		hasBg := cell.Bg != nil
		if hasBg {
			bg = cellColor(cell.Bg, ink)
		}
		px := float32(x) * cw
		pw := float32(span) * cw
		if cur != nil && (cur.bold != cell.Bold || cur.fg != fg || cur.hasBg != hasBg || cur.bg != bg) {
			flush()
		}
		if cur == nil {
			cur = &run{fg: fg, bg: bg, hasBg: hasBg, bold: cell.Bold, x: px}
		}
		cur.text += cell.Text
		cur.w += pw
		x += span
	}
	flush()
}

func (c *Console) cellSize(canvas widget.Canvas) (float32, float32) {
	size := c.theme.Code
	if size < 1 {
		size = 14
	}
	w := textWidth(canvas, "M", size, rgb(c.theme.Ink), false, ShellFace())
	if w < size*0.4 {
		w = size * 0.6
	}
	h := size * 1.35
	if h < 16 {
		h = 16
	}
	return w, h
}

func (c *Console) fit(width, height, cw, ch float32) {
	cols := int(width / cw)
	rows := int(height / ch)
	if cols < 2 {
		cols = 2
	}
	if rows < 2 {
		rows = 2
	}
	if cols == c.cols && rows == c.rows {
		return
	}
	c.cols, c.rows = cols, rows
	c.sess.Resize(cols, rows)
}

// stick keeps a scrolled-up view on the same lines when new output arrives.
func (c *Console) stick(v term.View) {
	if c.back > 0 && v.Hist > c.hist {
		c.back += v.Hist - c.hist
	}
	c.hist = v.Hist
	if c.back > v.Hist {
		c.back = v.Hist
	}
	if c.back < 0 {
		c.back = 0
	}
}

func cellColor(c color.Color, fallback widget.Color) widget.Color {
	if c == nil {
		return fallback
	}
	r, g, b, a := c.RGBA()
	if a == 0 {
		return fallback
	}
	return widget.RGBA(float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535)
}

// Pump marks the panel dirty when the shell has a new frame.
func (c *Console) Pump() bool {
	if c == nil || c.sess == nil {
		return false
	}
	gen := c.sess.View().Gen
	if gen == c.seen {
		return false
	}
	c.seen = gen
	c.SetNeedsRedraw(true)
	return true
}

// Event drags the top edge, and sends keystrokes when the panel is focused.
func (c *Console) Event(ctx widget.Context, e event.Event) bool {
	if !c.open || c.sess == nil {
		return false
	}
	if me, ok := e.(*event.MouseEvent); ok {
		return c.mouse(ctx, me)
	}
	if we, ok := e.(*event.WheelEvent); ok {
		if !c.Bounds().Contains(we.Position) {
			return false
		}
		v := c.sess.ViewAt(c.back)
		if !v.Alt {
			c.scrollBack(WheelScroll(we.Delta.Y, runtime.GOOS == "darwin"), v.Hist)
		}
		return true
	}
	v, ok := e.(*event.KeyEvent)
	if !ok || !c.IsFocused() {
		return false
	}
	// A focused shell owns the key. A press with no character still has to be
	// kept: "/" arrives that way, and letting it through focuses Search.
	if v.KeyType == event.KeyRelease {
		return true
	}
	if p := keyBytes(v); len(p) > 0 {
		c.back = 0
		c.sess.Write(p)
		c.SetNeedsRedraw(true)
	}
	return true
}

func (c *Console) mouse(ctx widget.Context, me *event.MouseEvent) bool {
	local := me.Position.Sub(c.Bounds().Min)
	onEdge := local.Y >= 0 && local.Y <= consoleDrag && local.X >= 0 && local.X <= c.Bounds().Width()
	switch me.MouseType {
	case event.MousePress:
		if me.Button != event.ButtonLeft || !c.Bounds().Contains(me.Position) {
			return false
		}
		if onEdge {
			c.dragging = true
			c.startY = me.Position.Y
			c.startH = c.pixelH()
			if cap, ok := ctx.(widget.PointerCapturer); ok {
				cap.CapturePointer(c)
			}
			resizeNS(ctx)
			return true
		}
		c.SetFocused(true)
		if ctx != nil {
			ctx.RequestFocus(c)
		}
		return true
	case event.MouseDrag, event.MouseMove:
		if c.dragging {
			if me.MouseType == event.MouseMove && !me.Buttons.IsLeftPressed() {
				c.dragging = false
				return false
			}
			c.height = clampH(c.startH + (c.startY - me.Position.Y))
			c.MarkNeedsLayout()
			c.SetNeedsRedraw(true)
			resizeNS(ctx)
			return true
		}
		if onEdge {
			resizeNS(ctx)
			return true
		}
		return false
	case event.MouseRelease:
		if c.dragging {
			c.dragging = false
			if cap, ok := ctx.(widget.PointerCapturer); ok {
				cap.ReleasePointer(c)
			}
			return true
		}
	}
	return false
}

func (c *Console) scrollBack(dy float32, hist int) {
	lines := int(dy / 16)
	if lines == 0 {
		if dy > 0 {
			lines = 1
		} else if dy < 0 {
			lines = -1
		}
	}
	c.back -= lines
	if c.back < 0 {
		c.back = 0
	}
	if c.back > hist {
		c.back = hist
	}
	c.SetNeedsRedraw(true)
}

func clampH(h float32) float32 {
	if h < consoleMin {
		return consoleMin
	}
	if h > consoleMax {
		return consoleMax
	}
	return h
}

func resizeNS(ctx widget.Context) {
	if ctx != nil {
		ctx.SetCursor(widget.CursorResizeNS)
	}
}

func keyBytes(v *event.KeyEvent) []byte {
	if v.IsCtrl() && !v.IsSuper() {
		if b, ok := ctrlByte(v); ok {
			return []byte{b}
		}
	}
	switch v.Key {
	case event.KeyEnter, event.KeyNumpadEnter:
		return []byte{'\r'}
	case event.KeyBackspace:
		return []byte{0x7f}
	case event.KeyDelete:
		return []byte("\x1b[3~")
	case event.KeyTab:
		return []byte{'\t'}
	case event.KeyEscape:
		return []byte{0x1b}
	case event.KeyUp:
		return []byte("\x1b[A")
	case event.KeyDown:
		return []byte("\x1b[B")
	case event.KeyRight:
		return []byte("\x1b[C")
	case event.KeyLeft:
		return []byte("\x1b[D")
	case event.KeyHome:
		return []byte("\x1b[H")
	case event.KeyEnd:
		return []byte("\x1b[F")
	case event.KeyPageUp:
		return []byte("\x1b[5~")
	case event.KeyPageDown:
		return []byte("\x1b[6~")
	}
	// Printable characters arrive as their own event, with the rune set.
	// The matching key press has an empty rune and must not be sent twice.
	if v.Rune != 0 && !v.IsSuper() && !v.IsCtrl() {
		return []byte(string(v.Rune))
	}
	return nil
}

func ctrlByte(v *event.KeyEvent) (byte, bool) {
	r := v.Rune
	if r >= 'A' && r <= 'Z' {
		r += 'a' - 'A'
	}
	if r >= 'a' && r <= 'z' {
		return byte(r - 'a' + 1), true
	}
	if v.Key >= event.KeyA && v.Key <= event.KeyZ {
		return byte(v.Key-event.KeyA) + 1, true
	}
	return 0, false
}

// Children returns nil.
func (c *Console) Children() []widget.Widget { return nil }
