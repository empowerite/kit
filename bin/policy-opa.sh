# shellcheck shell=bash
# policy-opa.sh - Open Policy Agent as the policy engine: the Rego under .kit/policy decides (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The engine bin/policy.sh uses for `policy_engine: opa`, or where .kit/policy holds a .rego file. Every .rego
# file there is loaded, and the set `data.kit.policy.deny` is the refusals, each a string. The facts document is
# the input, so a rule reads input.branch, input.commits, input.change, and input.policy, the patterns with the
# kit's defaults filled in. opa is the requirement, pinned by the repository's toolchain like any tool its gate
# runs. The policy-opa module delivers .kit/policy/kit.rego, the patterns engine's rules in Rego, to build on.

policy-opa:deny() {
  local input=$1 out
  if ! command -v opa >/dev/null 2>&1
  then
    error "pin opa in the repository's toolchain and run: kit toolchain install (the opa policy engine needs it)"
    return 1
  fi
  if ! out=$(opa eval --format json --data .kit/policy --input "${input}" 'data.kit.policy.deny' 2>&1)
  then
    error "opa could not evaluate .kit/policy: ${out}"
    return 1
  fi
  jq --raw-output '.result[0].expressions[0].value // [] | .[]' <<<"${out}"
}

# ----------------------------------------------------------------------------------------------------------------------
