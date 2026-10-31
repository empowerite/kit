# syntax.jq - a verb's syntax: a node's usage text, an argument's key, and the match of a command line (jq module)
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

# given_keys($node; $st) gives the keys of a flag or option node that the command line gave: the node's own key, or,
# for a wildcard option, every --x=y the line gave that no named option took.
def given_keys($node; $st):
  if $node.wildcard then $st.wild
  else [$node | key] | map(select(. as $k | $st.given | has($k))) end;

# held($alt; $st) gives the given keys an alternative holds anywhere in it, in grammar order.
def held($alt; $st):
  [$alt | .. | objects | select(.kind == "flag" or .kind == "option") | given_keys(.; $st)[]];

# firsts($alt) gives the keys an alternative can start with: its first node's key, through a group's alternatives.
def firsts($alt):
  if ($alt | length) == 0 then []
  elif $alt[0].kind == "group" then [$alt[0].alternatives[] | firsts(.)[]]
  else [$alt[0] | key] end;

# unknown_word($alts; $st) gives the refusal of the next word when it opens none of the alternatives though one of them
# opens with a literal, so a subcommand that does not exist is unknown, not missing; otherwise null.
def unknown_word($alts; $st):
  if ($st.pos | length) > 0 and ($st.pos[0].dash | not) and any($alts[]; length > 0 and .[0].kind == "literal")
  then "unknown argument '\($st.pos[0].word)'" else null end;

# walk_sequence($nodes; $st) matches the sorted words against one sequence, node by node: a literal must be the next
# word, an operand takes the next word (a repeated one every word left), a flag or option must have been given, and
# a group walks the alternative the words select, or is skipped when optional and nothing selects it.
#
# choose($alts; $s) picks the alternative the words select: the one a literal opens when the next word is that
# literal; else the one that holds a given flag or option, two such being a refusal; else the first whose walk
# completes, the one that takes the most words winning a tie. It gives {index}, {refusal}, or {} when nothing selects.
def walk_sequence($nodes; $st):
  def choose($alts; $s):
    ($s.pos[0].word) as $next
    | [range($alts | length)
        | select(($alts[.] | length) > 0 and $alts[.][0].kind == "literal" and $alts[.][0].name == $next)] as $by_literal
    | if ($by_literal | length) > 0 then {index: $by_literal[0]}
      else [range($alts | length) | {index: ., keys: held($alts[.]; $s)} | select(.keys | length > 0)] as $by_given
      | if ($by_given | length) > 1 then
          {refusal: "\($by_given[0].keys[0]) cannot be combined with \($by_given[1].keys[0])"}
        elif ($by_given | length) == 1 then {index: $by_given[0].index}
        else [range($alts | length) as $i
            | walk_sequence($alts[$i]; $s) | select(.refusal == null and .missing == null) | {index: $i, left: (.pos | length)}]
          | sort_by(.left) | first // {}
        end
      end;
  def walk_group($node; $s):
    if ($node.alternatives | length) == 1 and ($node.optional | not) then walk_sequence($node.alternatives[0]; $s)
    else choose($node.alternatives; $s) as $c
    | if $c.refusal != null then $s | .refusal = $c.refusal
      elif $c.index != null then walk_sequence($node.alternatives[$c.index]; $s)
      elif unknown_word($node.alternatives; $s) != null then $s | .refusal = unknown_word($node.alternatives; $s)
      elif $node.optional then $s
      else $s | .missing = [$node.alternatives[] | firsts(.)[]] end
    end;
  reduce $nodes[] as $node ($st;
    if .refusal != null or .missing != null then .
    elif $node.kind == "literal" then
      if (.pos | length) == 0 then .missing = [$node.name]
      elif .pos[0].word == $node.name then .pos |= .[1:] | .literals += [$node.name]
      else .refusal = "unknown argument '\(.pos[0].word)'" end
    elif $node.kind == "operand" then
      if (.pos | length) == 0 then .missing = ["<\($node.name)>"]
      elif .pos[0].dash then .refusal = "unknown flag '\(.pos[0].word)'"
      elif $node.repeat then (.pos | map(.word)) as $w | .operands += $w | .values["<\($node.name)>"] += $w | .pos = []
      else .pos[0].word as $w | .operands += [$w] | .values["<\($node.name)>"] += [$w] | .pos |= .[1:] end
    elif $node.kind == "group" then walk_group($node; .)
    else given_keys($node; .) as $keys
      | if ($keys | length) > 0 then .used += $keys else .missing = [$node | key] end
    end);

