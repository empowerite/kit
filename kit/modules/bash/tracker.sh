# shellcheck shell=bash
# tracker.sh - the seam over the issue trackers: forge issues, Azure DevOps work items (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# Verbs that read, assign, or create issues call tracker:of or tracker:at here. Each call is passed to the
# provider for the issue's tracker: tracker/<name>.sh beside it, whose functions are named
# tracker/<name>:<call>.
#
# The tracker is not always the forge (GitHub or GitLab): GitHub and GitLab issues are on the forge; Azure DevOps
# work items are not. A repository can use more than one tracker at once.
#
# An issue reference names its tracker with a prefix, written the way that tracker writes it:
#
#   - `#7` or `7`: an issue in the forge's own tracker
#   - `AB#4521`: an Azure DevOps work item; on an Azure DevOps forge, the forge's own, as `#4521` is
#
# The provider for a reference is chosen by:
#
#   1. a bare number is read as `#<number>`
#   2. the longest prefix that the reference starts with, followed only by digits, picks a record: one in the
#      `trackers` list in kit.yml, or the built-in `#` record for the forge's own tracker
#   3. that record's `provider` field names the provider
#
# A record in `trackers` declares a prefix, a provider, and the provider's own fields. A record with the prefix
# `#` replaces the built-in one.
#
#   trackers:
#     - prefix: "AB#"
#       provider: ado
#       organization: https://dev.azure.com/example
#       project: Claims
#
# Every provider implements the calls below. Each prints its result on stdout. On failure it prints a remedy. A
# call about one issue gets the reference's number, without the prefix, as <id>. TRACKER_PREFIX is set to the
# prefix, and the provider reads its record's fields with tracker:field.
#
#   title <id>              the issue's title
#   state <id>              OPEN or CLOSED
#   assignees <id>          the assignees' logins, space-separated
#   assign <id> <login>     adds the assignee
#   unassign <id> <login>   removes the assignee
#   close <id> <reason> [<other-id>]  closes the issue: completed, not-planned, or duplicate of the other issue in
#                           the same tracker
#   comment <id> <text>     adds a comment
#   me                      the caller's login in this tracker, which can differ from the forge login
#   new-url [<title>]       the browser URL a person files a new issue at, the title filled in where the page takes one
#   create <title> [<body>] the number of the issue it creates
#   url <id>                the issue's browser URL
#
# see: docs/trackers.md

use yaml settings forge

# tracker:records prints "<prefix>\x1f<provider>" for every declared tracker, then "#\x1fforge" for the forge's
# own tracker, unless a record declares the prefix `#`. On an Azure DevOps forge, whose own tracker is Azure Boards,
# "AB#\x1fforge" follows too, unless a record declares `AB#`: a work item is #7 and AB#7 alike there.
tracker:records() {
  local prefix provider forge_claimed=false boards_claimed=false row
  local -a rows
  list:read rows yaml:records "$(settings:file)" trackers prefix provider || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r prefix provider <<<"${row}"
    [[ -n "${prefix}" ]] || continue
    [[ "${prefix}" == '#' ]] && forge_claimed=true
    [[ "${prefix}" == 'AB#' ]] && boards_claimed=true
    printf '%s\x1f%s\n' "${prefix}" "${provider}"
  done
  [[ "${forge_claimed}" == true ]] || printf '#\x1fforge\n'
  if [[ "${boards_claimed}" != true ]] && [[ "$(forge:of "$(forge:primary 2>/dev/null)" 2>/dev/null)" == ado ]]
  then
    printf 'AB#\x1fforge\n'
  fi
}

# tracker:field <prefix> <field> prints a field of the prefix's record, or nothing.
tracker:field() {
  local prefix value found='' row
  local -a rows
  list:read rows yaml:records "$(settings:file)" trackers prefix "$2" || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r prefix value <<<"${row}"
    [[ -z "${found}" && "${prefix}" == "$1" ]] && found=${value}
  done
  [[ -z "${found}" ]] || printf '%s\n' "${found}"
}

