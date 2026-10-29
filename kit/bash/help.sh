# shellcheck shell=bash
# help.sh - the verbs' help, read from each verb's own header (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A verb is an executable kit-<verb> beside bin/kit. Its header is the only place its help is written:
#
#   # kit-<verb> - <purpose> (kit verb)        the banner, on line 2
#   # ---…                                     a line of dashes, if the verb has one
#   #
#   # kit <verb> <arguments>                   a usage line: a paragraph of one line that ends without . or :
#   #
#   # <description>                            every other paragraph, up to the first line that is not a comment
#
# A verb with no usage line takes no arguments, and its usage is kit <verb>. A `see:` line points into the kit's
# source, and is left out of the help.

# help:verbs [<pattern>] prints the name of every verb that matches, sorted: <pattern> itself when it names a verb,
# otherwise every verb whose name starts with <pattern>. With no pattern, it prints every verb.
help:verbs() {
  local pattern=${1:-} file
  # shellcheck disable=SC2154  # kit_root is exported by bash/1st.sh
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
  sed -n '2{s/^# kit-[a-z-]* - //; s/ (kit verb)$//; p;}' "${kit_root}/bin/kit-$1"
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

# help:page <verb> prints the verb's help page: its name and purpose, its usage, and its description. Headings are
# bold when stdout is a terminal and NO_COLOR is not set.
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
}

# ----------------------------------------------------------------------------------------------------------------------
