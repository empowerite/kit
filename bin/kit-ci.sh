#!/usr/bin/env bash
# kit-ci - the delivered CI job's checks, run anywhere: the gate, the kit's own check, the issue link, and the policy
# ----------------------------------------------------------------------------------------------------------------------
#
# Every check runs even when an earlier one refused, so one run reports every finding. The exit status is the
# number of checks that refused.
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
