package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const reportsName = ".pairti-reports.json"

// Report is one ordered publication. Files are relative to the library folder.
// A file is included only when it is listed.
type Report struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Template string   `json:"template,omitempty"`
	Files    []string `json:"files"`
}

// Book is the reports in one library folder.
type Book struct {
	Open    string   `json:"open,omitempty"`
	Reports []Report `json:"reports"`
}

// LoadBook reads the reports. A missing file is an empty book.
// An older .pairti-project.json contributes its saved list only, not a directory scan.
func LoadBook(root string) (Book, error) {
	b, err := os.ReadFile(filepath.Join(root, reportsName))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Book{}, err
		}
		return migrated(root), nil
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return Book{}, nil
	}
	var book Book
	if err := json.Unmarshal(b, &book); err != nil {
		return Book{}, err
	}
	if book.Reports == nil {
		book.Reports = []Report{}
	}
	return book, nil
}

func migrated(root string) Book {
	st, err := Load(root)
	if err != nil || len(st.Files) == 0 {
		return Book{}
	}
	return Book{
		Open: "report-1",
		Reports: []Report{{
			ID: "report-1", Name: "Report", Template: st.Template, Files: st.Files,
		}},
	}
}

// SaveBook writes the reports file.
func SaveBook(root string, book Book) error {
	clean := Book{Open: book.Open}
	for _, r := range book.Reports {
		r.Name = strings.TrimSpace(r.Name)
		r.ID = strings.TrimSpace(r.ID)
		if r.ID == "" || r.Name == "" {
			continue
		}
		var files []string
		seen := map[string]bool{}
		for _, rel := range r.Files {
			rel = cleanRel(rel)
			if rel == "" || seen[rel] {
				continue
			}
			seen[rel] = true
			files = append(files, rel)
		}
		r.Files = files
		r.Template = strings.TrimSpace(r.Template)
		clean.Reports = append(clean.Reports, r)
	}
	b, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(filepath.Join(root, reportsName), b, 0o644)
}

// Add creates an empty report and makes it the open one.
func (b *Book) Add() Report {
	n := 1
	for {
		id := fmt.Sprintf("report-%d", n)
		if b.byID(id) == nil {
			name := "Report"
			if n > 1 {
				name = fmt.Sprintf("Report %d", n)
			}
			r := Report{ID: id, Name: name}
			b.Reports = append(b.Reports, r)
			b.Open = id
			return r
		}
		n++
	}
}

// AddFile appends rel to the open report. A repeat is ignored.
func (b *Book) AddFile(rel string) bool {
	r := b.byID(b.Open)
	if r == nil {
		return false
	}
	rel = cleanRel(rel)
	if rel == "" {
		return false
	}
	for _, have := range r.Files {
		if have == rel {
			return false
		}
	}
	r.Files = append(r.Files, rel)
	return true
}

// Rename sets the open label. An empty name is refused.
func (b *Book) Rename(id, name string) bool {
	r := b.byID(id)
	name = strings.TrimSpace(name)
	if r == nil || name == "" {
		return false
	}
	r.Name = name
	return true
}

// Delete drops the report. The notes stay on disk.
func (b *Book) Delete(id string) {
	var keep []Report
	for _, r := range b.Reports {
		if r.ID != id {
			keep = append(keep, r)
		}
	}
	b.Reports = keep
	if b.Open == id {
		b.Open = ""
	}
}

// RemoveFile drops rel from one report.
func (b *Book) RemoveFile(id, rel string) {
	r := b.byID(id)
	if r == nil {
		return
	}
	rel = cleanRel(rel)
	var keep []string
	for _, have := range r.Files {
		if have != rel {
			keep = append(keep, have)
		}
	}
	r.Files = keep
}

// Move shifts rel up (dir -1) or down (dir 1) in one report.
func (b *Book) Move(id, rel string, dir int) {
	r := b.byID(id)
	if r == nil || dir == 0 {
		return
	}
	rel = cleanRel(rel)
	i := -1
	for n, have := range r.Files {
		if have == rel {
			i = n
			break
		}
	}
	j := i + dir
	if i < 0 || j < 0 || j >= len(r.Files) {
		return
	}
	r.Files[i], r.Files[j] = r.Files[j], r.Files[i]
}

// Current is the open report.
func (b *Book) Current() (Report, bool) {
	r := b.byID(b.Open)
	if r == nil {
		return Report{}, false
	}
	return *r, true
}

func (b *Book) byID(id string) *Report {
	for i := range b.Reports {
		if b.Reports[i].ID == id {
			return &b.Reports[i]
		}
	}
	return nil
}
