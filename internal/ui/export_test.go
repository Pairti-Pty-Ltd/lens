package ui

import (
	"runtime"
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestExportSheetIsSolid(t *testing.T) {
	sheet := NewExportSheet(nil)
	sheet.Show("orchard-survey-coverage.md", geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	canvas := uitest.DrawWidget(sheet)
	for _, r := range canvas.Rects {
		if r.Bounds.Width() > 700 && r.Color.A < 1 {
			t.Fatalf("scrim alpha %v", r.Color.A)
		}
	}
	if len(canvas.RoundRects) == 0 {
		t.Fatal("card was not drawn")
	}
	for _, r := range canvas.RoundRects {
		if r.Color.A < 1 {
			t.Fatalf("card alpha %v", r.Color.A)
		}
	}
}

func TestExportSheetShowsTheNoteOnce(t *testing.T) {
	sheet := NewExportSheet(nil)
	sheet.SetFormats([]ExportFormat{
		{ID: "cedar", Label: "Cedar (pdf)"},
		{ID: "default.pdf", Label: "Default (pdf)"},
		{ID: "grove-p", Label: "GROVE-P (pdf)"},
	})
	sheet.Show("orchard-survey-coverage.md", geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	got := painted(uitest.DrawWidget(sheet))
	if strings.Count(got, "orchard-survey-coverage.md") != 1 {
		t.Fatalf("filename:\n%s", got)
	}
	if strings.Count(got, "Default (pdf)") != 1 {
		t.Fatalf("format row:\n%s", got)
	}
	for _, absent := range []string{"Cedar", "GROVE-P", "HTML", "LaTeX", "Word", "docx"} {
		if strings.Contains(got, absent) {
			t.Fatalf("%s was listed while the menu was closed:\n%s", absent, got)
		}
	}
	for _, want := range []string{"Export", "Temporary directory", "Current directory", "Select directory", "Open after export"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s\n%s", want, got)
		}
	}
}

func TestExportSheetCutsALongName(t *testing.T) {
	const name = "orchard-survey-coverage-and-the-seasonal-planting-review-for-the-autumn-harvest.md"
	sheet := NewExportSheet(nil)
	sheet.Show(name, geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	got := painted(uitest.DrawWidget(sheet))
	if strings.Contains(got, name) {
		t.Fatalf("long name was painted whole:\n%s", got)
	}
	if !strings.Contains(got, "…") {
		t.Fatalf("name was not cut:\n%s", got)
	}
	if strings.Count(got, "…") != 1 {
		t.Fatalf("name repeated:\n%s", got)
	}
}

func TestExportSheetConfirmsTheChoice(t *testing.T) {
	var got ExportChoice
	sheet := NewExportSheet(func(c ExportChoice) { got = c })
	sheet.SetFormats([]ExportFormat{
		{ID: "cedar", Label: "Cedar (pdf)"},
		{ID: "default.pdf", Label: "Default (pdf)"},
		{ID: "grove-p", Label: "GROVE-P (pdf)"},
	})
	sheet.Show("orchard-survey-coverage.md", geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	click(t, sheet, "format")
	if !strings.Contains(painted(uitest.DrawWidget(sheet)), "Cedar (pdf)") {
		t.Fatal("format menu did not open")
	}
	click(t, sheet, "brand:cedar")
	click(t, sheet, "current")
	click(t, sheet, "open")
	click(t, sheet, "go")
	if got.Brand != "cedar" || got.Dest != DestCurrent || got.Open {
		t.Fatalf("choice %+v", got)
	}
	if sheet.Shown() {
		t.Fatal("sheet stayed open")
	}
}

func TestExportSheetDismisses(t *testing.T) {
	called := false
	sheet := NewExportSheet(func(ExportChoice) { called = true })
	sheet.Show("orchard-survey-coverage.md", geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	if !uitest.SimulateClick(sheet, 8, 8) {
		t.Fatal("scrim click was not consumed")
	}
	if sheet.Shown() || called {
		t.Fatal("scrim click exported or left the sheet open")
	}
	sheet.Show("orchard-survey-coverage.md", geometry.Rect{})
	if !sheet.Event(nil, event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, 0)) {
		t.Fatal("escape was not consumed")
	}
	if sheet.Shown() {
		t.Fatal("escape left the sheet open")
	}
}

func TestExportSheetProjectSettings(t *testing.T) {
	const long = "orchard-survey-coverage-and-the-seasonal-planting-review-for-the-autumn-harvest.md"
	var got ExportChoice
	var savedFiles []string
	var savedTmpl string
	sheet := NewExportSheet(func(c ExportChoice) { got = c })
	sheet.SetFormats([]ExportFormat{
		{ID: "cedar.pdf", Label: "Cedar (pdf)"},
		{ID: "cedar.docx", Label: "Cedar (docx)"},
		{ID: "default.pdf", Label: "Default (pdf)"},
		{ID: "default.docx", Label: "Default (docx)"},
		{ID: "grove-p.pdf", Label: "GROVE-P (pdf)"},
		{ID: "grove-p.docx", Label: "GROVE-P (docx)"},
	})
	sheet.SetProject("west-orchard", []string{"notes/" + long, "setting-up.md"}, "default.docx")
	sheet.OnProjectSave(func(files []string, template string) {
		savedFiles = append([]string(nil), files...)
		savedTmpl = template
	})
	sheet.Show("setting-up.md", geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))

	if _, ok := sheet.Spot("settings"); ok {
		t.Fatal("document export offered project settings")
	}
	doc := painted(uitest.DrawWidget(sheet))
	if strings.Count(doc, "setting-up.md") != 1 || strings.Contains(doc, long) {
		t.Fatalf("document sheet:\n%s", doc)
	}
	for _, absent := range []string{"Cedar", "GROVE-P", "Settings"} {
		if strings.Contains(doc, absent) {
			t.Fatalf("%s was on the document sheet:\n%s", absent, doc)
		}
	}

	click(t, sheet, "scope-project")
	proj := painted(uitest.DrawWidget(sheet))
	if strings.Count(proj, "west-orchard") != 1 || strings.Contains(proj, "setting-up.md") {
		t.Fatalf("project sheet:\n%s", proj)
	}
	if !strings.Contains(proj, "Default (pdf)") || strings.Contains(proj, "Cedar") || strings.Contains(proj, "GROVE-P") {
		t.Fatalf("template was not applied as the open format:\n%s", proj)
	}

	click(t, sheet, "settings")
	body := painted(uitest.DrawWidget(sheet))
	if strings.Count(body, "west-orchard") != 1 || strings.Count(body, "Default") != 1 {
		t.Fatalf("settings identity:\n%s", body)
	}
	if strings.Contains(body, long) || strings.Contains(body, "notes/") {
		t.Fatalf("full path was painted:\n%s", body)
	}
	if !strings.Contains(body, "…") || !strings.Contains(body, "setting-up.md") {
		t.Fatalf("short names:\n%s", body)
	}
	for _, absent := range []string{"Cedar", "GROVE-P", "(pdf)", "(docx)", "Temporary directory"} {
		if strings.Contains(body, absent) {
			t.Fatalf("%s was listed while the template menu was closed:\n%s", absent, body)
		}
	}

	click(t, sheet, "up:1")
	ordered := painted(uitest.DrawWidget(sheet))
	if strings.Index(ordered, "setting-up.md") > strings.Index(ordered, "…") {
		t.Fatalf("up did not move the file:\n%s", ordered)
	}
	click(t, sheet, "template")
	menu := painted(uitest.DrawWidget(sheet))
	if !strings.Contains(menu, "Cedar") || !strings.Contains(menu, "GROVE-P") {
		t.Fatalf("template menu:\n%s", menu)
	}
	click(t, sheet, "template:cedar")
	click(t, sheet, "done")
	if savedTmpl != "cedar" || len(savedFiles) != 2 || savedFiles[0] != "setting-up.md" {
		t.Fatalf("saved template %q files %v", savedTmpl, savedFiles)
	}
	back := painted(uitest.DrawWidget(sheet))
	if !strings.Contains(back, "Cedar (pdf)") || strings.Contains(back, "setting-up.md") {
		t.Fatalf("back on the export sheet:\n%s", back)
	}
	if _, ok := sheet.Spot("settings"); !ok {
		t.Fatal("settings did not return to the project export")
	}
	click(t, sheet, "current")
	click(t, sheet, "go")
	if !got.Project || got.Brand != "cedar.pdf" || got.Dest != DestCurrent || !got.Open {
		t.Fatalf("choice %+v", got)
	}
	if len(got.Files) != 2 || got.Files[0] != "setting-up.md" || got.Files[1] != "notes/"+long {
		t.Fatalf("files %v", got.Files)
	}
}

func TestExportSheetSettingsEscapeRestoresTheOrder(t *testing.T) {
	var got ExportChoice
	sheet := NewExportSheet(func(c ExportChoice) { got = c })
	sheet.SetProject("west-orchard", []string{"b.md", "a.md"}, "default")
	sheet.Show("setting-up.md", geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	click(t, sheet, "scope-project")
	click(t, sheet, "settings")
	click(t, sheet, "up:1")
	if !sheet.Event(nil, event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, 0)) {
		t.Fatal("escape was not consumed")
	}
	if !sheet.Shown() {
		t.Fatal("escape closed the export")
	}
	click(t, sheet, "go")
	if !got.Project || len(got.Files) != 2 || got.Files[0] != "b.md" || got.Files[1] != "a.md" {
		t.Fatalf("choice %+v", got)
	}
}

func TestExportSheetSettingsWindowHidesTheRest(t *testing.T) {
	sheet := NewExportSheet(nil)
	files := []string{"a.md", "b.md", "c.md", "d.md", "e.md", "f.md", "g.md", "h.md"}
	sheet.SetProject("west-orchard", files, "default")
	sheet.Show("a.md", geometry.Rect{})
	sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	click(t, sheet, "scope-project")
	click(t, sheet, "settings")
	got := painted(uitest.DrawWidget(sheet))
	if strings.Contains(got, "g.md") || strings.Contains(got, "h.md") {
		t.Fatalf("the order listed every file:\n%s", got)
	}
	delta := float32(4)
	if WheelScroll(delta, runtime.GOOS == "darwin") < 0 {
		delta = -delta
	}
	if !sheet.Event(nil, uitest.WheelScroll(40, 80, delta)) {
		t.Fatal("wheel was not consumed")
	}
	got = painted(uitest.DrawWidget(sheet))
	if strings.Contains(got, "a.md") || !strings.Contains(got, "g.md") || strings.Contains(got, "h.md") {
		t.Fatalf("after one step:\n%s", got)
	}
}

func TestCoverKeepsTheExportClick(t *testing.T) {
	got := ""
	bar := NewToolBar(DocumentTools(), func(name string) { got = name })
	sheet := NewExportSheet(nil)
	sheet.Show("orchard-survey-coverage.md", geometry.Rect{})
	cover := NewCover(bar, sheet)
	uitest.LayoutWidget(cover, 800, 600)
	cover.SetBounds(geometry.NewRect(0, 0, 800, 600))
	if !uitest.SimulateClick(cover, 12, 14) {
		t.Fatal("click was not consumed")
	}
	if got != "" {
		t.Fatalf("toolbar received %s", got)
	}
	if sheet.Shown() {
		t.Fatal("scrim click left the sheet open")
	}
}

func click(t *testing.T, sheet *ExportSheet, id string) {
	t.Helper()
	p, ok := sheet.Spot(id)
	if !ok {
		t.Fatalf("no control %s", id)
	}
	if !uitest.SimulateClick(sheet, p.X, p.Y) {
		t.Fatalf("click missed %s", id)
	}
}

func painted(c *uitest.MockCanvas) string {
	var b strings.Builder
	for _, tx := range c.StyledTexts {
		b.WriteString(tx.Text)
		b.WriteByte('\n')
	}
	for _, tx := range c.Texts {
		b.WriteString(tx.Text)
		b.WriteByte('\n')
	}
	return b.String()
}

func TestMissingSavedPackKeepsItsID(t *testing.T) {
	sheet := NewExportSheet(nil)
	sheet.SetFormats([]ExportFormat{{ID: "default.pdf", Label: "Default (pdf)"}})
	sheet.SetProject("Orchard", []string{"note.md"}, "retired")
	sheet.FixScope(true)
	if sheet.brand != "retired.pdf" {
		t.Fatalf("missing pack replaced with %q", sheet.brand)
	}
}
