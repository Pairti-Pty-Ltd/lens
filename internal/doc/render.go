package doc

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"strconv"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Footnote),
	goldmark.WithParserOptions(parser.WithBlockParsers(util.Prioritized(locatedRule{parser.NewThematicBreakParser()}, 199))),
)

// Goldmark discards the marker after recognizing a rule. Capture its source
// line so the formatted editor can navigate there and space the surrounding blocks.
type locatedRule struct{ parser.BlockParser }

func (r locatedRule) Open(parent ast.Node, reader text.Reader, pc parser.Context) (ast.Node, parser.State) {
	line, _ := reader.Position()
	n, state := r.BlockParser.Open(parent, reader, pc)
	if n != nil {
		n.SetAttributeString("sourceLine", line+1)
	}
	return n, state
}

// Blocks parses Markdown into the blocks the viewer draws.
// GitHub-flavoured Markdown is included: tables, task lists, and strike-through.
func Blocks(src []byte) []Block {
	reader := text.NewReader(src)
	node := markdown.Parser().Parse(reader)
	blocks := renderChildren(node, src, 0)
	body, n, ok := frontMatter(src)
	if !ok {
		return blocks
	}
	kept := make([]Block, 0, len(blocks)+1)
	kept = append(kept, Block{Kind: "front", Text: body, Line: 1, Lines: n})
	// Skip blocks consumed by the front matter while retaining later
	// generated content, which may have no source line.
	past := false
	for _, b := range blocks {
		if b.Line > n {
			past = true
			kept = append(kept, b)
			continue
		}
		if b.Line > 0 || !past {
			continue
		}
		kept = append(kept, b)
	}
	return kept
}

func renderChildren(n ast.Node, src []byte, depth int) []Block {
	var out []Block
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, renderNode(c, src, depth)...)
	}
	return out
}

func renderNode(n ast.Node, src []byte, depth int) []Block {
	switch n := n.(type) {
	case *ast.Heading:
		spans := inlineSpans(n, src)
		return []Block{block(fmt.Sprintf("h%d", n.Level), spans, depth, "", n, src)}
	case *ast.Paragraph, *ast.TextBlock:
		spans := inlineSpans(n, src)
		return []Block{block("p", spans, depth, "", n, src)}
	case *ast.List:
		return renderList(n, src, depth)
	case *ast.Blockquote:
		return renderQuote(n, src, depth)
	case *ast.FencedCodeBlock:
		return []Block{codeBlock("code", n, rawLines(n.Lines(), src), strings.TrimSpace(string(n.Language(src))), depth, src)}
	case *ast.CodeBlock:
		return []Block{codeBlock("code", n, rawLines(n.Lines(), src), "", depth, src)}
	case *ast.ThematicBreak:
		source, _ := n.AttributeString("sourceLine")
		line, _ := source.(int)
		return []Block{{Kind: "hr", Line: line, Lines: 1}}
	case *ast.HTMLBlock:
		body := rawLines(n.Lines(), src)
		if alt, srcURL, ok := imageFromHTML(body); ok {
			return []Block{block("p", []Span{{Text: alt, Link: srcURL, Image: true}}, depth, "", n, src)}
		}
		return []Block{codeBlock("html", n, body, "", depth, src)}
	case *east.Table:
		return renderTable(n, src)
	case *east.FootnoteList:
		out := []Block{{Kind: "hr"}}
		return append(out, renderChildren(n, src, depth)...)
	case *east.Footnote:
		return renderFootnote(n, src)
	default:
		return renderChildren(n, src, depth)
	}
}

func renderFootnote(n *east.Footnote, src []byte) []Block {
	inner := renderChildren(n, src, 0)
	label := strconv.Itoa(n.Index)
	if n.Index < 1 {
		label = string(n.Ref)
	}
	if len(inner) == 0 {
		return []Block{block("fn", nil, 0, label, n, src)}
	}
	for i := range inner {
		inner[i].Kind = "fn"
		if i == 0 {
			inner[i].Marker = label
		} else {
			inner[i].Marker = ""
		}
	}
	return inner
}

