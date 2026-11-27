package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// Cover draws the window and, when the export sheet is open, that sheet on top.
// The sheet takes the pointer and the keys until it closes.
type Cover struct {
	widget.WidgetBase
	body    widget.Widget
	sheet   *ExportSheet
	menu    *TreeMenu
	tabs    *TabMenu
	palette *Palette
	help    *MCPHelp
	rename  *Rename
	chord   func(*event.KeyEvent) bool
}

// NewCover returns the window with an export sheet above it.
func NewCover(body widget.Widget, sheet *ExportSheet) *Cover {
	c := &Cover{body: body, sheet: sheet}
	c.SetVisible(true)
	c.SetEnabled(true)
	type parentSetter interface{ SetParent(widget.Widget) }
	if p, ok := body.(parentSetter); ok {
		p.SetParent(c)
	}
	if sheet != nil {
		sheet.SetParent(c)
	}
	return c
}

// SetMenu draws the file menu above the window while it is open.
func (c *Cover) SetMenu(menu *TreeMenu) {
	c.menu = menu
	if menu != nil {
		menu.SetParent(c)
	}
}

// SetRename draws the report rename card above the window while it is open.
func (c *Cover) SetRename(r *Rename) {
	c.rename = r
	if r != nil {
		r.SetParent(c)
	}
}

// SetHelp draws the MCP card above the window while it is open.
func (c *Cover) SetHelp(help *MCPHelp) {
	c.help = help
	if help != nil {
		help.SetParent(c)
	}
}

// SetPalette draws the note search above the window while it is open.
func (c *Cover) SetPalette(p *Palette) {
	c.palette = p
	if p != nil {
		p.SetParent(c)
	}
}

// OnChord runs before the focused document, for shortcuts such as Cmd-K.
func (c *Cover) OnChord(fn func(*event.KeyEvent) bool) { c.chord = fn }

// SetTabMenu draws the tab menu above the window while it is open.
func (c *Cover) SetTabMenu(menu *TabMenu) {
	c.tabs = menu
	if menu != nil {
		menu.SetParent(c)
	}
}

// Layout gives both layers the whole window.
func (c *Cover) Layout(ctx widget.Context, cons geometry.Constraints) geometry.Size {
	size := cons.BiggestFinite(800, 600)
	if c.Bounds().IsEmpty() {
		c.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	}
	rect := geometry.NewRect(0, 0, size.Width, size.Height)
	tight := geometry.Constraints{
		MinWidth: size.Width, MaxWidth: size.Width,
		MinHeight: size.Height, MaxHeight: size.Height,
	}
	for _, child := range c.layers() {
		widget.LayoutChild(child, ctx, tight)
		if setter, ok := child.(interface{ SetBounds(geometry.Rect) }); ok {
			setter.SetBounds(rect)
		}
	}
	return size
}

// Draw paints the window, then the sheet.
func (c *Cover) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := c.Bounds()
	if b.IsEmpty() {
		return
	}
	canvas.PushTransform(b.Min)
	if c.body != nil {
		widget.StampScreenOrigin(c.body, canvas)
		c.body.Draw(ctx, canvas)
	}
	if c.sheet != nil && c.sheet.Shown() {
		widget.StampScreenOrigin(c.sheet, canvas)
		c.sheet.Draw(ctx, canvas)
	}
	if c.menu != nil && c.menu.Shown() {
		widget.StampScreenOrigin(c.menu, canvas)
		c.menu.Draw(ctx, canvas)
	}
	if c.tabs != nil && c.tabs.Shown() {
		widget.StampScreenOrigin(c.tabs, canvas)
		c.tabs.Draw(ctx, canvas)
	}
	if c.palette != nil && c.palette.Shown() {
		widget.StampScreenOrigin(c.palette, canvas)
		c.palette.Draw(ctx, canvas)
	}
	if c.help != nil && c.help.Shown() {
		widget.StampScreenOrigin(c.help, canvas)
		c.help.Draw(ctx, canvas)
	}
	if c.rename != nil && c.rename.Shown() {
		widget.StampScreenOrigin(c.rename, canvas)
		c.rename.Draw(ctx, canvas)
	}
	canvas.PopTransform()
}

// Event gives the open sheet the first chance at input.
func (c *Cover) Event(ctx widget.Context, e event.Event) bool {
	if ke, ok := e.(*event.KeyEvent); ok && ke.IsPress() && c.chord != nil && c.chord(ke) {
		return true
	}
	if c.rename != nil && c.rename.Shown() && c.send(ctx, c.rename, e) {
		return true
	}
	if c.help != nil && c.help.Shown() && c.send(ctx, c.help, e) {
		return true
	}
	if c.palette != nil && c.palette.Shown() && c.send(ctx, c.palette, e) {
		return true
	}
	if c.tabs != nil && c.tabs.Shown() && c.send(ctx, c.tabs, e) {
		return true
	}
	if c.menu != nil && c.menu.Shown() && c.send(ctx, c.menu, e) {
		return true
	}
	if c.sheet != nil && c.sheet.Shown() && c.send(ctx, c.sheet, e) {
		return true
	}
	return c.send(ctx, c.body, e)
}

// Children is the window and, while it is open, the sheet.
func (c *Cover) Children() []widget.Widget {
	if c.rename != nil && c.rename.Shown() {
		return []widget.Widget{c.body, c.rename}
	}
	if c.help != nil && c.help.Shown() {
		return []widget.Widget{c.body, c.help}
	}
	if c.palette != nil && c.palette.Shown() {
		return []widget.Widget{c.body, c.palette}
	}
	if c.tabs != nil && c.tabs.Shown() {
		return []widget.Widget{c.body, c.tabs}
	}
	if c.menu != nil && c.menu.Shown() {
		return []widget.Widget{c.body, c.menu}
	}
	if c.sheet != nil && c.sheet.Shown() {
		return []widget.Widget{c.body, c.sheet}
	}
	if c.body == nil {
		return nil
	}
	return []widget.Widget{c.body}
}

func (c *Cover) layers() []widget.Widget {
	out := []widget.Widget{c.body}
	if c.sheet != nil {
		out = append(out, c.sheet)
	}
	if c.menu != nil {
		out = append(out, c.menu)
	}
	if c.tabs != nil {
		out = append(out, c.tabs)
	}
	if c.palette != nil {
		out = append(out, c.palette)
	}
	if c.help != nil {
		out = append(out, c.help)
	}
	if c.rename != nil {
		out = append(out, c.rename)
	}
	return out
}

func (c *Cover) send(ctx widget.Context, child widget.Widget, e event.Event) bool {
	if child == nil {
		return false
	}
	if _, ok := e.(*event.KeyEvent); ok {
		return child.Event(ctx, e)
	}
	if me, ok := e.(*event.MouseEvent); ok {
		cp := *me
		cp.Position = me.Position.Sub(c.Bounds().Min)
		return child.Event(ctx, &cp)
	}
	if we, ok := e.(*event.WheelEvent); ok {
		cp := *we
		cp.Position = we.Position.Sub(c.Bounds().Min)
		return child.Event(ctx, &cp)
	}
	return child.Event(ctx, e)
}
