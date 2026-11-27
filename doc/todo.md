# What is left

The order and the exit checks are in `doc/viewer-plan.md`. The shell is parked until every window check in phase 1 of that plan has been done. A green test suite is not that check.

## 1. The report — this slice

One Typst template on the open project. One compile. The PDF is `out/report.pdf`. The window shows pages rasterised from that file, not a second layout.

Export on the toolbar runs it. Formatted returns to the note. A missing `typst` binary is named on the status line. `pdftoppm` turns that PDF into the page images. Without it, the PDF is still written.

The template is `template.typ` in the project, or the `template` key in `lens.toml`, or the built-in page. The note is turned into `body.typ` and included. The template owns page size and type. Pandoc is not involved.

## 2. Swap the template

A control that points the project at another `.typ` file and compiles again. The pages and `out/report.pdf` change together.

## 3. Word and HTML

Not the same pixels as the PDF. HTML is Typst's experimental HTML export, which the browser reflows. Word has no Typst target. Both stay labelled as a second file.

## 4. Controls that still only write their name

New document, tags, back, forward, and the inserts: code, link, image, table, footnote. The gear does not open a settings screen. Theme and fonts still come from `theme.json`.

Search filters the tree. It does not search inside files.

## 5. Editor

Undo. Dragging a selection. Vi has no visual mode.

## 6. Later, already asked for

Diagram types other than Mermaid, on the same registry. A viewer for a PDF or HTML file that is not the report. Language-model agents. The shell is a real process with colour codes stripped, not a full terminal emulator.
