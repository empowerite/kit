# shellcheck shell=bash
# tracker.sh - the loop's seam over the trackers that hold its issues (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# An issue lives in a tracker, and the tracker is not the forge: GitHub issues and GitLab issues happen to live in
# theirs, Azure DevOps work items and Jira do not, and a repository may use more than one at once. A reference
# names its tracker in its syntax, as the trackers themselves write it: `#7` or `7` for the forge's own, `AB#4521`
# for an Azure DevOps work item. `.kit/settings.yml` declares the others, a prefix and a provider each:
#
#   trackers:
#     - prefix: "AB#"
#       provider: ado
#       organization: https://dev.azure.com/example
#       project: Claims
#
# `#` is the forge's own tracker unless a record claims it. A provider is tracker-<name>.sh beside this file, its
# functions named tracker-<name>:<call>, each taking the reference's number first and the prefix's record through
# tracker:field. A provider answers these calls, printing its result on stdout and failing with a remedy:
#
#   title <id>              the issue's title
#   state <id>              OPEN or CLOSED
#   assignees <id>          the assignees' logins, space-separated
#   assign <id> <login>     adds the assignee
#   me                      the caller's login in this tracker, which need not be the forge's
#   create <title>          the number of the issue it creates
#   url <id>                the issue's browser URL
#
# see: docs/trackers.md

use release yaml

# tracker:records prints "<prefix>\x1f<provider>" for every declared tracker, then the forge's own as "#" unless a
# record claims it.
tracker:records() {
  local prefix provider forge_claimed=false
  while IFS=$'\x1f' read -r prefix provider
  do
    [[ -n "${prefix}" ]] || continue
    [[ "${prefix}" == '#' ]] && forge_claimed=true
    printf '%s\x1f%s\n' "${prefix}" "${provider}"
  done < <(yaml:records "$(release:settings)" trackers prefix provider)
  [[ "${forge_claimed}" == true ]] || printf '#\x1fforge\n'
}

# tracker:field <prefix> <field> prints a field of the prefix's record, or nothing. Every loop over records here
# reads to the end, so the reader is never left writing into a pipe nobody reads.
tracker:field() {
  local prefix value found=''
  while IFS=$'\x1f' read -r prefix value
  do
    [[ -z "${found}" && "${prefix}" == "$1" ]] && found=${value}
  done < <(yaml:records "$(release:settings)" trackers prefix "$2")
  [[ -z "${found}" ]] || printf '%s\n' "${found}"
}

