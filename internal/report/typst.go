package report

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// typstVersion is the compiler the app carries. Page breaks are part of the
// brand, so a Typst found on PATH is not the one export uses once this copy exists.
const typstVersion = "0.15.1"

// Digests are the sha256 fields published on the v0.15.1 GitHub release.
var typstAssets = map[string]struct {
	file string
	sha  string
}{
	"darwin/arm64":  {"typst-aarch64-apple-darwin.tar.xz", "48f62ed034aa3a7978309579ac6ca00045e2ef0da73114e8af27cfd8e74dc05a"},
	"darwin/amd64":  {"typst-x86_64-apple-darwin.tar.xz", "7f9fdd9584866245de9a79e0add8f9236fae6f40a8a45e2c4771ccc14db4e0fa"},
	"linux/arm64":   {"typst-aarch64-unknown-linux-musl.tar.xz", "5aa8d74a3d906e60ea12a66ac2f37f8eef1b14cbad7182a745e393a10c23dcee"},
	"linux/amd64":   {"typst-x86_64-unknown-linux-musl.tar.xz", "a6d077d0a95eed5a2eba715b2dae06be954f624ccbf85758a03f389ded33118c"},
	"windows/arm64": {"typst-aarch64-pc-windows-msvc.zip", "4ab28e1b71ec3184d38d580ab797f499b6770d952b6b19167be5cea5c2662e14"},
	"windows/amd64": {"typst-x86_64-pc-windows-msvc.zip", "19ce3551153c2fe7ee9fa2f95208310c8f4d3209fedb699e0333faf8913f6736"},
}

var typstMu sync.Mutex

