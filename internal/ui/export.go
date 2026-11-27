package ui

import (
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const (
	exportW       = float32(300)
	exportPad     = float32(16)
	exportTitle   = float32(22)
	exportNameH   = float32(20)
	exportLabel   = float32(18)
	exportRow     = float32(32)
	exportItem    = float32(28)
	exportGo      = float32(36)
	exportNudge   = float32(28)
	exportFileWin = 6

	DestTemp    = "temp"
	DestCurrent = "current"
	DestPick    = "pick"

	exportCard = uint32(0x2C2C2E)
	exportWell = uint32(0x3A3A3C)
)

// ExportFormat is one row in the format menu.
type ExportFormat struct {
	ID    string
	Label string
}

// ExportChoice is what the Export button confirmed.
type ExportChoice struct {
	Brand   string
	Dest    string
	Open    bool
	Project bool
	Files   []string
}

// ExportSheet is the export popover. The format list stays closed until the
// format row is pressed. A click outside, or Escape, dismisses it.
// Project settings list the export order. The template menu stays closed.
type ExportSheet struct {
	downloadOnly bool

	widget.WidgetBase
	theme       theme.Theme
	formats     []ExportFormat
	brand       string
	dest        string
	openFile    bool
	menu        bool
	shown       bool
	name        string
	anchor      geometry.Rect
	commit      func(ExportChoice)
	scope       string
	fixedScope  bool
	projectName string
	files       []string
	template    string
	settings    bool
	fileAt      int
	tmplMenu    bool
	onSave      func(files []string, template string)
	docBrand    string
	snapFiles   []string
	snapTmpl    string
	snapped     bool
}

// NewExportSheet returns a hidden sheet. commit runs when Export is pressed.
func NewExportSheet(commit func(ExportChoice)) *ExportSheet {
	e := &ExportSheet{
		theme:    theme.Pairti(),
		formats:  []ExportFormat{{ID: "default.pdf", Label: "Default (pdf)"}},
		brand:    "default.pdf",
		dest:     DestTemp,
		openFile: true,
		commit:   commit,
		scope:    "doc",
	}
	e.SetVisible(true)
	e.SetEnabled(true)
	return e
}

// SetDownloadOnly hides desktop destination controls when the browser delivers the file.
func (e *ExportSheet) SetDownloadOnly(on bool) { e.downloadOnly = on }

// SetTheme replaces the semantic theme.
func (e *ExportSheet) SetTheme(t theme.Theme) {
	e.theme = t
	e.SetNeedsRedraw(true)
}

// SetFormats replaces the format menu. The current brand is kept when it remains.
func (e *ExportSheet) SetFormats(list []ExportFormat) {
	if len(list) == 0 {
		return
	}
	e.formats = append([]ExportFormat(nil), list...)
	for _, f := range e.formats {
		if f.ID == e.brand {
			return
		}
	}
	for _, f := range e.formats {
		if f.ID == "default.pdf" {
			e.brand = f.ID
			return
		}
	}
	e.brand = e.formats[0].ID
}

// SetProject is the open note's project: its folder name, export order, and brand.
func (e *ExportSheet) SetProject(name string, files []string, template string) {
	e.projectName = name
	e.files = append([]string(nil), files...)
	e.template = brandOf(template)
	e.fileAt = 0
	e.SetNeedsRedraw(true)
}

// FixScope locks the sheet to the open note or the open report.
// The Document | Project switch is not drawn.
func (e *ExportSheet) FixScope(report bool) {
	e.fixedScope = true
	if report {
		e.applyTemplate()
		e.scope = "project"
		return
	}
	e.scope = "doc"
}

// OnProjectSave runs when the project settings are closed with Done.
func (e *ExportSheet) OnProjectSave(fn func(files []string, template string)) { e.onSave = fn }

// Show opens the sheet for one note. anchor is the toolbar slot in window coordinates.
func (e *ExportSheet) Show(name string, anchor geometry.Rect) {
	e.shown = true
	e.menu = false
	e.settings = false
	e.tmplMenu = false
	e.scope = "doc"
	e.fixedScope = false
	e.snapped = false
	e.docBrand = ""
	e.name = name
	e.anchor = anchor
	e.SetNeedsRedraw(true)
}

// Hide closes the sheet without exporting.
func (e *ExportSheet) Hide() {
	if e.settings {
		e.restoreSnap()
	}
	if !e.shown && !e.menu {
		return
	}
	e.shown = false
	e.menu = false
	e.settings = false
	e.tmplMenu = false
	e.SetNeedsRedraw(true)
}

// Shown reports whether the popover is open.
func (e *ExportSheet) Shown() bool { return e.shown }

// Layout fills the window. The card is placed inside Draw.
func (e *ExportSheet) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.BiggestFinite(800, 600)
}

