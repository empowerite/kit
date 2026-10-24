# shellcheck shell=bash
# git.sh - the picture of the worktrees, and the sweep that returns a checkout to pristine (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# git:status [--no-fetch] renders the main checkout on one line and one block per worktree under worktrees/
# with its branch, pull request, and dirty or clean state, then the local branches whose upstream is gone. It
# changes nothing.
#
# git:tidy [--dry-run] removes every worktree under worktrees/ whose pull request is merged or closed, prunes
# the worktree bookkeeping, and deletes every local branch whose upstream is gone. It never touches the default
# branch, a branch checked out in a live worktree, a dirty worktree, or the worktree it runs in. --dry-run
# prints what a sweep would do and changes nothing.

# git:pr-summary <branch> prints "<number>\t<Draft|Ready>" for the open pull request on <branch>, or nothing.
git:pr-summary() {
  gh pr list --head "$1" --state open --json number,isDraft \
    --jq '.[0] | select(.number) | "\(.number)\t\(if .isDraft then "Draft" else "Ready" end)"' 2>/dev/null || true
}

# git:gone-branches prints every local branch whose upstream no longer exists.
git:gone-branches() {
  git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads/ | awk '$2=="[gone]"{print $1}'
}

# git:each-worktree <callback> calls <callback> <path> <branch> <detached> for every linked worktree under
# worktrees/. <branch> is empty and <detached> is true for a detached checkout.
git:each-worktree() {
  local callback=$1 worktree_root path="" branch="" detached=false line
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
        [[ -n "${path}" ]] || continue
        case "${path}" in
          ("${worktree_root}"/*)
            "${callback}" "${path}" "${branch}" "${detached}"
        esac
        path=""
    esac
  done < <(git worktree list --porcelain; printf '\n')
}

# git:status-block <path> <branch> <detached> renders one worktree. git:status sets found, main_root, and
# default_branch before calling it through git:each-worktree.
git:status-block() {
  local path=$1 branch=$2 rel pr ahead_behind porcelain
  found=true
  rel=${path#"${main_root}"/}
  step "${rel}"
  info "  branch  ${branch:-(detached)}"
  if [[ -n "${branch}" ]]
  then
    pr=$(git:pr-summary "${branch}")
    ahead_behind=$(git rev-list --left-right --count "${branch}...${default_branch}" 2>/dev/null || printf '0\t0')
    if [[ -n "${pr}" ]]
    then
      info "  pr      #${pr%%$'\t'*} ${pr##*$'\t'} · ↑${ahead_behind%%$'\t'*} ↓${ahead_behind##*$'\t'} vs ${default_branch}"
    else
      info "  pr      (none open) · ↑${ahead_behind%%$'\t'*} ↓${ahead_behind##*$'\t'} vs ${default_branch}"
    fi
  else
    info "  pr      (no branch)"
  fi
  porcelain=$(git -C "${path}" status --porcelain 2>/dev/null || true)
  if [[ -z "${porcelain}" ]]
  then
    info "  status  clean"
  else
    info "  status  $(grep --count . <<<"${porcelain}") file(s) modified"
  fi
}

# git:tidy-worktree <path> <branch> <detached> decides one worktree. git:tidy sets found, self_root, dry_run,
# removed_worktrees, and freed before calling it through git:each-worktree.
git:tidy-worktree() {
  local path=$1 branch=$2 detached=$3 state="" number
  found=true
  if [[ "${path}" == "${self_root}" ]]
  then
    step "keeping ${path} (this command runs in it)"
    return 0
  fi
  if [[ -n "${branch}" ]]
  then
    state=$(gh pr list --head "${branch}" --state all --json state --jq '.[0].state // empty' 2>/dev/null || true)
    if [[ -z "${state}" && "${branch}" =~ ^([0-9]+)- ]]
    then
      number=${BASH_REMATCH[1]}
      [[ "$(gh issue view "${number}" --json state --jq .state 2>/dev/null)" == CLOSED ]] && state=CLOSED
    fi
  elif [[ "${detached}" == true && -z "$(git -C "${path}" status --porcelain 2>/dev/null)" ]]
  then
    state=DETACHED
  fi
  case "${state}" in
    (MERGED | CLOSED | DETACHED)
      if [[ -n "$(git -C "${path}" status --porcelain 2>/dev/null)" ]]
      then
        step "keeping ${path} (${state}, but it has uncommitted changes)"
        return 0
      fi
      if [[ "${dry_run}" == true ]]
      then
        step "would remove ${path} (${branch:-detached}, ${state})"
      else
        step "removing ${path} (${branch:-detached}, ${state})"
        git worktree remove "${path}"
      fi
      [[ -n "${branch}" ]] && freed[${branch}]=1
      (( ++removed_worktrees ))
      ;;
    (*)
      step "keeping ${path} (${branch:-detached}, pull request ${state:-open or unknown})"
  esac
  return 0
}

git:status() {
  local no_fetch=false arg
  for arg in "$@"
  do
    case "${arg}" in
      (--no-fetch)
        no_fetch=true
        ;;
      (*)
        error "unknown flag '${arg}' (only --no-fetch is recognized)"
        return 2
    esac
  done
  if [[ "${no_fetch}" != true ]]
  then
    git fetch --quiet --prune origin || warn "git fetch --prune failed; ahead and behind counts may be stale"
  fi

  local main_branch main_dirty counts
  main_root=$(git:main-root)
  default_branch=$(git:default-branch)
  main_branch=$(git -C "${main_root}" rev-parse --abbrev-ref HEAD 2>/dev/null || printf '?')
  main_dirty=clean
  [[ -z "$(git -C "${main_root}" status --porcelain 2>/dev/null)" ]] || main_dirty=dirty
  counts=$(git -C "${main_root}" rev-list --left-right --count "HEAD...origin/${default_branch}" 2>/dev/null || printf '0\t0')
  section "${main_root##*/}  (${main_branch}, ${main_dirty}, ↑${counts%%$'\t'*} ↓${counts##*$'\t'} vs origin/${default_branch})"

  section "worktrees/"
  found=false
  git:each-worktree git:status-block
  [[ "${found}" == true ]] || ok "none"

  section "branches whose upstream is gone (tidy removes these)"
  local gone
  gone=$(git:gone-branches)
  if [[ -n "${gone}" ]]
  then
    while IFS= read -r branch
    do
      info "${branch}"
    done <<<"${gone}"
  else
    ok "none"
  fi
}

git:tidy() {
  local dry_run=false flag
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
  guard:require gh || return 1

  local repo
  repo=$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null || true)
  if [[ -z "${repo}" ]]
  then
    error "this checkout does not resolve to a GitHub repository (run: gh repo view), so nothing is swept."
    return 1
  fi
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
  removed_worktrees=0
  found=false
  declare -A freed=()
  git:each-worktree git:tidy-worktree
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
  local removed_branches=0 checked_out branch
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

# ----------------------------------------------------------------------------------------------------------------------
