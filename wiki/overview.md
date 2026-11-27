---
title: Pairti Lens Wiki Overview
type: overview
sources: [doc:///concept.md, doc:///viewer-plan.md, doc:///export-plan.md, doc:///editor-keymap.md]
related: [entities/pairti-lens.md, entities/gogpu.md, entities/typst.md, entities/lens-editor.md, topics/single-compile-rendering.md, topics/viewer-verification-plan.md, topics/macos-native-integration.md, topics/browser-wasm-editor.md, topics/report-brand-packs.md]
updated: 2026-11-28
---

# Pairti Lens — Project Overview

[[Pairti Lens]] (`pairti-lense`) is a pure-Go (`CGO_ENABLED=0`) writing and publishing workspace for navigating multiple project folders, editing native Markdown files on disk, and publishing reports with [[Typst]] (PDF) and Pandoc (Word) across desktop and browser targets.

## North Stars & Core Architecture
- **Project-folder ownership & cheap switching**: [[Library and Projects]] keeps multiple ordinary folders open in one window, disambiguates same-titled notes by project and relative path, provides tree filtering plus lexical body search, and leaves files intact when a folder is removed.
- **Single editing history**: [[Goldmark and Markdown]] uses Goldmark to understand document structure while preserving unfamiliar markup and Git-readable diffs. [[Lens Editor]] shares one buffer, cursor, selection, and undo history across Raw and Formatted views with Vim-inspired modal keybindings ([[Editor Keymap and Modes]]).
- **Single-compile publishing**: [[Single-Compile Rendering]] pairs `(project path, report ID)` from `.pairti-reports.json` with [[Typst]] templates ([[lens.toml]]) and installable [[Report Brand Packs]] so PDF preview and download come from the exact same PDF, while Pandoc handles editable `.docx` exports.
- **Shared desktop & browser workspace**: Built on [[GoGPU]] (`gogpu/ui` `v0.1.55`, `gogpu` `v0.54.1`) with custom tree/split surfaces ([[UI Layout and Surfaces]]). Desktop integrates with macOS menus and Finder via `goffi` ([[macOS Native Integration]]) and includes the desktop terminal panel ([[Embedded Shell]]); browser library mode compiles the same `cmd/pairti-lense` shell to WASM backed by `cmd/lens-library` ([[Browser WASM Editor]]).
- **Verification discipline**: [[Viewer Verification Plan]] defines an 11-surface verification matrix combining automated canvas assertions with recorded live-window checks.

## Key Starting Points
- **Entities**: [[Pairti Lens]], [[GoGPU]], [[Typst]], [[lens.toml]], [[Library and Projects]], [[Goldmark and Markdown]], [[Lens Editor]], [[Embedded Shell]]
- **Topics**: [[Single-Compile Rendering]], [[Viewer Verification Plan]], [[Editor Keymap and Modes]], [[UI Layout and Surfaces]], [[macOS Native Integration]], [[Browser WASM Editor]], [[Report Brand Packs]]
