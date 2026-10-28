# shellcheck shell=bash
# tracker-forge.sh - the tracker provider for the forge's own issues, on GitHub or GitLab (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# bash/tracker.sh loads this provider for the prefix `#`, unless a record in `trackers` declares `#`. Each call
# except url passes straight to the matching issue call of the forge seam, bash/forge.sh, so the forge's
# provider does the work. url builds the issue's address from the repository's URL.

use forge

tracker-forge:title()     { forge:issue-title "$1"; }
tracker-forge:state()     { forge:issue-state "$1"; }
tracker-forge:assignees() { forge:issue-assignees "$1"; }
tracker-forge:assign()    { forge:issue-assign "$1" "$2"; }
tracker-forge:me()        { forge:me; }
tracker-forge:create()    { forge:issue-create "$1"; }

tracker-forge:url() {
  local repo_url
  repo_url=$(forge:repo-url) || return 1
  if [[ "$(forge:provider)" == gitlab ]]
  then printf '%s/-/issues/%s\n' "${repo_url}" "$1"
  else printf '%s/issues/%s\n' "${repo_url}" "$1"
  fi
}

# ----------------------------------------------------------------------------------------------------------------------
