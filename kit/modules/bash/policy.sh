# shellcheck shell=bash
# policy.sh - the seam over the engine that judges the branch, the commits, the change, and the tree (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# A policy sets what a branch name, a commit subject, a change title, or the tree of files may be. kit policy enforces
# it: it gathers the facts, reads the rules, and calls a provider (the policy engine). The provider lists every
# refusal and every warning. A provider is policy/<name>.sh beside it, whose functions are named policy/<name>:<call>.
#
# The facts are one JSON document, which modules/bash/facts.sh describes and builds. `kit facts` prints it, to write a
# rule against, and so does `kit policy --input`. It holds the moment, the branch, the commits, the change, the tree
# of files git holds, and the policy settings, with the kit's defaults filled in.
#
# The rules are the `policy:` map in kit.yml, one extended regular expression per subject. A subject
# that is not set uses the kit's default below. An empty pattern turns that rule off.
#
#   branch            the branch's name
#   commit_subject    the subject of every commit past the fork, merges excluded
#   change_title      the open change's title
#
# The provider is chosen by:
#
#   1. the `policy_engine` setting in kit.yml, if set
#   2. otherwise opa, if .kit/policy/ contains a .rego file
#   3. otherwise patterns
#
# kit commit reads the policy from the tree it will commit, never from the disk: kit.yml as that tree holds it, and
# each .rego file under .kit/policy/ from that tree when its source is a file of the repository's, such as a package
# of its own. A file the release delivers is read as placed, since the pinned release cannot change under a commit.
# So an edit on the disk that is not staged cannot weaken the judgment of the commit. kit policy reads the disk, which
# in CI is the change's HEAD. policy:from sets this up; policy:from-disk puts it back.
#
# The providers:
#
#   patterns          the default: matches each pattern in bash, with no tool needed
#   opa               runs Open Policy Agent over the Rego in .kit/policy/ (package kit.policy, rules deny and warn)
#
# Every provider implements the calls below, except where noted.
#
#   deny <input-file>       one refusal per line for the facts document in <input-file>
#   warn <input-file>       one warning per line; a provider may implement it, and without it there are none
#
# A refusal fails the command. A warning is printed and never fails it.
#
# A rule that a pattern cannot express can be written in Rego alongside the patterns. The policy/opa package
# delivers the default patterns as Rego, to start from.
#
# Bash matches a pattern as an extended regular expression; OPA matches it as RE2. Write patterns in the subset
# both support: character classes, anchors, alternation, and counted repetition.
#
# see: docs/policy.md

use yaml git settings facts file

# policy_rego is the directory of Rego the opa engine loads: .kit/policy, or the judged tree's copy (policy:from).
declare -g policy_rego=.kit/policy

# The kit's defaults follow the workflow in .kit/1st.md:
#
#   - a branch is <key>-<slug>, as kit work makes it
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
  settings=$(settings:file)
  while IFS=$'\t' read -r key value
  do
    [[ -n "${key}" ]] || continue
    if [[ -z "${rule[${key}]+x}" ]]
    then
      error "kit.yml: policy has a subject the kit does not know: ${key} (the subjects: branch," \
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

# policy:input prints the facts document at the ci moment, as facts:document builds it: the branch's commits, its
# open change, and the tree at HEAD.
policy:input() {
  facts:document ci
}

# policy:engine prints the provider's name, chosen as the file header describes. It fails, naming the engine,
# when the kit has no provider by that name.
policy:engine() {
  local engine
  engine=$(settings:value policy_engine)
  if [[ -z "${engine}" ]]
  then
    engine=patterns
    compgen -G "${policy_rego}/*.rego" >/dev/null && engine=opa
  fi
  if [[ -z "$(use:modules "policy/${engine}")" ]]
  then
    error "'${engine}' is not a policy engine the kit knows; the engines are: patterns opa (policy_engine in" \
      "kit.yml)"
    return 1
  fi
  printf '%s\n' "${engine}"
}

# policy:from <rev> <dir> makes the policy read its settings and its Rego as <rev> holds them, an empty <rev> being the
# index that GIT_INDEX_FILE names, or the default one. It writes them under <dir>, which it makes. A tree with no
# kit.yml reads as one with no settings, the kit's defaults.
policy:from() {
  local rev=$1 dir=$2 rego target root
  mkdir -p "${dir}/rego" || return 1
  git show "${rev}:kit.yml" > "${dir}/kit.yml" 2>/dev/null || : > "${dir}/kit.yml"
  root=$(git rev-parse --show-toplevel) || return 1
  for rego in .kit/policy/*.rego
  do
    [[ -e "${rego}" ]] || continue
    target=$(readlink -f "${rego}")
    if [[ -L "${rego}" && "${target}" == "${root}/"* ]]
    then
      # a source the judged tree lacks is not part of its policy
      git show "${rev}:${target#"${root}/"}" > "${dir}/rego/${rego##*/}" 2>/dev/null || rm -f "${dir}/rego/${rego##*/}"
    else
      cp "${rego}" "${dir}/rego/" || return 1
    fi
  done
  # shellcheck disable=SC2034  # settings:file reads settings_file
  settings_file=${dir}/kit.yml
  policy_rego=${dir}/rego
}

# policy:from-disk makes the policy read kit.yml and .kit/policy from the disk again.
policy:from-disk() {
  # shellcheck disable=SC2034  # settings:file reads settings_file
  settings_file=''
  policy_rego=.kit/policy
}

# policy:deny <input-file> prints every refusal the provider finds in the facts, one per line, sorted and without
# duplicates.
policy:deny() {
  local engine
  engine=$(policy:engine) || return 1
  use "policy/${engine}" || return 1
  # Each stage's status is checked, since a caller may not set pipefail.
  "policy/${engine}:deny" "$1" | LC_ALL=C sort --unique
  local -a statuses=("${PIPESTATUS[@]}")
  (( statuses[0] == 0 && statuses[1] == 0 ))
}

# policy:warn <input-file> prints every warning the provider finds in the facts, one per line, sorted and without
# duplicates. A provider with no warn call has none. A warn call that fails fails this one, so no warning is lost.
policy:warn() {
  local engine
  engine=$(policy:engine) || return 1
  use "policy/${engine}" || return 1
  declare -F "policy/${engine}:warn" >/dev/null || return 0
  "policy/${engine}:warn" "$1" | LC_ALL=C sort --unique
  local -a statuses=("${PIPESTATUS[@]}")
  (( statuses[0] == 0 && statuses[1] == 0 ))
}

# ----------------------------------------------------------------------------------------------------------------------
