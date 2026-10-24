#!/usr/bin/env bash
# kit-version - print this release's version (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release
release:version

# ----------------------------------------------------------------------------------------------------------------------
