# shellcheck shell=bash
# forge.sh - the seam over the forge (GitHub, GitLab) that hosts the repository (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# Verbs that read or change issues, branches, and pull requests call forge:<call> here. Each call is passed to a
# provider, forge/<name>.sh beside it, whose functions are named forge/<name>:<call>.
#
# A repository has one primary remote: the `primary` setting in kit.yml, else origin. Ahead and behind, the default
# branch, pull requests, builds, releases, and issues are all read from it. kit issue and kit submit still fetch from
# and push to origin. Each remote has a code forge, or none. The first answer wins:
#
#   1. for the primary, the `forge` setting in kit.yml
#   2. the remote's entry in the `forges` map in kit.yml: `forges: {upstream: gitlab}`
#   3. the remote's host, from git remote get-url, which applies url.*.insteadOf: github.com or a host under it is
#      github; gitlab.com or a host under it is gitlab
#   4. the forge tools' login files: a host gh has logged in to, in hosts.yml under GH_CONFIG_DIR, is github; a host
#      glab has, in config.yml under GLAB_CONFIG_DIR, is gitlab. Each directory defaults to its tool's own, under
#      XDG_CONFIG_HOME or ~/.config. This is what tells a self-hosted forge apart.
#   5. otherwise the remote has no forge: kit fetches it and counts its branches, and asks no forge about it
#
# The provider is the primary's forge.
#
# kit install, update, check, context, and lint never call the forge, so they work in a repository that has no
# forge.
#
# The open changes and the open issues are kept as a snapshot, so the stages of one load, which run as processes of
# their own and at once, share one read of each and agree on what is open. A snapshot lives under kit/forge/ in the
# repository's common git directory, which every worktree shares, one directory per forge and primary remote URL.
# It is used for a minute after the read that made it began: long enough for a load's stages, and for the ones a
# fetch moves to load again, and short enough that a person reading the picture sees the forge as it is. The first
# process to miss takes the lock, a symbolic link that names its process, and reads; the others wait for its
# snapshot. A lock whose process is gone is taken over. forge:fresh makes a process read afresh, and a call that
# changes an issue or a change drops the snapshot.
#
# Every provider implements the calls below. Each prints its result on stdout. On failure it prints a remedy on
# stderr. A call marked "or nothing" prints nothing when there is no match.
#
#   require                       the provider's own tool is present and authenticated
#   repo                          the repository as owner/name
#   repo-url                      the repository's browser URL
#   noun                          the change's reference sigil and its name, tab-separated: "#\tpull request"
#   pr-url <n>                    the change's browser URL
#   me                            the caller's login
#   issue-create <title>          the number of the issue it creates
#   issue-title <n>               the issue's title
#   issue-state <n>               OPEN or CLOSED
#   issue-assignees <n>           the assignees' logins, space-separated
#   issue-assign <n> <login>      adds the assignee
#   branch-for-issue <n>          the branch an open pull request for issue <n> is on, or nothing
#   link-branch <n> <default>     makes a branch linked to the issue, from <default>, pushed to the forge
#   pr-for-branch <branch>        the number of the open pull request on the branch, or nothing
#   pr-create <branch> <default> <title>   opens a draft pull request and prints its number
#   pr-title <n>                  the pull request's title
#   pr-body <n>                   the pull request's body
#   pr-set-body <n> <body>        replaces it
#   pr-author <n>                 the login of the change's author, as the forge shows it: dependabot[bot]
#   pr-state-for-branch <branch>  MERGED, CLOSED, or OPEN for the latest pull request on the branch, or nothing
#   pr-here                       the number of the pull request this checkout is working on, or nothing
#   pr-closing-issues <n>         one line per issue the pull request closes: "<number> <state> <owner/name>"
#   open-prs                      one JSON object per open pull request, the documents kit flow draws
#   open-issues                   one JSON object per open issue, oldest first, the documents the backlog stage draws
#   releases                      one JSON object per release, newest first, the documents the releases stage draws
#   runs                          one JSON array of the runs not owned by any pull request, newest first, the 20
#                                 newest at most: pushes to the default branch or a tag, scheduled and manual runs;
#                                 each is {id, number, attempt, workflow, branch, tag, event, sha, title, state, age,
#                                 took, url, jobs}, what the builds stage draws; each job is {name, state, took, step}
#
# see: docs/architecture.md

use settings file

