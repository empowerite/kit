# serve.jq - a verb as a kit serve tool: its input schema from its syntax, and a call's arguments as its words (jq module)
#
# import "serve" as serve; with jq -L "${kit_jq}". The input is a verb's spec, as syntax:spec prints it. $opts says
# what the server holds about the tool:
#
#   - refused: the words a call may never give, such as add for kit packages; a line that requires one is not
#     offered, and an optional part that holds one is taken out of its line
#   - implied: the flags the server gives itself to any line that takes them, such as no-fetch for kit flow
#   - word: the name of the property that holds a line's leading word, such as stage for kit flow
#   - choices: the values of each argument whose choices are closed, by key, such as <package>
#
# A flag is a boolean property, and an operand or an option is a string. One that repeats, alone or in a group that
# repeats, is an array of strings. Each takes its description from the verb's header. An argument is required when
# every line the tool offers requires it.
#
# see: modules/bash/serve.sh, modules/jq/syntax.jq

import "syntax" as syntax;

# word gives the word a call types for a node: a literal's name, or --name for a flag or an option.
def word:
  if .kind == "literal" then .name
  elif .kind == "flag" or .kind == "option" then "--" + .name
  else empty end;

# required_words gives the words a sequence requires: its own literals, flags, and options, and those of a group
# that is not optional.
def required_words:
  .[] | if .kind == "group" then (if .optional then empty else .alternatives[] | required_words end) else word end;

# words gives every word of a sequence, at any depth.
def words: [.. | objects | select(has("kind") and .kind != "group") | word];

# pruned($taken) takes out of a sequence each alternative that holds one of the words, and each group left empty. The
# parameter is not $words: jq would make it a filter of that name, which hides the def above.
def pruned($taken):
  map(if .kind == "group" then
      .alternatives |= map(select(words | any(IN($taken[])) | not) | pruned($taken))
      | select(.alternatives | length > 0)
    else . end);

# nodes gives every node of a sequence but the groups, each with repeat true when it, or a group around it, repeats.
def nodes:
  def walk($repeat):
    .[] | if .kind == "group" then . as $group | $group.alternatives[] | walk($repeat or $group.repeat)
      else . + {repeat: ($repeat or .repeat)} end;
  [walk(false)];

