---
title: Wiki Index
type: overview
sources: [doc:///concept.md, doc:///build-0.md, doc:///build-1.md, doc:///viewer-plan.md, doc:///export-plan.md, doc:///editor-keymap.md, doc:///todo.md]
related: [overview.md]
updated: 2026-11-28
---

# Wiki Index

Catalog of all wiki pages for Pairti Lens.

## Overview
- [[Pairti Lens Wiki Overview]] (`overview.md`) — High-level synthesis of Pairti Lens architecture, workflows, and verification gates.

## Entities
- [[Pairti Lens]] (`entities/pairti-lens.md`) — Pure-Go desktop app for multi-project Markdown writing and Typst report viewing.
- [[GoGPU]] (`entities/gogpu.md`) — Pinned pure-Go GPU UI stack (`gogpu/ui` v0.1.55) and the Build 0 stay-or-switch decision.
- [[Typst]] (`entities/typst.md`) — External layout compiler invoked to generate `out/report.pdf` and rasterized report pages.
- [[lens.toml]] (`entities/lens-toml.md`) — Per-project TOML configuration for project name, active template, and report metadata.
- [[Library and Projects]] (`entities/library-and-projects.md`) — Persisted multi-folder library (`library.json`), file discovery, title sniffing, and filtering.
- [[Goldmark and Markdown]] (`entities/goldmark-and-markdown.md`) — Native `.md`/`.markdown` storage, CommonMark subset, `1 MiB` read cap, and `internal/doc` parsing.
- [[Lens Editor]] (`entities/lens-editor.md`) — App-owned modal editor shared across Formatted and Raw views with dirty-buffer protection.
- [[Embedded Shell]] (`entities/embedded-shell.md`) — Parked subprocess command runner panel awaiting Phase 1 viewer sign-off.

## Topics
- [[Single-Compile Rendering]] (`topics/single-compile-rendering.md`) — One-compile Typst PDF + `pdftoppm` page rendering, template swapping, and excluded formats.
- [[Build Roadmap]] (`topics/build-roadmap.md`) — Builds 0–3 milestones, GoGPU decision, and remaining backlog items.
- [[Viewer Verification Plan]] (`topics/viewer-verification-plan.md`) — Canvas-level Test vs Window check policy and Phases 1–3 gating.
- [[Editor Keymap and Modes]] (`topics/editor-keymap-and-modes.md`) — View/Edit modes, motions, operators, text objects, Visual selections, Ex commands, and `Ctrl-W` behavior.
- [[UI Layout and Surfaces]] (`topics/ui-layout-and-surfaces.md`) — 4-line header budget, custom tree/split surfaces, toolbar toggles, tabs, outline, and footer.
- [[macOS Native Integration]] (`topics/macos-native-integration.md`) — Pure-Go AppKit menus via `goffi`, `OnUpdate` thread ownership, shortcut deduplication, Finder `Open With`, and hierarchical Reports menus.
- [[Browser WASM Editor]] (`topics/browser-wasm-editor.md`) — Shared `GOOS=js GOARCH=wasm` Lens shell (`cmd/pairti-lense`), Git-backed library server (`cmd/lens-library`), Canvas2D paint overlay, resize span caching, and JS/Blob file I/O.
- [[Report Brand Packs]] (`topics/report-brand-packs.md`) — User-configured `brand.json` report packs under `os.UserConfigDir()/pairti-lense/brands/<id>/`, neutral `default` fallback, and generalized Typst/Pandoc export pipelines.