func renderQuote(n *ast.Blockquote, src []byte, depth int) []Block {
	inner := renderChildren(n, src, depth)
	if len(inner) == 0 {
		return nil
	}
	if kind, ok := takeAlert(&inner[0]); ok {
		if strings.TrimSpace(inner[0].Text) == "" {
			inner = inner[1:]
		}
		if len(inner) == 0 {
			return []Block{{Kind: "alert", Marker: kind}}
		}
		lead := true
		for i := range inner {
			if inner[i].Kind != "p" && !strings.HasPrefix(inner[i].Kind, "h") {
				continue
			}
			inner[i].Kind = "alert"
			if lead {
				inner[i].Marker = kind
				lead = false
			}
		}
		return inner
	}
	for i := range inner {
		if inner[i].Kind == "p" || strings.HasPrefix(inner[i].Kind, "h") {
			inner[i].Kind = "quote"
		}
	}
	return inner
}

func takeAlert(b *Block) (string, bool) {
	kind, rest, ok := cutAlert(b.Text)
	if !ok {
		return "", false
	}
	b.Text = rest
	b.Spans = trimAlertSpans(b.Spans)
	return kind, true
}

func cutAlert(s string) (kind, rest string, ok bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToUpper(s), "[!") {
		return "", "", false
	}
	end := strings.Index(s, "]")
	if end < 0 {
		return "", "", false
	}
	tag := strings.ToUpper(s[:end+1])
	switch tag {
	case "[!NOTE]", "[!TIP]", "[!IMPORTANT]", "[!WARNING]", "[!CAUTION]":
		return strings.ToLower(tag[2 : len(tag)-1]), strings.TrimSpace(s[end+1:]), true
	default:
		return "", "", false
	}
}

func trimAlertSpans(spans []Span) []Span {
	if len(spans) == 0 {
		return spans
	}
	if _, rest, ok := cutAlert(spans[0].Text); ok {
		if strings.TrimSpace(rest) == "" {
			spans = append([]Span(nil), spans[1:]...)
		} else {
			spans[0].Text = strings.TrimSpace(rest)
		}
	}
	for len(spans) > 0 && strings.TrimSpace(spans[0].Text) == "" {
		spans = spans[1:]
	}
	return spans
}

func renderList(n *ast.List, src []byte, depth int) []Block {
	num := n.Start
	if num < 1 {
		num = 1
	}
	var out []Block
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		item, ok := c.(*ast.ListItem)
		if !ok {
			continue
		}
		marker := "·"
		if n.IsOrdered() {
			marker = fmt.Sprintf("%d.", num)
			num++
		}
		first := true
		for ch := item.FirstChild(); ch != nil; ch = ch.NextSibling() {
			switch ch := ch.(type) {
			case *ast.List:
				out = append(out, renderList(ch, src, depth+1)...)
			case *ast.Paragraph, *ast.TextBlock:
				if box := taskBox(ch); box != nil {
					if box.IsChecked {
						marker = "[x]"
					} else {
						marker = "[ ]"
					}
				}
				spans := inlineSpans(ch, src)
				b := block("li", spans, depth, marker, ch, src)
				if !first {
					b.Marker = ""
				}
				out = append(out, b)
				first = false
			default:
				out = append(out, renderNode(ch, src, depth)...)
			}
		}
	}
	return out
}

func taskBox(n ast.Node) *east.TaskCheckBox {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if box, ok := c.(*east.TaskCheckBox); ok {
			return box
		}
	}
	return nil
}

func renderTable(n *east.Table, src []byte) []Block {
	var out []Block
	first := 0
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if first == 0 {
			if line, _ := sourceLine(c, src); line > 0 {
				first = line
				if len(out) > 0 {
					// An empty header has no text segment. Count back over the delimiter too.
					first -= len(out) + 1
				}
			}
		}
		switch c.(type) {
		case *east.TableHeader:
			out = append(out, tableRowBlock("th", c, src))
		case *east.TableRow:
			out = append(out, tableRowBlock("tr", c, src))
		}
	}
	if len(out) > 0 && first > 0 {
		out[0].Line = first
		out[0].Lines = len(out) + 1 // header, delimiter, then body rows
	}
	return out
}

func tableRowBlock(kind string, n ast.Node, src []byte) Block {
	row := Block{Kind: kind}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		spans := inlineSpans(c, src)
		row.CellSpans = append(row.CellSpans, spans)
		row.Cells = append(row.Cells, strings.TrimSpace(plainSpans(spans)))
		link := ""
		for _, sp := range spans {
			if strings.HasPrefix(sp.Link, "wiki:") {
				link = sp.Link
				break
			}
		}
		row.CellLinks = append(row.CellLinks, link)
	}
	row.Text = strings.Join(row.Cells, " | ")
	return row
}

