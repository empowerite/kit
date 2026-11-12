# shellcheck shell=bash
# help.sh - the verbs' help, read from each verb's own header (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A verb is an executable kit-<verb> beside bin/kit. Its header is the only place its help is written, in the shape
# of the verb template, packages/kit/files/repo/.kit/templates/bash/verb:
#
#   # kit-<verb> - <purpose> (kit verb: <group>)   the banner, on line 2
#   # Usage:                                     then, indented, a usage line for each way to call it
#   # <description>                              the paragraphs that say what it does
#   # Arguments:                                 then, indented, each argument and what it means
#   # Exits <statuses>                           what each exit status means
#   # see: <paths>                               where in the kit's source to read more, left out of the help
#
# The header is read through the template, every field in one jq pass, and a line of dashes is dropped first, so a
# house style that adds one reads the same. A header that leaves the template is refused, naming the template line,
# and kit lint refuses it before it ships. The group in the banner is one of help_groups, and kit help lists the verbs
# under it.

# The groups kit help lists the verbs under, in the order it lists them. The last, plumbing, holds the verbs that
# scripts, the hooks, and the TUI call rather than a person: kit help sums them up in one line, and kit help plumbing
# lists them.
declare -gra help_groups=(workflow files checks installation interfaces help plumbing)

# The fields of each verb's header help:values has read, keyed <verb>:<field>: purpose, group, usage, description,
# arguments, exits, and see.
declare -gA help_value=()

