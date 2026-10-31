# shellcheck shell=bash
# syntax.sh - parses a verb's usage lines into its syntax, as JSON, and matches a command line to it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
# shellcheck disable=SC2034  # syntax_refusal, syntax_missing, syntax_options, syntax_values: read by guard and verbs
#
# A verb's header holds its usage lines, and help.sh reads them. This module reads what the lines say: after
# kit <verb>, a usage line is written in this grammar:
#
#   word                a literal subcommand
#   a | b               alternatives; several usage lines are alternatives too
#   <name>              an operand; <name>… repeats
#   --name              a flag
#   --name=<value>      an option whose value is inline; --name <value> takes it as the next word
#   --<hole>=<value>    a wildcard option: any --x=y
#   [ … ]               optional; ( … ) a required group
#   …                   after a word or a group: it may repeat; ... is read as …
#
# The syntax is JSON: an array of alternatives, each a sequence of nodes. A node's kind is literal, operand, flag,
# option, or group; every node carries repeat, a group carries optional and its alternatives, an option its value
# and inline, and a wildcard option wildcard. The jq module syntax.jq renders a node back as usage text and gives
# the key an argument is known by.

use help

# The parser's state: the words of one usage line, the next word to read, the JSON written so far, and the error
# that stopped it.
declare -ag syntax_tokens=()
declare -g syntax_pos=0 syntax_json='' syntax_error=''

# What syntax:match sorted from the last command line it judged.
declare -g syntax_refusal='' syntax_missing=''
declare -ag syntax_literals=() syntax_operands=()
declare -Ag syntax_options=() syntax_values=()

# syntax:spec <verb> prints the verb's syntax as the kit syntax JSON: verb, purpose, usage, syntax, and arguments, the
# descriptions in the header's argument block, each with choices, closed or open, when the verb's module lists them,
# and extends: true when the module answers --extend. When a usage line does not parse, the JSON carries error
# instead of syntax, and it returns 1.
syntax:spec() {
  local verb=$1 syntax arguments extends=false
  local -a usage
  readarray -t usage < <(help:usage "${verb}")
  if ! syntax=$(syntax:parse "${verb}" "${usage[@]}")
  then
    jq --null-input --arg verb "${verb}" --arg error "${syntax}" \
      '{verb: $verb, usage: $ARGS.positional, error: $error}' --args "${usage[@]}"
    return 1
  fi
  arguments=$(syntax:arguments "${verb}")
  syntax:answers "${verb}" extend && extends=true
  jq --null-input --arg verb "${verb}" --arg purpose "$(help:purpose "${verb}")" --argjson syntax "${syntax}" \
    --argjson arguments "${arguments}" --argjson kinds "$(syntax:kinds "${verb}" "${arguments}")" \
    --argjson extends "${extends}" \
    '{verb: $verb, purpose: $purpose, usage: $ARGS.positional, syntax: $syntax,
      arguments: ($arguments | with_entries(.value += (if $kinds[.key] then {choices: $kinds[.key]} else {} end)))}
      + (if $extends then {extends: true} else {} end)' \
    --args "${usage[@]}"
}