# tracker:parse <reference> prints "<prefix>\t<number>": the longest declared prefix the reference opens with, and
# the digits after it; a bare number is the forge's own. A reference no declared tracker answers is refused, naming
# the prefixes there are.
tracker:parse() {
  local ref=$1 prefix best='' provider
  [[ "${ref}" =~ ^[0-9]+$ ]] && ref="#${ref}"
  while IFS=$'\x1f' read -r prefix provider
  do
    if [[ "${ref}" == "${prefix}"* && "${ref#"${prefix}"}" =~ ^[0-9]+$ && ${#prefix} -gt ${#best} ]]
    then
      best=${prefix}
    fi
  done < <(tracker:records)
  if [[ -z "${best}" ]]
  then
    error "'$1' is not an issue reference here; the trackers answer $(tracker:records | cut -d $'\x1f' -f 1 \
      | sed 's/$/<number>/' | paste -s -d ' ' -) (trackers in .kit/settings.yml)"
    return 2
  fi
  printf '%s\t%s\n' "${best}" "${ref#"${best}"}"
}

# tracker:provider <prefix> prints the provider that answers the prefix.
tracker:provider() {
  local prefix provider found=''
  while IFS=$'\x1f' read -r prefix provider
  do
    [[ -z "${found}" && "${prefix}" == "$1" ]] && found=${provider}
  done < <(tracker:records)
  if [[ -z "${found}" ]]
  then
    error "no tracker answers '$1'"
    return 1
  fi
  printf '%s\n' "${found}"
}

# tracker:of <reference> <call> <argument>… runs the call of the tracker the reference names, with its number.
tracker:of() {
  local ref=$1 call=$2 prefix number provider
  shift 2
  IFS=$'\t' read -r prefix number < <(tracker:parse "${ref}") || return 2
  [[ -n "${prefix}" ]] || return 2
  provider=$(tracker:provider "${prefix}") || return 1
  use "tracker-${provider}" || return 1
  TRACKER_PREFIX=${prefix} "tracker-${provider}:${call}" "${number}" "$@"
}

# tracker:at <prefix> <call> <argument>… runs a call that names no issue, such as me or create, on the prefix's
# tracker.
tracker:at() {
  local prefix=$1 call=$2 provider
  shift 2
  provider=$(tracker:provider "${prefix}") || return 1
  use "tracker-${provider}" || return 1
  TRACKER_PREFIX=${prefix} "tracker-${provider}:${call}" "$@"
}

# tracker:normal <reference> prints the reference as its tracker writes it: `7` becomes `#7`.
tracker:normal() {
  local prefix number
  IFS=$'\t' read -r prefix number < <(tracker:parse "$1") || return 2
  [[ -n "${prefix}" ]] || return 2
  printf '%s%s\n' "${prefix}" "${number}"
}

# tracker:is-forge <reference> is true when the forge's own tracker answers the reference, so the forge can link a
# branch to it and close it on merge by itself.
tracker:is-forge() {
  local prefix number
  IFS=$'\t' read -r prefix number < <(tracker:parse "$1" 2>/dev/null) || return 1
  [[ -n "${prefix}" && "$(tracker:provider "${prefix}")" == forge ]]
}

# tracker:key <reference> prints the reference as a branch name opens with it: the forge's own issue 7 as `7`, a
# work item AB#4521 as `ab4521`, letters lowered and anything not a letter or a digit dropped.
tracker:key() {
  local ref
  ref=$(tracker:normal "$1") || return 2
  if tracker:is-forge "${ref}"
  then
    printf '%s\n' "${ref#\#}"
  else
    tr '[:upper:]' '[:lower:]' <<<"${ref}" | tr -cd 'a-z0-9'
    printf '\n'
  fi
}

# tracker:branch-ref <branch> prints the reference a branch was made for, read back from its key, or nothing.
tracker:branch-ref() {
  local branch=$1 prefix provider key found=''
  while IFS=$'\x1f' read -r prefix provider
  do
    [[ -z "${found}" ]] || continue
    if [[ "${provider}" == forge ]]
    then
      [[ "${branch}" =~ ^([0-9]+)- ]] && found="${prefix}${BASH_REMATCH[1]}"
    else
      key=$(tr '[:upper:]' '[:lower:]' <<<"${prefix}" | tr -cd 'a-z0-9')
      [[ "${branch}" =~ ^${key}([0-9]+)- ]] && found="${prefix}${BASH_REMATCH[1]}"
    fi
  done < <(tracker:records)
  [[ -z "${found}" ]] || printf '%s\n' "${found}"
}

# tracker:body-refs <body> prints every reference a closing keyword names in a change's body, for the trackers the
# forge does not link by itself: `Closes AB#4521`, `fixes AB#4521`, `Resolves: AB#4521`.
tracker:body-refs() {
  local body=$1 prefix provider escaped
  while IFS=$'\x1f' read -r prefix provider
  do
    [[ "${provider}" == forge ]] && continue
    escaped=$(sed 's/[][\.*^$+?(){}|/]/\\&/g' <<<"${prefix}")
    grep --only-matching --ignore-case --extended-regexp \
      "(close[sd]?|fix(e[sd])?|resolve[sd]?):? ${escaped}[0-9]+" <<<"${body}" \
      | grep --only-matching --extended-regexp "[0-9]+\$" | sed "s/^/${prefix//\//\\/}/" || true
  done < <(tracker:records)
}

# ----------------------------------------------------------------------------------------------------------------------
