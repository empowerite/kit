# shellcheck shell=bash
# status.sh - shows the checkouts and the open pull requests in progress (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# status:render [--no-fetch] [--json] [--mine | --all | --owner=<login>…] prints the checkouts and the open pull
# requests. It changes nothing.
#
# It calls the forge (GitHub or GitLab) once, for every open pull request with the issues it closes, its review
# threads, and the checks on its head commit. Everything else is read from local git. The forge call starts
# first, in the background, so it runs while the fetch runs and the checkouts are drawn.
#
# The output, in order:
#
#   1. what the fetch brought in, unless --no-fetch
#   2. the root checkout (the desk)
#   3. each worktree under worktrees/, with its uncommitted paths and then its commits
#   4. each open pull request, with its author, state, age, checks, the worktree that has its branch checked out,
#      and the issues it closes
#   5. the branches whose upstream is gone
#
# Which pull requests are shown:
#
#   --all              every open pull request (the default)
#   --mine             the pull requests the caller opened, without the author column
#   --owner=<login>    the pull requests the named author opened; repeat the flag for more authors
#
# Every choice except --mine shows the author before the state.
#
# A pull request's state is the first of these that applies. Its color shows the state's family, never the
# single state: green when nothing is left to decide, yellow when a human owes a decision, and blue when the next
# move is a machine's or a reviewer's. Detail is dim.
#
#   state              glyph   color
#   draft              🔨      dim
#   queued             🚦      blue
#   conflicted         💥      yellow
#   failing:gate       🔥      yellow
#   no-check           📭      blue
#   comments:open      💬      yellow
#   blocked:approval   👀      blue
#   ready              🛬      green
#
# --json prints the same data as one JSON document, {desk, worktrees, pullRequests, goneBranches}, with the pull
# requests chosen the same way.

use git

# status_commit_cap is how many uncommitted paths or commits a checkout block lists before it counts the rest.
status_commit_cap=10

# status:width prints the terminal's width in columns: COLUMNS, else the width of the terminal on stderr, else
# 100. A width under 60 also gives 100.
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

# status:state-sgr <state> prints the color code of a state's family.
status:state-sgr() {
  case "$1" in
    (ready)                                     printf '32' ;;
    (conflicted | failing:gate | comments:open) printf '33' ;;
    (queued | no-check | blocked:approval)      printf '34' ;;
    (*)                                         printf '90' ;;
  esac
}

# status:state-emoji <state> prints a state's glyph. Each glyph is one code point drawn two columns wide, so the
# next column stays aligned.
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

# status:refs prints "<ref> <sha>" for origin's branches and for the tags: the refs a fetch moves.
status:refs() {
  git for-each-ref --format='%(refname) %(objectname)' refs/remotes/origin refs/tags | grep --invert-match '/HEAD '
  return 0
}

# status:plural <count> <singular> <plural> prints "<count> <singular>" or "<count> <plural>".
status:plural() {
  if (( $1 == 1 ))
  then printf '1 %s' "$2"
  else printf '%s %s' "$1" "$3"
  fi
}

