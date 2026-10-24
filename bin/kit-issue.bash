#!/usr/bin/env bash
# kit-issue - start or resume an issue: linked branch, worktree, start commit, draft pull request (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bash/gh.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"
use gh

if [[ "${1:-}" == --help ]]
then
  sed -n '2s/^# kit-[a-z]* - \(.*\) (kit verb, run through bin/kit)$/\1/p' "${BASH_SOURCE[0]}" >&2
  exit 0
fi

gh:issue "$@"

# ----------------------------------------------------------------------------------------------------------------------
