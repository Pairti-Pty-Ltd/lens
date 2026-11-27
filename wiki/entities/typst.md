---
title: Typst
type: entity
sources: [doc:///concept.md, doc:///export-plan.md, doc:///viewer-plan.md, doc:///todo.md]
related: [entities/lens-toml.md, entities/pairti-lens.md, topics/single-compile-rendering.md, topics/build-roadmap.md]
updated: 2026-11-27
---

# Typst

Typst is the external CLI layout compiler that [[Pairti Lens]] invokes to turn project Markdown into pixel-identical report pages and PDF exports. In Pairti Lens, Typst is strictly a layout and template engine—not a second authoring language for users' notes.

## Key Facts
- **External CLI**: invoked as the `typst` binary on `PATH`; if missing, the status line reports the missing binary rather than crashing.
- **Role boundary**: math, typography, margins, headings, tables, and page breaking stay inside the Typst compiler and are never reimplemented in Go.
- **Template files**: `.typ` files are templates and partials (`templates/*.typ` or root `template.typ`) and are excluded from the Markdown document navigation list.
- **Compilation pipeline**:
  1. The open Markdown note is converted into `body.typ`.
  2. The chosen template is the compile root; if it does not include `body.typ`, the include is added automatically.
  3. Metadata fields are passed via `typst compile --input title=... --input author=... --input date=...`, which templates read with `sys.inputs.at`.
  4. Output is written to `out/report.pdf`.
  5. `pdftoppm` rasterizes `out/report.pdf` into page images displayed in the window (if `pdftoppm` is absent, the PDF is still written).
- **Testing**: automated tests use a stand-in `typst` binary that records CLI arguments and template text; visual window verification requires a real `typst` binary on `PATH`.

## How It Works
A single `typst compile` invocation produces `out/report.pdf`, and the viewer displays rasterized images of those exact pages. Swapping templates updates [[lens.toml]] and triggers a fresh compile so both the on-screen view and exported PDF change together.

## Relationships
- Configured per project by [[lens.toml]].
- Central to [[Single-Compile Rendering]] and Builds 2–3 in [[Build Roadmap]].
- Gated behind Phase 1 viewer sign-off in [[Viewer Verification Plan]].