// Bin is the Typst executable export runs. A copy beside the app or in the
// versioned config directory wins. The pinned release is downloaded when both
// are missing. PATH is used only if that download fails.
func Bin() (string, error) {
	if p, err := locateTypst(bundledDirs(), nil); err == nil {
		return p, nil
	}
	typstMu.Lock()
	defer typstMu.Unlock()
	if p, err := locateTypst(bundledDirs(), nil); err == nil {
		return p, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	p, installErr := fetchPinned(ctx)
	if installErr == nil {
		return p, nil
	}
	if found, err := exec.LookPath(typstName()); err == nil {
		return found, nil
	}
	return "", installErr
}

func executableDirs(appOnly bool) []string {
	var dirs []string
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dir := filepath.Dir(exe)
		parent := filepath.Dir(dir)
		if filepath.Base(dir) == "MacOS" && filepath.Base(parent) == "Contents" {
			dirs = append(dirs, dir, filepath.Join(parent, "Resources"))
		} else if !appOnly {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func bundledDirs() []string {
	dirs := executableDirs(false)
	if dir, err := installDir(); err == nil {
		dirs = append(dirs, dir)
	}
	return dirs
}

func installDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "pairti-lense", "typst", typstVersion), nil
}

func typstName() string {
	if runtime.GOOS == "windows" {
		return "typst.exe"
	}
	return "typst"
}

func locateTypst(dirs []string, lookup func(string) (string, error)) (string, error) {
	return locateTool(typstName(), dirs, lookup)
}

func locateTool(name string, dirs []string, lookup func(string) (string, error)) (string, error) {
	for _, dir := range dirs {
		p := filepath.Join(dir, name)
		if executable(p) {
			return p, nil
		}
	}
	if lookup != nil {
		if p, err := lookup(name); err == nil {
			return p, nil
		}
	}
	return "", &NotInstalledError{Name: name}
}

func executable(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return st.Mode()&0o111 != 0
}

func fetchPinned(ctx context.Context) (string, error) {
	dir, err := installDir()
	if err != nil {
		return "", err
	}
	spec, err := typstSpec()
	if err != nil {
		return "", err
	}
	spec.Dest = dir
	return fetchBinary(ctx, spec)
}

func typstSpec() (archiveSpec, error) {
	key := runtime.GOOS + "/" + runtime.GOARCH
	asset, ok := typstAssets[key]
	if !ok {
		return archiveSpec{}, fmt.Errorf("typst %s: no binary for %s", typstVersion, key)
	}
	return archiveSpec{URL: "https://github.com/typst/typst/releases/download/v" + typstVersion + "/" + asset.file,
		SHA: asset.sha, File: asset.file, Label: "typst", Version: typstVersion, Names: []string{typstName()}, Bin: typstName()}, nil
}

// archiveSpec is one pinned compiler download.
type archiveSpec struct {
	URL     string
	SHA     string
	File    string
	Dest    string
	Label   string
	Version string
	Names   []string
	Bin     string
}

func fetchBinary(ctx context.Context, spec archiveSpec) (string, error) {
	if err := os.MkdirAll(spec.Dest, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(spec.Dest, spec.Bin)
	if executable(dest) {
		return dest, nil
	}
	if spec.Label == "typst" {
		SetTransfer("Downloading Typst " + spec.Version)
	}
	got, err := fetchArchive(ctx, spec, dest)
	if err != nil {
		if spec.Label == "typst" {
			SetTransfer(firstLine(err.Error()))
		}
		return "", err
	}
	if spec.Label == "typst" {
		SetTransfer("")
	}
	return got, nil
}

func fetchArchive(ctx context.Context, spec archiveSpec, dest string) (string, error) {
	stage, err := os.MkdirTemp("", "lens-tool-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	if err := unpackRelease(ctx, spec, stage); err != nil {
		return "", err
	}
	src, err := findNamed(stage, spec.Names)
	if err != nil {
		return "", err
	}
	tmp := dest + ".download"
	if err := copyFile(src, tmp); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if runtime.GOOS == "darwin" {
		_ = exec.Command("xattr", "-d", "com.apple.quarantine", tmp).Run()
	}
	out, err := exec.Command(tmp, "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(out), spec.Version) {
		os.Remove(tmp)
		msg := strings.TrimSpace(string(out))
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "binary did not start"
		}
		return "", fmt.Errorf("%s %s: %s", spec.Label, spec.Version, firstLine(msg))
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return "", err
	}
	return dest, nil
}

// unpackRelease verifies the release bytes before extracting any files.
func unpackRelease(ctx context.Context, spec archiveSpec, stage string) error {
	archive, err := os.CreateTemp("", "lens-archive-*")
	if err != nil {
		return err
	}
	defer os.Remove(archive.Name())
	err = downloadVerified(ctx, spec, archive)
	closeErr := archive.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	switch {
	case strings.HasSuffix(spec.File, ".zip"):
		return extractZip(archive.Name(), stage)
	case strings.HasSuffix(spec.File, ".tar.gz"), strings.HasSuffix(spec.File, ".tgz"):
		return extractTarGZ(archive.Name(), stage)
	case strings.HasSuffix(spec.File, ".zst"):
		out, err := os.Create(filepath.Join(stage, spec.Bin))
		if err != nil {
			return err
		}
		cmd := exec.CommandContext(ctx, "zstd", "-dc", archive.Name())
		cmd.Stdout, cmd.Stderr = out, os.Stderr
		runErr := cmd.Run()
		closeErr := out.Close()
		if runErr != nil {
			return runErr
		}
		return closeErr
	default:
		return extractTarXZ(archive.Name(), stage)
	}
}

func downloadVerified(ctx context.Context, spec archiveSpec, archive io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "pairti-lense")
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %s", spec.Label, spec.Version, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s %s: download returned %s", spec.Label, spec.Version, resp.Status)
	}
	sum := sha256.New()
	total := resp.ContentLength
	if total < 0 {
		total = 0
	}
	limit := int64(80 << 20)
	if spec.Label == "pandoc" {
		limit = 96 << 20
	}
	body := &progressReader{r: io.LimitReader(resp.Body, limit), total: total, name: spec.Label + " " + spec.Version}
	if _, err := io.Copy(io.MultiWriter(archive, sum), body); err != nil {
		return err
	}
	if hex.EncodeToString(sum.Sum(nil)) != strings.ToLower(spec.SHA) {
		return fmt.Errorf("%s %s: checksum mismatch", spec.Label, spec.Version)
	}
	return nil
}

type progressReader struct {
	r     io.Reader
	n     int64
	total int64
	name  string
	next  time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if (n > 0 || err == io.EOF) && time.Now().After(p.next) {
		p.next = time.Now().Add(200 * time.Millisecond)
		SetTransfer(transferLine(p.name, p.n, p.total))
	}
	return n, err
}

func extractTarGZ(archive, dest string) error {
	return extractTar(archive, dest, "-xzf")
}

func extractTarXZ(archive, dest string) error {
	return extractTar(archive, dest, "-xJf")
}

func extractTar(archive, dest, flag string) error {
	tar, err := exec.LookPath("tar")
	if err != nil {
		return fmt.Errorf("typst %s: tar not installed", typstVersion)
	}
	cmd := exec.Command(tar, flag, archive, "-C", dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("typst %s: %s", typstVersion, firstLine(msg))
	}
	return nil
}

func findNamed(root string, names []string) (string, error) {
	want := map[string]bool{}
	for _, name := range names {
		want[name] = true
	}
	var found string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || found != "" {
			return err
		}
		if want[info.Name()] {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("archive had no compiler")
	}
	return found, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
