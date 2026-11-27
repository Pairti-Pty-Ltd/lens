package edit

import "testing"

func TestMergeKeepsBothSidesWhenTheyTouchDifferentLines(t *testing.T) {
	base := "a\nb\nc\n"
	ours := "a\nB\nc\n"
	theirs := "a\nb\nC\n"
	got, n := Merge(base, ours, theirs)
	if n != 0 || got != "a\nB\nC\n" {
		t.Fatalf("n=%d got %q", n, got)
	}
}

func TestMergeMarksTheSameLine(t *testing.T) {
	got, n := Merge("a\nb\nc\n", "a\nB\nc\n", "a\nX\nc\n")
	if n != 1 {
		t.Fatalf("conflicts %d\n%s", n, got)
	}
	want := "a\n<<<<<<< ours\nB\n=======\nX\n>>>>>>> theirs\nc\n"
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestMergeSameEditIsNotAConflict(t *testing.T) {
	got, n := Merge("a\nb\n", "a\nB\n", "a\nB\n")
	if n != 0 || got != "a\nB\n" {
		t.Fatalf("n=%d got %q", n, got)
	}
}

func TestMergeInsertionAndDeletion(t *testing.T) {
	got, n := Merge("a\nb\nc\n", "a\nX\nb\nc\n", "a\nc\n")
	if n != 0 || got != "a\nX\nc\n" {
		t.Fatalf("n=%d got %q", n, got)
	}
}

func TestMergeDeleteAgainstEditConflicts(t *testing.T) {
	_, n := Merge("a\nb\nc\n", "a\nc\n", "a\nB\nc\n")
	if n != 1 {
		t.Fatal(n)
	}
}
