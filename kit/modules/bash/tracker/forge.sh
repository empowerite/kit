# shellcheck shell=bash
# forge.sh - the tracker provider for the forge's own issues, on GitHub or GitLab (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# modules/bash/tracker.sh loads this provider for the prefix `#`, unless a record in `trackers` declares `#`. Each call
# except url and new-url passes straight to the matching issue call of the forge seam, modules/bash/forge.sh, so the
# forge's provider does the work. url and new-url build their addresses from the repository's URL: new-url the page a
# person files a new issue at, the title in its query, as title on GitHub and issue[title] on GitLab.

tracker/forge:assign()    { forge:issue-assign "$1" "$2"; }

tracker/forge:assignees() { forge:issue-assignees "$1"; }

tracker/forge:close()     { forge:issue-close "$1" "$2" "${3:-}"; }

tracker/forge:comment()   { forge:issue-comment "$1" "$2"; }

tracker/forge:create()    { forge:issue-create "$1" "${2:-}"; }

tracker/forge:me()        { forge:me; }

tracker/forge:new-url() {
  local repo_url
  repo_url=$(forge:repo-url) || return 1
  if [[ "$(forge:provider)" == gitlab ]]
  then tracker:query "${repo_url}/-/issues/new" 'issue[title]' "${1:-}"
  else tracker:query "${repo_url}/issues/new" title "${1:-}"
  fi
}

tracker/forge:state()     { forge:issue-state "$1"; }

tracker/forge:title()     { forge:issue-title "$1"; }

tracker/forge:unassign()  { forge:issue-unassign "$1" "$2"; }

tracker/forge:url() {
  local repo_url
  repo_url=$(forge:repo-url) || return 1
  if [[ "$(forge:provider)" == gitlab ]]
  then printf '%s/-/issues/%s\n' "${repo_url}" "$1"
  else printf '%s/issues/%s\n' "${repo_url}" "$1"
  fi
}

# The seam's calls end here; what follows is this provider's own.

use forge

# ----------------------------------------------------------------------------------------------------------------------
