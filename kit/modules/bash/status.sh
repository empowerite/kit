# shellcheck shell=bash
# status.sh - the picture of the work: six stages, drawn one at a time or all together (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# kit status draws the picture. Work moves left to right through six stages:
#
#   backlog → incoming → desk → worktrees → changes → releases
#
# The status setting in kit.yml names the stages the picture draws, in order. status:render draws one stage, or
# the picture: every stage under its heading: its glyph, its name, and its count, dim. The rows of backlog,
# changes, and releases live in the modules named after them. The incoming, desk, and worktrees stages, and the
# cells every table shares, are here.
#
# The desk stage is the root checkout, wherever kit runs in the repository. Where it runs decides what worktrees
# and changes draw:
#
#   - At a desk, the root checkout: every linked worktree, the branches whose upstream is gone, and every open pull
#     request.
#   - In a linked worktree: that worktree, and the pull request its branch holds, whoever opened it. The owner flags
#     never hide it.
#   - Above desks, in a directory inside no repository: it walks down. Every directory that holds a .git directory
#     is a desk. The walk stops there, and draws that desk's picture under its path, with the same arguments. It
#     skips hidden directories and symbolic links, and goes no deeper than status_walk_depth. The desks are drawn in
#     path order, each with a fetch of its own.
#
# backlog, incoming, desk, and releases draw the same anywhere in a repository.
#
# The picture collects first and draws after. The forge stages run in the background while origin is fetched. Then
# the checkouts are measured, and every stage is drawn in order. A repository with no origin draws its checkouts
# alone, and says so once.
#
# Which pull requests and issues are shown:
#
#   --all              every one (the default)
#   --mine             the pull requests the caller opened, and the issues assigned to the caller; no owner column
#   --others           anyone else's; unassigned issues included
#   --owner=<login>    the named login's; repeat the flag for more
#
# A pull request's state is the first of these that applies. Its color shows the state's family, never the
# single state: green when nothing is left to decide, yellow when a human owes a decision, and blue when the next
# move is a machine's or a reviewer's. Detail is dim. An issue's number is always cyan and a change's magenta,
# the two colors no family uses, so either reads as what it is in every table the kit draws.
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
# --json prints the picture as one object: stages, the names in order, and under each name what kit status <stage>
# --json prints. A stage that could not be read is null. Above desks it is an array of {path, status}, one per desk.
#
# see: kit/bin/kit-status, modules/bash/backlog.sh, modules/bash/changes.sh, modules/bash/releases.sh

use file git settings backlog changes releases

# status_walk_depth is how many directories below the one it runs in kit status looks for desks, above desks.
status_walk_depth=4

# status_commit_cap is how many uncommitted paths or commits a checkout block lists before it counts the rest.
status_commit_cap=10

# status_sep separates the cells of every row kit status prints, and status_gap replaces it before a cell that
# opens with a glyph, which marks the boundary already. A blank cell that stands in for one takes the gap too.
status_sep=' · '
status_gap='  '

# status_issue_sgr and status_change_sgr color an issue's number and a change's, in every table the kit draws.
status_issue_sgr=36
status_change_sgr=35

# status:stage-purpose <stage> prints what a stage lists, in a few words.
status:stage-purpose() {
  case "$1" in
    (backlog) printf 'the open issues no pull request closes, oldest first\n' ;;
    (incoming) printf 'what the fetch brought: commits on the default branch, branches new and gone\n' ;;
    (desk) printf 'the root checkout: branch, ahead and behind, commits, uncommitted paths\n' ;;
    (worktrees) printf 'every linked worktree: branch, ahead and behind, commits, uncommitted paths\n' ;;
    (changes) printf 'the open pull requests, each with the issues it closes\n' ;;
    (releases) printf 'the releases on the forge, newest first\n' ;;
  esac
}

