package ui

import (
	"sort"
	"strconv"
	"unicode/utf8"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/gitlog"
	"pairti-lense/internal/theme"
)

// DiffView lays out both revisions on the same vertical scroll surface.
type DiffView struct {
	widget.WidgetBase
	theme                       theme.Theme
	model                       gitlog.Comparison
	subject, date               string
	font                        string
	cell, width, height, gutter float32
	checkFont, sideBySide       bool
	rows                        []diffVisualRow
}

type diffRole int

const (
	diffCode diffRole = iota
	diffTitle
	diffDate
	diffFile
	diffColumns
	diffHunk
	diffNotice
)

type diffCell struct {
	number  int
	mark    string
	runs    []gitlog.DiffRun
	endNote bool
}
type diffVisualRow struct {
	role        diffRole
	label       string
	left, right diffCell
	y, h        float32
}

func NewDiffView() *DiffView {
	d := &DiffView{}
	d.SetVisible(true)
	d.SetEnabled(true)
	d.SetTheme(theme.Pairti())
	return d
}

func (d *DiffView) SetTheme(t theme.Theme) {
	d.theme = t
	ratio, _ := loadSourceFont()
	d.cell = max(1, t.Code*ratio)
	d.checkFont = true
	d.invalidate()
}

func (d *DiffView) invalidate() {
	d.width = 0
	d.MarkNeedsLayout()
	d.SetNeedsRedraw(true)
}

func (d *DiffView) SetComparison(subject, date string, result gitlog.Comparison) {
	d.subject, d.date, d.model = subject, date, result
	d.invalidate()
}

func (d *DiffView) SetMessage(message string) {
	d.SetComparison("", "", gitlog.Comparison{Notice: message})
}

func (d *DiffView) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	w := c.MaxWidth
	if w <= 0 || w >= geometry.Infinity {
		w = 900
	}
	d.reflow(w)
	return c.Constrain(geometry.Sz(w, d.height))
}

func (d *DiffView) add(row diffVisualRow) {
	row.y = d.height
	d.height += row.h
	d.rows = append(d.rows, row)
}

func (d *DiffView) label(role diffRole, label string, height float32) {
	if label != "" {
		d.add(diffVisualRow{role: role, label: label, h: height})
	}
}

func (d *DiffView) lineH() float32 { return max(16, d.theme.Code*1.55) }

func (d *DiffView) reflow(w float32) {
	if w == d.width && w > 0 {
		return
	}
	d.width = w
	d.height = 0
	d.rows = nil
	// Each revision should have room for a short source line plus its gutter.
	d.sideBySide = w >= max(720, 88*d.cell)
	h := d.lineH()
	d.label(diffTitle, d.subject, max(h, d.theme.H2*1.5))
	d.label(diffDate, d.date, max(h, d.theme.Meta*1.65))
	d.label(diffNotice, d.model.Notice, max(h, d.theme.Body*1.6))
	// Use the largest line number in this comparison to keep columns stable.
	digits := 2
	for _, f := range d.model.Files {
		for _, hk := range f.Hunks {
			for _, r := range hk.Rows {
				for _, line := range []*gitlog.DiffLine{r.Old, r.New} {
					if line != nil {
						digits = max(digits, len(strconv.Itoa(line.Number)))
					}
				}
			}
		}
	}
	// One gutter contains a right-aligned line number and a sign.
	d.gutter = float32(digits+3)*d.cell + 6
	contentWidth := w - d.gutter*2
	if d.sideBySide {
		contentWidth = w/2 - d.gutter
	}
	columns := max(1, int((contentWidth-8)/d.cell))
	for _, file := range d.model.Files {
		name := file.NewPath
		switch {
		case name == "":
			name = file.OldPath
		case file.OldPath != "" && file.OldPath != name:
			name = file.OldPath + " → " + name
		}
		if file.Status != "" {
			name = file.Status + "  ·  " + name
		}
		d.height += 8
		d.label(diffFile, name, max(h, d.theme.Body*1.85))
		if len(file.Hunks) == 0 {
			continue
		}
		d.add(diffVisualRow{role: diffColumns, h: h})
		for _, hk := range file.Hunks {
			d.label(diffHunk, hk.Header, h)
			d.addHunk(hk.Rows, columns)
		}
	}
	if len(d.model.Files) == 0 && d.model.Notice == "" {
		d.label(diffNotice, "No file changes", h)
	}
	d.height += 8
}

