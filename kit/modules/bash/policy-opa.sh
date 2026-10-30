# shellcheck shell=bash
# policy-opa.sh - the policy provider that runs Open Policy Agent over the Rego in .kit/policy (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# modules/bash/policy.sh uses this provider when the policy_engine setting is opa, or when the setting is not set and
# .kit/policy/ contains a .rego file.
#
# opa loads every .rego file in .kit/policy/. Each string in the set data.kit.policy.deny is a refusal. The input
# is the facts document, so a rule reads input.branch, input.commits, input.change, and input.policy (the
# patterns, with the kit's defaults filled in).
#
# The repository's toolchain must pin opa, like any tool its gate runs. The policy/opa package delivers
# .kit/policy/kit.rego, the patterns provider's rules written in Rego, to build on.

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
