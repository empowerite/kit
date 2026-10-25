# shellcheck shell=bash
# status.sh - the picture of the work in flight (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# status:render [--no-fetch] [--json] prints the checkouts and the open pull requests, and changes nothing. It asks
# GitHub once, for every open pull request with the issues it closes, its review threads, and the checks on its
# head; everything else is a local git read.
#
# The picture is three groups of blocks, a blank line between blocks and no headings:
#
#   🏠 the desk        the root checkout, the human's. Its branch, then how far it has moved from where that branch
#                      left origin's default branch, the lines its working tree adds and removes against that point,
#                      and its uncommitted paths; under it the commits it carries past that point, dim hash and
#                      subject, up to ten with the rest counted.
#   🌿 each worktree   under worktrees/, by issue number, drawn exactly as the desk is. The label is the path from
#                      the current directory with the branch underlined when the path ends in it, or "branch on
#                      path" when it does not.
#   🔀 each pull request  its title, its state with the state's glyph, its age, its checks (how many passed and how
#                      long they took, or which failed), and its unresolved review threads when there are any; under
#                      it the issues it closes and the worktree that holds its branch, when one here does.
#
# Then every local branch whose upstream is gone, which kit tidy removes, each on a 🔖 line.
#
# Color is by family, never by state: green when nothing is left to decide, yellow when a human owes a decision,
# blue when the next move is a machine's or a reviewer's, dim when someone already owns the work. Everything that
# is detail rather than signal is dim: hashes, ages, durations, zero counts, directory prefixes, and the lines under
# a row. The state is the first of these that holds:
#
#   draft              🔨  dim     unfinished work that belongs to someone
#   queued             🚦  blue    a check is still running
#   conflicted         💥  yellow  the branch does not apply cleanly to the default branch
#   failing:gate       🔥  yellow  a check failed; the row names which
#   no-check           📭  blue    nothing has reported on its head
#   comments:open      💬  yellow  unresolved review threads
#   blocked:approval   👀  blue    a review is required or requested and has not arrived
#   ready              🛬  green   green, thread-free, approved, and mergeable
#
# --json prints the same facts as one document instead: {desk, worktrees, pullRequests, goneBranches}, every
# measure a number and every list complete, for a consumer that draws its own picture.

use git

# status_commit_cap is how many commits a block lists before counting the rest.
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

# status:age <seconds> prints a duration in its coarsest informative unit: 45s, 12m, 3h, 47d.
status:age() {
  local secs=$1
  (( secs < 0 )) && secs=0
  if (( secs < 120 ))
  then printf '%ds' "${secs}"
  elif (( secs < 7200 ))
  then printf '%dm' $(( secs / 60 ))
  elif (( secs < 172800 ))
  then printf '%dh' $(( secs / 3600 ))
  else printf '%dd' $(( secs / 86400 ))
  fi
}

# status:collect prints one JSON object per open pull request, oldest first: number, branch, title, createdAt, age
# in seconds, state, checks {state, passed, total, failing, took in seconds or null}, unresolved, and closes
# [{number, title}]. The one request to GitHub.
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
    | (if $started == null then null
       elif $check == "PENDING" then $now - $started
       elif $completed == null then null
       else $completed - $started end) as $took
    | ([.reviewThreads.nodes[]? | select(.isResolved | not)] | length) as $unresolved
    | (if .isDraft then "draft"
       elif $check == "PENDING" then "queued"
       elif .mergeStateStatus == "DIRTY" then "conflicted"
       elif $check == "FAILURE" then "failing:gate"
       elif $check == "NONE" then "no-check"
       elif $unresolved > 0 then "comments:open"
       elif (.reviewDecision // "") | IN("REVIEW_REQUIRED", "CHANGES_REQUESTED") then "blocked:approval"
       else "ready" end) as $state
    | { number, branch: .headRefName, title, createdAt, age: ($now - (.createdAt | fromdateiso8601)), state: $state,
        checks: { state: $check, passed: $ok, total: $n, failing: $failing, took: $took },
        unresolved: $unresolved,
        closes: [.closingIssuesReferences.nodes[] | { number, title }] }
    | tojson'
  gh api graphql --field owner="${repo%%/*}" --field name="${repo#*/}" --raw-field query="${query}" --jq "${filter}" \
    2>/dev/null || true
}