# property($opts) gives the name of the property a node goes in: the word property for a literal, else its name.
def property($opts): if .kind == "literal" then ($opts.word // "subcommand") else .name end;

# ordered gives an array's values once each, in the order they first appear.
def ordered: reduce .[] as $value ([]; if index([$value]) then . else . + [$value] end);

# lines($opts) gives each usage line the tool offers, as {text, sequence}: a line that requires a refused word is left
# out, the others lose their refused parts, and the text is the line as the tool offers it, without what the server
# gives itself.
def lines($opts):
  . as $spec
  | ($opts.refused // []) as $refused
  | [($opts.implied // [])[] | "--" + .] as $implied
  | [.syntax[] | select([required_words] | any(IN($refused[])) | not) | pruned($refused)]
  | map({sequence: ., text: ("kit \($spec.verb) " + (pruned($implied) | syntax::render) | rtrimstr(" "))});

# schema($opts) gives the tool's inputSchema: a JSON Schema object with one property per argument the tool offers.
def schema($opts):
  . as $spec
  | ($opts.implied // []) as $implied
  | [lines($opts)[].sequence] as $sequences
  | [$sequences[] | nodes[] | select(.kind == "flag" and (.name | IN($implied[])) | not)] as $all
  | [$all[] | select(.kind == "literal")] as $literals
  | [$all[] | select(.kind != "literal")] as $others
  | ($opts.word // "subcommand") as $word
  | (if ($literals | length) > 0 then
      {($word): {type: "string", enum: ([$literals[].name] | ordered),
        description: ([$literals[].name] | ordered | map(. + ": " + ($spec.arguments[.].description // "")) | join("; "))}}
    else {} end) as $leading
  | ([$others[].name] | ordered | map(. as $name | [$others[] | select(.name == $name)] as $same | $same[0]
      | syntax::key as $key
      | {key: $name, value: (
          (if .kind == "flag" then {type: "boolean"}
           else ($opts.choices[$key] // null) as $enum
             | (if $enum then {type: "string", enum: $enum} else {type: "string"} end) as $one
             | if any($same[]; .repeat) then {type: "array", items: $one} else $one end
           end)
          + (($spec.arguments[$key].description // null) as $text | if $text then {description: $text} else {} end))})
    | from_entries) as $properties
  | [$sequences[] | [.[] | select(.kind != "group") | property($opts)] | ordered] as $requires
  | (if ($requires | length) > 0
      then reduce $requires[1:][] as $line ($requires[0]; map(select(IN($line[]))))
      else [] end) as $required
  | {type: "object", properties: ($leading + $properties), additionalProperties: false}
    + (if ($required | length) > 0 then {required: $required} else {} end);

# given gives the arguments a call gave: a false switch, a null, and an empty list count as not given.
def given: with_entries(select(.value != false and .value != null and .value != []));

# emit($a; $word) gives the words a sequence makes of the arguments $a: each literal $a[$word] names, each switch that
# is true, and each value of an operand or an option, in the order the line has them.
def emit($a; $word):
  if type == "array" then [.[] | emit($a; $word)[]]
  elif .kind == "group" then [.alternatives[] | emit($a; $word)[]]
  elif .kind == "literal" then (if $a[$word] == .name then [.name] else [] end)
  elif .kind == "flag" then (if $a[.name] == true then ["--" + .name] else [] end)
  else . as $node | ([$a[$node.name] // empty] | flatten) as $values
    | if $node.kind == "operand" then $values
      else $values | map("--\($node.name)=" + .) end
  end;

# cut gives a value as an error quotes it: its first 200 characters.
def cut: .[:200];

# nul is true when a value holds a NUL in a string, at any depth.
def nul: any(.. | strings; explode | any(. == 0));

# argv($args; $opts) gives {argv} for a call's arguments, the words of the first line they fit, with the flags the
# server gives itself; or {error}, saying what is wrong: a value that holds a NUL, which no command line carries, an
# unknown name, a value of the wrong type, an operand's value that starts with -, an operand's value that is a literal
# of the verb's syntax, offered or not, a value off its property's enum, or arguments that fit no line. A value an
# error quotes is cut to 200 characters.
def argv($args; $opts):
  . as $spec
  | schema($opts) as $schema
  | ($opts.word // "subcommand") as $word
  | ($opts.implied // []) as $implied
  | ($args | given) as $a
  | ($schema.properties | keys_unsorted) as $known
  | [lines($opts)[].sequence | nodes[] | select(.kind == "operand") | .name] as $operands
  | [$spec.syntax[] | nodes[] | select(.kind == "literal") | .name] as $literals
  | [$a | keys_unsorted[] | select(IN($known[]) | not)] as $unknown
  | ([$a | to_entries[] | select(.value | nul) | .key] | first) as $nul
  | if $nul then {error: "\($nul | cut)'s value holds a NUL, which no command line carries"}
    elif ($unknown | length) > 0 then
      {error: ("unknown argument\(if ($unknown | length) > 1 then "s" else "" end) "
        + "\($unknown | map(cut) | join(", ")); kit \($spec.verb) takes: \($known | join(", "))")}
    else
      ([$a | to_entries[] | .key as $key | .value as $value | $schema.properties[$key] as $property
        | ($property.enum // $property.items.enum) as $enum
        | if $property.type == "boolean" then
            (if ($value | type) != "boolean" then "\($key) is true or false" else empty end)
          elif $property.type == "array" and (($value | type) != "array" or any($value[]; type != "string")) then
            "\($key) is a list of strings"
          elif $property.type == "string" and ($value | type) != "string" then "\($key) is a string"
          else
            ([$value] | flatten) as $values
            | first(
                (if ($key | IN($operands[])) then
                  ($values[] | select(startswith("-"))
                    | "\($key)'s value \(cut) starts with -, so the verb would read it as a flag"),
                  ($values[] | select(IN($literals[])) | "\($key)'s value \(cut) is a word kit \($spec.verb) acts on")
                else empty end),
                (if $enum then
                  $values[] | select(IN($enum[]) | not) | "\($key)'s value \(cut) is not one of: \($enum | join(", "))"
                else empty end))
          end] | first // null) as $wrong
      | if $wrong then {error: $wrong}
        else
          lines($opts) as $lines
          | (first($lines[] | .sequence as $sequence
              | [$sequence | nodes[] | property($opts)] as $names
              | [$sequence[] | select(.kind != "group") | property($opts)] as $requires
              | [$sequence | nodes[] | select(.kind == "literal") | .name] as $leading
              | select(($a | keys) - $names == [])
              | select($requires - ($a | keys) == [])
              | select(if ($leading | length) > 0 then $a[$word] == $leading[0] else ($a | has($word) | not) end)
              | ([$implied[] | select(IN($names[]))] | map({(.): true}) | add // {}) as $own
              | {argv: ($sequence | emit($a + $own; $word))})
            // {error: ("these arguments fit no usage of kit \($spec.verb): " + ([$lines[].text] | join("; ")))})
        end
    end;
