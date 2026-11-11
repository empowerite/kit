# shellcheck shell=bash
# forge.sh - the seam over the forge (GitHub, GitLab) that hosts the repository (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# Verbs that read or change issues, branches, and pull requests call forge:<call> here. Each call is passed to a
# provider, forge/<name>.sh beside it, whose functions are named forge/<name>:<call>.
#
# A repository has one primary remote: the `primary` setting in kit.yml, else origin. Ahead and behind, the default
# branch, pull requests, builds, releases, and issues are all read from it. kit work and kit submit still fetch from
# and push to origin. Each remote has a code forge, or none. The first answer wins:
#
#   1. for the primary, the `forge` setting in kit.yml
#   2. the remote's entry in the `forges` map in kit.yml: `forges: {upstream: gitlab}`
#   3. the remote's host, from git remote get-url, which applies url.*.insteadOf: github.com or a host under it is
#      github; gitlab.com or a host under it is gitlab; dev.azure.com, ssh.dev.azure.com, or a host under
#      visualstudio.com is ado
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
# Every provider implements the calls below, and each call holds to one contract: it answers, answers none, fails, or
# is unsupported.
#
#   - A call returns 0 and prints its answer on stdout.
#   - A call marked "or nothing" returns 0 and prints nothing only when the forge answered that there is none.
#   - A call of a feature the forge lacks returns 3 and prints nothing, on stdout or stderr. The seam answers it so,
#     from the provider's declaration below, without running the provider: the caller says what it means there.
#   - Otherwise the call returns 1, prints nothing on stdout, and says one ❌ line on stderr, the remedy and then what
#     could not be read: "run: gh auth status (the pull requests could not be read)". An answer that is not the JSON
#     asked for is a read that failed.
#   - A list is whole or the call fails: the call reads every page, and a list that reaches forge_list_limit items
#     fails, saying "kit reads at most 10000".
#
# A call that changes the forge returns 0 when the change is made, 1 when it is not, and 3 when the forge lacks it.
#
# The contract has a version, forge_contract. A provider declares the version it implements and the features its forge
# lacks, before any call is made:
#
#   forge/<name>:contract         the version it implements; a provider without one implements version 1
#   forge/<name>:lacks            the features its forge does not have, one per line; without one, it lacks none
#
# A feature is a set of calls, or of fields of a document, that a forge has whole or not at all. Each feature names
# the contract version that added it, and a provider that implements an older version lacks it:
#
#   feature          since  calls, or the fields kit reads
#   issues           1      issue-create, issue-title, issue-state, issue-assignees, issue-assign, issue-unassign,
#                           issue-close, issue-comment, open-issues: the forge's own tracker
#   sub-issues       1      issue-add-sub
#   linked-branches  1      link-branch: a branch the forge links to an issue
#   drafts           1      pr-create opens a draft; without it, a change open for review
#   closing-issues   1      pr-closing-issues: the issues the forge says a change closes
#   review-threads   1      open-prs: each change's threads and whether they are resolved
#   checks           1      open-prs: each change's checks
#   quota            1      quota
#   marks            1      marks
#   releases         1      releases
#   runs             1      runs
#
# forge:capabilities prints the declaration as one JSON object, which kit flow --capabilities gives kittui: {contract,
# forge, provider, features}, provider being the version the provider implements, features each feature's state,
# native or none. A primary with no forge has every feature none.
#
#   require                       the provider's own tool is present and authenticated
#   repo                          the repository as owner/name
#   repo-url                      the repository's browser URL
#   noun                          the change's reference sigil and its name, tab-separated: "#\tpull request"
#   pr-url <n>                    the change's browser URL
#   me                            the caller's login
#   marks [<dir>]                 one JSON object, a mark for each kind of the repository's data, issues (with the
#                                 changes), runs, and releases, each a text that changes when that data does, read
#                                 cheaply; with <dir>, what makes the next read cheaper is kept there
#   quota                         one JSON object, the caller's rate limits on the forge: {forge, limits}, each limit
#                                 {name, limit, remaining, reset}, reset in seconds since the epoch; reading it costs
#                                 at most one request
#   issue-create <title> [<body>] the number of the issue it creates
#   issue-title <n>               the issue's title
#   issue-state <n>               OPEN or CLOSED
#   issue-assignees <n>           the assignees' logins, space-separated
#   issue-assign <n> <login>      adds the assignee
#   issue-unassign <n> <login>    removes the assignee
#   issue-close <n> <reason> [<m>]  closes the issue: completed, not-planned, or duplicate of issue <m>
#   issue-comment <n> <text>      adds a comment
#   issue-add-sub <n> <m>         puts issue <m> under issue <n>: a sub-issue on GitHub, a linked item on GitLab
#   branch-for-issue <n>          the branch an open pull request for issue <n> is on, or nothing
#   link-branch <n> <default>     makes a branch linked to the issue, from <default>, pushed to the forge
#   pr-for-branch <branch>        the number of the open pull request on the branch, or nothing
#   pr-create <branch> <default> <title>   opens a draft pull request and prints its number
#   pr-title <n>                  the pull request's title
#   pr-body <n>                   the pull request's body
#   pr-set-body <n> <body>        replaces it
#   pr-author <n>                 the login of the change's author, as the forge shows it: dependabot[bot]
#   pr-last-for-branch <branch>   the number of the latest pull request on the branch, in any state, or nothing
#   pr-state <n>                  MERGED, CLOSED, or OPEN, a tab, and the merge commit when it is merged
#   pr-state-for-branch <branch>  MERGED, CLOSED, or OPEN for the latest pull request on the branch, or nothing
#   pr-head-for-branch <branch>   the head commit of the latest pull request on the branch, in any state, or nothing
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

