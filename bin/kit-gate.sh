#!/usr/bin/env bash
# kit-gate - run the repository's gate, the root justfile's ci recipe, and refuse when there is none (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The delivered CI job's first step. A gate that cannot run clears nothing: a missing justfile, a justfile with no
# ci recipe, and a justfile that cannot be read are each refused, naming the remedy. On a runner, each outcome
# also lands in the step summary.
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
  error "add a root justfile with a ci recipe; there is none, so the gate cannot run"
  summary "### ❌ The gate cannot run: there is no root justfile" "" "Add a root justfile with a \`ci\` recipe."
  exit 1
fi
if ! recipes=$(just --summary 2>&1)
then
  error "the root justfile could not be read: ${recipes}"
  summary "### ❌ The root justfile could not be read" "" "${recipes}"
  exit 1
fi
if ! tr ' ' '\n' <<<"${recipes}" | grep --quiet --line-regexp ci
then
  error "add a ci recipe to the root justfile; it has none, so the gate cannot run"
  summary "### ❌ The gate cannot run: the root justfile has no ci recipe" "" "Add a \`ci\` recipe to it."
  exit 1
fi
just ci
summary "✅ just ci passed."

# ----------------------------------------------------------------------------------------------------------------------