func (d *DiffView) addHunk(rows []gitlog.DiffRow, cols int) {
	if d.sideBySide {
		for _, row := range rows {
			d.addLine(row, cols)
		}
		return
	}
	// Unified hunks group the old and new versions of each replacement in
	// source order, even when the wide view pairs several lines side by side.
	for i := 0; i < len(rows); {
		if rows[i].Context {
			d.addLine(rows[i], cols)
			i++
			continue
		}
		end := i
		for end < len(rows) && !rows[end].Context {
			end++
		}
		for _, row := range rows[i:end] {
			if row.Old != nil {
				d.addLine(gitlog.DiffRow{Old: row.Old}, cols)
			}
		}
		for _, row := range rows[i:end] {
			if row.New != nil {
				d.addLine(gitlog.DiffRow{New: row.New}, cols)
			}
		}
		i = end
	}
}

func (d *DiffView) addLine(row gitlog.DiffRow, cols int) {
	oldRows := wrapDiffLine(row.Old, cols)
	newRows := wrapDiffLine(row.New, cols)
	for i := 0; i < max(len(oldRows), len(newRows)); i++ {
		visual := diffVisualRow{role: diffCode, h: d.lineH()}
		visual.left = diffWrappedCell(row.Old, oldRows, i)
		visual.right = diffWrappedCell(row.New, newRows, i)
		if !row.Context {
			if row.Old != nil {
				visual.left.mark = "−"
			}
			if row.New != nil {
				visual.right.mark = "+"
			}
		}
		d.add(visual)
	}
	// Git can distinguish revisions solely by the presence of a terminating newline.
	oldEOF := row.Old != nil && row.Old.NoNewline
	newEOF := row.New != nil && row.New.NoNewline
	if oldEOF || newEOF {
		note := diffVisualRow{role: diffCode, h: d.lineH()}
		note.left.endNote, note.right.endNote = oldEOF, newEOF
		if !row.Context {
			if oldEOF {
				note.left.mark = "−"
			}
			if newEOF {
				note.right.mark = "+"
			}
		}
		d.add(note)
	}

}

func diffWrappedCell(line *gitlog.DiffLine, wrapped [][]gitlog.DiffRun, index int) diffCell {
	var c diffCell
	if index < len(wrapped) {
		c.runs = wrapped[index]
		if index == 0 {
			c.number = line.Number
		}
	}
	return c
}

func (d *DiffView) Draw(_ widget.Context, c widget.Canvas) {
	b := d.Bounds()
	if !d.IsVisible() || b.IsEmpty() {
		return
	}
	if d.checkFont {
		d.checkFont = false
		family, cell := measuredSourceFont(c, d.theme.RawFont, d.theme.Code)
		if cell > 0 && (cell != d.cell || family != d.font) {
			d.cell, d.font = cell, family
			d.invalidate()
		}
	}
	d.reflow(b.Width())
	c.PushClip(b)
	defer c.PopClip()
	// ClipBounds is in canvas coordinates, including the scroll transform.
	clip := c.ClipBounds()
	offset := c.TransformOffset()
	start, end := float32(0), b.Height()
	if !clip.IsEmpty() {
		start = clip.Min.Y - offset.Y - b.Min.Y
		end = clip.Max.Y - offset.Y - b.Min.Y
	}
	first := sort.Search(len(d.rows), func(i int) bool { return d.rows[i].y+d.rows[i].h > start })
	for _, row := range d.rows[first:] {
		if row.y >= end {
			break
		}
		rect := geometry.NewRect(b.Min.X, b.Min.Y+row.y, b.Width(), row.h)
		if row.role == diffCode {
			d.drawCode(c, rect, row)
			continue
		}
		d.drawHeader(c, rect, row)
	}
}

