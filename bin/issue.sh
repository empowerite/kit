# shellcheck shell=bash
# issue.sh - the issue on-ramp: a linked branch, a worktree, a start commit, and a draft change (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# issue:start [<reference>…] is the one command the loop needs and no vendor provides. A reference names its
# tracker, as bin/tracker.sh describes: `7` or `#7` for the forge's own issues, `AB#4521` for a declared one. The
# first is the primary issue. One of the forge's own is linked to the branch the forge makes, which closes it on
# merge; any other, and every further reference, a secondary, is closed by a `Closes <reference>` line in the
# change's body. Every listed issue is assigned to the caller, as each tracker knows the caller. It never switches
# the checkout it runs from: the branch is checked out in worktrees/<key>-<slug> under the main checkout, the key
# being the reference as a branch name opens with it, 7 or ab4521, and an existing branch, worktree, or change is
# reused, so a re-run reconciles.
#
# With no argument and a terminal on stdin it asks for a title and creates the issue; without a terminal the
# usage error stands, so an agent never hangs on a prompt. Uncommitted work in the root checkout, the human's
# desk, is swept into the new worktree, uncommitted, and the desk is left clean, and fast-forwarded when it is on
# the default branch; inside any other checkout, dirty files are that issue's own and stay. A sweep that cannot
# apply leaves the stash in place and stops.
#
# Every call to the forge goes through bin/forge.sh, and every call about an issue through bin/tracker.sh. Each
# step below prints its one result on stdout and talks on stderr, so issue:start reads them in order.

use git forge tracker

# issue:create asks for a title at the terminal and prints the reference of the issue the forge's tracker creates.
issue:create() {
  local title number
  read -r -p "Issue title: " title
  if [[ -z "${title}" ]]
  then
    error "an issue title is required"
    return 2
  fi
  number=$(tracker:at '#' create "${title}") || return 1
  ok "Created #${number}: ${title}"
  printf '#%s\n' "${number}"
}

# issue:assign <reference>… adds the caller, as each issue's tracker knows the caller, to each issue's assignees,
# warning where someone else is there already.
issue:assign() {
  local ref prefix me assignees
  local -A me_in=()
  for ref in "$@"
  do
    prefix=$(tracker:parse "${ref}" | cut -f 1)
    [[ -n "${me_in[${prefix}]:-}" ]] || me_in[${prefix}]=$(tracker:at "${prefix}" me) || return 1
    me=${me_in[${prefix}]}
    assignees=$(tracker:of "${ref}" assignees)
    [[ " ${assignees} " == *" ${me} "* ]] && continue
    [[ -z "${assignees}" ]] \
      || warn "${ref} is already assigned to ${assignees}; adding you as well. Coordinate before continuing."
    tracker:of "${ref}" assign "${me}" || warn "${ref} could not be assigned to ${me}; assign it by hand."
  done
}

