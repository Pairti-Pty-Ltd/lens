package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/ui"
)

func TestHistoryScopesAndOlderCommits(t *testing.T) {
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-b", "main").CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	var data strings.Builder
	for i := 1; i <= historyBatch+1; i++ {
		fmt.Fprintf(&data, "commit refs/heads/main\ncommitter Test <test@example.com> %d +0000\ndata 1\nx\n", i)
		if i == 1 {
			data.WriteString("M 100644 inline note.md\ndata 1\na\n")
		}
		data.WriteString("\n")
	}
	cmd := exec.Command("git", "-C", root, "fast-import", "--quiet")
	cmd.Stdin = strings.NewReader(data.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(out))
	}
	s := testShell()
	s.docPath = filepath.Join(root, "note.md")
	s.histMode = true
	s.loadHistory()
	if len(s.commits) != 1 || s.histMore {
		t.Fatalf("file commits %d", len(s.commits))
	}
	s.toggleHist(s.commits[0].Hash)
	rows := s.historyItems()
	if len(rows) != 3 || rows[1].Label != "1 file changed" || rows[2].Label != "note.md" {
		t.Fatalf("expanded rows %+v", rows)
	}
	s.histRepo = true
	s.loadHistory()
	rows = s.historyItems()
	if len(rows) != historyBatch+1 || rows[historyBatch].Label != "Load older commits" {
		t.Fatalf("repository rows %d", len(rows))
	}
	s.onHistory(ui.Item{Path: "history:older"})
	if len(s.commits) != historyBatch+1 || s.histMore {
		t.Fatalf("older commits %d, more %v", len(s.commits), s.histMore)
	}
}

func TestHistoryDiffShowsCommitWithoutEditingWorkingBuffer(t *testing.T) {
	s := testShell()
	path := openNote(t, s, "# Plan\n\nReview quarterly reports.\n")
	dir := filepath.Dir(path)
	git := func(args ...string) string {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Tester", "-c", "user.email=test@example.com"}, args...)...)
		b, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v: %s", args, err, b)
		}
		return strings.TrimSpace(string(b))
	}
	git("init", "-b", "main")
	git("add", ".")
	git("commit", "-m", "Initial plan")
	if err := os.WriteFile(path, []byte("# Plan\n\nReview monthly reports.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	git("commit", "-am", "Change review interval")
	hash := git("rev-parse", "HEAD")
	s.showRaw(true)
	s.buf.Place(2, 4)
	s.buf.InsertText("draft")
	draft, cursor := s.buf.Text(), s.buf.Cursor()
	s.histMode = true
	s.loadHistory()
	wake := make(chan struct{}, 4)
	s.wake = func() { wake <- struct{}{} }
	s.onHistory(ui.Item{Path: "commit:" + hash, Rel: hash})
	previous := s.diff
	s.onHistory(ui.Item{Path: "commitfile:" + hash + ":" + filepath.Base(path), Revision: hash, Rel: filepath.Base(path)})
	if previous == s.diff || s.diff == nil {
		t.Fatal("did not start the selected comparison")
	}
	deadline := time.After(5 * time.Second)
	for !s.receiveDiff() {
		select {
		case <-wake:
		case <-deadline:
			t.Fatal("comparison did not finish")
		}
	}
	s.frame.SetBounds(geometry.NewRect(0, 0, 1100, 1200))
	uitest.LayoutWidgetTight(s.frame, 1100, 1200)
	var drawn strings.Builder
	for _, run := range uitest.DrawWidget(s.frame).StyledTexts {
		drawn.WriteString(run.Text)
	}
	for _, label := range []string{"Before", "After", "quarterly", "monthly"} {
		if !strings.Contains(drawn.String(), label) {
			t.Fatalf("missing %q in %q", label, drawn.String())
		}
	}
	s.onTool("Table")
	s.pick("Close diff")
	if s.diff != nil || !s.editing || s.buf.Text() != draft || s.buf.Cursor() != cursor || !s.buf.Dirty() {
		t.Fatal("diff changed the working buffer or its editing position")
	}
}
