# shellcheck shell=bash
# policy-patterns.sh - the default policy provider: matches each subject against its pattern in bash (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# bash/policy.sh uses this provider unless the repository picks another. It needs no tool. It matches each rule in
# the facts' policy object against its subject with bash's =~. A rule with an empty pattern is off. The refusals
# use the same wording as the policy-opa package's Rego, so both providers print the same messages.

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
