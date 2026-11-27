package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddRemoveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pairti-lense", "library.json")
	a := filepath.Join(dir, "grove")
	b := filepath.Join(dir, "notes")
	if err := os.Mkdir(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(b, 0o755); err != nil {
		t.Fatal(err)
	}

	lib := &Library{}
	if err := lib.Add(a); err != nil {
		t.Fatal(err)
	}
	if err := lib.Add(a); err != nil {
		t.Fatal(err)
	}
	if err := lib.Add(b); err != nil {
		t.Fatal(err)
	}
	if len(lib.Paths) != 2 {
		t.Fatalf("paths = %d, want 2", len(lib.Paths))
	}
	if err := lib.Save(path); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(b); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Paths) != 2 {
		t.Fatalf("reloaded paths = %v, want both including the missing folder", got.Paths)
	}
	if !Absent(b) {
		t.Fatal("removed folder should be absent")
	}
	if Absent(a) {
		t.Fatal("existing folder should not be absent")
	}

	got.Remove(a)
	if err := got.Save(path); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Paths) != 1 || again.Paths[0] != filepath.Clean(b) {
		t.Fatalf("after remove = %v", again.Paths)
	}
}

func TestViewRoundTripOmitsTheDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "library.json")
	root := filepath.Join(dir, "pairti-lense")
	lib := &Library{}
	if err := lib.Add(root); err != nil {
		t.Fatal(err)
	}
	lib.SetView(root, View{ShowDot: true})
	if err := lib.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.View(root).ShowDot || got.View(root).ShowIgnored {
		t.Fatalf("view %+v", got.View(root))
	}
	got.SetView(root, View{})
	if err := got.Save(path); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "views") {
		t.Fatalf("default view was stored: %s", b)
	}
	got.SetView(root, View{Names: true})
	if err := got.Save(path); err != nil {
		t.Fatal(err)
	}
	named, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !named.View(root).Names {
		t.Fatal("file names were not saved")
	}
	named.SetView(root, View{AllDirs: true})
	if err := named.Save(path); err != nil {
		t.Fatal(err)
	}
	opened, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !opened.View(root).AllDirs {
		t.Fatal("all folders was not saved")
	}
}

func TestLoadMissing(t *testing.T) {
	lib, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lib.Paths) != 0 {
		t.Fatalf("paths = %v", lib.Paths)
	}
}
