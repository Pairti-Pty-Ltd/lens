package report

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"pairti-lense/internal/diagram/mermaid"
	"pairti-lense/internal/doc"
)

// Body is Typst source for the note. The template includes it.
// Image files are not copied. writeBody does that for a compile.
func Body(src []byte) string {
	return writeBody("", "", src)
}

// writeBody is the Typst note. Local images are copied into work when base
// is the directory those relative paths start from.
func writeBody(work, base string, src []byte) string {
	t := &typer{work: work, base: base, seen: map[string]string{}}
	blocks := doc.Blocks(src)
	for i := 0; i < len(blocks); i++ {
		bl := blocks[i]
		switch {
		case bl.Kind == "front":
			continue
		case bl.Kind == "th" || bl.Kind == "tr":
			j := i
			for j < len(blocks) && (blocks[j].Kind == "th" || blocks[j].Kind == "tr") {
				j++
			}
			writeTable(&t.b, blocks[i:j])
			i = j - 1
		case strings.HasPrefix(bl.Kind, "h"):
			level := 1
			if len(bl.Kind) == 2 && bl.Kind[1] >= '1' && bl.Kind[1] <= '6' {
				level = int(bl.Kind[1] - '0')
			}
			t.b.WriteString(strings.Repeat("=", level))
			t.b.WriteByte(' ')
			t.b.WriteString(t.inline(bl.Spans, bl.Text))
			t.b.WriteString("\n\n")
		case bl.Kind == "code" || bl.Kind == "html":
			if t.work != "" && strings.EqualFold(bl.Marker, "mermaid") {
				if fig := t.diagram(bl.Text); fig != "" {
					t.b.WriteString(fig)
					t.b.WriteString("\n\n")
					break
				}
			}
			fence := strings.Repeat("`", fenceLen(bl.Text)+1)
			t.b.WriteString(fence)
			t.b.WriteString(bl.Marker)
			t.b.WriteByte('\n')
			t.b.WriteString(bl.Text)
			if !strings.HasSuffix(bl.Text, "\n") {
				t.b.WriteByte('\n')
			}
			t.b.WriteString(fence)
			t.b.WriteString("\n\n")
		case bl.Kind == "li":
			indent := strings.Repeat("  ", bl.Depth)
			mark := "- "
			if bl.Marker != "" && bl.Marker != "·" {
				mark = bl.Marker + " "
			}
			t.b.WriteString(indent)
			t.b.WriteString(mark)
			t.b.WriteString(t.inline(bl.Spans, bl.Text))
			t.b.WriteByte('\n')
		case bl.Kind == "quote":
			t.b.WriteString("#quote[")
			t.b.WriteString(t.inline(bl.Spans, bl.Text))
			t.b.WriteString("]\n\n")
		case bl.Kind == "hr":
			t.b.WriteString("#line(length: 100%)\n\n")
		default:
			text := t.inline(bl.Spans, bl.Text)
			if text == "" {
				continue
			}
			t.b.WriteString(text)
			t.b.WriteString("\n\n")
		}
	}
	return t.b.String()
}

type typer struct {
	b          strings.Builder
	work, base string
	n          int
	diagrams   int
	seen       map[string]string
}

func (t *typer) diagram(source string) string {
	t.diagrams++
	names, ok := writeDiagram(t.work, t.diagrams, source)
	if !ok {
		t.diagrams--
		return ""
	}
	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(t.diagramFigure(name))
	}
	return b.String()
}

func writeTable(b *strings.Builder, rows []doc.Block) {
	n := 0
	for _, r := range rows {
		if len(r.Cells) > n {
			n = len(r.Cells)
		}
	}
	if n == 0 {
		return
	}
	b.WriteString("#table(\n  columns: ")
	b.WriteString(itoa(n))
	b.WriteString(",\n")
	for _, r := range rows {
		for i := 0; i < n; i++ {
			cell := ""
			if i < len(r.Cells) {
				cell = escape(r.Cells[i])
			}
			b.WriteString("  [")
			b.WriteString(cell)
			b.WriteString("],\n")
		}
	}
	b.WriteString(")\n\n")
}

