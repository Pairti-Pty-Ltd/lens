package main

import (
	"os"
	"path/filepath"
	"testing"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/session"
)

func TestQuarantineMovesACorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := session.Load(path)
	if err == nil {
		t.Fatal("expected an error")
	}
	moved := quarantine(path, err)
	if moved != path+".bad" {
		t.Fatalf("moved to %q", moved)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("original still present: %v", statErr)
	}
	body, err := os.ReadFile(moved)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "{not json" {
		t.Fatalf("aside copy = %q", body)
	}

	if err := os.WriteFile(path, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = session.Load(path)
	if err == nil {
		t.Fatal("expected an error")
	}
	second := quarantine(path, err)
	if second != path+".bad.1" {
		t.Fatalf("second copy = %q", second)
	}
	if _, statErr := os.Stat(moved); statErr != nil {
		t.Fatal(statErr)
	}
}

func TestQuarantineLeavesAMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if got := quarantine(path, os.ErrNotExist); got != "" {
		t.Fatalf("missing file moved to %q", got)
	}
	if got := quarantine(path, nil); got != "" {
		t.Fatalf("nil error moved to %q", got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("a missing file was created")
	}
}

func TestRestoreSkipsANoteThatPanics(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	good := filepath.Join(dir, "good.md")
	bad := filepath.Join(dir, "bad.md")
	for _, path := range []string{good, bad} {
		if err := os.WriteFile(path, []byte("# Note\n\nHello.\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s.projects = []doc.Project{{Name: "notes", Root: dir}}
	s.holdSave = true
	s.openSavedHook = func(path string) {
		if path == bad {
			panic("boom")
		}
	}
	s.restoreDocs(session.State{
		Docs: []session.Doc{
			{Path: bad},
			{Path: good},
		},
		Active: 1,
		Split:  -1,
	})
	if len(s.docs) != 1 || s.docs[0].path != good {
		t.Fatalf("docs %+v", s.docs)
	}
	if s.docPath != good || s.active != 0 {
		t.Fatalf("active %d path %s", s.active, s.docPath)
	}
}

func TestLineNumberPreferencesSurviveRestart(t *testing.T) {
	s := testShell()
	s.sessionPath = filepath.Join(t.TempDir(), "session.json")
	openNote(t, s, "# Title\n")
	s.onTool("Line numbers")
	s.pick("Raw")
	s.onTool("Line numbers")
	saved, err := session.Load(s.sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	restarted := testShell()
	restarted.restoreDocs(saved)
	if !restarted.formattedNumbers || !restarted.hideRawNumbers {
		t.Fatalf("restored formatted numbers %v, hide raw numbers %v", restarted.formattedNumbers, restarted.hideRawNumbers)
	}
}
