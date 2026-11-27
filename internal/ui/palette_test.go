package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestPaletteShowsTheSelectedNoteOnce(t *testing.T) {
	const (
		title = "Orchard survey coverage and the seasonal planting review"
		rel   = "grove/notes/orchard-survey-coverage-and-the-seasonal-planting-review-for-the-west-orchard.md"
		snip  = "the cohort calls we will not repeat"
		other = "a different secret phrase from the second note"
	)
	var opened string
	p := NewPalette(nil, func(h PaletteHit) { opened = h.Path })
	p.Show()
	p.SetBounds(geometry.NewRect(0, 0, 800, 600))
	p.query = "assay coverage"
	p.SetHits([]PaletteHit{
		{Title: title, Rel: rel, Snippet: snip, Path: "/Users/writer/src/grove/" + rel, Root: "/Users/writer/src/grove"},
		{Title: "Plate map", Rel: "notes/plate.md", Snippet: other, Path: "/notes/plate.md"},
	})
	text := painted(uitest.DrawWidget(p))
	if strings.Count(text, "assay coverage") != 1 {
		t.Fatalf("query:\n%s", text)
	}
	if strings.Contains(text, rel) || strings.Contains(text, "/Users/writer") {
		t.Fatalf("full path was painted:\n%s", text)
	}
	if strings.Count(text, snip) != 1 || strings.Contains(text, other) {
		t.Fatalf("snippet:\n%s", text)
	}
	if strings.Contains(text, "0.") {
		t.Fatalf("score was painted:\n%s", text)
	}
	down := event.NewKeyEvent(event.KeyPress, event.KeyDown, 0, 0)
	if !p.Event(nil, down) {
		t.Fatal("down was not used")
	}
	text = painted(uitest.DrawWidget(p))
	if strings.Contains(text, snip) || !strings.Contains(text, other) {
		t.Fatalf("after down:\n%s", text)
	}
	if !p.Event(nil, event.NewKeyEvent(event.KeyPress, event.KeyEnter, 0, 0)) {
		t.Fatal("enter was not used")
	}
	if opened != "/notes/plate.md" || p.Shown() {
		t.Fatalf("opened %q shown %v", opened, p.Shown())
	}
}

func TestPaletteGivesTheTitleADocumentMark(t *testing.T) {
	const (
		title = "Seasonal Observations from the Northern Orchard Survey"
		snip  = "topic sources: the proposal the second row must not repeat"
	)
	var opened string
	p := NewPalette(nil, func(h PaletteHit) { opened = h.Path })
	p.Show()
	p.SetBounds(geometry.NewRect(0, 0, 1200, 800))
	p.query = "Orchard Survey"
	p.SetHits([]PaletteHit{
		{
			Title:   "Orchard Survey and Sampling Plan",
			Rel:     "wiki/topics/orchard-survey-and-sampling-plan.md",
			Snippet: "…opic sources: [doc://research/orchard/PROPOSAL.md]",
			Path:    "/wiki/topics/orchard-survey-and-sampling-plan.md",
		},
		{Title: title, Snippet: snip, Path: "/notes/telemetry.md"},
	})
	card := p.card()
	if card.Width() < 600 {
		t.Fatalf("width %v", card.Width())
	}
	c := uitest.DrawWidget(p)
	var titleBox geometry.Rect
	for _, tx := range c.StyledTexts {
		if tx.Text == title {
			titleBox = tx.Bounds
		}
		if strings.Contains(tx.Text, snip) {
			t.Fatalf("closed row painted its snippet:\n%s", tx.Text)
		}
	}
	if titleBox.Width() == 0 {
		t.Fatal("title was not painted")
	}
	if titleBox.Min.X < card.Min.X+palPad+palIcon {
		t.Fatalf("title starts in the icon column: %v card %v", titleBox.Min.X, card.Min.X)
	}
	marked := false
	for _, ln := range c.Lines {
		if ln.From.X >= card.Min.X+palPad && ln.From.X < titleBox.Min.X && ln.From.Y >= titleBox.Min.Y && ln.From.Y <= titleBox.Max.Y {
			marked = true
		}
	}
	if !marked {
		t.Fatal("no document mark beside the title")
	}
	y := titleBox.Min.Y + 4
	if !p.Event(nil, &event.MouseEvent{MouseType: event.MousePress, Button: event.ButtonLeft, Position: geometry.Pt(titleBox.Min.X+8, y)}) {
		t.Fatal("click was not used")
	}
	if opened != "/notes/telemetry.md" {
		t.Fatalf("opened %q", opened)
	}
}
