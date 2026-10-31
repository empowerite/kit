# shellcheck shell=bash
# syntax.sh - parses a verb's usage lines into its syntax, as JSON, and matches a command line against it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
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

# syntax:spec <verb> prints the verb's syntax as the kit syntax JSON: verb, purpose, usage, syntax, and arguments, the
# descriptions in the header's argument block. When a usage line does not parse, the JSON carries error instead of
# syntax, and it returns 1.
syntax:spec() {
  local verb=$1 syntax
  local -a usage
  readarray -t usage < <(help:usage "${verb}")
  if ! syntax=$(syntax:parse "${verb}" "${usage[@]}")
  then
    jq --null-input --arg verb "${verb}" --arg error "${syntax}" \
      '{verb: $verb, usage: $ARGS.positional, error: $error}' --args "${usage[@]}"
    return 1
  fi
  jq --null-input --arg verb "${verb}" --arg purpose "$(help:purpose "${verb}")" --argjson syntax "${syntax}" \
    --argjson arguments "$(syntax:arguments "${verb}")" \
    '{verb: $verb, purpose: $purpose, usage: $ARGS.positional, syntax: $syntax, arguments: $arguments}' --args "${usage[@]}"
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
