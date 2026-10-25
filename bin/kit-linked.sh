#!/usr/bin/env bash
# kit-linked - refuse a pull request not linked to an open issue of this repository (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The delivered CI job's last step, and a check a desk can run on any pull request. The pull request is the
# argument, else the one the GitHub event names on a runner, else the one for the current branch.
#
# see: bin/gh.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use gh

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit linked [<pull-request>]" >&2
  exit 0
fi

number=${1:-}
if [[ -z "${number}" && -n "${GITHUB_EVENT_PATH:-}" ]]
then
  number=$(jq --raw-output '.pull_request.number // empty' "${GITHUB_EVENT_PATH}")
fi
if [[ -z "${number}" ]]
then
  number=$(gh pr view --json number --jq .number 2>/dev/null || true)
fi
if [[ ! "${number}" =~ ^[0-9]+$ ]]
then
  error "name the pull request: kit linked <number> (none was given, and the current branch has none)"
  exit 2
fi
gh:linked-open-issue "${number}"

# ----------------------------------------------------------------------------------------------------------------------
