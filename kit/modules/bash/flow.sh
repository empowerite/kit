# shellcheck shell=bash
# flow.sh - the picture of the work: eight stages, drawn one at a time or all together (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# kit flow draws the picture. Work moves left to right through eight stages:
#
#   backlog → underway → desk → worktrees → changes → incoming → builds → releases
#
# The flow setting in kit.yml names the stages the picture draws, in order. flow:render draws one stage, or
# the picture: every stage under its heading: its glyph, its name, and its count, dim. The rows of backlog,
# underway, changes, builds, and releases live in the modules named after them. The desk, worktrees, and incoming
# stages, and the cells every table shares, are here.
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
#     skips hidden directories and symbolic links, and goes no deeper than flow_walk_depth. The desks are drawn in
#     path order, each with a fetch of its own.
#
# backlog, underway, desk, incoming, builds, and releases draw the same anywhere in a repository.
#
# The primary remote is the primary setting in kit.yml, else origin; modules/bash/forge.sh says how. The incoming
# stage fetches every remote, and the rest measure against the primary.
#
# The picture collects first and draws after. The forge stages run in the background while the remotes are fetched.
# Then the checkouts are measured, and every stage is drawn in order. A repository with no primary remote draws its
# checkouts alone, and says so once.
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
# --json prints the picture as one object: stages, the names in order, and under each name what kit flow <stage>
# --json prints. A stage that could not be read is null. Above desks it is an array of {path, status}, one per desk.
#
# see: kit/bin/kit-flow, modules/bash/backlog.sh, modules/bash/underway.sh, modules/bash/changes.sh,
# modules/bash/builds.sh, modules/bash/releases.sh

use file git settings backlog underway changes builds releases

# flow_walk_depth is how many directories below the one it runs in kit flow looks for desks, above desks.
flow_walk_depth=4

# flow_commit_cap is how many uncommitted paths or commits a checkout block lists before it counts the rest.
flow_commit_cap=10

# flow_sep separates the cells of every row kit flow prints, and flow_gap replaces it before a cell that
# opens with a glyph, which marks the boundary already. A blank cell that stands in for one takes the gap too.
flow_sep=' · '
flow_gap='  '

# flow_issue_sgr and flow_change_sgr color an issue's number and a change's, in every table the kit draws.
flow_issue_sgr=36
flow_change_sgr=35

# flow:stage-purpose <stage> prints what a stage lists, in a few words.
flow:stage-purpose() {
  case "$1" in
    (backlog) printf 'the open issues no open pull request closes, oldest first\n' ;;
    (underway) printf 'the open issues an open pull request closes: who works on each, and where\n' ;;
    (incoming) printf 'what the fetch brought: commits on the default branch, branches new and gone\n' ;;
    (desk) printf 'the root checkout: branch, ahead and behind, commits, uncommitted paths\n' ;;
    (worktrees) printf 'every linked worktree: branch, ahead and behind, commits, uncommitted paths\n' ;;
    (changes) printf 'the open pull requests, each with the issues it closes\n' ;;
    (builds) printf 'the runs no pull request owns: the default branch, tags, schedules\n' ;;
    (releases) printf 'the releases on the forge, newest first\n' ;;
  esac
}

