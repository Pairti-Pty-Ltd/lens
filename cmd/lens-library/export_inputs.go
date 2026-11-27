package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/project"
	"pairti-lense/internal/report"
)

// The server publishes reports from its configured library root. Project
// loaders run against a private snapshot; they never receive a request path.
func reportInput(name string) bool {
	return doc.IsMarkdown(name) || report.SnapshotImagePath(name) == name || name == ".pairti-reports.json" || name == ".pairti-project.json" || name == "lens.toml"
}

func (l *library) readReportInput(name string) ([]byte, error) {
	before, err := l.root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("unavailable report input %q", name)
	}
	f, err := l.root.Open(name)
	if err != nil {
		return nil, fmt.Errorf("cannot open report input %q", name)
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) || !l.regular(name) {
		return nil, fmt.Errorf("changed report input %q", name)
	}
	if report.SnapshotImagePath(name) == name {
		const maxImageBytes = 8 * doc.MaxBytes
		b, err := io.ReadAll(io.LimitReader(f, maxImageBytes+1))
		if err != nil || len(b) > maxImageBytes {
			return nil, fmt.Errorf("report image %q exceeds 8 MiB or is unreadable", name)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
		if err != nil || int64(cfg.Width)*int64(cfg.Height) > 32_000_000 {
			return nil, fmt.Errorf("invalid or oversized report image %q", name)
		}
		return b, nil
	}
	b, err := io.ReadAll(io.LimitReader(f, doc.MaxBytes+1))
	if err != nil || len(b) > doc.MaxBytes || !utf8.Valid(b) {
		return nil, fmt.Errorf("report input %q must be UTF-8 and at most 1 MiB", name)
	}
	return b, nil
}

type reportSnapshot struct {
	book    project.Book
	tracked map[string]bool
	dir     string
}

func (l *library) reportSnapshot(ctx context.Context, dir string) (reportSnapshot, error) {
	files, err := l.tracked(ctx, "", reportInput)
	if err != nil {
		return reportSnapshot{}, errors.New("cannot read the Git index")
	}
	snap := reportSnapshot{dir: dir, tracked: make(map[string]bool, len(files))}
	for _, name := range files {
		snap.tracked[name] = true
		if doc.IsMarkdown(name) || report.SnapshotImagePath(name) == name {
			continue
		}
		b, err := l.readReportInput(name)
		if err != nil {
			return snap, err
		}
		if err = os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			return snap, errors.New("cannot stage project settings")
		}
	}
	snap.book, err = project.LoadBook(dir)
	if err != nil {
		return snap, errors.New("cannot read saved reports")
	}
	return snap, nil
}

func (l *library) reportSource(s reportSnapshot, id string) (project.Report, []byte, error) {
	s.book.Open = id
	choice, ok := s.book.Current()
	if !ok {
		return project.Report{}, nil, errors.New("unknown report")
	}
	// Keep job sizes bounded even if a report repeats the same file many times.
	const maxReportBytes = 16 * doc.MaxBytes
	total := 0
	for _, name := range choice.Files {
		if !s.tracked[name] || !doc.IsMarkdown(name) {
			return choice, nil, fmt.Errorf("report requires a tracked Markdown file: %q", name)
		}
		b, err := l.readReportInput(name)
		if err != nil {
			return choice, nil, err
		}
		total += len(b) + 2
		if total > maxReportBytes {
			return choice, nil, errors.New("report exceeds 16 MiB")
		}
		path := filepath.Join(s.dir, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return choice, nil, errors.New("cannot stage report folder")
		}
		if err = os.WriteFile(path, b, 0600); err != nil {
			return choice, nil, errors.New("cannot stage report file")
		}
	}
	src, err := project.Join(s.dir, choice.Files)
	if err != nil {
		return choice, nil, err
	}
	seen := map[string]bool{}
	for _, block := range doc.Blocks(src) {
		for _, span := range block.Spans {
			if !span.Image {
				continue
			}
			name := report.SnapshotImagePath(span.Link)
			if name == "" || !s.tracked[name] {
				return choice, nil, errors.New("report images must be tracked PNG, JPEG, or GIF files inside the library")
			}
			if seen[name] {
				continue
			}
			seen[name] = true
			b, err := l.readReportInput(name)
			if err != nil {
				return choice, nil, err
			}
			total += len(b)
			if total > maxReportBytes {
				return choice, nil, errors.New("report exceeds 16 MiB")
			}
			path := filepath.Join(s.dir, filepath.FromSlash(name))
			if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				return choice, nil, errors.New("cannot stage report image folder")
			}
			if err = os.WriteFile(path, b, 0600); err != nil {
				return choice, nil, errors.New("cannot stage report image")
			}
		}
	}
	return choice, src, nil
}

func (l *library) reportList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	dir, err := os.MkdirTemp("", "lens-report-list-")
	if err != nil {
		http.Error(w, "Cannot create report snapshot.", 500)
		return
	}
	defer os.RemoveAll(dir)
	snap, err := l.reportSnapshot(r.Context(), dir)
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Book    project.Book    `json:"book"`
		Outputs []report.Output `json:"outputs"`
	}{snap.book, report.Outputs()})
}

// Export jobs use independent scratch directories. One concurrent conversion
// per library keeps multiple writers from exhausting the server's memory.
func (l *library) exportReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !l.exportJob.TryLock() {
		http.Error(w, "Another report is exporting. Try again when it finishes.", http.StatusTooManyRequests)
		return
	}
	defer l.exportJob.Unlock()
	dir, err := os.MkdirTemp("", "lens-report-")
	if err != nil {
		http.Error(w, "Cannot create report snapshot.", 500)
		return
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "inputs")
	if err = os.Mkdir(input, 0700); err != nil {
		http.Error(w, "Cannot create report snapshot.", 500)
		return
	}
	snap, err := l.reportSnapshot(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	choice, src, err := l.reportSource(snap, r.URL.Query().Get("report"))
	if err != nil {
		http.Error(w, err.Error(), 422)
		return
	}
	output := r.URL.Query().Get("output")
	if output == "" {
		output = choice.Template
		if output == "" {
			output = "default.pdf"
		} else if !strings.HasSuffix(output, ".docx") && !strings.HasSuffix(output, ".pdf") {
			output += ".pdf"
		}
	}
	available := false
	for _, installed := range report.Outputs() {
		if installed.ID == output {
			available = true
			break
		}
	}
	if !available {
		http.Error(w, "Selected report template is unavailable. Install or repair its brand pack on the library server.", 422)
		return
	}
	// Converters have a 60-second process timeout.
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(90 * time.Second))
	result, err := report.Export(report.ExportRequest{LibrarySnapshot: true, Root: input, Base: input, Src: src, Name: choice.Name, Brand: output, Dir: filepath.Join(dir, "output")})
	if err != nil {
		var missing *report.NotInstalledError
		if errors.As(err, &missing) {
			http.Error(w, missing.Name+" is not installed on the library server.", 503)
			return
		}
		http.Error(w, "Export failed. Check the report and the selected template on the library server.", 422)
		return
	}
	f, err := os.Open(result.PDF)
	if err != nil {
		http.Error(w, "Could not read the export.", 500)
		return
	}
	defer f.Close()
	name := filepath.Base(result.PDF)
	if strings.HasSuffix(output, ".docx") {
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	} else {
		w.Header().Set("Content-Type", "application/pdf")
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	http.ServeContent(w, r, name, time.Time{}, f)
}
