# shellcheck shell=bash
# patterns.sh - the default policy provider: matches each subject against its pattern in bash (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# modules/bash/policy.sh uses this provider unless the repository picks another. It needs no tool. It matches each rule in
# the facts' policy object against its subject with bash's =~. A rule with an empty pattern is off. The refusals
# use the same wording as the policy/opa package's Rego, so both providers print the same messages: the subject, the
# pattern, and the rule's remedy in parentheses where policy.remedy has one.

# policy/patterns:deny <input-file> prints one refusal per subject that does not match its pattern.
policy/patterns:deny() {
  local input=$1 name pattern sha subject
  while IFS=$'\x1f' read -r name pattern
  do
    [[ -n "${name}" && -n "${pattern}" && ! "${name}" =~ ${pattern} ]] || continue
    policy/patterns:refusal "${input}" branch "branch '${name}'" "${pattern}"
  done < <(jq --raw-output '[.branch // "", .policy.branch // ""] | join("\u001f")' "${input}")
  while IFS=$'\x1f' read -r name pattern
  do
    [[ -n "${name}" && -n "${pattern}" && ! "${name}" =~ ${pattern} ]] || continue
    policy/patterns:refusal "${input}" worktree "worktree '${name}'" "${pattern}"
  done < <(jq --raw-output '[.worktree // "", .policy.worktree // ""] | join("\u001f")' "${input}")
  pattern=$(jq --raw-output '.policy.commit_subject // ""' "${input}")
  if [[ -n "${pattern}" ]]
  then
    while IFS=$'\x1f' read -r sha subject
    do
      [[ -n "${sha}" && ! "${subject}" =~ ${pattern} ]] || continue
      policy/patterns:refusal "${input}" commit_subject "commit ${sha:0:7} '${subject}'" "${pattern}"
    done < <(jq --raw-output '.commits // [] | .[] | "\(.sha)\u001f\(.subject)"' "${input}")
  fi
  [[ "$(jq --raw-output '.change != null' "${input}")" == true ]] || return 0
  name=$(jq --raw-output '.change.title // ""' "${input}")
  pattern=$(jq --raw-output '.policy.change_title // ""' "${input}")
  if [[ -n "${pattern}" && ! "${name}" =~ ${pattern} ]]
  then
    policy/patterns:refusal "${input}" change_title "change title '${name}'" "${pattern}"
  fi
  return 0
}

# The seam's calls end here; what follows is this provider's own.

# policy/patterns:refusal <input-file> <rule> <what> <pattern> prints the refusal of <what> by the rule: what does not
# match policy.<rule>, the pattern, and the rule's remedy in the facts in parentheses, where it has one.
policy/patterns:refusal() {
  local remedy
  remedy=$(jq --raw-output --arg rule "$2" '.policy.remedy[$rule]? // ""' "$1")
  printf '%s does not match policy.%s: %s%s\n' "$3" "$2" "$4" "${remedy:+ (${remedy})}"
}

# ----------------------------------------------------------------------------------------------------------------------
