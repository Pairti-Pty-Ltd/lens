---
title: Single-Compile Rendering
type: topic
sources: [doc:///concept.md, doc:///export-plan.md, doc:///viewer-plan.md, doc:///todo.md, beads]
related: [entities/typst.md, entities/lens-toml.md, entities/pairti-lens.md, topics/build-roadmap.md, topics/report-brand-packs.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# Single-Compile Rendering

Single-compile rendering is the core architectural guarantee of [[Pairti Lens]]: one [[Typst]] compile produces both the exported PDF (`out/report.pdf`) and the page images shown inside the application window. Two separate render paths are treated as a bug.

## Details
- **Compile pipeline**:
  1. The open Markdown note is converted into `body.typ`.
  2. Pairti Lens resolves the active template from [[lens.toml]] (`template = "default"` or missing → built-in A4 page; otherwise resolved under `templates/` then project root, matching `templates/*.typ` or root `template.typ`).
  3. The template is used as the compile root; if it does not include `body.typ`, the include is injected automatically.
  4. Metadata fields are passed as `typst compile --input title=... --input author=... --input date=...`. Empty `title` defaults to the first heading; empty `date` defaults to the compile date. Templates read inputs via `sys.inputs.at`.
  5. Typst writes `out/report.pdf`, and compiler errors stay attached to the job.
  6. `pdftoppm` rasterizes `out/report.pdf` into page images for the [[GoGPU]] canvas. If `pdftoppm` is absent, `out/report.pdf` is still written. If `typst` is absent, the status line names the missing binary.
- **Template swapping**:
  - `:template` lists available template names.
  - `:template <name>` writes the new name to [[lens.toml]] and recompiles so the window pages and `out/report.pdf` update together.
  - Clicking **Export** compiles with whatever `lens.toml` names; clicking **Formatted** returns to the Markdown note.
- **Verification**:
  - Automated tests substitute a stand-in `typst` binary that records CLI flags, `--input` fields, template text, PDF path, and template-swap recompiles.
  - Visual parity between `out/report.pdf` and window pages requires a real `typst` binary on `PATH` and manual inspection.

## Decisions & Tradeoffs
- **Why not a web-view reflow?** Browser/HTML preview reflows text differently from paged print layout and breaks pixel-identity with the exported report.
- **Why not Pandoc as layout engine?** Typst owns all page breaking, typography, math, and tables. Pandoc is not used for layout (and is not currently used for conversion either, though left open if a richer Markdown converter is needed later).
- **Why Word and HTML are deferred**: Typst's HTML export is experimental and reflows in browsers; Word has no Typst target. Both are deferred and explicitly labelled as separate output files that do not share the PDF's pixels.
- **One active template per project**: Multi-profile export configurations (as in Zettlr) are deferred until single-template compile and swap are proven.
- **Externalized report brand packs (`lens-c9n`)**: Branded PDF and Pandoc `.docx` exports load versioned user-installed packs from `os.UserConfigDir()/pairti-lense/brands/<id>/` rather than embedding private assets in the binary, with a built-in neutral `default` brand on clean installations. See [[Report Brand Packs]].
- **Browser library server-side publishing (`lens-77q.2`)**: `cmd/lens-library` (`POST /api/report/export`) reuses `internal/report` on saved tracked files inside a confined per-job directory (`LibrarySnapshot`), serving one PDF blob to both the browser `<dialog>` `<iframe>` preview and download link, and running Pandoc with `--sandbox` + `snapshot-media.lua` for `.docx` exports. See [[Browser WASM Editor]].

## See Also
- [[Typst]]
- [[lens.toml]]
- [[Build Roadmap]]
- [[Report Brand Packs]]
- [[Browser WASM Editor]]
