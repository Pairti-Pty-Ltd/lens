---
title: Typst
type: entity
sources: [doc:///export-plan.md]
related: [entities/lens-toml.md, entities/pairti-lens.md, topics/single-compile-rendering.md, topics/report-brand-packs.md]
updated: 2026-11-28
---

# Typst

Typst lays out PDF reports. The export path supplies ordered content, assets, and the selected design. The resulting PDF owns page size, typography, line wrapping, and pagination.

Desktop preview rasterizes that PDF with `pdftoppm`. Browser library mode presents the PDF returned by the server. Keep compiler diagnostics separate from errors opening or rasterizing an existing PDF. [Publishing design](../../doc/export-plan.md)
