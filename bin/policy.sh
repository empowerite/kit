# shellcheck shell=bash
# policy.sh - the seam over what the loop's names must be: the facts, the rules, and the engine (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# What a branch, a commit subject, or a change's title may be is a policy, and a policy is checked, not described.
# The seam has three parts:
#
#   the facts     policy:input gathers one JSON document: the branch, the default branch, the commits past the
#                 fork, the change on the branch with its title and body, and the policy settings with the kit's
#                 defaults filled in. `kit policy --input` prints it, so a rule can be written against it.
#   the rules     `policy:` in .kit/settings.yml, one extended regular expression per subject:
#                   branch            the branch's name
#                   commit_subject    every commit's subject past the fork, merges aside
#                   change_title      the open change's title
#                 A subject absent takes the kit's default below; an empty pattern turns that rule off.
#   the engine    policy-<name>.sh beside this file answers `deny <input-file>`, one refusal per line:
#                   patterns          the default: each pattern matched in bash, no tool needed
#                   opa               Open Policy Agent over the Rego under .kit/policy/, package kit.policy, rule
#                                     deny; chosen by `policy_engine: opa`, or by a .rego file there
#                 Both read the same document, so a rule a pattern cannot say is written in Rego beside the
#                 patterns, and the policy-opa module delivers the patterns themselves as Rego to start from.
#
# A pattern is matched as an extended regular expression by bash and as RE2 by OPA; the two agree on the common
# subset, character classes, anchors, alternation, and counted repetition, which is the subset to write in.
#
# see: docs/policy.md

use release yaml git

# The kit's defaults, the loop's own names from .kit/1st.md: a branch is <key>-<slug>, as kit issue makes it; a
# commit subject opens with its issue's reference, #7 or AB#4521; a change's title is a line of at most 120.
declare -gr policy_default_branch='^[a-z0-9]+-[a-z0-9][a-z0-9-]*$'
declare -gr policy_default_commit_subject='^([A-Z]+)?#[0-9]+ [^ ]'
declare -gr policy_default_change_title='^[^[:cntrl:]]{1,120}$'

# policy:rules prints the policy settings with the kit's defaults filled in, as a JSON object.
policy:rules() {
  local settings key value
  local -A rule=(
    [branch]=${policy_default_branch}
    [commit_subject]=${policy_default_commit_subject}
    [change_title]=${policy_default_change_title}
  )
  settings=$(release:settings)
  while IFS=$'\t' read -r key value
  do
    [[ -n "${key}" ]] || continue
    if [[ -z "${rule[${key}]+x}" ]]
    then
      error "${settings}: policy has a subject the kit does not know: ${key} (the subjects: branch," \
        "commit_subject, change_title)"
      return 1
    fi
    rule[${key}]=${value}
  done < <(yaml:map "${settings}" policy)
  jq --null-input --arg branch "${rule[branch]}" --arg commit_subject "${rule[commit_subject]}" \
    --arg change_title "${rule[change_title]}" \
    '{branch: $branch, commit_subject: $commit_subject, change_title: $change_title}'
}

# policy:branch prints the branch under check: the change's head on a runner, else the current branch, else
# nothing for a detached checkout.
policy:branch() {
  local branch=${GITHUB_HEAD_REF:-${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}}
  [[ -n "${branch}" ]] || branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  printf '%s\n' "${branch}"
}

# policy:input prints the facts document the file header describes. The commits are those past the fork from the
# default branch at origin; where the fork cannot be found, a shallow clone, they are omitted with a warning
# rather than guessed. The change is read through the forge, and omitted where no forge or no change answers.
policy:input() {
  local default_branch branch commits='[]' change='null' rules number title body
  rules=$(policy:rules) || return 1
  default_branch=$(git:default-branch)
  branch=$(policy:branch)
  if [[ -n "${branch}" && "${branch}" != "${default_branch}" ]]
  then
    if git rev-parse --verify --quiet "origin/${default_branch}" >/dev/null \
      && git merge-base "origin/${default_branch}" HEAD >/dev/null 2>&1
    then
      commits=$(git log --no-merges --format='%H%x1f%s' "origin/${default_branch}..HEAD" \
        | jq --raw-input --slurp 'split("\n") | map(select(. != "") | split("\u001f") | {sha: .[0], subject: .[1]})')
    else
      warn "the fork from origin/${default_branch} is not here, so no commit is checked; fetch with full history"
    fi
    if use forge 2>/dev/null && number=$(forge:pr-here 2>/dev/null) && [[ -n "${number}" ]]
    then
      title=$(forge:pr-title "${number}" 2>/dev/null || true)
      body=$(forge:pr-body "${number}" 2>/dev/null || true)
      change=$(jq --null-input --argjson number "${number}" --arg title "${title}" --arg body "${body}" \
        '{number: $number, title: $title, body: $body}')
    fi
  else
    branch=''
  fi
  jq --null-input --arg branch "${branch}" --arg default_branch "${default_branch}" --argjson commits "${commits}" \
    --argjson change "${change}" --argjson policy "${rules}" \
    '{branch: (if $branch == "" then null else $branch end), default_branch: $default_branch, commits: $commits,
      change: $change, policy: $policy}'
}

# policy:engine prints the engine: the policy_engine setting, else opa where .kit/policy holds a .rego file, else
# patterns. An engine the kit does not carry is refused by name.
policy:engine() {
  local engine
  engine=$(release:setting policy_engine)
  if [[ -z "${engine}" ]]
  then
    engine=patterns
    compgen -G '.kit/policy/*.rego' >/dev/null && engine=opa
  fi
  # shellcheck disable=SC2154  # kit_bin_dir is declared by bin/1st.sh
  if [[ ! -f "${kit_bin_dir}/policy-${engine}.sh" ]]
  then
    error "'${engine}' is not a policy engine the kit knows; the engines are: patterns opa (policy_engine in" \
      ".kit/settings.yml)"
    return 1
  fi
  printf '%s\n' "${engine}"
}

# policy:deny <input-file> prints every refusal the engine finds in the facts, one per line, sorted.
policy:deny() {
  local engine
  engine=$(policy:engine) || return 1
  use "policy-${engine}" || return 1
  "policy-${engine}:deny" "$1" | LC_ALL=C sort --unique
}

# ----------------------------------------------------------------------------------------------------------------------
