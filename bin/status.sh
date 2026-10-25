# shellcheck shell=bash
# status.sh - the picture of the work in flight (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# status:render [--no-fetch] [--json] prints the checkouts and the open pull requests, and changes nothing. It asks
# GitHub once, for every open pull request with the issues it closes, its review threads, and the checks on its
# head; everything else is a local git read.
#
# The picture: the desk, the root checkout, first; then each worktree under worktrees/; then each open pull request
# with its state, age, checks, the issues it closes, and the worktree holding its branch; then the branches whose
# upstream is gone. Color is by family, never by state: green when nothing is left to decide, yellow when a human
# owes a decision, blue when the next move is a machine's or a reviewer's, dim for detail. The states, in the
# order the first that holds wins:
#
#   draft 🔨, queued 🚦, conflicted 💥, failing:gate 🔥, no-check 📭, comments:open 💬, blocked:approval 👀,
#   ready 🛬
#
# --json prints the same facts as one document: {desk, worktrees, pullRequests, goneBranches}.

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
# [{number, title}]. The one request to GitHub; a request that fails is said so, and prints nothing.
status:collect() {
  local repo query filter
  use gh
  if ! repo=$(gh:repo)
  then
    warn "this checkout does not resolve to a GitHub repository, so no pull request is shown (run: gh repo view)"
    return 0
  fi
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
    2>/dev/null || warn "the pull requests could not be read from GitHub (run: gh auth status); none is shown"
}

# status:fork <branch> <default-branch> prints the commit where <branch> left the default branch, or nothing.
status:fork() {
  git merge-base "origin/$2" "$1" 2>/dev/null || git merge-base "$2" "$1" 2>/dev/null || true
}

# status:relative <path> prints <path> relative to the directory the command was typed in: ".", a descent, or a
# climb. Both sides are resolved physically, the target through its nearest directory that exists, so a symlinked
# directory on either side still climbs correctly.
status:relative() {
  local target=$1 here=${kit_invoked_in:-$(pwd -P)} up='' dir rest=''
  dir=${target}
  while [[ -n "${dir}" && ! -d "${dir}" ]]
  do
    rest="${dir##*/}${rest:+/${rest}}"
    dir=${dir%/*}
  done
  dir=$(cd "${dir:-/}" && pwd -P)
  target="${dir}${rest:+/${rest}}"
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

# status:paint-count <count> <sign> <sgr> <width> <side> prints <sign><count> padded to <width> characters on the
# left or the right, in <sgr> above zero and dim at zero. The padding is by character, since the arrows are
# multibyte and printf's field width counts bytes.
status:paint-count() {
  local count=$1 sign=$2 sgr=$3 width=$4 side=$5 text pad=''
  (( count > 0 )) || sgr=90
  text="${sign}${count}"
  (( ${#text} < width )) && printf -v pad '%*s' $(( width - ${#text} )) ''
  if [[ "${side}" == right ]]
  then
    printf '%s%s' "${pad}" "$(status:paint "${sgr}" "${text}")"
  else
    printf '%s%s' "$(status:paint "${sgr}" "${text}")" "${pad}"
  fi
}

# status:measures <checkout-json> [<widths>] prints "↑a ↓b · +x -y", and " · n uncommitted" in yellow when n is
# above zero. <widths> is status:count-widths' line; with it, ahead and added are right-aligned and behind and
# removed left-aligned in columns of those widths, so the dots of every row fall under each other.
status:measures() {
  local ahead behind added removed uncommitted ahead_w=0 behind_w=0 added_w=0 removed_w=0
  IFS=$'\t' read -r ahead behind added removed uncommitted \
    < <(jq --raw-output '[.ahead, .behind, .added, .removed, .uncommitted] | @tsv' <<<"$1")
  [[ -n "${2:-}" ]] && IFS=' ' read -r ahead_w behind_w added_w removed_w <<<"$2"
  # The last column is padded only when something follows it, so no row ends in blanks.
  (( uncommitted > 0 )) || removed_w=0
  printf '%s %s · %s %s' \
    "$(status:paint-count "${ahead}" '↑' '1;32' "${ahead_w}" right)" \
    "$(status:paint-count "${behind}" '↓' '1;31' "${behind_w}" left)" \
    "$(status:paint-count "${added}" '+' '1;32' "${added_w}" right)" \
    "$(status:paint-count "${removed}" '-' '1;31' "${removed_w}" left)"
  (( uncommitted > 0 )) && printf ' · %s' "$(status:paint 33 "${uncommitted} uncommitted")"
  return 0
}

# status:count-widths <checkout-json>… prints the widest "↑a", "↓b", "+x", and "-y" across the checkouts, as four
# numbers on one line.
status:count-widths() {
  printf '%s\n' "$@" | jq --raw-output --slurp '
    [ (map(.ahead) | max), (map(.behind) | max), (map(.added) | max), (map(.removed) | max) ]
    | map(tostring | length + 1) | join(" ")'
}

# status:checkout-block <glyph> <checkout-json> <label-width> [<widths>] prints a checkout: its row, with the
# counts in status:count-widths' columns, then its commits, dim hash and subject. Past status_commit_cap it prints
# the newest, a dim count of the hidden, and the oldest.
status:checkout-block() {
  local glyph=$1 doc=$2 label_w=$3 widths=${4:-} relative branch label width mark='' sha subject
  IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
  IFS=$'\t' read -r label width < <(status:label "${relative}" "${branch}")
  [[ -n "${branch}" ]] || mark="   $(status:paint 33 '❗ detached HEAD')"
  printf '%s %s%*s%s   %s\n' "${glyph}" "${label}" $(( label_w - width )) '' "${mark}" \
    "$(status:measures "${doc}" "${widths}")" >&2
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
       then (.commits[:$cap - 1][] | [.sha, .subject]), ["…", "and \($n - $cap) more"],
            (.commits[-1] | [.sha, .subject])
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
  row+="   $(status:state-emoji "${state}")"
  row+=" $(status:paint "$(status:state-sgr "${state}")" "$(status:cell "${state}" "${state_w}")")"
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

# status:render [--no-fetch] [--json] draws the picture the file header describes, or prints it as one JSON
# document.
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
  while IFS=$'\t' read -r path _ branch
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

  local widths
  widths=$(status:count-widths "${desk}" "${worktrees[@]}")
  status:checkout-block '🏠' "${desk}" "${label_w}" "${widths}"
  for doc in "${worktrees[@]}"
  do
    printf '\n' >&2
    status:checkout-block '🌿' "${doc}" "${label_w}" "${widths}"
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
