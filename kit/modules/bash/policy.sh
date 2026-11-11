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
# kit policy and kit linked judge a branch past the default one by the policy on its base, the ref the commits are
# counted from (policy:base), so a branch cannot loosen the rule that judges it: a pattern it blanks, a login it
# exempts, an engine it names, a file it stops a rule from reading, a forge it drops, a kind it redefines, or a file
# it removes, adds, or edits under .kit/policy/. On a CI runner the base is the one the change's event names, which no
# code of the branch can move, as it could move a ref (policy:runner-base): on GitHub the pull request's base commit,
# pull_request.base.sha in the event at GITHUB_EVENT_PATH, or a merge group's, merge_group.base_sha; on GitLab the
# target branch's tip, CI_MERGE_REQUEST_TARGET_BRANCH_SHA, or origin/<target> fetched when it is not here, never the
# merge base, which the branch chooses by where it forks. A runner that names a change and no base it can have fails
# both. Elsewhere the base is origin/<default>. kit ci runs both before the gate and kit check, which run commands of
# the branch's own.
#
# policy:from-base reads policy, policy_engine, linked_exempt, facts, forge, and forges from the base's kit.yml, each
# absent where the base has none, so the kit's default applies: for forge and forges, the provider the remote's host
# names. forges, the map from remotes to providers, is the base's as forge is, so a branch cannot point the primary at
# a forge that reads no change. Every other setting is the disk's; primary among them, since it names the remote the
# base is read from, and a primary that names no remote fails. A branch that adds a file to facts.read for a rule of
# its own changes nothing until it is merged, as the rule does.
#
# The Rego that judges a branch is exactly the base's, and nothing the branch adds or changes under .kit/policy/
# counts: every file the base holds under .kit/policy/, in a subdirectory too, a link resolved in the base's tree,
# and every file a package the base's kit.yml lists delivers there, a template rendered with the base's answers. So
# what opa loads, Rego and the data files beside it, is the base's. Nothing on the disk under .kit/policy/ is read.
# The kinds the facts' tree is classified by are the base's the same way: those the base holds under .kit/kinds/
# and those its packages deliver there, compiled into a matcher of their own, so a kind the branch redefines, to
# take a path out of a rule keyed on it, classifies nothing until it is merged. A package is found as resolve:home
# finds it, with the base's sources; one under the repository's root, the repository's own or the release in a
# checkout of the kit, is read as the base holds it, and a source's or a release's elsewhere from where it is. A
# package the base lists that is in none of those places fails both. A change to the policy is judged by the policy
# before it, and takes effect once it is on the default branch. On a CI runner the branch is the change's head
# branch, whatever its name, so a detached checkout of the change, a fork's main among them, is judged the same way;
# a merge group's is its head, gh-readonly-queue/<base>/pr-<n>-<sha>, which the kit's default branch pattern refuses,
# so a repository that runs kit ci on a merge queue sets policy.branch to allow it. On the default branch, or a
# detached HEAD outside a runner, both read the disk.
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

use yaml git settings facts file sources context render

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

# policy:branch prints the branch to check: the change's source branch on a CI runner, or a merge group's head there,
# otherwise the current branch, or nothing on a detached HEAD.
policy:branch() {
  local branch=${GITHUB_HEAD_REF:-${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}}
  if [[ -z "${branch}" && "${GITHUB_EVENT_NAME:-}" == merge_group ]]
  then
    # A merge group's event names its head, refs/heads/gh-readonly-queue/<base>/pr-<n>-<sha>; an event that cannot be
    # read names none, which policy:branch-past refuses.
    branch=$(jq --raw-output '.merge_group.head_ref // empty' "${GITHUB_EVENT_PATH:-/dev/null}" 2>/dev/null) \
      || branch=''
    printf '%s\n' "${branch#refs/heads/}"
    return 0
  fi
  # A detached HEAD has no branch, and prints nothing.
  [[ -n "${branch}" ]] || branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  printf '%s\n' "${branch}"
}

