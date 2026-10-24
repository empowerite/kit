# shellcheck shell=bash
# status.sh - the picture of the work in flight (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# status:render [--no-fetch] prints the open pull requests and the linked worktrees, and changes nothing. It asks
# GitHub once, for every open pull request with the issues it closes, its review threads, and the checks on its
# head; everything else is a local git read.
#
# The picture is a headline counting the pull requests by state, then three sections, each omitted when empty:
#
#   pull requests  — every open pull request: its row, the issues it closes, and, when a worktree under worktrees/
#                    holds its branch, up to ten of the commits it carries past its fork from the default branch
#                    and the worktree with how far that has moved from the fork and the size of its diff.
#   worktrees      — every worktree under worktrees/ with no open pull request, with the same measures.
#   gone branches  — every local branch whose upstream is gone, which kit tidy removes.
#
# A row is the title, the state with its glyph, the age, the checks, and, when there are any, the unresolved review
# threads. Every state has its own glyph, so a row is recognizable without reading the token. Color is by family,
# never by state: green when nothing is left to decide, yellow when a human owes a decision, blue when the next move
# is a machine's or a reviewer's, dim when someone already owns the work, and every timing is dim. The state is the
# first of these that holds:
#
#   draft              🔨  dim     unfinished work that belongs to someone
#   queued             🚦  blue    a check is still running
#   conflicted         💥  yellow  the branch does not apply cleanly to the default branch
#   failing:gate       🔥  yellow  a check failed; the row names which
#   no-check           📭  blue    nothing has reported on its head
#   comments:open      💬  yellow  unresolved review threads
#   blocked:approval   👀  blue    a review is required or requested and has not arrived
#   ready              🛬  green   green, thread-free, approved, and mergeable

use git

# status_commit_cap is how many commits a block lists before eliding the rest.
status_commit_cap=10

