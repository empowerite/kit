# shellcheck shell=bash
# issue.sh - start an issue: a linked branch, a worktree, a start commit, and a draft pull request (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# issue:start [<reference>…] starts or resumes work on one or more issues. Each reference names its tracker, as
# bash/tracker.sh describes:
#
#   - `7` or `#7`: an issue in the forge's own tracker (GitHub or GitLab)
#   - `AB#4521`: a work item in a declared Azure DevOps tracker
#
# The first reference is the primary issue. The others are secondary. Each issue is closed on merge by:
#
#   - a primary issue in the forge's tracker: the forge links it to the branch the forge creates
#   - any other primary issue, and every secondary one: a `Closes <reference>` line in the pull request's body
#
# Every listed issue is assigned to the caller, as each tracker identifies the caller.
#
# issue:start never switches the checkout it runs from. It checks the branch out in worktrees/<key>-<slug> under
# the main checkout, and hydrates it: what the kit delivers under .kit/ is in place before the first command runs
# there. The key is the reference as it starts a branch name: 7 or ab4521. An existing branch, worktree, or pull
# request is reused, so running it again fills in whatever is missing.
#
# With no argument and a terminal on stdin, it asks for a title and creates the issue. Without a terminal it fails
# with a usage error, so an agent never waits on a prompt.
#
# Uncommitted work is handled by where it is:
#
#   - in the root checkout (the desk, which belongs to the human): moved into the new worktree, still uncommitted.
#     The desk is left clean, and is fast-forwarded when it is on the default branch. If the work cannot be
#     applied in the worktree, it stays in the stash and the command stops.
#   - in any other checkout: left in place, since it belongs to that checkout's issue
#
# Every forge call goes through bash/forge.sh, and every issue call through bash/tracker.sh. Each helper below
# prints its one result on stdout and its messages on stderr, so issue:start can capture the results in order.

use git forge tracker

# issue:create asks for a title at the terminal, creates the issue in the forge's tracker, and prints its reference.
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

# issue:assign <reference>… adds the caller (as each issue's tracker identifies the caller) to each issue's
# assignees. It warns when someone else is already assigned, and when the assignment fails.
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

# issue:branch <reference> <default-branch> <title> prints the issue's branch. It reuses an existing branch, so a
# second run never creates a duplicate. It looks, in order, for:
#
#   1. the branch of an open pull request for the issue
#   2. a branch on origin named <key>-*
#
# When neither exists, it creates the branch:
#
#   - for an issue in the forge's tracker, the forge creates a branch linked to the issue
#   - for any other issue, kit pushes <key>-<slug of the title> to origin from the default branch
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

# issue:worktree <main-root> <branch> prints the worktree the branch is checked out in. If there is none, it
# creates <main-root>/worktrees/<branch>.
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

# issue:hydrate <worktree> places what the active packages deliver under .kit/ in the worktree, with the kit that
# runs there: the worktree's own kit/bin/kit when the repository carries its release, as the kit's own does, so the
# links lead into that worktree's sources, and otherwise this release's kit. A worktree whose .kit/settings.yml
# declares no kit version is left alone. What hydrate placed is said in one line; its own lines are shown only when
# it fails.
issue:hydrate() {
  local wt=$1 out
  grep --quiet '^version:' "${wt}/.kit/settings.yml" 2>/dev/null || return 0
  step "Hydrating ${wt}"
  if [[ -x "${wt}/kit/bin/kit" ]]
  then
    out=$(cd "${wt}" && kit_depth=$(( ${kit_depth:-0} + 1 )) kit/bin/kit hydrate 2>&1) && return 0
  else
    out=$(cd "${wt}" && run kit hydrate 2>&1) && return 0
  fi
  printf '%s\n' "${out}" >&2
  return 1
}

# issue:start-commit <worktree> <default-branch> <reference> <title> adds an empty first commit when the branch has
# no commits beyond the default branch, because a forge cannot open a pull request with nothing to compare. --only
# keeps the commit empty, so work already staged in a reused worktree stays in the index. The subject starts with
# the reference, as the workflow's commit subjects do: "<reference> start: <title>".
issue:start-commit() {
  local wt=$1 default_branch=$2 ref=$3 title=$4
  if [[ -z "$(git -C "${wt}" rev-list "origin/${default_branch}..HEAD" 2>/dev/null)" ]]
  then
    step "Adding the start commit"
    git -C "${wt}" commit --quiet --allow-empty --only --message "${ref} start: ${title}"
  fi
}

# issue:sweep-push <root> <message> stashes all uncommitted work at <root>, including untracked files, and prints
# the stash's commit hash. The hash stays valid even when other entries are pushed onto or dropped from the stash.
issue:sweep-push() {
  local root=$1 message=$2
  if ! git -C "${root}" stash push --include-untracked --message "${message}" >/dev/null
  then
    error "the uncommitted work at ${root} could not be stashed; resolve it by hand and re-run."
    return 1
  fi
  git -C "${root}" rev-parse 'stash@{0}'
}

# issue:sweep-apply <root> <worktree> <stash> <default-branch> applies the stash in the worktree, uncommitted, and
# drops it. Then, if the desk is on the default branch, it fast-forwards the now-clean desk. On a conflict it keeps
# the stash and returns non-zero, so the caller stops.
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

# issue:pull-request <branch> <default-branch> <title> prints the number of the branch's open pull request,
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

# issue:link-in-body <change> <reference>… adds a `Closes <reference>` line to the pull request's body for each
# reference the body does not already close, fix, or resolve. The references are the secondary issues, and a
# primary issue the forge cannot link by itself.
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

# issue:start [<reference>…] does what the file header describes. When the require_signing setting is true, it
# fails before writing anything if git has no signing key configured.
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
  issue:hydrate "${wt}" || return 1
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
