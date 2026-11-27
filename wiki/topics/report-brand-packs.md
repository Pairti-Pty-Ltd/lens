---
title: Report Brand Packs
type: topic
sources: [beads]
related: [entities/typst.md, entities/lens-toml.md, topics/single-compile-rendering.md]
updated: 2026-11-28
---

# Report Brand Packs

Introduced in `lens-c9n`, report brand packs externalize organisation-specific PDF and Word (`.docx`) report branding out of the Lens binary into user configuration. Clean installations export neutral PDF and Word documents using the built-in `default` brand without requiring any installed brand pack.

## Details

### Pack Discovery & Layout
- **Location**: `os.UserConfigDir()/pairti-lense/brands/<id>/` (beside the main Lens configuration in `cmd/pairti-lense/main.go`).
- **Stable IDs**: The directory name supplies the stable report template/brand ID (`^[a-z0-9][a-z0-9_-]*$`). `default` is reserved for the built-in neutral brand.
- **Manifest (`brand.json`)**: Versioned (`"version": 1`) manifest parsed by `internal/report/brand.go` with:
  - Required fields: `version` (`1`), non-empty `label`, pack-relative `logo` and `reference` (`.docx`) filenames, alphanumeric `stylePrefix` (default `"Report"`), and `logoWidth` (`1`–`150` mm, default `30`).
  - Palette: six-digit hex colours (`ink`, `accent`, `muted`, `h1`, `h2`, `tableInk`) with neutral defaults.
  - Typography & defaults: `bodyFont`, `headFont`, `subtitle`, `author`, `licence`, `disclosure`, `header`, plus an optional pack `fonts/` directory passed to Typst.

### PDF & Word Export Pipelines
- **Typst PDF (`internal/report/brand/brand.typ`, `compile.go`, `export.go`)**: Embedded `brand.typ` is generalized and driven by `--input` fields from `Brand.typstInputs(overrides)`. Pack fields act as defaults; non-empty document frontmatter/request fields override them. `stageBrand` copies the pack's logo to `logo.png` in the compile work directory and exposes `<pack>/fonts/` if present.
- **Pandoc Word (`internal/report/brand/docx.lua`, `docx.go`)**: For non-`default` brands, `exportDocx` stages `logo.png`, `reference.docx`, and the embedded generalized `docx.lua` filter, passing `--reference-doc=reference.docx` and `--lua-filter=filter.lua` plus environment variables (`LENS_STYLE_PREFIX`, `LENS_TABLE_INK`, `LENS_BODY_FONT`, `LENS_LOGO_WIDTH`, `LENS_LOGO_ALT`). For `default`, Pandoc runs without a reference doc or Lua filter to produce a neutral `.docx`.
- **Target suffixing**: `ExportRequest.Brand` supports target suffixes such as `<id>.docx` alongside explicit `Target` (`pdf` or `docx`).

## Pitfalls
- **Path traversal & asset validation**: `loadBrand` and `stageAsset` open each pack directory with `os.OpenRoot(dir)` and enforce `fs.ValidPath` / `filepath.IsLocal` plus regular-file checks so `logo` and `reference` cannot escape the pack directory.
- **Broken pack resilience**: `Brands()` catches invalid or incomplete packs during enumeration and surfaces them as `<id> (repair pack)` rather than failing startup; `loadBrand` returns an actionable error pointing at the missing/invalid `brand.json` path when exporting.

## See Also
- [[Typst]]
- [[Single-Compile Rendering]]
- [[lens.toml]]
