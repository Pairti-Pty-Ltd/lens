---
title: Single-Compile Rendering
type: topic
sources: [doc:///concept.md, doc:///export-plan.md, doc:///viewer-plan.md, beads]
related: [entities/typst.md, entities/lens-toml.md, entities/pairti-lens.md, topics/report-brand-packs.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# Single-Compile Rendering

Publishing in [[Pairti Lens]] is built on a single-compile guarantee: the project, report, ordered sources, and chosen template determine the publication, and PDF preview and PDF download come from the exact same [[Typst]]-compiled PDF so every page break, table, and image matches.

## Details
- **Ownership matrix** (`doc/export-plan.md`):
  - **Document selection & order**: owned by the project's saved report in `.pairti-reports.json`, which includes listed files in sequence. Because two projects may share report names, `(project path, report ID)` stay paired from menu selection through export.
  - **PDF page design**: owned by the [[Typst]] template (type, margins, headings, page breaks).
  - **Branding**: a neutral `default` ships with Lens; user-installed [[Report Brand Packs]] supply shared identity, assets, and Word reference styles.
  - **PDF preview**: derived from the exact PDF produced by that export—desktop rasterizes pages with `pdftoppm`, and browser library mode previews and downloads the same HTTP response.
  - **Editable Word file (`.docx`)**: produced by the Pandoc DOCX exporter using the selected brand's Word styles (check pagination in Word when pagination matters).
- **Project Typst templates**:
  - Place a custom design in `templates/brief.typ` (selected in [[lens.toml]]) or root `template.typ`; `default` selects the built-in design.
  - Lens converts the note to `body.typ`, passes `title`, `author`, and `date` via `sys.inputs`, and writes `out/report.pdf`.
  - The export dialog also supports named output files and installed brand packs.
- **Checking a publication** (`doc/export-plan.md`, `doc/viewer-plan.md`):
  - Export two notes with visibly different headings in a chosen order, then compare section order, images, and page breaks between preview and saved PDF; repeat after changing the template.
  - In browser library mode, save edits before exporting because the server reads the saved checkout.
  - Distinguish preview generation failure from PDF compile failure.
- **Browser library server-side publishing (`lens-77q.2`)**: `cmd/lens-library` (`POST /api/report/export`) reuses `internal/report` on saved tracked files inside a confined per-job directory (`LibrarySnapshot`), serving one PDF blob to both the browser `<dialog>` `<iframe>` preview and download link, and running Pandoc with `--sandbox` + `snapshot-media.lua` for `.docx` exports. See [[Browser WASM Editor]].

## Decisions & Tradeoffs
- **Single PDF for preview and download**: Avoids HTML/web-view reflow divergence between what the writer previews and what they publish.
- **Separate Word exporter**: Typst owns PDF layout; Pandoc owns `.docx` export with brand reference styles rather than trying to force Word pagination through Typst.
- **Externalized brand packs (`lens-c9n`)**: Organisation-specific branding lives in `os.UserConfigDir()/pairti-lense/brands/<id>/` rather than inside the binary, keeping clean installs neutral. See [[Report Brand Packs]].

## See Also
- [[Typst]]
- [[lens.toml]]
- [[Report Brand Packs]]
- [[Browser WASM Editor]]
- [[Viewer Verification Plan]]
