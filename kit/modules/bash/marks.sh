# shellcheck shell=bash
# marks.sh - a mark per stage of kit flow, which changes when what the stage draws does
# ----------------------------------------------------------------------------------------------------------------------
# A picture that has drawn a stage reads its mark again, cheaply, and knows the stage is stale once the mark differs
# from the one it read when the stage loaded. Each stage's mark is read from what it draws:
#
#   - backlog and underway: the forge's issues mark, of the issue or change updated last
#   - changes: the issues mark and the runs mark, as a change's checks are its runs
#   - builds: the runs mark
#   - releases: the releases mark
#   - desk: the root checkout's status, its branch and what it tracks among them, and the remote-tracking refs
#   - worktrees: the linked worktrees, each one's status, and the remote-tracking refs
#   - incoming: what every remote has, as git ls-remote lists its branches and tags
#
# The forge's marks are its seam's, as modules/bash/forge.sh says, of the issues, the runs, and the releases alone, so
# kit flow pays for no mark another app draws, and a kind it does not draw cannot fail it. A stage whose mark cannot be
# read has none, null, which never makes it stale. Where the primary has no forge, the forge's stages have none and
# nothing is said; where the forge cannot answer a kind, the stages of that kind have none, the provider's line is
# said, and the marks fail, every other stage's mark still printed. A forge that lacks marks gives its stages none, and
# nothing is said. Reaching a remote asks no one for a password: a remote that would ask has no mark.
#
# see: modules/bash/forge.sh, kit/bin/kit-flow

use forge settings

# marks:stages prints one JSON object, a mark for each stage the flow setting names, as the header says. It returns 1,
# after the object, when the forge's marks cannot be read.
marks:stages() {
  local stage forge='' text
  local -i unread=0
  local -a stages=() args=()
  text=$(settings:stages) || return 1
  readarray -t stages <<<"${text}"
  for stage in "${stages[@]}"
  do
    case "${stage}" in
      (backlog | underway | changes | builds | releases)
        [[ -z "${forge}" ]] || continue
        # A forge without marks has none, which is no failure: its stages are read again only when asked.
        if forge:none || ! forge:serves marks 2>/dev/null
        then
          forge='{}'
        else
          forge=$(forge:marks issues runs releases) || unread=1
          # A kind that could not be read is null in what the forge printed; nothing printed, every kind is.
          jq --exit-status 'type == "object"' <<<"${forge}" >/dev/null 2>&1 || forge='{}'
        fi
    esac
  done
  [[ -n "${forge}" ]] || forge='{}'
  for stage in "${stages[@]}"
  do
    case "${stage}" in
      (desk) args+=(--arg desk "$(marks:desk)") ;;
      (worktrees) args+=(--arg worktrees "$(marks:worktrees)") ;;
      (incoming) args+=(--arg incoming "$(marks:incoming)") ;;
    esac
  done
  jq --null-input --compact-output --argjson forge "${forge}" "${args[@]}" --args '
    def or_null: if . == "" then null else . end;
    def both($a; $b): if $a == null or $b == null then null else $a + " " + $b end;
    ($ARGS.named | del(.forge) | map_values(or_null)) as $local
    | [ $ARGS.positional[]
        | {key: ., value: (
            if . == "backlog" or . == "underway" then $forge.issues
            elif . == "changes" then both($forge.issues; $forge.runs)
            elif . == "builds" then $forge.runs
            elif . == "releases" then $forge.releases
            else $local[.] end)} ]
    | from_entries' "${stages[@]}" || return 1
  return "${unread}"
}

# marks:sum prints a short sum of what it reads, the mark a stage's local state makes.
marks:sum() { cksum | cut -d ' ' -f 1; }

# marks:refs prints the remote-tracking refs, each with the commit it names, as a fetch moves them. A repository git
# cannot read gives none: a mark only tells a picture when to read a stage again.
marks:refs() { git for-each-ref --format='%(objectname) %(refname)' refs/remotes 2>/dev/null || true; }

# marks:status <path> prints a checkout's status as git reports it with its branch: the commit, the branch, what it
# tracks and how far apart they are, and each path changed.
marks:status() {
  # A checkout git cannot read gives nothing: a mark only tells a picture when to read a stage again.
  git -C "$1" status --porcelain=v2 --branch 2>/dev/null || true
}

# marks:desk prints the desk's mark: its status and the remote-tracking refs, summed.
marks:desk() {
  local desk
  desk=$(git worktree list --porcelain 2>/dev/null | sed -n '1s/^worktree //p')
  [[ -n "${desk}" ]] || return 0
  { marks:status "${desk}"; marks:refs; } | marks:sum
}

# marks:worktrees prints the linked worktrees' mark: the list of them, each one's status, and the remote-tracking
# refs, summed.
marks:worktrees() {
  local list path
  list=$(git worktree list --porcelain 2>/dev/null) || return 0
  {
    printf '%s\n' "${list}"
    while IFS= read -r path
    do
      marks:status "${path}"
    done < <(sed -n 's/^worktree //p' <<<"${list}" | tail -n +2)
    marks:refs
  } | marks:sum
}

# marks:incoming prints what every remote has, as git ls-remote lists its branches and tags, summed; nothing when a
# remote cannot be reached without asking for a password, which git and ssh are told never to ask.
marks:incoming() {
  local remote text all=''
  text=$(forge:remotes 2>/dev/null) || return 0
  while IFS= read -r remote
  do
    [[ -n "${remote}" ]] || continue
    all+=$(GIT_TERMINAL_PROMPT=0 GIT_SSH_COMMAND="${GIT_SSH_COMMAND:-ssh -o BatchMode=yes}" \
      git ls-remote --heads --tags --end-of-options "${remote}" 2>/dev/null) || return 0
    all+=$'\n'
  done <<<"${text}"
  printf '%s' "${all}" | marks:sum
}

# ----------------------------------------------------------------------------------------------------------------------
