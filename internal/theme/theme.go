// Package theme is the semantic colour and type for the window.
// Pairti is the primary theme, taken from pairti.com.au (global.css v2.1):
// ground #050505, surface #111111, ink #ffffff, muted #a3a3a3,
// accent #2979ff, face Inter.
package theme

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
)

// Theme names roles, not raw paints. A missing file is Pairti.
type Theme struct {
	Name      string
	Canvas    uint32 // window ground
	Surface   uint32 // sidebar
	Ink       uint32 // reading text
	Muted     uint32 // paths, folders, rules of type
	Accent    uint32 // the one action colour
	Rule      uint32 // divider
	CodeFill  uint32 // fenced code ground
	String    uint32 // source string
	Number    uint32 // source number
	Selection uint32 // RRGGBBAA, selected row
	UIFont    string
	CodeFont  string
	RawFont   string // explicit codeFont override for the raw editor
	Title     float32
	Body      float32
	Meta      float32
	Code      float32
	H1        float32
	H2        float32
	H3        float32
	Line      float32
}

// Pairti is the site theme.
func Pairti() Theme {
	return Theme{
		Name:      "pairti",
		Canvas:    0x050505,
		Surface:   0x111111,
		Ink:       0xFFFFFF,
		Muted:     0xA3A3A3,
		Accent:    0x2979FF,
		Rule:      0x222222,
		CodeFill:  0x0A0A0A,
		String:    0x9DDBB5,
		Number:    0xE6C07B,
		Selection: 0x2979FF26,
		UIFont:    "Inter",
		CodeFont:  "Courier New",
		Title:     16,
		Body:      16,
		Meta:      13,
		Code:      14,
		H1:        32,
		H2:        22,
		H3:        18,
		Line:      1.6,
	}
}

type file struct {
	Name      *string  `json:"name"`
	Canvas    *string  `json:"canvas"`
	Surface   *string  `json:"surface"`
	Ink       *string  `json:"ink"`
	Muted     *string  `json:"muted"`
	Accent    *string  `json:"accent"`
	Rule      *string  `json:"rule"`
	CodeFill  *string  `json:"codeFill"`
	String    *string  `json:"string"`
	Number    *string  `json:"number"`
	Selection *string  `json:"selection"`
	UIFont    *string  `json:"uiFont"`
	CodeFont  *string  `json:"codeFont"`
	Title     *float32 `json:"title"`
	Body      *float32 `json:"body"`
	Meta      *float32 `json:"meta"`
	Code      *float32 `json:"code"`
	H1        *float32 `json:"h1"`
	H2        *float32 `json:"h2"`
	H3        *float32 `json:"h3"`
	Line      *float32 `json:"line"`
}

// Load reads a theme file. A missing file is Pairti. Present keys replace Pairti.
func Load(path string) (Theme, error) {
	t := Pairti()
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return t, nil
		}
		return t, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return t, err
	}
	if f.Name != nil {
		t.Name = *f.Name
	}
	if err := applyHex(&t.Canvas, f.Canvas, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.Surface, f.Surface, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.Ink, f.Ink, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.Muted, f.Muted, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.Accent, f.Accent, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.Rule, f.Rule, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.CodeFill, f.CodeFill, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.String, f.String, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.Number, f.Number, false); err != nil {
		return t, err
	}
	if err := applyHex(&t.Selection, f.Selection, true); err != nil {
		return t, err
	}
	if f.UIFont != nil && *f.UIFont != "" {
		t.UIFont = *f.UIFont
	}
	if f.CodeFont != nil && *f.CodeFont != "" {
		t.CodeFont = *f.CodeFont
		t.RawFont = *f.CodeFont
	}
	setSize(&t.Title, f.Title)
	setSize(&t.Body, f.Body)
	setSize(&t.Meta, f.Meta)
	setSize(&t.Code, f.Code)
	setSize(&t.H1, f.H1)
	setSize(&t.H2, f.H2)
	setSize(&t.H3, f.H3)
	setLine(&t.Line, f.Line)
	return t, nil
}

// A size below 1 or above 400, or a non-finite value, is ignored.
// The field keeps the Pairti size already stored in dst.
func setSize(dst *float32, v *float32) {
	if v == nil || !finite(*v) || *v < 1 || *v > 400 {
		return
	}
	*dst = *v
}

// Leading outside 0.8–3 is ignored. The field keeps Pairti's line height.
func setLine(dst *float32, v *float32) {
	if v == nil || !finite(*v) || *v < 0.8 || *v > 3 {
		return
	}
	*dst = *v
}

