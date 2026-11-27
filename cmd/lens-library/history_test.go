package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHistoryUsesOnlyReachableCommitsAndServedMarkdown(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s, %v", args, out, err)
		}
		return strings.TrimSpace(string(out))
	}
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	git("config", "user.email", "writer@example.org")
	git("config", "user.name", "Writer")
	write("note.md", "library line\n")
	write("internal.txt", "forbidden text\n")
	git("add", ".")
	git("commit", "-qm", "Mixed documents")
	first := git("rev-parse", "HEAD")
	write("internal.txt", "different private text\n")
	git("commit", "-qam", "Private subject")
	git("switch", "-qc", "unserved")
	write("note.md", "other branch\n")
	git("commit", "-qam", "Other branch")
	other := git("rev-parse", "HEAD")
	git("switch", "-q", "main")
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	h := (&library{root: root}).handler(t.TempDir())
	call := func(action, query string, code int) string {
		t.Helper()
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "http://localhost:8081/api/history/"+action+"?"+query, nil)
		h.ServeHTTP(w, r)
		if w.Code != code {
			t.Fatalf("%s?%s: %d %s", action, query, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	log := call("log", "", 200)
	if strings.Contains(log, "Private subject") || !strings.Contains(log, "Mixed documents") {
		t.Fatal(log)
	}
	if got := call("files", "commit="+first, 200); got != "[\"note.md\"]\n" {
		t.Fatal(got)
	}
	diff := call("diff", "commit="+first, 200)
	if !strings.Contains(diff, "library line") || strings.Contains(diff, "forbidden text") || strings.Contains(diff, "internal.txt") {
		t.Fatal(diff)
	}
	for _, q := range []string{"path=internal.txt", "path=untracked.md", "path=" + url.QueryEscape("../note.md"), "commit=--all", "commit=" + other} {
		call("diff", q, 404)
	}
	call("files", "commit="+other, 404)
	call("log", "skip=-1", 400)
	if got := call("log", "skip=1", 200); strings.Contains(got, "Mixed documents") {
		t.Fatal(got)
	}
}