// Draw paints the card. A closed format menu does not list the other brands.
// Project settings show short file names. The template menu stays closed.
func (e *ExportSheet) Draw(_ widget.Context, canvas widget.Canvas) {
	if !e.shown || canvas == nil {
		return
	}
	card, hits := e.place()
	e.drawCaret(canvas, card)
	fill := rgb(exportCard)
	canvas.PushClipRoundRect(card, 10)
	canvas.DrawRect(card, fill)
	canvas.PopClip()
	canvas.DrawRoundRect(card, fill, 10)
	canvas.StrokeRoundRect(card, rgb(e.theme.Rule), 10, 1)
	inner := geometry.NewRect(card.Min.X+exportPad, card.Min.Y+exportPad, card.Width()-exportPad*2, exportTitle)
	title := "Export"
	label := e.name
	if e.settings {
		title = "Project"
		label = e.projectName
	} else if e.scope == "project" && e.projectName != "" {
		label = e.projectName
	}
	drawCenter(canvas, title, inner, 16, rgb(e.theme.Ink), true, e.theme.UIFont)
	name := e.fitted(canvas, label, inner.Width(), 13)
	nameRow := geometry.NewRect(inner.Min.X, inner.Max.Y+2, inner.Width(), exportNameH)
	if name != "" {
		drawCenter(canvas, name, nameRow, 13, rgb(e.theme.Ink), true, e.theme.UIFont)
	}
	if !e.settings {
		e.drawScope(canvas, hits)
	}
	for _, h := range hits {
		if strings.HasPrefix(h.id, "scope-") {
			continue
		}
		e.drawHit(canvas, h)
	}
}

// Event handles the card. Escape and a click on the scrim dismiss it.
// Escape from project settings returns to the export sheet.
func (e *ExportSheet) Event(ctx widget.Context, ev event.Event) bool {
	if !e.shown {
		return false
	}
	if we, ok := ev.(*event.WheelEvent); ok {
		if e.settings {
			e.scrollFiles(we.DeltaY())
			invalidate(ctx)
		}
		return true
	}
	if ke, ok := ev.(*event.KeyEvent); ok {
		if ke.IsPress() && ke.Key == event.KeyEscape {
			if e.settings {
				e.restoreSnap()
				e.settings = false
				e.tmplMenu = false
				e.SetNeedsRedraw(true)
				invalidate(ctx)
				return true
			}
			e.Hide()
			invalidate(ctx)
		}
		return true
	}
	me, ok := ev.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return true
	}
	card, hits := e.place()
	if !card.Contains(me.Position) {
		e.Hide()
		invalidate(ctx)
		return true
	}
	for _, h := range hits {
		if h.box.Contains(me.Position) {
			e.press(h.id)
			invalidate(ctx)
			return true
		}
	}
	return true
}

// Children returns nil. The sheet is drawn by its parent.
func (e *ExportSheet) Children() []widget.Widget { return nil }

// Spot is the centre of a control, for tests.
func (e *ExportSheet) Spot(id string) (geometry.Point, bool) {
	_, hits := e.place()
	for _, h := range hits {
		if h.id == id {
			return geometry.Pt(h.box.Min.X+h.box.Width()/2, h.box.Min.Y+h.box.Height()/2), true
		}
	}
	return geometry.Point{}, false
}

type exportHit struct {
	id  string
	box geometry.Rect
}

