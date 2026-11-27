package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func emptyBrands(t *testing.T) string {
	t.Helper()
	config := t.TempDir()
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "AppData"} {
		t.Setenv(key, config)
	}
	dir, err := brandsDir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func installExample(t *testing.T, dir string) {
	t.Helper()
	if err := os.CopyFS(filepath.Join(dir, "cedar"), os.DirFS("../../testdata/brands/cedar")); err != nil {
		t.Fatal(err)
	}
}

func TestPackDiscoveryAndRepair(t *testing.T) {
	dir := emptyBrands(t)
	if got := Outputs(); len(got) != 2 || got[0].ID != "default.pdf" || got[1].ID != "default.docx" {
		t.Fatal(got)
	}
	req := ExportRequest{Brand: "cedar.pdf", Src: []byte("# Report"), Dir: t.TempDir()}
	if _, err := Export(req); err == nil || !strings.Contains(err.Error(), "brand.json") {
		t.Fatalf("missing pack: %v", err)
	}
	installExample(t, dir)
	if got := Outputs(); len(got) != 4 || got[2].ID != "cedar.pdf" {
		t.Fatal(got)
	}
	if err := os.WriteFile(filepath.Join(dir, "cedar", "brand.json"), []byte(`{"version":19}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Export(req); err == nil || !strings.Contains(err.Error(), "version 1") {
		t.Fatalf("invalid pack: %v", err)
	}
}
