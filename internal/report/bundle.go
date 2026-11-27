package report

import (
	"context"
	"debug/macho"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// StageMacTools installs verified releases for the native app build. Always
// verify fresh archives here, even if the developer has a cached executable.
func StageMacTools(ctx context.Context, contents string) error {
	const version = "26.06.0-1"
	assets := map[string]struct{ cpu, sha string }{
		"arm64": {"aarch64", "e4b77ba89b44d407b4e0930179229e847bd18018f7073b667b0727bfbb5fe321"},
		"amd64": {"x86_64", "4c626388e8db026d67218a63796952fd26310e97011ae8e571c86f6d79d9a86e"},
	}
	asset, ok := assets[runtime.GOARCH]
	if runtime.GOOS != "darwin" || !ok {
		return fmt.Errorf("unsupported app target %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	typst, err := typstSpec()
	if err != nil {
		return err
	}
	pandoc, err := pandocSpec()
	if err != nil {
		return err
	}
	file := "poppler-utils-" + version + "-" + asset.cpu + "-darwin.zst"
	poppler := archiveSpec{URL: "https://github.com/unpins/poppler-utils/releases/download/v" + version + "/" + file,
		SHA: asset.sha, File: file, Label: "pdftoppm", Version: version, Bin: "pdftoppm", Names: []string{"pdftoppm"}}
	for _, spec := range []archiveSpec{typst, pandoc, poppler} {
		if err := stageMacTool(ctx, contents, spec); err != nil {
			return fmt.Errorf("stage %s: %w", spec.Label, err)
		}
	}
	return nil
}

func stageMacTool(ctx context.Context, contents string, spec archiveSpec) error {
	stage, err := os.MkdirTemp("", "lens-bundle-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	if err := unpackRelease(ctx, spec, stage); err != nil {
		return err
	}
	src, err := findNamed(stage, spec.Names)
	if err != nil {
		return err
	}
	if spec.Label == "typst" {
		for _, name := range []string{"LICENSE", "NOTICE"} {
			if err := copyFile(filepath.Join(filepath.Dir(src), name), filepath.Join(contents, "Resources", "licenses", "typst-"+name)); err != nil {
				return err
			}
		}
	}
	if spec.Label == "pdftoppm" {
		// The unpins release appends a ZIP of man pages and command aliases.
		// codesign requires __LINKEDIT to end at EOF. Rendering data lives
		// inside the Mach-O image and stays covered by its signature.
		image, err := macho.Open(src)
		if err != nil {
			return err
		}
		linkedit := image.Segment("__LINKEDIT")
		image.Close()
		if linkedit == nil {
			return fmt.Errorf("missing __LINKEDIT in %s", spec.File)
		}
		if err := os.Truncate(src, int64(linkedit.Offset+linkedit.Filesz)); err != nil {
			return err
		}
	}
	return copyFile(src, filepath.Join(contents, "MacOS", spec.Bin))
}