func (t *typer) inline(spans []doc.Span, plain string) string {
	if len(spans) == 0 {
		return escape(plain)
	}
	var b strings.Builder
	for _, sp := range spans {
		if sp.Image {
			b.WriteString(t.image(sp))
			continue
		}
		text := escape(sp.Text)
		if text == "" {
			continue
		}
		switch {
		case sp.Code:
			text = "`" + strings.ReplaceAll(sp.Text, "`", "\\`") + "`"
		case sp.Bold && sp.Italic:
			text = "#strong[#emph[" + text + "]]"
		case sp.Bold:
			text = "*" + text + "*"
		case sp.Italic:
			text = "_" + text + "_"
		}
		if sp.Strike {
			text = "#strike[" + text + "]"
		}
		if sp.Link != "" && !sp.Image {
			text = "#link(\"" + escape(sp.Link) + "\")[" + text + "]"
		}
		b.WriteString(text)
	}
	return b.String()
}

func (t *typer) image(sp doc.Span) string {
	abs := resolveImage(t.base, sp.Link)
	if abs == "" || t.work == "" {
		label := strings.TrimSpace(sp.Text)
		if label == "" {
			label = sp.Link
		}
		if label == "" {
			return ""
		}
		return escape(label)
	}
	name, ok := t.seen[abs]
	if !ok {
		t.n++
		ext := strings.ToLower(filepath.Ext(abs))
		if ext == "" {
			ext = ".img"
		}
		name = fmt.Sprintf("img-%d%s", t.n, ext)
		if err := copyImage(abs, filepath.Join(t.work, name)); err != nil {
			label := strings.TrimSpace(sp.Text)
			if label == "" {
				return ""
			}
			return escape(label)
		}
		t.seen[abs] = name
	}
	return t.figure(name, abs)
}

func (t *typer) figure(name, path string) string {
	width := "100%"
	if w, h, ok := imagePixels(path); ok && w > 0 && h > 0 {
		wm := float64(w) / 96 * 25.4
		hm := float64(h) / 96 * 25.4
		scale := 1.0
		if wm > 116.2 {
			scale = 116.2 / wm
		}
		if hm*scale > 172.9 {
			scale = 172.9 / hm
		}
		width = fmt.Sprintf("%.1fmm", wm*scale)
	}
	return "#image(\"" + typstStr(name) + "\", width: " + width + ")"
}

// diagramFigure sizes a paper diagram at the scale PaperPNG draws (2),
// so the column width is the diagram's own width and not twice that.
func (t *typer) diagramFigure(name string) string {
	return "#image(\"" + typstStr(name) + "\", width: " + diagramWidth(filepath.Join(t.work, name)) + ")"
}

func diagramWidth(path string) string {
	w, h, ok := imagePixels(path)
	if !ok || w < 1 || h < 1 {
		return fmt.Sprintf("%.1fmm", diagramColumnMM)
	}
	wm := float64(w) / diagramDPI * 25.4
	hm := float64(h) / diagramDPI * 25.4
	scale := 1.0
	if wm > diagramColumnMM {
		scale = diagramColumnMM / wm
	}
	if hm*scale > diagramSliceMM {
		scale = diagramSliceMM / hm
	}
	if scale > 1 {
		scale = 1
	}
	return fmt.Sprintf("%.1fmm", wm*scale)
}

func resolveImage(base, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"), strings.HasPrefix(lower, "data:"):
		return ""
	case strings.HasPrefix(lower, "file:"):
		u, err := url.Parse(raw)
		if err != nil {
			return ""
		}
		return existingFile(u.Path)
	}
	u := raw
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	if dec, err := url.PathUnescape(u); err == nil {
		u = dec
	}
	u = filepath.FromSlash(u)
	if filepath.IsAbs(u) {
		return existingFile(u)
	}
	if base == "" {
		return ""
	}
	return existingFile(filepath.Clean(filepath.Join(base, u)))
}

func existingFile(path string) string {
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() || st.Size() == 0 {
		return ""
	}
	return path
}

