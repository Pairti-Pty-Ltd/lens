---
title: Library and Projects
type: entity
sources: [doc:///concept.md, beads]
related: [entities/pairti-lens.md, entities/lens-toml.md, entities/goldmark-and-markdown.md, topics/ui-layout-and-surfaces.md, topics/browser-wasm-editor.md, topics/macos-startup-performance.md]
updated: 2026-11-28
---

# Library and Projects

Add a project folder to keep it alongside other folders in one window. The desktop library stores paths in the user configuration’s `library.json`. Removing an entry leaves its contents on disk. A missing folder keeps its place in the list.

The sidebar reads one directory at a time. It can show Markdown, Typst, and recognised source files. Hidden and Git-ignored entries have separate controls. Generated directories such as `.git` and `node_modules` stay out of the tree. Small listings use a Markdown heading as the row title when available.

Use the tree filter for project names, titles, and relative paths. Use body search when the writer remembers a passage. Browser library mode exposes a narrower set of files from one checkout. [Browser library](../topics/browser-wasm-editor.md)

## Performance Notes
- Sidebar Git-ignore filtering in `internal/doc/list.go` (`readDir`) batches subdirectories and files into a single `dropIgnored(projectRoot, out)` call per directory (`864be61`), halving synchronous Git subprocesses during workspace startup and rescan. See [[macOS Startup Performance]].
