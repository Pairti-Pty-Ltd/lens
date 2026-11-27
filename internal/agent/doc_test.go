package agent

import (
	"strings"
	"testing"

	"pairti-lense/internal/edit"
)

func TestWindowNumbersTheCaretLine(t *testing.T) {
	got := Window("a\nhello\nc\n", 2, 1)
	if got != "1|a\n2|hello\n3|c" {
		t.Fatalf("window %q", got)
	}
}

func TestApplySpanIsOneUndo(t *testing.T) {
	b := edit.New("café\n")
	changed, err := Apply(b, ApplyIn{Span: true, From: 3, To: 4, Text: "e", Expect: "é"})
	if err != nil || !changed || b.Text() != "cafe\n" {
		t.Fatalf("text %q err %v changed %v", b.Text(), err, changed)
	}
	if !b.Dirty() {
		t.Fatal("edit was saved")
	}
	if !b.Undo() || b.Text() != "café\n" || b.Dirty() {
		t.Fatalf("undo %q dirty %v", b.Text(), b.Dirty())
	}
	if _, err := Apply(b, ApplyIn{Span: true, From: 3, To: 4, Text: "e", Expect: "e"}); err == nil {
		t.Fatal("stale expect was applied")
	}
	if b.Text() != "café\n" {
		t.Fatalf("text %q", b.Text())
	}
}

func TestApplyReplaceAllRefusesAStaleDocument(t *testing.T) {
	b := edit.New("one")
	if _, err := Apply(b, ApplyIn{ReplaceAll: true, Text: "two", Expect: "nope"}); err == nil {
		t.Fatal("stale document was replaced")
	}
	changed, err := Apply(b, ApplyIn{ReplaceAll: true, Text: "two", Expect: "one"})
	if err != nil || !changed || b.Text() != "two" {
		t.Fatalf("text %q err %v", b.Text(), err)
	}
	if !b.Undo() || b.Text() != "one" {
		t.Fatalf("undo %q", b.Text())
	}
}

func TestApplyRejectsAMissingMode(t *testing.T) {
	b := edit.New("one")
	if _, err := Apply(b, ApplyIn{Text: "two"}); err == nil || !strings.Contains(err.Error(), "replaceAll") {
		t.Fatalf("err %v", err)
	}
	if b.Text() != "one" {
		t.Fatalf("text %q", b.Text())
	}
}
