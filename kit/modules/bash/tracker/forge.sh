# shellcheck shell=bash
# forge.sh - the tracker provider for the forge's own issues (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# modules/bash/tracker.sh loads this provider for the prefix `#`, unless a record in `trackers` declares `#`. Each call
# except url and new-url passes straight to the matching issue call of the forge seam, modules/bash/forge.sh, so the
# forge's provider does the work. url and new-url build their addresses from the repository's URL: new-url the page a
# person files a new issue at, the title in its query, as title on GitHub and issue[title] on GitLab. A provider whose
# issues live elsewhere, as Azure DevOps' work items do, gives both itself: forge/<name>:issue-url <id> and
# forge/<name>:issue-new-url [<title>].

tracker/forge:assign()    { tracker/forge:own && forge:issue-assign "$1" "$2"; }

tracker/forge:assignees() { tracker/forge:own && forge:issue-assignees "$1"; }

tracker/forge:close()     { tracker/forge:own && forge:issue-close "$1" "$2" "${3:-}"; }

tracker/forge:comment()   { tracker/forge:own && forge:issue-comment "$1" "$2"; }

tracker/forge:create()    { tracker/forge:own && forge:issue-create "$1" "${2:-}"; }

tracker/forge:me()        { forge:me; }

tracker/forge:new-url() {
  local repo_url provider
  provider=$(forge:provider) || return 1
  use "forge/${provider}"
  if declare -F "forge/${provider}:issue-new-url" >/dev/null
  then
    "forge/${provider}:issue-new-url" "${1:-}"
    return
  fi
  repo_url=$(forge:repo-url) || return 1
  if [[ "${provider}" == gitlab ]]
  then tracker:query "${repo_url}/-/issues/new" 'issue[title]' "${1:-}"
  else tracker:query "${repo_url}/issues/new" title "${1:-}"
  fi
}

tracker/forge:state()     { tracker/forge:own && forge:issue-state "$1"; }

tracker/forge:title()     { tracker/forge:own && forge:issue-title "$1"; }

tracker/forge:unassign()  { tracker/forge:own && forge:issue-unassign "$1" "$2"; }

tracker/forge:url() {
  local repo_url provider
  provider=$(forge:provider) || return 1
  use "forge/${provider}"
  if declare -F "forge/${provider}:issue-url" >/dev/null
  then
    "forge/${provider}:issue-url" "$1"
    return
  fi
  repo_url=$(forge:repo-url) || return 1
  if [[ "${provider}" == gitlab ]]
  then printf '%s/-/issues/%s\n' "${repo_url}" "$1"
  else printf '%s/issues/%s\n' "${repo_url}" "$1"
  fi
}

# The seam's calls end here; what follows is this provider's own.

# tracker/forge:own succeeds when the forge has issues of its own. A forge without them, such as AWS CodeCommit, is
# refused with the remedy: another tracker claims the prefix #.
tracker/forge:own() {
  forge:lacks >/dev/null || return 1
  forge:has issues && return 0
  error "the $(forge:provider) forge has no issues of its own; claim \"#\" for another tracker under trackers in kit.yml"
  return 1
}

use forge

# ----------------------------------------------------------------------------------------------------------------------
