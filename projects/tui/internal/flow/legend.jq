# legend.jq - kittui's glyphs and legends, as Go, from the kit's glyphs (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "legend" as legend; with jq -L projects/tui/internal/flow. glyph.go's glyphs region is what go gives, gofmt'd,
# from the kit's glyphs, kit/modules/jq/glyphs.json, so kittui's legends say of each glyph what kit help flow says.
# It reads the table itself rather than through glyphs.jq, as jq 1.8 cannot import a module that imports data from
# another module.
#
# see: kit/modules/jq/glyphs.jq, projects/tui/internal/flow/glyph.go

import "glyphs" as $glyphs {search: "../../../../kit/modules/jq"};

# quote reads a string and gives it as a Go string literal.
def quote: tojson;

# glyph($d; $name) gives the glyph named $name in the table $d.
def glyph($d; $name): $d.glyphs[] | select(.name == $name);

# go($d) gives the Go the glyphs region holds for the table $d, unformatted, after flow's package clause, so gofmt
# takes it: a constant per glyph, its meaning as its comment; the title's entry; each stage's legend and marks; the
# states, then any other; and each stage's glyph.
def go($d):
  $d
  | "package flow",
    "// The glyphs, as the legend draws them, each beside its meaning.",
    "const (",
    ($d.glyphs[] | "\t\(.name) = \(.glyph | quote) // \(.meaning)"),
    ")",
    "",
    "// title is the title's glyph, which no stage draws.",
    "var title = entry{\($d.title), \(glyph($d; $d.title).meaning | quote)}",
    "",
    "// legends gives each stage's legend: every glyph the stage draws, its own first. A glyph may be in several.",
    "var legends = map[string][]entry{",
    ($d.stages[]
     | "\t\(.stage | quote): {",
       (.legend[] | "\t\t{\(.name), \(.name as $n | glyph($d; $n).meaning | quote)},"),
       "\t},"),
    "}",
    "",
    "// marks gives the mark after a glyph's meaning in a stage's legend, for each glyph a click on leads somewhere:",
    "// to the row of another stage, or to a page.",
    "var marks = map[string]map[string]string{",
    ($d.stages[] | select(any(.legend[]; .link))
     | [.legend[] | select(.link) | "\(.name): \(if .link == "row" then "ToRow" else "ToPage" end)"] as $marks
     | "\t\(.stage | quote): {\($marks | join(", "))},"),
    "}",
    "",
    "// states gives a pull request's glyph by its state word, the last for any other; it hangs under changes' legend,",
    "// under a heading marked ToPage, as a click on any of them opens the pull request's page.",
    "var states = []entry{",
    ($d.states[] | "\t{\(.glyph | quote), \(.state | quote)},"),
    "\t{Other, \(glyph($d; "Other").meaning | quote)},",
    "}",
    "",
    "// stageGlyphs gives each stage's glyph, drawn beside its name on the line of stages.",
    "var stageGlyphs = map[string]string{",
    ($d.stages[] | "\t\(.stage | quote): \(.glyph),"),
    "}";

# go gives the Go of glyphs.json, as go($d) does.
def go: go($glyphs::glyphs[0]);