func (d *DiffView) drawHeader(c widget.Canvas, b geometry.Rect, row diffVisualRow) {
	t := d.theme
	size, col, bold := t.Meta, rgb(t.Muted), false
	switch row.role {
	case diffTitle:
		size, col, bold = t.H2, rgb(t.Ink), true
	case diffFile:
		size, col, bold = t.Body, rgb(t.Ink), true
		c.DrawRect(b, rgb(t.Surface))
	case diffHunk:
		c.DrawRect(b, rgb(t.CodeFill))
	case diffColumns:
		c.DrawRect(b, rgb(t.Surface))
		if d.sideBySide {
			drawFace(c, "Before", geometry.NewRect(b.Min.X+8, b.Min.Y, b.Width()/2-10, b.Height()), size, col, false, t.UIFont)
			drawFace(c, "After", geometry.NewRect(b.Center().X+8, b.Min.Y, b.Width()/2-10, b.Height()), size, col, false, t.UIFont)
			return
		}
		row.label = "Before / After"
	}
	inner := geometry.NewRect(b.Min.X+8, b.Min.Y, max(1, b.Width()-16), b.Height())
	label := fitMeasured(c, row.label, inner.Width(), size, t.UIFont, col)
	drawFace(c, label, inner, size, col, bold, t.UIFont)
}

func (d *DiffView) drawCode(c widget.Canvas, b geometry.Rect, row diffVisualRow) {
	if d.sideBySide {
		w := b.Width() / 2
		d.drawCell(c, geometry.NewRect(b.Min.X, b.Min.Y, w, row.h), row.left, 0, false)
		d.drawCell(c, geometry.NewRect(b.Min.X+w, b.Min.Y, w, row.h), row.right, 0, false)
		c.DrawLine(geometry.Pt(b.Min.X+w, b.Min.Y), geometry.Pt(b.Min.X+w, b.Max.Y), rgb(d.theme.Rule), 1)
		return
	}
	cell := row.right
	if row.left.mark == "−" {
		cell = row.left
	}
	d.drawCell(c, b, cell, row.left.number, true)
}

func (d *DiffView) drawCell(c widget.Canvas, b geometry.Rect, cell diffCell, oldNum int, unified bool) {
	t := d.theme
	var tint uint32
	switch cell.mark {
	case "−":
		tint = 0xc74952
	case "+":
		tint = 0x489d65
	}
	if tint != 0 {
		c.DrawRect(b, mixHex(t.Canvas, tint, 0.18))
	}
	x := b.Min.X
	number := func(n int) {
		if n > 0 {
			s := strconv.Itoa(n)
			nw := float32(len(s)) * d.cell
			box := geometry.NewRect(x+d.gutter-2*d.cell-nw-4, b.Min.Y, nw+2, b.Height())
			drawFace(c, s, box, t.Code, rgb(t.Muted), false, sourceFamily(d.font, false, false))
		}
		x += d.gutter
	}
	if unified {
		number(oldNum)
	}
	newNum := cell.number
	if unified && cell.mark == "−" {
		newNum = 0
	}
	number(newNum)
	if cell.mark != "" && cell.number > 0 {
		drawFace(c, cell.mark, geometry.NewRect(x-2*d.cell, b.Min.Y, 2*d.cell, b.Height()), t.Code, rgb(t.Ink), false, sourceFamily(d.font, false, false))
	}
	c.PushClip(b)
	defer c.PopClip()
	if cell.endNote {
		drawFace(c, "No newline at end of file", geometry.NewRect(x, b.Min.Y, max(1, b.Max.X-x), b.Height()), t.Meta, rgb(t.Muted), false, t.UIFont)
		return
	}
	for _, run := range cell.runs {
		w := float32(utf8.RuneCountInString(run.Text)) * d.cell
		box := geometry.NewRect(x, b.Min.Y, w+1, b.Height())
		if run.Changed && tint != 0 {
			c.DrawRect(geometry.NewRect(x, b.Min.Y+1, w, b.Height()-2), mixHex(t.Canvas, tint, 0.46))
		}
		style := widget.TextStyle{FontFamily: sourceFamily(d.font, run.Bold, run.Italic), FontSize: t.Code, Color: toneColor(t, run.Tone), Bold: run.Bold, Italic: run.Italic, Align: widget.TextAlignLeft}
		if sd, ok := c.(widget.StyledTextDrawer); ok {
			sd.DrawStyledText(run.Text, box, style)
		} else {
			c.DrawText(run.Text, box, t.Code, style.Color, run.Bold, widget.TextAlignLeft)
		}
		x += w
	}
}

// Wheel input belongs to the enclosing scroll view. The comparison has no
// editable buffer or focus target.
func (d *DiffView) Event(widget.Context, event.Event) bool { return false }
func (d *DiffView) Children() []widget.Widget              { return nil }
