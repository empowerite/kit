# shellcheck shell=bash
# flow.sh - the picture of the work: five stages, drawn one at a time or all together (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# kit flow draws the picture. Work moves left to right through five stages:
#
#   issues → changes → builds → incoming → releases
#
# The flow setting in kit.yml names the stages the picture draws, in order. flow:render draws one stage, or
# the picture: every stage under its heading: its glyph, its name, and its count, dim. The rows of issues, changes,
# builds, and releases live in the modules named after them. The incoming stage, and the cells
# every table shares, are here. The desk and the worktrees are kit workspace worktrees', in modules/bash/workspace.sh.
#
# Where it runs decides what changes draws:
#
#   - At a desk, the root checkout: every open pull request.
#   - In a linked worktree: the pull request its branch holds, whoever opened it. The owner flags never hide it.
#   - Above desks, in a directory inside no repository's working tree, a bare repository or a .git directory too:
#     it walks down. Every directory that holds a .git directory is a desk. The walk stops there, and draws that
#     desk's picture under its path, with the same arguments. It skips hidden directories and symbolic links, and
#     goes no deeper than flow_walk_depth. The desks are drawn in path order, each with a fetch of its own.
#
# issues, incoming, builds, and releases draw the same anywhere in a repository.
#
# The primary remote is the primary setting in kit.yml, else origin; modules/bash/forge.sh says how. The incoming
# stage fetches every remote, and the rest measure against the primary.
#
# The picture collects first and draws after. The forge stages run in the background while the remotes are fetched.
# Then every stage is drawn in order. A repository with no primary remote fetches nothing, asks no forge, and says so
# once.
#
# Which pull requests and issues are shown:
#
#   --all              every one (the default)
#   --mine             the pull requests the caller opened, and the issues assigned to the caller; no owner column
#   --others           anyone else's; unassigned issues included
#   --owner=<login>    the named login's; repeat the flag for more, or join the logins with :, --owner=a:b
#
# A pull request's state is the first of these that applies. Its color shows the state's family, never the
# single state: green when nothing is left to decide, yellow when a human owes a decision, and blue when the next
# move is a machine's or a reviewer's. Detail is dim. An issue's number is always cyan and a change's magenta,
# the two colors no family uses, so either reads as what it is in every table the kit draws. Each state's glyph is
# modules/jq/glyphs.json's, as every glyph kit flow draws is.
#
#   state              color
#   draft              dim
#   queued             blue
#   conflicted         yellow
#   failing:gate       yellow
#   no-check           blue
#   comments:open      yellow
#   blocked:approval   blue
#   ready              green
#
# --json prints the picture as one object: stages, the names in order, and under each name what kit flow <stage>
# --json prints. A stage that could not be read is null. Above desks it is an array of {path, status}, one per desk.
#
# see: kit/bin/kit-flow, modules/bash/issues.sh, modules/bash/changes.sh, modules/bash/builds.sh,
# modules/bash/releases.sh

use file git settings workspaces issues changes builds releases

# flow_walk_depth is how many directories below the one it runs in kit flow looks for desks, above desks.
flow_walk_depth=4

# flow_sep separates the cells of every row kit flow prints, and flow_gap replaces it before a cell that
# opens with a glyph, which marks the boundary already. A blank cell that stands in for one takes the gap too.
flow_sep=' · '
flow_gap='  '

# flow_issue_sgr and flow_change_sgr color an issue's number and a change's, in every table the kit draws.
flow_issue_sgr=36
flow_change_sgr=35