func (e *ExportSheet) place() (geometry.Rect, []exportHit) {
	h := e.cardH()
	win := e.Bounds()
	anchor := e.anchorLocal()
	x := win.Min.X + (win.Width()-exportW)/2
	y := win.Min.Y + 48
	if anchor.Width() > 0 {
		x = anchor.Min.X + anchor.Width()/2 - exportW/2
		y = anchor.Max.Y + 10
	}
	if x < win.Min.X+8 {
		x = win.Min.X + 8
	}
	if y < win.Min.Y+8 {
		y = win.Min.Y + 8
	}
	if x+exportW > win.Max.X-8 {
		x = win.Max.X - 8 - exportW
	}
	if y+h > win.Max.Y-8 {
		y = win.Max.Y - 8 - h
	}
	return e.layout(geometry.Pt(x, y))
}

func (e *ExportSheet) cardH() float32 {
	card, _ := e.layout(geometry.Point{})
	return card.Height()
}

func (e *ExportSheet) layout(at geometry.Point) (geometry.Rect, []exportHit) {
	if e.settings {
		return e.settingsRows(at)
	}
	return e.rows(at)
}

func (e *ExportSheet) rows(at geometry.Point) (geometry.Rect, []exportHit) {
	x := at.X + exportPad
	inner := exportW - exportPad*2
	y := at.Y + exportPad + exportTitle + 2 + exportNameH + 12
	var hits []exportHit
	if !e.fixedScope {
		half := inner / 2
		hits = append(hits,
			exportHit{"scope-doc", geometry.NewRect(x, y, half, exportRow)},
			exportHit{"scope-project", geometry.NewRect(x+half, y, inner-half, exportRow)},
		)
		y += exportRow
		if e.scope == "project" {
			y += 8
			hits = append(hits, exportHit{"settings", geometry.NewRect(x, y, inner, exportRow)})
			y += exportRow
		}
	}
	y += 10
	add := func(id string, h float32) {
		hits = append(hits, exportHit{id, geometry.NewRect(x, y, inner, h)})
		y += h
	}
	y += exportLabel + 6
	add("format", exportRow)
	if e.menu {
		y += 4
		for _, f := range e.formats {
			add("brand:"+f.ID, exportItem)
		}
	}
	y += 10
	if e.downloadOnly {
		add("saved-files", exportItem)
	} else {
		add("temp", exportItem)
		add("current", exportItem)
		add("pick", exportItem)
		y += 8
		add("open", exportItem)
	}
	y += 12
	add("go", exportGo)
	y += exportPad
	return geometry.NewRect(at.X, at.Y, exportW, y-at.Y), hits
}

func (e *ExportSheet) settingsRows(at geometry.Point) (geometry.Rect, []exportHit) {
	x := at.X + exportPad
	inner := exportW - exportPad*2
	y := at.Y + exportPad + exportTitle + 2 + exportNameH + 12
	var hits []exportHit
	y += exportLabel + 6
	hits = append(hits, exportHit{"template", geometry.NewRect(x, y, inner, exportRow)})
	y += exportRow
	if e.tmplMenu {
		y += 4
		for _, f := range e.templates() {
			hits = append(hits, exportHit{"template:" + f.ID, geometry.NewRect(x, y, inner, exportItem)})
			y += exportItem
		}
	}
	y += 10
	e.clampFiles()
	end := e.fileAt + exportFileWin
	if end > len(e.files) {
		end = len(e.files)
	}
	for i := e.fileAt; i < end; i++ {
		hits = append(hits,
			exportHit{"up:" + strconv.Itoa(i), geometry.NewRect(x, y, exportNudge, exportItem)},
			exportHit{"down:" + strconv.Itoa(i), geometry.NewRect(x+inner-exportNudge, y, exportNudge, exportItem)},
			exportHit{"file:" + strconv.Itoa(i), geometry.NewRect(x+exportNudge, y, inner-exportNudge*2, exportItem)},
		)
		y += exportItem
	}
	y += 12
	hits = append(hits, exportHit{"done", geometry.NewRect(x, y, inner, exportGo)})
	y += exportGo + exportPad
	return geometry.NewRect(at.X, at.Y, exportW, y-at.Y), hits
}

