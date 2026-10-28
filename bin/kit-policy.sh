#!/usr/bin/env bash
# kit-policy - refuse a branch, a commit subject, or a change title the repository's policy does not allow (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The facts are gathered once, the branch, the commits past the fork, and the open change, and the policy engine,
# patterns by default or Open Policy Agent, names every refusal. --input prints the facts instead, the document a
# rule is written against. kit ci runs this beside the gate. On a runner the refusals land in the step summary.
#
# see: bin/policy.sh, docs/policy.md

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use policy

case "${1:-}" in
  (--help)
    echo "usage: kit policy [--input]" >&2
    exit 0
    ;;
  (--input)
    policy:input
    exit 0
    ;;
  ('') ;;
  (*)
    echo "usage: kit policy [--input]" >&2
    exit 2
esac

input=$(mktemp)
trap 'rm -f "${input}"' EXIT
policy:input > "${input}"
engine=$(policy:engine)
refusals=$(policy:deny "${input}") || exit 1
if [[ -n "${refusals}" ]]
then
  while IFS= read -r refusal
  do
    error "${refusal}"
  done <<<"${refusals}"
  count=$(wc -l <<<"${refusals}" | tr -d ' ')
  error "${count} refusal(s) by the ${engine} policy; .kit/settings.yml holds the patterns, docs/policy.md says how"
  summary "### ❌ ${count} refusal(s) by the policy" "" "- ${refusals//$'\n'/$'\n'- }"
  exit 1
fi
subjects=$(jq --raw-output '[if .branch then "the branch" else empty end,
  "\(.commits | length) commit(s)", if .change then "the change title" else empty end] | join(", ")' "${input}")
ok "the ${engine} policy allows ${subjects}"
summary "✅ The policy allows ${subjects}."

# ----------------------------------------------------------------------------------------------------------------------
