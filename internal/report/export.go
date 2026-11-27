package report

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"pairti-lense/internal/doc"
)

// ExportRequest is one branded export. Target is "pdf" or "docx".
// An empty Target is taken from a Brand suffix such as "studio.docx".
// An empty Dir writes a temporary directory.
type ExportRequest struct {
	// LibrarySnapshot confines rendering to staged report inputs and uses only installed tools.
	LibrarySnapshot bool
	Root            string
	Src             []byte
	Name            string
	Brand           string
	Target          string
	Dir             string
	Bin             string
	Pandoc          string
	Base            string
}

// Export compiles the note with a brand template and writes Dir/Name.pdf.
func Export(req ExportRequest) (Result, error) {
	brand, target := splitTarget(req.Brand, req.Target)
	b, err := loadBrand(brand)
	if err != nil {
		return Result{}, err
	}
	if req.LibrarySnapshot {
		if target == "docx" {
			req.Pandoc, err = pinnedPandoc()
			if err != nil {
				req.Pandoc, err = exec.LookPath("pandoc")
			}
		} else {
			req.Bin, err = locateTypst(bundledDirs(), exec.LookPath)
		}
		if err != nil {
			tool := "typst"
			if target == "docx" {
				tool = "pandoc"
			}
			return Result{}, &NotInstalledError{Name: tool}
		}
	}
	req.Brand = b.ID
	req.Target = target
	dir := req.Dir
	if dir == "" {
		made, err := os.MkdirTemp("", "pairti-export-")
		if err != nil {
			return Result{}, err
		}
		dir = made
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, err
	}
	name := safeName(req.Name)
	meta := doc.Meta(req.Src)
	prof := Profile{}
	if req.Root != "" {
		prof = LoadProfile(req.Root)
	}
	title := firstSet(meta["title"], prof.Title, doc.Title(req.Src), strings.ReplaceAll(name, "-", " "))
	author := firstSet(meta["author"], prof.Author, b.Author)
	date := firstSet(meta["date"], prof.Date)
	if date == "" {
		date = today()
	}
	licence := firstSet(meta["licence"], meta["license"], b.Licence)
	if req.Target == "docx" {
		return exportDocx(req, b, dir, name, title, author, date, licence, meta)
	}
	base := req.Base
	if base == "" {
		base = req.Root
	}
	return CompileJob(Job{
		Root: req.Root, Src: req.Src, Bin: req.Bin, Base: base,
		Title: title, Author: author, Date: date, Template: b.ID,
		Out:             filepath.Join(dir, name+".pdf"),
		brand:           &b,
		librarySnapshot: req.LibrarySnapshot,
		Inputs: map[string]string{
			"subtitle": meta["subtitle"], "client": meta["client"], "project": meta["project"],
			"licence": licence, "disclosure": meta["disclosure"],
		},
	})
}

func splitTarget(brand, target string) (string, string) {
	target = strings.TrimSpace(target)
	brand = strings.TrimSpace(brand)
	if strings.HasSuffix(brand, ".docx") {
		return strings.TrimSuffix(brand, ".docx"), "docx"
	}
	if strings.HasSuffix(brand, ".pdf") {
		brand = strings.TrimSuffix(brand, ".pdf")
		if target == "" {
			target = "pdf"
		}
	}
	if target == "" {
		target = "pdf"
	}
	return brand, target
}

// Output is one row in the export menu.
type Output struct {
	ID    string
	Label string
}

// Outputs is each brand as a PDF and as a Word file.
func Outputs() []Output {
	var out []Output
	for _, b := range Brands() {
		name := strings.TrimSuffix(b.Label, " (pdf)")
		out = append(out,
			Output{ID: b.ID + ".pdf", Label: name + " (pdf)"},
			Output{ID: b.ID + ".docx", Label: name + " (docx)"},
		)
	}
	return out
}

func firstSet(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func safeName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|', 0:
			b.WriteByte('-')
		default:
			if r < 32 {
				b.WriteByte('-')
				continue
			}
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), " .")
	if out == "" {
		return "report"
	}
	return out
}

func today() string {
	return time.Now().Format("2006-01-02")
}

// Open asks the operating system to open the file. A Word file opens in
// Microsoft Word when that application is installed.
func Open(path string) error {
	if runtime.GOOS == "darwin" && strings.EqualFold(filepath.Ext(path), ".docx") {
		err := exec.Command("open", "-a", "Microsoft Word", path).Run()
		if err == nil {
			return nil
		}
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		if strings.EqualFold(filepath.Ext(path), ".docx") {
			cmd = exec.Command("cmd", "/c", "start", "", "winword", path)
		} else {
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
		}
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Run()
}
