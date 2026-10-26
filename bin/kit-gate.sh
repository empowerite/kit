#!/usr/bin/env bash
# kit-gate - run the repository's gate, and refuse when there is none (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The gate is the `gate` setting in .kit/settings.yml, else the first of: a root justfile with a ci recipe, a
# Makefile with a ci target, a package.json with a ci script. A gate that cannot run clears nothing: a repository
# with none, or one whose justfile cannot be read, is refused, naming the remedy. On a runner, each outcome also
# lands in the step summary.
#
# see: modules/ci-github/templates/repo/.github/workflows/kit-ci.yml

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit gate" >&2
  exit 0
fi

# gate:detect prints the command of the first gate found, or nothing.
detect-gate() {
  local recipes
  if [[ -f justfile || -f Justfile || -f .justfile ]]
  then
    if ! recipes=$(just --summary 2>&1)
    then
      error "the root justfile could not be read: ${recipes}"
      summary "### ❌ The root justfile could not be read" "" "${recipes}"
      return 1
    fi
    tr ' ' '\n' <<<"${recipes}" | grep --quiet --line-regexp ci && printf 'just ci\n' && return 0
  fi
  if [[ -f Makefile ]] && grep --quiet '^ci:' Makefile
  then
    printf 'make ci\n'
    return 0
  fi
  if [[ -f package.json ]] && jq --exit-status '.scripts.ci' package.json >/dev/null 2>&1
  then
    printf 'npm run ci\n'
    return 0
  fi
  return 0
}

gate=$(release:setting gate)
[[ -n "${gate}" ]] || gate=$(detect-gate) || exit 1
if [[ -z "${gate}" ]]
then
  error "set gate in .kit/settings.yml, or give the root justfile a ci recipe; there is no gate to run"
  # shellcheck disable=SC2016  # markdown code spans, not expansions
  summary "### ❌ The gate cannot run: the repository has none" "" \
    'Set `gate` in `.kit/settings.yml`, or give the root justfile a `ci` recipe.'
  exit 1
fi
step "the gate: ${gate}"
if ! bash -c "${gate}"
then
  error "the gate failed: ${gate}"
  summary "### ❌ The gate failed: \`${gate}\`"
  exit 1
fi
summary "✅ The gate passed: \`${gate}\`."

# ----------------------------------------------------------------------------------------------------------------------