func (e *ExportSheet) press(id string) {
	switch id {
	case "format":
		e.menu = !e.menu
	case "template":
		e.tmplMenu = !e.tmplMenu
	case "temp":
		e.dest = DestTemp
		e.menu = false
	case "current":
		e.dest = DestCurrent
		e.menu = false
	case "pick":
		e.dest = DestPick
		e.menu = false
	case "open":
		e.openFile = !e.openFile
	case "scope-doc":
		if e.scope != "doc" {
			e.scope = "doc"
			if e.docBrand != "" {
				e.brand = e.docBrand
			}
		}
		e.menu = false
	case "scope-project":
		if e.scope != "project" {
			e.docBrand = e.brand
			e.scope = "project"
			e.applyTemplate()
		}
		e.menu = false
	case "settings":
		e.snapFiles = append([]string(nil), e.files...)
		e.snapTmpl = e.template
		e.snapped = true
		e.settings = true
		e.menu = false
		e.tmplMenu = false
		e.fileAt = 0
	case "done":
		changed := e.template != e.snapTmpl
		e.snapped = false
		e.settings = false
		e.tmplMenu = false
		if changed {
			e.applyTemplate()
		}
		if e.onSave != nil {
			e.onSave(append([]string(nil), e.files...), e.template)
		}
	case "go":
		choice := ExportChoice{Brand: e.brand, Dest: e.dest, Open: e.openFile, Project: e.scope == "project"}
		if choice.Project {
			choice.Files = append([]string(nil), e.files...)
		}
		e.Hide()
		if e.commit != nil {
			e.commit(choice)
		}
	default:
		switch {
		case strings.HasPrefix(id, "brand:"):
			e.brand = id[len("brand:"):]
			e.menu = false
		case strings.HasPrefix(id, "template:"):
			e.template = brandOf(id[len("template:"):])
			e.tmplMenu = false
		case strings.HasPrefix(id, "up:"):
			if i, ok := hitIndex(id, "up:"); ok {
				e.move(i, -1)
			}
		case strings.HasPrefix(id, "down:"):
			if i, ok := hitIndex(id, "down:"); ok {
				e.move(i, 1)
			}
		}
	}
	e.SetNeedsRedraw(true)
}

