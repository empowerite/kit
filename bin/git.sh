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

# git:open-prs prints one record per open pull request, fields separated by tabs: branch, number, Draft or
# Ready, age in days, checks, title, and the issues it closes as "number:title" joined by unit separators.
git:open-prs() {
  local repo query filter
  repo=$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null) || return 0
  # shellcheck disable=SC2016  # GraphQL variables, and a jq filter with jq variables
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      pullRequests(states: OPEN, first: 50, orderBy: {field: CREATED_AT, direction: ASC}) {
        nodes {
          number title isDraft headRefName createdAt
          closingIssuesReferences(first: 10) { nodes { number title } }
          commits(last: 1) { nodes { commit { statusCheckRollup { contexts(first: 100) { nodes {
            ... on CheckRun { conclusion status }
            ... on StatusContext { state }
          } } } } } }
        }
      }
    }
  }'
  # shellcheck disable=SC2016  # a jq filter, with jq variables
  filter='
    .data.repository.pullRequests.nodes[] | [
      .headRefName,
      (.number | tostring),
      (if .isDraft then "Draft" else "Ready" end),
      (((now - (.createdAt | fromdateiso8601)) / 86400 | floor | tostring) + "d"),
      ((.commits.nodes[0].commit.statusCheckRollup.contexts.nodes // []) as $c
        | ($c | length) as $n
        | ([$c[] | select(.conclusion == "SUCCESS" or .state == "SUCCESS")] | length) as $ok
        | ([$c[] | select((.conclusion // .state // "") as $r | $r == "FAILURE" or $r == "ERROR" or $r == "TIMED_OUT" or $r == "CANCELLED" or $r == "ACTION_REQUIRED" or $r == "STARTUP_FAILURE")] | length) as $bad
        | if $n == 0 then "·"
          elif $bad > 0 then "❌ \($n - $bad) / \($n)"
          elif $ok == $n then "✅ \($n) / \($n)"
          else "⏳ \($ok) / \($n)" end),
      .title,
      ([.closingIssuesReferences.nodes[] | "\(.number):\(.title)"] | join("\u001f"))
    ] | @tsv'
  gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --raw-field query="${query}" --jq "${filter}" \
    2>/dev/null || true
}

# git:pad <text> <width> prints <text> padded with spaces to <width> characters, counted as characters.
git:pad() {
  local text=$1 width=$2
  printf '%s' "${text}"
  (( ${#text} < width )) && printf '%*s' $(( width - ${#text} )) ''
  return 0
}

# git:dim, git:red, git:green wrap a text in a color when stderr is a terminal.
git:dim()   { printf '%s%s%s' "$(out:sgr 90)" "$*" "$(out:sgr 0)"; }
git:red()   { printf '%s%s%s' "$(out:sgr 31)" "$*" "$(out:sgr 0)"; }
git:green() { printf '%s%s%s' "$(out:sgr 32)" "$*" "$(out:sgr 0)"; }

# git:divergence <branch> <default-branch> prints "↑a ↓b · +ins -del", colored, for the branch against origin.
git:divergence() {
  local branch=$1 default_branch=$2 counts stat added=0 removed=0
  counts=$(git rev-list --left-right --count "${branch}...origin/${default_branch}" 2>/dev/null || printf '0\t0')
  stat=$(git diff --shortstat "origin/${default_branch}...${branch}" 2>/dev/null || true)
  [[ "${stat}" =~ ([0-9]+)\ insertion ]] && added=${BASH_REMATCH[1]}
  [[ "${stat}" =~ ([0-9]+)\ deletion ]] && removed=${BASH_REMATCH[1]}
  printf '%s %s · %s %s' "$(git:green "↑${counts%%$'\t'*}")" "$(git:red "↓${counts##*$'\t'}")" \
    "$(git:green "+${added}")" "$(git:red "-${removed}")"
}

# git:pr-block <record> <path> <main-root> <default-branch> <width> prints one pull request: its line, the issues
# it closes, up to ten of the commits it carries over the default branch, and its worktree's line.
git:pr-block() {
  local branch number state age checks title issues path=$2 main_root=$3 default_branch=$4 width=$5
  local total shown=0 line issue colored
  IFS=$'\t' read -r branch number state age checks title issues <<<"$1"
  if [[ "${state}" == Draft ]]
  then
    colored=$(git:dim "draft")
  else
    colored=$(git:green "ready")
  fi
  printf '🔀 %s   %s   %4s   %s\n' "$(git:pad "#${number} ${title}" "${width}")" "${colored}" "${age}" "${checks}" >&2
  if [[ -n "${issues}" ]]
  then
    while IFS= read -r issue
    do
      printf '   %s\n' "$(git:dim "#${issue%%:*} ${issue#*:}")" >&2
    done < <(tr '\037' '\n' <<<"${issues}")
  fi
  total=$(git rev-list --count "origin/${default_branch}..${branch}" 2>/dev/null || printf 0)
  while IFS= read -r line
  do
    printf '   %s %s\n' "$(git:dim "${line%% *}")" "${line#* }" >&2
    (( ++shown ))
  done < <(git log --format='%h %s' --max-count 10 "origin/${default_branch}..${branch}" 2>/dev/null)
  (( total > shown )) && printf '   %s\n' "$(git:dim "… and $(( total - shown )) more")" >&2
  if [[ -n "${path}" ]]
  then
    printf '   🌿 %s   %s%s\n' "$(git:dim "${path#"${main_root}"/}")" "$(git:divergence "${branch}" "${default_branch}")" "$(git:uncommitted "${path}")" >&2
  else
    printf '   %s\n' "$(git:dim "no worktree here")" >&2
  fi
}

# git:uncommitted <path> prints " · n uncommitted" for a dirty checkout, nothing for a clean one.
git:uncommitted() {
  git:clean "$1" && return 0
  printf ' · %s' "$(git:red "$(git -C "$1" status --porcelain | grep --count .) uncommitted")"
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
  local no_fetch=false arg main_root default_branch prs path branch detached record width=40 text drafts total gone
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
  while IFS=$'\t' read -r path branch detached
  do
    [[ -n "${branch}" ]] && worktree_of[${branch}]=${path}
  done < <(git:worktrees)

  # The name column is as wide as the widest name, within what the terminal allows beside the other columns.
  while IFS=$'\t' read -r branch text _
  do
    [[ -n "${branch}" ]] || continue
    text="#${text}"
    (( ${#text} > width )) && width=${#text}
  done < <(awk -F '\t' '{ print $1 "\t" $2 " " $6 }' <<<"${prs}")
  for branch in "${!worktree_of[@]}"
  do
    text=${worktree_of[${branch}]#"${main_root}"/}
    (( ${#text} > width )) && width=${#text}
  done
  (( width > $(tput cols 2>/dev/null || echo 160) - 40 )) && width=$(( $(tput cols 2>/dev/null || echo 160) - 40 ))

  drafts=$(awk -F '\t' '$3 == "Draft"' <<<"${prs}" | grep --count . || true)
  total=$(grep --count . <<<"${prs}" || true)
  printf '📋 %s open pull request(s) · %s draft · %s ready\n' "${total}" "${drafts}" "$(( total - drafts ))" >&2
  section "worktrees"
  while IFS= read -r record
  do
    [[ -n "${record}" ]] || continue
    branch=${record%%$'\t'*}
    git:pr-block "${record}" "${worktree_of[${branch}]:-}" "${main_root}" "${default_branch}" "${width}"
    unset 'worktree_of[${branch}]'
  done <<<"${prs}"
  for branch in $(printf '%s\n' "${!worktree_of[@]}" | sort)
  do
    path=${worktree_of[${branch}]}
    printf '🌿 %s   %s   %s%s\n' "$(git:pad "${path#"${main_root}"/}" "${width}")" "$(git:dim 'no open pull request')" \
      "$(git:divergence "${branch}" "${default_branch}")" "$(git:uncommitted "${path}")" >&2
  done
  [[ -n "${prs}" || ${#worktree_of[@]} -gt 0 ]] || ok "none"

  gone=$(git:gone-branches)
  if [[ -n "${gone}" ]]
  then
    section "branches whose upstream is gone (tidy removes these)"
    while IFS= read -r branch
    do
      info "${branch}"
    done <<<"${gone}"
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
