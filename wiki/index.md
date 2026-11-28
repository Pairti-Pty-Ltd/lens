---
title: Wiki Index
type: overview
sources: [doc:///concept.md, doc:///viewer-plan.md, doc:///export-plan.md, doc:///editor-keymap.md]
related: [overview.md]
updated: 2026-11-28
---

# Wiki Index

Catalog of all wiki pages for Pairti Lens.

## Overview
- [[Pairti Lens Wiki Overview]] (`overview.md`) — High-level synthesis of Pairti Lens north stars, architecture, publishing, and verification.

## Entities
- [[Pairti Lens]] (`entities/pairti-lens.md`) — Pure-Go desktop and browser workspace for multi-project Markdown writing and report publishing.
- [[GoGPU]] (`entities/gogpu.md`) — Pinned pure-Go GPU UI stack (`gogpu/ui` `v0.1.55`, `gogpu` `v0.54.1`), custom surfaces, Darwin thread rules, and WASM Canvas2D overlay.
- [[Typst]] (`entities/typst.md`) — External layout compiler invoked to generate pixel-identical PDF previews and report exports.
- [[lens.toml]] (`entities/lens-toml.md`) — Per-project TOML configuration selecting a project Typst template and `sys.inputs` metadata.
- [[Library and Projects]] (`entities/library-and-projects.md`) — Ordinary project folders, persisted library list, tree/tag filtering, body search, and `.pairti-reports.json` reports.
- [[Goldmark and Markdown]] (`entities/goldmark-and-markdown.md`) — Native `.md`/`.markdown` source storage, Goldmark structural parsing, unfamiliar markup preservation, and syntax-highlighted source view.
- [[Lens Editor]] (`entities/lens-editor.md`) — App-owned modal editor shared across Raw and Formatted views with single undo history and conflict-safe saving.
- [[Local Agent MCP]] (`entities/local-agent-mcp.md`) — Desktop local MCP server (`document` and `apply_edit`) exposing the focused buffer and applying single-undo edits with stale-edit refusal.
- [[Embedded Shell]] (`entities/embedded-shell.md`) — Desktop-only in-app terminal surface (`//go:build !js`) with shared focus and clipboard routing.

## Topics
- [[Single-Compile Rendering]] (`topics/single-compile-rendering.md`) — Report ownership (`.pairti-reports.json`), project Typst templates, identical PDF preview/download, Pandoc `.docx` export, and publication checks.
- [[Viewer Verification Plan]] (`topics/viewer-verification-plan.md`) — 13-surface verification matrix separating automated canvas assertions from recorded live-window checks.
- [[Editor Keymap and Modes]] (`topics/editor-keymap-and-modes.md`) — View/Edit modes, motions, operators, text objects, Visual selections, Ex commands, and `Ctrl-W` behavior.
- [[UI Layout and Surfaces]] (`topics/ui-layout-and-surfaces.md`) — Shared sidebar, tabs, split panes, outline rail, single-line footer, and custom GoGPU surfaces.
- [[macOS Native Integration]] (`topics/macos-native-integration.md`) — Pure-Go AppKit menus via `goffi`, `OnUpdate` thread ownership, shortcut deduplication, Finder `Open With`, and hierarchical Reports menus.
- [[macOS Startup Performance]] (`topics/macos-startup-performance.md`) — Native launch-to-first-frame benchmarks, pre-`main` cold-launch overhead, and batched per-directory Git ignore filtering.
- [[Browser WASM Editor]] (`topics/browser-wasm-editor.md`) — Shared `GOOS=js GOARCH=wasm` Lens shell (`cmd/pairti-lense`), Git-backed library server (`cmd/lens-library`), tracked history/diffs, confined server-side report publishing/preview, Canvas2D paint overlay, and browser-vs-desktop boundaries.
- [[Report Brand Packs]] (`topics/report-brand-packs.md`) — User-configured `brand.json` report packs under `os.UserConfigDir()/pairti-lense/brands/<id>/`, neutral `default` fallback, and generalized Typst/Pandoc export pipelines.