func block(kind string, spans []Span, depth int, marker string, n ast.Node, src []byte) Block {
	return placed(Block{
		Kind: kind, Text: strings.TrimSpace(plainSpans(spans)), Spans: spans, Depth: depth, Marker: marker,
	}, n, src)
}

func codeBlock(kind string, n ast.Node, body, marker string, depth int, src []byte) Block {
	b := placed(Block{Kind: kind, Text: body, Marker: marker, Depth: depth}, n, src)
	if b.Lines < 1 && body != "" {
		b.Lines = strings.Count(body, "\n") + 1
	}
	return b
}

func placed(b Block, n ast.Node, src []byte) Block {
	b.Line, b.Lines = sourceLine(n, src)
	if b.Line > 0 && b.Lines < 1 {
		b.Lines = 1
	}
	return b
}

// sourceLine is the 1-based line of the node's first source segment,
// and how many source lines it covers. Table rows keep their text on
// inline segments, not on Lines(), so those are walked.
func sourceLine(n ast.Node, src []byte) (line, nlines int) {
	if n == nil {
		return 0, 0
	}
	if n.Lines() != nil && n.Lines().Len() > 0 {
		lines := n.Lines()
		start := lines.At(0).Start
		if start < 0 {
			start = 0
		}
		if start > len(src) {
			start = len(src)
		}
		return bytes.Count(src[:start], []byte("\n")) + 1, lines.Len()
	}
	start, stop := -1, -1
	walkText(n, &start, &stop)
	if start < 0 {
		return 0, 0
	}
	if start > len(src) {
		start = len(src)
	}
	if stop < start {
		stop = start
	}
	if stop > len(src) {
		stop = len(src)
	}
	line = bytes.Count(src[:start], []byte("\n")) + 1
	end := bytes.Count(src[:stop], []byte("\n")) + 1
	nlines = end - line + 1
	if nlines < 1 {
		nlines = 1
	}
	return line, nlines
}

func walkText(n ast.Node, start, stop *int) {
	if n == nil {
		return
	}
	if t, ok := n.(*ast.Text); ok {
		if *start < 0 || t.Segment.Start < *start {
			*start = t.Segment.Start
		}
		if t.Segment.Stop > *stop {
			*stop = t.Segment.Stop
		}
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		walkText(c, start, stop)
	}
}

// CountWords is the number of words in the rendered text.
func CountWords(blocks []Block) int {
	n := 0
	for _, b := range blocks {
		n += len(strings.Fields(b.Text))
	}
	return n
}

func plainSpans(spans []Span) string {
	var b strings.Builder
	for _, sp := range spans {
		b.WriteString(sp.Text)
	}
	return b.String()
}

func inlineSpans(n ast.Node, src []byte) []Span {
	var out []Span
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, spanNode(c, src)...)
	}
	return linkWiki(mergeSpans(out))
}

// linkWiki runs after adjacent text is joined. Goldmark splits the opening
// bracket of [[target]] into its own text node, so the pair is only visible here.
func linkWiki(spans []Span) []Span {
	var out []Span
	changed := false
	for _, sp := range spans {
		if sp.Code || sp.Image || sp.Link != "" || !strings.Contains(sp.Text, "[[") {
			out = append(out, sp)
			continue
		}
		changed = true
		for _, part := range splitWiki(sp.Text) {
			part.Bold = sp.Bold || part.Bold
			part.Italic = sp.Italic || part.Italic
			part.Strike = sp.Strike || part.Strike
			if part.Link == "" && strings.Contains(strings.ToLower(part.Text), "doc:") {
				for _, d := range splitDocURIs(part.Text) {
					d.Bold = part.Bold
					d.Italic = part.Italic
					d.Strike = part.Strike
					out = append(out, d)
				}
				continue
			}
			out = append(out, part)
		}
	}
	if !changed {
		return spans
	}
	return out
}

