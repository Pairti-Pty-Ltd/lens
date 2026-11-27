---
title: Goldmark and Markdown
type: entity
sources: [doc:///concept.md, doc:///build-0.md, doc:///viewer-plan.md]
related: [entities/pairti-lens.md, entities/library-and-projects.md, entities/lens-editor.md, topics/ui-layout-and-surfaces.md]
updated: 2026-11-27
---

# Goldmark and Markdown

[[Pairti Lens]] treats Markdown on disk (`.md` and `.markdown`) as the sole source of truth and parses it into renderable blocks using `github.com/yuin/goldmark` inside `internal/doc`.

## Key Facts
- **Native storage**: no shadow document, no HTML round-trip, and no Typst source standing in for the note; editing writes the exact Markdown bytes back to disk.
- **Supported syntax**: CommonMark plus the subset bound by templates—headings, paragraphs, lists, quotes, tables, fenced code, links, images, and footnotes.
- **Read safety cap**: files larger than `1 MiB` are refused with a message reporting their size.
- **Isolation**: `internal/doc` handles directory walking, title sniffing, read capping, and Goldmark block conversion without importing [[GoGPU]].
- **Title handling**: the first ATX heading is used as the document title and shown once (not duplicated between list/header and body); long lines wrap within the reading column.
- **Media and diagrams**: local images render inline at native size (shrunk only if wider than the column); `mermaid` code fences render via `mmdc` (falling back to the raw fence with `mmdc not installed` if unavailable).
- **Test fixture**: `testdata/membrane-assay.md` (a multi-hundred-line Markdown file with a >40-character first heading) verifies title sniffing and block rendering.

## How It Works
Because `gogpu/ui` has no rich-text widget, `internal/doc` parses Markdown bytes with Goldmark into structured blocks, which `internal/ui` draws onto a scrollable GPU canvas. Source code files (`.ts`, `.go`, etc.) bypass Markdown parsing and render with syntax highlighting instead.

## Relationships
- Parsed blocks are rendered in [[UI Layout and Surfaces]].
- Edited in place via [[Lens Editor]].
- Converted to `body.typ` during report compilation in [[Single-Compile Rendering]].