# flow:choices <key> [<answer>…] answers kit syntax --choices for <stage>: the eight stages, a closed set, each with
# what it lists.
flow:choices() {
  local stage
  [[ "$1" == '<stage>' ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'closed\n'
    return 0
  fi
  for stage in "${settings_stages[@]}"
  do
    printf '%s\t%s\t\t\n' "${stage}" "$(flow:stage-purpose "${stage}")"
  done
}

# flow:width prints the terminal's width in columns: COLUMNS, else the width of the terminal on stderr, else
# 100. A width under 60 also gives 100.
flow:width() {
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

# flow:cell <text> <width> [right] prints <text> padded to <width> characters, or clipped to it with an ellipsis.
# The padding follows the text, or precedes it with right, which is for a number.
flow:cell() {
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

# flow:paint <sgr> <text> prints <text> in the color when stderr is a terminal.
flow:paint() {
  printf '%s%s%s' "$(out:sgr "$1")" "$2" "$(out:sgr 0)"
}

# flow:heading <stage> [<count>] prints a stage's heading to stderr: its glyph, its name in capitals and bold, then
# its count, dim, as kittui labels the stages on its line.
flow:heading() {
  local glyph line
  case "$1" in
    (backlog)   glyph=🎫 ;;
    (underway)  glyph=🛫 ;;
    (incoming)  glyph=📥 ;;
    (desk)      glyph=🏠 ;;
    (worktrees) glyph=🌿 ;;
    (changes)   glyph=🔀 ;;
    (builds)    glyph=🏭 ;;
    (releases)  glyph=📦 ;;
  esac
  line="${glyph:+${glyph} }$(flow:paint 1 "${1^^}")"
  [[ -z "${2:-}" ]] || line+=" $(flow:paint 90 "$2")"
  printf '%s\n' "${line}" >&2
}

# flow:none <text> prints one dim line to stderr, what an empty stage says instead of nothing.
flow:none() {
  printf '%s\n' "$(flow:paint 90 "$1")" >&2
}

# flow:state-sgr <state> prints the color code of a state's family.
flow:state-sgr() {
  case "$1" in
    (ready)                                     printf '32' ;;
    (conflicted | failing:gate | comments:open) printf '33' ;;
    (queued | no-check | blocked:approval)      printf '34' ;;
    (*)                                         printf '90' ;;
  esac
}

# flow:state-emoji <state> prints a state's glyph. Each glyph is one code point drawn two columns wide, so the
# next column stays aligned.
flow:state-emoji() {
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

# flow:age <seconds> prints a duration in its coarsest informative unit: 45s, 12m, 3h, 47d. It prints nothing
# for no seconds at all, as an issue without a time has.
flow:age() {
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

# flow:refs <remote> prints "<ref> <sha>" for the remote's branches and for the tags: the refs a fetch of it moves.
flow:refs() {
  git for-each-ref --format='%(refname) %(objectname)' "refs/remotes/$1" refs/tags | grep --invert-match '/HEAD '
  return 0
}

# flow:plural <count> <singular> <plural> prints "<count> <singular>" or "<count> <plural>".
flow:plural() {
  if (( $1 == 1 ))
  then printf '1 %s' "$2"
  else printf '%s %s' "$1" "$3"
  fi
}

# flow:json-list <item>… prints the items as one JSON array of strings.
flow:json-list() {
  jq --null-input --compact-output '$ARGS.positional' --args "$@"
}

# flow:fetch-diff <before> <after> prints what changed between two flow:refs snapshots, one line each: new,
# moved, or gone; the ref; the old sha, or -; and the new sha, or -.
flow:fetch-diff() {
  awk 'NR == FNR { old[$1] = $2; next }
       !($1 in old) { print "new", $1, "-", $2; next }
       old[$1] != $2 { print "moved", $1, old[$1], $2 }
       { delete old[$1] }
       END { for (ref in old) print "gone", ref, old[ref], "-" }' \
    <(printf '%s\n' "$1" | grep .) <(printf '%s\n' "$2" | grep .)
}

# flow:remote-json <remote> prints what kit knows of a remote before it fetches, as one object: name; forge, as
# forge:of answers, or null; and host, as forge:host answers, or null.
flow:remote-json() {
  local forge
  forge=$(forge:of "$1") || return $?
  jq --null-input --compact-output --arg name "$1" --arg forge "${forge}" --arg host "$(forge:host "$1")" \
    '{name: $name, forge: ($forge | select(. != "") // null), host: ($host | select(. != "") // null)}'
}

# flow:remote-default <remote> prints the remote's default branch: what its HEAD names, else <fallback> when the
# remote has a branch of that name, else nothing. A remote added after the clone has no HEAD until git remote
# set-head, so the primary's default branch is the fallback.
flow:remote-default() {
  local branch
  if branch=$(git symbolic-ref --quiet --short "refs/remotes/$1/HEAD" 2>/dev/null)
  then
    printf '%s\n' "${branch#"$1"/}"
  elif [[ -n "$2" ]] && git show-ref --verify --quiet "refs/remotes/$1/$2"
  then
    printf '%s\n' "$2"
  fi
  return 0
}

# flow:fetch-json <remote> <before> <after> <default-branch> prints what a fetch of <remote> brought, between two
# flow:refs snapshots, as flow:remote-json's object with more: default, the default branch, or null; commits, the
# ones new on it, newest first, each {sha, subject}; branches, the names new, updated, and gone on the remote; and
# tags, the names new.
flow:fetch-json() {
  local remote=$1 default_ref="refs/remotes/$1/$4" kind ref old new commits=''
  local -a added=() updated=() gone=() tags=()
  while IFS=' ' read -r kind ref old new
  do
    case "${kind}:${ref}" in
      (moved:"${default_ref}") commits=$(git log --format='%h%x09%s' "${old}..${new}" 2>/dev/null || true) ;;
      (moved:"refs/remotes/${remote}/"*) updated+=("${ref#"refs/remotes/${remote}/"}") ;;
      (new:"refs/remotes/${remote}/"*) added+=("${ref#"refs/remotes/${remote}/"}") ;;
      (gone:"refs/remotes/${remote}/"*) gone+=("${ref#"refs/remotes/${remote}/"}") ;;
      (new:refs/tags/*) tags+=("${ref#refs/tags/}") ;;
    esac
  done < <(flow:fetch-diff "$2" "$3")
  jq --compact-output --arg default "$4" --arg commits "${commits}" \
    --argjson new "$(flow:json-list "${added[@]}")" --argjson updated "$(flow:json-list "${updated[@]}")" \
    --argjson gone "$(flow:json-list "${gone[@]}")" --argjson tags "$(flow:json-list "${tags[@]}")" \
    '. + {default: ($default | select(. != "") // null),
          commits: ($commits | split("\n") | map(select(. != "") | split("\t")
                    | {sha: .[0], subject: (.[1:] | join("\t"))})),
          branches: {new: $new, updated: $updated, gone: $gone}, tags: $tags}' <<<"$(flow:remote-json "${remote}")"
}

# flow:fetched-line <remote-json> prints, to stderr, a fetched remote's line: ❌ and git's line when the fetch
# failed; 📥 unchanged; or 📥 fetched and what came, counted: new commits on the default branch, then branches
# updated, new, and pruned, then new tags.
flow:fetched-line() {
  local error default commits updated added pruned tags line part
  local -a parts=()
  IFS=$'\x1f' read -r error default commits updated added pruned tags < <(jq --raw-output \
    '[(.error // ""), (.default // ""), (.commits | length), (.branches.updated | length), (.branches.new | length),
      (.branches.gone | length), (.tags | length)] | map(tostring) | join("\u001f")' <<<"$1")
  if [[ -n "${error}" ]]
  then
    printf '❌ %s\n' "${error}" >&2
    return 0
  fi
  (( commits > 0 )) && parts+=("$(flow:plural "${commits}" 'new commit' 'new commits') on ${default}")
  (( updated > 0 )) && parts+=("$(flow:plural "${updated}" 'branch updated' 'branches updated')")
  (( added > 0 )) && parts+=("$(flow:plural "${added}" 'new branch' 'new branches')")
  (( pruned > 0 )) && parts+=("$(flow:plural "${pruned}" 'branch pruned' 'branches pruned')")
  (( tags > 0 )) && parts+=("$(flow:plural "${tags}" 'new tag' 'new tags')")
  if (( ${#parts[@]} == 0 ))
  then
    printf '📥 unchanged\n' >&2
    return 0
  fi
  line=${parts[0]}
  for part in "${parts[@]:1}"
  do
    line+="${flow_sep}${part}"
  done
  printf '📥 fetched %s\n' "${line}" >&2
}

# flow:incoming-text <incoming-json> prints flow:incoming's object to stderr: a block per remote, the primary
# first, a blank line between blocks. A block's first line is 📡, the remote's name, its forge or no forge, and its
# host, or its URL when it has none. Under it is the fetch's line, as flow:fetched-line draws it, or 📥 not fetched,
# dim. With no remote it says so in one dim line.
flow:incoming-text() {
  local fetched remote name forge host first=true
  fetched=$(jq '.fetched' <<<"$1")
  while IFS= read -r remote
  do
    [[ -n "${remote}" ]] || continue
    IFS=$'\x1f' read -r name forge host < <(jq --raw-output '[.name, (.forge // "no forge"), (.host // "")]
      | join("\u001f")' <<<"${remote}")
    [[ -n "${host}" ]] || host=$(git remote get-url --end-of-options "${name}" 2>/dev/null || true)
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    printf '📡 %s%s\n' "${name}" "$(flow:paint 90 "${flow_sep}${forge}${host:+${flow_sep}${host}}")" >&2
    if [[ "${fetched}" == true ]]
    then flow:fetched-line "${remote}"
    else flow:none '📥 not fetched'
    fi
  done < <(jq --compact-output '.remotes[]' <<<"$1")
  [[ "${first}" == false ]] || flow:none 'no remote'
}

# flow:incoming [--no-fetch] [--json] is the incoming stage. It fetches every remote, the primary first, each with
# git fetch --prune, and never rebases, merges, or pulls. It prints the fetch's blocks, as flow:incoming-text draws
# them. With --json it prints one object instead: fetched, true; primary, its name; and remotes, one
# flow:fetch-json object each. A remote whose fetch failed carries error, git's first line, and nothing came from it.
# With --no-fetch it fetches nothing: fetched is false, and each remote is flow:remote-json's object. It returns 1
# when a fetch failed, after every remote is tried.
flow:incoming() {
  local json=false no_fetch=false failed=false arg primary primary_default remote before said doc text
  local -a remotes=() docs=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (--no-fetch) no_fetch=true ;;
    esac
  done
  primary=$(forge:primary) || return $?
  text=$(forge:remotes) || return $?
  [[ -z "${text}" ]] || readarray -t remotes <<<"${text}"
  for remote in "${remotes[@]}"
  do
    if [[ "${no_fetch}" == true ]]
    then
      docs+=("$(flow:remote-json "${remote}")")
      continue
    fi
    before=$(flow:refs "${remote}")
    if said=$(git fetch --quiet --prune --end-of-options "${remote}" 2>&1 >/dev/null)
    then
      if [[ "${remote}" == "${primary}" ]]
      then
        primary_default=$(git:default-branch)
        docs+=("$(flow:fetch-json "${remote}" "${before}" "$(flow:refs "${remote}")" "${primary_default}")")
      else
        docs+=("$(flow:fetch-json "${remote}" "${before}" "$(flow:refs "${remote}")" \
          "$(flow:remote-default "${remote}" "${primary_default:-$(git:default-branch)}")")")
      fi
    else
      failed=true
      said=$(grep --max-count=1 . <<<"${said}" || true)
      docs+=("$(flow:fetch-json "${remote}" '' '' '' \
        | jq --compact-output --arg error "${said:-git fetch failed}" '. + {error: $error}')")
    fi
  done
  doc=$(printf '%s\n' "${docs[@]}" | jq --slurp --compact-output --argjson fetched "$([[ "${no_fetch}" == true ]] \
    && printf false || printf true)" --arg primary "${primary}" \
    '{fetched: $fetched, primary: $primary, remotes: .}')
  if [[ "${json}" == true ]]
  then
    printf '%s\n' "${doc}"
  else
    flow:incoming-text "${doc}"
  fi
  [[ "${failed}" == false ]]
}

# flow:choice <mine> <others> <owners> names what the owner flags chose, for the wording and the owner column: mine when
# --mine stands alone (the tables leave the owner column out), others when --others does without --mine, owner when
# logins were named, with --mine or not, and all otherwise, --mine with --others included.
flow:choice() {
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

# flow:owner-flags <argument>… reads the owner flags among the arguments into mine, others, and owners, variables
# the caller declares.
flow:owner-flags() {
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

# flow:choice-of <argument>… prints flow:choice's answer for the owner flags among the arguments.
flow:choice-of() {
  local mine others
  local -a owners
  flow:owner-flags "$@"
  flow:choice "${mine}" "${others}" "${#owners[@]}"
}

# flow:filter-of <argument>… prints flow:filter's line for the owner flags among the arguments.
flow:filter-of() {
  local mine others
  local -a owners
  flow:owner-flags "$@"
  flow:filter "${mine}" "${others}" "${owners[@]}"
}

# flow:filter <mine> <others> <login>… prints, on one line, the arguments flow:select and backlog:collect take to
# keep the rows the owner flags ask for, as the union of the filters: --except and the caller's login when --others
# is given without --mine and the caller is not among the logins; else the logins named, with the caller's under
# --mine; nothing when every row is kept. It asks the forge for the caller's login only when a flag needs it.
flow:filter() {
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

# flow:select [--except] <login>… reads pull request documents on stdin and keeps those opened by one of the
# named authors, or with --except, those opened by none of them. With no login, it keeps every one.
flow:select() {
  local except=false
  [[ "${1:-}" == --except ]] && { except=true; shift; }
  if (( $# == 0 ))
  then cat
  else
    jq --compact-output -L "${kit_jq}" --argjson except "${except}" \
      'import "flow" as flow; flow::by_authors($ARGS.positional; $except)' --args "$@"
  fi
}

# flow:fork <branch> <default-branch> prints the commit where <branch> left the default branch at the primary
# remote, or else the local one, or nothing.
flow:fork() {
  local primary
  primary=$(forge:primary) || return $?
  git merge-base --end-of-options "${primary}/$2" "$1" 2>/dev/null \
    || git merge-base --end-of-options "$2" "$1" 2>/dev/null || true
}

# flow:relative <path> prints <path> relative to the directory the command was run from: ".", a path down, or
# a path up through ../. Both paths are resolved physically (the target through its nearest existing directory),
# so a symlinked directory on either side still gives the right ../ path.
flow:relative() {
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

# flow:checkout <path> <branch> <default-branch> prints one JSON object that measures the checkout at <path>.
# The fork point is where the branch left the primary remote's default branch. The fields:
#
#   path, relative      the checkout's path, and that path relative to the caller's directory
#   branch, detached    the branch (empty when detached), and whether the checkout is detached
#   ahead, behind       the commits on the branch past the fork point, and on the primary's default branch past it
#   added, removed      the lines the working tree adds and removes against the fork point
#   uncommitted         the number of uncommitted paths
#   changes             each uncommitted path as {status, path}, with git's two-letter status
#   commits             the commits past the fork point, as [{sha, subject}]
#
# A detached checkout measures its lines against its own HEAD and lists no commits.
flow:checkout() {
  local path=$1 branch=$2 default_branch=$3 fork='' ahead=0 behind=0 added=0 removed=0 changes a r _ primary
  primary=$(forge:primary) || return $?
  [[ -n "${branch}" ]] && fork=$(flow:fork "${branch}" "${default_branch}")
  if [[ -n "${fork}" ]]
  then
    ahead=$(git rev-list --count "${fork}..${branch}" 2>/dev/null || printf 0)
    behind=$(git rev-list --count "${fork}..${primary}/${default_branch}" 2>/dev/null || printf 0)
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
    --arg path "${path}" --arg relative "$(flow:relative "${path}")" --arg branch "${branch}" \
    --argjson ahead "${ahead}" --argjson behind "${behind}" --argjson added "${added}" --argjson removed "${removed}" \
    --arg changes "${changes}" \
    'import "flow" as flow; flow::checkout($ARGS.named | del(.changes); $changes)'
}

# flow:label <relative> <branch> prints a checkout's label and its plain width, tab-separated. The label is:
#
#   - for a detached checkout: its path
#   - at ".": the branch alone
#   - when the path's last component is the branch or ends in "-<branch>": the path, with the directory dim and
#     the branch underlined
#   - otherwise: "<branch> on <path>"
flow:label() {
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
    printf '%s%s\t%s\n' "$(flow:paint 90 "${dir}")" "$(flow:paint 4 "${branch}")" "${#relative}"
  else
    printf '%s %s %s\t%s\n' "${branch}" "$(flow:paint 90 on)" "${relative}" $(( ${#branch} + 4 + ${#relative} ))
  fi
}

# flow:paint-count <count> <sign> <sgr> <width> <side> prints <sign><count> padded to <width> characters,
# aligned to <side> (left or right). The text is in <sgr> above zero and dim at zero. It pads by characters,
# because the arrows are multibyte and printf's field width counts bytes.
flow:paint-count() {
  local count=$1 sign=$2 sgr=$3 width=$4 side=$5 text pad=''
  (( count > 0 )) || sgr=90
  text="${sign}${count}"
  (( ${#text} < width )) && printf -v pad '%*s' $(( width - ${#text} )) ''
  if [[ "${side}" == right ]]
  then
    printf '%s%s' "${pad}" "$(flow:paint "${sgr}" "${text}")"
  else
    printf '%s%s' "$(flow:paint "${sgr}" "${text}")" "${pad}"
  fi
}

# flow:measures <checkout-json> [<widths>] prints "↑a ↓b · +x -y", then " · n uncommitted" in yellow when n is
# above zero. <widths> is flow:count-widths' line. With it, ahead and added are right-aligned, and behind and
# removed left-aligned, in columns of those widths, so the dots line up across rows.
flow:measures() {
  local ahead behind added removed uncommitted ahead_w=0 behind_w=0 added_w=0 removed_w=0
  IFS=$'\t' read -r ahead behind added removed uncommitted \
    < <(jq --raw-output '[.ahead, .behind, .added, .removed, .uncommitted] | @tsv' <<<"$1")
  [[ -n "${2:-}" ]] && IFS=' ' read -r ahead_w behind_w added_w removed_w <<<"$2"
  # The last column is padded only when something follows it, so no row ends in trailing spaces.
  (( uncommitted > 0 )) || removed_w=0
  printf '%s %s%s%s %s' \
    "$(flow:paint-count "${ahead}" '↑' '1;32' "${ahead_w}" right)" \
    "$(flow:paint-count "${behind}" '↓' '1;31' "${behind_w}" left)" \
    "${flow_sep}" \
    "$(flow:paint-count "${added}" '+' '1;32' "${added_w}" right)" \
    "$(flow:paint-count "${removed}" '-' '1;31' "${removed_w}" left)"
  (( uncommitted > 0 )) && printf '%s%s' "${flow_sep}" "$(flow:paint 33 "${uncommitted} uncommitted")"
  return 0
}

# flow:count-widths <checkout-json>… prints the widest "↑a", "↓b", "+x", and "-y" across the checkouts, as four
# numbers on one line.
flow:count-widths() {
  printf '%s\n' "$@" | jq -L "${kit_jq}" --raw-output --slurp 'import "flow" as flow; flow::widths'
}

# flow:checkout-block <glyph> <checkout-json> <label-width> [<widths>] prints a checkout:
#
#   1. its row, with the counts in flow:count-widths' columns
#   2. its uncommitted paths, each with its status in yellow
#   3. its commits, each with its hash dim
#
# Past flow_commit_cap, it prints the first paths and a count of the rest. For commits, it prints the newest
# ones, a dim count of the hidden ones, and the oldest one.
flow:checkout-block() {
  local glyph=$1 doc=$2 label_w=$3 widths=${4:-} relative branch label width mark='' sha subject code path
  IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
  IFS=$'\t' read -r label width < <(flow:label "${relative}" "${branch}")
  [[ -n "${branch}" ]] || mark="${flow_gap}$(flow:paint 33 '❗ detached HEAD')"
  printf '%s %s%*s%s%s%s\n' "${glyph}" "${label}" $(( label_w - width )) '' "${mark}" "${flow_sep}" \
    "$(flow:measures "${doc}" "${widths}")" >&2
  # A status can start with a space, which a tab-separated read would drop, so the separator is \x1f instead.
  while IFS=$'\x1f' read -r code path
  do
    [[ -n "${path}" ]] || continue
    if [[ "${code}" == '…' ]]
    then
      printf '   %s\n' "$(flow:paint 90 "… ${path}")" >&2
    else
      printf '   %s %s\n' "$(flow:paint 33 "${code}")" "${path}" >&2
    fi
  done < <(jq -L "${kit_jq}" --raw-output --argjson cap "${flow_commit_cap}" \
    'import "flow" as flow; flow::changes_shown($cap) | join("\u001f")' <<<"${doc}")
  while IFS=$'\t' read -r sha subject
  do
    [[ -n "${sha}" ]] || continue
    if [[ "${sha}" == '…' ]]
    then
      printf '   %s\n' "$(flow:paint 90 "… ${subject}")" >&2
    else
      printf '   %s %s\n' "$(flow:paint 90 "${sha}")" "${subject}" >&2
    fi
  done < <(jq -L "${kit_jq}" --raw-output --argjson cap "${flow_commit_cap}" \
    'import "flow" as flow; flow::commits_shown($cap) | @tsv' <<<"${doc}")
}

# flow:check-cell <checks-json> prints the checks cell, with the duration dim:
#
#   - passed: "✅ <passed> / <total> <took>"
#   - failed: "❌ <failing> failing <passed> / <total> <took>"
#   - running: "🏃 <passed> / <total> <elapsed>"
#
# <took> is the slowest check's own duration, and <elapsed> how long the pending one has waited. A check that ran
# again counts once, by its latest run.
#
# It prints nothing when no check has reported.
flow:check-cell() {
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
  [[ -n "${took}" ]] && printf ' %s' "$(flow:paint 90 "$(flow:age "${took}")")"
  return 0
}

# flow:issue-widths <issues-json> reads a JSON array of issue documents and prints the widths of the widest number,
# assignees, age, and branch across them, space-separated: zero where no issue has one, so the column is left out.
flow:issue-widths() {
  local ref assignees age branch text ref_w=0 owner_w=0 age_w=0 branch_w=0
  while IFS=$'\x1f' read -r ref assignees age branch
  do
    [[ -n "${ref}" ]] || continue
    (( ${#ref} > ref_w )) && ref_w=${#ref}
    (( ${#assignees} > owner_w )) && owner_w=${#assignees}
    text=$(flow:age "${age}")
    (( ${#text} > age_w )) && age_w=${#text}
    (( ${#branch} > branch_w )) && branch_w=${#branch}
  done < <(jq --raw-output '.[] | [(.ref // "#\(.number)"), ((.assignees // []) | join(" ")),
                                   (.age // "" | tostring), (.branch // "")] | join("\u001f")' <<<"$1")
  printf '%s %s %s %s\n' "${ref_w}" "${owner_w}" "${age_w}" "${branch_w}"
}

# flow:issue-row <issue-json> <widths> <state> <indent> prints one issue as the backlog and changes stages draw
# it: <indent>, the number right-aligned, the assignees when any issue in the table has one, then <state> with the
# age when <state> is given, as under a change, else the age alone, then the branch when any issue has one, and last
# the title, clipped to what the terminal's width leaves. <widths> is flow:issue-widths' line. Every cell after
# the number opens with a glyph, so flow_gap separates them, and a cell with nothing is blank, glyph included, so
# every title starts at the same column. The number is cyan; the state is blue, the family of a move that is
# someone else's; the detail is dim.
flow:issue-row() {
  local doc=$1 state=$3 indent=$4 ref_w owner_w age_w branch_w ref title assignees age branch row glyph width text
  read -r ref_w owner_w age_w branch_w <<<"$2"
  IFS=$'\x1f' read -r ref title assignees age branch \
    < <(jq --raw-output '[(.ref // "#\(.number)"), .title, ((.assignees // []) | join(" ")),
                          (.age // "" | tostring), (.branch // "")] | join("\u001f")' <<<"${doc}")
  row="${indent}$(flow:paint "${flow_issue_sgr}" "$(flow:cell "${ref}" "${ref_w}" right)")"
  width=$(( $(flow:width) - ${#indent} - ref_w - 5 ))
  if (( owner_w > 0 ))
  then
    glyph='   '
    [[ -n "${assignees}" ]] && glyph='👤 '
    row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "${assignees}" "${owner_w}")")"
    width=$(( width - 5 - owner_w ))
  fi
  text=$(flow:age "${age}")
  if [[ -n "${state}" ]]
  then
    row+="${flow_gap}🛫 $(flow:paint 34 "${state}")"
    width=$(( width - 5 - ${#state} ))
    if [[ -n "${text}" ]]
    then
      row+=" $(flow:paint 90 "$(flow:cell "${text}" "${age_w}" right)")"
      width=$(( width - 1 - age_w ))
    fi
  elif (( age_w > 0 ))
  then
    row+="${flow_gap}$(flow:paint 90 "$(flow:cell "${text}" "${age_w}" right)")"
    width=$(( width - 2 - age_w ))
  fi
  if (( branch_w > 0 ))
  then
    if [[ -n "${branch}" ]]
    then row+="${flow_gap}$(flow:paint 90 "🌿 $(flow:cell "${branch}" "${branch_w}")")"
    else row+="${flow_gap}   $(flow:cell '' "${branch_w}")"
    fi
    width=$(( width - 5 - branch_w ))
  fi
  (( width < 20 )) && width=20
  (( ${#title} > width )) && title="${title:0:width-1}…"
  printf '%s%s🎫 %s\n' "${row}" "${flow_gap}" "${title}" >&2
}

# flow:pr-block <pr-json> <title-width> <owner-width> <state-width> <age-width> <with-branch> <issue-widths>
# prints a pull request: its row, then the issues it closes, one row each, in flight, indented under it. The state
# and its age share a cell. The row shows the author before the state when <owner-width> is above zero, and ends
# with the branch when <with-branch> is not empty. <issue-widths> is flow:issue-widths' line over every issue in
# the table.
flow:pr-block() {
  local doc=$1 title_w=$2 owner_w=$3 state_w=$4 age_w=$5 worktree=$6 issue_widths=$7
  local number ref title author branch state age unresolved row issue checks
  IFS=$'\x1f' read -r number ref title author branch state age unresolved \
    < <(jq --raw-output '[.number, (.ref // "#\(.number)"), .title, .author, .branch, .state, .age, .unresolved]
          | map(tostring)
          | join("\u001f")' <<<"${doc}")
  row="🔀 $(flow:paint "${flow_change_sgr}" "${ref}") $(flow:cell "${title}" $(( title_w - ${#ref} - 1 )))"
  (( owner_w > 0 )) && row+="${flow_gap}👤 $(flow:paint 90 "$(flow:cell "${author}" "${owner_w}")")"
  row+="${flow_gap}$(flow:state-emoji "${state}")"
  row+=" $(flow:paint "$(flow:state-sgr "${state}")" "$(flow:cell "${state}" "${state_w}")")"
  row+=" $(flow:paint 90 "$(flow:cell "$(flow:age "${age}")" "${age_w}" right)")"
  checks=$(flow:check-cell "$(jq --compact-output .checks <<<"${doc}")")
  [[ -n "${checks}" ]] && row+="${flow_gap}${checks}"
  (( unresolved > 0 )) && row+="${flow_gap}$(flow:paint 33 "💬 ${unresolved} unresolved")"
  [[ -n "${worktree}" ]] && row+="${flow_gap}$(flow:paint 90 "🌿 ${branch}")"
  printf '%s\n' "${row}" >&2
  while IFS= read -r issue
  do
    [[ -n "${issue}" ]] || continue
    flow:issue-row "${issue}" "${issue_widths}" 'in flight' '   '
  done < <(jq --compact-output '.closes[]?' <<<"${doc}")
  return 0
}

# flow:pr-table <choice> <prs> draws the pull requests in <prs>, one JSON document per line, one block each, a
# blank line between blocks, their cells sized across the rows: the row's title, author, state, and age, and the
# issue rows' number, assignees, and age. Every row ends with its branch. <choice> mine leaves the author column out.
flow:pr-table() {
  local choice=$1 prs=$2 doc issue_widths first=true
  local title_w=0 owner_w=0 state_w=0 age_w=0 width text author state_text age_secs
  [[ -n "${prs}" ]] || return 0
  while IFS=$'\x1f' read -r text author state_text age_secs
  do
    [[ -n "${text}" ]] || continue
    (( ${#text} > title_w )) && title_w=${#text}
    [[ "${choice}" != mine ]] && (( ${#author} > owner_w )) && owner_w=${#author}
    (( ${#state_text} > state_w )) && state_w=${#state_text}
    text=$(flow:age "${age_secs}")
    (( ${#text} > age_w )) && age_w=${#text}
  done < <(jq --raw-output '["\(.ref // "#\(.number)") \(.title)", .author, .state, .age] | map(tostring)
      | join("\u001f")' \
    <<<"${prs}")
  width=$(( $(flow:width) - 3 - 2 - 3 - state_w - 3 - age_w - 3 - 24 - (owner_w > 0 ? owner_w + 5 : 0) ))
  (( width < 20 )) && width=20
  (( title_w > width )) && title_w=${width}
  issue_widths=$(flow:issue-widths "$(jq --slurp --compact-output 'map(.closes[]?)' <<<"${prs}")")
  while IFS= read -r doc
  do
    [[ -n "${doc}" ]] || continue
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    flow:pr-block "${doc}" "${title_w}" "${owner_w}" "${state_w}" "${age_w}" yes "${issue_widths}"
  done <<<"${prs}"
}

# flow_worktrees and flow_gone hold what flow:worktrees measured: each linked worktree's document in
# issue-number order, or the one worktree's, each flow:checkout's JSON; and the branches whose upstream is gone,
# one per line, none in a worktree.
declare -g flow_gone=''
declare -ga flow_worktrees=()

# flow:desk [--json] is the desk stage. It measures the root checkout, wherever kit runs in the repository, and
# draws its block. With --json it prints the checkout's object instead.
flow:desk() {
  local main_root branch doc default_branch
  default_branch=$(git:default-branch) || return $?
  main_root=$(git:main-root)
  branch=$(git -C "${main_root}" symbolic-ref --quiet --short HEAD 2>/dev/null || true)
  doc=$(flow:checkout "${main_root}" "${branch}" "${default_branch}") || return $?
  if [[ " $* " == *" --json "* ]]
  then
    printf '%s\n' "${doc}"
  else
    flow:draw-desk "${doc}"
  fi
}

# flow:draw-desk <checkout-json> draws the desk's block.
flow:draw-desk() {
  local relative branch width
  IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"$1")
  IFS=$'\t' read -r _ width < <(flow:label "${relative}" "${branch}")
  flow:checkout-block '🏠' "$1" "${width}"
}

# flow:linked <default-branch> measures every linked worktree, from the primary's default branch, into
# flow_worktrees.
flow:linked() {
  local path branch
  flow_worktrees=()
  while IFS=$'\t' read -r path _ branch
  do
    [[ -n "${path}" ]] || continue
    flow_worktrees+=("$(flow:checkout "${path}" "${branch}" "$1")")
  done < <(git:linked-worktrees | flow:by-number)
}

# flow:draw-checkouts draws what flow:worktrees measured: each worktree's block, the labels and the counts in
# columns, a blank line between blocks.
flow:draw-checkouts() {
  local label_w=0 width relative branch widths doc first=true
  for doc in "${flow_worktrees[@]}"
  do
    IFS=$'\x1f' read -r relative branch < <(jq --raw-output '[.relative, .branch] | join("\u001f")' <<<"${doc}")
    IFS=$'\t' read -r _ width < <(flow:label "${relative}" "${branch}")
    (( width > label_w )) && label_w=${width}
  done
  widths=$(flow:count-widths "${flow_worktrees[@]}")
  for doc in "${flow_worktrees[@]}"
  do
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    flow:checkout-block '🌿' "${doc}" "${label_w}" "${widths}"
  done
}

# flow:in-worktree succeeds in a linked worktree, whose git directory is not the repository's common one.
flow:in-worktree() {
  [[ "$(git rev-parse --path-format=absolute --git-dir)" \
    != "$(git rev-parse --path-format=absolute --git-common-dir)" ]]
}

# flow:worktrees [--json] is the worktrees stage. It measures the linked worktrees and draws them: at a desk,
# every one in issue-number order, and the branches whose upstream is gone; in a worktree, that worktree alone.
# With --json it prints flow:worktrees-json's object instead.
flow:worktrees() {
  local default_branch here branch
  default_branch=$(git:default-branch) || return $?
  flow_gone='' flow_worktrees=()
  if flow:in-worktree
  then
    here=$(git rev-parse --show-toplevel)
    branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)
    flow_worktrees=("$(flow:checkout "${here}" "${branch}" "${default_branch}")")
  else
    flow:linked "${default_branch}"
    flow_gone=$(git:gone-branches)
  fi
  if [[ " $* " == *" --json "* ]]
  then
    flow:worktrees-json
  else
    flow:draw-worktrees
  fi
}

# flow:worktrees-json prints what flow:worktrees measured as one object: worktrees, the checkouts' documents;
# and goneBranches, the names.
flow:worktrees-json() {
  local gone
  gone=$(printf '%s\n' "${flow_gone}" | jq --raw-input . | jq --slurp --compact-output 'map(select(. != ""))')
  jq --null-input --compact-output \
    --argjson worktrees "$(printf '%s\n' "${flow_worktrees[@]}" | jq --slurp --compact-output .)" \
    --argjson goneBranches "${gone}" '{worktrees: $worktrees, goneBranches: $goneBranches}'
}

# flow:draw-worktrees draws what flow:worktrees measured: the checkout blocks, or a dim line that says no
# worktree; then, after a blank line, each gone branch as 🔖 <branch> · upstream gone · kit tidy removes it.
flow:draw-worktrees() {
  local branch
  if (( ${#flow_worktrees[@]} == 0 ))
  then
    flow:none 'no worktree'
  else
    flow:draw-checkouts
  fi
  [[ -n "${flow_gone}" ]] || return 0
  printf '\n' >&2
  while IFS= read -r branch
  do
    printf '🔖 %s%s%s\n' "${branch}" "${flow_sep}" \
      "$(flow:paint 90 "upstream gone${flow_sep}kit tidy removes it")" >&2
  done <<<"${flow_gone}"
}

# flow:stage <stage> [<argument>…] prints one stage, without its heading, with the arguments the stage takes.
flow:stage() {
  local stage=$1
  shift
  case "${stage}" in
    (backlog) backlog:render "$@" ;;
    (underway) underway:render "$@" ;;
    (incoming) flow:incoming "$@" ;;
    (desk) flow:desk "$@" ;;
    (worktrees) flow:worktrees "$@" ;;
    (changes) changes:render "$@" ;;
    (builds) builds:render "$@" ;;
    (releases) releases:render "$@" ;;
  esac
}

# flow:stage-args <stage> <argument>… prints, one per line, the arguments among those given that the stage takes:
# --json for every stage, --no-fetch for incoming, and the owner flags for backlog, underway, and changes.
flow:stage-args() {
  local stage=$1 arg
  shift
  for arg in "$@"
  do
    case "${arg}" in
      (--json) printf '%s\n' "${arg}" ;;
      (--no-fetch) [[ "${stage}" != incoming ]] || printf '%s\n' "${arg}" ;;
      (--all | --mine | --others | --owner=?*)
        list:among "${stage}" backlog underway changes && printf '%s\n' "${arg}"
    esac
  done
  return 0
}

# flow:picture-document <scratch> <stage> <reachable> prints the document a stage collected under <scratch>, for the
# picture's JSON: null for a stage that failed or, with no primary remote, could not run. incoming has a document
# even when a fetch failed. With no primary remote, it is the remotes, not fetched.
flow:picture-document() {
  local scratch=$1 stage=$2 reachable=$3
  if [[ "${reachable}" == false && "${stage}" == incoming ]]
  then
    flow:incoming --no-fetch --json
  elif [[ "${stage}" == incoming && -s "${scratch}/${stage}.json" ]]
  then
    cat "${scratch}/${stage}.json"
  elif [[ -f "${scratch}/${stage}.status" && "$(<"${scratch}/${stage}.status")" == 0 ]]
  then
    cat "${scratch}/${stage}.json"
  else
    printf 'null\n'
  fi
}

# flow:picture [--no-fetch] [--json] [<owner flag>…] draws every stage the flow setting names, in its order, or
# prints the picture as one JSON object. It collects first and draws after: the forge stages run in the background
# while the remotes are fetched; then the checkouts are measured; then each stage is drawn in order, under its
# heading. A stage that failed is drawn as its refusal, and the picture returns 1 at the end, in either form. So does
# a fetch that failed: incoming draws it on its remote's line, and without incoming, the primary is fetched alone and
# a failure warns once. With no primary remote it fetches nothing and asks the forge nothing: it draws the checkouts,
# says so once, and returns 0. It returns 2 when the flow setting is wrong.
flow:picture() {
  local json=false no_fetch=false reachable=true failed=false first=true stage doc code choice scratch text primary
  local -a stages=() args=() named=()
  text=$(settings:stages) || return 2
  readarray -t stages <<<"${text}"
  [[ " $* " == *" --json "* ]] && json=true
  [[ " $* " == *" --no-fetch "* ]] && no_fetch=true
  primary=$(forge:primary) || return $?
  git remote get-url --end-of-options "${primary}" >/dev/null 2>&1 || reachable=false
  choice=$(flow:choice-of "$@")
  file:scratch scratch
  # The forge stages need nothing local and the forge is the slow call, so they start first, in the background.
  for stage in "${stages[@]}"
  do
    case "${stage}" in
      (backlog | underway | changes | builds | releases)
        [[ "${reachable}" == true ]] || continue
        readarray -t args < <(flow:stage-args "${stage}" --json "$@")
        (
          flow:stage "${stage}" "${args[@]}" && code=0 || code=$?
          printf '%s\n' "${code}" > "${scratch}/${stage}.status"
        ) > "${scratch}/${stage}.json" 2> "${scratch}/${stage}.err" &
    esac
  done
  # The fetch comes before the checkouts are measured, so ahead and behind count against the primary as it is now.
  if [[ "${reachable}" == false ]]
  then
    true
  elif list:among incoming "${stages[@]}"
  then
    readarray -t args < <(flow:stage-args incoming --json "$@")
    flow:incoming "${args[@]}" > "${scratch}/incoming.json" 2> "${scratch}/incoming.err" && code=0 || code=$?
    printf '%s\n' "${code}" > "${scratch}/incoming.status"
  elif [[ "${no_fetch}" == false ]] && ! git fetch --quiet --prune --end-of-options "${primary}" 2>/dev/null
  then
    warn "git fetch --prune failed; ahead and behind counts may be stale"
    failed=true
  fi
  for stage in desk worktrees
  do
    list:among "${stage}" "${stages[@]}" || continue
    flow:stage "${stage}" --json > "${scratch}/${stage}.json" 2> "${scratch}/${stage}.err" && code=0 || code=$?
    printf '%s\n' "${code}" > "${scratch}/${stage}.status"
  done
  wait
  # One rule for both forms: a stage that ran and failed makes the picture return 1, as a fetch that failed does. A
  # stage that did not run, with no primary remote, is no failure.
  for stage in "${stages[@]}"
  do
    [[ ! -f "${scratch}/${stage}.status" || "$(<"${scratch}/${stage}.status")" == 0 ]] || failed=true
  done
  if [[ "${json}" == true ]]
  then
    for stage in "${stages[@]}"
    do
      doc=$(flow:picture-document "${scratch}" "${stage}" "${reachable}")
      named+=(--argjson "${stage}" "${doc}")
      [[ ! -s "${scratch}/${stage}.err" ]] || cat "${scratch}/${stage}.err" >&2
    done
    [[ "${reachable}" == true ]] || warn "no ${primary}: nothing fetched, no pull request shown"
    jq --null-input --compact-output --argjson stages "$(flow:json-list "${stages[@]}")" "${named[@]}" \
      '{stages: $stages} + $ARGS.named'
    [[ "${failed}" == false ]]
    return
  fi
  if [[ "${reachable}" == false ]]
  then
    warn "no ${primary}: nothing fetched, no pull request shown"
    first=false
  fi
  for stage in "${stages[@]}"
  do
    [[ "${reachable}" == true || "${stage}" == desk || "${stage}" == worktrees ]] || continue
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    code=$(<"${scratch}/${stage}.status")
    # A fetch that failed is drawn on its remote's line, beside the remotes that were fetched.
    if (( code != 0 )) && [[ "${stage}" != incoming || ! -s "${scratch}/${stage}.json" ]]
    then
      flow:heading "${stage}"
      cat "${scratch}/${stage}.err" >&2
      continue
    fi
    doc=$(<"${scratch}/${stage}.json")
    case "${stage}" in
      (backlog)
        flow:heading backlog "$(jq 'length' <<<"${doc}")"
        backlog:draw "${doc}" "${choice}"
        ;;
      (underway)
        flow:heading underway "$(jq 'length' <<<"${doc}")"
        underway:draw "${doc}" "${choice}"
        ;;
      (incoming)
        flow:heading incoming
        flow:incoming-text "${doc}"
        ;;
      (desk)
        flow:heading desk 1
        flow:draw-desk "${doc}"
        ;;
      (worktrees)
        flow:heading worktrees "$(jq '.worktrees | length' <<<"${doc}")"
        flow:draw-worktrees
        ;;
      (changes)
        flow:heading changes "$(jq 'length' <<<"${doc}")"
        changes:draw "${doc}" "${choice}"
        ;;
      (builds)
        flow:heading builds "$(jq 'length' <<<"${doc}")"
        builds:draw "${doc}"
        ;;
      (releases)
        flow:heading releases "$(jq 'length' <<<"${doc}")"
        releases:draw "${doc}"
        ;;
    esac
    [[ ! -s "${scratch}/${stage}.err" ]] || cat "${scratch}/${stage}.err" >&2
  done
  [[ "${failed}" == false ]]
}

# flow:render [<stage>] [<argument>…] draws one stage, or the picture, as kit flow's page says. Above desks it
# draws each desk's under its path. --text, which kit flow takes to stay in text on a terminal, changes nothing
# here.
flow:render() {
  local stage='' arg
  local -a args=()
  if ! git rev-parse --git-dir >/dev/null 2>&1
  then
    flow:above "$@"
    return
  fi
  case "${1:-}" in
    (backlog | underway | desk | worktrees | changes | incoming | builds | releases)
      stage=$1
      shift
  esac
  for arg in "$@"
  do
    [[ "${arg}" == --text ]] || args+=("${arg}")
  done
  if [[ -n "${stage}" ]]
  then
    flow:stage "${stage}" "${args[@]}"
  else
    flow:picture "${args[@]}"
  fi
}

# flow:desks <dir> <depth> prints every desk at or below <dir>, at most <depth> levels down, in path order: a
# directory that holds a .git directory. The walk stops at a desk, so a worktree or a nested repository inside one
# is the desk's own, and it skips hidden directories and symbolic links.
flow:desks() {
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
    flow:desks "${sub}" $(( depth - 1 ))
  done
}

# flow:above <argument>… draws every desk below the current directory, each under its path, as kit flow draws it
# at that desk, with the same arguments. The desks run at once, each fetching on its own, and are drawn in path
# order.
flow:above() {
  local json=false arg desk index=0 color=''
  local -a desks
  for arg in "$@"
  do
    [[ "${arg}" == --json ]] && json=true
  done
  readarray -t desks < <(flow:desks "$(pwd -P)" "${flow_walk_depth}")
  if (( ${#desks[@]} == 0 ))
  then
    error "no repository at or below $(pwd -P), ${flow_walk_depth} levels down; run kit flow in one"
    return 1
  fi
  local scratch
  file:scratch scratch
  [[ "${out_color}" == 1 ]] && color=1
  for desk in "${desks[@]}"
  do
    ( cd "${desk}" && COLUMNS=$(flow:width) CLICOLOR_FORCE=${color} run kit flow "$@" ) \
      > "${scratch}/${index}.out" 2> "${scratch}/${index}.err" &
    index=$(( index + 1 ))
  done
  wait || true
  if [[ "${json}" == true ]]
  then
    for index in "${!desks[@]}"
    do
      jq --compact-output --arg path "$(flow:relative "${desks[${index}]}")" '{path: $path, status: .}' \
        "${scratch}/${index}.out" 2>/dev/null || jq --null-input --compact-output \
        --arg path "$(flow:relative "${desks[${index}]}")" '{path: $path, status: null}'
      cat "${scratch}/${index}.err" >&2
    done | jq --slurp .
    return 0
  fi
  for index in "${!desks[@]}"
  do
    (( index == 0 )) || printf '\n' >&2
    printf '%s\n' "$(flow:paint 1 "📂 $(flow:relative "${desks[${index}]}")")" >&2
    cat "${scratch}/${index}.err" "${scratch}/${index}.out" >&2
  done
}

# flow:by-number sorts git:worktrees lines by the issue number their directory name starts with. A directory
# name that does not start with a number sorts first.
flow:by-number() {
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
