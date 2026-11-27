package report

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// pandocVersion is the Word exporter the app carries. The reference document
// and the Lua filter were checked against this release.
const pandocVersion = "3.14"

// Digests are the sha256 lines published on the Pandoc 3.14 GitHub release.
var pandocAssets = map[string]struct {
	file string
	sha  string
}{
	"darwin/arm64":  {"pandoc-3.14-arm64-macOS.zip", "58d6593884fec9d4424b641b92503c316d9c36decb2891ff8d73ae74dd3b6304"},
	"darwin/amd64":  {"pandoc-3.14-x86_64-macOS.zip", "891708579061e7b1c1a6ce0df30ebb3cb2b6d0ca045ed3c8554ab830c1e0c270"},
	"linux/arm64":   {"pandoc-3.14-linux-arm64.tar.gz", "c692f3d78a415741bbbac4cdbbbe26fa7edca1f43d2301b982fbf4009cc8cf7f"},
	"linux/amd64":   {"pandoc-3.14-linux-amd64.tar.gz", "c5f4720cdefb3f7bdc5d07202ef579feef0d1370773c873804dd8fdaa8c4f1b8"},
	"windows/amd64": {"pandoc-3.14-windows-x86_64.zip", "0144b96e8c7b023fd382ec5ef5bdc475e873f1019d6fa72502492df2c9a94a3b"},
}

var pandocMu sync.Mutex

// PandocBin is the pinned Pandoc. A copy already in the versioned config
// directory or app bundle is used. Otherwise the 3.14 release is downloaded. PATH and the
// usual Homebrew locations are used only when that download fails.
func PandocBin() (string, error) {
	if p, err := pinnedPandoc(); err == nil {
		return p, nil
	}
	pandocMu.Lock()
	defer pandocMu.Unlock()
	if p, err := pinnedPandoc(); err == nil {
		return p, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	p, installErr := fetchPinnedPandoc(ctx)
	if installErr == nil {
		return p, nil
	}
	if found, err := findPandoc(exec.LookPath, pandocFallbacks()); err == nil {
		SetTransfer("")
		return found, nil
	}
	return "", installErr
}

func pinnedPandoc() (string, error) {
	if p, err := locateTool(pandocName(), executableDirs(true), nil); err == nil {
		return p, nil
	}
	dir, err := pandocDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, pandocName())
	if executable(p) {
		return p, nil
	}
	return "", os.ErrNotExist
}

func pandocDir() (string, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfg, "pairti-lense", "pandoc", pandocVersion), nil
}

func pandocName() string {
	if runtime.GOOS == "windows" {
		return "pandoc.exe"
	}
	return "pandoc"
}

func pandocFallbacks() []string {
	name := pandocName()
	return []string{
		filepath.Join("/opt/homebrew/bin", name),
		filepath.Join("/usr/local/bin", name),
	}
}

func fetchPinnedPandoc(ctx context.Context) (string, error) {
	dir, err := pandocDir()
	if err != nil {
		return "", err
	}
	spec, err := pandocSpec()
	if err != nil {
		return "", err
	}
	spec.Dest = dir
	SetTransfer("Downloading Pandoc " + pandocVersion)
	got, err := fetchBinary(ctx, spec)
	if err != nil {
		SetTransfer(firstLine(err.Error()))
		return "", err
	}
	SetTransfer("")
	return got, nil
}

func pandocSpec() (archiveSpec, error) {
	key := runtime.GOOS + "/" + runtime.GOARCH
	asset, ok := pandocAssets[key]
	if !ok {
		return archiveSpec{}, os.ErrNotExist
	}
	return archiveSpec{URL: "https://github.com/jgm/pandoc/releases/download/" + pandocVersion + "/" + asset.file,
		SHA: asset.sha, File: asset.file, Label: "pandoc", Version: pandocVersion, Names: []string{pandocName()}, Bin: pandocName()}, nil
}
