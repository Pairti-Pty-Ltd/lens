#let title = sys.inputs.at("title", default: "Report")
#let subtitle = sys.inputs.at("subtitle", default: "")
#let author = sys.inputs.at("author", default: "")
#let issued = sys.inputs.at("date", default: "")
#let client = sys.inputs.at("client", default: "")
#let project = sys.inputs.at("project", default: "")
#let licence = sys.inputs.at("licence", default: "")
#let disclosure = sys.inputs.at("disclosure", default: "")
#let running = sys.inputs.at("header", default: "")
#let ink = rgb("#" + sys.inputs.at("ink", default: "1A1A1A"))
#let accent = rgb("#" + sys.inputs.at("accent", default: "4A5260"))
#let muted = rgb("#" + sys.inputs.at("muted", default: "6B7280"))
#let h1c = rgb("#" + sys.inputs.at("h1", default: "4A5260"))
#let h2c = rgb("#" + sys.inputs.at("h2", default: "4A5260"))
#let body-font = sys.inputs.at("body-font", default: "Arial")
#let head-font = sys.inputs.at("head-font", default: "Georgia")
#let logo-mm = int(sys.inputs.at("logo-width", default: "30")) * 1mm

#set text(font: body-font, size: 10.5pt, fill: ink)
#set par(leading: 0.65em, spacing: 0.9em)
#set page(
  paper: "a4",
  margin: (top: 2.4cm, bottom: 2.2cm, x: 2.2cm),
  header: context {
    if counter(page).get().first() > 1 {
      set text(size: 8pt, fill: muted)
      running
      v(4pt)
      line(length: 100%, stroke: 0.4pt + accent)
    }
  },
  footer: context {
    if counter(page).get().first() > 1 {
      set text(size: 8pt, fill: muted)
      title
      h(1fr)
      [Page ]
      counter(page).display()
      [ of ]
      str(counter(page).final().first())
    }
  },
)
#show heading.where(level: 1): it => {
  set text(font: head-font, size: 16pt, weight: "bold", fill: h1c)
  block(above: 1.4em, below: 0.6em, it.body)
}
#show heading.where(level: 2): it => {
  set text(font: head-font, size: 13pt, weight: "bold", fill: h2c)
  block(above: 1.1em, below: 0.45em, it.body)
}
#set table(stroke: 0.4pt + rgb("#D0D5DD"))
#show table.cell.where(y: 0): set text(weight: "bold")

#image("logo.png", width: logo-mm)
#v(8mm)
#text(font: head-font, size: 23pt, weight: "bold", fill: h1c)[#title]
#v(2mm)
#text(size: 12pt, fill: muted)[#subtitle]
#v(8mm)
#if client != "" [
  #text(size: 9pt)[Client: #client]
  #linebreak()
]
#if project != "" [
  #text(size: 9pt)[Project: #project]
  #linebreak()
]
#text(size: 9pt)[Prepared by #author | #issued]

#pagebreak()

= Licence and Disclosure
== Licence
#licence

== Disclosure
#disclosure

#pagebreak()

#outline(title: [Contents], depth: 2)

#pagebreak()

#include "body.typ"
