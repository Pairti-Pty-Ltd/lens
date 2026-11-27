package doc

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

// lexerHit remembers which extensions the highlighter knows.
var lexerHit sync.Map

// IsMarkdown reports a Markdown document. Those stay on the reading path.
func IsMarkdown(path string) bool {
	lower := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

// IsTypst reports a Typst document.
func IsTypst(path string) bool {
	return strings.HasSuffix(strings.ToLower(filepath.Base(path)), ".typ")
}

// IsDocument reports a Markdown or Typst file.
func IsDocument(path string) bool {
	return IsMarkdown(path) || IsTypst(path)
}

// IsSource reports a file the highlighter knows, other than Markdown.
func IsSource(path string) bool {
	if path == "" || IsMarkdown(path) || binaryName(path) {
		return false
	}
	return matchLexer(filepath.Base(path))
}

// Render is the reading of path. Markdown stays Markdown. Everything else
// is source lines, so a program is not parsed as a document.
func Render(path string, src []byte) []Block {
	if path == "" || IsMarkdown(path) {
		return Blocks(src)
	}
	return SourceBlocks(path, string(src))
}

// SourceBlocks is one code row per source line, with a tone on each token.
func SourceBlocks(filename, src string) []Block {
	rows := TokenLines(filename, "", src)
	blocks := make([]Block, len(rows))
	for i, spans := range rows {
		blocks[i] = Block{
			Kind:  "code",
			Text:  spanText(spans),
			Spans: spans,
			Line:  i + 1,
			Lines: 1,
		}
	}
	if len(blocks) == 0 {
		return []Block{{Kind: "code", Line: 1, Lines: 1}}
	}
	return blocks
}

// MarkdownLines highlights Markdown source without changing the bytes.
// Headings, emphasis, links, YAML keys, fences, and table bars get a tone.
func MarkdownLines(src string) [][]Span {
	rows := tokenise(lexerFor("note.md", "markdown"), src, markdownTone)
	lines := strings.Split(src, "\n")
	for i := range rows {
		if i >= len(lines) || !plainRow(rows[i]) || !strings.Contains(lines[i], "|") {
			continue
		}
		if painted := tableRow(lines[i]); len(painted) > 0 {
			rows[i] = painted
		}
	}
	return rows
}

// TokenLines highlights src. lang is a fence name such as "go". When lang
// is empty, the lexer is chosen from filename.
func TokenLines(filename, lang, src string) [][]Span {
	return tokenise(lexerFor(filename, lang), src, func(t chroma.TokenType) (string, bool, bool) {
		return toneOf(t), false, false
	})
}

func tokenise(lexer chroma.Lexer, src string, kind func(chroma.TokenType) (string, bool, bool)) [][]Span {
	it, err := lexer.Tokenise(&chroma.TokeniseOptions{State: "root", EnsureLF: true}, src)
	n := len(strings.Split(src, "\n"))
	if n < 1 {
		n = 1
	}
	rows := make([][]Span, n)
	if err != nil || it == nil {
		for i, line := range strings.Split(src, "\n") {
			if i >= n {
				break
			}
			if line != "" {
				rows[i] = []Span{{Text: line}}
			}
		}
		return rows
	}
	line := 0
	for _, tok := range it.Tokens() {
		if tok.Value == "" {
			continue
		}
		parts := strings.Split(tok.Value, "\n")
		tone, bold, italic := kind(tok.Type)
		for i, part := range parts {
			if i > 0 {
				line++
				if line >= n {
					return rows
				}
			}
			if part == "" {
				continue
			}
			rows[line] = append(rows[line], Span{Text: part, Tone: tone, Bold: bold, Italic: italic})
		}
	}
	return rows
}

func markdownTone(t chroma.TokenType) (string, bool, bool) {
	switch t {
	case chroma.GenericHeading, chroma.GenericSubheading:
		return "heading", true, false
	case chroma.GenericStrong:
		return "strong", true, false
	case chroma.GenericEmph:
		return "emph", false, true
	case chroma.NameTag:
		return "keyword", false, false
	case chroma.NameAttribute:
		return "string", false, false
	case chroma.NameNamespace, chroma.Keyword:
		return "punct", false, false
	case chroma.Literal:
		return "comment", false, false
	default:
		return toneOf(t), false, false
	}
}

func plainRow(spans []Span) bool {
	if len(spans) == 0 {
		return true
	}
	for _, sp := range spans {
		if sp.Tone != "" || sp.Bold || sp.Italic {
			return false
		}
	}
	return true
}

func tableRow(line string) []Span {
	var spans []Span
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		spans = append(spans, Span{Text: b.String()})
		b.Reset()
	}
	for _, r := range line {
		if r == '|' {
			flush()
			spans = append(spans, Span{Text: "|", Tone: "punct"})
			continue
		}
		b.WriteRune(r)
	}
	flush()
	return spans
}

func spanText(spans []Span) string {
	var b strings.Builder
	for _, sp := range spans {
		b.WriteString(sp.Text)
	}
	return b.String()
}

func lexerFor(filename, lang string) chroma.Lexer {
	var lexer chroma.Lexer
	if lang != "" {
		lexer = lexers.Get(lang)
	}
	if lexer == nil && filename != "" {
		lexer = lexers.Match(filepath.Base(filename))
	}
	if lexer == nil && lang != "" {
		lexer = lexers.Match("file." + strings.ToLower(lang))
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	return chroma.Coalesce(lexer)
}

func toneOf(t chroma.TokenType) string {
	switch {
	case t.InCategory(chroma.Comment):
		return "comment"
	case t.InCategory(chroma.Keyword):
		return "keyword"
	case t.InCategory(chroma.LiteralString):
		return "string"
	case t.InCategory(chroma.LiteralNumber):
		return "num"
	case t.InCategory(chroma.Operator), t.InCategory(chroma.Punctuation):
		return "punct"
	default:
		return ""
	}
}

func matchLexer(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	key := ext
	if ext == "" {
		key = strings.ToLower(name)
	}
	if v, ok := lexerHit.Load(key); ok {
		return v.(bool)
	}
	hit := lexers.Match(name) != nil
	lexerHit.Store(key, hit)
	return hit
}

func binaryName(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := binaryExt[ext]
	return ok
}

var binaryExt = map[string]struct{}{
	".png": {}, ".jpg": {}, ".jpeg": {}, ".gif": {}, ".webp": {}, ".pdf": {},
	".zip": {}, ".gz": {}, ".tgz": {}, ".bz2": {}, ".xz": {}, ".rar": {}, ".7z": {},
	".wasm": {}, ".exe": {}, ".dll": {}, ".so": {}, ".dylib": {}, ".bin": {},
	".ico": {}, ".icns": {}, ".ttf": {}, ".otf": {}, ".woff": {}, ".woff2": {},
	".mp3": {}, ".mp4": {}, ".mov": {}, ".avi": {}, ".wav": {}, ".aiff": {},
	".sqlite": {}, ".db": {},
}
