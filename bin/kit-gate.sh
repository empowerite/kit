#!/usr/bin/env bash
# kit-gate - run the repository's gate, the root justfile's ci recipe, and say plainly when there is none (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The delivered CI job's first step. A missing justfile or a justfile with no ci recipe is a skip, reported as a
# skip, never as a pass; a justfile that cannot be read is a failure. On a runner, each outcome also lands in the
# step summary.
#
# see: modules/ci/templates/repo/.github/workflows/kit-ci.yml

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit gate" >&2
  exit 0
fi

if [[ ! -f justfile ]]
then
  warn "there is no root justfile, so the gate was skipped"
  summary "⚠️ The gate was skipped: there is no root justfile."
  exit 0
fi
if ! recipes=$(just --summary 2>&1)
then
  error "the root justfile could not be read: ${recipes}"
  summary "### ❌ The root justfile could not be read" "" "${recipes}"
  exit 1
fi
if ! tr ' ' '\n' <<<"${recipes}" | grep --quiet --line-regexp ci
then
  warn "the root justfile has no ci recipe, so the gate was skipped"
  summary "⚠️ The gate was skipped: the root justfile has no ci recipe."
  exit 0
fi
just ci
summary "✅ just ci passed."

# ----------------------------------------------------------------------------------------------------------------------
