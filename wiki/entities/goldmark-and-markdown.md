---
title: Goldmark and Markdown
type: entity
sources: [doc:///concept.md, doc:///viewer-plan.md, beads]
related: [entities/pairti-lens.md, entities/library-and-projects.md, entities/lens-editor.md, topics/ui-layout-and-surfaces.md]
updated: 2026-11-28
---

# Goldmark and Markdown

[[Pairti Lens]] treats Markdown on disk (`.md` and `.markdown`) as the sole source of truth and uses `github.com/yuin/goldmark` in `internal/doc` to understand document structure for rendering and navigation.

## Key Facts
- **Native round-trip**: a note saved in Lens must open cleanly in another editor, produce a readable Git diff, and reopen in Lens with identical contents.
- **Markup preservation**: opening a note, switching between Raw and Formatted views, or saving an edit must preserve unfamiliar markup untouched.
- **External modification safety**: if a file changes on disk underneath an open draft, Lens keeps the writer's edits available until they choose how to resolve the conflict.
- **Rendering expectations** (`doc/viewer-plan.md`):
  - Headings appear once in the body; long lines wrap inside the reading column.
  - Supports headings, tables, links, local images, and `mermaid` code fences (showing the rendered diagram or a visible fallback when a renderer is unavailable).
  - Source files (e.g., Go or TypeScript) render with syntax colours distinguishing comments and keywords while preserving literal characters such as `*` in comments.
  - Verified against `testdata/membrane-assay.md` and real notes.
- **Isolation**: `internal/doc` handles directory walking, read capping (`1 MiB`), and Goldmark parsing without importing [[GoGPU]].

## How It Works
`internal/doc` parses Markdown bytes with Goldmark into structured blocks that `internal/ui` draws onto the document surface, while [[Lens Editor]] edits the underlying Markdown text directly so Raw and Formatted views stay synchronized without lossy HTML round-trips.

## Custom Goldmark Extensions (Pandoc Grid Tables & Marked Spans)
Added in `lens-idf` and extended in `lens-4m9` to render Word documents converted by Pandoc without leaking ASCII borders or highlight markup in Formatted view:
- **Single- and multi-column Pandoc grid tables (`gridParser`, priority 190 in `internal/doc/render.go`)**:
  - Derives column widths from the opening dashed border (`+---+` or `+---+---+`), recognizes equals-sign header separators (`+===+`) and dashed row separators, and commits the reader only after seeing a complete closing border (rewinding with `r.SetPosition` otherwise so non-matching `+` lines fall back cleanly). Single-column cards still terminate at their equals border (`len(columns) == 1 && row.header`) so adjacent one-column cards stay separate.
  - **Column slicing with `uniseg` (`gridFields`)**: only `|` delimiters at the exact opening-border column display widths split cells; `github.com/rivo/uniseg` (`FirstGraphemeClusterInString`) tracks grapheme display width so cells containing literal pipes or multibyte characters do not misalign or split.
  - Assembles each cell's lines independently before parsing with Goldmark:
    - Single-row, single-column grids populate `Block.GridParagraphs` (`internal/doc/doc.go`) with document-relative `Line` offsets for `layoutGrid` / `mapFlowSource` (`internal/ui/table.go`).
    - Multi-column (or multi-row) grids populate `Block.GridRows` (`th`/`tr` rows with explicit `Line: raw.first`), joining multi-paragraph cell spans with `Span{Text: "\n\n"}` in `row.CellSpans`.
  - **Table layout & `Line == 0` guard (`internal/ui/page.go`, `internal/ui/table.go`)**: `layoutOne` routes `len(b.GridRows) > 0` to `layoutTable`; `wrapTableRuns` splits runs on `\n` so multi-paragraph cells wrap within their column and size each row to its tallest cell. Because grid rows carry explicit `r.Line` values (unlike GFM body rows where `Line == 0`), `layoutTable` only synthesizes `ln.src = rows[0].Line + i + 1` when `r.Line == 0`, and `figureRow` rejects `rows[1].Line != 0` so two-row Pandoc grid tables are not misclassified as GFM figure rows.
- **Pandoc bracketed mark spans (`markParser`, priority 150 in `internal/doc/render.go`)**:
  - Recognizes `[text]{.mark}` both inside and outside tables while letting Goldmark parse nested inline syntax (links, emphasis) via `parser.ProcessDelimiters`, propagating `Span.Marked` through `linkWiki` and `mergeSpans`.
  - Caches `bytes.LastIndex(r.Source(), []byte(".mark"))` on `parser.Context` so ordinary Markdown files avoid lookahead scans on every `[` bracket.
  - In `internal/ui/page.go`, `drawPiece` paints a highlight rect behind marked runs (trimming `Max.X--` for the glyph-edge pixel), `layoutFlow` keeps trailing tight punctuation attached when a highlight ends at the margin, and inter-word spaces are marked only when both adjacent pieces are marked.
- **Regression fixtures**: `testdata/orchard-grid.md` (consecutive one-column Pandoc grid cards with multi-paragraph cells and `.mark` spans) and `testdata/directory-grid.md` (two-column directory grid table).

## Formatted View Blockquote Panels & Inline Spacing
Added in `lens-ckd` and `lens-ry4` for faithful Markdown rendering in `internal/doc/render.go` and `internal/ui/page.go`:
- **Continuous blockquote inset panels (`lens-ckd`)**:
  - `renderQuote` (`internal/doc/render.go`) stamps `Block.QuoteLine` (`internal/doc/doc.go`) with the blockquote's first content line on all child blocks (excluding GitHub alert callouts), preserving quote identity across multi-paragraph passages and separating consecutive blockquotes.
  - `layoutBlocks` (`internal/ui/page.go`) propagates `QuoteLine` onto both block rows and inter-paragraph gap rows between blocks sharing the same `QuoteLine`, narrowing inner layout width by `2*quotePad` (`10px`). `Draw` paints one continuous tinted background (`mixHex(Canvas, Ink, 0.07)`) and 2px left rule across the entire contiguous `quoteLine` span.
  - **Hit-testing & cursor pitfall**: `pointSource` and `moveScreen` in `internal/ui/raw.go` must mirror `quotePad` (offsetting click `dx` by `quotePad` and narrowing `moveScreen` row width by `2*quotePad`) so caret placement and vertical motion align with inset quote lines.
  - Regression fixture: `testdata/cedar-interview.md` (`TestQuotePanelSpansThePassage` at 310px/950px in light and dark palettes).
- **Source whitespace & advance parity across inline spans (`lens-ry4`)**:
  - `layoutFlow` (`internal/ui/page.go`) replaced `strings.Fields` + synthetic-space heuristics (`tightPunct`, `stuckTo`) with explicit `gap` (whitespace ` \t`) and `word` (non-whitespace) buffers scanned directly from source spans. Adjacent non-whitespace pieces across span boundaries (e.g. `` (`identifier`) `` or inline code meeting punctuation) accumulate into one `word` and wrap together while keeping each piece's resolved font and color.
  - `Draw` now uses `p.pieceWidth(pc)` directly (removing the separate `p.advance` helper) so layout wrapping and painting always share identical glyph advances across proportional prose and monospace code faces.
  - Regression coverage: `TestInlineCodeSpacing` (`internal/ui/page_test.go`) across paragraphs and blockquotes at 265px/850px and 100%/150% type scale.

## Relationships
- Parsed blocks are rendered in [[UI Layout and Surfaces]].
- Edited in place via [[Lens Editor]].
- Converted to `body.typ` during PDF compilation in [[Single-Compile Rendering]].
- Verified under Markdown page, Source view, and Saving in [[Viewer Verification Plan]].