# help:values <verb>… reads the named verbs' headers through the verb template, in one jq pass over the ones not read
# yet, into help_value. It fails, naming the verb and the template line, when a header leaves the template.
# shellcheck disable=SC2154  # kit_root and kit_jq are exported by modules/bash/1st.sh
help:values() {
  local verb field value
  local -a files=()
  local -A named=()
  for verb in "$@"
  do
    # a verb named twice is read once: jq joins a file's lines by its name, so a second copy would double its text
    [[ -z "${named[${verb}]+set}" && -z "${help_value[${verb}:purpose]+set}" ]] || continue
    named[${verb}]=1
    files+=("${kit_root}/bin/kit-${verb}")
  done
  (( ${#files[@]} )) || return 0
  while IFS= read -r -d '' verb && IFS= read -r -d '' field && IFS= read -r -d '' value
  do
    if [[ "${field}" == error ]]
    then
      error "kit-${verb}: its header leaves the verb template at line ${value}"
      return 1
    fi
    help_value[${verb}:${field}]=${value}
  # awk hands jq each file's header alone, up to the template's set -o line, each line after its file's name and a
  # tab, and drops a line of dashes, so a house style that adds one reads the same
  done < <(awk '
    FNR == 1 { done = 0 }
    done || /^# -+$/ { next }
    { print FILENAME "\t" $0 }
    $0 == "set -o errexit -o nounset -o pipefail" { done = 1 }' "${files[@]}" |
    jq --null-input --raw-input --join-output -L "${kit_jq}" \
      --rawfile template "${kit_root}/packages/kit/files/repo/.kit/templates/bash/verb" '
      import "template" as template;
      ($template | template::parts) as $parts
      | reduce (inputs | capture("^(?<file>[^\t]*)\t(?<line>.*)$")) as $l ({}; .[$l.file] += $l.line + "\n")
      | to_entries[]
      | (.key | sub(".*/kit-"; "")) as $verb | .value as $text
      | ($parts | template::bind_parts({name: "kit-\($verb)"}) | template::match($text)) as $m
      | if $m.ok then $m.values | to_entries[] | select(.key != "body") | [$verb, .key, .value]
        else [$verb, "error", ($m.line // "end" | tostring)] end
      | map(. + "\u0000") | add')
  for verb in "$@"
  do
    if [[ -z "${help_value[${verb}:purpose]+set}" ]]
    then
      error "kit-${verb}: its header could not be read"
      return 1
    fi
    # a verb with no usage line could not be called, so its header is refused as one that leaves the template is
    if [[ -z "${help_value[${verb}:usage]//[[:space:]]/}" ]]
    then
      error "kit-${verb}: its Usage: block holds no usage line; a verb that takes no arguments has kit ${verb}"
      unset "help_value[${verb}:purpose]"
      return 1
    fi
  done
}

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
  help:values "$1" || return 1
  printf '%s\n' "${help_value[$1:purpose]}"
}

# help:group <verb> prints the group in the verb's banner.
help:group() {
  help:values "$1" || return 1
  printf '%s\n' "${help_value[$1:group]}"
}

# help:all reads every verb's header, in the shell that calls it, so the readers a caller runs in a subshell after it
# find them read, and a header that leaves the template stops the caller before it prints anything.
help:all() {
  local -a verbs
  readarray -t verbs < <(help:verbs)
  (( ${#verbs[@]} == 0 )) || help:values "${verbs[@]}"
}

# help:banners prints verb, group, and purpose, tab-separated, for every verb, sorted, read in one pass over every
# verb's header: a listing of every verb costs one process, and a caller reads it once.
help:banners() {
  local verb
  local -a verbs
  readarray -t verbs < <(help:verbs)
  (( ${#verbs[@]} )) || return 0
  help:values "${verbs[@]}" || return 1
  for verb in "${verbs[@]}"
  do
    printf '%s\t%s\t%s\n' "${verb}" "${help_value[${verb}:group]}" "${help_value[${verb}:purpose]}"
  done
}

# help:members <group> prints the name of every verb whose banner names the group, sorted, and for other, every verb
# whose banner names no group of help_groups.
help:members() {
  help:all || return 1
  local verb group purpose
  while IFS=$'\t' read -r verb group purpose
  do
    if [[ "${group}" == "$1" ]] || [[ "$1" == other && " ${help_groups[*]} " != *" ${group} "* ]]
    then
      printf '%s\n' "${verb}"
    fi
  done < <(help:banners)
}

# help:usage <verb> prints the verb's usage lines, the Usage: block's.
help:usage() {
  help:values "$1" || return 1
  grep --invert-match '^$' <<<"${help_value[$1:usage]}" || true  # a block with no line has nothing to print
}

# help:arguments <verb> prints the verb's Arguments: block, each line indented two spaces, as the description shows
# it; nothing for a verb that takes none.
help:arguments() {
  help:values "$1" || return 1
  [[ -n "${help_value[$1:arguments]}" ]] || return 0
  awk '{ print ($0 == "" ? "" : "  " $0) }' <<<"${help_value[$1:arguments]%$'\n'}"
}

# help:description <verb> prints what the verb's help page says below its usage: the description, without the lines
# that open and close a region kit generate fills, as the legend kit-flow's header has; then the argument block; then
# Exits. Paragraphs are one blank line apart.
help:description() {
  local arguments
  help:values "$1" || return 1
  arguments=$(help:arguments "$1") || return 1
  {
    # a description of region lines alone leaves nothing, which is no failure
    grep --invert-match --extended-regexp '^kit:(generate|end) ' <<<"${help_value[$1:description]}" || true
    printf '\n%s\n\nExits %s\n' "${arguments}" "${help_value[$1:exits]//$'\n# '/$'\n'}"
  } | awk '
    $0 == "" { blank = printed; next }
    { if (blank) print ""; blank = 0; print; printed = 1 }'
}

# help:summary <verb>… prints each verb's usage and purpose, one block per verb, as kit help lists them. A single
# usage line short enough shares its line with the purpose; otherwise the purpose follows the usage lines.
help:summary() {
  help:all || return 1
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
  help:all || return 1
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
  help:all || return 1
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
  printf '  kit work 42\n  kit context src/main.sh\n  kit commit --message %s\n' "'#42 area: subject'"
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
  help:all || return 1
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
  # the kind reads no header, so a broken one is refused here, by name, and never read as no choices
  help:all || return 1
  while IFS=$'\t' read -r verb group purpose
  do
    printf '%s\t%s\t\t\n' "${verb}" "${purpose}"
  done < <(help:banners)
}

# ----------------------------------------------------------------------------------------------------------------------
