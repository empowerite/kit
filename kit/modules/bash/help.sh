# shellcheck shell=bash
# help.sh - the verbs' help, read from each verb's own header (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A verb is an executable kit-<verb> beside bin/kit. Its header is the only place its help is written:
#
#   # kit-<verb> - <purpose> (kit verb: <group>)   the banner, on line 2
#   # ---…                                     a line of dashes, if the verb has one
#   #
#   # kit <verb> <arguments>                   a usage line: a paragraph of one line that ends without . or :
#   #
#   # <description>                            every other paragraph, up to the first line that is not a comment
#
# A verb with no usage line takes no arguments, and its usage is kit <verb>. A `see:` line points into the kit's
# source, and is left out of the help. The group in the banner is one of help_groups, and kit help lists the verbs
# under it.

# The groups kit help lists the verbs under, in the order it lists them.
declare -gra help_groups=(workflow files checks installation interfaces help)

# help:verbs [<pattern>] prints the name of every verb that matches, sorted: <pattern> itself when it names a verb,
# otherwise every verb whose name starts with <pattern>. With no pattern, it prints every verb.
help:verbs() {
  local pattern=${1:-} file
  # shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
  if [[ -n "${pattern}" && -x "${kit_root}/bin/kit-${pattern}" ]]
  then
    printf '%s\n' "${pattern}"
    return 0
  fi
  while IFS= read -r file
  do
    [[ -x "${file}" ]] && printf '%s\n' "${file##*/kit-}"
  done < <(compgen -G "${kit_root}/bin/kit-${pattern}*") | LC_ALL=C sort
}

# help:purpose <verb> prints the purpose in the verb's banner.
help:purpose() {
  sed -n '2{s/^# kit-[a-z-]* - //; s/ (kit verb[^)]*)$//; p;}' "${kit_root}/bin/kit-$1"
}

# help:group <verb> prints the group in the verb's banner, or nothing when it names none.
help:group() {
  sed -n '2{s/^.* (kit verb: \([a-z]*\))$/\1/p;}' "${kit_root}/bin/kit-$1"
}

