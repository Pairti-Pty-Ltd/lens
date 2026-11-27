package theme

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestPairtiMatchesSite(t *testing.T) {
	th := Pairti()
	if th.Canvas != 0x050505 || th.Surface != 0x111111 || th.Ink != 0xFFFFFF {
		t.Fatalf("ground = %06x %06x %06x", th.Canvas, th.Surface, th.Ink)
	}
	if th.Accent != 0x2979FF || th.Muted != 0xA3A3A3 {
		t.Fatalf("accent/muted = %06x %06x", th.Accent, th.Muted)
	}
	if th.UIFont != "Inter" || th.CodeFont == "" {
		t.Fatalf("fonts = %q %q", th.UIFont, th.CodeFont)
	}
}

func TestLoadMissingIsPairti(t *testing.T) {
	th, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if th.Accent != Pairti().Accent || th.UIFont != "Inter" {
		t.Fatalf("got %+v", th)
	}
}

func TestScaleRoundsToWholePoints(t *testing.T) {
	th := Pairti().Scale(1.25)
	if th.Body != 20 || th.H1 != 40 || th.H2 != 28 || th.Code != 18 || th.Meta != 16 {
		t.Fatalf("sizes %+v", th)
	}
	if th.Accent != Pairti().Accent || th.Line != Pairti().Line {
		t.Fatal("colour or leading changed")
	}
	for _, v := range []float32{th.Body, th.H1, th.H2, th.H3, th.Code, th.Meta} {
		if v != float32(int(v)) {
			t.Fatalf("fractional size %v", v)
		}
	}
}

func TestWriteZoomKeepsInk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pairti-lense", "theme.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{\"ink\":\"#112233\",\"body\":99}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteZoom(path, "125%"); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	label, err := ReadZoom(path)
	if err != nil || label != "125%" {
		t.Fatalf("zoom %q %v", label, err)
	}
	got := WithPairtiType(loaded).Scale(1.25)
	if loaded.Ink != 0x112233 || got.Body != 20 || got.H1 != 40 {
		t.Fatalf("ink %06x body %v h1 %v", loaded.Ink, got.Body, got.H1)
	}
}

func TestLoadKeepsPairtiSizesWhenTheFileIsAbsurd(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme.json")
	body := []byte("{\"body\":1e20,\"line\":80,\"ink\":\"#abcdef\"}\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	th, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if th.Body != Pairti().Body || th.Line != Pairti().Line || th.Ink != 0xABCDEF {
		t.Fatalf("body %v line %v ink %06x", th.Body, th.Line, th.Ink)
	}
	scaled := th.Scale(float32(math.NaN()))
	if scaled.Body != Pairti().Body || math.IsNaN(float64(scaled.Body)) || math.IsInf(float64(scaled.Line), 0) {
		t.Fatalf("scaled body %v line %v", scaled.Body, scaled.Line)
	}
	broken := Pairti()
	broken.Body = float32(math.NaN())
	broken.Line = float32(math.Inf(1))
	out := broken.Scale(1)
	if out.Body != Pairti().Body || out.Line != Pairti().Line {
		t.Fatalf("repaired body %v line %v", out.Body, out.Line)
	}
	if NearestZoom(float32(math.NaN())) != ZoomIndex("100%") {
		t.Fatal("NaN body picked a zoom stop")
	}
}

func TestLoadRejectsANonNumericSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "theme.json")
	if err := os.WriteFile(path, []byte("{\"body\":\"huge\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	th, err := Load(path)
	if err == nil {
		t.Fatal("expected an error")
	}
	if math.IsNaN(float64(th.Body)) || th.Body != Pairti().Body {
		t.Fatalf("body %v", th.Body)
	}
}

func TestLoadOverridesInkOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "theme.json")
	body := []byte("{\"ink\":\"#112233\",\"uiFont\":\"Newsreader\"}\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatal(err)
	}
	th, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if th.Ink != 0x112233 {
		t.Fatalf("ink = %06x", th.Ink)
	}
	if th.Accent != Pairti().Accent || th.Canvas != Pairti().Canvas {
		t.Fatal("override replaced unspecified roles")
	}
	if th.UIFont != "Newsreader" {
		t.Fatalf("font = %q", th.UIFont)
	}
}
