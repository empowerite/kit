# shellcheck shell=bash
# opa.sh - the policy provider that runs Open Policy Agent over the Rego in .kit/policy (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# modules/bash/policy.sh uses this provider when the policy_engine setting is opa, or when the setting is not set and
# .kit/policy/ contains a .rego file.
#
# opa loads every .rego file in .kit/policy/, as the judged tree holds them (modules/bash/policy.sh). Each string in
# the set data.kit.policy.deny is a refusal, and each in data.kit.policy.warn is a warning, which is printed and never
# fails. Rego that defines no deny, as when a file names another package, refuses: unjudged is never allowed. Rego
# that defines no warn has no warnings. The input is the facts document, so a rule
# reads input.moment, input.branch, input.commits, input.change, input.tree (the files git holds), and input.policy (the
# patterns, with the kit's defaults filled in).
#
# The repository's toolchain must pin opa, like any tool its gate runs. The policy/opa package delivers
# .kit/policy/kit.rego, the patterns provider's rules written in Rego, to build on.

policy/opa:deny() {
  policy/opa:evaluate "$1" deny
}

policy/opa:warn() {
  policy/opa:evaluate "$1" warn
}

# The seam's calls end here; what follows is this provider's own.

# policy/opa:evaluate <input-file> <rule> prints each string in the set data.kit.policy.<rule>, one per line.
policy/opa:evaluate() {
  local input=$1 rule=$2 out
  if ! command -v opa >/dev/null 2>&1
  then
    error "pin opa in the repository's toolchain and run: kit tools install (the opa policy engine needs it)"
    return 1
  fi
  if ! out=$(opa eval --format json --data "${policy_rego:-.kit/policy}" --input "${input}" "data.kit.policy.${rule}" 2>&1)
  then
    error "opa could not evaluate .kit/policy: ${out}"
    return 1
  fi
  if [[ "${rule}" == deny ]] && ! jq --exit-status 'has("result")' <<<"${out}" >/dev/null
  then
    error "the Rego in .kit/policy defines no data.kit.policy.deny, so nothing can be judged; each file there starts" \
      "with: package kit.policy"
    return 1
  fi
  jq --raw-output '.result[0].expressions[0].value // [] | .[]' <<<"${out}"
}

# ----------------------------------------------------------------------------------------------------------------------