# forge:setting <forge> <field> prints a field of the forge's own settings, forge_settings.<forge>.<field> in kit.yml,
# or nothing: forge_settings: {ado: {type: Bug}}.
forge:setting() {
  local doc
  doc=$(yaml:json "$(settings:file)" 2>/dev/null) || return 0
  jq --raw-output --arg forge "$1" --arg field "$2" \
    '.forge_settings[$forge][$field] // empty | select(type == "string" or type == "number")' <<<"${doc:-null}" \
    2>/dev/null || true  # a kit.yml that is no map has no forge settings
}

# forge_snapshot_seconds is how long a snapshot is used after its read began, and forge_snapshot_wait how long, in
# tenths of a second, a process waits for another's read before it reads on its own. forge_fresh_since, set by
# forge:fresh, is the second a snapshot must have been read in or after; empty, the last forge_snapshot_seconds do.
declare -g forge_snapshot_seconds=60 forge_snapshot_wait=1200 forge_fresh_since=''

# forge_list_limit is the most items a provider reads of one list; a list that reaches it fails, as the file header
# says.
declare -g forge_list_limit=10000

# forge_contract is the version of the contract the file header states. forge_features lists each feature with the
# version that added it, in the header's order, and forge_feature_of names the feature each gated call belongs to.
# forge_lacking holds, once read, the features the primary's forge lacks, one per line; forge_lacking_read says it was.
declare -gr forge_contract=2
declare -gra forge_features=(issues:1 sub-issues:1 linked-branches:1 drafts:1 closing-issues:1 review-threads:1
  checks:1 quota:1 marks:1 releases:1 runs:1)
declare -grA forge_feature_of=(
  [issue-create]=issues [issue-title]=issues [issue-state]=issues [issue-assignees]=issues [issue-assign]=issues
  [issue-unassign]=issues [issue-close]=issues [issue-comment]=issues [open-issues]=issues
  [issue-add-sub]=sub-issues [link-branch]=linked-branches [pr-closing-issues]=closing-issues
  [quota]=quota [marks]=marks [releases]=releases [runs]=runs
)
declare -g forge_lacking='' forge_lacking_read=false

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
  local primary remotes name row
  local -a rows
  primary=$(settings:value primary)
  if [[ -n "${primary}" ]]
  then
    # Outside a repository there is no remote, and the primary is refused below as none.
    remotes=$(git remote 2>/dev/null || true)
    if ! grep --quiet --line-regexp --fixed-strings -- "${primary}" <<<"${remotes}"
    then
      error "primary '${primary}' in kit.yml is not a remote; the remotes are: $(paste -s -d ' ' - <<<"${remotes}")"
      return 1
    fi
  fi
  list:read rows yaml:map "$(settings:file)" forges || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r name _ <<<"${row}"
    [[ -n "${name}" ]] || continue
    if [[ "${name}" == -* || ! "${name}" =~ ^[A-Za-z0-9._/-]+$ ]]
    then
      error "'${name}' in forges in kit.yml is not a remote name; name a remote, as git remote lists them"
      return 1
    fi
  done
  printf '%s\n' "${primary:-origin}"
}

# forge:remotes prints every remote, one per line: the primary first, when it is there, then the rest in git's order.
forge:remotes() {
  local primary remote
  local -a remotes rest=()
  primary=$(forge:primary) || return $?
  list:read remotes git remote || return 1
  for remote in "${remotes[@]}"
  do
    [[ -n "${remote}" ]] || continue
    if [[ "${remote}" == "${primary}" ]]
    then printf '%s\n' "${remote}"
    else rest+=("${remote}")
    fi
  done
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
  yaml:run "$1" "$2" 2>/dev/null || true  # an unreadable file names no host, as the comment above says
}

