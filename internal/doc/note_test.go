package doc

import (
	"path/filepath"
	"strings"
	"testing"
)

const compliance = "" +
	"---\n" +
	"title: Orchard Survey and Sampling Plan\n" +
	"type: topic\n" +
	"sources: [doc:///research/sample-orchard-field-survey/PROPOSAL.md, doc:///research/sample-orchard-field-survey/APPENDIX.md]\n" +
	"tags:\n" +
	"  - soc2\n" +
	"  - iso\n" +
	"---\n" +
	"\n" +
	"# Orchard Survey and Sampling Plan\n" +
	"\n" +
	"The orchard survey covers six sampling sites.[^1]\n" +
	"\n" +
	"A path in a code span stays a path: `doc:///research/sample-orchard-field-survey/PROPOSAL.md`\n" +
	"\n" +
	"See doc:///research/sample-orchard-field-survey/PROPOSAL.md for the proposal.\n" +
	"\n" +
	"[^1]: Count the trees before recording the plot area.\n"

func TestNoteTagsComeFromTheFrontMatter(t *testing.T) {
	got := Tags([]byte(compliance))
	want := map[string]bool{"topic": false, "soc2": false, "iso": false}
	for _, tag := range got {
		if _, ok := want[tag]; ok {
			want[tag] = true
		}
		if tag == "Orchard Survey and Sampling Plan" || strings.Contains(tag, "PROPOSAL") {
			t.Fatalf("tag %q", tag)
		}
	}
	for tag, ok := range want {
		if !ok {
			t.Fatalf("missing %s in %v", tag, got)
		}
	}
}

func TestDocLinkOpensUnderTheProjectDocFolder(t *testing.T) {
	const uri = "doc:///research/sample-orchard-field-survey/PROPOSAL.md"
	root := filepath.Join(string(filepath.Separator), "Users", "writer", "src", "grove-p-workspace")
	got, ok := DocFile(root, uri)
	want := filepath.Join(root, "doc", "research", "sample-orchard-field-survey", "PROPOSAL.md")
	if !ok || got != want {
		t.Fatalf("got %q ok %v", got, ok)
	}
	if _, ok := DocFile(root, "doc:///../library.json"); ok {
		t.Fatal("a doc link climbed out of doc/")
	}
	blocks := Blocks([]byte(compliance))
	var linked, coded bool
	var note string
	for _, b := range blocks {
		if strings.Contains(b.Text, "[^") {
			t.Fatalf("raw footnote in %s: %q", b.Kind, b.Text)
		}
		for _, sp := range b.Spans {
			if sp.Code && sp.Text == uri {
				coded = sp.Link == ""
			}
			if !sp.Code && sp.Link == uri {
				linked = true
			}
			if sp.Note {
				note = sp.Text
				if !strings.HasPrefix(sp.Link, "#line:") {
					t.Fatalf("footnote link %q", sp.Link)
				}
			}
		}
		if b.Kind == "fn" && !strings.Contains(b.Text, "Count the trees before recording the plot area.") {
			t.Fatalf("footnote %q", b.Text)
		}
	}
	if !linked || !coded || note != "1" {
		t.Fatalf("linked %v coded %v note %q", linked, coded, note)
	}
}
