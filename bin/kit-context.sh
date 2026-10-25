#!/usr/bin/env bash
# kit-context - name the rules that govern a file, and record the ask (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/context.sh, docs/context.md

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use context

case "${1:-}" in
  (--help)
    echo "usage: kit context <path>… [--json] | kit context --claude" >&2
    exit 0
    ;;
  (--claude)
    context:claude
    ;;
  (*)
    context:render "$@"
esac

# ----------------------------------------------------------------------------------------------------------------------