# forge:of <remote> prints the remote's forge, the first answer of the file header's list, or nothing for a remote
# with no forge.
forge:of() {
  local remote=$1 forge='' name value host hosts primary row
  local -a rows
  primary=$(forge:primary) || return $?
  [[ "${remote}" != "${primary}" ]] || forge=$(settings:value forge)
  if [[ -z "${forge}" ]]
  then
    list:read rows yaml:map "$(settings:file)" forges || return 1
    for row in "${rows[@]}"
    do
      IFS=$'\t' read -r name value <<<"${row}"
      [[ "${name}" == "${remote}" ]] && forge=${value}
    done
  fi
  if [[ -z "${forge}" ]]
  then
    host=$(forge:host "${remote}")
    case "${host}" in
      ('') ;;
      (github.com | *.github.com) forge=github ;;
      (gitlab.com | *.gitlab.com) forge=gitlab ;;
      (dev.azure.com | ssh.dev.azure.com | *.visualstudio.com) forge=ado ;;
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

# forge:none succeeds when the primary has no forge, the last answer of the file header's list. It fails when the
# primary has a forge, and when the settings that choose it cannot be read, so a wrong setting is never taken for none.
forge:none() {
  local primary forge
  primary=$(forge:primary 2>/dev/null) || return 1
  forge=$(forge:of "${primary}" 2>/dev/null) || return 1
  [[ -z "${forge}" ]]
}

# forge:providers prints the providers on the module path, space-separated.
forge:providers() {
  use:modules 'forge/*' | sed 's|^forge/||' | paste -s -d ' ' -
}

# forge:call <call> <argument>… runs the provider's function for <call>, loading the provider once. The provider's
# tool runs with color forced off: gh and glab honor CLICOLOR_FORCE and GH_FORCE_TTY, and color even the JSON kit
# reads, so a desk that sets them in its shell reaches the tool as one that does not.
#
# A call of a feature the forge lacks returns 3 and runs nothing, as the file header says.
forge:call() {
  local call=$1 provider feature=${forge_feature_of[$1]:-}
  shift
  provider=$(forge:provider) || return 1
  use "forge/${provider}"
  if [[ -n "${feature}" ]]
  then
    forge:lacks >/dev/null || return 1
    forge:has "${feature}" || return 3
  fi
  CLICOLOR_FORCE=0 GH_FORCE_TTY='' "forge/${provider}:${call}" "$@"
}

# forge:lacks prints the features the primary's forge lacks, one per line, as its provider declares them and as the
# version it implements leaves out, read once a process. It fails as forge:provider does, and refuses a declaration
# that names a version above forge_contract or a feature the contract does not have.
forge:lacks() {
  local provider version entry feature
  local -a declared=()
  if [[ "${forge_lacking_read}" != true ]]
  then
    provider=$(forge:provider) || return 1
    use "forge/${provider}"
    version=1
    if declare -F "forge/${provider}:contract" >/dev/null
    then
      version=$("forge/${provider}:contract")
    fi
    if [[ ! "${version}" =~ ^[1-9][0-9]*$ ]] || (( version > forge_contract ))
    then
      error "the ${provider} forge implements contract '${version}', and this kit knows 1 to ${forge_contract}; run: kit update"
      return 1
    fi
    if declare -F "forge/${provider}:lacks" >/dev/null
    then
      list:read declared "forge/${provider}:lacks" || return 1
    fi
    forge_lacking=''
    for feature in "${declared[@]}"
    do
      [[ -n "${feature}" ]] || continue
      if [[ " ${forge_features[*]%%:*} " != *" ${feature} "* ]]
      then
        error "the ${provider} forge lacks '${feature}', which is no feature of the forge contract"
        return 1
      fi
      forge_lacking+=${feature}$'\n'
    done
    for entry in "${forge_features[@]}"
    do
      (( ${entry##*:} <= version )) || forge_lacking+=${entry%%:*}$'\n'
    done
    forge_lacking_read=true
  fi
  printf '%s' "${forge_lacking}"
}

# forge:has <feature> succeeds when the primary's forge has the feature, and fails when it lacks it, or when its
# declaration cannot be read.
forge:has() {
  forge:lacks >/dev/null || return 1
  [[ $'\n'"${forge_lacking}" != *$'\n'"$1"$'\n'* ]]
}

# forge:capabilities prints the declaration of the primary's forge as one JSON object, as the file header says.
forge:capabilities() {
  local provider='' version=0 lacking entry
  local -a names=()
  for entry in "${forge_features[@]}"
  do
    names+=("${entry%%:*}")
  done
  if forge:none
  then
    lacking=$(printf '%s\n' "${names[@]}")
  else
    provider=$(forge:provider) || return 1
    forge:lacks >/dev/null || return 1
    lacking=${forge_lacking}
    version=1
    declare -F "forge/${provider}:contract" >/dev/null && version=$("forge/${provider}:contract")
  fi
  jq --null-input --compact-output --argjson contract "${forge_contract}" --arg forge "${provider}" \
    --argjson provider "${version}" --arg lacking "${lacking}" '
    ($lacking | split("\n") | map(select(. != ""))) as $lacks
    | {contract: $contract, forge: (if $forge == "" then null else $forge end), provider: $provider,
       features: ($ARGS.positional | map({key: ., value: (if IN($lacks[]) then "none" else "native" end)})
         | from_entries)}' --args "${names[@]}"
}

forge:require()             { forge:call require "$@"; }
forge:repo()                { forge:call repo "$@"; }
forge:repo-url()            { forge:call repo-url "$@"; }
forge:me()                  { forge:call me "$@"; }
forge:quota()               { forge:call quota "$@"; }

# forge:marks prints the forge's marks, as the seam's header says, keeping what makes the next read cheaper beside the
# snapshots, or nowhere when they have no place.
forge:marks() {
  local dir=''
  dir=$(forge:snapshots 2>/dev/null) && mkdir -p "${dir}" 2>/dev/null || dir=''
  forge:call marks "${dir}"
}
forge:noun()                { forge:call noun "$@"; }
forge:pr-url()              { forge:call pr-url "$@"; }
forge:issue-create()        { forge:changing issue-create "$@"; }
forge:issue-title()         { forge:call issue-title "$@"; }
forge:issue-state()         { forge:call issue-state "$@"; }
forge:issue-assignees()     { forge:call issue-assignees "$@"; }
forge:issue-assign()        { forge:changing issue-assign "$@"; }
forge:issue-unassign()      { forge:changing issue-unassign "$@"; }
forge:issue-close()         { forge:changing issue-close "$@"; }
forge:issue-comment()       { forge:changing issue-comment "$@"; }
forge:issue-add-sub()       { forge:changing issue-add-sub "$@"; }
forge:branch-for-issue()    { forge:call branch-for-issue "$@"; }
forge:link-branch()         { forge:changing link-branch "$@"; }
forge:pr-for-branch()       { forge:call pr-for-branch "$@"; }
forge:pr-create()           { forge:changing pr-create "$@"; }
forge:pr-title()            { forge:call pr-title "$@"; }
forge:pr-body()             { forge:call pr-body "$@"; }
forge:pr-set-body()         { forge:changing pr-set-body "$@"; }
forge:pr-author()           { forge:call pr-author "$@"; }
forge:pr-last-for-branch()  { forge:call pr-last-for-branch "$@"; }
forge:pr-state()            { forge:call pr-state "$@"; }
forge:pr-state-for-branch() { forge:call pr-state-for-branch "$@"; }
forge:pr-head-for-branch()  { forge:call pr-head-for-branch "$@"; }
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
  local call=$1 dir file lock floor holder waited=0 feature=${forge_feature_of[$1]:-}
  # A call the forge lacks is answered so before any snapshot is looked for, as forge:call answers it.
  if [[ -n "${feature}" ]]
  then
    forge:lacks >/dev/null || return 1
    forge:has "${feature}" || return 3
  fi
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
  # A snapshot that cannot be kept costs only the next read, which asks the forge again.
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

# forge:json <text> succeeds when the text is one JSON document or more, as a provider's tool answers.
forge:json() {
  jq --exit-status true <<<"$1" >/dev/null 2>&1
}

# forge:fold <remedy> <what> <answer> <jq argument>… prints what jq, given the kit's jq modules, makes of a provider's
# answer. When jq fails, as on an answer not in the shape asked for, it prints nothing, says the remedy and that <what>
# could not be read, and returns 1.
forge:fold() {
  local remedy=$1 what=$2 answer=$3 out
  shift 3
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  if ! out=$(jq -L "${kit_jq}" "$@" <<<"${answer}" 2>/dev/null)
  then
    error "${remedy} (${what} could not be read)"
    return 1
  fi
  [[ -z "${out}" ]] || printf '%s\n' "${out}"
}

# forge:whole <what> <count> succeeds when a provider's list of <what>, <count> items long, is whole, and otherwise
# says so and fails: a list that reaches forge_list_limit may hold more than kit reads.
forge:whole() {
  [[ "$2" =~ ^[0-9]+$ ]] && (( $2 < forge_list_limit )) && return 0
  error "kit reads at most ${forge_list_limit} items of a list (${1} could not be read)"
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
