#!/usr/bin/env bash
# kit-version - print this release's version (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit version takes no arguments. It prints the version of the kit that runs, or unreleased for a checkout of the
# kit that has no version file.

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit version" >&2
  exit 0
fi

release:version

# ----------------------------------------------------------------------------------------------------------------------
