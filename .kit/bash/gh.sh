# shellcheck shell=bash
# gh.sh - the issue on-ramp: a linked branch, a worktree, a start commit, and a draft pull request (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# gh:issue [<number>…] is the one command the loop needs and no vendor provides. The first number is the
# primary issue, linked to the branch under Development by `gh issue develop`, which closes it on merge. Any
# further number is a secondary, closed by a `Closes #N` line in the pull request body. Every listed issue is
# assigned to the caller.
#
# It never switches the checkout it is run from. The branch is checked out in worktrees/<N>-<slug> under the
# main checkout's root, and an existing branch, worktree, or pull request is reused, so a re-run reconciles
# rather than duplicates.
#
# With no argument and a terminal on stdin it asks for a title, creates the issue, and continues. Without a
# terminal the usage error stands, so an agent never hangs on a prompt.
#
# If the invoking checkout is on the default branch and carries uncommitted changes, they are swept into the
# new worktree as uncommitted changes and the invoking checkout is left clean, then fast-forwarded to origin.
# The sweep runs only from the default-branch checkout: inside another worktree, dirty files are that issue's
# own work and are never moved. A sweep that cannot apply leaves the stash in place and stops, so nothing is
# lost.

gh:issue() {
  if (( $# == 0 ))
  then
    if [[ ! -t 0 ]]
    then
      error "usage: kit issue <issue-number> [<issue-number>…]"
      return 2
    fi
    guard:require gh || return 1
    local new_title new_number
    read -r -p "Issue title: " new_title
    if [[ -z "${new_title}" ]]
    then
      error "an issue title is required"
      return 2
    fi
    new_number=$(gh issue create --title "${new_title}" --body "" 2>/dev/null | sed -n 's@.*/issues/@@p')
    if [[ ! "${new_number}" =~ ^[0-9]+$ ]]
    then
      error "run: gh auth status (the issue could not be created)"
      return 1
    fi
    ok "Created #${new_number}: ${new_title}"
    set -- "${new_number}"
  fi

  local n
  for n in "$@"
  do
    if [[ ! "${n}" =~ ^[0-9]+$ ]]
    then
      error "not an issue number: '${n}'"
      return 2
    fi
  done
  local primary=$1

  guard:require gh || return 1
  git fetch --quiet --prune origin

  local default_branch main_root invoking_root invoking_branch
  default_branch=$(git:default-branch)
  main_root=$(git:main-root)
  invoking_root=$(git rev-parse --show-toplevel)
  invoking_branch=$(git -C "${invoking_root}" rev-parse --abbrev-ref HEAD)

  local do_sweep=false stash_ref=""
  if [[ "${invoking_branch}" == "${default_branch}" && -n "$(git -C "${invoking_root}" status --porcelain)" ]]
  then
    do_sweep=true
  fi

  local title
  title=$(gh issue view "${primary}" --json title --jq '.title')
  if [[ -z "${title}" ]]
  then
    error "run: gh auth status (the title of #${primary} could not be read)"
    return 1
  fi

  local me assignees
  me=$(gh api user --jq '.login')
  if [[ -z "${me}" ]]
  then
    error "run: gh auth status (the current user could not be read)"
    return 1
  fi
  for n in "$@"
  do
    assignees=$(gh issue view "${n}" --json assignees --jq '[.assignees[].login] | join(" ")')
    [[ " ${assignees} " == *" ${me} "* ]] && continue
    if [[ -n "${assignees}" ]]
    then
      warn "#${n} is already assigned to ${assignees}; adding you as well. Coordinate before continuing."
    fi
    if ! gh issue edit "${n}" --add-assignee "${me}" >/dev/null 2>&1
    then
      warn "#${n} could not be assigned to ${me}; assign it by hand."
    fi
  done

  section "Starting #${primary}: ${title}"

  # The branch gh made for the issue, found by an open pull request's head, then by the remote ref. Any hit is
  # reused, so a second run never forks a <N>-<slug>-1 duplicate.
  local branch
  branch=$(gh pr list --state open --json headRefName \
    --jq "[.[] | select(.headRefName | test(\"^${primary}-\"))] | .[0].headRefName // empty" 2>/dev/null || true)
  [[ -n "${branch}" ]] || branch=$(git ls-remote --heads origin "${primary}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
  if [[ -z "${branch}" ]]
  then
    step "Linking a branch to #${primary} (gh issue develop)"
    gh issue develop "${primary}" --base "${default_branch}" >/dev/null
    git fetch --quiet origin
    branch=$(git ls-remote --heads origin "${primary}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
    if [[ -z "${branch}" ]]
    then
      error "the branch gh made for #${primary} could not be found; run: gh issue develop ${primary} --list"
      return 1
    fi
  fi

  if [[ "${do_sweep}" == true ]]
  then
    step "Sweeping the uncommitted work on ${default_branch} into the worktree"
    if ! git -C "${invoking_root}" stash push --include-untracked --message "kit issue #${primary} → ${branch}" >/dev/null
    then
      error "the uncommitted work on ${default_branch} could not be stashed; resolve it by hand and re-run."
      return 1
    fi
    stash_ref=$(git -C "${invoking_root}" rev-parse 'stash@{0}')
  fi

  local wt
  wt=$(git worktree list --porcelain 2>/dev/null \
    | awk -v b="branch refs/heads/${branch}" '/^worktree /{w=substr($0,10)} $0==b{print w; exit}')
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

  # shellcheck disable=SC2154  # kit_bash_dir is set by the loader, 1st.sh, which sources this file
  if [[ "$(git config core.hooksPath 2>/dev/null)" != "$(cd "${kit_bash_dir}/.." && pwd)/hooks" ]]
  then
    warn "run: kit setup (core.hooksPath does not name this release's hooks, so the desk guard is off in this clone)"
  fi

  # GitHub cannot open a pull request on a branch with nothing to compare, so the first commit is an empty one.
  if [[ -z "$(git -C "${wt}" rev-list "origin/${default_branch}..HEAD" 2>/dev/null)" ]]
  then
    step "Adding the start commit"
    git -C "${wt}" commit --quiet --allow-empty --message "start: #${primary} ${title}"
  fi
  git -C "${wt}" push --quiet --set-upstream origin "${branch}"

  if [[ -n "${stash_ref}" ]]
  then
    if git -C "${wt}" stash apply "${stash_ref}" >/dev/null 2>&1
    then
      local slot
      slot=$(git -C "${invoking_root}" stash list --format='%gd %H' | awk -v h="${stash_ref}" '$2==h{print $1; exit}')
      [[ -n "${slot}" ]] && git -C "${invoking_root}" stash drop "${slot}" >/dev/null
      ok "Swept the work into ${wt}, uncommitted; commit it there with a real message."
      if ! git -C "${invoking_root}" merge --ff-only "origin/${default_branch}" >/dev/null 2>&1
      then
        warn "${default_branch} could not fast-forward to origin; it is left as it was."
      fi
    else
      error "the swept work conflicts in ${wt}. It is kept in the stash: git stash apply ${stash_ref}"
      return 1
    fi
  fi

  local pr
  pr=$(gh pr list --head "${branch}" --state open --json number --jq '.[0].number // empty')
  if [[ -z "${pr}" ]]
  then
    step "Opening a draft pull request"
    gh pr create --draft --base "${default_branch}" --head "${branch}" --title "${title}" --body "" >/dev/null
    pr=$(gh pr list --head "${branch}" --state open --json number --jq '.[0].number')
  fi

  if (( $# > 1 ))
  then
    local body changed=false
    body=$(gh pr view "${pr}" --json body --jq '.body // ""')
    for n in "${@:2}"
    do
      if ! grep --quiet --ignore-case --extended-regexp "(close[sd]?|fix(e[sd])?|resolve[sd]?):? #${n}([^0-9]|\$)" <<<"${body}"
      then
        body="${body}"$'\n'"Closes #${n}"
        changed=true
      fi
    done
    if [[ "${changed}" == true ]]
    then
      step "Linking the secondary issues in the pull request body"
      gh pr edit "${pr}" --body "${body}" >/dev/null
    fi
  fi

  ok "Draft pull request #${pr} on ${branch}, primary #${primary}${2:+, secondaries: ${*:2}}"
  ok "Worktree ready: ${wt}"
  step "cd '${wt}'    # do all work there; this checkout stays on ${default_branch}"
}

# ----------------------------------------------------------------------------------------------------------------------