# syntax:parse <verb> <usage-line>… prints the syntax JSON of the usage lines: the alternatives of every line, in
# order. When a line does not parse, it prints the error instead and returns 1.
syntax:parse() {
  local verb=$1 line alternatives=''
  shift
  for line in "$@"
  do
    syntax:tokens "${verb}" "${line}"
    syntax_pos=0 syntax_json='' syntax_error=''
    if ! syntax:parse-alternatives
    then
      printf '%s\n' "${syntax_error}"
      return 1
    fi
    if (( syntax_pos < ${#syntax_tokens[@]} ))
    then
      printf "stray '%s'\n" "${syntax_tokens[syntax_pos]}"
      return 1
    fi
    # each line's alternatives, without the brackets around them, join one array
    alternatives+="${alternatives:+,}${syntax_json:1:${#syntax_json}-2}"
  done
  printf '[%s]\n' "${alternatives}"
}

# syntax:tokens <verb> <usage-line> fills syntax_tokens with the words of the line after kit <verb>: ... is …, and
# each of [ ] ( ) | … is a word of its own.
syntax:tokens() {
  local text=$2
  text=${text#kit "$1"}
  text=${text//.../…}
  text=${text//\[/ [ }
  text=${text//\]/ ] }
  text=${text//\(/ ( }
  text=${text//\)/ ) }
  text=${text//|/ | }
  text=${text//…/ … }
  read -r -a syntax_tokens <<<"${text}"
}

# syntax:parse-alternatives appends the alternatives up to a closing bracket or the end, as an array of sequences.
syntax:parse-alternatives() {
  syntax_json+='['
  syntax:parse-sequence || return 1
  while (( syntax_pos < ${#syntax_tokens[@]} )) && [[ "${syntax_tokens[syntax_pos]}" == '|' ]]
  do
    syntax_pos=$(( syntax_pos + 1 ))
    syntax_json+=','
    syntax:parse-sequence || return 1
  done
  syntax_json+=']'
}

# syntax:parse-sequence appends the nodes up to a |, a closing bracket, or the end, as an array.
syntax:parse-sequence() {
  local first=1
  syntax_json+='['
  while (( syntax_pos < ${#syntax_tokens[@]} ))
  do
    case "${syntax_tokens[syntax_pos]}" in
      ('|' | ']' | ')') break ;;
    esac
    (( first )) || syntax_json+=','
    first=0
    syntax:parse-item || return 1
  done
  syntax_json+=']'
}

# syntax:parse-item appends one node: a bracketed group of alternatives, or a word, with its repeat mark.
syntax:parse-item() {
  local token=${syntax_tokens[syntax_pos]} close optional repeat=false
  case "${token}" in
    ('[' | '(')
      if [[ "${token}" == '[' ]]
      then
        close=']' optional=true
      else
        close=')' optional=false
      fi
      syntax_pos=$(( syntax_pos + 1 ))
      syntax_json+="{\"kind\":\"group\",\"optional\":${optional},\"alternatives\":"
      syntax:parse-alternatives || return 1
      if (( syntax_pos >= ${#syntax_tokens[@]} )) || [[ "${syntax_tokens[syntax_pos]}" != "${close}" ]]
      then
        syntax_error="'${token}' is not closed"
        return 1
      fi
      syntax_pos=$(( syntax_pos + 1 ))
      ;;
    (']' | ')' | '…')
      syntax_error="stray '${token}'"
      return 1
      ;;
    (*)
      syntax_pos=$(( syntax_pos + 1 ))
      syntax:parse-word "${token}"
  esac
  if (( syntax_pos < ${#syntax_tokens[@]} )) && [[ "${syntax_tokens[syntax_pos]}" == '…' ]]
  then
    repeat=true
    syntax_pos=$(( syntax_pos + 1 ))
  fi
  syntax_json+=",\"repeat\":${repeat}}"
}

# syntax:parse-word <word> appends the node of one word by its shape, left open for the repeat mark: a wildcard
# option, an inline option, a flag or an option whose value is the next word, an operand, or a literal.
syntax:parse-word() {
  local word=$1 name value
  case "${word}" in
    (--\<*\>=*)
      name=${word#--<}
      name=${name%%>=*}
      value=${word#*=}
      syntax_json+='{"kind":"option","wildcard":true,"inline":true,"name":'
      syntax:emit-string "${name}"
      syntax_json+=',"value":'
      syntax:emit-value "${value}"
      ;;
    (--?*=*)
      name=${word#--}
      name=${name%%=*}
      value=${word#*=}
      syntax_json+='{"kind":"option","inline":true,"name":'
      syntax:emit-string "${name}"
      syntax_json+=',"value":'
      syntax:emit-value "${value}"
      ;;
    (--?*)
      name=${word#--}
      if (( syntax_pos < ${#syntax_tokens[@]} )) && [[ "${syntax_tokens[syntax_pos]}" == \<*\> ]]
      then
        value=${syntax_tokens[syntax_pos]}
        syntax_pos=$(( syntax_pos + 1 ))
        syntax_json+='{"kind":"option","inline":false,"name":'
        syntax:emit-string "${name}"
        syntax_json+=',"value":'
        syntax:emit-value "${value}"
      else
        syntax_json+='{"kind":"flag","name":'
        syntax:emit-string "${name}"
      fi
      ;;
    (\<*\>)
      syntax_json+='{"kind":"operand","name":'
      syntax:emit-value "${word}"
      ;;
    (*)
      syntax_json+='{"kind":"literal","name":'
      syntax:emit-string "${word}"
  esac
}

# syntax:emit-value <word> appends the name inside an operand or value word, <name>, as a JSON string.
syntax:emit-value() {
  local word=$1
  word=${word#<}
  syntax:emit-string "${word%>}"
}

# syntax:emit-string <text> appends the text as a JSON string. A name in a usage line holds no control character,
# so escaping the backslash and the quote is enough.
syntax:emit-string() {
  local text=$1
  text=${text//\\/\\\\}
  text=${text//\"/\\\"}
  syntax_json+="\"${text}\""
}

# syntax:arguments <verb> prints the arguments object the header's indented block describes: {"<key>": {"description":
# "<text>"}}. A block line starts with two spaces, the argument as the header writes it, two or more spaces, and the
# text; a line indented four or more spaces continues the text; a blank line, or any other line, ends the block. The
# key is the argument without its value and repeat mark: --name, <name>, word, or --<hole>, as syntax.jq's key gives it.
syntax:arguments() {
  help:description "$1" | awk '
    function flush() { if (key != "") printf "%s\t%s\n", key, text; key = "" }
    /^  (--|<|[a-z])[^ ]*( [^ ]+)*  +[^ ]/ {
      flush()
      line = $0
      sub(/^  /, "", line)
      text = line
      sub(/^[^ ]+( [^ ]+)*  +/, "", text)
      key = line
      sub(/ .*$/, "", key)
      if (key ~ /^--/) sub(/=.*$/, "", key)
      sub(/(…|\.\.\.)$/, "", key)
      next
    }
    /^    / && key != "" { line = $0; sub(/^ +/, "", line); text = text " " line; next }
    { flush() }
    END { flush() }' |
    jq --raw-input --slurp 'split("\n") | map(select(. != "") | split("\t") | {key: .[0], value: {description: .[1]}})
      | from_entries'
}

# syntax:check <verb> judges the header's discipline: it warns once for a usage line that does not parse, and once
# for each argument the usage names that the block does not describe, and returns 1 when there is any finding.
syntax:check() {
  local verb=$1 spec key found=0
  if ! spec=$(syntax:spec "${verb}")
  then
    warn "kit ${verb}: usage does not parse: $(jq --raw-output .error <<<"${spec}")"
    return 1
  fi
  while IFS= read -r key
  do
    warn "kit ${verb}: ${key} has no description in the header"
    found=1
  done < <(jq --raw-output -L "${kit_jq}" 'import "syntax" as syntax; syntax::undescribed[]' <<<"${spec}")
  (( found == 0 ))
}

# syntax:match <verb> <argument>… judges the arguments against the verb's syntax, as syntax.jq's match does: a named
# flag or option is taken wherever it stands, with its value inline or as the next word; -- ends the flags; a literal
# selects its alternative, as does a flag or option given; operands fill in order, and a repeated one takes every word
# left, dash-words included. It returns 0 when the arguments complete the syntax, 1 when a required literal, operand, or
# group is missing, and 2 when a word is refused, and sets what it sorted: syntax_refusal (the whole line, on 2),
# syntax_missing (the keys that could come next, joined by " or ", on 1), syntax_literals and syntax_operands (in
# order), syntax_options (each flag or option given, by its key, to its values joined by newlines; a flag's is 1), and
# syntax_values (every operand and option key given, to its values the same way). When the arguments complete the
# syntax, a value off a closed set of choices is refused too: <key> '<value>' is not one of the choices.
syntax:match() {
  local verb=$1 syntax status=0 key name count value i n
  local -a fields usage
  shift
  syntax_refusal='' syntax_missing=''
  syntax_literals=() syntax_operands=() syntax_options=() syntax_values=()
  readarray -t usage < <(help:usage "${verb}")
  if ! syntax=$(syntax:parse "${verb}" "${usage[@]}")
  then
    syntax_refusal="kit ${verb}: usage does not parse: ${syntax}"
    return 2
  fi
  readarray -d '' -t fields < <(jq --null-input --raw-output0 -L "${kit_jq}" --arg verb "${verb}" \
    --argjson syntax "${syntax}" \
    'import "syntax" as syntax; {verb: $verb, syntax: $syntax} | syntax::match($ARGS.positional) | syntax::fields' \
    --args -- "$@")
  if (( ${#fields[@]} == 0 ))
  then
    syntax_refusal="kit ${verb}: the arguments could not be judged; run: kit help ${verb}"
    return 2
  fi
  i=0
  while (( i < ${#fields[@]} ))
  do
    key=${fields[i]}
    case "${key}" in
      (status) status=${fields[i + 1]}; i=$(( i + 2 )) ;;
      (refusal) syntax_refusal=${fields[i + 1]}; i=$(( i + 2 )) ;;
      (missing) syntax_missing=${fields[i + 1]}; i=$(( i + 2 )) ;;
      (literals | operands)
        count=${fields[i + 1]}
        for (( n = 0; n < count; n++ ))
        do
          if [[ "${key}" == literals ]]
          then
            syntax_literals+=("${fields[i + 2 + n]}")
          else
            syntax_operands+=("${fields[i + 2 + n]}")
          fi
        done
        i=$(( i + 2 + count ))
        ;;
      (option | argument)
        name=${fields[i + 1]} count=${fields[i + 2]} value=''
        for (( n = 0; n < count; n++ ))
        do
          value+="${value:+$'\n'}${fields[i + 3 + n]}"
        done
        if [[ "${key}" == option ]]
        then
          syntax_options[${name}]=${value}
        else
          syntax_values[${name}]=${value}
        fi
        i=$(( i + 3 + count ))
        ;;
      (*)
        syntax_refusal="kit ${verb}: the arguments could not be judged; run: kit help ${verb}"
        return 2
    esac
  done
  (( status == 0 )) || return "${status}"
  syntax:closed "${verb}"
}

# syntax:closed <verb> refuses, returning 2 and setting syntax_refusal, a value syntax:match sorted that is off a closed
# set of choices: the verb's module says which keys have choices and lists them, given every answer so far, and a glob
# counts when it matches one. An open set, or a key the module does not answer, is never refused.
syntax:closed() {
  local verb=$1 key value kind choices
  local -a answers=()
  syntax:answers "${verb}" choices || return 0
  for key in "${!syntax_values[@]}"
  do
    while IFS= read -r value
    do
      answers+=("${key}=${value}")
    done <<<"${syntax_values[${key}]}"
  done
  for key in "${!syntax_values[@]}"
  do
    kind=$(syntax:kind "${verb}" "${key}") || continue
    [[ "${kind}" == closed ]] || continue
    choices=$(syntax:choices "${verb}" "${key}" "${answers[@]}") || continue
    choices=$(cut -f 1 <<<"${choices}")
    while IFS= read -r value
    do
      syntax:offered "${value}" "${choices}" && continue
      syntax_refusal="${key} '${value}' is not one of the choices; run: kit help ${verb}"
      return 2
    done <<<"${syntax_values[${key}]}"
  done
  return 0
}

# syntax:offered <value> <choices> succeeds when the value is one of the choices, one per line, or is a glob that
# matches one of them, as kit package expands a glob.
syntax:offered() {
  local choice
  while IFS= read -r choice
  do
    [[ -n "${choice}" ]] || continue
    if [[ "$1" == *[*?[]* ]]
    then
      # shellcheck disable=SC2053  # the value is a glob on purpose
      [[ "${choice}" == $1 ]] && return 0
    else
      [[ "${choice}" == "$1" ]] && return 0
    fi
  done <<<"$2"
  return 1
}

# syntax:answers <verb> <name> succeeds when a module named after the verb defines <verb>:<name>, loading the module
# first, so a verb's choices and extensions live beside its code and the syntax itself never names them.
syntax:answers() {
  use:load-path || return 2
  [[ -n "$(use:find "$1")" ]] || return 1
  use "$1" || return 2
  declare -F "$1:$2" >/dev/null
}

# syntax:kind <verb> <key> prints closed or open when the verb's module answers choices for the key, and nothing,
# returning 1, when it does not.
syntax:kind() {
  syntax:answers "$1" choices || return 1
  "$1:choices" "$2" --kind
}

# syntax:choices <verb> <key> [<answer>…] prints the choices of one argument, one row per value, tab-separated: the
# value, its description, its state (active, excluded, or empty), and the line the verb would print about it given
# the answers so far, each <key>=<value>. It returns 1 when the verb's module answers nothing for the key.
syntax:choices() {
  local verb=$1
  shift
  syntax:answers "${verb}" choices || return 1
  "${verb}:choices" "$@"
}

# syntax:extend <verb> [<answer>…] prints the arguments the verb learns from the answers so far, each <key>=<value>,
# as an object of arguments entries marked extended, for a form to add as fields; kit new answers with its
# template's holes. It returns 1 when the verb's module has no <verb>:extend.
syntax:extend() {
  local verb=$1
  shift
  syntax:answers "${verb}" extend || return 1
  "${verb}:extend" "$@"
}

# syntax:kinds <verb> <arguments-json> prints {"<key>": "closed" | "open"} for every key of the arguments object the
# verb's module answers choices for, and {} when it answers none.
syntax:kinds() {
  local verb=$1 key kind
  if ! syntax:answers "${verb}" choices
  then
    printf '{}\n'
    return 0
  fi
  while IFS= read -r key
  do
    kind=$("${verb}:choices" "${key}" --kind) || continue
    printf '%s\t%s\n' "${key}" "${kind}"
  done < <(jq --raw-output 'keys[]' <<<"$2") |
    jq --raw-input --slurp 'split("\n") | map(select(. != "") | split("\t") | {key: .[0], value: .[1]}) | from_entries'
}

# syntax:render <json> prints a node, or a sequence of nodes, as usage text: --message <text>…, [--all],
# (--message <text>… | --file <path>), list, <path>….
syntax:render() {
  jq --raw-output -L "${kit_jq}" 'import "syntax" as syntax; syntax::render' <<<"$1"
}

# syntax:key <node-json> prints the key an argument is known by, as the arguments object is keyed: --name for a flag
# or an option, --<name> for a wildcard option, <name> for an operand, and the word itself for a literal.
syntax:key() {
  jq --raw-output -L "${kit_jq}" 'import "syntax" as syntax; syntax::key' <<<"$1"
}

# ----------------------------------------------------------------------------------------------------------------------
