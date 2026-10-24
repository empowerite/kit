# shellcheck shell=bash
# git.sh - the default branch, the worktrees, the sweep, and the rebase (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# git:status [--no-fetch] draws the picture of the work in flight: the open pull requests, and under each its
# state, age, checks, the commits it carries, and the worktree it is written in, with how far that is ahead of and
# behind the default branch and the size of its diff; then the worktrees with no pull request; then the local
# branches whose upstream is gone. It changes nothing, and asks GitHub once.
#
# git:tidy [--dry-run] removes every worktree under worktrees/ whose pull request is merged or closed, or that is
# detached, clean, and at a commit the default branch already has; prunes the worktree bookkeeping; and deletes
# every local branch whose upstream is gone. It never touches the default branch, a branch checked out in a
# live worktree, a dirty worktree, a detached worktree whose commit nothing else reaches, or the worktree it
# runs in. --dry-run says what a sweep would do and changes nothing.
#
# git:rebase [<path>] rebases the branch checked out at <path>, the current checkout by default, onto
# origin/<default> and pushes it with --force-with-lease. It refuses the default branch, the human's desk, and
# on a conflict aborts and leaves the worktree as it was.

# git:default-branch [<path>] prints the default branch of the repository at <path>: origin/HEAD, else main.
git:default-branch() {
  local branch
  branch=$(git -C "${1:-.}" symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>/dev/null | sed 's@^origin/@@' || true)
  [[ -n "${branch}" ]] || branch=main
  printf '%s\n' "${branch}"
}

# git:main-root [<path>] prints the root of the main checkout, the first entry of the worktree list. worktrees/
# lives under it whether the caller stands in the main checkout or in a linked worktree.
git:main-root() {
  git -C "${1:-.}" worktree list --porcelain | sed -n 's@^worktree @@p' | head -n 1
}

