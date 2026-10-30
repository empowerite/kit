# template.jq - a template's holes: the expression that checks a file, and the file a template makes (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "template" as template; with jq -L "${kit_root}/modules/jq". A template is a file with holes written «name»:
#
#   a hole within a line      matches any text on that line; a name used twice must match the same text each time
#   a hole that is a line     matches any run of lines, including none
#
# see: modules/bash/template.sh, docs/templates.md

# lines splits text into its lines, without the empty string after a final newline.
def lines: split("\n") | if .[-1] == "" then .[:-1] else . end;

# literal escapes every character a regular expression gives a meaning.
def literal: gsub("(?<c>[.^$|?*+()\\[\\]{}\\\\])"; "\\\(.c)");

# regex reads a template and gives the expression that matches a file with its shape, from start to end.
def regex:
  reduce lines[] as $line
    ({re: "", seen: {}};
     if ($line | test("^«[a-z_]+»$")) then .re += "(?:[^\\n]*\\n)*?"
     else
       ([$line | match("«([a-z_]+)»"; "g") | .captures[0].string]) as $names
       | ([$line | splits("«[a-z_]+»")] | map(literal)) as $parts
       | reduce range(0; $names | length) as $i
           (.line = $parts[0];
            if .seen[$names[$i]] then .line += "\\k<\($names[$i])>"
            else .line += "(?<\($names[$i])>[^\\n]*?)" | .seen[$names[$i]] = true end
            | .line += $parts[$i + 1])
       | .re += .line + "\\n" | del(.line)
     end)
  | "\\A" + .re + "\\z";

# render($values) reads a template and gives the file it makes. A hole within a line takes its value, and stays a
# hole without one. A hole that is a line takes its value's lines, else a blank line, else nothing when it is the
# template's last line.
def render($values):
  lines
  | length as $n
  | to_entries
  | map(.key as $i | .value
        | if test("^«[a-z_]+»$")
          then ($values[ltrimstr("«") | rtrimstr("»")] // (if $i == $n - 1 then null else "" end))
          else gsub("«(?<k>[a-z_]+)»"; $values[.k] // "«\(.k)»") end
        | select(. != null) | . + "\n")
  | add // "";

# unmatched($regex; $texts) reads [{path, variants}] and gives each entry that some variant applies to and whose
# text, at the same index in $texts, matches none of its variants' expressions in $regex, a {variant: expression}.
def unmatched($regex; $texts):
  to_entries[]
  | .key as $i | .value
  | select((.variants | length) > 0)
  | select(any(.variants[]; . as $v | $texts[$i] | test($regex[$v])) | not);
