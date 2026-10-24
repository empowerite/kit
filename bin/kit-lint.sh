#!/usr/bin/env bash
# kit-lint - run the pre-commit hook's linters by hand (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# An alias for `lefthook run pre-commit`, so the linters a commit meets are the ones this runs, over the staged
# paths, or over every path with --all-files. The arguments are Lefthook's.

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit lint [--all-files] [<lefthook run option>…]" >&2
  exit 0
fi

guard:require tool:lefthook
exec lefthook run pre-commit "$@"

# ----------------------------------------------------------------------------------------------------------------------
