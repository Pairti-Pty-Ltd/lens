package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingSessionIsEmpty(t *testing.T) {
	st, err := Load(filepath.Join(t.TempDir(), "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if st.RestoreTree || len(st.Docs) != 0 || st.Split != -1 {
		t.Fatalf("%+v", st)
	}
}

func TestCorruptSessionIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Load(path)
	if err == nil {
		t.Fatal("expected an error")
	}
	if st.RestoreTree || len(st.Docs) != 0 {
		t.Fatalf("corrupt file was applied: %+v", st)
	}
}

func TestSessionRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairti-lense", "session.json")
	in := State{
		Docs:     []Doc{{Path: "/notes/a.md"}, {Path: "/notes/b.md", Raw: true}},
		Active:   1,
		Split:    -1,
		Expanded: []string{"/notes", "/notes/nest"},
		Scroll:   40,
		Sidebar:  280,
	}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.RestoreTree || got.Active != 1 || len(got.Docs) != 2 || !got.Docs[1].Raw {
		t.Fatalf("%+v", got)
	}
	if got.Scroll != 40 || got.Sidebar != 280 || len(got.Expanded) != 2 {
		t.Fatalf("%+v", got)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "/notes/b.md") {
		t.Fatalf("%s", b)
	}
}
