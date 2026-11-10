# shellcheck shell=bash
# git.sh - git helpers: the default branch, worktrees, kit tidy, and kit rebase (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit tidy (git:tidy) decides whether a worktree's work is finished by asking:
#
#   - the forge (GitHub, GitLab), through modules/bash/forge.sh: is the branch's pull request merged or closed?
#   - when the branch has no pull request, the issue's tracker, through modules/bash/tracker.sh: is the branch's issue
#     closed?

use forge

# git:desks <dir> <depth> prints every desk at or below <dir>, at most <depth> levels down, in path order: a
# directory that holds a .git directory. The walk stops at a desk, so a worktree or a nested repository inside one
# is the desk's own, and it skips hidden directories and symbolic links.
git:desks() {
  local dir=$1 depth=$2 sub
  if [[ -d "${dir}/.git" ]]
  then
    printf '%s\n' "${dir}"
    return 0
  fi
  (( depth > 0 )) || return 0
  for sub in "${dir}"/*/
  do
    sub=${sub%/}
    [[ -d "${sub}" && ! -L "${sub}" ]] || continue
    git:desks "${sub}" $(( depth - 1 ))
  done
}

# git:default-branch [<path>] prints the default branch of the repository at <path>: the primary remote's HEAD, else
# main.
git:default-branch() {
  local branch primary
  primary=$(forge:primary) || return $?
  branch=$(git -C "${1:-.}" symbolic-ref --quiet --short "refs/remotes/${primary}/HEAD" 2>/dev/null || true)
  branch=${branch#"${primary}"/}
  [[ -n "${branch}" ]] || branch=main
  printf '%s\n' "${branch}"
}

# git:main-root [<path>] prints the root of the main checkout, which contains worktrees/. It works from the main
# checkout or from any linked worktree.
git:main-root() {
  dirname "$(git -C "${1:-.}" rev-parse --path-format=absolute --git-common-dir)"
}

# git:linked-worktrees [<path>] prints "<path>\t<detached>\t<branch>" for every linked worktree of the repository at
# <path>, or of the current one, wherever it lies: every one git worktree list names but the main checkout, which it
# names first. <detached> is true or false, and <branch> is empty for a detached checkout.
git:linked-worktrees() {
  local path="" branch="" detached=false line first=true
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
        [[ -n "${path}" && "${first}" == false ]] && printf '%s\t%s\t%s\n' "${path}" "${detached}" "${branch}"
        [[ -n "${path}" ]] && first=false
        path=""
    esac
  done < <(git -C "${1:-.}" worktree list --porcelain; printf '\n')
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

# git:unpushed <branch> prints how many commits of the local <branch> no remote-tracking branch reaches. When some are
# left, it asks the forge for the head commit of the branch's latest pull request, and when this clone has that
# commit, it leaves out what the head reaches too: a squash-merged branch whose remote branch is gone counts 0.
git:unpushed() {
  local branch=$1 count head
  count=$(git rev-list --count "refs/heads/${branch}" --not --remotes) \
    || { error "the commits of ${branch} could not be counted"; return 1; }
  if (( count > 0 ))
  then
    head=$(forge:pr-head-for-branch "${branch}")
    if [[ "${head}" =~ ^[0-9a-f]+$ ]] && git cat-file -e "${head}^{commit}" 2>/dev/null
    then
      count=$(git rev-list --count "refs/heads/${branch}" --not --remotes "${head}") \
        || { error "the commits of ${branch} could not be counted"; return 1; }
    fi
  fi
  printf '%s\n' "${count}"
}

# git:tidy-settle <branch> settles the issues of the branch's latest pull request, as kit settle does, once kit tidy
# has removed its worktree: what the forge does not close itself is closed or put back here, in case nothing else did.
# A failure is said and does not stop the sweep.
git:tidy-settle() {
  local pr
  [[ -n "$1" ]] || return 0
  pr=$(forge:pr-last-for-branch "$1")
  [[ -n "${pr}" ]] || return 0
  use issue
  issue:settle "${pr}" || warn "the issues of change ${pr} could not all be settled; run: kit settle ${pr}"
}

# git:tidy-worktree <path> <branch> <detached> <default-branch> <self-root> <dry-run> decides whether to remove
# one worktree and prints the reason. A worktree whose branch has commits no remote ever had is kept, whatever its
# change's state. It returns 0 when the worktree was removed (or would be, under a dry run) and 1 when it is kept,
# git's refusal to remove it included.
git:tidy-worktree() {
  local path=$1 branch=$2 detached=$3 default_branch=$4 self_root=$5 dry_run=$6 state="" ref primary unpushed
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
    primary=$(forge:primary) || return 1
    if git -C "${path}" merge-base --is-ancestor HEAD "${primary}/${default_branch}" 2>/dev/null
    then
      state=DETACHED
    else
      step "keeping ${path} (detached at a commit ${primary}/${default_branch} does not reach)"
      return 1
    fi
  fi
  case "${state}" in
    (MERGED | CLOSED | DETACHED)
      if [[ -n "${branch}" ]]
      then
        unpushed=$(git:unpushed "${branch}") || return 1
        if (( unpushed > 0 ))
        then
          step "keeping ${path} (${branch}, change ${state}, ${unpushed} commit(s) no remote ever had)"
          return 1
        fi
      fi
      if [[ "${dry_run}" == true ]]
      then
        step "would remove ${path} (${branch:-detached}, ${state})"
      else
        step "removing ${path} (${branch:-detached}, ${state})"
        if ! git worktree remove "${path}"
        then
          step "keeping ${path} (${branch:-detached}, ${state}, git would not remove it)"
          return 1
        fi
        git:tidy-settle "${branch}"
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
#   1. fetches every remote with --prune: the primary, and the one a branch's upstream is on, as in a fork
#   2. removes every clean worktree under worktrees/ whose pull request is merged or closed (or, with no pull
#      request, whose issue is closed), or that is detached at a commit the default branch already has
#   3. prunes git's worktree bookkeeping
#   4. deletes every local branch whose upstream is gone
#
# It never touches:
#
#   - the default branch
#   - a branch checked out in a worktree it keeps
#   - a branch with commits no remote ever had, and the worktree it is checked out in, whatever its change's state;
#     a commit the change's head reaches, as the forge saw it, was pushed
#   - a worktree with uncommitted changes
#   - a detached worktree whose commit nothing else reaches
#   - a worktree git will not remove, as a locked one
#   - the worktree it runs in
#
# --dry-run prints what it would do and changes nothing.
git:tidy() {
  local dry_run=false flag repo default_branch self_root found=false path branch detached remote remotes
  local removed_worktrees=0 removed_branches=0 checked_out unpushed
  local -A freed=()
  for flag in "$@"
  do
    case "${flag}" in
      (--dry-run) dry_run=true
    esac
  done
  forge:require || return 1
  repo=$(forge:repo) || return 1
  if [[ "${dry_run}" == true ]]
  then
    section "kit tidy, dry run: nothing is changed"
  else
    section "kit tidy: sweeping ${repo} back to pristine"
  fi

  default_branch=$(git:default-branch) || return $?
  self_root=$(git rev-parse --show-toplevel)
  remotes=$(forge:remotes) || return $?
  while IFS= read -r remote
  do
    [[ -n "${remote}" ]] || continue
    if [[ "${dry_run}" == true ]]
    then
      step "would run: git fetch --prune ${remote}"
    else
      step "git fetch --prune ${remote}"
      git fetch --quiet --prune --end-of-options "${remote}"
    fi
  done <<<"${remotes}"

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
    unpushed=$(git:unpushed "${branch}") || continue
    if (( unpushed > 0 ))
    then
      step "keeping ${branch} (${unpushed} commit(s) no remote ever had)"
      continue
    fi
    if [[ "${dry_run}" == true ]]
    then
      step "would delete ${branch}"
    else
      step "deleting ${branch}"
      if ! git branch -D "${branch}" >/dev/null
      then
        warn "${branch} could not be deleted; see above"
        continue
      fi
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

# git:pushes-to <branch> [<path>] prints the remote that <branch>, in the repository at <path>, pushes to. When the
# branch's upstream is the branch of the same name, it prints branch.<branch>.remote and returns 0. Otherwise, with no
# upstream or one of another name, as a branch started from origin/main has, it prints origin and returns 1.
git:pushes-to() {
  local branch=$1 path=${2:-.} remote merge
  remote=$(git -C "${path}" config --get "branch.${branch}.remote" 2>/dev/null || true)
  merge=$(git -C "${path}" config --get "branch.${branch}.merge" 2>/dev/null || true)
  if [[ -n "${remote}" && "${merge}" == "refs/heads/${branch}" ]]
  then
    printf '%s\n' "${remote}"
    return 0
  fi
  printf 'origin\n'
  return 1
}

# git:push <path> <branch> [<option>…] pushes refs/heads/<branch> of the checkout at <path>, quietly and with the
# options, to the same name on the remote git:pushes-to prints, and sets that as the upstream when git:pushes-to
# returned 1. It names the refspec, so push.default never decides where the branch goes.
git:push() {
  local path=$1 branch=$2 remote upstream=()
  shift 2
  remote=$(git:pushes-to "${branch}" "${path}") || upstream=(--set-upstream)
  git -C "${path}" push --quiet "${upstream[@]}" "$@" --end-of-options "${remote}" \
    "refs/heads/${branch}:refs/heads/${branch}"
}

# git:rebase [<path>] rebases the branch checked out at <path> (default: the current checkout) onto the default
# branch at the primary remote, then pushes it to its own name on the remote git:pushes-to prints. It refuses:
#
#   - a checkout with no branch checked out
#   - the root checkout (the desk, which belongs to the human), whatever branch it has
#   - the default branch, in any checkout
#   - a worktree with uncommitted changes
#   - with require_signing true, a git with no signing key
#
# When that remote's branch has commits this checkout lacks, another session's, it takes them first: it rebases onto
# that tip, then onto the default branch. With require_signing true, every commit the rebase rewrites is signed. The
# push forces only over the tip it read, so a commit pushed after the fetch makes the push fail rather than vanish. On
# a conflict it aborts the rebase and leaves the worktree as it was.
git:rebase() {
  local worktree=${1:-.} branch default_branch primary remote before theirs target targets=() sign=()
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
  if [[ "$(settings:value require_signing)" == true ]]
  then
    guard:require signing-key || return 1
    sign=(--gpg-sign)
  fi
  primary=$(forge:primary) || return $?
  remote=$(git:pushes-to "${branch}" "${worktree}") || true
  section "kit rebase ${branch} onto ${primary}/${default_branch}"
  before=$(git -C "${worktree}" rev-parse HEAD)
  git -C "${worktree}" fetch --quiet --prune --end-of-options "${primary}"
  [[ "${remote}" == "${primary}" ]] || git -C "${worktree}" fetch --quiet --prune --end-of-options "${remote}"
  theirs=$(git -C "${worktree}" rev-parse --quiet --verify "refs/remotes/${remote}/${branch}^{commit}" || true)
  if [[ -n "${theirs}" ]] && ! git -C "${worktree}" merge-base --is-ancestor "${theirs}" HEAD
  then
    step "taking $(git -C "${worktree}" rev-list --count "HEAD..${theirs}") commit(s)" \
      "${remote}/${branch} has and this checkout lacks"
    targets+=("${remote}/${branch}")
  fi
  targets+=("${primary}/${default_branch}")
  for target in "${targets[@]}"
  do
    if ! git -C "${worktree}" rebase --quiet "${sign[@]}" "${target}"
    then
      git -C "${worktree}" rebase --abort 2>/dev/null || true
      git -C "${worktree}" reset --quiet --hard "${before}"
      error "${branch} conflicts with ${target}; the rebase was aborted and the worktree is as it was"
      return 1
    fi
  done
  if ! git:push "${worktree}" "${branch}" "--force-with-lease=refs/heads/${branch}:${theirs}"
  then
    error "${branch} is rebased here and was not pushed; see above, then run: kit rebase"
    return 1
  fi
  ok "${branch} is rebased onto ${primary}/${default_branch} and pushed."
}

# ----------------------------------------------------------------------------------------------------------------------