# issue:branch <reference> <default-branch> <title> prints the branch made for the issue, making one if none
# exists: linked by the forge for one of its own issues, and named <key>-<slug of the title> from the default
# branch for any other. An existing one is found by an open change's head, then by the remote ref, so a second run
# never forks a duplicate.
issue:branch() {
  local ref=$1 default_branch=$2 title=$3 key branch slug
  key=$(tracker:key "${ref}") || return 2
  branch=$(forge:branch-for-issue "${key}")
  [[ -n "${branch}" ]] \
    || branch=$(git ls-remote --heads origin "${key}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
  if [[ -z "${branch}" ]] && tracker:is-forge "${ref}"
  then
    step "Linking a branch to ${ref}"
    forge:link-branch "${key}" "${default_branch}" || return 1
    git fetch --quiet origin
    branch=$(git ls-remote --heads origin "${key}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
    if [[ -z "${branch}" ]]
    then
      error "the branch the forge made for ${ref} could not be found on origin; link one by hand and re-run"
      return 1
    fi
  elif [[ -z "${branch}" ]]
  then
    slug=$(tr '[:upper:]' '[:lower:]' <<<"${title}" | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' | cut -c 1-60)
    branch="${key}-${slug%-}"
    step "Making ${branch} for ${ref} from ${default_branch}"
    git push --quiet origin "refs/remotes/origin/${default_branch}:refs/heads/${branch}" || return 1
    git fetch --quiet origin
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

# issue:start-commit <worktree> <default-branch> <reference> <title> gives the branch its first commit when it has
# none: a forge cannot open a change with nothing to compare. --only keeps the commit empty, so work already
# staged in a reused worktree stays in the index. The subject follows the loop's shape, the reference first.
issue:start-commit() {
  local wt=$1 default_branch=$2 ref=$3 title=$4
  if [[ -z "$(git -C "${wt}" rev-list "origin/${default_branch}..HEAD" 2>/dev/null)" ]]
  then
    step "Adding the start commit"
    git -C "${wt}" commit --quiet --allow-empty --only --message "${ref} start: ${title}"
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

# issue:pull-request <branch> <default-branch> <title> prints the number of the open change for the branch,
# opening a draft if there is none.
issue:pull-request() {
  local branch=$1 default_branch=$2 title=$3 pr
  pr=$(forge:pr-for-branch "${branch}")
  if [[ -z "${pr}" ]]
  then
    step "Opening a draft change"
    pr=$(forge:pr-create "${branch}" "${default_branch}" "${title}") || return 1
  fi
  printf '%s\n' "${pr}"
}

# issue:link-in-body <change> <reference>… adds a `Closes <reference>` line to the change's body for each reference
# not there: a secondary issue, or a primary the forge cannot link by itself.
issue:link-in-body() {
  local pr=$1 ref escaped body changed=false
  shift
  body=$(forge:pr-body "${pr}")
  for ref in "$@"
  do
    escaped=$(ere:quote "${ref}")
    if ! grep --quiet --ignore-case --extended-regexp \
      "(close[sd]?|fix(e[sd])?|resolve[sd]?):? ${escaped}([^0-9]|\$)" <<<"${body}"
    then
      body="${body}"$'\n'"Closes ${ref}"
      changed=true
    fi
  done
  if [[ "${changed}" == true ]]
  then
    step "Linking the issues in the change's body"
    forge:pr-set-body "${pr}" "${body}"
  fi
}

# issue:start [<reference>…] is the on-ramp the file header describes. A desk without a signing key is refused
# where the require_signing setting is true, before anything is written.
issue:start() {
  local n primary title default_branch main_root desk desk_branch branch wt stash="" pr created sigil noun
  local -a refs=() in_body=()
  forge:require || return 1
  if [[ "$(release:setting require_signing)" == true ]]
  then
    guard:require signing-key || return 1
  fi
  if (( $# == 0 ))
  then
    if [[ ! -t 0 ]]
    then
      error "usage: kit issue <reference> [<reference>…]"
      return 2
    fi
    created=$(issue:create) || return
    set -- "${created}"
  fi
  for n in "$@"
  do
    n=$(tracker:normal "${n}") || return 2
    refs+=("${n}")
  done
  set -- "${refs[@]}"
  primary=$1
  tracker:is-forge "${primary}" || in_body+=("${primary}")
  in_body+=("${@:2}")

  git fetch --quiet --prune origin
  default_branch=$(git:default-branch)
  main_root=$(git:main-root)
  desk=$(git rev-parse --show-toplevel)
  desk_branch=$(git -C "${desk}" rev-parse --abbrev-ref HEAD)
  title=$(tracker:of "${primary}" title) || return 1
  issue:assign "$@" || return
  section "Starting ${primary}: ${title}"

  branch=$(issue:branch "${primary}" "${default_branch}" "${title}") || return
  if [[ "${desk}" == "${main_root}" ]] && ! git:clean "${desk}"
  then
    step "Sweeping the uncommitted work on ${desk_branch} into the worktree"
    stash=$(issue:sweep-push "${desk}" "kit issue ${primary} → ${branch}") || return
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
  (( ${#in_body[@]} > 0 )) && issue:link-in-body "${pr}" "${in_body[@]}"

  IFS=$'\t' read -r sigil noun < <(forge:noun)
  ok "Draft ${noun} ${sigil}${pr} on ${branch}, primary ${primary}${2:+, secondaries: ${*:2}}"
  ok "Worktree ready: ${wt}"
  step "cd '${wt}'    # do all work there; this checkout stays on ${desk_branch}"
}

# ----------------------------------------------------------------------------------------------------------------------
