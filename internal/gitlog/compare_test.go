package gitlog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func comparisonRepo(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	run(t, d, "init", "-b", "main")
	run(t, d, "config", "user.email", "dev@example.com")
	run(t, d, "config", "user.name", "Lens")
	return d
}
func putDiffFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
func commitFixture(t *testing.T, dir, msg string) {
	t.Helper()
	run(t, dir, "add", "-A")
	run(t, dir, "commit", "-m", msg)
}

func TestCompareWordSyntaxAndRename(t *testing.T) {
	d := comparisonRepo(t)
	oldName, newName := "before [a].md", "after\t[a].md"
	putDiffFixture(t, d, oldName, "# Review\n\nThe **proposal** has forty points and five questions.\n"+strings.Repeat("Stable context.\n", 8))
	commitFixture(t, d, "create")
	root, err := Compare(d, "HEAD", oldName)
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Files) != 1 || root.Files[0].OldPath != "" || root.Files[0].Hunks[0].Rows[0].New.Number != 1 {
		t.Fatalf("root %+v", root)
	}
	run(t, d, "mv", oldName, newName)
	putDiffFixture(t, d, newName, "# Review\n\nThe **proposal** has fifty points and six questions.\n"+strings.Repeat("Stable context.\n", 8))
	commitFixture(t, d, "revise")
	c, err := Compare(d, "HEAD", newName)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Files) != 1 || c.Files[0].OldPath != oldName || c.Files[0].NewPath != newName || c.Files[0].Status != "Renamed" {
		t.Fatalf("files %+v", c)
	}
	row := c.Files[0].Hunks[0].Rows[2]
	if row.Context || row.Old.Number != 3 || row.New.Number != 3 {
		t.Fatalf("row %+v", row)
	}
	var changed []string
	bold := false
	for _, r := range row.New.Runs {
		if r.Changed {
			changed = append(changed, r.Text)
		}
		if r.Bold && strings.Contains(r.Text, "proposal") {
			bold = true
		}
	}
	if strings.Join(changed, "|") != "fifty|six" || !bold {
		t.Fatalf("runs %+v", row.New.Runs)
	}
}
func TestCompareModeBinaryAndTypeOrdering(t *testing.T) {
	d := comparisonRepo(t)
	putDiffFixture(t, d, "a", "base")
	putDiffFixture(t, d, "b.bin", "before\x00")
	putDiffFixture(t, d, "c", "base")
	commitFixture(t, d, "create")
	if err := os.Remove(filepath.Join(d, "a")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("b.bin", filepath.Join(d, "a")); err != nil {
		t.Fatal(err)
	}
	putDiffFixture(t, d, "b.bin", "after\x00")
	if err := os.Chmod(filepath.Join(d, "c"), 0755); err != nil {
		t.Fatal(err)
	}
	commitFixture(t, d, "change")
	c, err := Compare(d, "HEAD", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Files) != 3 || len(c.Files[0].Hunks) != 2 || !strings.Contains(c.Files[1].Status, "Binary") || !strings.Contains(c.Files[2].Status, "100755") || c.Notice != "" {
		t.Fatalf("comparison %+v", c)
	}
	if !c.Files[0].Hunks[0].Rows[0].Old.NoNewline {
		t.Fatal("missing EOF marker")
	}
}
func TestCompareAlignmentAndLimit(t *testing.T) {
	d := comparisonRepo(t)
	putDiffFixture(t, d, "n.md", "First old sentence\nSecond old sentence\n")
	commitFixture(t, d, "create")
	putDiffFixture(t, d, "n.md", "First new sentence\nInserted material\nSecond new sentence\n")
	commitFixture(t, d, "edit")
	c, err := Compare(d, "HEAD", "n.md")
	if err != nil {
		t.Fatal(err)
	}
	rows := c.Files[0].Hunks[0].Rows
	if len(rows) != 3 || rows[1].Old != nil || rows[2].Old.Number != 2 || rows[2].New.Number != 3 {
		t.Fatalf("rows %+v", rows)
	}
	putDiffFixture(t, d, "huge", strings.Repeat("12345678\n", 70000))
	commitFixture(t, d, "large")
	c, err = Compare(d, "HEAD", "")
	if err != nil || c.Notice == "" || len(c.Files) != 1 || len(c.Files[0].Hunks[0].Rows) > maxPatchRows {
		t.Fatal(c.Notice, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CompareContext(ctx, d, "HEAD", ""); err != context.Canceled {
		t.Fatal(err)
	}
}

func TestCompareMergeUsesFirstParentAndFullSourceSyntax(t *testing.T) {
	d := comparisonRepo(t)
	first := "package lib\n\n/* block\n" + strings.Repeat("comment line\n", 8) + "original word in context\n" + strings.Repeat("comment line\n", 8) + "*/\n"
	putDiffFixture(t, d, "note.go", first)
	putDiffFixture(t, d, "retired.txt", "removed\n")
	commitFixture(t, d, "base")
	run(t, d, "checkout", "-b", "topic")
	putDiffFixture(t, d, "note.go", strings.Replace(first, "original", "updated", 1))
	run(t, d, "rm", "retired.txt")
	commitFixture(t, d, "revise")
	run(t, d, "checkout", "main")
	putDiffFixture(t, d, "main.txt", "on main\n")
	commitFixture(t, d, "main update")
	run(t, d, "merge", "--no-ff", "-m", "merge topic", "topic")
	c, err := Compare(d, "HEAD", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Files) != 2 || c.Files[0].OldPath != "note.go" || c.Files[1].Status != "Deleted" || c.Files[1].NewPath != "" {
		t.Fatalf("files %+v", c.Files)
	}
	var found bool
	for _, row := range c.Files[0].Hunks[0].Rows {
		if row.New == nil || !strings.HasPrefix(row.New.Text, "updated") {
			continue
		}
		for _, r := range row.New.Runs {
			if r.Text == "updated" && r.Tone == "comment" && r.Changed {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("multiline comment lost its context: %+v", c.Files[0].Hunks)
	}
}
