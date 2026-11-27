---
title: lens.toml
type: entity
sources: [doc:///concept.md, doc:///export-plan.md, doc:///todo.md]
related: [entities/typst.md, entities/library-and-projects.md, topics/single-compile-rendering.md]
updated: 2026-11-27
---

# lens.toml

`lens.toml` is an optional per-project TOML configuration file located in a project's root directory. It defines the project's display name, active [[Typst]] report template, and document metadata fields passed to the compiler.

## Key Facts
- **Location**: project root directory (not in the user config dir).
- **Schema fields**:
  ```toml
  template = "brief.typ"
  title = "Assay report"
  author = "Pairti"
  date = "2026-11-22"
  ```
- **Defaults and fallbacks**:
  - **Project name**: read from `lens.toml` if present; otherwise falls back to the folder name.
  - **Template**: missing `lens.toml` or `template = "default"` uses the built-in A4 template; a filename resolves under `templates/` first, then the project root (`templates/*.typ` or root `template.typ`).
  - **Title**: empty `title` falls back to the note's first Markdown heading.
  - **Date**: empty `date` falls back to the day of the compile.
- **One active template per project**: multiple export profiles are deferred; swapping templates via `:template <name>` rewrites `lens.toml` and recompiles.

## How It Works
When compiling a report, Pairti Lens reads `lens.toml`, resolves the template path, fills empty `title`/`date` defaults, and passes `title`, `author`, and `date` to `typst compile --input ...`. Both the built-in template and custom project templates read these values via `sys.inputs.at`.

## Relationships
- Drives template selection and metadata inputs for [[Typst]].
- Provides the project display name for [[Library and Projects]].
- Updated by `:template` commands in [[Single-Compile Rendering]].
