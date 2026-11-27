package gitlog

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"pairti-lense/internal/doc"
)

// Load both revisions in two batched Git calls, avoiding a subprocess per
// line or file. Large blobs use hunk-local highlighting within the same budget.
func highlightComparison(ctx context.Context, dir string, raw []rawFile, files []FileDiff) error {
	byPath := make(map[string]rawFile, len(raw))
	for _, f := range raw {
		byPath[f.OldPath+"\x00"+f.NewPath] = f
	}
	var ids []string
	seen := map[string]bool{}
	for _, f := range files {
		if len(f.Hunks) == 0 {
			continue
		}
		entry := byPath[f.OldPath+"\x00"+f.NewPath]
		for _, id := range []string{entry.oldBlob, entry.newBlob} {
			if !seen[id] && strings.Trim(id, "0") != "" {
				ids = append(ids, id)
				seen[id] = true
			}
		}
	}
	sources, err := loadBlobs(ctx, dir, ids)
	if err != nil {
		return err
	}
	for i := range files {
		f := &files[i]
		if len(f.Hunks) == 0 {
			continue
		}
		entry := byPath[f.OldPath+"\x00"+f.NewPath]
		for side, path := range []string{f.OldPath, f.NewPath} {
			if path == "" {
				continue
			}
			id := entry.oldBlob
			if side == 1 {
				id = entry.newBlob
			}
			source, full := sources[id]
			var spans [][]doc.Span
			if full {
				spans = syntax(path, source)
			}
			for h := range f.Hunks {
				if err := ctx.Err(); err != nil {
					return err
				}
				var lines []*DiffLine
				for _, row := range f.Hunks[h].Rows {
					line := row.Old
					if side == 1 {
						line = row.New
					}
					if line != nil {
						lines = append(lines, line)
					}
				}
				if !full {
					var text []string
					for _, line := range lines {
						text = append(text, line.Text)
					}
					spans = syntax(path, strings.Join(text, "\n"))
				}
				for j, line := range lines {
					index := j
					if full {
						index = line.Number - 1
					}
					if index >= 0 && index < len(spans) {
						applySyntax(line, spans[index])
					} else {
						line.Runs = []DiffRun{{Text: line.Text}}
					}
				}
			}
		}
		for _, h := range f.Hunks {
			for _, row := range h.Rows {
				if row.Context || row.Old == nil || row.New == nil {
					continue
				}
				a, b := words(row.Old.Text), words(row.New.Text)
				changedWords(a, b)
				markWords(row.Old, a)
				markWords(row.New, b)
			}
		}
	}
	return nil
}

func syntax(path, text string) [][]doc.Span {
	if doc.IsMarkdown(path) {
		return doc.MarkdownLines(text)
	}
	return doc.TokenLines(path, "", text)
}
func applySyntax(line *DiffLine, spans []doc.Span) {
	offset := 0
	for _, sp := range spans {
		if !strings.HasPrefix(line.Text[offset:], sp.Text) {
			break
		}
		line.Runs = append(line.Runs, DiffRun{Text: sp.Text, Tone: sp.Tone, Bold: sp.Bold, Italic: sp.Italic})
		offset += len(sp.Text)
	}
	if offset != len(line.Text) {
		line.Runs = []DiffRun{{Text: line.Text}}
	}
}

func loadBlobs(ctx context.Context, dir string, ids []string) (map[string]string, error) {
	sources := map[string]string{}
	if len(ids) == 0 {
		return sources, nil
	}
	out, over, err := compareGit(ctx, dir, strings.Join(ids, "\n")+"\n", len(ids)*128, "cat-file", "--batch-check")
	if err != nil {
		return nil, err
	}
	if over {
		return nil, errors.New("invalid Git object metadata")
	}
	var selected []string
	const maxBlob = 256 * 1024
	budget := 2 * 1024 * 1024
	for record := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(record)
		if len(fields) != 3 || fields[1] != "blob" {
			continue
		}
		size, err := strconv.Atoi(fields[2])
		if err != nil || size < 0 || size > min(budget, maxBlob) {
			continue
		}
		budget -= size
		selected = append(selected, fields[0])
	}
	if len(selected) == 0 {
		return sources, nil
	}
	out, over, err = compareGit(ctx, dir, strings.Join(selected, "\n")+"\n", (2<<20)+len(selected)*128, "cat-file", "--batch")
	if err != nil {
		return nil, err
	}
	if over {
		return nil, errors.New("Git blob output exceeded expected size")
	}
	for out != "" {
		head, rest, ok := strings.Cut(out, "\n")
		fields := strings.Fields(head)
		if !ok || len(fields) != 3 {
			return nil, errors.New("invalid Git blob header")
		}
		n, err := strconv.Atoi(fields[2])
		if err != nil || n < 0 || n >= len(rest) || rest[n] != '\n' {
			return nil, errors.New("invalid Git blob length")
		}
		sources[fields[0]] = rest[:n]
		out = rest[n+1:]
	}
	return sources, nil
}
