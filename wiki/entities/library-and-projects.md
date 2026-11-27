---
title: Library and Projects
type: entity
sources: [doc:///concept.md, doc:///build-0.md, doc:///viewer-plan.md]
related: [entities/pairti-lens.md, entities/lens-toml.md, entities/goldmark-and-markdown.md, topics/ui-layout-and-surfaces.md]
updated: 2026-11-27
---

# Library and Projects

The Library is the persisted list of open project folders in [[Pairti Lens]], allowing users to work across multiple projects in a single window without closing one to open another. Each Project corresponds to one filesystem folder containing Markdown documents and optional templates.

## Key Facts
- **Persistence**: stored as a JSON list of absolute folder paths in `os.UserConfigDir()/pairti-lense/library.json` (never inside a project folder); written on quit.
- **Add / remove behavior**:
  - Adding a project uses the macOS folder picker via `osascript` (keeping `CGO_ENABLED=0`) or a path field on other OSes.
  - Removing a project drops the path from the library without deleting files on disk.
  - Missing paths remain listed and render as absent rather than disappearing silently.
- **Document discovery**:
  - Recursively walks project folders for `*.md` and `*.markdown` (with later support for known source files such as `.ts` and `.go` in the viewer).
  - Always skips `.git`, `out/`, and `node_modules`.
  - Excludes `.typ` files from the document list.
  - Sidebar toolbar toggles control visibility of dot-leading (Hidden) and gitignored (Ignored) files.
- **Title sniffing**: each row displays the first ATX heading or, if absent, the filename; no YAML frontmatter parser is used. Titles are cached per `(path, size, mtime)` with no index database.
- **Filtering**: a single sidebar filter field matches project names, document titles, and relative paths without scanning file bodies; clearing it restores the full tree.

## How It Works
`internal/library` manages adding, removing, and persisting absolute paths to `library.json`. `internal/doc` walks each project directory, filters excluded directories, and sniffs titles. Path segments longer than ~40 characters are middle-truncated in the header while document titles are kept whole.

## Relationships
- Project names can be overridden by [[lens.toml]].
- Document walking and title sniffing are implemented in `internal/doc` under [[Goldmark and Markdown]].
- Sidebar presentation and Hidden/Ignored toggles are described in [[UI Layout and Surfaces]].
