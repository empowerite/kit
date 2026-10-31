# template.jq - the template language: a template's parts, the file it makes, and the values a file gives it (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "template" as template; with jq -L "${kit_jq}". A template is the text of a file with holes in it:
#
#   «name»                 within a line: any text on that line
#   «name…»                within a line: any text, across lines
#   «name» on its own line any run of whole lines, including none
#   «name=default»        any of the above, with the text it makes when no value is given; \n in it is a newline
#
# A name is [a-z_]+, or a dotted path in kit.yml, such as answers.ci_runner, whose value fills it. path,
# name, and stem, and a dotted name the settings set, are bound before anything else, so they are literal text. A name that appears once takes a string, and one that appears more than once takes an array, one element per
# appearance in template order. The lines between a kit:generate directive and its kit:end line are a region: a
# command owns them, so a file may hold anything there, and the template makes them as it has them.
#
# A template is matched in one pass, with no regular expression: each hole extends to the first place the literal
# after it appears, which leaves the most text for what follows, so a file matches exactly when this pass finds it.
#
# see: modules/bash/template.sh, docs/templates.md

# hole_re matches one hole: its name, its … mark, and its default.
def hole_re: "«(?<name>[a-z_]+(?:\\.[a-z_]+)*)(?<span>…)?(?:=(?<default>[^»]*))?»";

# bind($bound) replaces each hole whose name is a key of $bound with the key's value, in the template's text.
def bind($bound):
  reduce ($bound | to_entries[]) as $e (.; gsub("«\($e.key | gsub("\\."; "\\."))(…)?(=[^»]*)?»"; $e.value));

# lines_of splits text into lines, each with its newline, the last without one when the text does not end in one.
def lines_of: [match("[^\n]*\n|[^\n]+$"; "g").string];

# parts reads a template's text, its bound names already bound, and gives its parts in order, each with the template
# line it starts on: {literal, line}, or {hole: name, kind: "inline" | "span" | "lines", default, line}. A region's
# lines are one lines hole, region: true, that keeps the template's text. Two literals in a row are joined.
def parts:
  lines_of as $lines
  | reduce range(0; $lines | length) as $i ({parts: [], region: null};
      $lines[$i] as $line
      | if .region != null then
          if ($line | test("kit:end\\b")) then
            .parts += [{hole: "region_\(.region.at)", kind: "lines", region: true, text: .region.text, line: .region.at}]
            | .region = null | .parts += [{literal: $line, line: ($i + 1)}]
          else .region.text += $line end
        elif ($line | test("^" + hole_re + "\n?$")) then
          ($line | capture("^" + hole_re)) as $h
          | .parts += [{hole: $h.name, kind: "lines", default: $h.default, line: ($i + 1)}]
        else
          ([$line | match(hole_re; "g")]) as $holes
          | reduce range(0; ($holes | length) + 1) as $k (.;
              (if $k == 0 then 0 else $holes[$k - 1].offset + $holes[$k - 1].length end) as $from
              | (if $k < ($holes | length) then $holes[$k].offset else ($line | length) end) as $to
              | (if $from < $to then .parts += [{literal: $line[$from:$to], line: ($i + 1)}] else . end)
              | if $k < ($holes | length) then
                  ($holes[$k].captures | map({(.name): .string}) | add) as $h
                  | .parts += [{hole: $h.name, kind: (if $h.span then "span" else "inline" end), default: $h.default,
                      line: ($i + 1)}]
                else . end)
          | if ($line | test("kit:generate\\b")) then .region = {at: ($i + 1), text: ""} else . end
        end)
  | .parts
  | reduce .[] as $p ([]; if ($p.literal != null) and (.[-1].literal != null) then .[-1].literal += $p.literal
                          else . + [$p] end);

# bind_parts($bound) reads a template's parts and turns each hole whose name is a key of $bound into literal text, the
# key's value, joining it with the literals beside it: bind on parts already read, so a template is read once.
def bind_parts($bound):
  map(.hole as $h | if $h != null and (.region | not) and ($bound | has($h)) then {literal: $bound[$h], line} else . end)
  | reduce .[] as $p ([]; if ($p.literal != null) and (.[-1].literal != null) then .[-1].literal += $p.literal
                          else . + [$p] end);

# ambiguous reads a template's parts and gives the line of each hole that follows another hole with no text between
# them, which no match could split.
def ambiguous:
  . as $parts
  | range(1; length) | select($parts[.].hole != null and $parts[. - 1].hole != null) | $parts[.].line;

# arrays reads a template's parts and gives the names that appear more than once, which take an array.
def arrays: [.[] | select(.hole != null and (.region | not)) | .hole] | group_by(.) | map(select(length > 1) | .[0]);

# fits($kind; $text) is true when $text is something a hole of that kind may hold.
def fits($kind; $text):
  if $kind == "inline" then ($text | contains("\n") | not)
  elif $kind == "lines" then $text == "" or ($text | endswith("\n"))
  else true end;