# git:worktrees prints "<path>\t<branch>\t<detached>" for every linked worktree under worktrees/; <branch> is
# empty and <detached> is true for a detached checkout.
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
        # git ends every record with a blank line; the printf below adds one so the last record ends too.
        [[ "${path}" == "${worktree_root}"/* ]] && printf '%s\t%s\t%s\n' "${path}" "${branch}" "${detached}"
        path=""
    esac
  done < <(git worktree list --porcelain; printf '\n')
}

# git:pr-summary <branch> prints "<number>\t<Draft|Ready>" for the open pull request on <branch>, or nothing.
git:pr-summary() {
  gh pr list --head "$1" --state open --json number,isDraft \
    --jq '.[0] | select(.number) | "\(.number)\t\(if .isDraft then "Draft" else "Ready" end)"' 2>/dev/null || true
}

# git:gone-branches prints every local branch whose upstream no longer exists.
git:gone-branches() {
  git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads/ | awk '$2 == "[gone]" { print $1 }'
}

# git:clean <path> is true when the checkout at <path> has no uncommitted change.
git:clean() {
  [[ -z "$(git -C "$1" status --porcelain 2>/dev/null)" ]]
}

# git:open-prs prints one line per open pull request: "<branch>\t<number>\t<Draft|Ready>\t<age in days>\t<checks>\t<title>",
# where <checks> is "✅ n/n", "❌ k/n", "⏳ n", or "·" for none. One request, for the whole picture.
git:open-prs() {
  # shellcheck disable=SC2016  # a jq filter, with jq variables
  gh pr list --state open --limit 100 --json number,title,isDraft,headRefName,createdAt,statusCheckRollup --jq '
    .[] | [
      .headRefName,
      (.number | tostring),
      (if .isDraft then "Draft" else "Ready" end),
      (((now - (.createdAt | fromdateiso8601)) / 86400 | floor | tostring) + "d"),
      ((.statusCheckRollup // []) as $c
        | ($c | length) as $n
        | ([$c[] | select(.conclusion == "SUCCESS")] | length) as $ok
        | ([$c[] | select(.conclusion != null and .conclusion != "SUCCESS" and .conclusion != "SKIPPED" and .conclusion != "NEUTRAL")] | length) as $bad
        | if $n == 0 then "·"
          elif $bad > 0 then "❌ \($bad)/\($n)"
          elif $ok == $n then "✅ \($n)/\($n)"
          else "⏳ \($n - $ok)/\($n)" end),
      .title
    ] | @tsv' 2>/dev/null || true
}

# git:short <text> [<width>] prints <text> cut to <width> columns, 16 by default, with an ellipsis when cut.
git:short() {
  local text=$1 width=${2:-16}
  if (( ${#text} > width ))
  then
    printf '%s…\n' "${text:0:width-1}"
  else
    printf '%s\n' "${text}"
  fi
}

# git:worktree-line <path> <branch> <main-root> <default-branch> prints the 🌿 line: the worktree's path, how far
# its branch is ahead of and behind the default branch, and the size of its diff.
git:worktree-line() {
  local path=$1 branch=$2 main_root=$3 default_branch=$4 counts="0\t0" stat added=0 removed=0 dirty=""
  if [[ -n "${branch}" ]]
  then
    counts=$(git rev-list --left-right --count "${branch}...origin/${default_branch}" 2>/dev/null || printf '0\t0')
    stat=$(git diff --shortstat "origin/${default_branch}...${branch}" 2>/dev/null || true)
    [[ "${stat}" =~ ([0-9]+)\ insertion ]] && added=${BASH_REMATCH[1]}
    [[ "${stat}" =~ ([0-9]+)\ deletion ]] && removed=${BASH_REMATCH[1]}
  fi
  git:clean "${path}" || dirty=" · $(git -C "${path}" status --porcelain | grep --count .) uncommitted"
  printf '   🌿 %s   ↑%s ↓%s · +%s -%s%s\n' "$(git:short "${path#"${main_root}"/}" 40)" "${counts%%$'\t'*}" "${counts##*$'\t'}" "${added}" "${removed}" "${dirty}" >&2
}

# git:pr-block <pr-line> <path> <main-root> <default-branch> prints one pull request: its line, the commits it
# carries over the default branch, up to ten, and its worktree's line.
git:pr-block() {
  local branch number state age checks title path=$2 main_root=$3 default_branch=$4 shown=0 total line
  IFS=$'\t' read -r branch number state age checks title <<<"$1"
  printf '🔀 #%s %-17s %-6s %4s   %s\n' "${number}" "$(git:short "${title}")" "${state}" "${age}" "${checks}" >&2
  total=$(git rev-list --count "origin/${default_branch}..${branch}" 2>/dev/null || printf 0)
  while IFS= read -r line
  do
    printf '   %s\n' "${line}" >&2
    (( ++shown ))
  done < <(git log --format='%h %s' --max-count 10 "origin/${default_branch}..${branch}" 2>/dev/null)
  (( total > shown )) && printf '   … and %s more\n' "$(( total - shown ))" >&2
  git:worktree-line "${path}" "${branch}" "${main_root}" "${default_branch}"
}

# git:tidy-worktree <path> <branch> <detached> <default-branch> <self-root> <dry-run> decides one worktree and
# says why. It returns 0 when the worktree was removed, or would be under a dry run, and 1 when it is kept.
git:tidy-worktree() {
  local path=$1 branch=$2 detached=$3 default_branch=$4 self_root=$5 dry_run=$6 state="" number
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
    state=$(gh pr list --head "${branch}" --state all --json state --jq '.[0].state // empty' 2>/dev/null || true)
    if [[ -z "${state}" && "${branch}" =~ ^([0-9]+)- ]]
    then
      number=${BASH_REMATCH[1]}
      [[ "$(gh issue view "${number}" --json state --jq .state 2>/dev/null)" == CLOSED ]] && state=CLOSED
    fi
  elif [[ "${detached}" == true ]]
  then
    # A detached checkout is residue only when its commit is already on the default branch; otherwise its HEAD
    # may be the only thing that reaches that commit.
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
      step "keeping ${path} (${branch:-detached}, pull request ${state:-open or unknown})"
      return 1
  esac
}

git:status() {
  local no_fetch=false arg main_root default_branch prs path branch detached pr_line counts gone
  local -A worktree_of=()
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
  main_root=$(git:main-root)
  default_branch=$(git:default-branch)
  prs=$(git:open-prs)

  counts=$(git -C "${main_root}" rev-list --left-right --count "HEAD...origin/${default_branch}" 2>/dev/null || printf '0\t0')
  printf '📋 %s open pull request(s) · %s %s%s, ↑%s ↓%s vs origin/%s\n' \
    "$(grep --count . <<<"${prs}" || true)" "${main_root##*/}" \
    "$(git -C "${main_root}" rev-parse --abbrev-ref HEAD 2>/dev/null || printf '?')" \
    "$(git:clean "${main_root}" || printf ' (dirty)')" "${counts%%$'\t'*}" "${counts##*$'\t'}" "${default_branch}" >&2

  while IFS=$'\t' read -r path branch detached
  do
    [[ -n "${branch}" ]] && worktree_of[${branch}]=${path}
  done < <(git:worktrees)

  section "pull requests"
  if [[ -n "${prs}" ]]
  then
    while IFS= read -r pr_line
    do
      branch=${pr_line%%$'\t'*}
      if [[ -n "${worktree_of[${branch}]+x}" ]]
      then
        git:pr-block "${pr_line}" "${worktree_of[${branch}]}" "${main_root}" "${default_branch}"
        unset 'worktree_of[${branch}]'
      else
        git:pr-block "${pr_line}" "" "${main_root}" "${default_branch}" 2>&1 | sed '$d' >&2
        printf '   (no worktree here)\n' >&2
      fi
    done <<<"${prs}"
  else
    ok "none"
  fi

  section "worktrees with no open pull request"
  if (( ${#worktree_of[@]} > 0 ))
  then
    for branch in "${!worktree_of[@]}"
    do
      git:worktree-line "${worktree_of[${branch}]}" "${branch}" "${main_root}" "${default_branch}"
    done
  else
    ok "none"
  fi

  section "branches whose upstream is gone (tidy removes these)"
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
  guard:require gh || return 1
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
  while IFS=$'\t' read -r path branch detached
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

git:rebase() {
  local worktree=${1:-.} branch default_branch
  branch=$(git -C "${worktree}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  if [[ -z "${branch}" ]]
  then
    error "${worktree} has no branch checked out, so there is nothing to rebase"
    return 1
  fi
  default_branch=$(git:default-branch "${worktree}")
  if [[ "${branch}" == "${default_branch}" ]]
  then
    error "refusing to rebase ${branch}: it is the default branch, the human's desk; it advances by: git pull --ff-only"
    return 1
  fi
  section "kit rebase ${branch} onto origin/${default_branch}"
  git -C "${worktree}" fetch --quiet --prune origin
  if ! git -C "${worktree}" rebase --quiet "origin/${default_branch}"
  then
    git -C "${worktree}" rebase --abort
    error "${branch} conflicts with origin/${default_branch}; the rebase was aborted and the worktree is as it was"
    return 1
  fi
  git -C "${worktree}" push --quiet --force-with-lease
  ok "${branch} is rebased onto origin/${default_branch} and pushed."
}

# ----------------------------------------------------------------------------------------------------------------------
