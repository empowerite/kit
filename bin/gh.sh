# shellcheck shell=bash
# gh.sh - the issue on-ramp: a linked branch, a worktree, a start commit, and a draft pull request (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# gh:issue [<number>…] is the one command the loop needs and no vendor provides. The first number is the primary
# issue, linked to the branch under Development by `gh issue develop`, which closes it on merge; any further
# number is a secondary, closed by a `Closes #N` line in the pull request body. Every listed issue is assigned to
# the caller. It never switches the checkout it runs from: the branch is checked out in worktrees/<N>-<slug>
# under the main checkout, and an existing branch, worktree, or pull request is reused, so a re-run reconciles.
#
# With no argument and a terminal on stdin it asks for a title and creates the issue; without a terminal the
# usage error stands, so an agent never hangs on a prompt. Uncommitted work in the root checkout, the human's
# desk, is swept into the new worktree, uncommitted, and the desk is left clean, and fast-forwarded when it is on
# the default branch; inside any other checkout, dirty files are that issue's own and stay. A sweep that cannot
# apply leaves the stash in place and stops.
#
# Each step below prints its one result on stdout and talks on stderr, so gh:issue reads them in order.

# gh:repo prints the repository this checkout resolves to, as owner/name, or fails.
gh:repo() {
  gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null
}

# gh:me prints the authenticated login. A revoked token passes `gh auth status`, so this is the real probe.
gh:me() {
  local me
  me=$(gh api user --jq '.login' 2>/dev/null || true)
  if [[ -z "${me}" ]]
  then
    error "run: gh auth login (the GitHub CLI is not authenticated)"
    return 1
  fi
  printf '%s\n' "${me}"
}

# gh:issue-create asks for a title at the terminal and prints the number of the issue it creates.
gh:issue-create() {
  local title number
  read -r -p "Issue title: " title
  if [[ -z "${title}" ]]
  then
    error "an issue title is required"
    return 2
  fi
  number=$(gh issue create --title "${title}" --body "" 2>/dev/null | sed -n 's@.*/issues/@@p')
  if [[ ! "${number}" =~ ^[0-9]+$ ]]
  then
    error "run: gh auth status (the issue could not be created)"
    return 1
  fi
  ok "Created #${number}: ${title}"
  printf '%s\n' "${number}"
}

# gh:assign <me> <number>… adds <me> to each issue's assignees, warning where someone else is there already.
gh:assign() {
  local me=$1 n assignees
  shift
  for n in "$@"
  do
    assignees=$(gh issue view "${n}" --json assignees --jq '[.assignees[].login] | join(" ")')
    [[ " ${assignees} " == *" ${me} "* ]] && continue
    [[ -z "${assignees}" ]] \
      || warn "#${n} is already assigned to ${assignees}; adding you as well. Coordinate before continuing."
    gh issue edit "${n}" --add-assignee "${me}" >/dev/null 2>&1 \
      || warn "#${n} could not be assigned to ${me}; assign it by hand."
  done
}