# status:fetch-summary <before> <after> <default-branch> prints one line listing what changed between two
# status:refs snapshots: new commits on origin's default branch, then branches updated, new, and pruned, then new
# tags. It prints "nothing new" when nothing changed.
status:fetch-summary() {
  local default_ref="refs/remotes/origin/$3" kind ref old new commits=0 updated=0 added=0 pruned=0 tags=0 line part
  local -a parts=()
  while IFS=' ' read -r kind ref old new
  do
    case "${kind}:${ref}" in
      (moved:"${default_ref}") commits=$(git rev-list --count "${old}..${new}" 2>/dev/null || printf 0) ;;
      (moved:refs/remotes/*) updated=$(( updated + 1 )) ;;
      (new:refs/remotes/*) added=$(( added + 1 )) ;;
      (gone:refs/remotes/*) pruned=$(( pruned + 1 )) ;;
      (new:refs/tags/*) tags=$(( tags + 1 )) ;;
    esac
  done < <(awk 'NR == FNR { old[$1] = $2; next }
                !($1 in old) { print "new", $1, "-", $2; next }
                old[$1] != $2 { print "moved", $1, old[$1], $2 }
                { delete old[$1] }
                END { for (ref in old) print "gone", ref, old[ref], "-" }' \
             <(printf '%s\n' "$1" | grep .) <(printf '%s\n' "$2" | grep .))
  (( commits > 0 )) && parts+=("$(status:plural "${commits}" 'new commit' 'new commits') on $3")
  (( updated > 0 )) && parts+=("$(status:plural "${updated}" 'branch updated' 'branches updated')")
  (( added > 0 )) && parts+=("$(status:plural "${added}" 'new branch' 'new branches')")
  (( pruned > 0 )) && parts+=("$(status:plural "${pruned}" 'branch pruned' 'branches pruned')")
  (( tags > 0 )) && parts+=("$(status:plural "${tags}" 'new tag' 'new tags')")
  (( ${#parts[@]} > 0 )) || parts=('nothing new')
  line=${parts[0]}
  for part in "${parts[@]:1}"
  do
    line+=" · ${part}"
  done
  printf '📥 fetched origin: %s\n' "${line}"
}

# status:fetch <default-branch> fetches origin with --prune, then prints status:fetch-summary's line and a blank
# line. When the fetch fails, it prints a warning instead of the summary.
status:fetch() {
  local before
  before=$(status:refs)
  if git fetch --quiet --prune origin
  then
    status:fetch-summary "${before}" "$(status:refs)" "$1" >&2
  else
    warn "git fetch --prune failed; ahead and behind counts may be stale"
  fi
  printf '\n' >&2
}

# status:collect prints one JSON object per open pull request, as the forge provider builds them. When the forge
# cannot be reached, it prints nothing and a warning.
status:collect() {
  use forge
  forge:open-prs || warn "no pull request is shown; see above"
}

# status:select <login>… reads pull request documents on stdin and keeps those opened by one of the named
# authors. With no login, it keeps every one.
status:select() {
  if (( $# == 0 ))
  then cat
  else jq --compact-output --arg logins "$*" 'select(.author as $a | ($logins | split(" ")) | index($a))'
  fi
}

# status:fork <branch> <default-branch> prints the commit where <branch> left the default branch, or nothing.
status:fork() {
  git merge-base "origin/$2" "$1" 2>/dev/null || git merge-base "$2" "$1" 2>/dev/null || true
}

# status:relative <path> prints <path> relative to the directory the command was run from: ".", a path down, or
# a path up through ../. Both paths are resolved physically (the target through its nearest existing directory),
# so a symlinked directory on either side still gives the right ../ path.
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

# status:checkout <path> <branch> <default-branch> prints one JSON object that measures the checkout at <path>.
# The fork point is where the branch left origin's default branch. The fields:
#
#   path, relative      the checkout's path, and that path relative to the caller's directory
#   branch, detached    the branch (empty when detached), and whether the checkout is detached
#   ahead, behind       the commits on the branch past the fork point, and on origin's default branch past it
#   added, removed      the lines the working tree adds and removes against the fork point
#   uncommitted         the number of uncommitted paths
#   changes             each uncommitted path as {status, path}, with git's two-letter status
#   commits             the commits past the fork point, as [{sha, subject}]
#
# A detached checkout measures its lines against its own HEAD and lists no commits.
status:checkout() {
  local path=$1 branch=$2 default_branch=$3 fork='' ahead=0 behind=0 added=0 removed=0 changes a r _
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
  changes=$(git -C "${path}" status --porcelain 2>/dev/null || true)
  {
    [[ -n "${fork}" ]] && git -C "${path}" log --format='%h%x09%s' "${fork}..${branch}" 2>/dev/null
    true
  } | jq --raw-input --slurp --compact-output \
    --arg path "${path}" --arg relative "$(status:relative "${path}")" --arg branch "${branch}" \
    --argjson ahead "${ahead}" --argjson behind "${behind}" --argjson added "${added}" --argjson removed "${removed}" \
    --arg changes "${changes}" '
    [$changes | split("\n")[] | select(. != "") | { status: .[0:2], path: .[3:] }] as $changes
    | { path: $path, relative: $relative, branch: $branch, detached: ($branch == ""),
        ahead: $ahead, behind: $behind, added: $added, removed: $removed, uncommitted: ($changes | length),
        changes: $changes,
        commits: [split("\n")[] | select(. != "") | split("\t") | { sha: .[0], subject: (.[1:] | join("\t")) }] }'
}

# status:label <relative> <branch> prints a checkout's label and its plain width, tab-separated. The label is:
#
#   - for a detached checkout: its path
#   - at ".": the branch alone
#   - when the path's last component is the branch or ends in "-<branch>": the path, with the directory dim and
#     the branch underlined
#   - otherwise: "<branch> on <path>"
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

# status:paint-count <count> <sign> <sgr> <width> <side> prints <sign><count> padded to <width> characters,
# aligned to <side> (left or right). The text is in <sgr> above zero and dim at zero. It pads by characters,
# because the arrows are multibyte and printf's field width counts bytes.
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

# status:measures <checkout-json> [<widths>] prints "↑a ↓b · +x -y", then " · n uncommitted" in yellow when n is
# above zero. <widths> is status:count-widths' line. With it, ahead and added are right-aligned, and behind and
# removed left-aligned, in columns of those widths, so the dots line up across rows.
status:measures() {
  local ahead behind added removed uncommitted ahead_w=0 behind_w=0 added_w=0 removed_w=0
  IFS=$'\t' read -r ahead behind added removed uncommitted \
    < <(jq --raw-output '[.ahead, .behind, .added, .removed, .uncommitted] | @tsv' <<<"$1")
  [[ -n "${2:-}" ]] && IFS=' ' read -r ahead_w behind_w added_w removed_w <<<"$2"
  # The last column is padded only when something follows it, so no row ends in trailing spaces.
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

# status:checkout-block <glyph> <checkout-json> <label-width> [<widths>] prints a checkout:
#
#   1. its row, with the counts in status:count-widths' columns
#   2. its uncommitted paths, each with its status in yellow
#   3. its commits, each with its hash dim
#
# Past status_commit_cap, it prints the first paths and a count of the rest. For commits, it prints the newest
# ones, a dim count of the hidden ones, and the oldest one.
status:checkout-block() {
  local glyph=$1 doc=$2 label_w=$3 widths=${4:-} relative branch label width mark='' sha subject code path
  IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
  IFS=$'\t' read -r label width < <(status:label "${relative}" "${branch}")
  [[ -n "${branch}" ]] || mark="   $(status:paint 33 '❗ detached HEAD')"
  printf '%s %s%*s%s   %s\n' "${glyph}" "${label}" $(( label_w - width )) '' "${mark}" \
    "$(status:measures "${doc}" "${widths}")" >&2
  # A status can start with a space, which a tab-separated read would drop, so the separator is \x1f instead.
  while IFS=$'\x1f' read -r code path
  do
    [[ -n "${path}" ]] || continue
    if [[ "${code}" == '…' ]]
    then
      printf '   %s\n' "$(status:paint 90 "… ${path}")" >&2
    else
      printf '   %s %s\n' "$(status:paint 33 "${code}")" "${path}" >&2
    fi
  done < <(jq --raw-output --argjson cap "${status_commit_cap}" '
    (.changes // []) as $c | ($c | length) as $n
    | (if $n > $cap + 1
       then ($c[:$cap][] | [.status, .path]), ["…", "and \($n - $cap) more"]
       else $c[] | [.status, .path] end)
    | join("\u001f")' <<<"${doc}")
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

# status:check-cell <checks-json> prints the checks cell, with the duration dim:
#
#   - passed: "✅ <passed> / <total> · <took>"
#   - failed: "❌ <failing> · <passed> / <total> · <took>"
#   - running: "🏃 <passed> / <total> · <elapsed>"
#
# It prints nothing when no check has reported.
status:check-cell() {
  local state passed total failing took text
  # A field can be empty, and a tab-separated read collapses an empty field, so the separator is \x1f instead.
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

# status:pr-block <pr-json> <title-width> <owner-width> <state-width> <age-width> <worktree> prints a pull request:
# its row, then the issues it closes. The row shows the author before the state when <owner-width> is above zero,
# and ends with the branch when a worktree here has it checked out.
status:pr-block() {
  local doc=$1 title_w=$2 owner_w=$3 state_w=$4 age_w=$5 worktree=$6
  local number ref title author branch state age unresolved row issue checks
  IFS=$'\x1f' read -r number ref title author branch state age unresolved \
    < <(jq --raw-output '[.number, (.ref // "#\(.number)"), .title, .author, .branch, .state, .age, .unresolved]
          | map(tostring)
          | join("\u001f")' <<<"${doc}")
  row="🔀 $(status:cell "${ref} ${title}" "${title_w}")"
  (( owner_w > 0 )) && row+="   👤 $(status:paint 90 "$(status:cell "${author}" "${owner_w}")")"
  row+="   $(status:state-emoji "${state}")"
  row+=" $(status:paint "$(status:state-sgr "${state}")" "$(status:cell "${state}" "${state_w}")")"
  row+="   $(status:paint 90 "$(status:cell "$(status:age "${age}")" "${age_w}")")"
  checks=$(status:check-cell "$(jq --compact-output .checks <<<"${doc}")")
  [[ -n "${checks}" ]] && row+="   ${checks}"
  (( unresolved > 0 )) && row+="   $(status:paint 33 "💬 ${unresolved} unresolved")"
  [[ -n "${worktree}" ]] && row+="   $(status:paint 90 "🌿 ${branch}")"
  printf '%s\n' "${row}" >&2
  while IFS= read -r issue
  do
    [[ -n "${issue}" ]] && printf '   %s\n' "$(status:paint 90 "closes ${issue}")" >&2
  done < <(jq --raw-output '.closes[] | "#\(.number) \(.title)"' <<<"${doc}")
  return 0
}

# status:render [--no-fetch] [--json] [--mine | --all | --owner=<login>…] prints the output the file header
# describes, or the same data as one JSON document.
status:render() {
  local no_fetch=false json=false choice=all arg main_root default_branch desk_branch path branch doc gone
  local -a owners=()
  for arg in "$@"
  do
    case "${arg}" in
      (--no-fetch) no_fetch=true ;;
      (--json) json=true ;;
      (--mine) choice=mine ;;
      (--all) choice=all ;;
      (--owner=?*) choice=owner; owners+=("${arg#--owner=}") ;;
      (*)
        error "unknown flag '${arg}' (usage: kit status [--no-fetch] [--json] [--mine | --all | --owner=<login>…])"
        return 2
    esac
  done
  # The forge call is the slow one and needs nothing local, so it starts first and runs during the fetch and the
  # local reads. Its warnings are saved and printed where the pull requests appear, so they do not land among the
  # checkouts. With --mine, the same background job also asks the forge for the caller's login.
  local scratch prs_job prs_failed=false
  scratch=$(mktemp -d "${TMPDIR:-/tmp}/kit-status.XXXXXX")
  # shellcheck disable=SC2064  # the path is fixed now; the trap outlives this function's locals
  trap "rm -rf '${scratch}'" EXIT
  (
    if [[ "${choice}" == mine ]]
    then
      use forge
      owners=("$(forge:me)") || exit 1
    fi
    status:collect | status:select "${owners[@]}"
  ) > "${scratch}/prs" 2> "${scratch}/prs.err" &
  prs_job=$!

  main_root=$(git:main-root)
  default_branch=$(git:default-branch)
  if [[ "${no_fetch}" == true ]]
  then
    true
  elif [[ "${json}" == true ]]
  then
    status:fetch "${default_branch}" 2>/dev/null
  else
    status:fetch "${default_branch}"
  fi
  desk_branch=$(git -C "${main_root}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  gone=$(git:gone-branches)

  # Measure the desk, then the worktrees in issue-number order, each as JSON.
  local desk prs
  local -a worktrees=()
  desk=$(status:checkout "${main_root}" "${desk_branch}" "${default_branch}")
  while IFS=$'\t' read -r path _ branch
  do
    [[ -n "${path}" ]] || continue
    worktrees+=("$(status:checkout "${path}" "${branch}" "${default_branch}")")
  done < <(git:worktrees | status:by-number)

  if [[ "${json}" == true ]]
  then
    wait "${prs_job}" || prs_failed=true
    cat "${scratch}/prs.err" >&2
    [[ "${prs_failed}" == false ]] || return 1
    prs=$(<"${scratch}/prs")
    jq --null-input --argjson desk "${desk}" \
      --argjson worktrees "$(printf '%s\n' "${worktrees[@]}" | jq --slurp .)" \
      --argjson pullRequests "$(jq --slurp . <<<"${prs}")" \
      --argjson goneBranches "$(jq --raw-input . <<<"${gone}" | jq --slurp 'map(select(. != ""))')" \
      '{ desk: $desk, worktrees: $worktrees, pullRequests: $pullRequests, goneBranches: $goneBranches }'
    return 0
  fi

  # Draw the checkouts while the forge call may still be running.
  local label_w=0 width relative widths
  local -A worktree_on=()
  for doc in "${desk}" "${worktrees[@]}"
  do
    IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
    IFS=$'\t' read -r _ width < <(status:label "${relative}" "${branch}")
    (( width > label_w )) && label_w=${width}
    [[ -n "${branch}" && "${doc}" != "${desk}" ]] && worktree_on[${branch}]=${relative}
  done
  widths=$(status:count-widths "${desk}" "${worktrees[@]}")
  status:checkout-block '🏠' "${desk}" "${label_w}" "${widths}"
  for doc in "${worktrees[@]}"
  do
    printf '\n' >&2
    status:checkout-block '🌿' "${doc}" "${label_w}" "${widths}"
  done

  wait "${prs_job}" || prs_failed=true
  if [[ -s "${scratch}/prs.err" ]]
  then
    printf '\n' >&2
    cat "${scratch}/prs.err" >&2
  fi
  [[ "${prs_failed}" == false ]] || return 1
  prs=$(<"${scratch}/prs")

  # Compute the column widths across the pull requests. The author column is shown unless --mine is given.
  local title_w=0 owner_w=0 state_w=0 age_w=0 text author state_text age_secs
  while IFS=$'\x1f' read -r text author state_text age_secs
  do
    [[ -n "${text}" ]] || continue
    (( ${#text} > title_w )) && title_w=${#text}
    [[ "${choice}" != mine ]] && (( ${#author} > owner_w )) && owner_w=${#author}
    (( ${#state_text} > state_w )) && state_w=${#state_text}
    text=$(status:age "${age_secs}")
    (( ${#text} > age_w )) && age_w=${#text}
  done < <(jq --raw-output '["\(.ref // "#\(.number)") \(.title)", .author, .state, .age] | map(tostring)
      | join("\u001f")' \
    <<<"${prs}")
  width=$(( $(status:width) - 3 - 3 - 3 - state_w - 3 - age_w - 3 - 24 - (owner_w > 0 ? owner_w + 6 : 0) ))
  (( width < 20 )) && width=20
  (( title_w > width )) && title_w=${width}

  while IFS= read -r doc
  do
    [[ -n "${doc}" ]] || continue
    printf '\n' >&2
    branch=$(jq --raw-output .branch <<<"${doc}")
    status:pr-block "${doc}" "${title_w}" "${owner_w}" "${state_w}" "${age_w}" "${worktree_on[${branch}]:-}"
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

# status:by-number sorts git:worktrees lines by the issue number their directory name starts with. A directory
# name that does not start with a number sorts first.
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