# tracker:parse <reference> prints "<prefix>\t<number>": the longest declared prefix that the reference starts
# with, and the digits after it. A bare number is in the forge's own tracker. When no tracker matches the
# reference, it fails and lists the prefixes.
tracker:parse() {
  local ref=$1 prefix best='' provider row
  local -a rows
  [[ "${ref}" =~ ^[0-9]+$ ]] && ref="#${ref}"
  list:read rows tracker:records || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r prefix provider <<<"${row}"
    if [[ "${ref}" == "${prefix}"* && "${ref#"${prefix}"}" =~ ^[0-9]+$ && ${#prefix} -gt ${#best} ]]
    then
      best=${prefix}
    fi
  done
  if [[ -z "${best}" ]]
  then
    error "'$1' is not an issue reference here; the trackers answer $(printf '%s\n' "${rows[@]}" \
      | cut -d $'\x1f' -f 1 | sed 's/$/<number>/' | paste -s -d ' ' -) (trackers in kit.yml)"
    return 2
  fi
  printf '%s\t%s\n' "${best}" "${ref#"${best}"}"
}

# tracker:provider <prefix> prints the provider for the prefix.
tracker:provider() {
  local prefix provider found='' row
  local -a rows
  list:read rows tracker:records || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r prefix provider <<<"${row}"
    [[ -z "${found}" && "${prefix}" == "$1" ]] && found=${provider}
  done
  if [[ -z "${found}" ]]
  then
    error "no tracker answers '$1'"
    return 1
  fi
  printf '%s\n' "${found}"
}

# tracker:of <reference> <call> <argument>… runs the call on the reference's tracker, passing the reference's
# number first.
tracker:of() {
  local ref=$1 call=$2 prefix number provider
  shift 2
  IFS=$'\t' read -r prefix number < <(tracker:parse "${ref}") || return 2
  [[ -n "${prefix}" ]] || return 2
  provider=$(tracker:provider "${prefix}") || return 1
  use "tracker/${provider}" || return 1
  TRACKER_PREFIX=${prefix} "tracker/${provider}:${call}" "${number}" "$@"
}

# tracker:at <prefix> <call> <argument>… runs a call that names no issue, such as me or create, on the prefix's
# tracker.
tracker:at() {
  local prefix=$1 call=$2 provider
  shift 2
  provider=$(tracker:provider "${prefix}") || return 1
  use "tracker/${provider}" || return 1
  TRACKER_PREFIX=${prefix} "tracker/${provider}:${call}" "$@"
}

# tracker:query <url> <key> <value> prints the URL with ?<key>=<value> after it, each encoded for a URL, or the URL
# alone when the value is empty: the query a tracker's new-issue page reads a field from.
tracker:query() {
  if [[ -z "$3" ]]
  then
    printf '%s\n' "$1"
    return
  fi
  jq --null-input --raw-output --arg url "$1" --arg key "$2" --arg value "$3" '"\($url)?\($key | @uri)=\($value | @uri)"'
}

# tracker:normal <reference> prints the reference in its tracker's own form: `7` becomes `#7`.
tracker:normal() {
  local prefix number
  IFS=$'\t' read -r prefix number < <(tracker:parse "$1") || return 2
  [[ -n "${prefix}" ]] || return 2
  printf '%s%s\n' "${prefix}" "${number}"
}

# tracker:is-forge <reference> succeeds when the reference is in the forge's own tracker. The forge can then link
# a branch to the issue and close the issue on merge by itself.
tracker:is-forge() {
  local prefix number
  IFS=$'\t' read -r prefix number < <(tracker:parse "$1" 2>/dev/null) || return 1
  [[ -n "${prefix}" && "$(tracker:provider "${prefix}")" == forge ]]
}

# tracker:key <reference> prints the reference in the form a branch name starts with: forge issue 7 as `7`, work
# item AB#4521 as `ab4521` (lowercased, with everything but letters and digits removed).
tracker:key() {
  local ref
  ref=$(tracker:normal "$1") || return 2
  # The forge's own issue is its number, whatever prefix names it: #7, or AB#7 on Azure DevOps.
  if tracker:is-forge "${ref}"
  then
    printf '%s\n' "${ref##*#}"
  else
    tr '[:upper:]' '[:lower:]' <<<"${ref}" | tr -cd 'a-z0-9'
    printf '\n'
  fi
}

# tracker:branch-ref <branch> prints the reference a branch was made for, read back from the key the branch name
# starts with, or nothing.
tracker:branch-ref() {
  local branch=$1 prefix provider key found='' row
  local -a rows
  list:read rows tracker:records || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r prefix provider <<<"${row}"
    [[ -z "${found}" ]] || continue
    if [[ "${provider}" == forge ]]
    then
      [[ "${branch}" =~ ^([0-9]+)- ]] && found="${prefix}${BASH_REMATCH[1]}"
    else
      key=$(tr '[:upper:]' '[:lower:]' <<<"${prefix}" | tr -cd 'a-z0-9')
      [[ "${branch}" =~ ^${key}([0-9]+)- ]] && found="${prefix}${BASH_REMATCH[1]}"
    fi
  done
  [[ -z "${found}" ]] || printf '%s\n' "${found}"
}

# tracker:body-refs <body> prints every reference that follows a closing keyword in a change's body, for the
# trackers the forge does not link by itself: `Closes AB#4521`, `fixes AB#4521`, `Resolves: AB#4521`.
tracker:body-refs() {
  local body=$1 prefix provider escaped row
  local -a rows
  list:read rows tracker:records || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r prefix provider <<<"${row}"
    [[ "${provider}" == forge ]] && continue
    escaped=$(ere:quote "${prefix}")
    # grep's exit 1 is a body that names no reference to this tracker.
    grep --only-matching --ignore-case --extended-regexp \
      "(close[sd]?|fix(e[sd])?|resolve[sd]?):? ${escaped}[0-9]+" <<<"${body}" \
      | grep --only-matching --extended-regexp "[0-9]+\$" | sed "s/^/${prefix//\//\\/}/" || true
  done
}

# ----------------------------------------------------------------------------------------------------------------------
