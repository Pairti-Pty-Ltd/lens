package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchSeesAWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "assay.md")
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := New(func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.Set(path)
	time.Sleep(30 * time.Millisecond)
	if err := os.WriteFile(path, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if text, ok := w.Take(); ok {
			if text != "two\n" {
				t.Fatalf("text %q", text)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("write was not reported")
}

func TestWatchSeesANewDirectory(t *testing.T) {
	root := t.TempDir()
	runs := filepath.Join(root, "runs")
	if err := os.Mkdir(runs, 0o755); err != nil {
		t.Fatal(err)
	}
	w, err := New(func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.Tree([]string{runs})
	time.Sleep(30 * time.Millisecond)
	if err := os.Mkdir(filepath.Join(runs, "2026-11-23"), 0o755); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, dir := range w.TakeDirs() {
			if dir == runs {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("new directory was not reported")
}

func TestWatchSeesAFileInsideANewDirectory(t *testing.T) {
	root := t.TempDir()
	w, err := New(func() {})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	w.Tree([]string{root})
	time.Sleep(40 * time.Millisecond)
	docDir := filepath.Join(root, "doc")
	if err := os.Mkdir(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !waitDir(t, w, root) {
		t.Fatal("new directory was not reported")
	}
	if err := os.WriteFile(filepath.Join(docDir, "note.md"), []byte("# Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitDir(t, w, root) {
		t.Fatal("file inside the new directory was not reported")
	}
}

func waitDir(t *testing.T, w *Watch, want string) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, dir := range w.TakeDirs() {
			if dir == want {
				return true
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