# status:choices <key> [<answer>…] answers kit syntax --choices for <stage>: the six stages, a closed set, each with
# what it lists.
status:choices() {
  local stage
  [[ "$1" == '<stage>' ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'closed\n'
    return 0
  fi
  for stage in "${settings_stages[@]}"
  do
    printf '%s\t%s\t\t\n' "${stage}" "$(status:stage-purpose "${stage}")"
  done
}

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

# status:cell <text> <width> [right] prints <text> padded to <width> characters, or clipped to it with an ellipsis.
# The padding follows the text, or precedes it with right, which is for a number.
status:cell() {
  local text=$1 width=$2 side=${3:-left}
  if (( ${#text} > width ))
  then
    printf '%s…' "${text:0:width-1}"
  elif [[ "${side}" == right ]]
  then
    printf '%*s%s' $(( width - ${#text} )) '' "${text}"
  else
    printf '%s%*s' "${text}" $(( width - ${#text} )) ''
  fi
}

# status:paint <sgr> <text> prints <text> in the color when stderr is a terminal.
status:paint() {
  printf '%s%s%s' "$(out:sgr "$1")" "$2" "$(out:sgr 0)"
}

# status:heading <stage> [<count>] prints a stage's heading to stderr: its glyph, its name in capitals and bold, then
# its count, dim, as kittui labels the stages on its line.
status:heading() {
  local glyph line
  case "$1" in
    (backlog)   glyph=🎫 ;;
    (incoming)  glyph=📥 ;;
    (desk)      glyph=🏠 ;;
    (worktrees) glyph=🌿 ;;
    (changes)   glyph=🔀 ;;
    (releases)  glyph=📦 ;;
  esac
  line="${glyph:+${glyph} }$(status:paint 1 "${1^^}")"
  [[ -z "${2:-}" ]] || line+=" $(status:paint 90 "$2")"
  printf '%s\n' "${line}" >&2
}

# status:none <text> prints one dim line to stderr, what an empty stage says instead of nothing.
status:none() {
  printf '%s\n' "$(status:paint 90 "$1")" >&2
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

# status:age <seconds> prints a duration in its coarsest informative unit: 45s, 12m, 3h, 47d. It prints nothing
# for no seconds at all, as an issue without a time has.
status:age() {
  local secs=$1
  [[ -n "${secs}" && "${secs}" != null ]] || return 0
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

# status:json-list <item>… prints the items as one JSON array of strings.
status:json-list() {
  jq --null-input --compact-output '$ARGS.positional' --args "$@"
}

# status:fetch-diff <before> <after> prints what changed between two status:refs snapshots, one line each: new,
# moved, or gone; the ref; the old sha, or -; and the new sha, or -.
status:fetch-diff() {
  awk 'NR == FNR { old[$1] = $2; next }
       !($1 in old) { print "new", $1, "-", $2; next }
       old[$1] != $2 { print "moved", $1, old[$1], $2 }
       { delete old[$1] }
       END { for (ref in old) print "gone", ref, old[ref], "-" }' \
    <(printf '%s\n' "$1" | grep .) <(printf '%s\n' "$2" | grep .)
}

# status:fetch-json <before> <after> <default-branch> prints what a fetch brought, between two status:refs snapshots,
# as one object: fetched, true; default, the default branch; commits, the ones new on it at origin, newest first,
# each {sha, subject}; branches, the names new, updated, and gone on origin; and tags, the names new.
status:fetch-json() {
  local default_ref="refs/remotes/origin/$3" kind ref old new commits=''
  local -a added=() updated=() gone=() tags=()
  while IFS=' ' read -r kind ref old new
  do
    case "${kind}:${ref}" in
      (moved:"${default_ref}") commits=$(git log --format='%h%x09%s' "${old}..${new}" 2>/dev/null || true) ;;
      (moved:refs/remotes/origin/*) updated+=("${ref#refs/remotes/origin/}") ;;
      (new:refs/remotes/origin/*) added+=("${ref#refs/remotes/origin/}") ;;
      (gone:refs/remotes/origin/*) gone+=("${ref#refs/remotes/origin/}") ;;
      (new:refs/tags/*) tags+=("${ref#refs/tags/}") ;;
    esac
  done < <(status:fetch-diff "$1" "$2")
  jq --null-input --compact-output --arg default "$3" --arg commits "${commits}" \
    --argjson new "$(status:json-list "${added[@]}")" --argjson updated "$(status:json-list "${updated[@]}")" \
    --argjson gone "$(status:json-list "${gone[@]}")" --argjson tags "$(status:json-list "${tags[@]}")" \
    '{fetched: true, default: $default,
      commits: ($commits | split("\n") | map(select(. != "") | split("\t")
                | {sha: .[0], subject: (.[1:] | join("\t"))})),
      branches: {new: $new, updated: $updated, gone: $gone}, tags: $tags}'
}

# status:incoming-text <incoming-json> prints the fetch's line, from status:incoming's object, to stderr. Nothing
# fetched says 📥 origin: not fetched, dim. Else it is 📥 origin: unchanged, or what the fetch brought, counted: new
# commits on the default branch, then branches updated, new, and pruned, then new tags.
status:incoming-text() {
  local default commits updated added pruned tags line part
  local -a parts=()
  if [[ "$(jq '.fetched' <<<"$1")" != true ]]
  then
    status:none '📥 origin: not fetched'
    return 0
  fi
  IFS=$'\t' read -r default commits updated added pruned tags < <(jq --raw-output \
    '[.default, (.commits | length), (.branches.updated | length), (.branches.new | length),
      (.branches.gone | length), (.tags | length)] | @tsv' <<<"$1")
  (( commits > 0 )) && parts+=("$(status:plural "${commits}" 'new commit' 'new commits') on ${default}")
  (( updated > 0 )) && parts+=("$(status:plural "${updated}" 'branch updated' 'branches updated')")
  (( added > 0 )) && parts+=("$(status:plural "${added}" 'new branch' 'new branches')")
  (( pruned > 0 )) && parts+=("$(status:plural "${pruned}" 'branch pruned' 'branches pruned')")
  (( tags > 0 )) && parts+=("$(status:plural "${tags}" 'new tag' 'new tags')")
  if (( ${#parts[@]} == 0 ))
  then
    printf '📥 origin: unchanged\n' >&2
    return 0
  fi
  line=${parts[0]}
  for part in "${parts[@]:1}"
  do
    line+="${status_sep}${part}"
  done
  printf '📥 origin: fetched %s\n' "${line}" >&2
}

# status:incoming [--no-fetch] [--json] is the incoming stage. It fetches origin, pruning the branches gone from it,
# and prints the fetch's line, as status:incoming-text draws it. With --json it prints status:fetch-json's object
# instead. With --no-fetch it fetches nothing: the line says so, and the object is {"fetched": false}. It returns 1
# when the fetch fails, with a warning.
status:incoming() {
  local json=false no_fetch=false arg before doc
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (--no-fetch) no_fetch=true ;;
    esac
  done
  if [[ "${no_fetch}" == true ]]
  then
    doc='{"fetched":false}'
  else
    before=$(status:refs)
    if ! git fetch --quiet --prune origin
    then
      warn "git fetch --prune failed; ahead and behind counts may be stale"
      return 1
    fi
    doc=$(status:fetch-json "${before}" "$(status:refs)" "$(git:default-branch)")
  fi
  if [[ "${json}" == true ]]
  then
    printf '%s\n' "${doc}"
  else
    status:incoming-text "${doc}"
  fi
}

# status:choice <mine> <others> <owners> names what the owner flags chose, for the wording and the owner column: mine when
# --mine stands alone (the tables leave the owner column out), others when --others does without --mine, owner when
# logins were named, with --mine or not, and all otherwise, --mine with --others included.
status:choice() {
  if [[ "$1" == true && "$2" == true ]]
  then
    printf 'all\n'
  elif [[ "$2" == true ]]
  then
    printf 'others\n'
  elif (( $3 > 0 ))
  then
    printf 'owner\n'
  elif [[ "$1" == true ]]
  then
    printf 'mine\n'
  else
    printf 'all\n'
  fi
}

# status:owner-flags <argument>… reads the owner flags among the arguments into mine, others, and owners, variables
# the caller declares.
status:owner-flags() {
  local arg
  mine=false others=false owners=()
  for arg in "$@"
  do
    case "${arg}" in
      (--mine) mine=true ;;
      (--others) others=true ;;
      (--owner=?*) owners+=("${arg#--owner=}") ;;
    esac
  done
}

# status:choice-of <argument>… prints status:choice's answer for the owner flags among the arguments.
status:choice-of() {
  local mine others
  local -a owners
  status:owner-flags "$@"
  status:choice "${mine}" "${others}" "${#owners[@]}"
}

# status:filter-of <argument>… prints status:filter's line for the owner flags among the arguments.
status:filter-of() {
  local mine others
  local -a owners
  status:owner-flags "$@"
  status:filter "${mine}" "${others}" "${owners[@]}"
}

# status:filter <mine> <others> <login>… prints, on one line, the arguments status:select and backlog:collect take to
# keep the rows the owner flags ask for, as the union of the filters: --except and the caller's login when --others
# is given without --mine and the caller is not among the logins; else the logins named, with the caller's under
# --mine; nothing when every row is kept. It asks the forge for the caller's login only when a flag needs it.
status:filter() {
  local mine=$1 others=$2 me login
  local -a logins=()
  shift 2
  if [[ "${mine}" == true || "${others}" == true ]]
  then
    use forge
    me=$(forge:me) || return 1
  fi
  if [[ "${others}" == true ]]
  then
    [[ "${mine}" != true ]] || return 0
    for login in "$@"
    do
      [[ "${login}" != "${me}" ]] || return 0
    done
    printf -- '--except %s\n' "${me}"
    return 0
  fi
  logins=("$@")
  [[ "${mine}" != true ]] || logins+=("${me}")
  printf '%s\n' "${logins[*]}"
}

# status:select [--except] <login>… reads pull request documents on stdin and keeps those opened by one of the
# named authors, or with --except, those opened by none of them. With no login, it keeps every one.
status:select() {
  local except=false
  [[ "${1:-}" == --except ]] && { except=true; shift; }
  if (( $# == 0 ))
  then cat
  else
    jq --compact-output -L "${kit_jq}" --argjson except "${except}" \
      'import "status" as status; status::by_authors($ARGS.positional; $except)' --args "$@"
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
  } | jq -L "${kit_jq}" --raw-input --slurp --compact-output \
    --arg path "${path}" --arg relative "$(status:relative "${path}")" --arg branch "${branch}" \
    --argjson ahead "${ahead}" --argjson behind "${behind}" --argjson added "${added}" --argjson removed "${removed}" \
    --arg changes "${changes}" \
    'import "status" as status; status::checkout($ARGS.named | del(.changes); $changes)'
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
  printf '%s %s%s%s %s' \
    "$(status:paint-count "${ahead}" '↑' '1;32' "${ahead_w}" right)" \
    "$(status:paint-count "${behind}" '↓' '1;31' "${behind_w}" left)" \
    "${status_sep}" \
    "$(status:paint-count "${added}" '+' '1;32' "${added_w}" right)" \
    "$(status:paint-count "${removed}" '-' '1;31' "${removed_w}" left)"
  (( uncommitted > 0 )) && printf '%s%s' "${status_sep}" "$(status:paint 33 "${uncommitted} uncommitted")"
  return 0
}

# status:count-widths <checkout-json>… prints the widest "↑a", "↓b", "+x", and "-y" across the checkouts, as four
# numbers on one line.
status:count-widths() {
  printf '%s\n' "$@" | jq -L "${kit_jq}" --raw-output --slurp 'import "status" as status; status::widths'
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
  [[ -n "${branch}" ]] || mark="${status_gap}$(status:paint 33 '❗ detached HEAD')"
  printf '%s %s%*s%s%s%s\n' "${glyph}" "${label}" $(( label_w - width )) '' "${mark}" "${status_sep}" \
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
  done < <(jq -L "${kit_jq}" --raw-output --argjson cap "${status_commit_cap}" \
    'import "status" as status; status::changes_shown($cap) | join("\u001f")' <<<"${doc}")
  while IFS=$'\t' read -r sha subject
  do
    [[ -n "${sha}" ]] || continue
    if [[ "${sha}" == '…' ]]
    then
      printf '   %s\n' "$(status:paint 90 "… ${subject}")" >&2
    else
      printf '   %s %s\n' "$(status:paint 90 "${sha}")" "${subject}" >&2
    fi
  done < <(jq -L "${kit_jq}" --raw-output --argjson cap "${status_commit_cap}" \
    'import "status" as status; status::commits_shown($cap) | @tsv' <<<"${doc}")
}

# status:check-cell <checks-json> prints the checks cell, with the duration dim:
#
#   - passed: "✅ <passed> / <total> <took>"
#   - failed: "❌ <failing> failing <passed> / <total> <took>"
#   - running: "🏃 <passed> / <total> <elapsed>"
#
# <took> is the slowest check's own duration, and <elapsed> how long the pending one has waited. A check that ran
# again counts once, by its latest run.
#
# It prints nothing when no check has reported.
status:check-cell() {
  local state passed total failing took text
  # A field can be empty, and a tab-separated read collapses an empty field, so the separator is \x1f instead.
  IFS=$'\x1f' read -r state passed total failing took \
    < <(jq --raw-output '[.state, .passed, .total, .failing, (.took // "")] | map(tostring) | join("\u001f")' <<<"$1")
  case "${state}" in
    (SUCCESS) text="✅ ${total} / ${total}" ;;
    (FAILURE) text="❌ ${failing} failing ${passed} / ${total}" ;;
    (PENDING) text="🏃 ${passed} / ${total}" ;;
    (*) return 0 ;;
  esac
  printf '%s' "${text}"
  [[ -n "${took}" ]] && printf ' %s' "$(status:paint 90 "$(status:age "${took}")")"
  return 0
}

# status:issue-widths <issues-json> reads a JSON array of issue documents and prints the widths of the widest number,
# assignees, age, and branch across them, space-separated: zero where no issue has one, so the column is left out.
status:issue-widths() {
  local ref assignees age branch text ref_w=0 owner_w=0 age_w=0 branch_w=0
  while IFS=$'\x1f' read -r ref assignees age branch
  do
    [[ -n "${ref}" ]] || continue
    (( ${#ref} > ref_w )) && ref_w=${#ref}
    (( ${#assignees} > owner_w )) && owner_w=${#assignees}
    text=$(status:age "${age}")
    (( ${#text} > age_w )) && age_w=${#text}
    (( ${#branch} > branch_w )) && branch_w=${#branch}
  done < <(jq --raw-output '.[] | [(.ref // "#\(.number)"), ((.assignees // []) | join(" ")),
                                   (.age // "" | tostring), (.branch // "")] | join("\u001f")' <<<"$1")
  printf '%s %s %s %s\n' "${ref_w}" "${owner_w}" "${age_w}" "${branch_w}"
}

# status:issue-row <issue-json> <widths> <state> <indent> prints one issue as the backlog and changes stages draw
# it: <indent>, the number right-aligned, the assignees when any issue in the table has one, then <state> with the
# age when <state> is given, as under a change, else the age alone, then the branch when any issue has one, and last
# the title, clipped to what the terminal's width leaves. <widths> is status:issue-widths' line. Every cell after
# the number opens with a glyph, so status_gap separates them, and a cell with nothing is blank, glyph included, so
# every title starts at the same column. The number is cyan; the state is blue, the family of a move that is
# someone else's; the detail is dim.
status:issue-row() {
  local doc=$1 state=$3 indent=$4 ref_w owner_w age_w branch_w ref title assignees age branch row glyph width text
  read -r ref_w owner_w age_w branch_w <<<"$2"
  IFS=$'\x1f' read -r ref title assignees age branch \
    < <(jq --raw-output '[(.ref // "#\(.number)"), .title, ((.assignees // []) | join(" ")),
                          (.age // "" | tostring), (.branch // "")] | join("\u001f")' <<<"${doc}")
  row="${indent}$(status:paint "${status_issue_sgr}" "$(status:cell "${ref}" "${ref_w}" right)")"
  width=$(( $(status:width) - ${#indent} - ref_w - 5 ))
  if (( owner_w > 0 ))
  then
    glyph='   '
    [[ -n "${assignees}" ]] && glyph='👤 '
    row+="${status_gap}${glyph}$(status:paint 90 "$(status:cell "${assignees}" "${owner_w}")")"
    width=$(( width - 5 - owner_w ))
  fi
  text=$(status:age "${age}")
  if [[ -n "${state}" ]]
  then
    row+="${status_gap}🛫 $(status:paint 34 "${state}")"
    width=$(( width - 5 - ${#state} ))
    if [[ -n "${text}" ]]
    then
      row+=" $(status:paint 90 "$(status:cell "${text}" "${age_w}" right)")"
      width=$(( width - 1 - age_w ))
    fi
  elif (( age_w > 0 ))
  then
    row+="${status_gap}$(status:paint 90 "$(status:cell "${text}" "${age_w}" right)")"
    width=$(( width - 2 - age_w ))
  fi
  if (( branch_w > 0 ))
  then
    if [[ -n "${branch}" ]]
    then row+="${status_gap}$(status:paint 90 "🌿 $(status:cell "${branch}" "${branch_w}")")"
    else row+="${status_gap}   $(status:cell '' "${branch_w}")"
    fi
    width=$(( width - 5 - branch_w ))
  fi
  (( width < 20 )) && width=20
  (( ${#title} > width )) && title="${title:0:width-1}…"
  printf '%s%s🎫 %s\n' "${row}" "${status_gap}" "${title}" >&2
}

# status:pr-block <pr-json> <title-width> <owner-width> <state-width> <age-width> <with-branch> <issue-widths>
# prints a pull request: its row, then the issues it closes, one row each, in flight, indented under it. The state
# and its age share a cell. The row shows the author before the state when <owner-width> is above zero, and ends
# with the branch when <with-branch> is not empty. <issue-widths> is status:issue-widths' line over every issue in
# the table.
status:pr-block() {
  local doc=$1 title_w=$2 owner_w=$3 state_w=$4 age_w=$5 worktree=$6 issue_widths=$7
  local number ref title author branch state age unresolved row issue checks
  IFS=$'\x1f' read -r number ref title author branch state age unresolved \
    < <(jq --raw-output '[.number, (.ref // "#\(.number)"), .title, .author, .branch, .state, .age, .unresolved]
          | map(tostring)
          | join("\u001f")' <<<"${doc}")
  row="🔀 $(status:paint "${status_change_sgr}" "${ref}") $(status:cell "${title}" $(( title_w - ${#ref} - 1 )))"
  (( owner_w > 0 )) && row+="${status_gap}👤 $(status:paint 90 "$(status:cell "${author}" "${owner_w}")")"
  row+="${status_gap}$(status:state-emoji "${state}")"
  row+=" $(status:paint "$(status:state-sgr "${state}")" "$(status:cell "${state}" "${state_w}")")"
  row+=" $(status:paint 90 "$(status:cell "$(status:age "${age}")" "${age_w}" right)")"
  checks=$(status:check-cell "$(jq --compact-output .checks <<<"${doc}")")
  [[ -n "${checks}" ]] && row+="${status_gap}${checks}"
  (( unresolved > 0 )) && row+="${status_gap}$(status:paint 33 "💬 ${unresolved} unresolved")"
  [[ -n "${worktree}" ]] && row+="${status_gap}$(status:paint 90 "🌿 ${branch}")"
  printf '%s\n' "${row}" >&2
  while IFS= read -r issue
  do
    [[ -n "${issue}" ]] || continue
    status:issue-row "${issue}" "${issue_widths}" 'in flight' '   '
  done < <(jq --compact-output '.closes[]?' <<<"${doc}")
  return 0
}

# status:pr-table <choice> <prs> draws the pull requests in <prs>, one JSON document per line, one block each, a
# blank line between blocks, their cells sized across the rows: the row's title, author, state, and age, and the
# issue rows' number, assignees, and age. Every row ends with its branch. <choice> mine leaves the author column out.
status:pr-table() {
  local choice=$1 prs=$2 doc issue_widths first=true
  local title_w=0 owner_w=0 state_w=0 age_w=0 width text author state_text age_secs
  [[ -n "${prs}" ]] || return 0
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
  width=$(( $(status:width) - 3 - 2 - 3 - state_w - 3 - age_w - 3 - 24 - (owner_w > 0 ? owner_w + 5 : 0) ))
  (( width < 20 )) && width=20
  (( title_w > width )) && title_w=${width}
  issue_widths=$(status:issue-widths "$(jq --slurp --compact-output 'map(.closes[]?)' <<<"${prs}")")
  while IFS= read -r doc
  do
    [[ -n "${doc}" ]] || continue
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    status:pr-block "${doc}" "${title_w}" "${owner_w}" "${state_w}" "${age_w}" yes "${issue_widths}"
  done <<<"${prs}"
}

# status_worktrees and status_gone hold what status:worktrees measured: each linked worktree's document in
# issue-number order, or the one worktree's, each status:checkout's JSON; and the branches whose upstream is gone,
# one per line, none in a worktree.
declare -g status_gone=''
declare -ga status_worktrees=()

# status:desk [--json] is the desk stage. It measures the root checkout, wherever kit runs in the repository, and
# draws its block. With --json it prints the checkout's object instead.
status:desk() {
  local main_root branch doc
  main_root=$(git:main-root)
  branch=$(git -C "${main_root}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  doc=$(status:checkout "${main_root}" "${branch}" "$(git:default-branch)")
  if [[ " $* " == *" --json "* ]]
  then
    printf '%s\n' "${doc}"
  else
    status:draw-desk "${doc}"
  fi
}

# status:draw-desk <checkout-json> draws the desk's block.
status:draw-desk() {
  local relative branch width
  IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"$1")
  IFS=$'\t' read -r _ width < <(status:label "${relative}" "${branch}")
  status:checkout-block '🏠' "$1" "${width}"
}

# status:linked <default-branch> measures every linked worktree, from origin's default branch, into
# status_worktrees.
status:linked() {
  local path branch
  status_worktrees=()
  while IFS=$'\t' read -r path _ branch
  do
    [[ -n "${path}" ]] || continue
    status_worktrees+=("$(status:checkout "${path}" "${branch}" "$1")")
  done < <(git:linked-worktrees | status:by-number)
}

# status:draw-checkouts draws what status:worktrees measured: each worktree's block, the labels and the counts in
# columns, a blank line between blocks.
status:draw-checkouts() {
  local label_w=0 width relative branch widths doc first=true
  for doc in "${status_worktrees[@]}"
  do
    IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
    IFS=$'\t' read -r _ width < <(status:label "${relative}" "${branch}")
    (( width > label_w )) && label_w=${width}
  done
  widths=$(status:count-widths "${status_worktrees[@]}")
  for doc in "${status_worktrees[@]}"
  do
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    status:checkout-block '🌿' "${doc}" "${label_w}" "${widths}"
  done
}

# status:in-worktree succeeds in a linked worktree, whose git directory is not the repository's common one.
status:in-worktree() {
  [[ "$(git rev-parse --path-format=absolute --git-dir)" \
    != "$(git rev-parse --path-format=absolute --git-common-dir)" ]]
}

# status:worktrees [--json] is the worktrees stage. It measures the linked worktrees and draws them: at a desk,
# every one in issue-number order, and the branches whose upstream is gone; in a worktree, that worktree alone.
# With --json it prints status:worktrees-json's object instead.
status:worktrees() {
  local default_branch here branch
  default_branch=$(git:default-branch)
  status_gone='' status_worktrees=()
  if status:in-worktree
  then
    here=$(git rev-parse --show-toplevel)
    branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
    status_worktrees=("$(status:checkout "${here}" "${branch}" "${default_branch}")")
  else
    status:linked "${default_branch}"
    status_gone=$(git:gone-branches)
  fi
  if [[ " $* " == *" --json "* ]]
  then
    status:worktrees-json
  else
    status:draw-worktrees
  fi
}

# status:worktrees-json prints what status:worktrees measured as one object: worktrees, the checkouts' documents;
# and goneBranches, the names.
status:worktrees-json() {
  local gone
  gone=$(printf '%s\n' "${status_gone}" | jq --raw-input . | jq --slurp --compact-output 'map(select(. != ""))')
  jq --null-input --compact-output \
    --argjson worktrees "$(printf '%s\n' "${status_worktrees[@]}" | jq --slurp --compact-output .)" \
    --argjson goneBranches "${gone}" '{worktrees: $worktrees, goneBranches: $goneBranches}'
}

# status:draw-worktrees draws what status:worktrees measured: the checkout blocks, or a dim line that says no
# worktree; then, after a blank line, each gone branch as 🔖 <branch> · upstream gone · kit tidy removes it.
status:draw-worktrees() {
  local branch
  if (( ${#status_worktrees[@]} == 0 ))
  then
    status:none 'no worktree'
  else
    status:draw-checkouts
  fi
  [[ -n "${status_gone}" ]] || return 0
  printf '\n' >&2
  while IFS= read -r branch
  do
    printf '🔖 %s%s%s\n' "${branch}" "${status_sep}" \
      "$(status:paint 90 "upstream gone${status_sep}kit tidy removes it")" >&2
  done <<<"${status_gone}"
}

# status:stage <stage> [<argument>…] prints one stage, without its heading, with the arguments the stage takes.
status:stage() {
  local stage=$1
  shift
  case "${stage}" in
    (backlog) backlog:render "$@" ;;
    (incoming) status:incoming "$@" ;;
    (desk) status:desk "$@" ;;
    (worktrees) status:worktrees "$@" ;;
    (changes) changes:render "$@" ;;
    (releases) releases:render "$@" ;;
  esac
}

# status:stage-args <stage> <argument>… prints, one per line, the arguments among those given that the stage takes:
# --json for every stage, --no-fetch for incoming, and the owner flags for backlog and changes.
status:stage-args() {
  local stage=$1 arg
  shift
  for arg in "$@"
  do
    case "${arg}" in
      (--json) printf '%s\n' "${arg}" ;;
      (--no-fetch) [[ "${stage}" != incoming ]] || printf '%s\n' "${arg}" ;;
      (--all | --mine | --others | --owner=?*)
        [[ "${stage}" != backlog && "${stage}" != changes ]] || printf '%s\n' "${arg}"
    esac
  done
  return 0
}

# status:picture-document <scratch> <stage> <origin> prints the document a stage collected under <scratch>, for the
# picture's JSON: null for a stage that failed or, with no origin, could not run; {"fetched": false} for incoming
# with no origin.
status:picture-document() {
  local scratch=$1 stage=$2 origin=$3
  if [[ "${origin}" == false && "${stage}" == incoming ]]
  then
    printf '{"fetched":false}\n'
  elif [[ -f "${scratch}/${stage}.status" && "$(<"${scratch}/${stage}.status")" == 0 ]]
  then
    cat "${scratch}/${stage}.json"
  else
    printf 'null\n'
  fi
}

# status:picture [--no-fetch] [--json] [<owner flag>…] draws every stage the status setting names, in its order, or
# prints the picture as one JSON object. It collects first and draws after: the forge stages run in the background
# while origin is fetched; then the checkouts are measured; then each stage is drawn in order, under its heading. A
# stage that failed is drawn as its refusal, and the picture returns 1 at the end, in either form; so does a fetch
# that failed, after one warning, whether or not incoming is drawn. With no origin it fetches nothing and asks the
# forge nothing: it draws the checkouts, says so once, and returns 0. It returns 2 when the status setting is wrong.
status:picture() {
  local json=false no_fetch=false origin=true failed=false first=true stage doc code choice scratch text
  local -a stages=() args=() named=()
  text=$(settings:stages) || return 2
  readarray -t stages <<<"${text}"
  [[ " $* " == *" --json "* ]] && json=true
  [[ " $* " == *" --no-fetch "* ]] && no_fetch=true
  git remote get-url origin >/dev/null 2>&1 || origin=false
  choice=$(status:choice-of "$@")
  file:scratch scratch
  # The forge stages need nothing local and the forge is the slow call, so they start first, in the background.
  for stage in "${stages[@]}"
  do
    case "${stage}" in
      (backlog | changes | releases)
        [[ "${origin}" == true ]] || continue
        readarray -t args < <(status:stage-args "${stage}" --json "$@")
        (
          status:stage "${stage}" "${args[@]}" && code=0 || code=$?
          printf '%s\n' "${code}" > "${scratch}/${stage}.status"
        ) > "${scratch}/${stage}.json" 2> "${scratch}/${stage}.err" &
    esac
  done
  # The fetch comes before the checkouts are measured, so ahead and behind count against origin as it is now.
  if [[ "${origin}" == false ]]
  then
    true
  elif list:among incoming "${stages[@]}"
  then
    readarray -t args < <(status:stage-args incoming --json "$@")
    status:incoming "${args[@]}" > "${scratch}/incoming.json" 2> "${scratch}/incoming.err" && code=0 || code=$?
    printf '%s\n' "${code}" > "${scratch}/incoming.status"
  elif [[ "${no_fetch}" == false ]] && ! git fetch --quiet --prune origin 2>/dev/null
  then
    warn "git fetch --prune failed; ahead and behind counts may be stale"
    failed=true
  fi
  for stage in desk worktrees
  do
    list:among "${stage}" "${stages[@]}" || continue
    status:stage "${stage}" --json > "${scratch}/${stage}.json" 2> "${scratch}/${stage}.err" && code=0 || code=$?
    printf '%s\n' "${code}" > "${scratch}/${stage}.status"
  done
  wait
  # One rule for both forms: a stage that ran and failed makes the picture return 1, as a fetch that failed does. A
  # stage that did not run, with no origin, is no failure.
  for stage in "${stages[@]}"
  do
    [[ ! -f "${scratch}/${stage}.status" || "$(<"${scratch}/${stage}.status")" == 0 ]] || failed=true
  done
  if [[ "${json}" == true ]]
  then
    for stage in "${stages[@]}"
    do
      doc=$(status:picture-document "${scratch}" "${stage}" "${origin}")
      named+=(--argjson "${stage}" "${doc}")
      [[ ! -s "${scratch}/${stage}.err" ]] || cat "${scratch}/${stage}.err" >&2
    done
    [[ "${origin}" == true ]] || warn "no origin: nothing fetched, no pull request shown"
    jq --null-input --compact-output --argjson stages "$(status:json-list "${stages[@]}")" "${named[@]}" \
      '{stages: $stages} + $ARGS.named'
    [[ "${failed}" == false ]]
    return
  fi
  if [[ "${origin}" == false ]]
  then
    warn "no origin: nothing fetched, no pull request shown"
    first=false
  fi
  for stage in "${stages[@]}"
  do
    [[ "${origin}" == true || "${stage}" == desk || "${stage}" == worktrees ]] || continue
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    code=$(<"${scratch}/${stage}.status")
    if (( code != 0 ))
    then
      status:heading "${stage}"
      cat "${scratch}/${stage}.err" >&2
      continue
    fi
    doc=$(<"${scratch}/${stage}.json")
    case "${stage}" in
      (backlog)
        status:heading backlog "$(jq 'length' <<<"${doc}")"
        backlog:draw "${doc}" "${choice}"
        ;;
      (incoming)
        status:heading incoming
        status:incoming-text "${doc}"
        ;;
      (desk)
        status:heading desk 1
        status:draw-desk "${doc}"
        ;;
      (worktrees)
        status:heading worktrees "$(jq '.worktrees | length' <<<"${doc}")"
        status:draw-worktrees
        ;;
      (changes)
        status:heading changes "$(jq 'length' <<<"${doc}")"
        changes:draw "${doc}" "${choice}"
        ;;
      (releases)
        status:heading releases "$(jq 'length' <<<"${doc}")"
        releases:draw "${doc}"
        ;;
    esac
    [[ ! -s "${scratch}/${stage}.err" ]] || cat "${scratch}/${stage}.err" >&2
  done
  [[ "${failed}" == false ]]
}

# status:render [<stage>] [<argument>…] draws one stage, or the picture, as kit status's page says. Above desks it
# draws each desk's under its path. --text, which kit status takes to stay in text on a terminal, changes nothing
# here.
status:render() {
  local stage='' arg
  local -a args=()
  if ! git rev-parse --git-dir >/dev/null 2>&1
  then
    status:above "$@"
    return
  fi
  case "${1:-}" in
    (backlog | incoming | desk | worktrees | changes | releases)
      stage=$1
      shift
  esac
  for arg in "$@"
  do
    [[ "${arg}" == --text ]] || args+=("${arg}")
  done
  if [[ -n "${stage}" ]]
  then
    status:stage "${stage}" "${args[@]}"
  else
    status:picture "${args[@]}"
  fi
}

# status:desks <dir> <depth> prints every desk at or below <dir>, at most <depth> levels down, in path order: a
# directory that holds a .git directory. The walk stops at a desk, so a worktree or a nested repository inside one
# is the desk's own, and it skips hidden directories and symbolic links.
status:desks() {
  local dir=$1 depth=$2 sub
  if [[ -d "${dir}/.git" ]]
  then
    printf '%s\n' "${dir}"
    return 0
  fi
  (( depth > 0 )) || return 0
  for sub in "${dir}"/*/
  do
    sub=${sub%/}
    [[ -d "${sub}" && ! -L "${sub}" ]] || continue
    status:desks "${sub}" $(( depth - 1 ))
  done
}

# status:above <argument>… draws every desk below the current directory, each under its path, as kit status draws it
# at that desk, with the same arguments. The desks run at once, each fetching on its own, and are drawn in path
# order.
status:above() {
  local json=false arg desk index=0 color=''
  local -a desks
  for arg in "$@"
  do
    [[ "${arg}" == --json ]] && json=true
  done
  readarray -t desks < <(status:desks "$(pwd -P)" "${status_walk_depth}")
  if (( ${#desks[@]} == 0 ))
  then
    error "no repository at or below $(pwd -P), ${status_walk_depth} levels down; run kit status in one"
    return 1
  fi
  local scratch
  file:scratch scratch
  [[ "${out_color}" == 1 ]] && color=1
  for desk in "${desks[@]}"
  do
    ( cd "${desk}" && COLUMNS=$(status:width) CLICOLOR_FORCE=${color} run kit status "$@" ) \
      > "${scratch}/${index}.out" 2> "${scratch}/${index}.err" &
    index=$(( index + 1 ))
  done
  wait || true
  if [[ "${json}" == true ]]
  then
    for index in "${!desks[@]}"
    do
      jq --compact-output --arg path "$(status:relative "${desks[${index}]}")" '{path: $path, status: .}' \
        "${scratch}/${index}.out" 2>/dev/null || jq --null-input --compact-output \
        --arg path "$(status:relative "${desks[${index}]}")" '{path: $path, status: null}'
      cat "${scratch}/${index}.err" >&2
    done | jq --slurp .
    return 0
  fi
  for index in "${!desks[@]}"
  do
    (( index == 0 )) || printf '\n' >&2
    printf '%s\n' "$(status:paint 1 "📂 $(status:relative "${desks[${index}]}")")" >&2
    cat "${scratch}/${index}.err" "${scratch}/${index}.out" >&2
  done
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
