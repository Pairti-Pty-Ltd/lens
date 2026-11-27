package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// CommandLine is the : line at the bottom of the text.
// It is one row, and it takes no space until a command is open.
type CommandLine struct {
	widget.WidgetBase
	theme  theme.Theme
	on     bool
	text   string
	prompt rune
}

// NewCommandLine returns a hidden command line.
func NewCommandLine() *CommandLine {
	c := &CommandLine{theme: theme.Pairti()}
	c.SetVisible(true)
	c.SetEnabled(true)
	return c
}

// SetTheme replaces the semantic theme.
func (c *CommandLine) SetTheme(t theme.Theme) { c.theme = t; c.SetNeedsRedraw(true) }

// Set shows or hides the line. text is the command without the colon.
func (c *CommandLine) SetPrompt(prompt rune) { c.prompt = prompt; c.SetNeedsRedraw(true) }

func (c *CommandLine) Set(on bool, text string) {
	if c.on == on && c.text == text {
		return
	}
	c.on = on
	c.text = text
	c.MarkNeedsLayout()
	c.SetNeedsRedraw(true)
}

// Layout is one row, or nothing while a command is not open.
func (c *CommandLine) Layout(_ widget.Context, cons geometry.Constraints) geometry.Size {
	w := cons.MaxWidth
	if w <= 0 || w >= geometry.Infinity {
		w = 200
	}
	h := float32(0)
	if c.on {
		h = 26
	}
	return cons.Constrain(geometry.Sz(w, h))
}

// Draw paints the colon and the command.
func (c *CommandLine) Draw(_ widget.Context, canvas widget.Canvas) {
	if !c.on || !c.IsVisible() {
		return
	}
	b := c.Bounds()
	if b.Height() < 1 {
		return
	}
	canvas.DrawRect(b, rgb(c.theme.Surface))
	prompt := c.prompt
	if prompt == 0 {
		prompt = ':'
	}
	text := string(prompt) + c.text
	drawFace(canvas, text, geometry.NewRect(b.Min.X+8, b.Min.Y+4, b.Width()-16, 18), c.theme.Meta, rgb(c.theme.Ink), false, c.theme.CodeFont)
	w := textWidth(canvas, text, c.theme.Meta, rgb(c.theme.Ink), false, c.theme.CodeFont)
	canvas.DrawRect(geometry.NewRect(b.Min.X+8+w, b.Min.Y+6, 1.5, 14), rgb(c.theme.Accent))
}

// Event is unused. Keys stay with the text.
func (c *CommandLine) Event(widget.Context, event.Event) bool { return false }

// Children returns nil.
func (c *CommandLine) Children() []widget.Widget { return nil }
