# shellcheck shell=bash
# help.sh - the verbs' help, read from each verb's own header (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A verb is an executable kit-<verb> beside bin/kit. Its header is the only place its help is written:
#
#   # kit-<verb> - <purpose> (kit verb: <group>)   the banner, on line 2
#   # ---…                                     a line of dashes, if the verb has one
#   # kit <verb> <arguments>                   a usage line: a paragraph of one line that ends without . or :
#   #
#   # <description>                            every other paragraph, up to the first line that is not a comment
#
# A verb with no usage line takes no arguments, and its usage is kit <verb>. A `see:` line points into the kit's
# source, and is left out of the help. The group in the banner is one of help_groups, and kit help lists the verbs
# under it.

# The groups kit help lists the verbs under, in the order it lists them. The last, plumbing, holds the verbs that
# scripts, the hooks, and the TUI call rather than a person: kit help sums them up in one line, and kit help plumbing
# lists them.
declare -gra help_groups=(workflow files checks installation interfaces help plumbing)

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

# help:banners prints verb, group, and purpose, tab-separated, for every verb, sorted, read from the banners in one
# awk pass over every verb's file: a listing of every verb costs one process, where help:purpose and help:group cost
# one each, and a caller reads it once. The group is empty where the banner names none.
help:banners() {
  local verb
  local -a files=()
  for verb in $(help:verbs)
  do
    files+=("${kit_root}/bin/kit-${verb}")
  done
  (( ${#files[@]} )) || return 0
  awk '
    FNR == 2 {
      verb = FILENAME; sub(/.*\/kit-/, "", verb)
      purpose = $0; sub(/^# kit-[a-z-]* - /, "", purpose)
      group = ""
      if (match(purpose, / \(kit verb: [a-z]*\)$/)) group = substr(purpose, RSTART + 12, RLENGTH - 13)
      sub(/ \(kit verb[^)]*\)$/, "", purpose)
      printf "%s\t%s\t%s\n", verb, group, purpose
    }' "${files[@]}"
}

# help:members <group> prints the name of every verb whose banner names the group, sorted, and for other, every verb
# whose banner names no group of help_groups.
help:members() {
  local verb group purpose
  while IFS=$'\t' read -r verb group purpose
  do
    if [[ "${group}" == "$1" ]] || [[ "$1" == other && " ${help_groups[*]} " != *" ${group} "* ]]
    then
      printf '%s\n' "${verb}"
    fi
  done < <(help:banners)
}

# help:header <verb> prints the verb's header text below the banner, without the comment marks, and without a line of
# dashes, the blank lines at either end, the see: lines, and the lines that open and close a region kit generate fills,
# as the legend kit-flow's header has.
help:header() {
  awk '
    NR <= 2 { next }
    /^# -+$/ { next }
    !/^#/ { exit }
    /^# see: / { next }
    /^# kit:(generate|end) / { next }
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
  local verb line purpose group
  local -a usage
  local -A purposes=()
  while IFS=$'\t' read -r verb group purpose
  do
    purposes[${verb}]=${purpose}
  done < <(help:banners)
  for verb in "$@"
  do
    readarray -t usage < <(help:usage "${verb}")
    purpose=${purposes[${verb}]:-}
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

# help:reflow prints stdin as a page's description, indented two spaces, with every line rewrapped to 80 columns. A
# paragraph is consecutive lines that start at column 0, whose words are joined and laid out again. A bullet at
# column 0, `- …`, `• …`, or `1. …`, starts its own paragraph, hanging after its mark, and the lines after it that are
# not bullets continue it. An indented line starts an entry, and the lines indented deeper than it continue it: an
# argument, `  <token>   description`, whose description hangs at its own column; a bullet, `  - …` or `  1. …`, whose
# text hangs after the mark; and any other indented line, hanging at its indent. A blank line separates them. Nothing
# printed exceeds 80 columns but a word wider than its room. Width is counted in terminal columns: awk runs in the C
# locale, where a byte that continues a multibyte character is told by its value, and those are left out of the
# count; an emoji, four bytes or one of the few of three that terminals draw as one, counts two, so the text after a
# glyph hangs under its own column.
help:reflow() {
  LC_ALL=C awk '
    function width(s,   wide) {
      wide = gsub(/[\360-\364]|⌚|⌛|⏩|⏪|⏫|⏬|⏰|⏳|✅|✊|✋|✨|❌|❎|❓|❔|❕|❗|➕|➖|➗|⬛|⬜|⭐|⭕/, "&", s)
      gsub(/[\200-\277]/, "", s)
      return length(s) + wide
    }
    function emit(prefix, hang, text,   i, n, word, line, room) {
      room = 78 - hang
      n = split(text, word, " ")
      line = ""
      for (i = 1; i <= n; i++) {
        if (line == "") line = word[i]
        else if (width(line) + 1 + width(word[i]) <= room) line = line " " word[i]
        else { print "  " prefix line; line = word[i]; prefix = sprintf("%" hang "s", "") }
      }
      print "  " prefix line
    }
    function flush() {
      if (text == "") return
      emit(prefix, hang, text)
      text = ""; prefix = ""; hang = 0; indent = -1
    }
    BEGIN { indent = -1 }
    /^$/ { flush(); print ""; next }
    /^ / {
      match($0, /^ +/)
      if (indent >= 0 && RLENGTH > indent) { line = $0; sub(/^ +/, "", line); text = text " " line; next }
      flush()
      indent = RLENGTH
      if (match($0, /^ +(- |[0-9]+\. )/) || match($0, /^ +[^ ]+( [^ ]+)*  +/)) prefix = substr($0, 1, RLENGTH)
      else prefix = substr($0, 1, indent)
      hang = width(prefix)
      text = substr($0, length(prefix) + 1)
      next
    }
    /^(- |\342\200\242 |[0-9]+\. )/ {
      flush()
      match($0, /^(- |\342\200\242 |[0-9]+\. )/)
      indent = 0
      prefix = substr($0, 1, RLENGTH)
      hang = width(prefix)
      text = substr($0, RLENGTH + 1)
      next
    }
    { if (indent > 0) flush(); text = text (text == "" ? "" : " ") $0 }
    END { flush() }'
}

# help:styled is true when a help page's headings are bold: NO_COLOR is not set, and stdout is a terminal or
# CLICOLOR_FORCE asks for the effects anyway, as kit tui does for the page it shows beside the verbs.
help:styled() {
  [[ -z "${NO_COLOR:-}" ]] || return 1
  [[ -t 1 ]] || [[ -n "${CLICOLOR_FORCE:-}" && "${CLICOLOR_FORCE}" != 0 ]]
}

# help:heading <group> prints the group's heading, its name in capitals, bold as help:styled says.
help:heading() {
  if help:styled
  then
    printf '\x1b[1m%s\x1b[0m\n' "${1^^}"
  else
    printf '%s\n' "${1^^}"
  fi
}

# help:listing <group> <verb>… prints the group's heading and each verb named with its purpose, one per line, as kit
# help lists a group.
help:listing() {
  local group=$1 verb named purpose
  local -A purposes=()
  shift
  while IFS=$'\t' read -r verb named purpose
  do
    purposes[${verb}]=${purpose}
  done < <(help:banners)
  help:heading "${group}"
  for verb in "$@"
  do
    printf '  %-12s %s\n' "${verb}" "${purposes[${verb}]:-}"
  done
}

# help:overview prints kit help's page: what kit is, its usage, each verb's name and purpose under its group, the
# plumbing summed up in one line, a few examples, and where to learn more, as gh help lays itself out. Headings are
# bold as help:styled says. The banners are read once, and each verb goes under its group, or under other when its
# banner names no group of help_groups.
help:overview() {
  local bold='' plain='' group verb purpose
  local -a members
  local -A purposes=() members_of=()
  if help:styled
  then
    bold=$'\x1b[1m' plain=$'\x1b[0m'
  fi
  while IFS=$'\t' read -r verb group purpose
  do
    purposes[${verb}]=${purpose}
    [[ " ${help_groups[*]} " == *" ${group} "* ]] || group=other
    members_of[${group}]+="${verb} "
  done < <(help:banners)
  printf 'kit keeps the rules a repository asks its agents to follow, and the checks that\nhold the work to them.\n\n'
  printf '%sUSAGE%s\n  kit <verb> [<argument>…]\n' "${bold}" "${plain}"
  for group in "${help_groups[@]}" other
  do
    read -r -a members <<<"${members_of[${group}]:-}"
    (( ${#members[@]} )) || continue
    printf '\n'
    if [[ "${group}" == plumbing ]]
    then
      printf '%sPLUMBING%s\n  %d verbs that scripts, hooks, and the TUI call; kit help plumbing lists them\n' \
        "${bold}" "${plain}" "${#members[@]}"
      continue
    fi
    help:heading "${group}"
    for verb in "${members[@]}"
    do
      printf '  %-12s %s\n' "${verb}" "${purposes[${verb}]}"
    done
  done
  printf '\n%sEXAMPLES%s\n' "${bold}" "${plain}"
  printf '  kit issue 42\n  kit context src/main.sh\n  kit commit --message %s\n' "'#42 area: subject'"
  printf '\n%sLEARN MORE%s\n' "${bold}" "${plain}"
  printf '  • kit help <verb>… prints more about each verb, as kit <verb> --help does: its\n'
  printf '    usage, what it does, and the verbs beside it\n'
  printf '  • kit help <group> lists one group'"'"'s verbs; kit help plumbing the verbs\n'
  printf '    scripts and the TUI call\n'
  printf '  • .kit/1st.md is what every agent reads first; .kit/workflow.md, when the\n'
  printf '    workflow package is active, is the workflow\n'
}

# help:page <verb> prints the verb's help page: its name and purpose, its usage, its description rewrapped to 80
# columns, and the other verbs of its group, wrapped at 80 columns between verbs. Headings are bold as help:styled
# says.
help:page() {
  local bold='' plain='' line verb group named own='' purpose='' other
  local -a banners others=()
  if help:styled
  then
    bold=$'\x1b[1m' plain=$'\x1b[0m'
  fi
  readarray -t banners < <(help:banners)
  for line in "${banners[@]}"
  do
    IFS=$'\t' read -r verb group named <<<"${line}"
    [[ "${verb}" == "$1" ]] || continue
    own=${group} purpose=${named}
    break
  done
  printf '%skit %s%s - %s\n\n' "${bold}" "$1" "${plain}" "${purpose}"
  printf '%sUsage%s\n' "${bold}" "${plain}"
  help:usage "$1" | sed 's/^/  /'
  printf '\n%sDescription%s\n' "${bold}" "${plain}"
  help:description "$1" | help:reflow
  [[ -n "${own}" ]] || return 0
  for line in "${banners[@]}"
  do
    IFS=$'\t' read -r verb group named <<<"${line}"
    [[ "${verb}" != "$1" && "${group}" == "${own}" ]] && others+=("kit ${verb}")
  done
  printf '\n%sGroup%s\n' "${bold}" "${plain}"
  if (( ${#others[@]} == 0 ))
  then
    printf '  %s\n' "${own}"
    return 0
  fi
  line="  ${own}: also ${others[0]}"
  for other in "${others[@]:1}"
  do
    if (( ${#line} + 2 + ${#other} <= 80 ))
    then
      line+=", ${other}"
    else
      printf '%s,\n' "${line}"
      printf -v line '%*s%s' $(( ${#own} + 9 )) '' "${other}"
    fi
  done
  printf '%s\n' "${line}"
}

# help:choices <key> [<answer>…] answers kit syntax --choices for <verb>: an open set, every verb with its purpose, since
# kit help takes the start of a name too.
help:choices() {
  local verb group purpose
  [[ "$1" == '<verb>' ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'open\n'
    return 0
  fi
  while IFS=$'\t' read -r verb group purpose
  do
    printf '%s\t%s\t\t\n' "${verb}" "${purpose}"
  done < <(help:banners)
}

# ----------------------------------------------------------------------------------------------------------------------