func (e *ExportSheet) drawHit(canvas widget.Canvas, h exportHit) {
	ink := rgb(e.theme.Ink)
	muted := rgb(e.theme.Muted)
	switch {
	case h.id == "format":
		canvas.DrawRoundRect(h.box, rgb(exportWell), 6)
		drawFace(canvas, "Format", geometry.NewRect(h.box.Min.X, h.box.Min.Y-exportLabel-6, h.box.Width(), exportLabel), 13, ink, false, e.theme.UIFont)
		label := e.fitted(canvas, e.brandLabel(), h.box.Width()-36, 13)
		drawFace(canvas, label, geometry.NewRect(h.box.Min.X+10, h.box.Min.Y+6, h.box.Width()-36, 20), 13, ink, false, e.theme.UIFont)
		drawFace(canvas, "▾", geometry.NewRect(h.box.Max.X-22, h.box.Min.Y+6, 16, 20), 13, muted, false, e.theme.UIFont)
	case h.id == "template":
		canvas.DrawRoundRect(h.box, rgb(exportWell), 6)
		drawFace(canvas, "Template", geometry.NewRect(h.box.Min.X, h.box.Min.Y-exportLabel-6, h.box.Width(), exportLabel), 13, ink, false, e.theme.UIFont)
		label := e.fitted(canvas, e.templateLabel(), h.box.Width()-36, 13)
		drawFace(canvas, label, geometry.NewRect(h.box.Min.X+10, h.box.Min.Y+6, h.box.Width()-36, 20), 13, ink, false, e.theme.UIFont)
		drawFace(canvas, "▾", geometry.NewRect(h.box.Max.X-22, h.box.Min.Y+6, 16, 20), 13, muted, false, e.theme.UIFont)
	case h.id == "settings":
		canvas.DrawRoundRect(h.box, rgb(exportWell), 6)
		drawFace(canvas, "Settings", geometry.NewRect(h.box.Min.X+10, h.box.Min.Y+6, h.box.Width()-36, 20), 13, ink, false, e.theme.UIFont)
		drawFace(canvas, "›", geometry.NewRect(h.box.Max.X-22, h.box.Min.Y+6, 16, 20), 13, muted, false, e.theme.UIFont)
	case h.id == "saved-files":
		drawFace(canvas, "Uses the files saved on the server.", h.box, 12, muted, false, e.theme.UIFont)
	case h.id == "go":
		canvas.DrawRoundRect(h.box, rgb(exportWell), 6)
		drawCenter(canvas, "Export", h.box, 13, ink, false, e.theme.UIFont)
	case h.id == "done":
		canvas.DrawRoundRect(h.box, rgb(exportWell), 6)
		drawCenter(canvas, "Done", h.box, 13, ink, false, e.theme.UIFont)
	case h.id == "open":
		e.drawCheck(canvas, h.box, e.openFile)
		drawFace(canvas, "Open after export", geometry.NewRect(h.box.Min.X+28, h.box.Min.Y+4, h.box.Width()-28, 20), 13, ink, false, e.theme.UIFont)
	case h.id == "temp" || h.id == "current" || h.id == "pick":
		e.drawRadio(canvas, h.box, e.dest == h.id)
		drawFace(canvas, destLabel(h.id), geometry.NewRect(h.box.Min.X+28, h.box.Min.Y+4, h.box.Width()-28, 20), 13, ink, false, e.theme.UIFont)
	case strings.HasPrefix(h.id, "brand:"):
		e.drawChoice(canvas, h, h.id[len("brand:"):], e.brand, e.brandText)
	case strings.HasPrefix(h.id, "template:"):
		id := h.id[len("template:"):]
		e.drawChoice(canvas, h, id, e.template, e.templateText)
	case strings.HasPrefix(h.id, "up:") || strings.HasPrefix(h.id, "down:"):
		e.drawNudge(canvas, h)
	case strings.HasPrefix(h.id, "file:"):
		i, ok := hitIndex(h.id, "file:")
		if !ok {
			return
		}
		label := e.fitted(canvas, e.fileLabel(i), h.box.Width()-8, 13)
		drawFace(canvas, label, geometry.NewRect(h.box.Min.X+4, h.box.Min.Y+4, h.box.Width()-8, 20), 13, ink, false, e.theme.UIFont)
	}
}

func (e *ExportSheet) drawChoice(canvas widget.Canvas, h exportHit, id, current string, text func(string) string) {
	ink := rgb(e.theme.Ink)
	if id == current {
		canvas.DrawRoundRect(h.box, rgba(e.theme.Selection), 4)
		drawFace(canvas, "✓", geometry.NewRect(h.box.Min.X+4, h.box.Min.Y+4, 16, 20), 13, rgb(e.theme.Accent), false, e.theme.UIFont)
	}
	drawFace(canvas, text(id), geometry.NewRect(h.box.Min.X+24, h.box.Min.Y+4, h.box.Width()-28, 20), 13, ink, false, e.theme.UIFont)
}

func (e *ExportSheet) drawNudge(canvas widget.Canvas, h exportHit) {
	up := strings.HasPrefix(h.id, "up:")
	prefix := "down:"
	mark := "↓"
	if up {
		prefix = "up:"
		mark = "↑"
	}
	i, ok := hitIndex(h.id, prefix)
	if !ok {
		return
	}
	col := rgb(e.theme.Ink)
	if (up && i == 0) || (!up && i == len(e.files)-1) {
		col = rgb(e.theme.Muted)
	}
	drawCenter(canvas, mark, h.box, 13, col, false, e.theme.UIFont)
}

