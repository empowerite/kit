# shellcheck shell=bash
# tracker-forge.sh - the forge's own issues, GitHub's or GitLab's, as a tracker (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The provider bin/tracker.sh loads for `#`, unless a declared tracker claims it: every call is the forge seam's
# issue call of the same meaning, so the forge's provider answers and this file adds nothing of its own.

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
