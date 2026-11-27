package gitlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Comparison holds the hunks displayed in History. Missing sides are nil.
type Comparison struct {
	Files  []FileDiff
	Notice string
}
type FileDiff struct {
	OldPath, NewPath, Status string
	Hunks                    []Hunk
}
type Hunk struct {
	Header string
	Rows   []DiffRow
}
type DiffRow struct {
	Old, New *DiffLine
	Context  bool
}
type DiffLine struct {
	Number    int
	Text      string
	Runs      []DiffRun
	NoNewline bool
}
type DiffRun struct {
	Text, Tone            string
	Bold, Italic, Changed bool
}

type rawFile struct {
	FileDiff
	oldBlob, newBlob string
	oldMode, newMode string
}

const maxPatchBytes = 512 * 1024
const maxPatchRows = 8000

// Compare reads the selected commit against its first parent. An empty rel
// selects all paths; a file selection includes both sides of a rename.
func Compare(dir, hash, rel string) (Comparison, error) {
	return CompareContext(context.Background(), dir, hash, rel)
}

func CompareContext(parent context.Context, dir, hash, rel string) (Comparison, error) {
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()
	args := []string{"--literal-pathspecs", "-c", "diff.suppressBlankEmpty=false", "show", "--format=", "--no-notes", "--root", "--diff-merges=first-parent", "--find-renames", "--no-ext-diff", "--no-textconv", "--no-color", "--no-abbrev", "--raw", "-z"}
	if strings.HasPrefix(hash, "-") || hash == "" {
		return Comparison{}, errors.New("invalid commit")
	}
	var paths []string
	if rel != "" {
		// Path filtering before rename detection would turn a rename into an add.
		out, over, err := compareGit(ctx, dir, "", 4<<20, append(args, hash, "--")...)
		if err != nil {
			return Comparison{}, err
		}
		if over {
			return Comparison{}, errors.New("changed file list is too large")
		}
		files, err := rawFiles(out)
		if err != nil {
			return Comparison{}, err
		}
		for _, f := range files {
			if f.OldPath != rel && f.NewPath != rel {
				continue
			}
			if f.OldPath != "" {
				paths = append(paths, f.OldPath)
			}
			if f.NewPath != "" && f.NewPath != f.OldPath {
				paths = append(paths, f.NewPath)
			}
		}
		if len(paths) == 0 {
			return Comparison{Notice: "This file has no changes in this commit."}, nil
		}
	}
	args = append(args, "--patch", "--unified=3", "--output-indicator-new=+", "--output-indicator-old=-", "--output-indicator-context= ", hash, "--")
	out, over, err := compareGit(ctx, dir, "", maxPatchBytes, append(args, paths...)...)
	if err != nil {
		return Comparison{}, err
	}
	var result Comparison
	if over {
		result.Notice = "Large comparison. Showing the first part of the patch."
	}
	raw, patch, ok := strings.Cut(out, "\x00\x00")
	if !ok {
		if over {
			return result, nil
		}
		if out == "" {
			result.Notice = "This commit contains no file changes."
			return result, nil
		}
		return Comparison{}, errors.New("unexpected Git patch header")
	}
	files, err := rawFiles(raw)
	if err != nil {
		return Comparison{}, err
	}
	if over {
		if end := strings.LastIndexByte(patch, '\n'); end >= 0 {
			patch = patch[:end+1]
		} else {
			patch = ""
		}
	}
	sections := strings.Split(strings.TrimPrefix(patch, "diff --git "), "\ndiff --git ")
	rowsLeft := maxPatchRows
	section := 0
	for i := range files {
		f := &files[i]
		if section >= len(sections) || rowsLeft <= 0 {
			result.Notice = "Large comparison. Showing the first part of the patch."
			break
		}
		if err := ctx.Err(); err != nil {
			return Comparison{}, err
		}
		// Git emits a delete and an add patch for a change of object type.
		count := 1
		if strings.HasPrefix(f.Status, "Type changed") && section+1 < len(sections) {
			count = 2
		}
		for j := 0; j < count; j++ {
			parsePatch(f, sections[section], &rowsLeft)
			section++
		}
		if rel != "" && f.OldPath != rel && f.NewPath != rel {
			continue
		}
		result.Files = append(result.Files, f.FileDiff)
	}
	if rowsLeft < 0 {
		result.Notice = "Large comparison. Showing the first part of the patch."
	}
	if err := highlightComparison(ctx, dir, files, result.Files); err != nil {
		return Comparison{}, err
	}
	return result, nil
}