func (e *ExportSheet) drawScope(canvas widget.Canvas, hits []exportHit) {
	var doc, proj geometry.Rect
	seen := false
	for _, h := range hits {
		switch h.id {
		case "scope-doc":
			doc = h.box
			seen = true
		case "scope-project":
			proj = h.box
		}
	}
	if !seen {
		return
	}
	track := geometry.NewRect(doc.Min.X, doc.Min.Y, proj.Max.X-doc.Min.X, doc.Height())
	canvas.DrawRoundRect(track, rgb(e.theme.CodeFill), 6)
	active := doc
	if e.scope == "project" {
		active = proj
	}
	canvas.DrawRoundRect(active, rgb(exportWell), 6)
	canvas.StrokeRoundRect(track, rgb(e.theme.Rule), 6, 1)
	canvas.DrawLine(
		geometry.Pt(doc.Max.X, track.Min.Y+4),
		geometry.Pt(doc.Max.X, track.Max.Y-4),
		rgb(e.theme.Rule), 1,
	)
	e.scopeLabel(canvas, doc, "Document", e.scope != "project")
	e.scopeLabel(canvas, proj, "Project", e.scope == "project")
}

func (e *ExportSheet) scopeLabel(canvas widget.Canvas, box geometry.Rect, text string, on bool) {
	col := rgb(e.theme.Muted)
	if on {
		col = rgb(e.theme.Accent)
	}
	drawCenter(canvas, text, box, 13, col, false, e.theme.UIFont)
}

func (e *ExportSheet) drawRadio(canvas widget.Canvas, row geometry.Rect, on bool) {
	c := geometry.Pt(row.Min.X+10, row.Min.Y+row.Height()/2)
	canvas.StrokeCircle(c, 7, rgb(e.theme.Muted), 1)
	if on {
		canvas.DrawCircle(c, 4, rgb(e.theme.Accent))
	}
}

func (e *ExportSheet) drawCheck(canvas widget.Canvas, row geometry.Rect, on bool) {
	box := geometry.NewRect(row.Min.X+2, row.Min.Y+(row.Height()-16)/2, 16, 16)
	if on {
		canvas.DrawRoundRect(box, rgb(e.theme.Accent), 3)
		drawFace(canvas, "✓", box, 12, rgb(0xFFFFFF), false, e.theme.UIFont)
		return
	}
	canvas.StrokeRoundRect(box, rgb(e.theme.Muted), 3, 1)
}

func (e *ExportSheet) drawCaret(canvas widget.Canvas, card geometry.Rect) {
	anchor := e.anchorLocal()
	if anchor.Width() <= 0 {
		return
	}
	tipX := anchor.Min.X + anchor.Width()/2
	if tipX < card.Min.X+16 {
		tipX = card.Min.X + 16
	}
	if tipX > card.Max.X-16 {
		tipX = card.Max.X - 16
	}
	base := card.Min.Y
	color := rgb(exportCard)
	for i := float32(0); i < 8; i++ {
		half := (8 - i)
		canvas.DrawLine(geometry.Pt(tipX-half, base-8+i), geometry.Pt(tipX+half, base-8+i), color, 1)
	}
}

func (e *ExportSheet) anchorLocal() geometry.Rect {
	if e.anchor.Width() <= 0 {
		return geometry.Rect{}
	}
	origin := screenOrigin(e)
	a := e.anchor
	return geometry.NewRect(a.Min.X-origin.X, a.Min.Y-origin.Y, a.Width(), a.Height())
}

func (e *ExportSheet) brandLabel() string {
	return e.brandText(e.brand)
}

func (e *ExportSheet) brandText(id string) string {
	for _, f := range e.formats {
		if f.ID == id {
			return f.Label
		}
	}
	return id
}

func (e *ExportSheet) templateLabel() string {
	id := e.template
	if id == "" {
		id = brandOf(e.brand)
	}
	return e.templateText(id)
}

func (e *ExportSheet) templateText(id string) string {
	for _, f := range e.templates() {
		if f.ID == id {
			return f.Label
		}
	}
	if id == "" {
		return "Template"
	}
	return id
}

func (e *ExportSheet) templates() []ExportFormat {
	seen := map[string]bool{}
	var out []ExportFormat
	for _, f := range e.formats {
		id := brandOf(f.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, ExportFormat{ID: id, Label: strings.TrimSuffix(strings.TrimSuffix(f.Label, " (docx)"), " (pdf)")})
	}
	return out
}

