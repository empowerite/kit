#!/usr/bin/env bash
# kit-ci - run the delivered CI job's four checks on any machine (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit ci takes no arguments. It runs these checks in order:
#
#   1. kit gate: the repository's gate (its own CI command) passes
#   2. kit check: every delivered file matches the pinned release
#   3. kit linked: the pull request is linked to an open issue
#   4. kit policy: the branch, commit subjects, and pull request title match the policy
#
# Every check runs even when an earlier one fails, so one run reports every finding.
#
# Exits 0 when all four pass. Otherwise the exit status is the number of checks that failed.
#
# see: bin/kit-gate.sh, bin/kit-check.sh, bin/kit-linked.sh, bin/kit-policy.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit ci" >&2
  exit 0
fi

refused=0
for verb in gate check linked policy
do
  section "kit ${verb}"
  "${kit_root}/bin/kit-${verb}.sh" || refused=$(( refused + 1 ))
done
if (( refused > 0 ))
then
  error "${refused} of 4 checks refused; see above"
  exit "${refused}"
fi
ok "kit ci: the gate, the kit's check, the issue link, and the policy all passed."

# ----------------------------------------------------------------------------------------------------------------------
