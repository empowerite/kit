# shellcheck shell=bash
# policy-patterns.sh - the default policy engine: each subject matched against its pattern in bash (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The engine bin/policy.sh uses unless the repository names another. It needs no tool: each rule of the facts'
# policy object is matched with bash's =~ against its subject, and a rule whose pattern is empty is off. The
# refusals are worded exactly as the policy-opa module's Rego words them, so either engine says the same thing.

# policy-patterns:deny <input-file> prints one refusal per subject that does not match its pattern.
policy-patterns:deny() {
  local input=$1 branch pattern subject sha title
  branch=$(jq --raw-output '.branch // ""' "${input}")
  pattern=$(jq --raw-output '.policy.branch' "${input}")
  if [[ -n "${branch}" && -n "${pattern}" && ! "${branch}" =~ ${pattern} ]]
  then
    printf "branch '%s' does not match policy.branch: %s\n" "${branch}" "${pattern}"
  fi
  pattern=$(jq --raw-output '.policy.commit_subject' "${input}")
  if [[ -n "${pattern}" ]]
  then
    while IFS=$'\x1f' read -r sha subject
    do
      [[ -n "${sha}" && ! "${subject}" =~ ${pattern} ]] || continue
      printf "commit %s '%s' does not match policy.commit_subject: %s\n" "${sha:0:7}" "${subject}" "${pattern}"
    done < <(jq --raw-output '.commits[] | "\(.sha)\u001f\(.subject)"' "${input}")
  fi
  title=$(jq --raw-output '.change.title // ""' "${input}")
  pattern=$(jq --raw-output '.policy.change_title' "${input}")
  [[ "$(jq --raw-output '.change != null' "${input}")" == true ]] || return 0
  if [[ -n "${pattern}" && ! "${title}" =~ ${pattern} ]]
  then
    printf "change title '%s' does not match policy.change_title: %s\n" "${title}" "${pattern}"
  fi
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
