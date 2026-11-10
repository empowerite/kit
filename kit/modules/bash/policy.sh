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
# So an edit on the disk that is not staged cannot weaken the judgment of the commit. policy:from sets this up;
# policy:from-disk puts it back.
#
# kit policy and kit linked judge a branch past the default one by the policy on its base, origin/<default>, the ref
# the commits are counted from (policy:base), so a branch cannot loosen the rule that judges it: a pattern it blanks, a
# login it exempts, an engine it names, a file it stops a rule from reading, a forge it drops, or a .rego file it
# removes, adds, or edits. policy:from-base reads policy, policy_engine, linked_exempt, facts, and forge from the
# base's kit.yml, each absent where the base has none, so the kit's default applies: for forge, the provider the
# remote's host names. Every other setting is the disk's; primary among them, since it names the remote the base is
# read from, and a primary that names no remote fails. A branch that adds a file to facts.read for a rule of its own
# changes nothing until it is merged, as the rule does.
#
# The Rego that judges a branch is exactly the base's, and nothing the branch adds or changes under .kit/policy/
# counts: each .rego file the base holds under .kit/policy/, a link resolved in the base's tree, and each .rego file a
# package the base's kit.yml lists delivers there. A package is found as resolve:home finds it, with the base's
# sources; one under the repository's root, the repository's own or the release in a checkout of the kit, is read as
# the base holds it, and a source's or a release's elsewhere from where it is. A package the base lists that is in
# none of those places fails both. A change to the policy is judged by the policy before it, and takes effect once it
# is on the default branch. On a CI runner the branch is the change's head branch, whatever its name, so a detached
# checkout of the change, a fork's main among them, is judged the same way. On the default branch, or a detached HEAD
# outside a runner, both read the disk.
#
# A base with no kit.yml is one the kit is not installed on: there, both read the checkout's own settings and Rego, as
# on the default branch, so the change that installs the kit sets its first policy. A base that is not here, as in a
# clone without it, fails both, since its policy cannot be read; so does a read of the base's files that fails.
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

use yaml git settings facts file sources

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
  local settings key value row
  local -a rows
  local -A rule=(
    [branch]=${policy_default_branch}
    [commit_subject]=${policy_default_commit_subject}
    [change_title]=${policy_default_change_title}
  )
  settings=$(settings:file)
  list:read rows yaml:map "${settings}" policy || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r key value <<<"${row}"
    [[ -n "${key}" ]] || continue
    if [[ -z "${rule[${key}]+x}" ]]
    then
      error "kit.yml: policy has a subject the kit does not know: ${key} (the subjects: branch," \
        "commit_subject, change_title)"
      return 1
    fi
    rule[${key}]=${value}
  done
  jq --null-input --arg branch "${rule[branch]}" --arg commit_subject "${rule[commit_subject]}" \
    --arg change_title "${rule[change_title]}" \
    '{branch: $branch, commit_subject: $commit_subject, change_title: $change_title}'
}