# match($text) reads a template's parts and matches $text against them. It gives {ok: true, values} when the text has
# the template's shape, values each hole's text (an array for a name that appears more than once, and a region's under
# region_<line>); otherwise {ok: false, line}, the template line where the text leaves the template, or null at its end.
def match($text):
  . as $parts
  | arrays as $arrays
  | reduce $parts[] as $p ({pos: 0, hole: null, values: {}, line: null, ok: true};
      if (.ok | not) then .
      elif $p.hole != null then .hole = $p
      elif .hole == null then
        if $text[.pos:.pos + ($p.literal | length)] == $p.literal then .pos += ($p.literal | length)
        else .ok = false
          | .line = ($p.line + ([$text[.pos:] | lines_of[]] as $have | $p.literal | lines_of
              | first(range(0; length) as $k | select($have[$k] != .[$k]) | $k) // 0))
        end
      else .hole as $h | .pos as $from
        | ([$text | indices($p.literal)[] | select(. >= $from)
            | select($h.kind != "lines" or . == $from or $text[. - 1:.] == "\n")] | first) as $at
        | if $at == null or (fits($h.kind; $text[$from:$at]) | not) then .ok = false
            # A literal that opens with the newline ending the line before is reported by the line its text is on.
            | .line = $p.line + (if ($p.literal | startswith("\n")) and ($p.literal | length) > 1 then 1 else 0 end)
          else .values |= (if ($arrays | index($h.hole)) then .[$h.hole] += [$text[$from:$at]]
                           else .[$h.hole] = $text[$from:$at] end)
            | .pos = $at + ($p.literal | length) | .hole = null end
      end)
  | if (.ok | not) then {ok: false, line}
    elif .hole != null then
      .hole as $h | $text[.pos:] as $rest
      | if fits($h.kind; $rest) or ($h.kind == "lines" and ($rest | contains("\n") | not)) then
          {ok: true, values: (.values | if ($arrays | index($h.hole)) then .[$h.hole] += [$rest]
                                         else .[$h.hole] = $rest end)}
        else {ok: false, line: $h.line} end
    elif .pos == ($text | length) then {ok: true, values}
    else {ok: false, line: null} end;

# render($values) reads a template's parts and gives the file they make. A hole takes its value, the element for its
# appearance when the value is an array; else its default; else a hole within a line stays a hole, and a hole that is
# a line gives a blank line, or nothing when it is the template's last part. A region gives its value, else the
# template's own lines.
def render($values):
  . as $parts
  | length as $n
  | reduce range(0; $n) as $i ({out: "", seen: {}};
      $parts[$i] as $p
      | if $p.literal != null then .out += $p.literal
        elif $p.region then .out += ($values[$p.hole] // $p.text)
        else (.seen[$p.hole] // 0) as $k | .seen[$p.hole] = $k + 1
          | ($values[$p.hole] | if type == "array" then .[$k] else . end) as $v
          | ($p.default | if . == null then null else gsub("\\\\n"; "\n") end) as $d
          | if $p.kind == "lines" then
              if $v != null then .out += (if $v == "" or ($v | endswith("\n")) then $v else $v + "\n" end)
              elif $d != null then .out += $d + "\n"
              elif $i == $n - 1 then .
              else .out += "\n" end
            else .out += ($v // $d // "«\($p.hole)»") end
        end)
  | .out;

# holes reads a template's parts and gives the names a caller must fill to make a file: the holes within a line, or
# across lines, that have no default, once each, in order.
def holes:
  [.[] | select(.hole != null and .kind != "lines" and .default == null) | .hole] | reduce .[] as $h ([];
    if index($h) then . else . + [$h] end);

# check($templates; $files; $settings) reads nothing and judges each file: $templates maps a variant to its template's
# text, $files is [{path, text, variants}], and $settings the values kit.yml gives dotted names, bound with
# path, name, and stem. It gives one {path, variants, line} per file that matches none of the variants
# its path allows: line is the furthest template line any of them reached, and variant the one that reached it, both
# null when every one ran to its end.
def check($templates; $files; $settings):
  ($templates | map_values(parts)) as $parsed
  | $files[] | . as $f
  | select(($f.variants | length) > 0)
  | ($f.path | split("/") | last) as $name
  | ({path: $f.path, name: $name, stem: ($name | sub("\\.[^.]*$"; ""))} + $settings) as $bound
  | [$f.variants[] as $v | $parsed[$v] | bind_parts($bound) | match($f.text) | .variant = $v] as $results
  | select(any($results[]; .ok) | not)
  | ([$results[] | select(.line != null)] | max_by(.line)) as $furthest
  | {path: $f.path, variants: $f.variants, line: $furthest.line, variant: $furthest.variant};

# unapply($bound; $text) reads a template's text and gives the values $text fills its holes with, or null when the
# text does not have the template's shape.
def unapply($bound; $text):
  bind($bound) | parts | match($text) | if .ok then .values else null end;
