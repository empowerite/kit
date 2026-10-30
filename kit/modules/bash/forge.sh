# shellcheck shell=bash
# forge.sh - the seam over the forge (GitHub, GitLab) that hosts the repository (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# Verbs that read or change issues, branches, and pull requests call forge:<call> here. Each call is passed to a
# provider, forge-<name>.sh in this directory, whose functions are named forge-<name>:<call>.
#
# The provider is chosen by:
#
#   1. the `forge` setting in .kit/settings.yml, if set
#   2. otherwise origin's host: github.com is github, gitlab.com is gitlab
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
#   link-branch <n> <default>     makes a branch linked to the issue, from <default>, pushed to origin
#   pr-for-branch <branch>        the number of the open pull request on the branch, or nothing
#   pr-create <branch> <default> <title>   opens a draft pull request and prints its number
#   pr-title <n>                  the pull request's title
#   pr-body <n>                   the pull request's body
#   pr-set-body <n> <body>        replaces it
#   pr-author <n>                 the login of the change's author, as the forge shows it: dependabot[bot]
#   pr-state-for-branch <branch>  MERGED, CLOSED, or OPEN for the latest pull request on the branch, or nothing
#   pr-here                       the number of the pull request this checkout is working on, or nothing
#   pr-closing-issues <n>         one line per issue the pull request closes: "<number> <state> <owner/name>"
#   open-prs                      one JSON object per open pull request, the documents kit status draws
#   open-issues                   one JSON object per open issue, oldest first, the documents kit todo draws
#
# see: docs/architecture.md

use release

# forge:provider prints the provider's name, chosen as the file header describes. A self-hosted forge, or a
# repository on two forges, needs the setting. It fails with a remedy when neither the setting nor origin's host
# names a provider the kit has.
forge:provider() {
  local provider host=''
  provider=$(release:setting forge)
  if [[ -z "${provider}" ]]
  then
    host=$(git remote get-url origin 2>/dev/null | sed -E 's#^([a-z]+://)?([^@/]+@)?([^/:]+).*#\3#')
    case "${host}" in
      (github.com | *.github.com) provider=github ;;
      (gitlab.com | *.gitlab.com) provider=gitlab ;;
    esac
  fi
  if [[ -z "${provider}" ]]
  then
    error "set forge in .kit/settings.yml; origin's host '${host:-none}' names no forge the kit knows" \
      "($(forge:providers))"
    return 1
  fi
  if [[ -z "$(use:modules "forge-${provider}")" ]]
  then
    error "'${provider}' is not a forge the kit knows; the forges are: $(forge:providers) (forge in .kit/settings.yml)"
    return 1
  fi
  printf '%s\n' "${provider}"
}

# forge:providers prints the providers on the module path, space-separated.
forge:providers() {
  use:modules 'forge-*' | sed 's/^forge-//' | paste -s -d ' ' -
}

# forge:call <call> <argument>… runs the provider's function for <call>, loading the provider once. The provider's
# tool runs with color forced off: gh and glab honor CLICOLOR_FORCE and GH_FORCE_TTY, and color even the JSON kit
# reads, so a desk that sets them in its shell reaches the tool as one that does not.
forge:call() {
  local call=$1 provider
  shift
  provider=$(forge:provider) || return 1
  use "forge-${provider}"
  CLICOLOR_FORCE=0 GH_FORCE_TTY='' "forge-${provider}:${call}" "$@"
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

# ----------------------------------------------------------------------------------------------------------------------