func (e *ExportSheet) applyTemplate() {
	id := brandOf(e.template)
	if id == "" {
		return
	}
	suffix := ""
	switch {
	case strings.HasSuffix(e.brand, ".docx"):
		suffix = ".docx"
	case strings.HasSuffix(e.brand, ".pdf"):
		suffix = ".pdf"
	}
	want := id + suffix
	for _, f := range e.formats {
		if f.ID == want {
			e.brand = f.ID
			return
		}
	}
	for _, f := range e.formats {
		if brandOf(f.ID) == id {
			e.brand = f.ID
			return
		}
	}
	// Preserve an unavailable saved ID so Export can report the missing pack.
	e.brand = want
}

func (e *ExportSheet) fileLabel(i int) string {
	if i < 0 || i >= len(e.files) {
		return ""
	}
	rel := e.files[i]
	base := filepath.Base(rel)
	for j, other := range e.files {
		if j == i || filepath.Base(other) != base {
			continue
		}
		parent := filepath.Base(filepath.Dir(filepath.FromSlash(rel)))
		if parent != "" && parent != "." {
			return parent + "/" + base
		}
	}
	return base
}

func (e *ExportSheet) move(i, dir int) {
	j := i + dir
	if i < 0 || j < 0 || i >= len(e.files) || j >= len(e.files) {
		return
	}
	e.files[i], e.files[j] = e.files[j], e.files[i]
	if j < e.fileAt {
		e.fileAt = j
	}
	if j >= e.fileAt+exportFileWin {
		e.fileAt = j - exportFileWin + 1
	}
}

func (e *ExportSheet) scrollFiles(deltaY float32) {
	if len(e.files) <= exportFileWin {
		return
	}
	shift := WheelScroll(deltaY, runtime.GOOS == "darwin")
	if shift > 0 {
		e.fileAt++
	} else if shift < 0 {
		e.fileAt--
	}
	e.clampFiles()
	e.SetNeedsRedraw(true)
}

func (e *ExportSheet) clampFiles() {
	maxAt := len(e.files) - exportFileWin
	if maxAt < 0 {
		maxAt = 0
	}
	if e.fileAt < 0 {
		e.fileAt = 0
	}
	if e.fileAt > maxAt {
		e.fileAt = maxAt
	}
}

func (e *ExportSheet) restoreSnap() {
	if !e.snapped {
		return
	}
	e.files = append([]string(nil), e.snapFiles...)
	e.template = e.snapTmpl
	e.snapped = false
}

func (e *ExportSheet) fitted(canvas widget.Canvas, text string, width, size float32) string {
	if text == "" || textWidth(canvas, text, size, rgb(e.theme.Ink), true, e.theme.UIFont) <= width {
		return text
	}
	r := []rune(text)
	for n := len(r) - 1; n > 8; n-- {
		head := n / 2
		tail := n - head
		if tail > len(r) {
			continue
		}
		s := string(r[:head]) + "…" + string(r[len(r)-tail:])
		if textWidth(canvas, s, size, rgb(e.theme.Ink), true, e.theme.UIFont) <= width {
			return s
		}
	}
	return "…"
}

func brandOf(id string) string {
	id = strings.TrimSuffix(id, ".docx")
	return strings.TrimSuffix(id, ".pdf")
}

func hitIndex(id, prefix string) (int, bool) {
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(id[len(prefix):])
	if err != nil {
		return 0, false
	}
	return n, true
}

func destLabel(id string) string {
	switch id {
	case DestCurrent:
		return "Current directory"
	case DestPick:
		return "Select directory"
	default:
		return "Temporary directory"
	}
}

func drawCenter(canvas widget.Canvas, text string, rect geometry.Rect, size float32, col widget.Color, bold bool, family string) {
	w := textWidth(canvas, text, size, col, bold, family)
	x := rect.Min.X + (rect.Width()-w)/2
	if x < rect.Min.X {
		x = rect.Min.X
	}
	drawFace(canvas, text, geometry.NewRect(x, rect.Min.Y, rect.Max.X-x, rect.Height()), size, col, bold, family)
}

func invalidate(ctx widget.Context) {
	if ctx != nil {
		ctx.Invalidate()
	}
}
