#!/usr/bin/env bash
# kit-version - print this release's version (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------

set -o errexit -o nounset -o pipefail
cat "${kit_root:?run through bin/kit}/version"

# ----------------------------------------------------------------------------------------------------------------------
