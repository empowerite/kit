# completion.jq - what the word being typed on kit's command line may be, from the verb's syntax (jq module)
#
# import "completion" as completion; with jq -L "${kit_jq}". The input is kit syntax's JSON for one verb.
#
# see: modules/bash/completion.sh, modules/bash/syntax.sh

import "syntax" as syntax;

# nodes gives every node of a syntax, the groups' alternatives included, the groups too.
def nodes:
  .[][] | if .kind == "group" then ., (.alternatives | nodes) else . end;

# next($typed; $current) gives what the word being typed may be, after the words typed before it, all after
# kit <verb>, as {mode, key, value, prefix, operands, candidates}:
#
#   - flags: $current starts with -; candidates are the flags and options not given yet, a flag as --name, an option
#     as --name=value, or any that repeats, each [word, description], an option's word ending in =
#   - value: $current is --name=…; key is the option, value the name of its value, and prefix what a candidate
#     starts with: --name=, or, for a list that joins, what is typed up to its last :
#   - positional: anything else; candidates are the literal words when the verb has some and none is typed yet, and
#     operands otherwise lists the operands' keys
def next($typed; $current):
  [.syntax | nodes] as $nodes
  | (.arguments // {}) as $args
  | ([$nodes[] | select(.kind == "literal") | .name] | unique) as $literals
  | if ($current | test("^--[^=]+=")) then
      ($current | sub("=.*"; "")) as $key
      | [$nodes[] | select(.kind == "option" and syntax::key == $key)] as $options
      # a list that joins takes its next value after the last :, so what is typed before it stays
      | {mode: "value", key: $key,
         prefix: (if any($options[]; syntax::joins) then $current | sub("[^:=]*$"; "") else $key + "=" end),
         value: ($options | map(.value) | first // "")}
    elif ($current | startswith("-")) then
      {mode: "flags", candidates: ([$nodes[]
        | select(.kind == "flag" or (.kind == "option" and (.wildcard | not)))
        | syntax::key as $key
        | select(.repeat or (($typed | map(sub("=.*"; "")) | index($key)) == null))
        | [$key + (if .kind == "option" then "=" else "" end), ($args[$key].description // "")]]
        | unique)}
    elif ($literals | length > 0) and ([$typed[] | select(. as $word | $literals | index($word))] | length == 0) then
      {mode: "positional", candidates: [$literals[] | [., ($args[.].description // "")]]}
    else
      {mode: "positional", candidates: [],
       operands: ([$nodes[] | select(.kind == "operand") | syntax::key] | unique)}
    end;
