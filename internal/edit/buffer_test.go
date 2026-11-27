package edit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUndoDropsTheLastCharacter(t *testing.T) {
	b := New("hi")
	b.Place(0, 2)
	b.Insert('!')
	b.Insert('?')
	if !b.Undo() || b.Text() != "hi!" {
		t.Fatalf("text %q", b.Text())
	}
	if !b.Redo() || b.Text() != "hi!?" {
		t.Fatalf("redo %q", b.Text())
	}
	if b.Cursor() != 4 {
		t.Fatalf("caret %d", b.Cursor())
	}
}

func TestInsertSessionIsOneUndo(t *testing.T) {
	b := New("go")
	b.Place(0, 2)
	b.BeginGroup()
	b.Insert(' ')
	b.Insert('l')
	b.Insert('n')
	b.EndGroup()
	if !b.Undo() || b.Text() != "go" {
		t.Fatalf("undo group %q", b.Text())
	}
	if !b.Redo() || b.Text() != "go ln" {
		t.Fatalf("redo group %q", b.Text())
	}
	b.Insert('x')
	if b.Redo() {
		t.Fatal("redo survived a new edit")
	}
}

func TestDragSelectionDeletesTheWord(t *testing.T) {
	b := New("hello")
	b.Select(0, 2)
	if !b.DeleteSelection() || b.Text() != "llo" {
		t.Fatalf("text %q", b.Text())
	}
}

func TestFindLineWraps(t *testing.T) {
	text := "alpha\nbeta membrane\ngamma\n"
	if FindLine(text, "membrane", 2) != 2 {
		t.Fatalf("line %d", FindLine(text, "membrane", 2))
	}
	if FindLine(text, "missing", 0) != 0 {
		t.Fatal("hit")
	}
}

func TestInsertBackspaceAndSave(t *testing.T) {
	b := New("# Hi\n")
	if b.Dirty() {
		t.Fatal("fresh buffer is dirty")
	}
	b.Place(0, 4)
	b.Insert('!')
	if b.Text() != "# Hi!\n" {
		t.Fatalf("text = %q", b.Text())
	}
	if !b.Dirty() {
		t.Fatal("edit was not dirty")
	}
	b.Backspace()
	if b.Text() != "# Hi\n" || b.Dirty() {
		t.Fatalf("after backspace %q dirty=%v", b.Text(), b.Dirty())
	}
	b.Insert('x')
	b.MarkSaved()
	if b.Dirty() || b.Text() != "# Hix\n" {
		t.Fatalf("saved = %q dirty=%v", b.Text(), b.Dirty())
	}
}

func TestUpDownKeepsColumn(t *testing.T) {
	b := New("abcd\nxy\nz")
	b.Place(0, 3)
	b.Down()
	if b.CaretLine() != 1 || b.CaretCol() != 2 {
		t.Fatalf("down line=%d col=%d", b.CaretLine(), b.CaretCol())
	}
	b.Down()
	if b.CaretLine() != 2 || b.CaretCol() != 1 {
		t.Fatalf("second down line=%d col=%d", b.CaretLine(), b.CaretCol())
	}
	b.Up()
	b.Up()
	if b.CaretLine() != 0 || b.CaretCol() != 3 {
		t.Fatalf("back up line=%d col=%d", b.CaretLine(), b.CaretCol())
	}
}

func TestNewline(t *testing.T) {
	b := New("ab")
	b.Place(0, 1)
	b.Insert('\n')
	if b.Text() != "a\nb" || b.CaretLine() != 1 || b.CaretCol() != 0 {
		t.Fatalf("text %q line=%d col=%d", b.Text(), b.CaretLine(), b.CaretCol())
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("# Old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, "# New\n"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# New\n" {
		t.Fatalf("disk = %q", got)
	}
}
