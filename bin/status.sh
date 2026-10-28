# shellcheck shell=bash
# status.sh - the picture of the work in flight (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# status:render [--no-fetch] [--json] [--mine | --all | --owner=<login>…] prints the checkouts and the open pull
# requests, and changes nothing. It asks the forge once, for every open pull request with the issues it closes,
# its review threads, and the checks on its head; everything else is a local git read. The forge is asked first,
# in the background, so its answer arrives while the fetch and the checkouts are drawn.
#
# The picture: what the fetch brought, unless --no-fetch; the desk, the root checkout; then each worktree under
# worktrees/, each with its uncommitted paths before its commits; then each open pull request with
# its author, state, age, checks, the worktree holding its branch, and the issues it closes; then the branches whose
# upstream is gone. --mine shows the pull requests the caller opened and no author column; --all, the default, shows
# every open one, and --owner=<login>, given once or more, those the named authors opened, each with the author before
# the state. Color is by family, never by state: green when nothing is left to decide, yellow when a human owes a
# decision, blue when the next move is a machine's or a reviewer's, dim for detail. The states, in the order the first
# that holds wins:
#
#   draft 🔨, queued 🚦, conflicted 💥, failing:gate 🔥, no-check 📭, comments:open 💬,
#   blocked:approval 👀,
#   ready 🛬
#
# --json prints the same facts as one document: {desk, worktrees, pullRequests, goneBranches}, the pull requests
# chosen the same way.

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

# status:refs prints "<ref> <sha>" for origin's branches and the tags, the refs a fetch moves.
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

# status:fetch-summary <before> <after> <default-branch> prints one line naming what moved between two status:refs
# snapshots: the commits new on origin's default branch, then the branches updated, new, and pruned, then the tags
# new; "nothing new" when nothing moved.
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

# status:fetch <default-branch> fetches origin, pruning, and prints status:fetch-summary's line and a blank line;
# where the fetch fails, a warning in its place.
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

# status:collect prints one JSON object per open pull request, the documents the forge provider folds, or
# nothing with a warning where the forge cannot be asked.
status:collect() {
  use forge
  forge:open-prs || warn "no pull request is shown; see above"
}

# status:select <login>… keeps, of the pull request documents on stdin, those an author named here opened; with
# no login named, every one.
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
# default branch, the lines the working tree adds and removes against that point, the count of uncommitted paths
# and each as {status, path} with git's two-letter status, and the commits past that point as [{sha, subject}]. A detached checkout measures its lines against its own head
# and lists no commits.
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
# counts in status:count-widths' columns, then its uncommitted paths, yellow status and path, then its commits,
# dim hash and subject. Past status_commit_cap it prints the first paths and a count of the rest, and the newest
# commits, a dim count of the hidden, and the oldest.
status:checkout-block() {
  local glyph=$1 doc=$2 label_w=$3 widths=${4:-} relative branch label width mark='' sha subject code path
  IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
  IFS=$'\t' read -r label width < <(status:label "${relative}" "${branch}")
  [[ -n "${branch}" ]] || mark="   $(status:paint 33 '❗ detached HEAD')"
  printf '%s %s%*s%s   %s\n' "${glyph}" "${label}" $(( label_w - width )) '' "${mark}" \
    "$(status:measures "${doc}" "${widths}")" >&2
  # A status can open with a blank, which a tab read would drop, so the separator is one it cannot.
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

# status:pr-block <pr-json> <title-width> <owner-width> <state-width> <age-width> <worktree> prints a pull request:
# its row, with the author before the state where the owner width is above zero and ending with its branch where
# a worktree here holds it, then the issues it closes.
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

# status:render [--no-fetch] [--json] [--mine | --all | --owner=<login>…] draws the picture the file header
# describes, or prints it as one JSON document.
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
  # The forge is the slow read and needs nothing local, so it starts first and runs while the fetch and the local
  # reads do; its warnings are held for the pull requests' place in the picture, so they do not land among the
  # checkouts. --mine asks the forge who the caller is in the same job.
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

  # The desk and the worktrees by issue number, each as JSON.
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

  # The checkouts are drawn while the forge may still be answering.
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

  # Column widths across the pull requests; the author column is there unless --mine chose the rows.
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
