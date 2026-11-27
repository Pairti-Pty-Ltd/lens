package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestSharedWorkingTree(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%s: %v", out, err)
		}
	}
	git("init", "-q")
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("note.md", "café\n")
	write("private.md", "private")
	write("script.go", "private")
	git("add", "note.md", "script.go")
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	handler := (&library{root: root}).handler(t.TempDir())
	request := func(method, url, text, tag string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost:8081"+url, strings.NewReader(text))
		r.Header.Set("If-Match", tag)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if got := request("GET", "/api/documents", "", "").Body.String(); got != "[\"note.md\"]\n" {
		t.Fatal(got)
	}
	for _, p := range []string{"private.md", "script.go", "../note.md", ".git/config"} {
		if w := request("GET", "/api/document?path="+p, "", ""); w.Code != 404 {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
	const url = "/api/document?path=note.md"
	w := request("GET", url, "", "")
	if w.Code != 200 || w.Body.String() != "café\n" {
		t.Fatal(w)
	}
	tag := w.Header().Get("ETag")
	codes := make([]int, 2)
	var group sync.WaitGroup
	for i, text := range []string{"one", "two"} {
		group.Go(func() { codes[i] = request("PUT", url, text, tag).Code })
	}
	group.Wait()
	slices.Sort(codes)
	if !slices.Equal(codes, []int{204, 412}) {
		t.Fatal(codes)
	}
	tag = request("GET", url, "", "").Header().Get("ETag")
	write("note.md", "external")
	if w := request("PUT", url, "stale", tag); w.Code != http.StatusPreconditionFailed {
		t.Fatal(w)
	}
	if w := request("GET", url, "", ""); w.Body.String() != "external" {
		t.Fatal(w)
	}
	if err := os.Remove(filepath.Join(dir, "note.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("private.md", filepath.Join(dir, "note.md")); err != nil {
		t.Skip(err)
	}
	if w := request("GET", url, "", ""); w.Code != 404 {
		t.Fatal(w)
	}
}