# policy:branch prints the branch to check: the change's source branch on a CI runner, otherwise the current
# branch, or nothing on a detached HEAD.
policy:branch() {
  local branch=${GITHUB_HEAD_REF:-${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}}
  # A detached HEAD has no branch, and prints nothing.
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

# policy:branch-past prints the branch policy:branch names when it is past the default one, and nothing on the default
# branch or a detached HEAD. A change's head branch on a CI runner is past its base whatever its name, as a fork's
# main is. It returns 1 when the default branch cannot be read.
policy:branch-past() {
  local branch default_branch
  branch=$(policy:branch)
  [[ -n "${branch}" ]] || return 0
  if [[ -z "${GITHUB_HEAD_REF:-}${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}" ]]
  then
    default_branch=$(git:default-branch) || return 1
    [[ "${branch}" != "${default_branch}" ]] || return 0
  fi
  printf '%s\n' "${branch}"
}

# policy:base prints origin/<default branch>, the ref facts:document counts commits from, whose policy judges a branch
# past it. It returns 1, saying so, when the ref is not here.
policy:base() {
  local default_branch
  default_branch=$(git:default-branch) || return 1
  if ! git rev-parse --verify --quiet "origin/${default_branch}^{commit}" >/dev/null
  then
    error "the base, origin/${default_branch}, is not here, so its policy cannot be read; fetch it with full history"
    return 1
  fi
  printf 'origin/%s\n' "${default_branch}"
}

# policy:object <rev> <path> prints the object name of <path> in <rev>, an empty <rev> being the index, and nothing
# when <rev> holds no such path. It returns 1, saying so, when git cannot list it.
policy:object() {
  local rev=$1 path=$2 entry
  local -a entries
  if [[ -n "${rev}" ]]
  then
    if ! entry=$(git ls-tree --full-tree "${rev}" -- "${path}" 2>/dev/null)
    then
      error "run: git fsck (${path} in ${rev} could not be read)"
      return 1
    fi
    # "<mode> <type> <object>\t<path>"
    entry=${entry#* * }
    printf '%s\n' "${entry%%$'\t'*}"
    return 0
  fi
  if ! list:read entries git ls-files --stage -- "${path}" 2>/dev/null
  then
    error "run: git fsck (${path} in the index could not be read)"
    return 1
  fi
  # "<mode> <object> <stage>\t<path>", of which stage 0 is the one git show :<path> reads
  for entry in "${entries[@]}"
  do
    entry=${entry#* }
    [[ "${entry}" == *" 0"$'\t'* ]] || continue
    printf '%s\n' "${entry%% *}"
    return 0
  done
  printf '\n'
}

# policy:blob <object> <file> <what> writes the blob <object> to <file>. It returns 1, naming <what>, when git cannot
# read it.
policy:blob() {
  if ! git cat-file blob "$1" > "$2" 2>/dev/null
  then
    error "run: git fsck ($3 could not be read)"
    return 1
  fi
}

# policy:from <rev> <dir> makes the policy read its settings and its Rego as <rev> holds them, an empty <rev> being the
# index that GIT_INDEX_FILE names, or the default one. It writes them under <dir>, which it makes. A tree with no
# kit.yml reads as one with no settings, the kit's defaults, and a source of a link under .kit/policy/ that the tree
# lacks is not part of its policy. It returns 1 when a file the tree holds cannot be read.
policy:from() {
  local rev=$1 dir=$2 rego target root object where=${1:-the index}
  mkdir -p "${dir}/rego" || return 1
  object=$(policy:object "${rev}" kit.yml) || return 1
  if [[ -z "${object}" ]]
  then
    : > "${dir}/kit.yml"
  else
    policy:blob "${object}" "${dir}/kit.yml" "kit.yml in ${where}" || return 1
  fi
  root=$(git rev-parse --show-toplevel) || return 1
  for rego in .kit/policy/*.rego
  do
    [[ -e "${rego}" ]] || continue
    target=$(readlink -f "${rego}")
    if [[ -L "${rego}" && "${target}" == "${root}/"* ]]
    then
      object=$(policy:object "${rev}" "${target#"${root}/"}") || return 1
      rm -f "${dir}/rego/${rego##*/}"
      [[ -z "${object}" ]] || policy:blob "${object}" "${dir}/rego/${rego##*/}" \
        "${target#"${root}/"} in ${where}" || return 1
    else
      cp "${rego}" "${dir}/rego/" || return 1
    fi
  done
  # shellcheck disable=SC2034  # settings:file reads settings_file
  settings_file=${dir}/kit.yml
  policy_rego=${dir}/rego
}

# policy:from-base <dir> makes the policy read, under <dir>, what judges a branch past the default one, as the file
# header describes: policy:from the base, then <dir>/kit.yml as the disk's kit.yml with policy, policy_engine,
# linked_exempt, facts, and forge as the base's kit.yml has them, absent where it has none, and <dir>/rego as
# policy:base-rego makes it, the base's Rego and no other. Where the base has no kit.yml, it leaves the policy reading
# the disk. It returns 1 when the base is not here, or a file of it, a source it names, or a package it lists cannot be
# read.
policy:from-base() {
  local dir=$1 base object disk_doc base_doc
  base=$(policy:base) || return 1
  object=$(policy:object "${base}" kit.yml) || return 1
  # The kit is not installed on the base, so the branch that installs it sets the first policy.
  [[ -n "${object}" ]] || return 0
  policy:from "${base}" "${dir}" || return 1
  mv "${dir}/kit.yml" "${dir}/base.yml" || return 1
  disk_doc=$(yaml:json kit.yml) || return 1
  base_doc=$(yaml:json "${dir}/base.yml") || return 1
  # JSON is YAML, so the settings readers read the merged document as they read kit.yml.
  if ! jq --null-input --compact-output --argjson disk "${disk_doc:-null}" --argjson base "${base_doc:-null}" '
    ["policy", "policy_engine", "linked_exempt", "facts", "forge"] as $judging
    | (($disk // {}) | with_entries(select(.key as $key | $judging | index($key) | not)))
      + (($base // {}) | with_entries(select(.key as $key | $judging | index($key))))' > "${dir}/kit.yml"
  then
    error "run: kit check (kit.yml on the disk or on ${base} is not a map of settings, so the policy cannot be read)"
    return 1
  fi
  # shellcheck disable=SC2034  # settings:file reads settings_file
  settings_file=${dir}/kit.yml
  policy:base-rego "${base}" "${dir}"
}

# policy:base-rego <base> <dir> makes <dir>/rego hold exactly the Rego <base> holds or delivers, whatever the disk
# holds: each .rego file <base> holds under .kit/policy/, a link resolved in <base>'s tree (policy:base-policy), and
# each .rego file that a package <dir>/base.yml lists under packages delivers to .kit/policy/, the later package in the
# list winning, as delivery does. A package is looked up as resolve:home looks it up, with the base's sources: in the
# sources the base names, the last first, fetched where the cache lacks them, then in the release, ${kit_root}, then
# in the repository's own .kit/packages/. A home under the repository's root, as the repository's own packages are,
# and the release is in a checkout of the kit itself, is read as <base> holds it, never from the disk; a source's or
# a release's elsewhere from where it is. It returns 1 when a list or a file cannot be read, a source cannot be
# fetched, or a package is in none of those places, naming it.
policy:base-rego() {
  local base=$1 dir=$2 package rego root home row object
  local -a packages rows homes=()
  root=$(cd "$(git rev-parse --show-toplevel)" && pwd -P) || return 1
  rm -f "${dir}/rego/"*.rego || return 1
  policy:base-policy "${base}" "${dir}/rego" || return 1
  list:read packages yaml:list "${dir}/base.yml" packages || return 1
  (( ${#packages[@]} > 0 )) || return 0
  sources:fetch-all "${dir}/base.yml" || return 1
  list:read rows sources:roots "${dir}/base.yml" || return 1
  # The last source first, as resolve:home looks; a home under the root is named from it, and read from the base.
  for row in "${rows[@]}"
  do
    [[ -z "${row}" ]] || homes=("${row#*$'\t'}" "${homes[@]}")
  done
  # shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
  home=$(cd "${kit_root}" && pwd -P) || return 1
  homes+=("${home}" "${root}/.kit")
  for package in "${packages[@]}"
  do
    [[ -n "${package}" ]] || continue
    for home in "${homes[@]}" ''
    do
      [[ -n "${home}" ]] || break
      if [[ "${home}/" != "${root}/"* ]]
      then
        [[ -f "${home}/packages/${package}/package.yml" ]] && break
        continue
      fi
      object=$(policy:object "${base}" "${home#"${root}/"}/packages/${package}/package.yml") || return 1
      [[ -z "${object}" ]] || break
    done
    if [[ -z "${home}" ]]
    then
      error "rebase onto ${base} (the package ${package}, which its kit.yml lists, is in none of its sources, the" \
        "release, or its .kit/packages/, so its Rego cannot be read)"
      return 1
    fi
    if [[ "${home}/" == "${root}/"* ]]
    then
      list:read rows policy:tree-rego "${base}" "${home#"${root}/"}/packages/${package}/files/repo/.kit/policy/" \
        "${dir}/rego" || return 1
      continue
    fi
    for rego in "${home}/packages/${package}/files/repo/.kit/policy/"*.rego
    do
      [[ ! -e "${rego}" ]] || cp -f "${rego}" "${dir}/rego/" || return 1
    done
  done
}

# policy:base-policy <base> <rego> writes into the directory <rego> each .rego file <base> holds under .kit/policy/: a
# file as <base> holds it, and a link as <base> holds its target, resolved in <base>'s tree. A link whose target <base>
# does not hold is not part of its policy. It returns 1 when a file cannot be read, or a link leaves the tree or names
# another link.
policy:base-policy() {
  local base=$1 into=$2 entry name object target
  local -a entries
  if ! list:read0 entries git ls-tree -z --full-tree "${base}" -- .kit/policy/ 2>/dev/null
  then
    error "run: git fsck (.kit/policy/ in ${base} could not be read)"
    return 1
  fi
  for entry in "${entries[@]}"
  do
    # "<mode> <type> <object>\t<path>"
    name=${entry#*$'\t'}
    name=${name##*/}
    [[ "${name}" == *.rego && "${entry}" == 1[02]0* ]] || continue
    object=${entry#* * }
    object=${object%%$'\t'*}
    if [[ "${entry}" == 120000* ]]
    then
      if ! target=$(git cat-file blob "${object}" 2>/dev/null)
      then
        error "run: git fsck (.kit/policy/${name} in ${base} could not be read)"
        return 1
      fi
      if ! target=$(policy:tree-path .kit/policy "${target}")
      then
        error "make .kit/policy/${name} on ${base} a link inside the repository (it leaves the tree)"
        return 1
      fi
      if ! entry=$(git ls-tree --full-tree "${base}" -- "${target}" 2>/dev/null)
      then
        error "run: git fsck (${target} in ${base} could not be read)"
        return 1
      fi
      [[ -n "${entry}" ]] || continue
      if [[ "${entry}" != 100* ]]
      then
        error "make .kit/policy/${name} on ${base} a link to a file (${target} is not one)"
        return 1
      fi
      object=${entry#* * }
      object=${object%%$'\t'*}
    fi
    policy:blob "${object}" "${into}/${name}" ".kit/policy/${name} in ${base}" || return 1
  done
}

# policy:tree-path <dir> <target> prints the repository-relative path a link in <dir> to <target> names, its . and ..
# taken out. It returns 1 for a target that is absolute or leaves the tree.
policy:tree-path() {
  local part
  local -a parts kept=()
  [[ "$2" != /* ]] || return 1
  IFS=/ read -r -a parts <<<"$1/$2"
  for part in "${parts[@]}"
  do
    case "${part}" in
      ('' | .) ;;
      (..)
        (( ${#kept[@]} > 0 )) || return 1
        unset 'kept[-1]'
        ;;
      (*) kept+=("${part}")
    esac
  done
  (( ${#kept[@]} > 0 )) || return 1
  local IFS=/
  printf '%s\n' "${kept[*]}"
}

# policy:tree-rego <rev> <path> <rego> writes each .rego file directly under <path> in <rev> into the directory <rego>,
# over a file of the same name, and prints each name it writes. A regular file counts, since the links the kit places
# are ones git ignores. It returns 1 when the list or a file cannot be read.
policy:tree-rego() {
  local rev=$1 path=$2 into=$3 entry rego
  local -a entries
  if ! list:read0 entries git ls-tree -z --full-tree "${rev}" -- "${path}" 2>/dev/null
  then
    error "run: git fsck (${path} in ${rev} could not be read)"
    return 1
  fi
  for entry in "${entries[@]}"
  do
    # "<mode> <type> <object>\t<path>"
    rego=${entry#*$'\t'}
    [[ "${rego}" == *.rego && "${entry}" == 100* ]] || continue
    entry=${entry#* * }
    policy:blob "${entry%%$'\t'*}" "${into}/${rego##*/}" "${rego} in ${rev}" || return 1
    printf '%s\n' "${rego##*/}"
  done
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
