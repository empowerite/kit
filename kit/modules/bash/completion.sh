# shellcheck shell=bash
# completion.sh - shell completion for kit: the zsh script, and the words a script offers, read from the verbs (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root and kit_jq are exported by modules/bash/1st.sh
#
# The completion script knows no verb. On each completion it runs kit completion candidates with the words typed so
# far, and offers what comes back. Those candidates are read from the verbs themselves: the verbs and their purposes
# from their headers, and a verb's flags, options, literal words, and operands from kit syntax, with the values a
# verb's module lists as an argument's choices. So a new verb, flag, or choice completes with no edit here.
#
# see: modules/bash/syntax.sh, modules/jq/completion.jq, docs/completion.md

use help syntax

# completion:zsh prints the zsh completion script. It completes kit, and every kit-<verb> command called directly.
completion:zsh() {
  cat <<'ZSH'
#compdef kit -p kit-*
# The completion for kit, printed by kit completion zsh. It asks kit for the candidates on every completion.

_kit() {
  local -a typed lines plain inline
  local line word files=0
  typed=("${(@)words[2,CURRENT]}")
  [[ "${service}" == kit-* ]] && typed=("${service#kit-}" "${(@)typed}")
  lines=("${(@f)$(kit completion candidates kit "${(@)typed}" 2>/dev/null)}")
  for line in "${(@)lines}"
  do
    [[ -n "${line}" ]] || continue
    if [[ "${line}" == ':files' ]]
    then
      files=1
      continue
    fi
    word=${line%%$'\t'*}
    word=${word//:/\\:}
    if [[ "${word}" == *= ]]
    then
      inline+=("${word}:${line#*$'\t'}")
    else
      plain+=("${word}:${line#*$'\t'}")
    fi
  done
  (( ${#plain} )) && _describe -t kit 'kit' plain
  (( ${#inline} )) && _describe -t kit-options 'option' inline -S ''
  (( files )) && _files
  return 0
}

# Autoloaded from fpath as _kit, the file is the function's body and completes now; evaluated from .zshrc, it registers.
if [[ "${funcstack[1]}" == _kit ]]
then
  _kit "$@"
else
  compdef _kit kit
  compdef -p _kit 'kit-*'
fi
ZSH
}

# completion:candidates kit <word>… prints what the last word may be, after the words before it, one candidate per
# line: the word, a tab, and its description. A line :files asks the shell to offer the files as well. The first
# word is kit itself, so a word that starts with a dash is never the first.
completion:candidates() {
  shift
  local verb purpose current spec next mode key value prefix operand
  local -a typed operands
  if (( $# <= 1 ))
  then
    while IFS=$'\t' read -r verb _ purpose
    do
      printf '%s\t%s\n' "${verb}" "${purpose}"
    done < <(help:banners)
    return 0
  fi
  verb=$1
  shift
  [[ -x "${kit_root}/bin/kit-${verb}" ]] || return 0
  typed=("${@:1:$# - 1}")
  current=${!#}
  spec=$(syntax:spec "${verb}") || return 0
  next=$(jq --compact-output -L "${kit_jq}" --arg current "${current}" \
    'import "completion" as completion; completion::next($ARGS.positional; $current)' \
    --args -- "${typed[@]}" <<<"${spec}")
  IFS=$'\t' read -r mode key value prefix < <(jq --raw-output '[.mode, .key // "-", .value // "-", .prefix // ""] | @tsv' <<<"${next}")
  # A description is cut to its first sentence, so a list stays one line per candidate.
  jq --raw-output '.candidates // [] | .[] | [.[0], (.[1] | sub("\\. .*"; ""))] | @tsv' <<<"${next}"
  case "${mode}" in
    (value) completion:values "${verb}" "${key}" "${value}" "${prefix}" ;;
    (positional)
      readarray -t operands < <(jq --raw-output '.operands // [] | .[]' <<<"${next}")
      for operand in "${operands[@]}"
      do
        completion:values "${verb}" "${operand}" "${operand}" ''
      done
  esac
  return 0
}

# completion:values <verb> <key> <value> <prefix> prints the choices the verb's module lists for the argument, each
# after <prefix>, or :files when it lists none and the value names a path, a file, or a directory.
completion:values() {
  local choices
  if choices=$(syntax:choices "$1" "$2" 2>/dev/null) && [[ -n "${choices}" ]]
  then
    cut -f 1,2 <<<"${choices}" | sed "s|^|$4|"
  elif [[ "$3" =~ (^|[<-])(path|file|dir|directory)s?(>|$) ]]
  then
    printf ':files\n'
  fi
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
