# shellcheck shell=bash
# issue.sh - the issue on-ramp: a linked branch, a worktree, a start commit, and a draft pull request (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# issue:start [<number>…] is the one command the loop needs and no vendor provides. The first number is the
# primary issue, linked to the branch the forge makes, which closes it on merge; any further number is a
# secondary, closed by a `Closes #N` line in the pull request body. Every listed issue is assigned to the caller.
# It never switches the checkout it runs from: the branch is checked out in worktrees/<N>-<slug> under the main
# checkout, and an existing branch, worktree, or pull request is reused, so a re-run reconciles.
#
# With no argument and a terminal on stdin it asks for a title and creates the issue; without a terminal the
# usage error stands, so an agent never hangs on a prompt. Uncommitted work in the root checkout, the human's
# desk, is swept into the new worktree, uncommitted, and the desk is left clean, and fast-forwarded when it is on
# the default branch; inside any other checkout, dirty files are that issue's own and stay. A sweep that cannot
# apply leaves the stash in place and stops.
#
# Every call to the forge goes through bin/forge.sh. Each step below prints its one result on stdout and talks
# on stderr, so issue:start reads them in order.

use git forge

# issue:create asks for a title at the terminal and prints the number of the issue the forge creates.
issue:create() {
  local title number
  read -r -p "Issue title: " title
  if [[ -z "${title}" ]]
  then
    error "an issue title is required"
    return 2
  fi
  number=$(forge:issue-create "${title}") || return 1
  ok "Created #${number}: ${title}"
  printf '%s\n' "${number}"
}

# issue:assign <me> <number>… adds <me> to each issue's assignees, warning where someone else is there already.
issue:assign() {
  local me=$1 n assignees
  shift
  for n in "$@"
  do
    assignees=$(forge:issue-assignees "${n}")
    [[ " ${assignees} " == *" ${me} "* ]] && continue
    [[ -z "${assignees}" ]] \
      || warn "#${n} is already assigned to ${assignees}; adding you as well. Coordinate before continuing."
    forge:issue-assign "${n}" "${me}" || warn "#${n} could not be assigned to ${me}; assign it by hand."
  done
}

