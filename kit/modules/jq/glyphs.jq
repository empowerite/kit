# glyphs.jq - the glyphs kit flow draws, what each means, and each stage's legend (jq module)
# ----------------------------------------------------------------------------------------------------------------------
# import "glyphs" as glyphs; with jq -L "${kit_jq}". The glyphs live in one place, glyphs.json beside this module, so
# kit help flow's legend and kittui's cannot say different things of one glyph. Its keys:
#
#   glyphs    each glyph: name, the identifier kittui gives it; glyph; meaning, a few words, which kittui's legend
#             shows; and more, what kit help flow adds after a colon, when there is more to say
#   states    a pull request's states, the first that holds as kit flow reads them: glyph, state, and more
#   title     the name of the title's glyph, which no stage draws
#   stages    each stage in kit flow's order: stage; glyph, the name of the one beside its tab; legend, the glyphs
#             it draws, each by name, with link, row or page, when a click on it in kittui leads to a row of another
#             stage or to a page; and states, page, when the stage's legend lists the states, and a click on one opens
#             the pull request's page
#
# A meaning is one glyph's wherever it is drawn, so a stage's legend never words it otherwise.
#
# see: bin/kit-flow, modules/bash/help.sh

import "glyphs" as $glyphs;

# data gives glyphs.json.
def data: $glyphs::glyphs[0];

# glyph($d; $name) gives the glyph named $name in the table $d.
def glyph($d; $name): $d.glyphs[] | select(.name == $name);

# wrap($width; $indent) reads a line and gives it in lines of at most $width characters, broken at spaces, each after
# the first starting with $indent.
def wrap($width; $indent):
  reduce (split(" ")[]) as $word ([""];
    if (.[-1] | length) == 0 then .[-1] = $word
    elif (.[-1] | length) + 1 + ($word | length) <= $width then .[-1] += " " + $word
    else . + [$indent + $word] end)
  | .[];

# line($glyph; $meaning; $more) gives one line of the help's legend: the glyph, two spaces, the meaning, and more after
# a colon when there is more, wrapped under the meaning to fit 120 columns, the glyph's two among them.
def line($glyph; $meaning; $more):
  "#   \($glyph)  \($meaning)\(if $more then ": " + $more else "" end)" | wrap(118; "#       ");

# legend($d) gives the legend of the table $d as kit help flow prints it, the comment lines of kit/bin/kit-flow's
# header: the title's glyph, then each stage's glyphs under its name, and under a stage that lists them the states,
# then any other.
def legend($d):
  glyph($d; "Other") as $other
  | (glyph($d; $d.title) | line(.glyph; .meaning; .more)), "#",
    ($d.stages[]
     | "# \(.stage):", "#", (.legend[] | glyph($d; .name) | line(.glyph; .meaning; .more)), "#",
       (select(.states) | "# \(.stage): a pull request's state, the first of these that holds:", "#",
          ($d.states[] | line(.glyph; .state; .more)), line($other.glyph; $other.meaning; null), "#"));

# legend gives glyphs.json's legend, as legend($d) does.
def legend: legend(data);

# quote reads a string and gives it as a bash word in single quotes.
def quote: "'" + gsub("'"; "'\\''") + "'";

# bash($d) gives the lines of bash that declare the table $d's glyphs for modules/bash/flow.sh's glyphs region: each
# glyph by its name, each stage's glyph by the stage, and each state's glyph by the state word.
def bash($d):
  "# flow_glyphs gives each glyph by its name, flow_stage_glyphs each stage's by the stage, and flow_state_glyphs a",
  "# pull request state's by its word; a state word not among them draws flow_glyphs[Other].",
  "declare -gA flow_glyphs=(",
  ($d.glyphs[] | "  [\(.name)]=\(.glyph | quote)"),
  ")",
  "declare -gA flow_stage_glyphs=(",
  ($d.stages[] | "  [\(.stage)]=\(glyph($d; .glyph).glyph | quote)"),
  ")",
  "declare -gA flow_state_glyphs=(",
  ($d.states[] | "  [\(.state)]=\(.glyph | quote)"),
  ")";

# bash gives glyphs.json's declarations, as bash($d) does.
def bash: bash(data);