# help:header <verb> prints the verb's header text below the banner, without the comment marks, and without a line of
# dashes, the blank lines at either end, and the see: lines.
help:header() {
  awk '
    NR <= 2 { next }
    /^# -+$/ { next }
    !/^#/ { exit }
    /^# see: / { next }
    { sub(/^# ?/, ""); line[++n] = $0 }
    END {
      first = 1; while (first <= n && line[first] == "") first++
      last = n; while (last >= first && line[last] == "") last--
      for (i = first; i <= last; i++) print line[i]
    }' "${kit_root}/bin/kit-$1"
}

# help:usage <verb> prints the verb's usage lines, or kit <verb> when it has none.
help:usage() {
  local usage
  usage=$(help:header "$1" | awk -v verb="$1" '
    function flush() { if (count == 1 && para ~ "^kit " verb "( |$)" && para !~ /[.:]$/) print para; count = 0 }
    $0 == "" { flush(); next }
    { para = $0; count++ }
    END { flush() }')
  printf '%s\n' "${usage:-kit $1}"
}

# help:description <verb> prints the verb's header without its usage lines.
help:description() {
  help:header "$1" | awk -v verb="$1" '
    function flush(   i) {
      if (count == 1 && para[1] ~ "^kit " verb "( |$)" && para[1] !~ /[.:]$/) { count = 0; return }
      if (count > 0 && printed) print ""
      for (i = 1; i <= count; i++) print para[i]
      if (count > 0) printed = 1
      count = 0
    }
    $0 == "" { flush(); next }
    { para[++count] = $0 }
    END { flush() }'
}

# help:summary <verb>… prints each verb's usage and purpose, one block per verb, as kit help lists them. A single
# usage line short enough shares its line with the purpose; otherwise the purpose follows the usage lines.
help:summary() {
  local verb line purpose
  local -a usage
  for verb in "$@"
  do
    readarray -t usage < <(help:usage "${verb}")
    purpose=$(help:purpose "${verb}")
    line=${usage[0]#kit }
    if (( ${#usage[@]} == 1 && ${#line} <= 20 ))
    then
      # printf pads by bytes, and a usage line may hold a multibyte …, so the padding is counted in characters.
      printf '  %s%*s  %s\n' "${line}" $(( 20 - ${#line} )) '' "${purpose}"
      continue
    fi
    for line in "${usage[@]}"
    do
      printf '  %s\n' "${line#kit }"
    done
    printf '  %-20s  %s\n' '' "${purpose}"
  done
}

# help:overview prints kit help's page: what kit is, its usage, each verb's name and purpose under its group, a few
# examples, and where to learn more, as gh help lays itself out. Headings are bold when stdout is a
# terminal and NO_COLOR is not set.
help:overview() {
  local bold='' plain='' group verb
  local -a verbs members
  if [[ -t 1 && -z "${NO_COLOR:-}" ]]
  then
    bold=$'\x1b[1m' plain=$'\x1b[0m'
  fi
  readarray -t verbs < <(help:verbs)
  printf 'kit keeps the rules a repository asks its agents to follow, and the checks that hold the work to them.\n\n'
  printf '%sUSAGE%s\n  kit <verb> [<argument>…]\n' "${bold}" "${plain}"
  for group in "${help_groups[@]}" other
  do
    members=()
    for verb in "${verbs[@]}"
    do
      if [[ "$(help:group "${verb}")" == "${group}" ]] \
        || [[ "${group}" == other && " ${help_groups[*]} " != *" $(help:group "${verb}") "* ]]
      then
        members+=("${verb}")
      fi
    done
    (( ${#members[@]} )) || continue
    printf '\n%s%s%s\n' "${bold}" "${group^^}" "${plain}"
    for verb in "${members[@]}"
    do
      printf '  %-12s %s\n' "${verb}:" "$(help:purpose "${verb}")"
    done
  done
  printf '\n%sEXAMPLES%s\n' "${bold}" "${plain}"
  printf '  kit issue 42\n  kit context src/main.sh\n  kit commit --message %s\n' "'#42 area: subject'"
  printf '\n%sLEARN MORE%s\n' "${bold}" "${plain}"
  printf '  kit help <verb> prints a verb'"'"'s page, kit usage <verb> its usage lines, and kit syntax <verb> its grammar\n'
  printf '  as JSON. .kit/1st.md describes the workflow.\n'
}

# help:page <verb> prints the verb's help page: its name and purpose, its usage, its description, and the other verbs
# of its group. Headings are bold when stdout is a terminal and NO_COLOR is not set.
help:page() {
  local bold='' plain=''
  if [[ -t 1 && -z "${NO_COLOR:-}" ]]
  then
    bold=$'\x1b[1m' plain=$'\x1b[0m'
  fi
  printf '%skit %s%s - %s\n\n' "${bold}" "$1" "${plain}" "$(help:purpose "$1")"
  printf '%sUsage%s\n' "${bold}" "${plain}"
  help:usage "$1" | sed 's/^/  /'
  printf '\n%sDescription%s\n' "${bold}" "${plain}"
  help:description "$1" | sed 's/^./  &/'
  local group verb others=''
  group=$(help:group "$1")
  [[ -n "${group}" ]] || return 0
  for verb in $(help:verbs)
  do
    [[ "${verb}" != "$1" && "$(help:group "${verb}")" == "${group}" ]] && others+="${others:+, }kit ${verb}"
  done
  printf '\n%sGroup%s\n  %s%s\n' "${bold}" "${plain}" "${group}" "${others:+: also ${others}}"
}

# help:choices <key> [<answer>…] answers kit syntax --choices for <verb>: an open set, every verb with its purpose, since
# kit help takes the start of a name too.
help:choices() {
  local verb
  [[ "$1" == '<verb>' ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'open\n'
    return 0
  fi
  for verb in $(help:verbs)
  do
    printf '%s\t%s\t\t\n' "${verb}" "$(help:purpose "${verb}")"
  done
}

# ----------------------------------------------------------------------------------------------------------------------
