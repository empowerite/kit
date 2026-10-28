# shellcheck shell=bash
# git.sh - git helpers: the default branch, worktrees, kit tidy, and kit rebase (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit tidy (git:tidy) decides whether a worktree's work is finished by asking:
#
#   - the forge, the service that hosts the repository (GitHub, GitLab), through bash/forge.sh: is the branch's pull
#     request merged or closed?
#   - when the branch has no pull request, the issue's tracker, through bash/tracker.sh: is the branch's issue
#     closed?

use forge

# git:default-branch [<path>] prints the default branch of the repository at <path>: origin/HEAD, else main.
git:default-branch() {
  local branch
  branch=$(git -C "${1:-.}" symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's@^origin/@@' \
    || true)
  [[ -n "${branch}" ]] || branch=main
  printf '%s\n' "${branch}"
}

# git:main-root [<path>] prints the root of the main checkout, which contains worktrees/. It works from the main
# checkout or from any linked worktree.
git:main-root() {
  dirname "$(git -C "${1:-.}" rev-parse --path-format=absolute --git-common-dir)"
}

# git:worktrees prints "<path>\t<detached>\t<branch>" for every linked worktree under worktrees/. <detached> is
# true or false. <branch> is empty for a detached checkout. The empty field comes last so a tab-separated read
# keeps it.
git:worktrees() {
  local worktree_root path="" branch="" detached=false line
  worktree_root="$(git:main-root)/worktrees"
  while IFS= read -r line
  do
    case "${line}" in
      (worktree\ *)
        path=${line#worktree }
        branch=""
        detached=false
        ;;
      (branch\ refs/heads/*)
        branch=${line#branch refs/heads/}
        ;;
      (detached)
        detached=true
        ;;
      ("")
        # git ends each record with a blank line. The printf below adds one more so the last record is printed too.
        [[ "${path}" == "${worktree_root}"/* ]] && printf '%s\t%s\t%s\n' "${path}" "${detached}" "${branch}"
        path=""
    esac
  done < <(git worktree list --porcelain; printf '\n')
}

# git:gone-branches prints every local branch whose upstream no longer exists.
git:gone-branches() {
  git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads/ | awk '$2 == "[gone]" { print $1 }'
}

# git:clean <path> is true when the checkout at <path> has no uncommitted changes.
git:clean() {
  [[ -z "$(git -C "$1" status --porcelain 2>/dev/null)" ]]
}

# git:tidy-worktree <path> <branch> <detached> <default-branch> <self-root> <dry-run> decides whether to remove
# one worktree and prints the reason. It returns 0 when the worktree was removed (or would be, under a dry run)
# and 1 when it is kept.
git:tidy-worktree() {
  local path=$1 branch=$2 detached=$3 default_branch=$4 self_root=$5 dry_run=$6 state="" ref
  if [[ "${path}" == "${self_root}" ]]
  then
    step "keeping ${path} (this command runs in it)"
    return 1
  fi
  if ! git:clean "${path}"
  then
    step "keeping ${path} (${branch:-detached}, it has uncommitted changes)"
    return 1
  fi
  if [[ -n "${branch}" ]]
  then
    state=$(forge:pr-state-for-branch "${branch}")
    if [[ -z "${state}" ]]
    then
      use tracker
      ref=$(tracker:branch-ref "${branch}")
      [[ -n "${ref}" && "$(tracker:of "${ref}" state)" == CLOSED ]] && state=CLOSED
    fi
  elif [[ "${detached}" == true ]]
  then
    # Remove a detached checkout only when its commit is already on the default branch. Otherwise its HEAD may be
    # the only ref that reaches that commit.
    if git -C "${path}" merge-base --is-ancestor HEAD "origin/${default_branch}" 2>/dev/null
    then
      state=DETACHED
    else
      step "keeping ${path} (detached at a commit origin/${default_branch} does not reach)"
      return 1
    fi
  fi
  case "${state}" in
    (MERGED | CLOSED | DETACHED)
      if [[ "${dry_run}" == true ]]
      then
        step "would remove ${path} (${branch:-detached}, ${state})"
      else
        step "removing ${path} (${branch:-detached}, ${state})"
        git worktree remove "${path}"
      fi
      return 0
      ;;
    (*)
      step "keeping ${path} (${branch:-detached}, change ${state:-open or unknown})"
      return 1
  esac
}

# git:tidy [--dry-run] cleans up finished work. It:
#
#   1. fetches origin with --prune
#   2. removes every clean worktree under worktrees/ whose pull request is merged or closed (or, with no pull
#      request, whose issue is closed), or that is detached at a commit the default branch already has
#   3. prunes git's worktree bookkeeping
#   4. deletes every local branch whose upstream is gone
#
# It never touches:
#
#   - the default branch
#   - a branch checked out in a worktree it keeps
#   - a worktree with uncommitted changes
#   - a detached worktree whose commit nothing else reaches
#   - the worktree it runs in
#
# --dry-run prints what it would do and changes nothing.
git:tidy() {
  local dry_run=false flag repo default_branch self_root found=false path branch detached
  local removed_worktrees=0 removed_branches=0 checked_out
  local -A freed=()
  for flag in "$@"
  do
    case "${flag}" in
      (--dry-run)
        dry_run=true
        ;;
      (*)
        error "unknown flag '${flag}' (only --dry-run is recognized)"
        return 2
    esac
  done
  use forge
  forge:require || return 1
  repo=$(forge:repo) || return 1
  if [[ "${dry_run}" == true ]]
  then
    section "kit tidy, dry run: nothing is changed"
  else
    section "kit tidy: sweeping ${repo} back to pristine"
  fi

  default_branch=$(git:default-branch)
  self_root=$(git rev-parse --show-toplevel)
  if [[ "${dry_run}" == true ]]
  then
    step "would run: git fetch --prune origin"
  else
    step "git fetch --prune origin"
    git fetch --quiet --prune origin
  fi

  section "worktrees whose pull request is merged or closed"
  while IFS=$'\t' read -r path detached branch
  do
    found=true
    if git:tidy-worktree "${path}" "${branch}" "${detached}" "${default_branch}" "${self_root}" "${dry_run}"
    then
      (( ++removed_worktrees ))
      [[ -n "${branch}" ]] && freed[${branch}]=1
    fi
  done < <(git:worktrees)
  [[ "${found}" == true ]] || ok "none"

  section "worktree bookkeeping"
  if [[ "${dry_run}" == true ]]
  then
    git worktree prune --verbose --dry-run || true
  else
    git worktree prune --verbose || true
  fi
  ok "pruned"

  section "local branches whose upstream is gone"
  checked_out=$(git worktree list --porcelain | sed -n 's@^branch refs/heads/@@p')
  while IFS= read -r branch
  do
    [[ -n "${branch}" ]] || continue
    [[ "${branch}" == "${default_branch}" ]] && continue
    if [[ -z "${freed[${branch}]+x}" ]] && grep --quiet --line-regexp --fixed-strings "${branch}" <<<"${checked_out}"
    then
      step "keeping ${branch} (checked out in a worktree)"
      continue
    fi
    if [[ "${dry_run}" == true ]]
    then
      step "would delete ${branch}"
    else
      step "deleting ${branch}"
      git branch -D "${branch}" >/dev/null
    fi
    (( ++removed_branches ))
  done <<<"$(git:gone-branches)"
  (( removed_branches > 0 )) || ok "none"

  if [[ "${dry_run}" == true ]]
  then
    ok "Dry run: would remove ${removed_worktrees} worktree(s) and ${removed_branches} branch(es)."
  else
    ok "Tidied ${repo}: removed ${removed_worktrees} worktree(s) and ${removed_branches} branch(es)."
  fi
}

# git:rebase [<path>] rebases the branch checked out at <path> (default: the current checkout) onto the default
# branch at origin, then pushes it with --force-with-lease. It refuses:
#
#   - a checkout with no branch checked out
#   - the root checkout (the desk, which belongs to the human), whatever branch it has
#   - the default branch, in any checkout
#   - a worktree with uncommitted changes
#
# On a conflict it aborts the rebase and leaves the worktree as it was.
git:rebase() {
  local worktree=${1:-.} branch default_branch
  branch=$(git -C "${worktree}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  if [[ -z "${branch}" ]]
  then
    error "${worktree} has no branch checked out, so there is nothing to rebase"
    return 1
  fi
  if [[ "$(git -C "${worktree}" rev-parse --show-toplevel)" == "$(git:main-root "${worktree}")" ]]
  then
    error "refusing to rebase ${branch}: it is the root checkout, the human's desk; rebase in the branch's worktree"
    return 1
  fi
  default_branch=$(git:default-branch "${worktree}")
  if [[ "${branch}" == "${default_branch}" ]]
  then
    error "refusing to rebase ${branch}: it is the default branch; it advances by: git pull --ff-only"
    return 1
  fi
  if ! git:clean "${worktree}"
  then
    error "refusing to rebase ${branch}: the worktree has uncommitted changes; commit or stash them first"
    return 1
  fi
  section "kit rebase ${branch} onto origin/${default_branch}"
  git -C "${worktree}" fetch --quiet --prune origin
  if ! git -C "${worktree}" rebase --quiet "origin/${default_branch}"
  then
    git -C "${worktree}" rebase --abort 2>/dev/null || true
    error "${branch} conflicts with origin/${default_branch}; the rebase was aborted and the worktree is as it was"
    return 1
  fi
  git -C "${worktree}" push --quiet --force-with-lease
  ok "${branch} is rebased onto origin/${default_branch} and pushed."
}

# ----------------------------------------------------------------------------------------------------------------------
