# shellcheck shell=bash
# forge.sh - the seam over the forge (GitHub, GitLab) that hosts the repository (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# Verbs that read or change issues, branches, and pull requests call forge:<call> here. Each call is passed to a
# provider, forge/<name>.sh beside it, whose functions are named forge/<name>:<call>.
#
# A repository has one primary remote: the `primary` setting in kit.yml, else origin. Ahead and behind, the default
# branch, pull requests, builds, releases, and issues are all read from it. Each remote has a forge, or none. The
# first answer wins:
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

use settings

# forge:primary prints the name of the primary remote: the primary setting in kit.yml, else origin. A name from a
# setting reaches git as an argument, so it must be safe first. It refuses, and returns 2, a settings error:
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
      return 2
    fi
  fi
  while IFS=$'\t' read -r name _
  do
    [[ -n "${name}" ]] || continue
    if [[ "${name}" == -* || ! "${name}" =~ ^[A-Za-z0-9._/-]+$ ]]
    then
      error "'${name}' in forges in kit.yml is not a remote name; name a remote, as git remote lists them"
      return 2
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
    error "set forge in kit.yml; ${primary}'s host '${host:-none}' names no forge the kit knows" \
      "($(forge:providers))"
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
forge:issue-create()        { forge:call issue-create "$@"; }
forge:issue-title()         { forge:call issue-title "$@"; }
forge:issue-state()         { forge:call issue-state "$@"; }
forge:issue-assignees()     { forge:call issue-assignees "$@"; }
forge:issue-assign()        { forge:call issue-assign "$@"; }
forge:branch-for-issue()    { forge:call branch-for-issue "$@"; }
forge:link-branch()         { forge:call link-branch "$@"; }
forge:pr-for-branch()       { forge:call pr-for-branch "$@"; }
forge:pr-create()           { forge:call pr-create "$@"; }
forge:pr-title()            { forge:call pr-title "$@"; }
forge:pr-body()             { forge:call pr-body "$@"; }
forge:pr-set-body()         { forge:call pr-set-body "$@"; }
forge:pr-author()           { forge:call pr-author "$@"; }
forge:pr-state-for-branch() { forge:call pr-state-for-branch "$@"; }
forge:pr-here()             { forge:call pr-here "$@"; }
forge:pr-closing-issues()   { forge:call pr-closing-issues "$@"; }
forge:open-prs()            { forge:call open-prs "$@"; }
forge:open-issues()         { forge:call open-issues "$@"; }
forge:releases()            { forge:call releases "$@"; }
forge:runs()                { use git; forge:call runs "$@"; }

# forge:tags prints the repository's tags as one JSON array of names, for the providers' runs: a run on a tag is a
# push whose branch names one.
forge:tags() {
  git for-each-ref --format='%(refname:lstrip=2)' refs/tags | jq --raw-input --slurp --compact-output \
    'split("\n") | map(select(. != ""))'
}

# ----------------------------------------------------------------------------------------------------------------------
