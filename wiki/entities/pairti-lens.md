---
title: Pairti Lens
type: entity
sources: [doc:///concept.md, doc:///build-0.md, doc:///build-1.md, doc:///viewer-plan.md, doc:///todo.md, beads]
related: [entities/gogpu.md, entities/typst.md, entities/library-and-projects.md, entities/lens-editor.md, topics/build-roadmap.md]
updated: 2026-11-27
---

# Pairti Lens

Pairti Lens (Go module `pairti-lense`) is a pure-Go desktop application for navigating multiple project folders, writing native Markdown files on disk, and viewing or exporting template-driven reports laid out by Typst. Its core product decision is: which report, in which project, and whether the rendered pages match the template.

## Key Facts
- **Four core features** (narrowed from Zettlr, which serves as reference rather than spec):
  1. **Multi-project navigation** — multiple project folders stay open in one window simultaneously.
  2. **Native Markdown** — edits write directly to `.md`/`.markdown` files on disk with no shadow document or HTML round-trip.
  3. **Typst viewing** — report views display the exact pages from a single Typst compile; export writes that same PDF.
  4. **Template-driven reports** — a Typst template fixes page size, typography, margins, and layout so identical sources and templates produce identical pixels.
- **Explicit non-goals**: no Electron or web-view shell, no Pandoc as layout engine, no writing reports directly in Typst, no Zettelkasten IDs/tags/snippets/writing statistics as defining features, and no per-format viewer proliferation.
- **Module layout** (established in Build 0):
  - `main.go` — window, project tree, header, folder panel
  - `internal/library/` — add, remove, and persist project paths
  - `internal/doc/` — directory walk, title sniffing, read cap, Goldmark parsing (no GoGPU dependency)
  - `internal/ui/` — scrollable Markdown page surface
  - `testdata/` — long-title fixture (`testdata/membrane-assay.md`)
- **Executable layout** (updated in `lens-qr5`, commit `79c786a`):
  - `cmd/pairti-lense/` — houses all 22 `main*.go` desktop entry-point files and command tests; the repository root contains no Go files, while `internal/` packages remain in place.
  - Build and run directly via `CGO_ENABLED=0 go build -o build/pairti-lense ./cmd/pairti-lense` (or `make binary`) and `CGO_ENABLED=0 go run ./cmd/pairti-lense`.

## How It Works
Pairti Lens opens a single native GPU-rendered window using [[GoGPU]] (`CGO_ENABLED=0`, Go 1.25+). Users add project folders to a persisted library, browse Markdown documents by their first heading, edit them in place via [[Lens Editor]], and compile the active project's Markdown through [[Typst]] using the template specified in [[lens.toml]] to produce `out/report.pdf` and matching rasterized preview pages.

## Relationships
- Built on [[GoGPU]] for windowing, widgets, and GPU drawing.
- Uses [[Goldmark and Markdown]] for parsing and rendering Markdown documents.
- Compiles reports via [[Typst]] configured by [[lens.toml]].
- Organizes open folders via [[Library and Projects]].
- Progresses through milestones tracked in [[Build Roadmap]] and [[Viewer Verification Plan]].