func copyImage(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func imagePixels(path string) (int, int, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil || cfg.Width < 1 || cfg.Height < 1 {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// Paper diagrams are drawn at scale 2. 192 px is one inch on the page.
// A picture stays at the 70% cap. A diagram uses the text column, and a
// diagram taller than one page is cut into strips so the type stays readable.
const (
	diagramDPI      = 192.0
	diagramColumnMM = 166.0
	diagramSliceMM  = 180.0
)

// writeDiagram renders one mermaid fence into work. The names are the PNG
// strips, top to bottom. One strip is diag-N.png. Further strips are
// diag-N-2.png and so on.
func writeDiagram(work string, n int, source string) ([]string, bool) {
	if work == "" || n < 1 {
		return nil, false
	}
	data, err := mermaid.PaperPNG(context.Background(), source)
	if err != nil || len(data) < 8 {
		return nil, false
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	parts := sliceDiagram(img)
	names := make([]string, 0, len(parts))
	for i, part := range parts {
		name := fmt.Sprintf("diag-%d.png", n)
		if i > 0 {
			name = fmt.Sprintf("diag-%d-%d.png", n, i+1)
		}
		f, err := os.Create(filepath.Join(work, name))
		if err != nil {
			return nil, false
		}
		err = png.Encode(f, part)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			return nil, false
		}
		names = append(names, name)
	}
	return names, len(names) > 0
}

// sliceDiagram cuts a tall diagram into page-height strips. The cut is in
// pixels of the scale-2 PNG, using the same column width the page will apply.
func sliceDiagram(img image.Image) []image.Image {
	b := img.Bounds()
	if b.Dx() < 1 || b.Dy() < 1 {
		return nil
	}
	mmW := float64(b.Dx()) / diagramDPI * 25.4
	mmH := float64(b.Dy()) / diagramDPI * 25.4
	if mmW > diagramColumnMM {
		mmH *= diagramColumnMM / mmW
	}
	n := 1
	if mmH > diagramSliceMM {
		n = int(math.Ceil(mmH / diagramSliceMM))
	}
	if n < 2 {
		return []image.Image{img}
	}
	sub, ok := img.(interface {
		SubImage(image.Rectangle) image.Image
	})
	if !ok {
		return []image.Image{img}
	}
	span := b.Dy() / n
	out := make([]image.Image, 0, n)
	top := b.Min.Y
	for top < b.Max.Y {
		bot := top + span
		if bot >= b.Max.Y-span/5 {
			bot = b.Max.Y
		} else {
			bot = lightestRow(img, bot-span/8, bot+span/8)
			if bot <= top {
				bot = top + span
			}
		}
		out = append(out, sub.SubImage(image.Rect(b.Min.X, top, b.Max.X, bot)))
		top = bot
	}
	return out
}

// lightestRow is the row in [from, to) whose pixels are closest to white,
// so a page cut lands in a gap when the diagram has one there.
func lightestRow(img image.Image, from, to int) int {
	b := img.Bounds()
	if from < b.Min.Y {
		from = b.Min.Y
	}
	if to > b.Max.Y {
		to = b.Max.Y
	}
	best := from
	bestScore := -1.0
	target := (from + to) / 2
	for y := from; y < to; y += 2 {
		var sum float64
		n := 0
		for x := b.Min.X; x < b.Max.X; x += 4 {
			r, g, bl, _ := img.At(x, y).RGBA()
			sum += float64(r + g + bl)
			n++
		}
		if n == 0 {
			continue
		}
		score := sum / float64(n)
		closer := math.Abs(float64(y-target)) < math.Abs(float64(best-target))
		if score > bestScore+1 || (closer && score >= bestScore-1) {
			best, bestScore = y, score
		}
	}
	return best
}

// stageDiagrams replaces mermaid fences with pictures beside the note Pandoc
// reads. A fence that does not render stays source, so the export continues.
func stageDiagrams(work string, src []byte) []byte {
	if work == "" {
		return src
	}
	s := string(src)
	var b strings.Builder
	n := 0
	for {
		open, end, lang, body, ok := nextFence(s)
		if !ok {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:open])
		if strings.EqualFold(lang, "mermaid") {
			n++
			if names, ok := writeDiagram(work, n, body); ok {
				for i, name := range names {
					if i > 0 {
						b.WriteString("\n\n")
					}
					b.WriteString("![](")
					b.WriteString(name)
					b.WriteString(")")
				}
				b.WriteString("\n\n")
				s = s[end:]
				continue
			}
			n--
		}
		b.WriteString(s[open:end])
		s = s[end:]
	}
	return []byte(b.String())
}

// nextFence returns the first fenced block. end is the index after the
// closing line. An unclosed fence is not a fence.
func nextFence(s string) (open, end int, lang, body string, ok bool) {
	i := 0
	for i <= len(s) {
		nl := strings.IndexByte(s[i:], '\n')
		row := s[i:]
		next := len(s)
		if nl >= 0 {
			row = s[i : i+nl]
			next = i + nl + 1
		}
		indent, mark, count, info, isOpen := fenceOpen(row)
		if isOpen {
			closeAt, inner, found := fenceClose(s[next:], mark, count, indent)
			if found {
				return i, next + closeAt, info, inner, true
			}
		}
		if nl < 0 {
			break
		}
		i = next
	}
	return 0, 0, "", "", false
}

func fenceOpen(row string) (indent int, mark byte, count int, info string, ok bool) {
	row = strings.TrimRight(row, "\r")
	i := 0
	for i < len(row) && i < 3 && row[i] == ' ' {
		i++
	}
	if i >= len(row) {
		return 0, 0, 0, "", false
	}
	mark = row[i]
	if mark != '`' && mark != '~' {
		return 0, 0, 0, "", false
	}
	n := 0
	for i+n < len(row) && row[i+n] == mark {
		n++
	}
	if n < 3 {
		return 0, 0, 0, "", false
	}
	rest := strings.TrimSpace(row[i+n:])
	if i := strings.IndexByte(rest, ' '); i >= 0 {
		rest = rest[:i]
	}
	return i, mark, n, rest, true
}

func fenceClose(s string, mark byte, count, indent int) (end int, body string, ok bool) {
	i := 0
	bodyEnd := 0
	for i <= len(s) {
		nl := strings.IndexByte(s[i:], '\n')
		row := s[i:]
		next := len(s)
		if nl >= 0 {
			row = s[i : i+nl]
			next = i + nl + 1
		}
		if fenceCloses(row, mark, count, indent) {
			return next, strings.TrimRight(s[:bodyEnd], "\n"), true
		}
		bodyEnd = next
		if nl < 0 {
			break
		}
		i = next
	}
	return 0, "", false
}

func fenceCloses(row string, mark byte, count, indent int) bool {
	row = strings.TrimRight(row, "\r")
	i := 0
	for i < len(row) && row[i] == ' ' {
		i++
	}
	if i > indent || i >= len(row) || row[i] != mark {
		return false
	}
	n := 0
	for i+n < len(row) && row[i+n] == mark {
		n++
	}
	if n < count {
		return false
	}
	return strings.TrimSpace(row[i+n:]) == ""
}

// stageMarkdownImages copies local pictures next to the note Pandoc reads
// and rewrites the links to those copies. Pandoc does not search the note's
// own folder once the markdown has been moved into a temp directory.
func stageMarkdownImages(work, base string, src []byte) []byte {
	text := string(src)
	seen := map[string]string{}
	var links []string
	used := map[string]bool{}
	for _, block := range doc.Blocks(src) {
		for _, sp := range block.Spans {
			if !sp.Image || sp.Link == "" || used[sp.Link] {
				continue
			}
			used[sp.Link] = true
			links = append(links, sp.Link)
		}
	}
	sort.Slice(links, func(i, j int) bool { return len(links[i]) > len(links[j]) })
	n := 0
	for _, link := range links {
		abs := resolveImage(base, link)
		if abs == "" {
			continue
		}
		name, ok := seen[abs]
		if !ok {
			n++
			ext := strings.ToLower(filepath.Ext(abs))
			if ext == "" {
				ext = ".img"
			}
			name = fmt.Sprintf("img-%d%s", n, ext)
			if err := copyImage(abs, filepath.Join(work, name)); err != nil {
				continue
			}
			seen[abs] = name
		}
		text = replaceMarkdownLink(text, link, name)
	}
	return []byte(text)
}

func replaceMarkdownLink(text, old, new string) string {
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(text[i:], old)
		if j < 0 {
			b.WriteString(text[i:])
			break
		}
		j += i
		prefix := text[:j]
		if strings.HasSuffix(prefix, "](") || strings.HasSuffix(prefix, "src=\"") || strings.HasSuffix(prefix, "src='") || strings.HasSuffix(prefix, "]: ") {
			b.WriteString(text[i:j])
			b.WriteString(new)
			i = j + len(old)
			continue
		}
		b.WriteString(text[i : j+len(old)])
		i = j + len(old)
	}
	return b.String()
}

func typstStr(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

func escape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '#', '$', '@', '<', '>', '*', '_', '`', '[', ']':
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func fenceLen(s string) int {
	n := 2
	run := 0
	for _, r := range s {
		if r == '`' {
			run++
			if run > n {
				n = run
			}
			continue
		}
		run = 0
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [8]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	return string(d[i:])
}