# status:width prints the terminal's width in columns: COLUMNS, else the terminal on stderr, else 100.
status:width() {
  local cols=''
  if [[ "${COLUMNS:-}" =~ ^[0-9]+$ ]]
  then
    cols=$(( 10#${COLUMNS} ))
  elif [[ -t 2 ]]
  then
    cols=$(stty size <&2 2>/dev/null | cut -d' ' -f2 || true)
  fi
  if [[ "${cols}" =~ ^[0-9]+$ ]] && (( cols >= 60 ))
  then
    printf '%s\n' "${cols}"
  else
    printf '100\n'
  fi
}

# status:cell <text> <width> prints <text> padded to <width> characters, or clipped to it with an ellipsis.
status:cell() {
  local text=$1 width=$2
  if (( ${#text} > width ))
  then
    printf '%s…' "${text:0:width-1}"
  else
    printf '%s%*s' "${text}" $(( width - ${#text} )) ''
  fi
}

# status:paint <sgr> <text> prints <text> in the color when stderr is a terminal.
status:paint() {
  printf '%s%s%s' "$(out:sgr "$1")" "$2" "$(out:sgr 0)"
}

# status:state-sgr <state> prints the family color of a state.
status:state-sgr() {
  case "$1" in
    (ready)                                     printf '32' ;;
    (conflicted | failing:gate | comments:open) printf '33' ;;
    (queued | no-check | blocked:approval)      printf '34' ;;
    (*)                                         printf '90' ;;
  esac
}

# status:state-emoji <state> prints the glyph of a state. Every glyph is one code point drawn two columns wide, so
# the column after it stays aligned.
status:state-emoji() {
  case "$1" in
    (draft)            printf '🔨' ;;
    (queued)           printf '🚦' ;;
    (conflicted)       printf '💥' ;;
    (failing:gate)     printf '🔥' ;;
    (no-check)         printf '📭' ;;
    (comments:open)    printf '💬' ;;
    (blocked:approval) printf '👀' ;;
    (ready)            printf '🛬' ;;
    (*)                printf '🔳' ;;
  esac
}

# status:collect prints one record per open pull request, oldest first, with the fields separated by the unit
# separator: number, branch, title, state, age, check glyph, check text, how long the checks took, the count of
# unresolved review threads, and the issues it closes as "#n title" separated by the record separator. The one
# request to GitHub.
status:collect() {
  local repo query filter
  repo=$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null) || return 0
  # shellcheck disable=SC2016  # GraphQL variables
  query='query($owner: String!, $name: String!) {
    repository(owner: $owner, name: $name) {
      pullRequests(states: OPEN, first: 100, orderBy: {field: CREATED_AT, direction: ASC}) {
        nodes {
          number title isDraft headRefName createdAt mergeStateStatus reviewDecision
          closingIssuesReferences(first: 10) { nodes { number title } }
          reviewThreads(first: 100) { nodes { isResolved } }
          commits(last: 1) { nodes { commit { statusCheckRollup { contexts(first: 100) { nodes {
            __typename
            ... on CheckRun { name status conclusion startedAt completedAt }
            ... on StatusContext { context state createdAt }
          } } } } } }
        }
      }
    }
  }'
  # shellcheck disable=SC2016  # a jq filter, with jq variables
  filter='
    def age: if . < 120 then "\(.)s" elif . < 7200 then "\(. / 60 | floor)m"
             elif . < 172800 then "\(. / 3600 | floor)h" else "\(. / 86400 | floor)d" end;
    def verdict: if .__typename == "CheckRun" then (if .status == "COMPLETED" then .conclusion else "PENDING" end)
                 else .state end;
    def pending: IN("PENDING", "EXPECTED");
    def passed: IN("SUCCESS", "NEUTRAL", "SKIPPED");
    (now | floor) as $now
    | .data.repository.pullRequests.nodes[]
    | [.commits.nodes[0].commit.statusCheckRollup.contexts.nodes[]?] as $checks
    | ($checks | length) as $n
    | ([$checks[] | verdict | select(pending)] | length) as $running
    | ([$checks[] | verdict | select(passed)] | length) as $ok
    | ([$checks[] | select((verdict | pending or passed) | not) | (.name // .context)] | first // "") as $failing
    | (if $n == 0 then "NONE" elif $failing != "" then "FAILURE" elif $running > 0 then "PENDING" else "SUCCESS" end)
      as $check
    | ([$checks[] | .startedAt // .createdAt // empty | fromdateiso8601] | min) as $started
    | ([$checks[] | .completedAt // empty | fromdateiso8601] | max) as $completed
    | (if $started == null then ""
       elif $check == "PENDING" then (($now - $started) | age)
       elif $completed == null then ""
       else (($completed - $started) | age) end) as $took
    | (if $check == "SUCCESS" then ["✅", "\($n) / \($n)"]
       elif $check == "FAILURE" then ["❌", "\($failing) · \($ok) / \($n)"]
       elif $check == "PENDING" then ["🏃", "\($n - $running) / \($n)"]
       else ["", ""] end) as $cell
    | ([.reviewThreads.nodes[]? | select(.isResolved | not)] | length) as $unresolved
    | (if .isDraft then "draft"
       elif $check == "PENDING" then "queued"
       elif .mergeStateStatus == "DIRTY" then "conflicted"
       elif $check == "FAILURE" then "failing:gate"
       elif $check == "NONE" then "no-check"
       elif $unresolved > 0 then "comments:open"
       elif (.reviewDecision // "") | IN("REVIEW_REQUIRED", "CHANGES_REQUESTED") then "blocked:approval"
       else "ready" end) as $state
    | [ (.number | tostring), .headRefName, .title, $state,
        (($now - (.createdAt | fromdateiso8601)) | age),
        $cell[0], $cell[1], (if $cell[1] == "" then "" else $took end), ($unresolved | tostring),
        ([.closingIssuesReferences.nodes[] | "#\(.number) \(.title)"] | join("\u001e")) ]
    | join("\u001f")'
  gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --raw-field query="${query}" --jq "${filter}" \
    2>/dev/null || true
}

# status:fork <branch> <default-branch> prints the commit where <branch> left the default branch, or nothing.
status:fork() {
  git merge-base "origin/$2" "$1" 2>/dev/null || git merge-base "$2" "$1" 2>/dev/null || true
}

# status:divergence <path> <branch> <fork> prints "<ahead>\t<behind>\t<added>\t<removed>": the commits on <branch>
# past <fork>, the commits on the default branch past it, and the lines the working tree at <path> adds and removes
# against it. A detached checkout has no branch and no fork, and measures its lines against its own head.
status:divergence() {
  local path=$1 branch=$2 fork=$3 default_branch=$4 ahead=0 behind=0 added=0 removed=0 a r _
  if [[ -n "${fork}" ]]
  then
    ahead=$(git rev-list --count "${fork}..${branch}" 2>/dev/null || printf 0)
    behind=$(git rev-list --count "${fork}..origin/${default_branch}" 2>/dev/null || printf 0)
  fi
  while IFS=$'\t' read -r a r _
  do
    [[ -n "${a}" && "${a}" != '-' ]] && added=$(( added + a ))
    [[ -n "${r}" && "${r}" != '-' ]] && removed=$(( removed + r ))
  done < <(git -C "${path}" diff --numstat "${fork:-HEAD}" 2>/dev/null || true)
  printf '%s\t%s\t%s\t%s\n' "${ahead}" "${behind}" "${added}" "${removed}"
}

# status:paint-counts <ahead> <behind> <added> <removed> prints "↑a ↓b · +x -y": a non-zero count in bold green or
# bold red, a zero dim.
status:paint-counts() {
  local ahead=$1 behind=$2 added=$3 removed=$4
  printf '%s %s · %s %s' \
    "$(status:paint-count "${ahead}" '↑' '1;32')" "$(status:paint-count "${behind}" '↓' '1;31')" \
    "$(status:paint-count "${added}" '+' '1;32')" "$(status:paint-count "${removed}" '-' '1;31')"
}

# status:paint-count <count> <sign> <sgr> prints <sign><count>, in <sgr> when the count is above zero and dim otherwise.
status:paint-count() {
  local count=$1 sign=$2 sgr=$3
  (( count > 0 )) || sgr=90
  status:paint "${sgr}" "${sign}${count}"
}

# status:commits <path> <range> prints the commits in <range>, dim hash then subject, indented under a row. Past
# status_commit_cap, the first commits, a dim "… and N more", and the oldest.
status:commits() {
  local path=$1 range=$2 line total hidden=0 i
  local -a commits=()
  mapfile -t commits < <(git -C "${path}" log --format='%h %s' "${range}" 2>/dev/null || true)
  total=${#commits[@]}
  (( total > status_commit_cap + 1 )) && hidden=$(( total - status_commit_cap ))
  for (( i = 0; i < total; i++ ))
  do
    if (( hidden > 0 && i == status_commit_cap - 1 ))
    then
      printf '   %s\n' "$(status:paint 90 "… and ${hidden} more")" >&2
      i=$(( total - 2 ))
      continue
    fi
    line=${commits[i]}
    printf '   %s %s\n' "$(status:paint 90 "${line%% *}")" "${line#* }" >&2
  done
}

# status:pr-block <record> <title-width> <state-width> <age-width> <check-width> <path> <main-root> <default-branch>
# prints one pull request: its row, the issues it closes, and, when <path> names its worktree, the commits it
# carries past its fork and the worktree's line.
status:pr-block() {
  local title_w=$2 state_w=$3 age_w=$4 check_w=$5 path=$6 main_root=$7 default_branch=$8
  local number branch title state age glyph check took unresolved issues issue row fork counts
  IFS=$'\x1f' read -r number branch title state age glyph check took unresolved issues <<<"$1"
  row="🔀 $(status:cell "#${number} ${title}" "${title_w}")"
  row+="   $(status:state-emoji "${state}") $(status:paint "$(status:state-sgr "${state}")" "$(status:cell "${state}" "${state_w}")")"
  row+="   $(status:paint 90 "$(status:cell "${age}" "${age_w}")")"
  if (( unresolved > 0 ))
  then
    row+="   $(status:check-cell "${glyph}" "${check}" "${took}" "${check_w}")   $(status:paint 33 "💬 ${unresolved} unresolved")"
  elif [[ -n "${glyph}" ]]
  then
    row+="   $(status:check-cell "${glyph}" "${check}" "${took}" 0)"
  fi
  printf '%s\n' "${row}" >&2
  if [[ -n "${issues}" ]]
  then
    while IFS= read -r issue
    do
      printf '   %s\n' "$(status:paint 90 "closes ${issue}")" >&2
    done < <(tr '\036' '\n' <<<"${issues}")
  fi
  [[ -n "${path}" ]] || return 0
  fork=$(status:fork "${branch}" "${default_branch}")
  [[ -n "${fork}" ]] && status:commits "${path}" "${fork}..${branch}"
  counts=$(status:divergence "${path}" "${branch}" "${fork}" "${default_branch}")
  # shellcheck disable=SC2086  # the four counts are tab-separated words
  printf '   🌿 %s   %s%s\n' "$(status:paint 90 "${path#"${main_root}"/}")" "$(status:paint-counts ${counts})" \
    "$(status:uncommitted "${path}")" >&2
}

# status:check-cell <glyph> <text> <took> <width> prints the checks as "<glyph> <text> · <took>", the duration dim,
# padded to <width> characters when something follows it; nothing when no check has reported.
status:check-cell() {
  local glyph=$1 text=$2 took=$3 width=$4 plain=''
  if [[ -n "${glyph}" ]]
  then
    plain="${glyph} ${text}${took:+ · ${took}}"
    printf '%s %s%s' "${glyph}" "${text}" "${took:+ · $(status:paint 90 "${took}")}"
  fi
  (( ${#plain} < width )) && printf '%*s' $(( width - ${#plain} )) ''
  return 0
}

# status:worktree-alone <path> <branch> <title-width> <main-root> <default-branch> prints a worktree with no open
# pull request as a row: its path, a mark when it is detached, and its divergence.
status:worktree-alone() {
  local path=$1 branch=$2 title_w=$3 main_root=$4 default_branch=$5 mark='' fork='' counts
  if [[ -n "${branch}" ]]
  then
    fork=$(status:fork "${branch}" "${default_branch}")
  else
    mark="$(status:paint 33 '❗ detached HEAD')   "
  fi
  counts=$(status:divergence "${path}" "${branch}" "${fork}" "${default_branch}")
  # shellcheck disable=SC2086  # the four counts are tab-separated words
  printf '🌿 %s   %s%s%s\n' "$(status:cell "${path#"${main_root}"/}" "${title_w}")" "${mark}" \
    "$(status:paint-counts ${counts})" "$(status:uncommitted "${path}")" >&2
}

# status:uncommitted <path> prints " · n uncommitted" for a dirty checkout, nothing for a clean one.
status:uncommitted() {
  git:clean "$1" && return 0
  printf ' · %s' "$(status:paint 33 "$(git -C "$1" status --porcelain | grep --count .) uncommitted")"
}

# status:by-number <branch>… prints "<number>\t<branch>" for each branch whose worktree directory opens with an issue
# number, 0 for one that does not, sorted by that number.
status:by-number() {
  local branch dir number
  for branch in "$@"
  do
    dir=${worktree_of[${branch}]##*/}
    number=${dir%%-*}
    [[ "${number}" =~ ^[0-9]+$ ]] || number=0
    printf '%s\t%s\n' "${number}" "${branch}"
  done | sort -n -k1,1
}

# status:badges <state>… prints "<glyph> <count>" per state, most populous first.
status:badges() {
  local -A tally=()
  local state count out=''
  for state in "$@"
  do
    tally[${state}]=$(( ${tally[${state}]:-0} + 1 ))
  done
  while IFS=$'\t' read -r count state
  do
    out+="${out:+  }$(status:state-emoji "${state}") ${count}"
  done < <(
    for state in "${!tally[@]}"
    do
      printf '%s\t%s\n' "${tally[${state}]}" "${state}"
    done | sort -t$'\t' -k1,1nr -k2,2
  )
  printf '%s' "${out}"
}

status:render() {
  local no_fetch=false arg main_root default_branch records path branch record
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
  records=$(status:collect)

  # The linked worktrees: by branch, and the detached ones.
  local -A worktree_of=()
  local -a detached_paths=()
  while IFS=$'\t' read -r path branch _
  do
    [[ -n "${path}" ]] || continue
    if [[ -n "${branch}" ]]
    then
      worktree_of[${branch}]=${path}
    else
      detached_paths+=("${path}")
    fi
  done < <(git:worktrees)

  # The rows, and the width of each column across them.
  local -a rows=() states=()
  local -A pr_on=()
  local number title state age glyph check took unresolved issues count=0 state_w=0 age_w=0 check_w=0 title_w=0 issue
  while IFS= read -r record
  do
    [[ -n "${record}" ]] || continue
    IFS=$'\x1f' read -r number branch title state age glyph check took unresolved issues <<<"${record}"
    rows+=("${record}")
    states+=("${state}")
    pr_on[${branch}]=1
    (( ${#state} > state_w )) && state_w=${#state}
    (( ${#age} > age_w )) && age_w=${#age}
    check="${glyph:+${glyph} ${check}${took:+ · ${took}}}"
    (( ${#check} + 1 > check_w )) && check_w=$(( ${#check} + 1 ))
    title="#${number} ${title}"
    (( ${#title} > title_w )) && title_w=${#title}
    while IFS= read -r issue
    do
      (( ${#issue} > title_w )) && title_w=${#issue}
    done < <(tr '\036' '\n' <<<"${issues}")
  done <<<"${records}"
  count=${#rows[@]}
  for path in "${worktree_of[@]}" "${detached_paths[@]}"
  do
    path=${path#"${main_root}"/}
    (( ${#path} > title_w )) && title_w=${#path}
  done
  local avail
  avail=$(( $(status:width) - 3 - 3 - 3 - state_w - 3 - age_w - 3 - check_w - 3 - 16 ))
  (( avail < 20 )) && avail=20
  (( title_w > avail )) && title_w=${avail}

  if (( count == 0 ))
  then
    out:emit '📋' '1' 'no open pull requests'
  else
    out:emit '📋' '1' "${count} open pull request$( (( count == 1 )) || printf s) · $(status:badges "${states[@]}")"
  fi

  # Every pull request, with its worktree when one here holds its branch.
  if (( count > 0 ))
  then
    section "pull requests"
    for record in "${rows[@]}"
    do
      IFS=$'\x1f' read -r _ branch _ <<<"${record}"
      status:pr-block "${record}" "${title_w}" "${state_w}" "${age_w}" "${check_w}" "${worktree_of[${branch}]:-}" \
        "${main_root}" "${default_branch}"
    done
  fi

  # The worktrees with no pull request, by issue number, then the detached ones.
  local -a lone=()
  for branch in "${!worktree_of[@]}"
  do
    [[ -n "${pr_on[${branch}]:-}" ]] || lone+=("${branch}")
  done
  if (( ${#lone[@]} + ${#detached_paths[@]} > 0 ))
  then
    section "worktrees with no open pull request"
    while IFS=$'\t' read -r _ branch
    do
      [[ -n "${branch}" ]] || continue
      status:worktree-alone "${worktree_of[${branch}]}" "${branch}" "${title_w}" "${main_root}" "${default_branch}"
    done < <(status:by-number "${lone[@]}")
    for path in "${detached_paths[@]}"
    do
      status:worktree-alone "${path}" '' "${title_w}" "${main_root}" "${default_branch}"
    done
  fi

  local gone
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

# ----------------------------------------------------------------------------------------------------------------------