# lex($rest; $st; $by_key; $wild) sorts the words: a named flag or option is taken wherever it stands, with its value
# inline or as the next word; -- ends the flags; every other word is kept in order, a dash-word tentatively, for a
# repeated operand to take or the walk to refuse.
def lex($rest; $st; $by_key; $wild):
  if $st.refusal != null or ($rest | length) == 0 then $st
  else $rest[0] as $w
  | if $st.plain or ($w | startswith("-") | not) or $w == "-" then
      lex($rest[1:]; $st | .pos += [{word: $w, dash: false}]; $by_key; $wild)
    elif $w == "--" then lex($rest[1:]; $st | .plain = true; $by_key; $wild)
    else ($w | index("=")) as $eq
    | (if $eq == null then $w else $w[:$eq] end) as $name
    | (if $eq == null then null else $w[$eq + 1:] end) as $inline
    | $by_key[$name] as $node
    | if $node != null and $node.kind == "flag" then
        if $eq != null then lex($rest[1:]; $st | .pos += [{word: $w, dash: true}]; $by_key; $wild)
        else lex($rest[1:]; $st | .given[$name] = ["1"]; $by_key; $wild) end
      elif $node != null then
        if $eq != null then
          if $inline == "" then $st | .refusal = "\($name) needs a value"
          else lex($rest[1:]; $st | .given[$name] += [$inline]; $by_key; $wild) end
        elif $node.inline or ($rest | length) < 2 then $st | .refusal = "\($name) needs a value"
        else lex($rest[2:]; $st | .given[$name] += [$rest[1]]; $by_key; $wild) end
      elif $wild and $eq != null and ($name | test("^--[^=]+$")) then
        if $inline == "" then $st | .refusal = "\($name) needs a value"
        else lex($rest[1:]; $st | .given[$name] += [$inline] | .wild += [$name]; $by_key; $wild) end
      else lex($rest[1:]; $st | .pos += [{word: $w, dash: true}]; $by_key; $wild) end
    end
  end;

# match($words) judges a command line against the spec's syntax, as the loader's guard:arguments does. It gives
# status 0 with literals, operands, options (each key to its values, a flag's being ["1"]), and values (every operand
# and option key to its values) when the words complete the syntax; status 1 with missing, the keys that could come
# next joined by " or ", and the literals so far, when a required literal, operand, or group is missing; and status 2
# with refusal, the whole line, when a word is refused.
def match($words):
  . as $spec
  | ($spec.syntax | [.. | objects | select(.kind == "flag" or .kind == "option")]) as $named
  | ([$named[] | select(.wildcard | not) | {key: key, value: .}] | from_entries) as $by_key
  | ($named | any(.wildcard)) as $wild
  | if $spec.syntax == [[]] and ($words | length) > 0 then
      {status: 2, refusal: "kit \($spec.verb) takes no arguments; run: kit help \($spec.verb)"}
    else lex($words; {pos: [], given: {}, wild: [], used: [], literals: [], operands: [], values: {}, refusal: null,
        missing: null, plain: false}; $by_key; $wild)
    | if .refusal != null then .
      else walk_sequence([{kind: "group", optional: false, alternatives: $spec.syntax, repeat: false}]; .) end
    | (.pos | map(select(.dash))) as $dashed
    | if .refusal != null then {status: 2, refusal: "\(.refusal); run: kit help \($spec.verb)"}
      elif .missing != null and ($dashed | length) > 0 then
        {status: 2, refusal: "unknown flag '\($dashed[0].word)'; run: kit help \($spec.verb)"}
      elif .missing != null then
        {status: 1, missing: (.missing | reduce .[] as $k ([]; if index($k) then . else . + [$k] end) | join(" or ")),
          literals: .literals, operands: .operands, options: .given, values: (.given + .values)}
      elif (.pos | length) > 0 then
        {status: 2, refusal: "\(if .pos[0].dash then "unknown flag" else "unknown argument" end) '\(.pos[0].word)'; run: kit help \($spec.verb)"}
      else ((.given | keys_unsorted) - .used) as $unused
      | if ($unused | length) > 0 then
          {status: 2, refusal: "\($unused[0]) cannot be combined with \((.literals + .used + .operands)[0]); run: kit help \($spec.verb)"}
        else {status: 0, literals: .literals, operands: .operands, options: .given, values: (.given + .values)} end
      end
    end;

# fields flattens a match for a shell to read, one string per record: status and its value; refusal and missing with
# theirs (empty when absent); literals and operands, each as a count and the words; option, key, count, values for
# every option given; and argument, key, count, values for every operand and option key.
def fields:
  ["status", (.status | tostring), "refusal", (.refusal // ""), "missing", (.missing // ""),
    "literals", (.literals // [] | length | tostring), (.literals // [])[],
    "operands", (.operands // [] | length | tostring), (.operands // [])[],
    ((.options // {}) | to_entries[] | "option", .key, (.value | length | tostring), .value[]),
    ((.values // {}) | to_entries[] | "argument", .key, (.value | length | tostring), .value[])]
  | .[];

# ----------------------------------------------------------------------------------------------------------------------
