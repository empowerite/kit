# shellcheck shell=bash
# forge.sh - the loop's seam over the forge that hosts the repository (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The loop speaks of an issue, a branch linked to it, a pull request, its checks and review threads and linked
# issues. Every verb that needs one asks here, by these names, and a provider answers: forge-<name>.sh beside this
# file, its functions named forge-<name>:<call>. The provider is the `forge` setting in .kit/settings.yml, else
# the one origin's host names. The core verbs never come here, so a repository on no forge still installs,
# updates, checks, and runs kit context and kit lint.
#
# A provider answers these calls, each printing its result on stdout and failing with a remedy on stderr:
#
#   require                       the provider's own tool is present and authenticated
#   repo                          the repository as owner/name
#   repo-url                      the repository's browser URL
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
#   pr-body <n>                   the pull request's body
#   pr-set-body <n> <body>        replaces it
#   pr-state-for-branch <branch>  MERGED, CLOSED, or OPEN for the latest pull request on the branch, or nothing
#   pr-here                       the number of the pull request this checkout is working on, or nothing
#   pr-closing-issues <n>         one line per issue the pull request closes: "<number> <state> <owner/name>"
#   open-prs                      one JSON object per open pull request, the documents kit status draws

use release

# forge:provider prints the provider's name: the forge setting, else github for an origin at github.com. It
# fails, naming the remedy, where neither names a provider the kit carries.
forge:provider() {
  local provider host=''
  provider=$(release:setting forge)
  if [[ -z "${provider}" ]]
  then
    host=$(git remote get-url origin 2>/dev/null | sed -E 's#^([a-z]+://)?([^@/]+@)?([^/:]+).*#\3#')
    case "${host}" in
      (github.com | *.github.com) provider=github ;;
    esac
  fi
  if [[ -z "${provider}" ]]
  then
    error "set forge in .kit/settings.yml; origin's host '${host:-none}' names no forge the kit knows ($(forge:providers))"
    return 1
  fi
  # shellcheck disable=SC2154  # kit_bin_dir is declared by bin/1st.sh, which loaded this module
  if [[ ! -f "${kit_bin_dir}/forge-${provider}.sh" ]]
  then
    error "'${provider}' is not a forge the kit knows; the forges are: $(forge:providers) (forge in .kit/settings.yml)"
    return 1
  fi
  printf '%s\n' "${provider}"
}

# forge:providers prints the providers this release carries, space-separated.
forge:providers() {
  (cd "${kit_bin_dir}" && find . -maxdepth 1 -name 'forge-*.sh' | sed 's@^\./forge-@@; s@\.sh$@@' | LC_ALL=C sort \
    | paste -s -d ' ' -)
}

# forge:call <call> <argument>… runs the provider's function for <call>, loading the provider once.
forge:call() {
  local call=$1 provider
  shift
  provider=$(forge:provider) || return 1
  use "forge-${provider}"
  "forge-${provider}:${call}" "$@"
}

forge:require()             { forge:call require "$@"; }
forge:repo()                { forge:call repo "$@"; }
forge:repo-url()            { forge:call repo-url "$@"; }
forge:me()                  { forge:call me "$@"; }
forge:issue-create()        { forge:call issue-create "$@"; }
forge:issue-title()         { forge:call issue-title "$@"; }
forge:issue-state()         { forge:call issue-state "$@"; }
forge:issue-assignees()     { forge:call issue-assignees "$@"; }
forge:issue-assign()        { forge:call issue-assign "$@"; }
forge:branch-for-issue()    { forge:call branch-for-issue "$@"; }
forge:link-branch()         { forge:call link-branch "$@"; }
forge:pr-for-branch()       { forge:call pr-for-branch "$@"; }
forge:pr-create()           { forge:call pr-create "$@"; }
forge:pr-body()             { forge:call pr-body "$@"; }
forge:pr-set-body()         { forge:call pr-set-body "$@"; }
forge:pr-state-for-branch() { forge:call pr-state-for-branch "$@"; }
forge:pr-here()             { forge:call pr-here "$@"; }
forge:pr-closing-issues()   { forge:call pr-closing-issues "$@"; }
forge:open-prs()            { forge:call open-prs "$@"; }

# ----------------------------------------------------------------------------------------------------------------------