# issue:branch <number> <default-branch> prints the branch linked to the issue, making and linking one if none
# exists. An existing one is found by an open pull request's head, then by the remote ref, so a second run never
# forks a duplicate.
issue:branch() {
  local number=$1 default_branch=$2 branch
  branch=$(forge:branch-for-issue "${number}")
  [[ -n "${branch}" ]] \
    || branch=$(git ls-remote --heads origin "${number}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
  if [[ -z "${branch}" ]]
  then
    step "Linking a branch to #${number}"
    forge:link-branch "${number}" "${default_branch}" || return 1
    git fetch --quiet origin
    branch=$(git ls-remote --heads origin "${number}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
    if [[ -z "${branch}" ]]
    then
      error "the branch the forge made for #${number} could not be found on origin; link one by hand and re-run"
      return 1
    fi
  fi
  printf '%s\n' "${branch}"
}

# issue:worktree <main-root> <branch> prints the worktree the branch is checked out in, making it if there is none.
issue:worktree() {
  local main_root=$1 branch=$2 wt
  wt=$(git worktree list --porcelain 2>/dev/null \
    | awk -v b="branch refs/heads/${branch}" '/^worktree / { w = substr($0, 10) } $0 == b { print w; exit }')
  if [[ -n "${wt}" ]]
  then
    step "Reusing worktree ${wt}"
  else
    wt="${main_root}/worktrees/${branch}"
    if [[ -e "${wt}" ]]
    then
      error "${wt} exists but is not a worktree of ${branch}; remove it (or run: kit tidy) and re-run."
      return 2
    fi
    step "Creating worktree ${wt}"
    mkdir -p "${main_root}/worktrees"
    if git show-ref --verify --quiet "refs/heads/${branch}"
    then
      git worktree add --quiet "${wt}" "${branch}"
    else
      git worktree add --quiet "${wt}" -b "${branch}" "origin/${branch}"
    fi
  fi
  printf '%s\n' "${wt}"
}

# issue:start-commit <worktree> <default-branch> <number> <title> gives the branch its first commit when it has
# none: a forge cannot open a pull request with nothing to compare. --only keeps the commit empty, so work
# already staged in a reused worktree stays in the index. The subject follows the loop's shape.
issue:start-commit() {
  local wt=$1 default_branch=$2 number=$3 title=$4
  if [[ -z "$(git -C "${wt}" rev-list "origin/${default_branch}..HEAD" 2>/dev/null)" ]]
  then
    step "Adding the start commit"
    git -C "${wt}" commit --quiet --allow-empty --only --message "#${number} start: ${title}"
  fi
}

# issue:sweep-push <root> <message> stashes everything uncommitted at <root>, untracked files included, and
# prints the stash's commit, which stays valid however the stash list shifts.
issue:sweep-push() {
  local root=$1 message=$2
  if ! git -C "${root}" stash push --include-untracked --message "${message}" >/dev/null
  then
    error "the uncommitted work at ${root} could not be stashed; resolve it by hand and re-run."
    return 1
  fi
  git -C "${root}" rev-parse 'stash@{0}'
}

# issue:sweep-apply <root> <worktree> <stash> <default-branch> lands the stash in the worktree, uncommitted,
# drops it, and fast-forwards the now-clean desk when it is on the default branch. On a conflict the stash is
# kept and the caller stops.
issue:sweep-apply() {
  local root=$1 wt=$2 stash=$3 default_branch=$4 slot
  if ! git -C "${wt}" stash apply "${stash}" >/dev/null 2>&1
  then
    error "the swept work conflicts in ${wt}. It is kept in the stash: git stash apply ${stash}"
    return 1
  fi
  slot=$(git -C "${root}" stash list --format='%gd %H' | awk -v h="${stash}" '$2 == h { print $1; exit }')
  [[ -n "${slot}" ]] && git -C "${root}" stash drop "${slot}" >/dev/null
  ok "Swept the work into ${wt}, uncommitted; commit it there with a real message."
  [[ "$(git -C "${root}" rev-parse --abbrev-ref HEAD)" == "${default_branch}" ]] || return 0
  git -C "${root}" merge --ff-only "origin/${default_branch}" >/dev/null 2>&1 \
    || warn "${default_branch} could not fast-forward to origin; it is left as it was."
}

# issue:pull-request <branch> <default-branch> <title> prints the number of the open pull request for the
# branch, opening a draft if there is none.
issue:pull-request() {
  local branch=$1 default_branch=$2 title=$3 pr
  pr=$(forge:pr-for-branch "${branch}")
  if [[ -z "${pr}" ]]
  then
    step "Opening a draft pull request"
    pr=$(forge:pr-create "${branch}" "${default_branch}" "${title}") || return 1
  fi
  printf '%s\n' "${pr}"
}

# issue:link-secondaries <pr> <number>… adds a `Closes #N` line to the pull request body for each number not
# there.
issue:link-secondaries() {
  local pr=$1 n body changed=false
  shift
  body=$(forge:pr-body "${pr}")
  for n in "$@"
  do
    if ! grep --quiet --ignore-case --extended-regexp \
      "(close[sd]?|fix(e[sd])?|resolve[sd]?):? #${n}([^0-9]|\$)" <<<"${body}"
    then
      body="${body}"$'\n'"Closes #${n}"
      changed=true
    fi
  done
  if [[ "${changed}" == true ]]
  then
    step "Linking the secondary issues in the pull request body"
    forge:pr-set-body "${pr}" "${body}"
  fi
}

# issue:start [<number>…] is the on-ramp the file header describes. A desk without a signing key is refused
# where the require_signing setting is true, before anything is written.
issue:start() {
  local me n primary title default_branch main_root desk desk_branch branch wt stash="" pr created
  forge:require || return 1
  me=$(forge:me) || return 1
  if [[ "$(release:setting require_signing)" == true ]]
  then
    guard:require signing-key || return 1
  fi
  if (( $# == 0 ))
  then
    if [[ ! -t 0 ]]
    then
      error "usage: kit issue <issue-number> [<issue-number>…]"
      return 2
    fi
    created=$(issue:create) || return
    set -- "${created}"
  fi
  for n in "$@"
  do
    if [[ ! "${n}" =~ ^[0-9]+$ ]]
    then
      error "not an issue number: '${n}'"
      return 2
    fi
  done
  primary=$1

  git fetch --quiet --prune origin
  default_branch=$(git:default-branch)
  main_root=$(git:main-root)
  desk=$(git rev-parse --show-toplevel)
  desk_branch=$(git -C "${desk}" rev-parse --abbrev-ref HEAD)
  title=$(forge:issue-title "${primary}") || return 1
  issue:assign "${me}" "$@"
  section "Starting #${primary}: ${title}"

  branch=$(issue:branch "${primary}" "${default_branch}") || return
  if [[ "${desk}" == "${main_root}" ]] && ! git:clean "${desk}"
  then
    step "Sweeping the uncommitted work on ${desk_branch} into the worktree"
    stash=$(issue:sweep-push "${desk}" "kit issue #${primary} → ${branch}") || return
  fi
  wt=$(issue:worktree "${main_root}" "${branch}") || return
  git -C "${main_root}" check-ignore --quiet worktrees 2>/dev/null \
    || warn "worktrees/ is not ignored; add /worktrees/ to .gitignore so a worktree never appears in a commit"
  issue:start-commit "${wt}" "${default_branch}" "${primary}" "${title}"
  git -C "${wt}" push --quiet --set-upstream origin "${branch}"
  if [[ -n "${stash}" ]]
  then
    issue:sweep-apply "${desk}" "${wt}" "${stash}" "${default_branch}" || return
  fi
  pr=$(issue:pull-request "${branch}" "${default_branch}" "${title}") || return
  (( $# > 1 )) && issue:link-secondaries "${pr}" "${@:2}"

  ok "Draft pull request #${pr} on ${branch}, primary #${primary}${2:+, secondaries: ${*:2}}"
  ok "Worktree ready: ${wt}"
  step "cd '${wt}'    # do all work there; this checkout stays on ${desk_branch}"
}

# ----------------------------------------------------------------------------------------------------------------------