# gh:issue-branch <number> <default-branch> prints the branch linked to the issue, making and linking one if none
# exists. An existing one is found by an open pull request's head, then by the remote ref, so a second run never
# forks a <N>-<slug>-1 duplicate.
gh:issue-branch() {
  local number=$1 default_branch=$2 branch
  branch=$(gh pr list --state open --json headRefName \
    --jq "[.[] | select(.headRefName | test(\"^${number}-\"))] | .[0].headRefName // empty" 2>/dev/null || true)
  [[ -n "${branch}" ]] \
    || branch=$(git ls-remote --heads origin "${number}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
  if [[ -z "${branch}" ]]
  then
    step "Linking a branch to #${number} (gh issue develop)"
    gh issue develop "${number}" --base "${default_branch}" >/dev/null
    git fetch --quiet origin
    branch=$(git ls-remote --heads origin "${number}-*" 2>/dev/null | sed -n '1s@.*refs/heads/@@p')
    if [[ -z "${branch}" ]]
    then
      error "the branch gh made for #${number} could not be found; run: gh issue develop ${number} --list"
      return 1
    fi
  fi
  printf '%s\n' "${branch}"
}

# gh:worktree <main-root> <branch> prints the worktree the branch is checked out in, making it if there is none.
gh:worktree() {
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

# gh:start-commit <worktree> <default-branch> <number> <title> gives the branch its first commit when it has
# none: GitHub cannot open a pull request with nothing to compare. --only keeps the commit empty, so work already
# staged in a reused worktree stays in the index. The subject follows the loop's shape, #<N> <area>: <subject>.
gh:start-commit() {
  local wt=$1 default_branch=$2 number=$3 title=$4
  if [[ -z "$(git -C "${wt}" rev-list "origin/${default_branch}..HEAD" 2>/dev/null)" ]]
  then
    step "Adding the start commit"
    git -C "${wt}" commit --quiet --allow-empty --only --message "#${number} start: ${title}"
  fi
}

# gh:sweep-push <root> <message> stashes everything uncommitted at <root>, untracked files included, and prints
# the stash's commit, which stays valid however the stash list shifts.
gh:sweep-push() {
  local root=$1 message=$2
  if ! git -C "${root}" stash push --include-untracked --message "${message}" >/dev/null
  then
    error "the uncommitted work at ${root} could not be stashed; resolve it by hand and re-run."
    return 1
  fi
  git -C "${root}" rev-parse 'stash@{0}'
}

# gh:sweep-apply <root> <worktree> <stash> <default-branch> lands the stash in the worktree, uncommitted, drops it,
# and fast-forwards the now-clean desk when it is on the default branch. On a conflict the stash is kept and the
# caller stops.
gh:sweep-apply() {
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

# gh:pull-request <branch> <default-branch> <title> prints the number of the open pull request for the branch,
# opening a draft if there is none.
gh:pull-request() {
  local branch=$1 default_branch=$2 title=$3 pr
  pr=$(gh pr list --head "${branch}" --state open --json number --jq '.[0].number // empty')
  if [[ -z "${pr}" ]]
  then
    step "Opening a draft pull request"
    gh pr create --draft --base "${default_branch}" --head "${branch}" --title "${title}" --body "" >/dev/null
    pr=$(gh pr list --head "${branch}" --state open --json number --jq '.[0].number')
  fi
  printf '%s\n' "${pr}"
}

# gh:link-secondaries <pr> <number>… adds a `Closes #N` line to the pull request body for each number not there.
gh:link-secondaries() {
  local pr=$1 n body changed=false
  shift
  body=$(gh pr view "${pr}" --json body --jq '.body // ""')
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
    gh pr edit "${pr}" --body "${body}" >/dev/null
  fi
}

# gh:linked-open-issue <number> refuses pull request <number> unless it is linked, under Development, to an issue
# of this repository that is open. A linked issue of another repository, or a closed one, is noted and does not
# count. On a runner the remedy also lands in the step summary.
gh:linked-open-issue() {
  local number=$1 repo query refs n state issue_repo open=0 closes url remedy
  if ! IFS=$'\t' read -r repo url < <(gh repo view --json nameWithOwner,url --jq '[.nameWithOwner, .url] | @tsv')
  then
    error "run: gh auth status (the repository could not be read, so pull request #${number} was not checked)"
    return 1
  fi
  remedy=$(printf '%s\n' \
    "Link an open issue of this repository under **Development** in [the sidebar](${url}/pull/${number}):" "" \
    "1. click the gear (⚙) beside **Development**" \
    "2. pick one or more open issues" \
    "3. re-run the check: linking fires no event, so it does not re-run on its own" "" \
    "\`kit issue <N>\` makes this link when it starts the work.")
  # shellcheck disable=SC2016  # GraphQL variables
  query='query($owner: String!, $name: String!, $number: Int!) {
    repository(owner: $owner, name: $name) {
      pullRequest(number: $number) {
        closingIssuesReferences(first: 100) { nodes { number state repository { nameWithOwner } } }
      }
    }
  }'
  if ! refs=$(gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --field number="${number}" \
    --raw-field query="${query}" \
    --jq '.data.repository.pullRequest.closingIssuesReferences.nodes[]
          | "\(.number) \(.state) \(.repository.nameWithOwner)"')
  then
    error "run: gh auth status (pull request #${number} could not be read)"
    return 1
  fi
  if [[ -z "${refs}" ]]
  then
    error "pull request #${number} is not linked to any issue"
    summary "### ❌ Pull request #${number} is not linked to any issue" "" "${remedy}"
    return 1
  fi
  while read -r n state issue_repo
  do
    if [[ "${issue_repo}" != "${repo}" ]]
    then
      warn "skipping ${issue_repo}#${n}: it belongs to another repository"
    elif [[ "${state}" != OPEN ]]
    then
      warn "#${n} is ${state}; merging will not change it"
    else
      open=$(( open + 1 ))
    fi
  done <<<"${refs}"
  if (( open == 0 ))
  then
    error "every issue pull request #${number} is linked to is closed or belongs to another repository"
    summary "### ❌ Every linked issue is closed or belongs to another repository" "" "${remedy}"
    return 1
  fi
  closes=$(cut -d ' ' -f 1 <<<"${refs}" | sed 's/^/#/' | paste -s -d ' ' -)
  ok "pull request #${number} is linked to ${closes}"
  summary "✅ Linked to ${closes}."
}

# gh:issue [<number>…] is the issue on-ramp the file header describes: assign, link a branch, make the worktree,
# push a start commit, sweep the desk's uncommitted work into the worktree, and open a draft pull request. Every
# commit the loop makes is signed, so a desk without a signing key is refused before anything is written.
gh:issue() {
  use git
  local me n primary title default_branch main_root desk desk_branch branch wt stash="" pr created
  me=$(gh:me) || return 1
  guard:require signing-key || return 1
  if (( $# == 0 ))
  then
    if [[ ! -t 0 ]]
    then
      error "usage: kit issue <issue-number> [<issue-number>…]"
      return 2
    fi
    created=$(gh:issue-create) || return
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
  title=$(gh issue view "${primary}" --json title --jq '.title')
  if [[ -z "${title}" ]]
  then
    error "run: gh auth status (the title of #${primary} could not be read)"
    return 1
  fi
  gh:assign "${me}" "$@"
  section "Starting #${primary}: ${title}"

  branch=$(gh:issue-branch "${primary}" "${default_branch}") || return
  if [[ "${desk}" == "${main_root}" ]] && ! git:clean "${desk}"
  then
    step "Sweeping the uncommitted work on ${desk_branch} into the worktree"
    stash=$(gh:sweep-push "${desk}" "kit issue #${primary} → ${branch}") || return
  fi
  wt=$(gh:worktree "${main_root}" "${branch}") || return
  git -C "${main_root}" check-ignore --quiet worktrees 2>/dev/null \
    || warn "worktrees/ is not ignored; add /worktrees/ to .gitignore so a worktree never appears in a commit"
  gh:start-commit "${wt}" "${default_branch}" "${primary}" "${title}"
  git -C "${wt}" push --quiet --set-upstream origin "${branch}"
  if [[ -n "${stash}" ]]
  then
    gh:sweep-apply "${desk}" "${wt}" "${stash}" "${default_branch}" || return
  fi
  pr=$(gh:pull-request "${branch}" "${default_branch}" "${title}")
  (( $# > 1 )) && gh:link-secondaries "${pr}" "${@:2}"

  ok "Draft pull request #${pr} on ${branch}, primary #${primary}${2:+, secondaries: ${*:2}}"
  ok "Worktree ready: ${wt}"
  step "cd '${wt}'    # do all work there; this checkout stays on ${desk_branch}"
}

# ----------------------------------------------------------------------------------------------------------------------