// limitedOutput retains only the prefix while draining the child process.
type limitedOutput struct {
	buffer bytes.Buffer
	limit  int
	over   bool
}

func (w *limitedOutput) Write(p []byte) (int, error) {
	n := min(len(p), w.limit-w.buffer.Len())
	w.buffer.Write(p[:n])
	w.over = w.over || n < len(p)
	return len(p), nil
}
func compareGit(ctx context.Context, dir, input string, limit int, args ...string) (string, bool, error) {
	if dir == "" {
		return "", false, ErrNone
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(input)
	out := &limitedOutput{limit: limit}
	stderr := &limitedOutput{limit: 4096}
	cmd.Stdout, cmd.Stderr = out, stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", false, ctx.Err()
		}
		return "", false, fmt.Errorf("git comparison: %w: %s", err, strings.TrimSpace(stderr.buffer.String()))
	}
	return out.buffer.String(), out.over, nil
}
func rawFiles(raw string) ([]rawFile, error) {
	items := strings.Split(strings.TrimSuffix(raw, "\x00"), "\x00")
	var files []rawFile
	for i := 0; i < len(items) && items[i] != ""; {
		fields := strings.Fields(strings.TrimPrefix(items[i], ":"))
		if len(fields) != 5 || len(fields[4]) == 0 || i+1 >= len(items) {
			return nil, errors.New("invalid Git file record")
		}
		f := rawFile{oldMode: fields[0], newMode: fields[1], oldBlob: fields[2], newBlob: fields[3]}
		f.OldPath, f.NewPath = items[i+1], items[i+1]
		i += 2
		switch fields[4][0] {
		case 'A':
			f.OldPath = ""
			f.Status = "Added"
		case 'D':
			f.NewPath = ""
			f.Status = "Deleted"
		case 'R', 'C':
			if i >= len(items) {
				return nil, errors.New("missing renamed path")
			}
			f.NewPath = items[i]
			i++
			f.Status = "Renamed"
			if fields[4][0] == 'C' {
				f.Status = "Copied"
			}
		case 'T':
			f.Status = "Type changed"
		default:
			f.Status = "Modified"
		}
		if f.oldMode != f.newMode && f.OldPath != "" && f.NewPath != "" {
			f.Status += " · " + f.oldMode + " → " + f.newMode
		}
		files = append(files, f)
	}
	return files, nil
}

var hunkHead = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

func parsePatch(f *rawFile, patch string, left *int) {
	var h *Hunk
	var minus, plus []*DiffLine
	var old, new int
	var lastOld, lastNew *DiffLine
	flush := func() {
		if h != nil {
			h.Rows = append(h.Rows, alignChanges(minus, plus)...)
		}
		minus = nil
		plus = nil
	}
	for line := range strings.SplitSeq(patch, "\n") {
		if m := hunkHead.FindStringSubmatch(line); m != nil {
			flush()
			old, _ = strconv.Atoi(m[1])
			new, _ = strconv.Atoi(m[2])
			f.Hunks = append(f.Hunks, Hunk{Header: line})
			h = &f.Hunks[len(f.Hunks)-1]
			continue
		}
		if h == nil {
			if strings.HasPrefix(line, "Binary files ") || line == "GIT binary patch" {
				f.Status += " · Binary file"
			}
			continue
		}
		if strings.HasPrefix(line, "\\ No newline") {
			if lastOld != nil {
				lastOld.NoNewline = true
			}
			if lastNew != nil {
				lastNew.NoNewline = true
			}
			continue
		}
		if len(line) == 0 {
			continue
		}
		if *left <= 0 {
			*left = -1
			break
		}
		*left--
		text := strings.ToValidUTF8(line[1:], "�")
		lastOld, lastNew = nil, nil
		switch line[0] {
		case ' ':
			flush()
			lastOld = &DiffLine{Number: old, Text: text}
			lastNew = &DiffLine{Number: new, Text: text}
			old++
			new++
			h.Rows = append(h.Rows, DiffRow{Old: lastOld, New: lastNew, Context: true})
		case '-':
			lastOld = &DiffLine{Number: old, Text: text}
			old++
			minus = append(minus, lastOld)
		case '+':
			lastNew = &DiffLine{Number: new, Text: text}
			new++
			plus = append(plus, lastNew)
		}
	}
	flush()
}
