---
title: Pairti Lens Wiki Overview
type: overview
sources: [doc:///concept.md, doc:///build-0.md, doc:///build-1.md, doc:///viewer-plan.md, doc:///export-plan.md, doc:///editor-keymap.md, doc:///todo.md]
related: [entities/pairti-lens.md, entities/gogpu.md, entities/typst.md, entities/lens-editor.md, topics/single-compile-rendering.md, topics/viewer-verification-plan.md, topics/build-roadmap.md, topics/macos-native-integration.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# Pairti Lens — Project Overview

[[Pairti Lens]] (`pairti-lense`) is a pure-Go (`CGO_ENABLED=0`) desktop application for navigating multiple project folders, writing native Markdown files on disk, and viewing or exporting pixel-identical reports compiled with [[Typst]].

## Core Architecture
- **UI Shell**: Built on [[GoGPU]] (`gogpu/ui` `v0.1.55`, `gogpu/gogpu`, `gogpu/gg`, Go 1.25+), chosen and confirmed in Build 0 with custom project tree and split-pane surfaces. See [[UI Layout and Surfaces]].
- **Multi-Project Navigation**: [[Library and Projects]] persists open folder paths in `library.json`, discovers Markdown and source files, sniffs first-heading titles without a database, and filters by name/title/path.
- **Native Markdown & Editing**: [[Goldmark and Markdown]] parses `.md`/`.markdown` files in `internal/doc` (with a `1 MiB` read cap and no GoGPU dependency), while [[Lens Editor]] provides a shared modal buffer across Formatted and Raw views. See [[Editor Keymap and Modes]].
- **Single-Compile Reports**: [[Single-Compile Rendering]] invokes [[Typst]] with project settings from [[lens.toml]] to produce `out/report.pdf` and rasterizes those exact pages via `pdftoppm` for the viewer.
- **Platform & Browser Targets**: [[macOS Native Integration]] wires pure-Go AppKit menus and Finder `Open With` via `goffi` on the `OnUpdate` thread, while [[Browser WASM Editor]] builds a standalone `GOOS=js GOARCH=wasm` Raw editor in `cmd/lens-wasm`.

## Current Status & Gates
- **Builds 0 & 1** established the GoGPU shell, multi-project viewer, and in-place editing; **Builds 2 & 3** cover Typst report compilation and template swapping. See [[Build Roadmap]].
- **Release gate**: [[Viewer Verification Plan]] requires manual window sign-off on all Phase 1 viewer items (search, toggles, Markdown/source/diagram rendering, vi caret, tabs/split, outline, footer, scroll) before moving to Phase 2 features or resuming the parked [[Embedded Shell]].

## Key Starting Points
- Entities: [[Pairti Lens]], [[GoGPU]], [[Typst]], [[lens.toml]], [[Library and Projects]], [[Goldmark and Markdown]], [[Lens Editor]], [[Embedded Shell]]
- Topics: [[Single-Compile Rendering]], [[Build Roadmap]], [[Viewer Verification Plan]], [[Editor Keymap and Modes]], [[UI Layout and Surfaces]], [[macOS Native Integration]], [[Browser WASM Editor]]
