# Publishing design

Select a project and a report before exporting. Two projects may have reports with the same name. Keep the project path and report ID together from menu selection to export.

| Writer’s choice | What owns it |
| --- | --- |
| Which documents appear, and in what order | The project's saved report in `.pairti-reports.json`. Include the listed files in sequence. |
| Page design for a project template | The Typst template controls type, margins, headings, and page breaks. |
| Branding for a PDF or Word export | A neutral default ships with Lens. User-installed brand packs supply shared identity, assets, and Word reference styles. |
| What the PDF preview shows | The PDF produced by that export. Desktop rasterizes its pages with `pdftoppm`. Browser library mode previews and downloads the same response. |
| An editable Word file | The Pandoc DOCX exporter uses the selected brand’s Word styles. Check the document in Word when pagination matters. |

## Project Typst templates

For the project-template compile path, put a design in `templates/brief.typ` and select it in the project’s `lens.toml`. The project can also supply a root `template.typ`. The built-in design is available as `default`.

```toml
template = "brief.typ"
title = "Assay report"
author = "Research team"
date = "2026-11-22"
```

Lens converts the note to `body.typ`. The selected template receives the body and can read title, author, and date through `sys.inputs`. This path writes `out/report.pdf`. The export dialog also supports named output files and installed brands. [Brand pack setup](../README.md#report-brand-packs)

## Check a publication

Use two notes with visibly different headings. Select their order, export, and inspect the first heading on each section and the page breaks in the preview and saved PDF. Change the template and repeat. In browser library mode, save edits before exporting. The server reads the saved checkout. If preview generation fails, distinguish that failure from failure to compile the PDF.
