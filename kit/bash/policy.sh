# shellcheck shell=bash
# policy.sh - the seam over the engine that checks branch names, commit subjects, and change titles (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# A policy sets what a branch name, a commit subject, or a change title may be. kit policy enforces it: it gathers
# the facts, reads the rules, and calls a provider (the policy engine) to list every refusal. A provider is
# policy-<name>.sh in this directory, whose functions are named policy-<name>:<call>.
#
# The facts are one JSON document, built by policy:input. `kit policy --input` prints it, to write a rule against.
# It contains:
#
#   - the branch and the default branch
#   - the commits past the fork from the default branch, merges excluded
#   - the open change on the branch, with its title and body
#   - the policy settings, with the kit's defaults filled in
#
# The rules are the `policy:` map in .kit/settings.yml, one extended regular expression per subject. A subject
# that is not set uses the kit's default below. An empty pattern turns that rule off.
#
#   branch            the branch's name
#   commit_subject    the subject of every commit past the fork, merges excluded
#   change_title      the open change's title
#
# The provider is chosen by:
#
#   1. the `policy_engine` setting in .kit/settings.yml, if set
#   2. otherwise opa, if .kit/policy/ contains a .rego file
#   3. otherwise patterns
#
# The providers:
#
#   patterns          the default: matches each pattern in bash, with no tool needed
#   opa               runs Open Policy Agent over the Rego in .kit/policy/ (package kit.policy, rule deny)
#
# Every provider implements one call:
#
#   deny <input-file>       one refusal per line for the facts document in <input-file>
#
# Both providers read the same facts document. A rule that a pattern cannot express can be written in Rego
# alongside the patterns. The policy/opa package delivers the default patterns as Rego, to start from.
#
# Bash matches a pattern as an extended regular expression; OPA matches it as RE2. Write patterns in the subset
# both support: character classes, anchors, alternation, and counted repetition.
#
# see: docs/policy.md

use release yaml git

# The kit's defaults follow the workflow in .kit/1st.md:
#
#   - a branch is <key>-<slug>, as kit issue makes it
#   - a commit subject starts with its issue's reference, #7 or AB#4521
#   - a change title is one line of at most 120 characters
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

# policy:branch prints the branch to check: the change's source branch on a CI runner, otherwise the current
# branch, or nothing on a detached HEAD.
policy:branch() {
  local branch=${GITHUB_HEAD_REF:-${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}}
  [[ -n "${branch}" ]] || branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  printf '%s\n' "${branch}"
}

# policy:input prints the facts document described in the file header. The commits are those past the fork from
# origin's default branch. When the fork point is missing, as in a shallow clone, it warns and omits the commits
# instead of guessing. The change is read from the forge (GitHub or GitLab). It is omitted when there is no forge
# or no open change. On the default branch or a detached HEAD, the branch is null and there are no commits or
# change.
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

# policy:engine prints the provider's name, chosen as the file header describes. It fails, naming the engine,
# when the kit has no provider by that name.
policy:engine() {
  local engine
  engine=$(release:setting policy_engine)
  if [[ -z "${engine}" ]]
  then
    engine=patterns
    compgen -G '.kit/policy/*.rego' >/dev/null && engine=opa
  fi
  if [[ -z "$(use:modules "policy-${engine}")" ]]
  then
    error "'${engine}' is not a policy engine the kit knows; the engines are: patterns opa (policy_engine in" \
      ".kit/settings.yml)"
    return 1
  fi
  printf '%s\n' "${engine}"
}

# policy:deny <input-file> prints every refusal the provider finds in the facts, one per line, sorted and without
# duplicates.
policy:deny() {
  local engine
  engine=$(policy:engine) || return 1
  use "policy-${engine}" || return 1
  "policy-${engine}:deny" "$1" | LC_ALL=C sort --unique
}

# ----------------------------------------------------------------------------------------------------------------------
