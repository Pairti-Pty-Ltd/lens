package report

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// resolvePandoc finds the Word exporter. A GUI launch does not inherit
// Homebrew's PATH, so a bare name is followed by the usual install locations.
func resolvePandoc(bin string) (string, error) {
	if bin != "" && bin != "pandoc" {
		if _, err := exec.LookPath(bin); err != nil {
			return "", &NotInstalledError{Name: "pandoc"}
		}
		return bin, nil
	}
	return PandocBin()
}

func findPandoc(lookup func(string) (string, error), extra []string) (string, error) {
	if lookup != nil {
		if p, err := lookup("pandoc"); err == nil {
			return p, nil
		}
	}
	for _, p := range extra {
		if executable(p) {
			return p, nil
		}
	}
	return "", &NotInstalledError{Name: "pandoc"}
}

// exportDocx writes a branded Word file with Pandoc, one Lua filter, and the
// brand's reference document.
func resourcePath(work, base string) string {
	if base == "" {
		return work
	}
	return work + string(os.PathListSeparator) + base
}

func exportDocx(req ExportRequest, b Brand, dir, name, title, author, date, licence string, meta map[string]string) (Result, error) {
	bin, err := resolvePandoc(req.Pandoc)
	if err != nil {
		return Result{}, err
	}
	work, err := os.MkdirTemp("", "pairti-docx-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(work)
	args := []string{"--from=markdown", "--to=docx", "--resource-path=" + resourcePath(work, req.Base)}
	if b.ID != "default" {
		if err := b.stageAsset(b.Logo, filepath.Join(work, "logo.png")); err != nil {
			return Result{}, err
		}
		if err := b.stageAsset(b.Reference, filepath.Join(work, "reference.docx")); err != nil {
			return Result{}, err
		}
		filter, err := brandFS.ReadFile("brand/docx.lua")
		if err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(filepath.Join(work, "filter.lua"), filter, 0600); err != nil {
			return Result{}, err
		}
		args = append(args, "--reference-doc=reference.docx", "--lua-filter=filter.lua")
	}
	body := stageMarkdownImages(work, req.Base, stageDiagrams(work, stripFront(req.Src)))
	if err := os.WriteFile(filepath.Join(work, "note.md"), body, 0o644); err != nil {
		return Result{}, err
	}
	out := filepath.Join(dir, name+".docx")
	fields := map[string]string{
		"brand":      b.ID,
		"lang":       "en-AU",
		"title":      title,
		"author":     author,
		"date":       date,
		"subtitle":   firstSet(meta["subtitle"], b.Subtitle),
		"client":     meta["client"],
		"project":    meta["project"],
		"licence":    licence,
		"disclosure": firstSet(meta["disclosure"], b.Disclosure),
	}
	keys := []string{"brand", "lang", "title", "author", "date", "subtitle", "client", "project", "licence", "disclosure"}
	for _, k := range keys {
		v := strings.TrimSpace(fields[k])
		if v == "" {
			continue
		}
		args = append(args, "--metadata="+k+"="+strings.ReplaceAll(v, "\n", " "))
	}
	args = append(args, "note.md", "-o", out)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), "LENS_STYLE_PREFIX="+b.StylePrefix, "LENS_TABLE_INK="+b.TableInk,
		"LENS_BODY_FONT="+b.BodyFont, "LENS_LOGO_WIDTH="+fmt.Sprintf("%dmm", b.LogoWidth), "LENS_LOGO_ALT="+b.Label+" logo")
	output, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		return Result{}, fmt.Errorf("pandoc: %s", firstLine(msg))
	}
	return Result{PDF: out, Template: b.ID + ".docx"}, nil
}

func stripFront(src []byte) []byte {
	s := string(src)
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return src
	}
	rest := s
	if strings.HasPrefix(rest, "---\r\n") {
		rest = rest[5:]
	} else {
		rest = rest[4:]
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return src
	}
	body := rest[idx+4:]
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	return []byte(body)
}
