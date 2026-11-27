package report

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const defaultTemplate = `#let title = sys.inputs.at("title", default: "Report")
#let author = sys.inputs.at("author", default: "")
#let issued = sys.inputs.at("date", default: "")
#set page(paper: "a4", margin: 2.2cm, header: align(right, text(9pt, author)))
#set text(size: 11pt)
#set par(leading: 0.7em)
#align(center)[
  #text(18pt, title)
  #if issued != "" [ #linebreak() #text(10pt, issued) ]
]
#include "body.typ"
`

// Job is one export. Empty Template uses the profile, then the built-in page.
// Out, when set, is the PDF path and the page images are not written.
type Job struct {
	brand    *Brand
	Root     string
	Src      []byte
	Bin      string
	Title    string
	Author   string
	Date     string
	Template string
	Out      string
	Inputs   map[string]string
	Base     string
}

// NotInstalledError is a missing typst binary.
type NotInstalledError struct{ Name string }

func (e *NotInstalledError) Error() string {
	if e.Name == "" {
		return "typst not installed"
	}
	return e.Name + " not installed"
}

// Result is one compile. Pages are images of that PDF, in order.
type Result struct {
	PDF      string
	Pages    []string
	Note     string
	Template string
}

// Compile writes out/report.pdf using the project's profile.
// bin is the typst executable. Empty uses the pinned compiler carried by the app.
func Compile(root string, src []byte, bin string) (Result, error) {
	p := LoadProfile(root)
	return CompileJob(Job{
		Root: root, Src: src, Bin: bin,
		Title: p.Title, Author: p.Author, Date: p.Date, Template: p.Template,
	})
}

// CompileJob writes out/report.pdf for one note and rasters that file.
func CompileJob(j Job) (Result, error) {
	if strings.TrimSpace(string(j.Src)) == "" {
		return Result{}, fmt.Errorf("open a document")
	}
	bin := j.Bin
	if bin == "" {
		found, err := Bin()
		if err != nil {
			return Result{}, err
		}
		bin = found
	} else if _, err := exec.LookPath(bin); err != nil {
		return Result{}, &NotInstalledError{Name: "typst"}
	}
	work, err := os.MkdirTemp("", "pairti-report-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(work)
	name := j.Template
	if name == "" {
		name = LoadProfile(j.Root).Template
	}
	if name == "" {
		name = "default"
	}
	// A saved profile may refer to a pack directly, without the export sheet.
	if j.brand == nil && name != "default" && resolveTemplate(j.Root, name) == "" {
		b, err := loadBrand(name)
		if err != nil {
			return Result{}, err
		}
		j.brand = &b
	}
	if j.brand != nil && j.brand.ID != "default" {
		j.Inputs = j.brand.typstInputs(j.Inputs)
		j.Author = firstSet(j.Author, j.brand.Author)
	}
	base := j.Base
	if base == "" {
		base = j.Root
	}
	main, fontDir, err := writeJob(work, j.Root, base, j.Src, name, j.brand)
	if err != nil {
		return Result{}, err
	}
	pdf := j.Out
	outDir := ""
	if pdf == "" {
		outDir = filepath.Join(j.Root, "out")
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return Result{}, err
		}
		pdf = filepath.Join(outDir, "report.pdf")
	} else if err := os.MkdirAll(filepath.Dir(pdf), 0o755); err != nil {
		return Result{}, err
	}
	args := []string{"compile"}
	if fontDir != "" {
		args = append(args, "--font-path", fontDir)
	}
	for _, field := range inputFields(j) {
		args = append(args, "--input", field.k+"="+field.v)
	}
	args = append(args, main, pdf)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = work
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return Result{}, fmt.Errorf("%s", firstLine(msg))
	}
	res := Result{PDF: pdf, Template: name}
	if j.Out != "" {
		return res, nil
	}
	pages, err := raster(pdf, outDir)
	if err != nil {
		res.Note = err.Error()
		return res, nil
	}
	res.Pages = pages
	return res, nil
}

func writeJob(work, root, base string, src []byte, name string, pack *Brand) (string, string, error) {
	body := filepath.Join(work, "body.typ")
	if err := os.WriteFile(body, []byte(writeBody(work, base, src)), 0o644); err != nil {
		return "", "", err
	}
	main := filepath.Join(work, "main.typ")
	if pack != nil && pack.ID != "default" {
		fontDir, err := stageBrand(work, *pack)
		if err != nil {
			return "", "", err
		}
		return main, fontDir, nil
	}
	text := defaultTemplate
	if name != "default" {
		path := resolveTemplate(root, name)
		if path == "" {
			return "", "", fmt.Errorf("missing project template %q: install it under templates/", name)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return "", "", err
		}
		text = string(b)
		if !strings.Contains(text, "body.typ") {
			text += "\n#include \"body.typ\"\n"
		}
	}
	if err := os.WriteFile(main, []byte(text), 0o644); err != nil {
		return "", "", err
	}
	return main, "", nil
}

func inputFields(j Job) []struct{ k, v string } {
	vals := map[string]string{}
	put := func(k, v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		v = strings.ReplaceAll(v, "\n", " ")
		vals[k] = v
	}
	put("title", j.Title)
	put("author", j.Author)
	put("date", j.Date)
	for k, v := range j.Inputs {
		put(k, v)
	}
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]struct{ k, v string }, 0, len(keys))
	for _, k := range keys {
		out = append(out, struct{ k, v string }{k, vals[k]})
	}
	return out
}

func raster(pdf, dir string) ([]string, error) {
	bin, err := exec.LookPath("pdftoppm")
	if err != nil {
		return nil, fmt.Errorf("pdftoppm not installed")
	}
	prefix := filepath.Join(dir, "page")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-png", "-r", "144", pdf, prefix)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", firstLine(msg))
	}
	matches, err := filepath.Glob(prefix + "-*.png")
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no pages")
	}
	return matches, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