# forge_snapshot_seconds is how long a snapshot is used after its read began, and forge_snapshot_wait how long, in
# tenths of a second, a process waits for another's read before it reads on its own. forge_fresh_since, set by
# forge:fresh, is the second a snapshot must have been read in or after; empty, the last forge_snapshot_seconds do.
declare -g forge_snapshot_seconds=60 forge_snapshot_wait=1200 forge_fresh_since=''

# forge:primary prints the name of the primary remote: the primary setting in kit.yml, else origin. A name from a
# setting reaches git as an argument, so it must be safe first. It refuses, and returns 1, as kit cannot go on with a
# wrong setting:
#
#   - a primary that names no remote here; git remote add refuses a name that starts with -, so a remote is never an
#     option
#   - a key of the forges map that is no remote name: one that starts with -, or holds a character git refuses
#
# A forges key may name a remote this desk does not have; it is only ever compared with a remote's name.
forge:primary() {
  local primary remotes name
  primary=$(settings:value primary)
  if [[ -n "${primary}" ]]
  then
    remotes=$(git remote 2>/dev/null || true)
    if ! grep --quiet --line-regexp --fixed-strings -- "${primary}" <<<"${remotes}"
    then
      error "primary '${primary}' in kit.yml is not a remote; the remotes are: $(paste -s -d ' ' - <<<"${remotes}")"
      return 1
    fi
  fi
  while IFS=$'\t' read -r name _
  do
    [[ -n "${name}" ]] || continue
    if [[ "${name}" == -* || ! "${name}" =~ ^[A-Za-z0-9._/-]+$ ]]
    then
      error "'${name}' in forges in kit.yml is not a remote name; name a remote, as git remote lists them"
      return 1
    fi
  done < <(yaml:map "$(settings:file)" forges)
  printf '%s\n' "${primary:-origin}"
}

