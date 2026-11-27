---
title: Build Roadmap
type: topic
sources: [doc:///concept.md, doc:///build-0.md, doc:///build-1.md, doc:///todo.md]
related: [entities/pairti-lens.md, entities/gogpu.md, entities/typst.md, entities/embedded-shell.md, topics/viewer-verification-plan.md]
updated: 2026-11-27
---

# Build Roadmap

[[Pairti Lens]] is sequenced into four incremental builds (Builds 0–3), each proving one core capability before layering the next, followed by a prioritized backlog of viewer, export, and tooling features.

## Details
- **Build 0 — Multi-project Markdown viewer**:
  - Proves: `CGO_ENABLED=0` window on macOS, persisted multi-folder library (`library.json`) with absent-folder handling, recursive `*.md`/`*.markdown` discovery skipping `.git`/`out/`, ATX first-heading title sniffing (with `1 MiB` read cap and `testdata/membrane-assay.md` fixture), Goldmark block rendering, sidebar filtering, and the [[GoGPU]] stay-or-switch decision.
  - Outcome: **Stay on GoGPU** (`gogpu/ui` `v0.1.55`), with custom tree and split surfaces replacing the toolkit's built-ins.
- **Build 1 — Edit and save**:
  - Proves: in-place editing of the selected Markdown file in an app-owned buffer (typing, Enter, Backspace, Delete, arrows), saving back to the same path with sidebar title refresh, dirty-file protection against switching, Escape-to-discard, and `go test ./...` coverage including a UI test that types and reads drawn text back.
  - Header adds a single stateful button (`Edit` → `Save` → `Done`) and an `Unsaved` marker on the format line.
- **Build 2 — Single Typst template & export** (current report slice in `todo.md`):
  - Proves: one Typst template (`template.typ`, `lens.toml`, or built-in A4) on one project, one compile to `out/report.pdf`, and rasterized page viewing via `pdftoppm`.
- **Build 3 — Template swap**:
  - Proves: switching the project to another `.typ` template updates both on-screen pages and `out/report.pdf` together.
- **Remaining backlog (`doc/todo.md`)**:
  1. Unwired controls that currently only echo their name to the status line: New document, tags, back/forward, markup inserts (code, link, image, table, footnote), and the Settings gear (theme/fonts still load from `theme.json`).
  2. In-file and project-wide body search (sidebar search only filters the tree).
  3. Editor gaps tracked in `todo.md`: undo, pointer selection drag, and Visual mode verification.
  4. Word and HTML export (explicitly labelled as non-pixel-identical secondary outputs).
  5. Later requests: additional diagram renderers on the Mermaid registry, standalone PDF/HTML viewer, language-model agents, and the [[Embedded Shell]].

## Decisions & Tradeoffs
- **Prove the shell before templates**: Evaluating [[GoGPU]] against real project folders in Build 0 prevented investing in template work on an unsuitable UI toolkit (with Gio held as fallback).
- **Gate shell and agents behind viewer sign-off**: All Phase 1 manual window checks in [[Viewer Verification Plan]] must pass before resuming [[Embedded Shell]] or agent work.

## See Also
- [[Pairti Lens]]
- [[Viewer Verification Plan]]
- [[Single-Compile Rendering]]
