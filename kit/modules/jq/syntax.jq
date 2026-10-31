# syntax.jq - a verb's syntax: the usage text of a node, and the key an argument is known by (jq module)
# ----------------------------------------------------------------------------------------------------------------------
#
# import "syntax" as syntax; with jq -L "${kit_jq}". The syntax is the JSON modules/bash/syntax.sh parses from a
# verb's usage lines: an array of alternatives, each a sequence of nodes.
#
# see: modules/bash/syntax.sh

# key gives the key of a node, as the arguments object is keyed: --name for a flag or an option, --<name> for a
# wildcard option, <name> for an operand, and the word itself for a literal.
def key:
  if .kind == "flag" or .kind == "option" then
    if .wildcard then "--<\(.name)>" else "--\(.name)" end
  elif .kind == "operand" then "<\(.name)>"
  else .name end;

# render gives a node, or a sequence of nodes, as usage text: a group in its brackets with its alternatives
# separated by |, an option with its value inline or apart, and … after what repeats.
def render:
  def mark: if .repeat then "…" else "" end;
  if type == "array" then map(render) | join(" ")
  elif .kind == "group" then
    (.alternatives | map(render) | join(" | ")) as $inner
    | (if .optional then "[\($inner)]" else "(\($inner))" end) + mark
  elif .kind == "literal" then .name + mark
  elif .kind == "operand" then "<\(.name)>" + mark
  elif .kind == "flag" then "--\(.name)"
  elif .wildcard then "--<\(.name)>=<\(.value)>" + mark
  elif .inline then "--\(.name)=<\(.value)>" + mark
  else "--\(.name) <\(.value)>" + mark end;

# undescribed gives the keys a spec's syntax names, in order of first appearance, that its arguments object does not
# describe.
def undescribed:
  (.arguments | keys) as $described
  | [.syntax | .. | objects | select(has("kind") and .kind != "group") | key]
  | reduce .[] as $key ([]; if index($key) then . else . + [$key] end)
  | map(select(. as $key | $described | index($key) | not));

# ----------------------------------------------------------------------------------------------------------------------