# forge:remotes prints every remote, one per line: the primary first, when it is there, then the rest in git's order.
forge:remotes() {
  local primary remote
  local -a rest=()
  primary=$(forge:primary) || return $?
  while IFS= read -r remote
  do
    [[ -n "${remote}" ]] || continue
    if [[ "${remote}" == "${primary}" ]]
    then printf '%s\n' "${remote}"
    else rest+=("${remote}")
    fi
  done < <(git remote 2>/dev/null)
  (( ${#rest[@]} == 0 )) || printf '%s\n' "${rest[@]}"
}

# forge:url-host <url> prints the host a git URL names, without user or port, or nothing for a path or a file:// URL:
# a remote on this machine.
forge:url-host() {
  local url=$1 host=''
  if [[ "${url}" =~ ^([a-z][a-z0-9+.-]*)://([^/]*) ]]
  then
    [[ "${BASH_REMATCH[1]}" == file ]] || host=${BASH_REMATCH[2]##*@}
    host=${host%%:*}
  elif [[ "${url}" =~ ^([^@/:]+@)?([^/:]+): ]]
  then
    # scp-like, [user@]host:path, with no slash before the colon
    host=${BASH_REMATCH[2]}
  fi
  [[ -z "${host}" ]] || printf '%s\n' "${host}"
}

# forge:host <remote> prints the host of the remote's URL, as git resolves it, or nothing: a remote on this machine,
# or no such remote.
forge:host() {
  local url
  url=$(git remote get-url --end-of-options "$1" 2>/dev/null) || return 0
  forge:url-host "${url}"
}

# forge:logged-in <file> <expression> prints the hosts a forge tool's login file names, one per line, read by the
# yq <expression>. The file holds tokens, so it is read by yaml:run, which keeps no copy in the cache. A file that is
# missing or unreadable names no host.
forge:logged-in() {
  [[ -f "$1" ]] || return 0
  yaml:run "$1" "$2" 2>/dev/null || true
}

# forge:of <remote> prints the remote's forge, the first answer of the file header's list, or nothing for a remote
# with no forge.
forge:of() {
  local remote=$1 forge='' name value host hosts primary
  primary=$(forge:primary) || return $?
  [[ "${remote}" != "${primary}" ]] || forge=$(settings:value forge)
  if [[ -z "${forge}" ]]
  then
    while IFS=$'\t' read -r name value
    do
      [[ "${name}" == "${remote}" ]] && forge=${value}
    done < <(yaml:map "$(settings:file)" forges)
  fi
  if [[ -z "${forge}" ]]
  then
    host=$(forge:host "${remote}")
    case "${host}" in
      ('') ;;
      (github.com | *.github.com) forge=github ;;
      (gitlab.com | *.gitlab.com) forge=gitlab ;;
      (*)
        hosts=$(forge:logged-in "${GH_CONFIG_DIR:-${XDG_CONFIG_HOME:-${HOME}/.config}/gh}/hosts.yml" 'keys | .[]')
        if grep --quiet --line-regexp --fixed-strings "${host}" <<<"${hosts}"
        then
          forge=github
        else
          hosts=$(forge:logged-in "${GLAB_CONFIG_DIR:-${XDG_CONFIG_HOME:-${HOME}/.config}/glab-cli}/config.yml" \
            '.hosts | keys | .[]')
          grep --quiet --line-regexp --fixed-strings "${host}" <<<"${hosts}" && forge=gitlab
        fi
    esac
  fi
  [[ -z "${forge}" ]] || printf '%s\n' "${forge}"
}

# forge:provider prints the provider's name: the primary's forge, chosen as the file header describes. It fails with
# a remedy when nothing names a provider, or names one the kit does not have.
forge:provider() {
  local primary provider host
  primary=$(forge:primary) || return $?
  provider=$(forge:of "${primary}") || return $?
  if [[ -z "${provider}" ]]
  then
    host=$(forge:host "${primary}")
    if ! git remote get-url --end-of-options "${primary}" >/dev/null 2>&1
    then
      error "this repository does not have a remote named ${primary}, so kit cannot tell its forge;" \
        "run: git remote add ${primary} <url>, or set forge in kit.yml ($(forge:providers))"
    elif [[ -z "${host}" ]]
    then
      error "${primary} is a local path, so it does not name a forge; set forge in kit.yml ($(forge:providers))"
    else
      error "set forge in kit.yml; ${primary}'s host '${host}' names no forge the kit knows ($(forge:providers))"
    fi
    return 1
  fi
  if [[ -z "$(use:modules "forge/${provider}")" ]]
  then
    error "'${provider}' is not a forge the kit knows; the forges are: $(forge:providers) (forge in kit.yml)"
    return 1
  fi
  printf '%s\n' "${provider}"
}

# forge:providers prints the providers on the module path, space-separated.
forge:providers() {
  use:modules 'forge/*' | sed 's|^forge/||' | paste -s -d ' ' -
}

# forge:call <call> <argument>… runs the provider's function for <call>, loading the provider once. The provider's
# tool runs with color forced off: gh and glab honor CLICOLOR_FORCE and GH_FORCE_TTY, and color even the JSON kit
# reads, so a desk that sets them in its shell reaches the tool as one that does not.
forge:call() {
  local call=$1 provider
  shift
  provider=$(forge:provider) || return 1
  use "forge/${provider}"
  CLICOLOR_FORCE=0 GH_FORCE_TTY='' "forge/${provider}:${call}" "$@"
}

forge:require()             { forge:call require "$@"; }
forge:repo()                { forge:call repo "$@"; }
forge:repo-url()            { forge:call repo-url "$@"; }
forge:me()                  { forge:call me "$@"; }
forge:noun()                { forge:call noun "$@"; }
forge:pr-url()              { forge:call pr-url "$@"; }
forge:issue-create()        { forge:changing issue-create "$@"; }
forge:issue-title()         { forge:call issue-title "$@"; }
forge:issue-state()         { forge:call issue-state "$@"; }
forge:issue-assignees()     { forge:call issue-assignees "$@"; }
forge:issue-assign()        { forge:changing issue-assign "$@"; }
forge:branch-for-issue()    { forge:call branch-for-issue "$@"; }
forge:link-branch()         { forge:changing link-branch "$@"; }
forge:pr-for-branch()       { forge:call pr-for-branch "$@"; }
forge:pr-create()           { forge:changing pr-create "$@"; }
forge:pr-title()            { forge:call pr-title "$@"; }
forge:pr-body()             { forge:call pr-body "$@"; }
forge:pr-set-body()         { forge:changing pr-set-body "$@"; }
forge:pr-author()           { forge:call pr-author "$@"; }
forge:pr-state-for-branch() { forge:call pr-state-for-branch "$@"; }
forge:pr-here()             { forge:call pr-here "$@"; }
forge:pr-closing-issues()   { forge:call pr-closing-issues "$@"; }
forge:open-prs()            { forge:snapshot open-prs; }
forge:open-issues()         { forge:snapshot open-issues; }
forge:releases()            { forge:call releases "$@"; }
forge:runs()                { use git; forge:call runs "$@"; }

# forge:changing <call> <argument>… runs a call that changes the forge, as forge:call does, then drops the
# snapshots, whether it succeeded or not, so the next read sees the change.
forge:changing() {
  local status=0
  forge:call "$@" || status=$?
  forge:forget
  return "${status}"
}

# forge:fresh makes every snapshot read before this second too old for this process, and for the stages it starts,
# so each reads the forge afresh; processes that start with it at once still share one read.
forge:fresh() {
  forge_fresh_since=$(date +%s)
}

# forge:snapshots prints the directory that keeps this repository's snapshots, named for the forge and the primary's
# URL, or fails outside a repository, or for a primary with no URL.
forge:snapshots() {
  local common provider primary url
  common=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || return 1
  provider=$(forge:provider 2>/dev/null) || return 1
  primary=$(forge:primary 2>/dev/null) || return 1
  url=$(git remote get-url --end-of-options "${primary}" 2>/dev/null) || return 1
  printf '%s/kit/forge/%s-%s\n' "${common}" "${provider}" "$(printf '%s' "${url}" | cksum | cut -d ' ' -f 1)"
}

# forge:forget drops this repository's snapshots.
forge:forget() {
  local dir
  dir=$(forge:snapshots) || return 0
  rm -f "${dir}"/*.json
}

# forge:snapshot <call> prints the call's answer from the snapshot when one is recent enough, and otherwise reads it,
# as the file header says, and keeps the answer as the new snapshot. Its first line is the second the read began.
# Where there is no place for a snapshot, it reads. A read that fails keeps nothing, and the next process reads again.
forge:snapshot() {
  local call=$1 dir file lock floor holder waited=0
  if ! dir=$(forge:snapshots) || ! mkdir -p "${dir}"
  then
    forge:call "${call}"
    return
  fi
  file=${dir}/${call}.json lock=${dir}/${call}.lock
  floor=${forge_fresh_since:-$(( $(date +%s) - forge_snapshot_seconds ))}
  while ! forge:snapshot-read "${file}" "${floor}"
  do
    # ln -s fails when the link is there, so one process alone makes it, and the link names that process as it is
    # made. The process that made it checks the snapshot once more, as the one before may have just written it.
    if ln -s "${BASHPID}" "${lock}" 2>/dev/null
    then
      file:at-exit forge:unlock "${lock}" "${BASHPID}"
      forge:snapshot-read "${file}" "${floor}" || forge:snapshot-take "${call}" "${file}" || {
        forge:unlock "${lock}" "${BASHPID}"
        return 1
      }
      forge:unlock "${lock}" "${BASHPID}"
      return 0
    fi
    # A lock whose process is gone is taken over. Every other turn waits, so a lock that cannot be made or removed
    # ends in a read of the process's own.
    if holder=$(readlink "${lock}" 2>/dev/null) \
      && { [[ ! "${holder}" =~ ^[0-9]+$ ]] || ! kill -0 "${holder}" 2>/dev/null; }
    then
      forge:unlock "${lock}" "${holder}"
      [[ -L "${lock}" ]] || continue
    fi
    if (( waited >= forge_snapshot_wait ))
    then
      forge:call "${call}"
      return
    fi
    sleep 0.1
    waited=$(( waited + 1 ))
  done
}

# forge:snapshot-read <file> <floor> prints the snapshot's answer when it was read in the second <floor> or after,
# and fails otherwise, or when there is none. A snapshot is replaced by a rename, so the one open is read whole.
forge:snapshot-read() {
  local taken
  [[ -f "$1" ]] || return 1
  {
    IFS= read -r taken && [[ "${taken}" =~ ^[0-9]+$ ]] && (( taken >= $2 )) || return 1
    cat
  } < "$1"
}

# forge:snapshot-take <call> <file> reads the call's answer from the forge, prints it, and keeps it in <file>, under
# the second the read began.
forge:snapshot-take() {
  local taken answer
  taken=$(date +%s)
  answer=$(forge:call "$1") || return 1
  file:replace "$2" printf '%s\n' "${taken}" ${answer:+"${answer}"} || true
  [[ -z "${answer}" ]] || printf '%s\n' "${answer}"
}

# forge:unlock <lock> <process> removes the lock when it still names the process, so a process never removes a lock
# another has taken since.
forge:unlock() {
  [[ "$(readlink "$1" 2>/dev/null)" != "$2" ]] || rm -f "$1"
}

# forge:tags prints the repository's tags as one JSON array of names, for the providers' runs: a run on a tag is a
# push whose branch names one.
forge:tags() {
  git for-each-ref --format='%(refname:lstrip=2)' refs/tags | jq --raw-input --slurp --compact-output \
    'split("\n") | map(select(. != ""))'
}

# ----------------------------------------------------------------------------------------------------------------------
