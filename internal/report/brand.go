package report

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

//go:embed brand/brand.typ brand/docx.lua brand/snapshot-media.lua
var brandFS embed.FS

// Brand is a version 1 manifest. The containing directory supplies its stable ID.
type Brand struct {
	ID                                                   string `json:"-"`
	Version                                              int
	Label, Subtitle, Author, Licence, Disclosure, Header string
	Ink, Accent, Muted, H1, H2, BodyFont, HeadFont       string
	Logo, Reference, StylePrefix, TableInk               string
	LogoWidth                                            int
	dir                                                  string
}

var packID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
var colorHex = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)
var styleName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)

func brandsDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "pairti-lense", "brands"), nil
}

func loadBrand(id string) (Brand, error) {
	if id == "" || id == "default" {
		return Brand{ID: "default", Label: "Default"}, nil
	}
	if !packID.MatchString(id) {
		return Brand{}, fmt.Errorf("invalid brand ID %q", id)
	}
	dir, err := brandsDir()
	if err != nil {
		return Brand{}, err
	}
	dir = filepath.Join(dir, id)
	file := filepath.Join(dir, "brand.json")
	data, err := os.ReadFile(file)
	if err != nil {
		return Brand{}, fmt.Errorf("install or repair brand %q at %s: %w", id, file, err)
	}
	b := Brand{ID: id, dir: dir, LogoWidth: 30, Ink: "222222", Accent: "444444", Muted: "666666", H1: "222222", H2: "444444", TableInk: "222222", BodyFont: "Libertinus Serif", HeadFont: "Libertinus Serif", StylePrefix: "Report"}
	if err := json.Unmarshal(data, &b); err != nil {
		return Brand{}, fmt.Errorf("%s: %w", file, err)
	}
	if b.Version != 1 || strings.TrimSpace(b.Label) == "" || !styleName.MatchString(b.StylePrefix) || b.LogoWidth < 1 || b.LogoWidth > 150 {
		return Brand{}, fmt.Errorf("%s: require version 1, a label, an alphanumeric stylePrefix and logoWidth of 1–150 mm", file)
	}
	for _, c := range []string{b.Ink, b.Accent, b.Muted, b.H1, b.H2, b.TableInk} {
		if !colorHex.MatchString(c) {
			return Brand{}, fmt.Errorf("%s: invalid six-digit colour %q", file, c)
		}
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Brand{}, err
	}
	defer root.Close()
	for _, asset := range []string{b.Logo, b.Reference} {
		if !fs.ValidPath(asset) || asset == "." || !filepath.IsLocal(asset) {
			return Brand{}, fmt.Errorf("%s: set logo and reference to pack-relative filenames", file)
		}
		info, err := root.Stat(asset)
		if err != nil || !info.Mode().IsRegular() {
			return Brand{}, fmt.Errorf("%s: missing or invalid asset %q", file, asset)
		}
	}
	return b, nil
}

func Brands() []Brand {
	neutral, _ := loadBrand("default")
	list := []Brand{neutral}
	dir, err := brandsDir()
	if err != nil {
		return list
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		id := entry.Name()
		if !entry.IsDir() || !packID.MatchString(id) || id == "default" {
			continue
		}
		b, err := loadBrand(id)
		if err != nil {
			b = Brand{ID: id, Label: id + " (repair pack)"}
		}
		list = append(list, b)
	}
	return list
}

func (b Brand) stageAsset(name, dest string) error {
	root, err := os.OpenRoot(b.dir)
	if err != nil {
		return err
	}
	defer root.Close()
	data, err := root.ReadFile(name)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0600)
}

// typstInputs supplies pack defaults; non-empty document fields override them.
func (b Brand) typstInputs(overrides map[string]string) map[string]string {
	fields := map[string]string{
		"logo-width": strconv.Itoa(b.LogoWidth), "subtitle": b.Subtitle, "licence": b.Licence,
		"disclosure": b.Disclosure, "header": b.Header, "ink": b.Ink, "accent": b.Accent,
		"muted": b.Muted, "h1": b.H1, "h2": b.H2, "body-font": b.BodyFont, "head-font": b.HeadFont,
	}
	for k, v := range overrides {
		if v != "" {
			fields[k] = v
		}
	}
	return fields
}

func stageBrand(work string, b Brand) (string, error) {
	if err := b.stageAsset(b.Logo, filepath.Join(work, "logo.png")); err != nil {
		return "", err
	}
	text, err := brandFS.ReadFile("brand/brand.typ")
	if err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(work, "main.typ"), text, 0600); err != nil {
		return "", err
	}
	fonts := filepath.Join(b.dir, "fonts")
	if info, err := os.Stat(fonts); err == nil && info.IsDir() {
		return fonts, nil
	}
	return "", nil
}
