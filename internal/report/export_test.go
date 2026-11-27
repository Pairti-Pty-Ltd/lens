package report

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeNameKeepsTheReportTitle(t *testing.T) {
	got := safeName("Field Guide - Orchard Survey Manual")
	if got != "Field Guide - Orchard Survey Manual" {
		t.Fatalf("%q", got)
	}
	if safeName("a/b:c") != "a-b-c" {
		t.Fatalf("%q", safeName("a/b:c"))
	}
	if safeName("  ") != "report" {
		t.Fatal("blank name")
	}
}

func TestExportWritesTheBrandFile(t *testing.T) {
	installExample(t, emptyBrands(t))
	dir := t.TempDir()
	bin := filepath.Join(dir, "typst")
	script := "#!/bin/sh\nmain=\nout=\nfor a in \"$@\"; do\n  case \"$a\" in *.typ) main=$a ;; *.pdf) out=$a ;; esac\ndone\n{\n  printf '%s\\n' \"$*\"\n  cat \"$main\"\n} > \"$out\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	src := []byte("---\ntitle: Coverage\nclient: West\n---\n\n# Other\n\nA note.\n")
	res, err := Export(ExportRequest{
		Root: dir, Src: src, Name: "orchard-survey-coverage", Brand: "cedar", Dir: dir, Bin: bin,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.PDF != filepath.Join(dir, "orchard-survey-coverage.pdf") {
		t.Fatalf("pdf %s", res.PDF)
	}
	b, err := os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"title=Coverage", "client=West", "Licence and Disclosure", "logo.png", "Cedar field notes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s\n%s", want, got)
		}
	}
	if strings.Contains(got, "title: Coverage") {
		t.Fatalf("front matter was copied into the body\n%s", got)
	}
}

func TestBrandedExportCompiles(t *testing.T) {
	bin, err := Bin()
	if err != nil {
		t.Skip(err)
	}
	src := []byte("---\ntitle: Coverage\nclient: West\n---\n\n# Assay\n\nA note about the question.\n")
	packs := emptyBrands(t)
	for _, id := range []string{"default", "cedar"} {
		if id == "cedar" {
			installExample(t, packs)
		}
		dir := t.TempDir()
		res, err := Export(ExportRequest{
			Root: dir, Src: src, Name: "orchard-survey-coverage", Brand: id, Dir: dir, Bin: bin,
		})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		b, err := os.ReadFile(res.PDF)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(b), "%PDF") {
			t.Fatalf("%s did not write a pdf", id)
		}
		if _, err := exec.LookPath("pdftotext"); err != nil {
			continue
		}
		text, err := exec.Command("pdftotext", "-layout", res.PDF, "-").Output()
		if err != nil {
			t.Fatalf("%s text: %v", id, err)
		}
		got := string(text)
		if id == "default" {
			if !strings.Contains(got, "Coverage") || strings.Contains(got, "Cedar field notes") {
				t.Fatal(got)
			}
			continue
		}
		if images, err := exec.Command("pdfimages", "-list", res.PDF).Output(); err != nil || !strings.Contains(string(images), "image ") {
			t.Fatalf("PDF logo: %s %v", images, err)
		}
		for _, want := range []string{"Coverage", "Client: West", "Licence and Disclosure", "Assay", "Cedar field notes", "Prepared from fictional observations."} {
			if !strings.Contains(got, want) {
				t.Fatalf("%s missing %s\n%s", id, want, got)
			}
		}
		if strings.Contains(got, "title: Coverage") {
			t.Fatalf("%s printed the front matter\n%s", id, got)
		}
		contents, err := exec.Command("pdftotext", "-f", "3", "-l", "3", "-layout", res.PDF, "-").Output()
		if err != nil {
			t.Fatalf("%s contents page: %v", id, err)
		}
		if strings.Contains(string(contents), "A note about the question") {
			t.Fatalf("%s body starts on the contents page\n%s", id, contents)
		}
		body, err := exec.Command("pdftotext", "-f", "4", "-l", "4", "-layout", res.PDF, "-").Output()
		if err != nil {
			t.Fatalf("%s body page: %v", id, err)
		}
		if !strings.Contains(string(body), "A note about the question") {
			t.Fatalf("%s body is not on its own page\n%s", id, body)
		}
	}
}
