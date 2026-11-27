package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileWritesThePDF(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "typst")
	script := "#!/bin/sh\nout=\nfor a in \"$@\"; do\n  case \"$a\" in *.pdf) out=$a ;; esac\ndone\nprintf '%%PDF-1.1 report\\n' > \"$out\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Compile(root, []byte("# Assay\n\nA note.\n"), bin)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "report") {
		t.Fatalf("pdf %q", b)
	}
	if _, err := os.Stat(filepath.Join(root, "out", "report.pdf")); err != nil {
		t.Fatal(err)
	}
}

func TestCompileEmbedsAMermaidDiagram(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "typst")
	script := "#!/bin/sh\nout=\nfor a in \"$@\"; do\n  case \"$a\" in *.pdf) out=$a ;; esac\ndone\ncat body.typ > \"$out\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := CompileJob(Job{
		Root: root, Src: []byte("# Note\n\n```mermaid\ngraph TD\nA-->B\n```\n\nAfter.\n"),
		Bin: bin, Out: filepath.Join(root, "out.pdf"), Template: "default",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if strings.Contains(got, "A-->B") {
		t.Fatalf("fence kept:\n%s", got)
	}
	if !strings.Contains(got, `#image("diag-1.png"`) || !strings.Contains(got, "After.") {
		t.Fatalf("diagram missing:\n%s", got)
	}
}

func TestCompileNamesAMissingCompiler(t *testing.T) {
	_, err := Compile(t.TempDir(), []byte("# A\n"), filepath.Join(t.TempDir(), "missing-typst"))
	if err == nil {
		t.Fatal("expected an error")
	}
	if _, ok := err.(*NotInstalledError); !ok {
		t.Fatalf("err %v", err)
	}
}

func TestSwapTemplateChangesTheCompile(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "brief.typ"), []byte("#set text(size: 9pt)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "templates", "memo.typ"), []byte("#set text(size: 14pt)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lens.toml"), []byte("title = \"Assay\"\nauthor = \"Pairti\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "typst")
	script := "#!/bin/sh\nmain=\nout=\nfor a in \"$@\"; do\n  case \"$a\" in *.typ) main=$a ;; *.pdf) out=$a ;; esac\ndone\n{\n  printf '%s\\n' \"$*\"\n  cat \"$main\"\n} > \"$out\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SetTemplate(root, "brief.typ"); err != nil {
		t.Fatal(err)
	}
	res, err := CompileJob(Job{
		Root: root, Src: []byte("# Assay\n\nA note.\n"), Bin: bin,
		Title: "Assay", Author: "Pairti", Date: "2026-11-22", Template: "brief.typ",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, "9pt") || !strings.Contains(got, "title=Assay") || !strings.Contains(got, "author=Pairti") || !strings.Contains(got, "date=2026-11-22") {
		t.Fatalf("brief compile:\n%s", got)
	}
	if res.Template != "brief.typ" {
		t.Fatalf("template %q", res.Template)
	}
	if err := SetTemplate(root, "memo.typ"); err != nil {
		t.Fatal(err)
	}
	res, err = Compile(root, []byte("# Assay\n\nA note.\n"), bin)
	if err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "14pt") || strings.Contains(string(b), "9pt") {
		t.Fatalf("memo compile:\n%s", b)
	}
}

func TestProjectTemplateIsUsed(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "lens.toml"), []byte("template = \"brief.typ\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "brief.typ"), []byte("#set text(size: 10pt)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "typst")
	script := "#!/bin/sh\nmain=\nout=\nfor a in \"$@\"; do\n  case \"$a\" in *.typ) main=$a ;; *.pdf) out=$a ;; esac\ndone\ncp \"$main\" \"$out\"\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	res, err := Compile(root, []byte("# Title\n"), bin)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(res.PDF)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "10pt") || !strings.Contains(string(b), "body.typ") {
		t.Fatalf("template was not the job:\n%s", b)
	}
}