# The glyphs are modules/jq/glyphs.json's, which kit help flow's legend and kittui's are made from too.
# kit:generate glyphs `k=$(command -v kit); jq -nr -L "${k%/bin/kit}/modules/jq" 'import "glyphs" as g; g::bash'`
# flow_glyphs gives each glyph by its name, flow_stage_glyphs each stage's by the stage, and flow_state_glyphs a
# pull request state's by its word; a state word not among them draws flow_glyphs[Other].
declare -gA flow_glyphs=(
  [Workspace]='📚'
  [Issue]='🎫'
  [GitHub]='🐙'
  [GitLab]='🦊'
  [Waiting]='💤'
  [InFlight]='🛫'
  [Desk]='💻'
  [Branch]='🌿'
  [Sprout]='🌱'
  [Worktree]='🌳'
  [Commit]='🔘'
  [Default]='⚓'
  [Change]='🔀'
  [Owner]='👤'
  [Fork]='🍴'
  [Pushed]='📤'
  [Path]='📂'
  [Fetched]='🛬'
  [Remote]='📡'
  [Gone]='🔖'
  [Tag]='📌'
  [Behind]='📥'
  [Detached]='❗'
  [Threads]='💬'
  [Running]='🏃'
  [Passed]='✅'
  [Failed]='❌'
  [Cancelled]='🛑'
  [Build]='🏭'
  [Release]='📦'
  [Pre]='🧪'
  [Draft]='📝'
  [Latest]='⭐'
  [Asset]='📎'
  [Package]='🧩'
  [Active]='🟢'
  [Excluded]='⛔'
  [Milestone]='⛳'
  [Iteration]='📅'
  [Swimlane]='🏊'
  [Review]='🧐'
  [Check]='🩺'
  [Merged]='🤝'
  [Artifact]='📀'
  [Environment]='🏢'
  [Deployment]='🚀'
  [Person]='🧑'
  [Team]='👪'
  [Agent]='🤖'
  [Evidence]='🧾'
  [Finding]='🐛'
  [Control]='🔒'
  [Waiver]='🔓'
  [Policy]='📏'
  [Enforcement]='🔐'
  [Service]='🔌'
  [Toolchain]='🧰'
  [Usable]='✅'
  [Inside]='⚡'
  [InUse]='📌'
  [Adds]='➕'
  [Tool]='🔩'
  [Setting]='🔧'
  [Other]='🔳'
)
declare -gA flow_stage_glyphs=(
  [issues]='🎫'
  [desk]='💻'
  [worktrees]='🌿'
  [changes]='🔀'
  [builds]='🏭'
  [incoming]='🛬'
  [releases]='📦'
)
declare -gA flow_state_glyphs=(
  [draft]='🔨'
  [queued]='🚦'
  [conflicted]='💥'
  [failing:gate]='🔥'
  [no-check]='📭'
  [comments:open]='💬'
  [blocked:approval]='👀'
  [ready]='🏁'
)
# kit:end glyphs

# flow:stage-purpose <stage> prints what a stage lists, in a few words.
flow:stage-purpose() {
  case "$1" in
    (issues) printf 'every open issue, oldest first, in the backlog or underway: who works on each, and where\n' ;;
    (incoming) printf 'what the fetch brought: commits on the default branch, branches new and gone\n' ;;
    (changes) printf 'the open pull requests, each with the issues it closes\n' ;;
    (builds) printf 'the runs not owned by any pull request: the default branch, tags, schedules\n' ;;
    (releases) printf 'the releases on the forge, newest first\n' ;;
  esac
}

# flow:choices <key> [<answer>…] answers kit syntax --choices for <stage>: the five stages, a closed set, each with
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
    cols=$(stty size <&2 2>/dev/null | cut -d' ' -f2 || true)  # a size stty cannot read gives the default
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
  local glyph=${flow_stage_glyphs[$1]:-} line
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
  printf '%s' "${flow_state_glyphs[$1]:-${flow_glyphs[Other]}}"
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
      # A range git cannot walk, from an old commit that is not here as in a shallow clone, lists no commit; the
      # picture says only that the branch moved, as an updated branch.
      (moved:"${default_ref}")
        commits=$(git log --format='%h%x09%s' "${old}..${new}" 2>/dev/null) || updated+=("$4") ;;
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

# flow:tip <rev> prints a commit's facts, unit-separated: its short hash, its author, its committer time in seconds
# since the epoch, and its subject; the rev cut to seven characters and nothing else when git does not have it.
flow:tip() {
  git log -1 --format='%h%x1f%an%x1f%ct%x1f%s' "$1" -- 2>/dev/null || printf '%s\x1f\x1f\x1f\n' "${1:0:7}"
}

# flow:row <field>… prints one row for flow:incoming-rows to read: its fields unit-separated, a list's items in one
# field record-separated.
flow:row() {
  local IFS=$'\x1f'
  printf '%s\n' "$*"
}