# status:fork <branch> <default-branch> prints the commit where <branch> left the default branch, or nothing.
status:fork() {
  git merge-base "origin/$2" "$1" 2>/dev/null || git merge-base "$2" "$1" 2>/dev/null || true
}

# status:relative <path> prints <path> relative to the current directory: ".", a descent, or a climb.
status:relative() {
  local target=$1 here=${PWD} up=''
  if [[ "${target}" == "${here}" ]]
  then
    printf '.\n'
    return 0
  fi
  while [[ -n "${here}" && "${target}" != "${here}" && "${target}" != "${here}"/* ]]
  do
    here=${here%/*}
    up+='../'
  done
  if [[ "${target}" == "${here}" ]]
  then
    printf '%s\n' "${up%/}"
  else
    printf '%s%s\n' "${up}" "${target#"${here}"/}"
  fi
}

# status:checkout <path> <branch> <default-branch> prints one JSON object measuring the checkout at <path>: path,
# relative, branch (empty when detached), detached, ahead of and behind the point where the branch left origin's
# default branch, the lines the working tree adds and removes against that point, the count of uncommitted paths,
# and the commits past that point as [{sha, subject}]. A detached checkout measures its lines against its own head
# and lists no commits.
status:checkout() {
  local path=$1 branch=$2 default_branch=$3 fork='' ahead=0 behind=0 added=0 removed=0 uncommitted a r _
  [[ -n "${branch}" ]] && fork=$(status:fork "${branch}" "${default_branch}")
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
  uncommitted=$(git -C "${path}" status --porcelain | grep --count . || true)
  {
    [[ -n "${fork}" ]] && git -C "${path}" log --format='%h%x09%s' "${fork}..${branch}" 2>/dev/null
    true
  } | jq --raw-input --slurp --compact-output \
    --arg path "${path}" --arg relative "$(status:relative "${path}")" --arg branch "${branch}" \
    --argjson ahead "${ahead}" --argjson behind "${behind}" --argjson added "${added}" --argjson removed "${removed}" \
    --argjson uncommitted "${uncommitted}" '
    { path: $path, relative: $relative, branch: $branch, detached: ($branch == ""),
      ahead: $ahead, behind: $behind, added: $added, removed: $removed, uncommitted: $uncommitted,
      commits: [split("\n")[] | select(. != "") | split("\t") | { sha: .[0], subject: (.[1:] | join("\t")) }] }'
}

# status:label <relative> <branch> prints the label of a checkout and its plain width, tab-separated: the path with
# the directory dim and the branch underlined when the path's last component is the branch or ends in "-<branch>",
# the branch alone at ".", or "branch on path" otherwise. A detached checkout is its path.
status:label() {
  local relative=$1 branch=$2 dir
  if [[ -z "${branch}" ]]
  then
    printf '%s\t%s\n' "${relative}" "${#relative}"
  elif [[ "${relative}" == "." ]]
  then
    printf '%s\t%s\n' "${branch}" "${#branch}"
  elif [[ "${relative##*/}" == "${branch}" || "${relative##*/}" == *"-${branch}" ]]
  then
    dir=${relative%"${branch}"}
    printf '%s%s\t%s\n' "$(status:paint 90 "${dir}")" "$(status:paint 4 "${branch}")" "${#relative}"
  else
    printf '%s %s %s\t%s\n' "${branch}" "$(status:paint 90 on)" "${relative}" $(( ${#branch} + 4 + ${#relative} ))
  fi
}

# status:paint-count <count> <sign> <sgr> prints <sign><count>, in <sgr> above zero and dim at zero.
status:paint-count() {
  local count=$1 sign=$2 sgr=$3
  (( count > 0 )) || sgr=90
  status:paint "${sgr}" "${sign}${count}"
}

# status:measures <checkout-json> prints "↑a ↓b · +x -y", and " · n uncommitted" in yellow when n is above zero.
status:measures() {
  local ahead behind added removed uncommitted
  IFS=$'\t' read -r ahead behind added removed uncommitted \
    < <(jq --raw-output '[.ahead, .behind, .added, .removed, .uncommitted] | @tsv' <<<"$1")
  printf '%s %s · %s %s' "$(status:paint-count "${ahead}" '↑' '1;32')" "$(status:paint-count "${behind}" '↓' '1;31')" \
    "$(status:paint-count "${added}" '+' '1;32')" "$(status:paint-count "${removed}" '-' '1;31')"
  (( uncommitted > 0 )) && printf ' · %s' "$(status:paint 33 "${uncommitted} uncommitted")"
  return 0
}

# status:checkout-block <glyph> <checkout-json> <label-width> prints a checkout: its row, then its commits, dim
# hash and subject. Past status_commit_cap it prints the newest, a dim count of the hidden, and the oldest.
status:checkout-block() {
  local glyph=$1 doc=$2 label_w=$3 relative branch label width mark='' sha subject
  IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
  IFS=$'\t' read -r label width < <(status:label "${relative}" "${branch}")
  [[ -n "${branch}" ]] || mark="   $(status:paint 33 '❗ detached HEAD')"
  printf '%s %s%*s%s   %s\n' "${glyph}" "${label}" $(( label_w - width )) '' "${mark}" "$(status:measures "${doc}")" >&2
  while IFS=$'\t' read -r sha subject
  do
    [[ -n "${sha}" ]] || continue
    if [[ "${sha}" == '…' ]]
    then
      printf '   %s\n' "$(status:paint 90 "… ${subject}")" >&2
    else
      printf '   %s %s\n' "$(status:paint 90 "${sha}")" "${subject}" >&2
    fi
  done < <(jq --raw-output --argjson cap "${status_commit_cap}" '
    (.commits | length) as $n
    | (if $n > $cap + 1
       then (.commits[:$cap - 1][] | [.sha, .subject]), ["…", "and \($n - $cap) more"], (.commits[-1] | [.sha, .subject])
       else .commits[] | [.sha, .subject] end)
    | @tsv' <<<"${doc}")
}

# status:check-cell <checks-json> prints the checks as "✅ <passed> / <total> · <took>", the duration dim, or
# "❌ <failing> · <passed> / <total> · <took>", or "🏃 <passed> / <total> · <elapsed>"; nothing when none has
# reported.
status:check-cell() {
  local state passed total failing took text
  # A field here can be empty, and a tab read collapses an empty field, so the separator is one that cannot.
  IFS=$'\x1f' read -r state passed total failing took \
    < <(jq --raw-output '[.state, .passed, .total, .failing, (.took // "")] | map(tostring) | join("\u001f")' <<<"$1")
  case "${state}" in
    (SUCCESS) text="✅ ${total} / ${total}" ;;
    (FAILURE) text="❌ ${failing} · ${passed} / ${total}" ;;
    (PENDING) text="🏃 ${passed} / ${total}" ;;
    (*) return 0 ;;
  esac
  printf '%s' "${text}"
  [[ -n "${took}" ]] && printf ' · %s' "$(status:paint 90 "$(status:age "${took}")")"
  return 0
}

# status:pr-block <pr-json> <title-width> <state-width> <age-width> <worktree-relative> prints a pull request: its
# row, the issues it closes, and the worktree here that holds its branch, when one does.
status:pr-block() {
  local doc=$1 title_w=$2 state_w=$3 age_w=$4 worktree=$5 number title state age unresolved row issue checks
  IFS=$'\t' read -r number title state age unresolved \
    < <(jq --raw-output '[.number, .title, .state, .age, .unresolved] | @tsv' <<<"${doc}")
  row="🔀 $(status:cell "#${number} ${title}" "${title_w}")"
  row+="   $(status:state-emoji "${state}") $(status:paint "$(status:state-sgr "${state}")" "$(status:cell "${state}" "${state_w}")")"
  row+="   $(status:paint 90 "$(status:cell "$(status:age "${age}")" "${age_w}")")"
  checks=$(status:check-cell "$(jq --compact-output .checks <<<"${doc}")")
  [[ -n "${checks}" ]] && row+="   ${checks}"
  (( unresolved > 0 )) && row+="   $(status:paint 33 "💬 ${unresolved} unresolved")"
  printf '%s\n' "${row}" >&2
  while IFS= read -r issue
  do
    [[ -n "${issue}" ]] && printf '   %s\n' "$(status:paint 90 "closes ${issue}")" >&2
  done < <(jq --raw-output '.closes[] | "#\(.number) \(.title)"' <<<"${doc}")
  [[ -n "${worktree}" ]] && printf '   %s\n' "$(status:paint 90 "🌿 ${worktree}")" >&2
  return 0
}

status:render() {
  local no_fetch=false json=false arg main_root default_branch desk_branch path branch doc gone
  for arg in "$@"
  do
    case "${arg}" in
      (--no-fetch) no_fetch=true ;;
      (--json) json=true ;;
      (*)
        error "unknown flag '${arg}' (usage: kit status [--no-fetch] [--json])"
        return 2
    esac
  done
  if [[ "${no_fetch}" != true ]]
  then
    git fetch --quiet --prune origin || warn "git fetch --prune failed; ahead and behind counts may be stale"
  fi
  main_root=$(git:main-root)
  default_branch=$(git:default-branch)
  desk_branch=$(git -C "${main_root}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  gone=$(git:gone-branches)

  # The desk, the worktrees by issue number, and the pull requests, each as JSON.
  local desk prs
  local -a worktrees=()
  desk=$(status:checkout "${main_root}" "${desk_branch}" "${default_branch}")
  while IFS=$'\t' read -r path branch _
  do
    [[ -n "${path}" ]] || continue
    worktrees+=("$(status:checkout "${path}" "${branch}" "${default_branch}")")
  done < <(git:worktrees | status:by-number)
  prs=$(status:collect)

  if [[ "${json}" == true ]]
  then
    jq --null-input --argjson desk "${desk}" \
      --argjson worktrees "$(printf '%s\n' "${worktrees[@]}" | jq --slurp .)" \
      --argjson pullRequests "$(jq --slurp . <<<"${prs}")" \
      --argjson goneBranches "$(jq --raw-input . <<<"${gone}" | jq --slurp 'map(select(. != ""))')" \
      '{ desk: $desk, worktrees: $worktrees, pullRequests: $pullRequests, goneBranches: $goneBranches }'
    return 0
  fi

  # Column widths across the rows that share a column.
  local label_w=0 width title_w=0 state_w=0 age_w=0 text state_text age_secs relative
  local -A worktree_on=()
  for doc in "${desk}" "${worktrees[@]}"
  do
    IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
    IFS=$'\t' read -r _ width < <(status:label "${relative}" "${branch}")
    (( width > label_w )) && label_w=${width}
    [[ -n "${branch}" && "${doc}" != "${desk}" ]] && worktree_on[${branch}]=${relative}
  done
  while IFS=$'\t' read -r text state_text age_secs
  do
    [[ -n "${text}" ]] || continue
    (( ${#text} > title_w )) && title_w=${#text}
    (( ${#state_text} > state_w )) && state_w=${#state_text}
    text=$(status:age "${age_secs}")
    (( ${#text} > age_w )) && age_w=${#text}
  done < <(jq --raw-output '["#\(.number) \(.title)", .state, .age] | @tsv' <<<"${prs}")
  width=$(( $(status:width) - 3 - 3 - 3 - state_w - 3 - age_w - 3 - 24 ))
  (( width < 20 )) && width=20
  (( title_w > width )) && title_w=${width}

  status:checkout-block '🏠' "${desk}" "${label_w}"
  for doc in "${worktrees[@]}"
  do
    printf '\n' >&2
    status:checkout-block '🌿' "${doc}" "${label_w}"
  done
  while IFS= read -r doc
  do
    [[ -n "${doc}" ]] || continue
    printf '\n' >&2
    branch=$(jq --raw-output .branch <<<"${doc}")
    status:pr-block "${doc}" "${title_w}" "${state_w}" "${age_w}" "${worktree_on[${branch}]:-}"
  done <<<"${prs}"
  if [[ -n "${gone}" ]]
  then
    printf '\n' >&2
    while IFS= read -r branch
    do
      printf '🔖 %s   %s\n' "${branch}" "$(status:paint 90 'upstream gone · kit tidy removes it')" >&2
    done <<<"${gone}"
  fi
}

# status:by-number sorts git:worktrees lines by the issue number their directory opens with; a directory that
# opens with no number sorts first.
status:by-number() {
  local path rest dir number
  while IFS=$'\t' read -r path rest
  do
    [[ -n "${path}" ]] || continue
    dir=${path##*/}
    number=${dir%%-*}
    [[ "${number}" =~ ^[0-9]+$ ]] || number=0
    printf '%s\t%s\t%s\n' "${number}" "${path}" "${rest}"
  done | sort -n -k1,1 | cut -f2-
}

# ----------------------------------------------------------------------------------------------------------------------
