---
title: lens.toml
type: entity
sources: [doc:///export-plan.md]
related: [entities/typst.md, entities/library-and-projects.md, topics/single-compile-rendering.md, topics/report-brand-packs.md]
updated: 2026-11-28
---

# lens.toml

For project-template compilation, `lens.toml` in the project root selects the Typst template and can supply title, author, and date. `internal/report/profile.go` loads these values and lists available designs. Use the [template example](../../doc/export-plan.md#project-typst-templates) to configure a project.

Named reports have their own IDs, ordered file lists, and template selections in `.pairti-reports.json`. User-installed brand assets belong in the Lens configuration directory. [Report brand packs](../topics/report-brand-packs.md)