# flow:on-branch <upstream> prints, for flow:incoming-rows, the local branches on a remote branch and the linked
# worktrees they are checked out in: two fields, each a list, from flow:incoming-rows' locals and trees.
flow:on-branch() {
  local names=${locals[$1]:-} each wt=''
  for each in ${names//$'\x1e'/ }
  do
    [[ -z "${trees[${each}]:-}" ]] || wt+="${trees[${each}]}"$'\x1e'
  done
  printf '%s\x1f%s' "${names}" "${wt}"
}

# flow:incoming-rows <remote-json> <primary> <default-branch> <state> [<diff>] prints what is incoming from a remote as
# rows, one JSON object a line, every row with the remote's facts: remote, its name; primary, "primary" for the
# primary, else null; forge, host, and url, its fetch URL; and default, its default branch. kind says what the row is:
#   remote    the remote itself, once: change is fetched, unchanged, not fetched, or failed, and subject git's line
#             when it failed
#   commit    a commit new on the default branch, newest first: change is new
#   branch    a branch on the remote: change is new, updated, forced (its old tip is not under its new one), or gone;
#             commits counts the commits it moved by, or, new, those not on the default branch
#   tag       a tag the fetch brought: change is new, or moved where it names another commit
#   tracking  a local branch whose upstream on the remote is ahead of it: change is behind, and ahead and behind
#             count against it; this needs no fetch, so --no-fetch shows it as the last fetch left it
# Each has its key, the remote, its kind, and its ref or hash; ref, the branch or tag; sha and old, the short hashes of
# its commit and of the one before, where it has them; author, age in seconds, and subject, its commit's, or a tag's
# own; issues, the issues its branch's name starts with and its subject names, as #12; local, the local branches on it;
# worktrees, the linked worktrees those are checked out in; and ahead and behind, a tracking row's counts. A field a row
# has no value for is null, or an empty list. The rows stand in the order of the kinds above, each kind's in git's.
# <diff> is flow:fetch-diff's, for a fetch that ran.
flow:incoming-rows() {
  local remote_json=$1 primary=$2 default=$3 state=$4 diff=${5:-} remote kind ref old new name upstream path key value
  local base change commits url counts ahead behind desk='' sep=$'\x1f' items=$'\x1e' line
  local -a heads=() porcelain=()
  local -A locals=() trees=()
  remote=$(jq --raw-output '.name' <<<"${remote_json}")
  base="refs/remotes/${remote}/${default}"
  # a remote with no url to fetch from, as one only pushed to, has none to name
  url=$(git remote get-url --end-of-options "${remote}" 2>/dev/null || true)
  # The local branches on each of the remote's branches, and the linked worktree each local branch is checked out in.
  list:read heads git for-each-ref --format="%(refname:short)${sep}%(upstream)" refs/heads || return 1
  for line in "${heads[@]}"
  do
    IFS="${sep}" read -r name upstream <<<"${line}"
    [[ "${upstream}" == "refs/remotes/${remote}/"* ]] && locals[${upstream}]+="${name}${items}"
  done
  list:read porcelain git worktree list --porcelain || return 1
  for line in "${porcelain[@]}"
  do
    read -r key value <<<"${line}"
    case "${key}" in
      (worktree) path=${value} desk=${desk:-${value}} ;;
      (branch) [[ "${path}" == "${desk}" ]] || trees[${value#refs/heads/}]=${path} ;;
    esac
  done
  {
    change=${state}
    [[ "${state}" != fetched || -n "${diff}" ]] || change=unchanged
    flow:row remote "${change}" "${default}" '' '' '' "$(jq --raw-output '.error // ""' <<<"${remote_json}")" '' '' '' \
      '' '' ''
    while IFS=' ' read -r kind ref old new
    do
      [[ -n "${kind}" ]] || continue
      case "${kind}:${ref}" in
        (moved:"${base}")
          if git log --format="commit${sep}new${sep}${default}${sep}%h${sep}%an${sep}%ct${sep}%s${sep}${sep}${sep}${sep}${sep}${sep}" \
            "${old}..${new}" -- 2>/dev/null
          then
            git merge-base --is-ancestor "${old}" "${new}" 2>/dev/null \
              || flow:row branch forced "${default}" "$(flow:tip "${new}")" "${old:0:7}" '' "$(flow:on-branch "${base}")" '' ''
          else
            flow:row branch updated "${default}" "$(flow:tip "${new}")" "${old:0:7}" '' "$(flow:on-branch "${base}")" '' ''
          fi ;;
        (moved:"refs/remotes/${remote}/"*)
          change=updated
          git merge-base --is-ancestor "${old}" "${new}" 2>/dev/null || change=forced
          # a shallow fetch may lack the old commit, which leaves how many commits unknown
          commits=$(git rev-list --count "${old}..${new}" -- 2>/dev/null || true)
          flow:row branch "${change}" "${ref#"refs/remotes/${remote}/"}" "$(flow:tip "${new}")" "${old:0:7}" "${commits}" \
            "$(flow:on-branch "${ref}")" '' '' ;;
        (new:"refs/remotes/${remote}/"*)
          commits=''
          # a shallow fetch may lack the base's history, which leaves how many commits unknown
          git show-ref --verify --quiet "${base}" && commits=$(git rev-list --count "${base}..${new}" -- 2>/dev/null || true)
          flow:row branch new "${ref#"refs/remotes/${remote}/"}" "$(flow:tip "${new}")" '' "${commits}" "$(flow:on-branch "${ref}")" '' '' ;;
        (gone:"refs/remotes/${remote}/"*)
          flow:row branch gone "${ref#"refs/remotes/${remote}/"}" "$(flow:tip "${old}")" "${old:0:7}" '' "$(flow:on-branch "${ref}")" \
            '' '' ;;
        (new:refs/tags/* | moved:refs/tags/*)
          change=new
          [[ "${kind}" == new ]] || change=moved
          flow:row tag "${change}" "${ref#refs/tags/}" "$(git for-each-ref --count=1 \
            --format="%(if)%(*objectname)%(then)%(*objectname:short)%(else)%(objectname:short)%(end)${sep}%(taggername)%(authorname)${sep}%(creatordate:unix)${sep}%(contents:subject)" \
            "${ref}")" '' '' '' '' '' '' ;;
      esac
    done <<<"${diff}"
    for line in "${heads[@]}"
    do
      IFS="${sep}" read -r name upstream <<<"${line}"
      [[ "${upstream}" == "refs/remotes/${remote}/"* ]] || continue
      git show-ref --verify --quiet "${upstream}" || continue
      counts=$(git rev-list --left-right --count "refs/heads/${name}...${upstream}" -- 2>/dev/null) || continue
      read -r ahead behind <<<"${counts}"
      (( behind > 0 )) || continue
      flow:row tracking behind "${upstream#"refs/remotes/${remote}/"}" "$(flow:tip "${upstream}")" '' "${behind}" \
        "${name}" "${trees[${name}]:-}" "${ahead}" "${behind}"
    done
  } | jq --raw-input --compact-output --argjson remote "${remote_json}" --arg primary "${primary}" \
    --arg default "${default}" --arg url "${url}" --argjson now "$(date +%s)" '
      def some: if . == "" or . == null then null else . end;
      def count: if . == "" or . == null then null else tonumber end;
      def list: if . == null then [] else split("\u001e") | map(select(. != "")) end;
      split("\u001f") as $f
      | {kind: $f[0], change: $f[1], ref: ($f[2] | some), sha: ($f[3] | some), old: ($f[7] | some),
         commits: ($f[8] | count), author: ($f[4] | some), age: ($f[5] | count | if . == null then null else
         ([$now - ., 0] | max) end), subject: ($f[6] | some), local: ($f[9] | list), worktrees: ($f[10] | list),
         ahead: ($f[11] | count), behind: ($f[12] | count)}
      | .issues = ([(.ref // "" | capture("^(?<n>[0-9]+)-").n | "#" + .)?] + [.subject // "" | scan("#[0-9]+")]
                   | unique)
      | {key: ([$remote.name, .kind, (if .kind == "commit" then .sha elif .kind == "remote" then null else .ref end)]
               | map(select(. != null)) | join(" ")),
         remote: $remote.name, primary: (if $remote.name == $primary then "primary" else null end),
         forge: ($remote.forge | some),
         host: ($remote.host | some), url: ($url | some), default: ($default | some)} + .' \
    | jq --slurp --compact-output 'sort_by({remote: 0, commit: 1, branch: 2, tag: 3, tracking: 4}[.kind]) | .[]'
}

# flow:fetched-line <remote-json> prints, to stderr, a fetched remote's line: ❌ and git's line when the fetch
# failed; 🛬 unchanged; or 🛬 fetched and what came, counted: new commits on the default branch, then branches
# updated, new, and pruned, then new tags.
flow:fetched-line() {
  local error default commits updated added pruned tags line part
  local -a parts=()
  IFS=$'\x1f' read -r error default commits updated added pruned tags < <(jq --raw-output \
    '[(.error // ""), (.default // ""), (.commits | length), (.branches.updated | length), (.branches.new | length),
      (.branches.gone | length), (.tags | length)] | map(tostring) | join("\u001f")' <<<"$1")
  if [[ -n "${error}" ]]
  then
    printf '%s %s\n' "${flow_glyphs[Failed]}" "${error}" >&2
    return 0
  fi
  (( commits > 0 )) && parts+=("$(flow:plural "${commits}" 'new commit' 'new commits') on ${default}")
  (( updated > 0 )) && parts+=("$(flow:plural "${updated}" 'branch updated' 'branches updated')")
  (( added > 0 )) && parts+=("$(flow:plural "${added}" 'new branch' 'new branches')")
  (( pruned > 0 )) && parts+=("$(flow:plural "${pruned}" 'branch pruned' 'branches pruned')")
  (( tags > 0 )) && parts+=("$(flow:plural "${tags}" 'new tag' 'new tags')")
  if (( ${#parts[@]} == 0 ))
  then
    printf '%s unchanged\n' "${flow_glyphs[Fetched]}" >&2
    return 0
  fi
  line=${parts[0]}
  for part in "${parts[@]:1}"
  do
    line+="${flow_sep}${part}"
  done
  printf '%s fetched %s\n' "${flow_glyphs[Fetched]}" "${line}" >&2
}

# flow:incoming-text <incoming-json> prints flow:incoming's object to stderr: a block per remote, the primary
# first, a blank line between blocks. A block's first line is 📡, the remote's name, its forge or no forge, and its
# host, or its URL when it has none. Under it is the fetch's line, as flow:fetched-line draws it, or 🛬 not fetched,
# dim. With no remote it says so in one dim line.
flow:incoming-text() {
  local fetched remote name forge host first=true
  fetched=$(jq '.fetched' <<<"$1")
  while IFS= read -r remote
  do
    [[ -n "${remote}" ]] || continue
    IFS=$'\x1f' read -r name forge host < <(jq --raw-output '[.name, (.forge // "no forge"), (.host // "")]
      | join("\u001f")' <<<"${remote}")
    # A remote with no URL git can give is drawn with none.
    [[ -n "${host}" ]] || host=$(git remote get-url --end-of-options "${name}" 2>/dev/null || true)
    if [[ "${first}" == true ]]
    then first=false
    else printf '\n' >&2
    fi
    printf '%s %s%s\n' "${flow_glyphs[Remote]}" "${name}" "$(flow:paint 90 "${flow_sep}${forge}${host:+${flow_sep}${host}}")" >&2
    if [[ "${fetched}" == true ]]
    then flow:fetched-line "${remote}"
    else flow:none "${flow_glyphs[Fetched]} not fetched"
    fi
  done < <(jq --compact-output '.remotes[]' <<<"$1")
  [[ "${first}" == false ]] || flow:none 'no remote'
}

# flow:incoming [--no-fetch] [--json] is the incoming stage. It fetches every remote, the primary first, each with
# git fetch --prune, and never rebases, merges, or pulls. It prints the fetch's blocks, as flow:incoming-text draws
# them. With --json it prints one object instead: fetched, true; primary, its name; remotes, one flow:fetch-json
# object each; and rows, every remote's flow:incoming-rows, the primary's first, each saying where it is from, git, as
# workspaces:placed says. A remote whose fetch failed carries error, git's first line, and nothing came from it. With
# --no-fetch it fetches nothing: fetched is false, each remote is flow:remote-json's object, and the rows are each
# remote's own and its tracking rows, as the last fetch left them. It returns 1 when a fetch failed, after every
# remote is tried.
flow:incoming() {
  local json=false no_fetch=false failed=false arg primary primary_default='' remote before after said doc text
  local default
  local -a remotes=() docs=() rows=()
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
    [[ -n "${primary_default}" ]] || primary_default=$(git:default-branch) || return 1
    if [[ "${remote}" == "${primary}" ]]
    then default=${primary_default}
    else default=$(flow:remote-default "${remote}" "${primary_default}")
    fi
    if [[ "${no_fetch}" == true ]]
    then
      docs+=("$(flow:remote-json "${remote}")")
      rows+=("$(flow:incoming-rows "${docs[-1]}" "${primary}" "${default}" 'not fetched')")
      continue
    fi
    before=$(flow:refs "${remote}")
    if said=$(git fetch --quiet --prune --end-of-options "${remote}" 2>&1 >/dev/null)
    then
      # A remote's HEAD can name its default branch only after the fetch, as for a remote fetched the first time.
      [[ "${remote}" == "${primary}" ]] || default=$(flow:remote-default "${remote}" "${primary_default}")
      after=$(flow:refs "${remote}")
      docs+=("$(flow:fetch-json "${remote}" "${before}" "${after}" "${default}")")
      rows+=("$(flow:incoming-rows "${docs[-1]}" "${primary}" "${default}" fetched \
        "$(flow:fetch-diff "${before}" "${after}")")")
    else
      failed=true
      said=$(grep --max-count=1 . <<<"${said}" || true)  # grep's exit 1: git said nothing, named below
      docs+=("$(flow:fetch-json "${remote}" '' '' '' \
        | jq --compact-output --arg error "${said:-git fetch failed}" '. + {error: $error}')")
      rows+=("$(flow:incoming-rows "${docs[-1]}" "${primary}" "${default}" failed)")
    fi
  done
  doc=$(printf '%s\n' "${docs[@]}" | jq --slurp --compact-output --argjson fetched "$([[ "${no_fetch}" == true ]] \
    && printf false || printf true)" --arg primary "${primary}" \
    --argjson rows "$(printf '%s\n' "${rows[@]}" | jq --slurp --compact-output '.')" \
    '{fetched: $fetched, primary: $primary, remotes: ., rows: $rows}')
  if [[ "${json}" == true ]]
  then
    workspaces:placed git "${doc}" || return 1
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
# the caller declares; an --owner's logins may come joined by :, --owner=a:b for --owner=a --owner=b.
flow:owner-flags() {
  local arg
  local -a joined
  mine=false others=false owners=()
  for arg in "$@"
  do
    case "${arg}" in
      (--mine) mine=true ;;
      (--others) others=true ;;
      (--owner=?*)
        IFS=: read -ra joined <<<"${arg#--owner=}"
        owners+=("${joined[@]}")
        ;;
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

# flow:filter <mine> <others> <login>… prints, on one line, the arguments flow:select and issues:documents take to
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
    (SUCCESS) text="${flow_glyphs[Passed]} ${total} / ${total}" ;;
    (FAILURE) text="${flow_glyphs[Failed]} ${failing} failing ${passed} / ${total}" ;;
    (PENDING) text="${flow_glyphs[Running]} ${passed} / ${total}" ;;
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

# flow:issue-row <issue-json> <widths> <state> <indent> prints one issue as the changes stage draws
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
    [[ -n "${assignees}" ]] && glyph="${flow_glyphs[Owner]} "
    row+="${flow_gap}${glyph}$(flow:paint 90 "$(flow:cell "${assignees}" "${owner_w}")")"
    width=$(( width - 5 - owner_w ))
  fi
  text=$(flow:age "${age}")
  if [[ -n "${state}" ]]
  then
    row+="${flow_gap}${flow_glyphs[InFlight]} $(flow:paint 34 "${state}")"
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
    then row+="${flow_gap}$(flow:paint 90 "${flow_glyphs[Branch]} $(flow:cell "${branch}" "${branch_w}")")"
    else row+="${flow_gap}   $(flow:cell '' "${branch_w}")"
    fi
    width=$(( width - 5 - branch_w ))
  fi
  (( width < 20 )) && width=20
  (( ${#title} > width )) && title="${title:0:width-1}…"
  printf '%s%s%s %s\n' "${row}" "${flow_gap}" "${flow_glyphs[Issue]}" "${title}" >&2
}

# flow:pr-block <pr-json> <title-width> <owner-width> <state-width> <age-width> <with-branch> <issue-widths>
# prints a pull request: its row, then the issues it closes, one row each, in flight, indented under it. The state
# and its age share a cell. The row shows the author before the state when <owner-width> is above zero, and ends
# with the branch when <with-branch> is not empty. <issue-widths> is flow:issue-widths' line over every issue in
# the table.
flow:pr-block() {
  local doc=$1 title_w=$2 owner_w=$3 state_w=$4 age_w=$5 worktree=$6 issue_widths=$7
  local ref title author branch state age unresolved row issue checks
  IFS=$'\x1f' read -r ref title author branch state age unresolved \
    < <(jq --raw-output '[(.ref // "#\(.number)"), .title, .author, .branch, .state, .age, .unresolved]
          | map(tostring)
          | join("\u001f")' <<<"${doc}")
  row="${flow_glyphs[Change]} $(flow:paint "${flow_change_sgr}" "${ref}") $(flow:cell "${title}" $(( title_w - ${#ref} - 1 )))"
  (( owner_w > 0 )) && row+="${flow_gap}${flow_glyphs[Owner]} $(flow:paint 90 "$(flow:cell "${author}" "${owner_w}")")"
  row+="${flow_gap}$(flow:state-emoji "${state}")"
  row+=" $(flow:paint "$(flow:state-sgr "${state}")" "$(flow:cell "${state}" "${state_w}")")"
  row+=" $(flow:paint 90 "$(flow:cell "$(flow:age "${age}")" "${age_w}" right)")"
  checks=$(flow:check-cell "$(jq --compact-output .checks <<<"${doc}")")
  [[ -n "${checks}" ]] && row+="${flow_gap}${checks}"
  (( unresolved > 0 )) && row+="${flow_gap}$(flow:paint 33 "${flow_glyphs[Threads]} ${unresolved} unresolved")"
  [[ -n "${worktree}" ]] && row+="${flow_gap}$(flow:paint 90 "${flow_glyphs[Branch]} ${branch}")"
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

# flow:in-worktree succeeds in a linked worktree, whose git directory is not the repository's common one.
flow:in-worktree() {
  [[ "$(git rev-parse --path-format=absolute --git-dir)" \
    != "$(git rev-parse --path-format=absolute --git-common-dir)" ]]
}

# flow:stage <stage> [<argument>…] prints one stage, without its heading, with the arguments the stage takes.
flow:stage() {
  local stage=$1
  shift
  case "${stage}" in
    (issues) issues:render "$@" ;;
    (incoming) flow:incoming "$@" ;;
    (changes) changes:render "$@" ;;
    (builds) builds:render "$@" ;;
    (releases) releases:render "$@" ;;
  esac
}

# flow:kept <stage> <argument>… prints the path that keeps the stage's last JSON, as this checkout printed it with these
# arguments: under kit/flow/ in the repository's common git directory, which every worktree shares, named for the
# stage, the checkout, and the arguments, since changes draws what the checkout it runs in sees.
flow:kept() {
  local common top
  common=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || return 1
  top=$(git rev-parse --show-toplevel 2>/dev/null) || return 1
  printf '%s/kit/flow/%s-%s.json\n' "${common}" "$1" "$(printf '%s\n' "${top}" "$@" | cksum | cut -d ' ' -f 1)"
}

# flow:stage-json <stage> <cached> <argument>… prints the stage's JSON, with --json among the arguments, and keeps it
# where flow:kept says; with <cached> true, prints what was kept there instead, and reads nothing, or refuses when
# nothing is. A stage that fails prints what it printed, as incoming does for every remote when one fails, and keeps
# what was kept before.
flow:stage-json() {
  local stage=$1 cached=$2 kept out status=0
  shift 2
  kept=$(flow:kept "${stage}" "$@") || kept=''
  if [[ "${cached}" == true ]]
  then
    if [[ -z "${kept}" || ! -f "${kept}" ]]
    then
      error "kit flow ${stage} has printed nothing here yet; run it without --cached"
      return 1
    fi
    cat "${kept}"
    return 0
  fi
  out=$(flow:stage "${stage}" "$@") || status=$?
  if (( status != 0 ))
  then
    [[ -z "${out}" ]] || printf '%s\n' "${out}"
    return "${status}"
  fi
  printf '%s\n' "${out}"
  # A copy that cannot be kept costs only the next --cached, so it never fails the stage.
  [[ -z "${kept}" || -z "${out}" ]] || file:replace "${kept}" printf '%s\n' "${out}" 2>/dev/null || true
}

# flow:stage-args <stage> <argument>… prints, one per line, the arguments among those given that the stage takes:
# --json for every stage, --no-fetch for incoming, and the owner flags for issues and changes.
flow:stage-args() {
  local stage=$1 arg
  shift
  for arg in "$@"
  do
    case "${arg}" in
      (--json) printf '%s\n' "${arg}" ;;
      (--no-fetch) [[ "${stage}" != incoming ]] || printf '%s\n' "${arg}" ;;
      (--all | --mine | --others | --owner=?*)
        list:among "${stage}" issues changes && printf '%s\n' "${arg}"
    esac
  done
  return 0
}

# flow:asks-forge <stage>… succeeds when one of the stages is answered by the forge.
flow:asks-forge() {
  local stage
  for stage in "$@"
  do
    case "${stage}" in
      (issues | changes | builds | releases) return 0
    esac
  done
  return 1
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
# while the remotes are fetched; then each stage is drawn in order, under its heading. A stage that failed is drawn as
# its refusal, and the picture returns 1 at the end, in either form. So does a fetch that failed: incoming draws it on
# its remote's line, and without incoming, the primary is fetched alone and a failure warns once. With no primary
# remote it fetches nothing and asks the forge nothing: it says so once, and returns 0. A primary whose forge kit
# cannot tell is fetched, and drawn with incoming; the forge stages are left out, said once. It returns 1 when a
# setting is wrong.
flow:picture() {
  local json=false no_fetch=false reachable=true forged=true failed=false first=true stage doc code choice scratch text
  local primary host said
  local -a stages=() args=() named=()
  text=$(settings:stages) || return $?
  readarray -t stages <<<"${text}"
  [[ " $* " == *" --json "* ]] && json=true
  [[ " $* " == *" --no-fetch "* ]] && no_fetch=true
  primary=$(forge:primary) || return $?
  git remote get-url --end-of-options "${primary}" >/dev/null 2>&1 || reachable=false
  # A primary whose forge kit cannot tell is still fetched; only the forge stages are left out, said once, when the
  # flow setting names one.
  if [[ "${reachable}" == true ]] && flow:asks-forge "${stages[@]}" && [[ -z "$(forge:of "${primary}")" ]]
  then
    forged=false
    host=$(forge:host "${primary}")
    if [[ -z "${host}" ]]
    then said="${primary} is a local path, so it does not name a forge"
    else said="${primary}'s host '${host}' does not name a forge the kit knows"
    fi
    said+="; kit does not show any pull request; set forge in kit.yml ($(forge:providers))"
  fi
  choice=$(flow:choice-of "$@")
  file:scratch scratch
  # The forge stages need nothing local and the forge is the slow call, so they start first, in the background.
  for stage in "${stages[@]}"
  do
    case "${stage}" in
      (issues | changes | builds | releases)
        [[ "${reachable}" == true && "${forged}" == true ]] || continue
        readarray -t args < <(flow:stage-args "${stage}" --json "$@")
        (
          flow:stage "${stage}" "${args[@]}" && code=0 || code=$?
          printf '%s\n' "${code}" > "${scratch}/${stage}.status"
        ) > "${scratch}/${stage}.json" 2> "${scratch}/${stage}.err" &
    esac
  done
  # The fetch keeps the primary's branches as it has them now, the branches that claim an issue in the backlog.
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
    warn "git fetch --prune failed; the branches on ${primary} may be stale"
    failed=true
  fi
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
    [[ "${forged}" == true ]] || warn "${said}"
    jq --null-input --compact-output --argjson stages "$(flow:json-list "${stages[@]}")" "${named[@]}" \
      '{stages: $stages} + $ARGS.named'
    [[ "${failed}" == false ]]
    return
  fi
  if [[ "${reachable}" == false ]]
  then
    warn "no ${primary}: nothing fetched, no pull request shown"
    first=false
  elif [[ "${forged}" == false ]]
  then
    warn "${said}"
    first=false
  fi
  for stage in "${stages[@]}"
  do
    case "${stage}" in
      (incoming) [[ "${reachable}" == true ]] || continue ;;
      (*) [[ "${reachable}" == true && "${forged}" == true ]] || continue
    esac
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
      (issues)
        flow:heading issues "$(jq 'length' <<<"${doc}")"
        issues:draw "${doc}" "${choice}"
        ;;
      (incoming)
        flow:heading incoming
        flow:incoming-text "${doc}"
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
# here. --fresh makes every stage read the forge afresh, not from the snapshot of the last minute, as forge:fresh
# says.
flow:render() {
  local stage='' arg cached=false
  local -a args=()
  if ! workspaces:in-repository >/dev/null
  then
    flow:above "$@"
    return
  fi
  case "${1:-}" in
    (issues | changes | incoming | builds | releases)
      stage=$1
      shift
  esac
  for arg in "$@"
  do
    case "${arg}" in
      (--text) ;;
      (--fresh)
        use forge
        forge:fresh
        ;;
      (--cached) cached=true ;;
      (*) args+=("${arg}")
    esac
  done
  if [[ "${cached}" == true && ( -z "${stage}" || " ${args[*]} " != *" --json "* ) ]]
  then
    error "--cached prints a stage's JSON: run kit flow <stage> --json --cached"
    return 2
  fi
  if [[ -n "${stage}" && " ${args[*]} " == *" --json "* ]]
  then
    flow:stage-json "${stage}" "${cached}" "${args[@]}"
  elif [[ -n "${stage}" ]]
  then
    flow:stage "${stage}" "${args[@]}"
  else
    flow:picture "${args[@]}"
  fi
}

# flow:above <argument>… draws every desk below the current directory, each under its path, as kit flow draws it
# at that desk, with the same arguments. The desks run at once, each fetching on its own, and are drawn in path
# order. It returns 1, after every desk is drawn, when a desk's kit flow did not exit 0.
flow:above() {
  local json=false arg desk index=0 color='' failed=false code
  local -a desks
  for arg in "$@"
  do
    [[ "${arg}" == --json ]] && json=true
  done
  list:read desks git:desks "$(pwd -P)" "${flow_walk_depth}" || return 1
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
    {
      code=0
      ( cd "${desk}" && COLUMNS=$(flow:width) CLICOLOR_FORCE=${color} run kit flow "$@" ) \
        > "${scratch}/${index}.out" 2> "${scratch}/${index}.err" || code=$?
      printf '%s' "${code}" > "${scratch}/${index}.status"
    } &
    index=$(( index + 1 ))
  done
  wait
  for index in "${!desks[@]}"
  do
    [[ -f "${scratch}/${index}.status" && "$(<"${scratch}/${index}.status")" == 0 ]] || failed=true
  done
  if [[ "${json}" == true ]]
  then
    for index in "${!desks[@]}"
    do
      jq --compact-output --arg path "$(flow:relative "${desks[${index}]}")" '{path: $path, status: .}' \
        "${scratch}/${index}.out" 2>/dev/null || jq --null-input --compact-output \
        --arg path "$(flow:relative "${desks[${index}]}")" '{path: $path, status: null}'
      cat "${scratch}/${index}.err" >&2
    done | jq --slurp . || return 1
    [[ "${failed}" == false ]]
    return
  fi
  for index in "${!desks[@]}"
  do
    (( index == 0 )) || printf '\n' >&2
    printf '%s\n' "$(flow:paint 1 "${flow_glyphs[Path]} $(flow:relative "${desks[${index}]}")")" >&2
    cat "${scratch}/${index}.err" "${scratch}/${index}.out" >&2
  done
  [[ "${failed}" == false ]]
}

# ----------------------------------------------------------------------------------------------------------------------
