---
title: Goldmark and Markdown
type: entity
sources: [doc:///concept.md, doc:///viewer-plan.md]
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

## Relationships
- Parsed blocks are rendered in [[UI Layout and Surfaces]].
- Edited in place via [[Lens Editor]].
- Converted to `body.typ` during PDF compilation in [[Single-Compile Rendering]].
- Verified under Markdown page, Source view, and Saving in [[Viewer Verification Plan]].