# policy:change-named succeeds where the environment names the change a CI runner builds: a pull request's head
# branch, GITHUB_HEAD_REF; a merge request's, CI_MERGE_REQUEST_SOURCE_BRANCH_NAME; or a merge group's event,
# GITHUB_EVENT_NAME merge_group. Such a change is past its base whatever its branch is named.
policy:change-named() {
  [[ -n "${GITHUB_HEAD_REF:-}${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}" || "${GITHUB_EVENT_NAME:-}" == merge_group ]]
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
# branch or a detached HEAD. A change a CI runner names is past its base whatever its branch is named, as a fork's
# main is (policy:change-named). It returns 1 when the default branch cannot be read, or a merge group's head is not
# in its event.
policy:branch-past() {
  local branch default_branch
  branch=$(policy:branch)
  if [[ -z "${branch}" ]]
  then
    [[ "${GITHUB_EVENT_NAME:-}" == merge_group ]] || return 0
    error "the merge group's head is not in the runner's event (merge_group.head_ref is not in" \
      "${GITHUB_EVENT_PATH:+the event at }${GITHUB_EVENT_PATH:-GITHUB_EVENT_PATH, which is not set}), so its policy" \
      "cannot be read"
    return 1
  fi
  if ! policy:change-named
  then
    default_branch=$(git:default-branch) || return 1
    [[ "${branch}" != "${default_branch}" ]] || return 0
  fi
  printf '%s\n' "${branch}"
}

# policy:base-name prints the name of a branch's base, whose policy judges it and from which facts:document counts
# its commits: on a runner, the commit the change's event names (policy:runner-base); elsewhere
# refs/remotes/origin/<default>, written in full, since a tag or a branch named origin/<default> outranks the short
# name. It returns 1, saying so, when a runner names a change and its base cannot be had.
policy:base-name() {
  local base
  base=$(policy:runner-base) || return 1
  if [[ -z "${base}" ]]
  then
    base=$(git:default-branch) || return 1
    base=refs/remotes/origin/${base}
  fi
  printf '%s\n' "${base}"
}

# policy:runner-base prints the base a CI runner's event names for the change it builds, which no code of the branch
# can move, and nothing where no change is named, as at a desk or in a push build. On GitHub it is the pull request's
# base commit, pull_request.base.sha in the event at GITHUB_EVENT_PATH, or a merge group's, merge_group.base_sha. On
# GitLab it is the target branch's tip: CI_MERGE_REQUEST_TARGET_BRANCH_SHA where a merged-results pipeline sets it,
# else the commit at the tip of the branch CI_MERGE_REQUEST_TARGET_BRANCH_NAME names, as origin has it
# (policy:target-here); never CI_MERGE_REQUEST_DIFF_BASE_SHA, the merge base, which a branch chooses by where it
# forks. It returns 1, saying so, where a change is named (policy:change-named) and its base is not in the event, is
# no commit, or cannot be had.
policy:runner-base() {
  local event=${GITHUB_EVENT_PATH:-} key=pull_request.base.sha commit='' target unread=''
  if [[ -n "${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-}" ]]
  then
    commit=${CI_MERGE_REQUEST_TARGET_BRANCH_SHA:-}
    target=${CI_MERGE_REQUEST_TARGET_BRANCH_NAME:-}
    if [[ -z "${commit}" && -n "${target}" ]]
    then
      policy:target-here "${target}" || return 1
      return 0
    fi
    [[ -n "${commit}" ]] || unread='CI_MERGE_REQUEST_TARGET_BRANCH_NAME is not set'
  else
    [[ "${GITHUB_EVENT_NAME:-}" != merge_group ]] || key=merge_group.base_sha
    if [[ -n "${event}" ]]
    then
      # An event with no change, as a push's, names no base; so does one that cannot be read, which fails below where
      # a change is named.
      commit=$(jq --raw-output ".${key} // empty" "${event}" 2>/dev/null) || commit=''
    fi
    if [[ -z "${commit}" ]] && policy:change-named
    then
      unread="${key} is not in ${event:+the event at }${event:-GITHUB_EVENT_PATH, which is not set}"
    fi
  fi
  if [[ -z "${unread}" && -n "${commit}" && ! "${commit}" =~ ^[0-9a-f]{40}([0-9a-f]{24})?$ ]]
  then
    unread="'${commit}' is not a commit"
  fi
  if [[ -n "${unread}" ]]
  then
    error "the base of the change is not in the runner's event (${unread}), so its policy cannot be read"
    return 1
  fi
  [[ -z "${commit}" ]] || printf '%s\n' "${commit}"
}

# policy:target-here <branch> fetches a merge request's target from origin into refs/remotes/origin/<branch>, forced
# and with no tags, so a ref an earlier job left or moved counts for nothing, and prints the commit at its tip. The
# name reaches git as an argument, so one git refuses as a branch name is refused first, named. It returns 1, saying
# so, when the target cannot be had.
policy:target-here() {
  local target=$1
  if [[ "${target}" == -* ]] || ! git check-ref-format --branch "${target}" >/dev/null 2>&1
  then
    error "the base of the change cannot be named ('${target}' in CI_MERGE_REQUEST_TARGET_BRANCH_NAME is not a" \
      "branch name), so its policy cannot be read"
    return 1
  fi
  if ! git fetch --quiet --no-tags origin "+refs/heads/${target}:refs/remotes/origin/${target}" 2>/dev/null \
    || ! git rev-parse --verify --quiet "refs/remotes/origin/${target}^{commit}"
  then
    error "the base of the change, origin/${target}, the merge request's target, could not be fetched, so its" \
      "policy cannot be read"
    return 1
  fi
}

# policy:base prints a branch's base, as policy:base-name names it, once it is here. It returns 1, saying so, when the
# base cannot be named or is not here.
policy:base() {
  local base
  base=$(policy:base-name) || return 1
  if ! git rev-parse --verify --quiet "${base}^{commit}" >/dev/null
  then
    error "the base, ${base#refs/remotes/}, is not here, so its policy cannot be read; fetch it with full history"
    return 1
  fi
  printf '%s\n' "${base}"
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

# policy:settings <rev> <file> writes the kit.yml <rev> holds to <file>, an empty <rev> being the index that
# GIT_INDEX_FILE names, or the default one. A tree with no kit.yml reads as one with no settings, the kit's defaults,
# and <file> is empty. It returns 1 when the file <rev> holds cannot be read.
policy:settings() {
  local rev=$1 file=$2 object
  object=$(policy:object "${rev}" kit.yml) || return 1
  if [[ -z "${object}" ]]
  then
    : > "${file}"
  else
    policy:blob "${object}" "${file}" "kit.yml in ${rev:-the index}" || return 1
  fi
}

# policy:from <rev> <dir> makes the policy read its settings and its Rego as <rev> holds them, an empty <rev> being the
# index that GIT_INDEX_FILE names, or the default one. It writes them under <dir>, which it makes. A tree with no
# kit.yml reads as one with no settings, the kit's defaults, and a source of a link under .kit/policy/ that the tree
# lacks is not part of its policy. It returns 1 when a file the tree holds cannot be read.
policy:from() {
  local rev=$1 dir=$2 rego target root object where=${1:-the index}
  mkdir -p "${dir}/rego" || return 1
  policy:settings "${rev}" "${dir}/kit.yml" || return 1
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
# header describes: <dir>/kit.yml as the disk's kit.yml with policy, policy_engine, linked_exempt, facts, forge, and
# forges as the base's kit.yml has them, absent where it has none; <dir>/rego as the base holds and delivers
# .kit/policy/, and no other; and the facts' kinds as the base holds and delivers .kit/kinds/, compiled into
# <dir>/kinds.awk (policy:base-tree). Nothing on the disk under .kit/ is read. Where the base has no kit.yml, it
# leaves the policy reading the disk. It returns 1 when the base cannot be named or is not here, or a file of it, a
# source it names, or a package it lists cannot be read.
policy:from-base() {
  local dir=$1 base object disk_doc base_doc
  base=$(policy:base) || return 1
  object=$(policy:object "${base}" kit.yml) || return 1
  # The kit is not installed on the base, so the branch that installs it sets the first policy.
  [[ -n "${object}" ]] || return 0
  mkdir -p "${dir}" || return 1
  policy:blob "${object}" "${dir}/base.yml" "kit.yml in ${base}" || return 1
  disk_doc=$(yaml:json kit.yml) || return 1
  base_doc=$(yaml:json "${dir}/base.yml") || return 1
  # JSON is YAML, so the settings readers read the merged document as they read kit.yml.
  if ! jq --null-input --compact-output --argjson disk "${disk_doc:-null}" --argjson base "${base_doc:-null}" '
    ["policy", "policy_engine", "linked_exempt", "facts", "forge", "forges"] as $judging
    | (($disk // {}) | with_entries(select(.key as $key | $judging | index($key) | not)))
      + (($base // {}) | with_entries(select(.key as $key | $judging | index($key))))' > "${dir}/kit.yml"
  then
    error "run: kit check (kit.yml on the disk or on ${base} is not a map of settings, so the policy cannot be read)"
    return 1
  fi
  # shellcheck disable=SC2034  # settings:file reads settings_file
  settings_file=${dir}/kit.yml
  policy:base-tree "${base}" "${dir}" || return 1
  policy_rego=${dir}/rego
  context:kinds-from "${dir}/kinds" "${dir}/kinds.awk"
}

# policy:base-tree <base> <dir> makes <dir>/rego hold exactly what <base> holds or delivers under .kit/policy/, and
# <dir>/kinds what it holds or delivers under .kit/kinds/, whatever the disk holds: every file <base> holds under the
# directory, at its path below it, a link resolved in <base>'s tree (policy:base-held), then what each package
# <dir>/base.yml lists under packages delivers there, the later package in the list winning, as delivery does
# (policy:home-held). A package is looked up as resolve:home looks it up, with the base's sources: in the sources
# the base names, the last first, fetched where the cache lacks them, then in the release, ${kit_root}, then in the
# repository's own .kit/packages/. A home under the repository's root, as the repository's own packages are, and the
# release is in a checkout of the kit itself, is read as <base> holds it, never from the disk; a source's or a
# release's elsewhere from where it is. It returns 1 when a list or a file cannot be read, a source cannot be
# fetched, a template cannot be rendered, or a package is in none of those places, naming it.
policy:base-tree() {
  local base=$1 dir=$2 package root home row object
  local -a packages rows homes=()
  root=$(cd "$(git rev-parse --show-toplevel)" && pwd -P) || return 1
  rm -rf "${dir:?}/rego" "${dir:?}/kinds" "${dir:?}/kinds.awk" "${dir:?}/home" || return 1
  mkdir -p "${dir}/rego" "${dir}/kinds" || return 1
  policy:base-held "${base}" .kit/policy "${dir}/rego" || return 1
  policy:base-held "${base}" .kit/kinds "${dir}/kinds" || return 1
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
    policy:home-held "${base}" "${root}" "${home}" "${package}" "${dir}" policy "${dir}/rego" || return 1
    policy:home-held "${base}" "${root}" "${home}" "${package}" "${dir}" kinds "${dir}/kinds" || return 1
  done
}

# policy:base-held <base> <path> <into> writes into the directory <into> every file <base> holds under the directory
# <path>, at its path below it: a file as <base> holds it, and a link as <base> holds its target, resolved in
# <base>'s tree. A link whose target <base> does not hold is not part of it, and a submodule is no file. It returns 1
# when a file cannot be read, or a link leaves the tree or names something other than a file.
policy:base-held() {
  local base=$1 path=${2%/} into=$3 entry name object target
  local -a entries
  if ! list:read0 entries git ls-tree -z -r --full-tree "${base}" -- "${path}/" 2>/dev/null
  then
    error "run: git fsck (${path}/ in ${base} could not be read)"
    return 1
  fi
  for entry in "${entries[@]}"
  do
    # "<mode> <type> <object>\t<path>"
    name=${entry#*$'\t'}
    [[ "${entry}" == 1[02]0* ]] || continue
    object=${entry#* * }
    object=${object%%$'\t'*}
    if [[ "${entry}" == 120000* ]]
    then
      if ! target=$(git cat-file blob "${object}" 2>/dev/null)
      then
        error "run: git fsck (${name} in ${base} could not be read)"
        return 1
      fi
      if ! target=$(policy:tree-path "${name%/*}" "${target}")
      then
        error "make ${name} on ${base} a link inside the repository (it leaves the tree)"
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
        error "make ${name} on ${base} a link to a file (${target} is not one)"
        return 1
      fi
      object=${entry#* * }
      object=${object%%$'\t'*}
    fi
    name=${name#"${path}/"}
    mkdir -p "${into}/$(dirname "${name}")" || return 1
    policy:blob "${object}" "${into}/${name}" "${path}/${name} in ${base}" || return 1
  done
}

# policy:home-held <base> <root> <home> <package> <dir> <sub> <into> writes into the directory <into> what the
# package at <home> delivers under .kit/<sub>/: every file under its files/repo/.kit/<sub>/, at its path below it,
# then its templates/repo/.kit/<sub>/, rendered as kit hydrate renders a template (render:fill) with the answers the
# base's kit.yml, <dir>/base.yml, and the package's package.yml give (settings:answers). A home under <root> is read
# as <base> holds it, into <dir>/home/, never from the disk; one elsewhere from where it is. It returns 1 when a file
# cannot be read or a template cannot be rendered.
policy:home-held() {
  local base=$1 root=$2 home=$3 package=$4 dir=$5 sub=$6 into=$7 held object
  local -a rows
  held=${home}/packages/${package}
  if [[ "${home}/" == "${root}/"* ]]
  then
    held=${dir}/home/${home#"${root}/"}/packages/${package}
    policy:base-held "${base}" "${home#"${root}/"}/packages/${package}/files/repo/.kit/${sub}" \
      "${held}/files/repo/.kit/${sub}" || return 1
    policy:base-held "${base}" "${home#"${root}/"}/packages/${package}/templates/repo/.kit/${sub}" \
      "${held}/templates/repo/.kit/${sub}" || return 1
    if [[ -d "${held}/templates/repo/.kit/${sub}" ]]
    then
      object=$(policy:object "${base}" "${home#"${root}/"}/packages/${package}/package.yml") || return 1
      policy:blob "${object}" "${held}/package.yml" "${home#"${root}/"}/packages/${package}/package.yml in ${base}" \
        || return 1
    fi
  fi
  if [[ -d "${held}/files/repo/.kit/${sub}" ]]
  then
    cp -R -L "${held}/files/repo/.kit/${sub}/." "${into}/" || return 1
  fi
  [[ -d "${held}/templates/repo/.kit/${sub}" ]] || return 0
  list:read rows settings:answers "${held}/package.yml" "${dir}/base.yml" || return 1
  render:fill "${held}/templates/repo/.kit/${sub}" "${into}" "${package}" "${rows[@]}"
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