func finite(v float32) bool {
	return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0)
}

// Scale multiplies the reading sizes by k and rounds each to a whole point.
// Fractional sizes make the glyph cache draw letters on top of each other.
// A non-finite or absurd size falls back to Pairti. Colours and fonts stay put.
func (t Theme) Scale(k float32) Theme {
	if !finite(k) || k <= 0 {
		k = 1
	}
	p := Pairti()
	round := func(v, fallback float32) float32 {
		if !finite(v) || v < 1 || v > 400 {
			v = fallback
		}
		n := float32(math.Round(float64(v * k)))
		if !finite(n) || n < 1 {
			return 1
		}
		if n > 400 {
			return 400
		}
		return n
	}
	t.Body = round(t.Body, p.Body)
	t.Meta = round(t.Meta, p.Meta)
	t.Code = round(t.Code, p.Code)
	t.Title = round(t.Title, p.Title)
	t.H1 = round(t.H1, p.H1)
	t.H2 = round(t.H2, p.H2)
	t.H3 = round(t.H3, p.H3)
	if !finite(t.Line) || t.Line < 0.8 || t.Line > 3 {
		t.Line = p.Line
	}
	return t
}

// WithPairtiType keeps t's colours and fonts and restores Pairti's sizes.
func WithPairtiType(t Theme) Theme {
	p := Pairti()
	t.Title = p.Title
	t.Body = p.Body
	t.Meta = p.Meta
	t.Code = p.Code
	t.H1 = p.H1
	t.H2 = p.H2
	t.H3 = p.H3
	t.Line = p.Line
	return t
}

// Zoom stops. Fit width is computed by the window from the column.
var zoomStops = []struct {
	label string
	k     float32
	fit   bool
}{
	{"50%", 0.5, false},
	{"75%", 0.75, false},
	{"100%", 1, false},
	{"125%", 1.25, false},
	{"150%", 1.5, false},
	{"200%", 2, false},
	{"Fit width", 0, true},
}

// ZoomCount is the number of stops.
func ZoomCount() int { return len(zoomStops) }

// ZoomLabel is the stop name. An out-of-range index is 100%.
func ZoomLabel(i int) string {
	if i < 0 || i >= len(zoomStops) {
		return "100%"
	}
	return zoomStops[i].label
}

// ZoomScale is the factor for a stop. fit is true for Fit width.
func ZoomScale(i int) (k float32, fit bool) {
	if i < 0 || i >= len(zoomStops) {
		return 1, false
	}
	return zoomStops[i].k, zoomStops[i].fit
}

// ZoomIndex finds a label. Anything else is 100%.
func ZoomIndex(label string) int {
	for i, s := range zoomStops {
		if s.label == label {
			return i
		}
	}
	for i, s := range zoomStops {
		if s.label == "100%" {
			return i
		}
	}
	return 0
}

// NearestZoom picks the percent stop closest to body/16, preferring the lower one.
func NearestZoom(body float32) int {
	if !finite(body) || body < 1 {
		body = 16
	}
	pct := body / 16
	best, dist := ZoomIndex("100%"), float32(10)
	for i, s := range zoomStops {
		if s.fit {
			continue
		}
		d := pct - s.k
		if d < 0 {
			d = -d
		}
		if d < dist-0.001 || (d <= dist+0.001 && s.k < zoomStops[best].k) {
			best, dist = i, d
		}
	}
	return best
}

// ReadZoom is the saved stop, or "" when the file does not name one.
func ReadZoom(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	var f struct {
		Zoom *string `json:"zoom"`
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return "", nil
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return "", err
	}
	if f.Zoom == nil {
		return "", nil
	}
	return *f.Zoom, nil
}

// WriteZoom stores the stop. Colour keys already in the file stay.
func WriteZoom(path, label string) error {
	m := map[string]any{}
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if len(strings.TrimSpace(string(b))) > 0 {
		if err := json.Unmarshal(b, &m); err != nil {
			return err
		}
	}
	m["zoom"] = label
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepathDir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func filepathDir(path string) string {
	i := strings.LastIndex(path, string(os.PathSeparator))
	if i <= 0 {
		return "."
	}
	return path[:i]
}

func applyHex(dst *uint32, raw *string, alpha bool) error {
	if raw == nil {
		return nil
	}
	n, err := parseHex(*raw, alpha)
	if err != nil {
		return err
	}
	*dst = n
	return nil
}

func parseHex(s string, alpha bool) (uint32, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if alpha && len(s) == 6 {
		s += "ff"
	}
	n, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, err
	}
	return uint32(n), nil
}