func spanNode(n ast.Node, src []byte) []Span {
	switch n := n.(type) {
	case *ast.Text:
		seg := n.Segment
		t := string((&seg).Value(src))
		t = strings.TrimRight(t, "\n")
		if t == "" {
			if n.HardLineBreak() {
				return []Span{{Text: "\n"}}
			}
			if n.SoftLineBreak() {
				return []Span{{Text: " "}}
			}
			return nil
		}
		spans := splitDocURIs(t)
		if n.HardLineBreak() {
			spans[len(spans)-1].Text += "\n"
		} else if n.SoftLineBreak() {
			spans[len(spans)-1].Text += " "
		}
		return spans
	case *ast.String:
		return []Span{{Text: string(n.Value)}}
	case *ast.CodeSpan:
		return []Span{{Text: string(n.Text(src)), Code: true}}
	case *ast.Emphasis:
		inner := inlineSpans(n, src)
		for i := range inner {
			if n.Level >= 2 {
				inner[i].Bold = true
			} else {
				inner[i].Italic = true
			}
		}
		return inner
	case *east.Strikethrough:
		inner := inlineSpans(n, src)
		for i := range inner {
			inner[i].Strike = true
		}
		return inner
	case *ast.Link:
		inner := inlineSpans(n, src)
		return withLink(inner, string(n.Destination))
	case *ast.AutoLink:
		label := string(n.Label(src))
		url := string(n.URL(src))
		if label == "" {
			label = url
		}
		return withLink([]Span{{Text: label}}, url)
	case *ast.Image:
		alt := strings.TrimSpace(plainSpans(inlineSpans(n, src)))
		return []Span{{Text: alt, Link: string(n.Destination), Image: true}}
	case *ast.RawHTML:
		raw := rawSegments(n.Segments, src)
		if alt, srcURL, ok := imageFromHTML(raw); ok {
			return []Span{{Text: alt, Link: srcURL, Image: true}}
		}
		return []Span{{Text: raw, Code: true}}
	case *east.TaskCheckBox:
		return nil
	case *east.FootnoteLink:
		label := strconv.Itoa(n.Index)
		link := ""
		if line := footnoteLine(n, src); line > 0 {
			link = "#line:" + strconv.Itoa(line)
		}
		return []Span{{Text: label, Link: link, Note: true}}
	case *east.FootnoteBacklink:
		return nil
	default:
		return inlineSpans(n, src)
	}
}

func footnoteLine(link *east.FootnoteLink, src []byte) int {
	root := ast.Node(link)
	for root.Parent() != nil {
		root = root.Parent()
	}
	var line int
	_ = ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		fn, ok := n.(*east.Footnote)
		if !entering || !ok || fn.Index != link.Index {
			return ast.WalkContinue, nil
		}
		line, _ = sourceLine(fn, src)
		return ast.WalkStop, nil
	})
	return line
}

var htmlAttrRE = regexp.MustCompile(`(?i)(?:^|[\s<])([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'=<>` + "`" + `]+))`)

// imageFromHTML reads a fragment that is a single img tag.
func imageFromHTML(s string) (alt, src string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.Count(s, "<") != 1 || !strings.HasSuffix(s, ">") {
		return "", "", false
	}
	head := s
	if len(head) > 5 {
		head = head[:5]
	}
	if !strings.HasPrefix(strings.ToLower(head), "<img") {
		return "", "", false
	}
	src = htmlAttr(s, "src")
	if src == "" {
		return "", "", false
	}
	return htmlAttr(s, "alt"), src, true
}

func htmlAttr(tag, name string) string {
	for _, m := range htmlAttrRE.FindAllStringSubmatch(tag, -1) {
		if len(m) < 5 || !strings.EqualFold(m[1], name) {
			continue
		}
		for _, g := range m[2:] {
			if g != "" {
				return g
			}
		}
		return ""
	}
	return ""
}

func withLink(spans []Span, url string) []Span {
	url = strings.TrimSpace(url)
	if len(spans) == 0 {
		if url == "" {
			return nil
		}
		return []Span{{Text: url, Link: url}}
	}
	for i := range spans {
		spans[i].Link = url
	}
	return spans
}

func mergeSpans(spans []Span) []Span {
	var out []Span
	for _, sp := range spans {
		if sp.Text == "" && !sp.Image {
			continue
		}
		if len(out) == 0 {
			out = append(out, sp)
			continue
		}
		prev := &out[len(out)-1]
		if prev.Bold == sp.Bold && prev.Italic == sp.Italic && prev.Code == sp.Code && prev.Strike == sp.Strike && prev.Link == sp.Link && prev.Note == sp.Note {
			prev.Text += sp.Text
			continue
		}
		out = append(out, sp)
	}
	return out
}

func rawLines(lines *text.Segments, src []byte) string {
	if lines == nil {
		return ""
	}
	var b strings.Builder
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write((&seg).Value(src))
	}
	return strings.TrimRight(b.String(), "\n")
}

func rawSegments(lines *text.Segments, src []byte) string {
	return rawLines(lines, src)
}
