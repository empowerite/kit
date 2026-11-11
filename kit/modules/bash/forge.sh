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
#   4. on a CI runner, the server the runner belongs to, when its host is the remote's: the host of GITHUB_SERVER_URL
#      is github, CI_SERVER_HOST gitlab, and the host of SYSTEM_COLLECTIONURI, Azure Pipelines' collection, ado
#   5. the forge tools' login files: a host gh has logged in to, in hosts.yml under GH_CONFIG_DIR, is github; a host
#      glab has, in config.yml under GLAB_CONFIG_DIR, is gitlab. Each directory defaults to its tool's own, under
#      XDG_CONFIG_HOME or ~/.config. This and the runner's server are what tell a self-hosted forge apart.
#   6. otherwise the remote has no forge: kit fetches it and counts its branches, and asks no forge about it
#
# A primary on a host that nothing names is not taken for one with no forge: forge:none fails for it, so a read of
# its change fails, naming forge in kit.yml, and a rule that judges the change never passes it unread. A primary that
# is a path on this machine, or no remote at all, has no host, and no forge.
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
#   - A call of a feature the forge has may still return 3 and print nothing when the forge says this repository has it
#     off: a scanner disabled, a tier without it. The seam passes the 3 on, and the caller says what it means there.
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
#   require                       the provider's own tool is present and authenticated
#   repo                          the repository as owner/name
#   repo-url                      the repository's browser URL
#   noun                          the change's reference sigil and its name, tab-separated: "#\tpull request"
#   pr-url <n>                    the change's browser URL
#   me                            the caller's login
#   marks [<dir>]                 one JSON object, a mark for each kind of the repository's data, issues (with the
#                                 changes), runs, releases, milestones, deployments, environments, and alerts, each a
#                                 text that changes when that data does, read cheaply, or null where the forge refuses
#                                 the caller that kind alone; with <dir>, what makes the next read cheaper is kept
#                                 there
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
#   pr-create <branch> <default> <title> [<draft>]   opens a pull request and prints its number: a draft, unless
#                                 <draft> is false (since version 3), when it opens ready for review
#   pr-ready <n>                  makes the draft ready for review (since 3)
#   pr-title <n>                  the pull request's title
#   pr-set-title <n> <title>      replaces it (since 3)
#   pr-body <n>                   the pull request's body
#   pr-set-body <n> <body>        replaces it
#   pr-label-add <n> <label>      puts the label on the pull request (since 3)
#   pr-label-remove <n> <label>   takes it off (since 3)
#   pr-author <n>                 the login of the change's author, as the forge shows it: dependabot[bot]
#   pr-last-for-branch <branch>   the number of the latest pull request on the branch, in any state, or nothing
#   pr-state <n>                  MERGED, CLOSED, or OPEN, a tab, and the merge commit when it is merged
#   pr-state-for-branch <branch>  MERGED, CLOSED, or OPEN for the latest pull request on the branch, or nothing
#   pr-head-for-branch <branch>   the head commit of the latest pull request on the branch, in any state, or nothing
#   pr-here                       the number of the pull request this checkout is working on, or nothing
#   pr-closing-issues <n>         one line per issue the pull request closes: "<number> <state> <owner/name>"
#   open-prs                      one JSON object per open pull request, the documents kit flow draws, and kit
#                                 changes' reviews and checks: modules/jq/forge.jq's header names their fields
#   open-issues                   one JSON object per open issue, oldest first, the documents the backlog stage draws
#   releases                      one JSON object per release, newest first, the documents the releases stage draws:
#                                 {version, tag, name, kind, latest, at, age, url, notes, assets}; kind is release,
#                                 pre-release, or draft; url its page; notes its text, empty for none; and assets each
#                                 file it offers, {name, url}, where it downloads
#   runs                          one JSON array of the runs not owned by any pull request, newest first, the 20
#                                 newest at most: pushes to the default branch or a tag, scheduled and manual runs;
#                                 each is {id, number, attempt, workflow, branch, tag, event, sha, title, state, age,
#                                 took, url, jobs}, what the builds stage draws; each job is {name, state, took, step}
#   milestones                    one JSON object per milestone: {number, title, state, due, open, closed, url}; state
#                                 is open or closed, due seconds since the epoch or null (since 4)
#   iterations                    one JSON object per iteration, by start: {id, title, start, end, url}, start and end
#                                 seconds since the epoch (since 4); a repository with none prints none, and a forge
#                                 whose tier has none answers 3
#   merged-prs                    one JSON object per merged change, the 50 merged last, merged last first: {number,
#                                 title, author, branch, merged, by, url}, merged seconds since the epoch, by who
#                                 merged it (since 4)
#   artifacts                     one JSON object per artifact of a run that can still be downloaded, newest first,
#                                 the newest 100 at most: {id, name, digest, size, run, age, expires, url}; id unique
#                                 among them, as <run>/<id> where the forge numbers them within a run; digest
#                                 sha256:… or null, size in bytes,
#                                 run the run's id, age seconds since it was made, expires seconds since the epoch or
#                                 null, url its page (since 4)
#   environments                  one JSON object per deployment environment: {name, protection, approvers, branches,
#                                 url}; protection the kinds of rule that guard it, sorted: reviewers, wait, branches,
#                                 deployers, or the forge's own; approvers by login; branches the refs that may deploy
#                                 there, a tag's as refs/tags/…, null for any; protection and approvers null where the
#                                 forge does not show them (since 4)
#   deployments                   one JSON object per deployment, newest first, the newest 50 at most: {id, environment,
#                                 ref, sha, state, actor, age, url, commit}; state success, failure, running, waiting,
#                                 cancelled, or inactive; url its log or page, commit the commit's page (since 4)
#   collaborators                 one JSON object per person with access: {login, name, role, kind}; name null where
#                                 the forge shows none, role the forge's own name for it, kind user, bot, or group, for
#                                 a group a team holds (since 4); a forge that shows them only to who may push fails
#                                 for any other, naming who can give that access
#   teams                         one JSON object per team, in the forge's order: {team, name, role, members, url},
#                                 team unique among them, role the access it has, or null, members their logins, or
#                                 null for a team the caller cannot see; 20 teams at most, 100 members each, then
#                                 {"limited": N} after the N drawn where there are more of either (since 4)
#   bots                          one JSON object per bot that acted, the last to act first: {login, acted, last, url},
#                                 acted a list of commit, change, and review, null where the forge does not say, last
#                                 seconds since the epoch, or null, url its page, or null where the forge gives none
#                                 (since 4)
#   attestations                  one JSON object per attestation of the 5 newest releases' assets and the 20 newest
#                                 artifacts, by digest, 50 digests at most, then {"limited": N} where there are more
#                                 of any: {subject, digest, kind, signer, age, url}; subject the asset's or artifact's
#                                 name, digest sha256:…, kind the statement's predicate type, signer the workflow that
#                                 signed it, each null where the forge does not say, age seconds since it was made,
#                                 url its page (since 4)
#   code-scanning-alerts          one JSON object per code scanning alert, open or dismissed: {number, rule, severity,
#                                 state, path, age, url}; rule what found it, severity in lower case or null, state
#                                 open or dismissed, path where it was found or null, age seconds since it was made;
#                                 a forge that says the scanner is off for this repository answers 3 (since 4)
#   dependabot-alerts             the same, one per Dependabot alert, a dependency scanner's (since 4)
#   secret-scanning-alerts        the same, one per secret scanning alert (since 4)
#   rulesets                      one JSON object per control on the default branch: {id, control, target, enforcement,
#                                 rules, source, url}; id unique among them, target the refs it names, enforcement
#                                 active, evaluate, or disabled, rules its kinds of rule, sorted, or null where the
#                                 forge does not show them, source ruleset or protection (since 4)
#   bypasses                      one JSON object per waiver: {waives, actor, kind, reason, by, until, url}; waives the
#                                 id of a control rulesets prints, or an alert, "<feature> <number>"; actor who may
#                                 bypass, or null for a dismissal or where the forge hides it; kind bypass or
#                                 dismissal; reason the bypass's mode, or the dismissal's reason and comment; by who
#                                 dismissed it; until when it ends, or null (since 4)
#
# A call that lists is capped when the call says so above: it prints {"limited": N} as its last line when it stopped
# at N items, and the N before it are the newest.
#
# The contract has a version, forge_contract. A provider declares the version it implements and the features its forge
# lacks, before any call is made:
#
#   forge/<name>:contract         the version it implements; a provider without one implements version 1
#   forge/<name>:lacks            the features its forge does not have, one per line; without one, it lacks none
#
# A call marked "since" above is unsupported for a provider that implements an older version, as the seam answers it.
# A feature is a set of calls, or of fields of a document, that a forge has whole or not at all. Each feature names
# the contract version that added it, and a provider that implements an older version lacks it:
#
#   feature          since  calls, or the fields kit reads
#   issues           1      issue-create, issue-title, issue-state, issue-assignees, issue-assign, issue-unassign,
#                           issue-close, issue-comment, open-issues: the forge's own tracker
#   sub-issues       1      issue-add-sub
#   linked-branches  1      link-branch: a branch the forge links to an issue
#   drafts           1      pr-create opens a draft, pr-ready ends it; without it, a change open for review
#   closing-issues   1      pr-closing-issues: the issues the forge says a change closes
#   review-threads   1      open-prs: each change's threads and whether they are resolved
#   checks           1      open-prs: each change's checks
#   quota            1      quota
#   marks            1      marks
#   releases         1      releases
#   runs             1      runs
#   labels           3      pr-label-add, pr-label-remove
#   milestones       4      milestones
#   iterations       4      iterations
#   reviewers        4      open-prs: each change's reviewers: [{login, state, at}], state requested, approved,
#                           changes, or commented
#   merged           4      merged-prs
#   artifacts        4      artifacts
#   environments     4      environments
#   deployments      4      deployments
#   collaborators    4      collaborators
#   teams            4      teams
#   bots             4      bots
#   attestations     4      attestations
#   code-scanning    4      code-scanning-alerts
#   dependabot       4      dependabot-alerts
#   secret-scanning  4      secret-scanning-alerts
#   rulesets         4      rulesets
#   bypasses         4      bypasses
#
# A repository chooses a stand-in for a feature in kit.yml, under the forge it is for, and the seam then answers the
# feature's calls itself, from the forge's other calls, whether the forge has the feature or not:
#
#   forge_settings:
#     ado:
#       stand_ins:
#         drafts: {label: draft}       # a change is a draft while it has the label; or {title_prefix: "[draft] "},
#                                      # while its title starts with the prefix, which pr-title leaves off
#         closing-issues: body         # the forge's own issues the change's body closes: Closes #7
#         marks: {every: 60}           # a mark that changes every 60 seconds, 10 at the least
#
# A stand-in that needs what the forge lacks is refused: a label on a forge without labels, a title prefix on a
# provider older than version 3, which can neither open a change ready nor set its title.
#
# forge:capabilities prints the declaration as one JSON object, which kit flow --capabilities gives kittui: {contract,
# forge, provider, features, stand_ins}, provider being the version the provider implements, features each feature's
# state, native, none, or stand-in, and stand_ins the repository's choices. A primary with no forge has every feature
# none.
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
declare -gr forge_contract=4
declare -gra forge_features=(issues:1 sub-issues:1 linked-branches:1 drafts:1 closing-issues:1 review-threads:1
  checks:1 quota:1 marks:1 releases:1 runs:1 labels:3
  milestones:4 iterations:4 reviewers:4 merged:4 artifacts:4 environments:4 deployments:4 collaborators:4 teams:4
  bots:4 attestations:4 code-scanning:4 dependabot:4 secret-scanning:4 rulesets:4 bypasses:4)
declare -grA forge_feature_of=(
  [issue-create]=issues [issue-title]=issues [issue-state]=issues [issue-assignees]=issues [issue-assign]=issues
  [issue-unassign]=issues [issue-close]=issues [issue-comment]=issues [open-issues]=issues
  [issue-add-sub]=sub-issues [link-branch]=linked-branches [pr-closing-issues]=closing-issues
  [quota]=quota [marks]=marks [releases]=releases [runs]=runs [pr-ready]=drafts
  [pr-label-add]=labels [pr-label-remove]=labels
  [milestones]=milestones [iterations]=iterations [merged-prs]=merged [artifacts]=artifacts
  [environments]=environments [deployments]=deployments [collaborators]=collaborators [teams]=teams [bots]=bots
  [attestations]=attestations [code-scanning-alerts]=code-scanning [dependabot-alerts]=dependabot
  [secret-scanning-alerts]=secret-scanning [rulesets]=rulesets [bypasses]=bypasses
)
# forge_call_since names the version that added each call added after the first; forge_provider_contract holds, once
# forge:lacks has read it, the version the provider implements. forge_stand_ins holds the stand-ins, as one JSON object,
# once forge:stand-ins has read them.
declare -grA forge_call_since=([pr-ready]=3 [pr-set-title]=3 [pr-label-add]=3 [pr-label-remove]=3)
declare -g forge_lacking='' forge_lacking_read=false forge_provider_contract=1 forge_stand_ins=''

# forge:primary prints the name of the primary remote: the primary setting in kit.yml, else origin. A name from a
# setting reaches git as an argument, so it must be safe first. It refuses, and returns 1, as kit cannot go on with a
# wrong setting:
#
#   - a primary that names no remote here; git remote add refuses a name that starts with -, so a remote is never an
#     option
#   - a primary set where git cannot list the remotes, as outside a repository: it cannot be checked
#   - a key of the forges map that is no remote name: one that starts with -, or holds a character git refuses
#
# A forges key may name a remote this desk does not have; it is only ever compared with a remote's name.
forge:primary() {
  local primary remotes name row
  local -a rows
  primary=$(settings:value primary)
  if [[ -n "${primary}" ]]
  then
    if ! remotes=$(git remote 2>/dev/null)
    then
      error "run: git remote (the remotes could not be read, so primary '${primary}' in kit.yml cannot be checked)"
      return 1
    fi
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

# forge:runner-of <host> prints the forge of the CI server whose host is <host>, as the file header's list says, or
# nothing off a runner or for another host.
forge:runner-of() {
  local host=$1
  if [[ -n "${GITHUB_SERVER_URL:-}" && "$(forge:url-host "${GITHUB_SERVER_URL}")" == "${host}" ]]
  then
    printf 'github\n'
  elif [[ -n "${CI_SERVER_HOST:-}" && "${CI_SERVER_HOST}" == "${host}" ]]
  then
    printf 'gitlab\n'
  elif [[ -n "${SYSTEM_COLLECTIONURI:-}" && "$(forge:url-host "${SYSTEM_COLLECTIONURI}")" == "${host}" ]]
  then
    printf 'ado\n'
  fi
}

# forge:login-of <host> prints the forge whose tool has logged in to <host>, as the file header's list says, or
# nothing.
forge:login-of() {
  local host=$1 hosts
  hosts=$(forge:logged-in "${GH_CONFIG_DIR:-${XDG_CONFIG_HOME:-${HOME}/.config}/gh}/hosts.yml" 'keys | .[]')
  if grep --quiet --line-regexp --fixed-strings "${host}" <<<"${hosts}"
  then
    printf 'github\n'
    return 0
  fi
  hosts=$(forge:logged-in "${GLAB_CONFIG_DIR:-${XDG_CONFIG_HOME:-${HOME}/.config}/glab-cli}/config.yml" \
    '.hosts | keys | .[]')
  if grep --quiet --line-regexp --fixed-strings "${host}" <<<"${hosts}"
  then
    printf 'gitlab\n'
  fi
}

# forge:named <remote> prints the forge kit.yml sets for the remote: forge for the primary, else its entry under
# forges; none where kit.yml says the remote has no forge; nothing where kit.yml sets none. It fails when kit.yml
# cannot be read.
forge:named() {
  local remote=$1 forge='' name value primary row
  local -a rows
  primary=$(forge:primary) || return $?
  if [[ "${remote}" == "${primary}" ]]
  then
    forge=$(settings:value forge) || return 1
  fi
  if [[ -z "${forge}" ]]
  then
    list:read rows yaml:map "$(settings:file)" forges || return 1
    for row in "${rows[@]}"
    do
      IFS=$'\t' read -r name value <<<"${row}"
      [[ "${name}" == "${remote}" ]] && forge=${value}
    done
  fi
  [[ -z "${forge}" ]] || printf '%s\n' "${forge}"
}

# forge:of <remote> prints the remote's forge, the first answer of the file header's list, or nothing for a remote
# with no forge, as one kit.yml names none.
forge:of() {
  local remote=$1 forge host
  forge=$(forge:named "${remote}") || return $?
  [[ "${forge}" != none ]] || return 0
  if [[ -z "${forge}" ]]
  then
    host=$(forge:host "${remote}")
    case "${host}" in
      ('') ;;
      (github.com | *.github.com) forge=github ;;
      (gitlab.com | *.gitlab.com) forge=gitlab ;;
      (dev.azure.com | ssh.dev.azure.com | *.visualstudio.com) forge=ado ;;
      (*)
        forge=$(forge:runner-of "${host}")
        [[ -n "${forge}" ]] || forge=$(forge:login-of "${host}")
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
    if [[ "$(forge:named "${primary}")" == none ]]
    then
      error "set forge in kit.yml to the primary's forge ($(forge:providers)); kit.yml sets forge: none, so" \
        "there is no forge to ask"
    elif ! git remote get-url --end-of-options "${primary}" >/dev/null 2>&1
    then
      error "this repository does not have a remote named ${primary}, so kit cannot tell its forge;" \
        "run: git remote add ${primary} <url>, or set forge in kit.yml ($(forge:providers), or none)"
    elif [[ -z "${host}" ]]
    then
      error "${primary} is a local path, so it does not name a forge; set forge in kit.yml" \
        "($(forge:providers), or none)"
    else
      error "set forge in kit.yml; ${primary}'s host '${host}' names no forge the kit knows" \
        "($(forge:providers), or none)"
    fi
    return 1
  fi
  if [[ -z "$(use:modules "forge/${provider}")" ]]
  then
    error "'${provider}' is not a forge the kit knows; the forges are: $(forge:providers), or none (forge in kit.yml)"
    return 1
  fi
  printf '%s\n' "${provider}"
}

# forge:none succeeds when the primary has no forge, the last answer of the file header's list, or kit.yml says it has
# none. It fails when the primary has a forge, when the settings that choose it cannot be read, so a wrong setting is
# never taken for none, and when the primary is on a host that nothing names, whose forge the kit cannot tell.
forge:none() {
  local primary forge
  primary=$(forge:primary 2>/dev/null) || return 1
  forge=$(forge:named "${primary}" 2>/dev/null) || return 1
  [[ "${forge}" != none ]] || return 0
  forge=$(forge:of "${primary}" 2>/dev/null) || return 1
  [[ -z "${forge}" ]] || return 1
  [[ -z "$(forge:host "${primary}")" ]]
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
  forge:lacks >/dev/null || return 1
  (( ${forge_call_since[${call}]:-1} <= forge_provider_contract )) || return 3
  if [[ -n "${feature}" ]]
  then
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
      error "the ${provider} forge implements contract '${version}', and this kit knows 1 to ${forge_contract}; move to" \
        "a later kit: kit --version=<version> update"
      return 1
    fi
    if declare -F "forge/${provider}:lacks" >/dev/null
    then
      list:read declared "forge/${provider}:lacks" || return 1
    fi
    forge_provider_contract=${version}
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

# forge:stand-ins prints the stand-ins the repository chose for the primary's forge, as one JSON object of feature to
# choice, {} for none, read once a process. It refuses a stand-in the file header does not list, or one that needs what
# the forge lacks, naming the setting.
forge:stand-ins() {
  local provider doc choices refusal
  if [[ -z "${forge_stand_ins}" ]] && forge:none
  then
    # a primary with no forge has nothing to stand in for
    forge_stand_ins='{}'
  elif [[ -z "${forge_stand_ins}" ]]
  then
    provider=$(forge:provider) || return 1
    forge:lacks >/dev/null || return 1
    doc=$(yaml:json "$(settings:file)" 2>/dev/null) || doc=null
    choices=$(jq --compact-output --arg forge "${provider}" '.forge_settings[$forge].stand_ins // {}' \
      <<<"${doc:-null}" 2>/dev/null) || choices='{}'
    refusal=$(jq --raw-output --arg lacking "${forge_lacking}" --argjson version "${forge_provider_contract}" '
      ($lacking | split("\n")) as $lacks
      | first(to_entries[] | .key as $f | .value as $v
      | if $f == "drafts" then
          if ($v | type) == "object" and ($v | keys) == ["label"] and ($v.label | type) == "string" and $v.label != ""
          then (if IN($lacks[]; "labels") then "a label, which the forge lacks" else empty end)
          elif ($v | type) == "object" and ($v | keys) == ["title_prefix"] and ($v.title_prefix | type) == "string"
            and $v.title_prefix != ""
          then (if $version < 3 then "a title prefix, which needs a provider of version 3" else empty end)
          else "{label: <name>} or {title_prefix: <text>}, not \($v | tojson)" end
        elif $f == "closing-issues" then (if $v == "body" then empty else "body, not \($v | tojson)" end)
        elif $f == "marks" then
          (if ($v | type) == "object" and ($v | keys) == ["every"] and ($v.every | type) == "number" and $v.every >= 10
           then empty else "{every: <seconds>}, 10 at the least, not \($v | tojson)" end)
        else "no stand-in: the stand-ins are drafts, closing-issues, and marks" end
      | "\($f): \(.)") // empty' <<<"${choices}" 2>/dev/null) \
      || refusal='the stand-ins are no map'
    if [[ -n "${refusal}" ]]
    then
      error "forge_settings.${provider}.stand_ins in kit.yml: ${refusal}"
      return 1
    fi
    forge_stand_ins=${choices}
  fi
  printf '%s\n' "${forge_stand_ins}"
}

# forge:stand-in <feature> prints the repository's stand-in for the feature as JSON, or nothing when it chose none. It
# fails as forge:stand-ins does.
forge:stand-in() {
  forge:stand-ins >/dev/null || return 1
  jq --compact-output --arg f "$1" '.[$f] // empty' <<<"${forge_stand_ins}"
}

# forge:serves <feature> succeeds when the primary's forge answers the feature, natively or through a stand-in.
forge:serves() {
  local choice
  choice=$(forge:stand-in "$1") || return 1
  [[ -n "${choice}" ]] || forge:has "$1"
}

# forge:capabilities prints the declaration of the primary's forge as one JSON object, as the file header says.
forge:capabilities() {
  local provider='' version=0 lacking entry stand_ins='{}'
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
    version=${forge_provider_contract}
    stand_ins=$(forge:stand-ins) || return 1
  fi
  jq --null-input --compact-output --argjson contract "${forge_contract}" --arg forge "${provider}" \
    --argjson provider "${version}" --arg lacking "${lacking}" --argjson stand_ins "${stand_ins}" '
    ($lacking | split("\n") | map(select(. != ""))) as $lacks
    | {contract: $contract, forge: (if $forge == "" then null else $forge end), provider: $provider,
       features: ($ARGS.positional | map({key: ., value: (if $stand_ins[.] != null then "stand-in"
                                                          elif IN($lacks[]) then "none" else "native" end)})
         | from_entries),
       stand_ins: $stand_ins}' --args "${names[@]}"
}

forge:require()             { forge:call require "$@"; }
forge:repo()                { forge:call repo "$@"; }
forge:repo-url()            { forge:call repo-url "$@"; }
forge:me()                  { forge:call me "$@"; }
forge:quota()               { forge:call quota "$@"; }

# forge:marks prints the forge's marks, as the seam's header says, keeping what makes the next read cheaper beside the
# snapshots, or nowhere when they have no place. With the marks stand-in, each mark is the span of its seconds that now
# falls in, so a stage is stale once that span is past.
forge:marks() {
  local dir='' choice span
  forge:stand-ins >/dev/null || return 1
  choice=$(forge:stand-in marks) || return 1
  if [[ -n "${choice}" ]]
  then
    span=$(( $(date +%s) / $(jq .every <<<"${choice}") ))
    jq --null-input --compact-output --arg span "${span}" \
      '{issues: $span, runs: $span, releases: $span, milestones: $span, deployments: $span, environments: $span,
        alerts: $span}'
    return 0
  fi
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
# forge:pr-create <branch> <default> <title> opens the change as a draft, the forge's own or the stand-in the repository
# chose: a change opened ready, then labelled, or titled with the prefix. It prints the change's number.
forge:pr-create() {
  local choice number
  forge:stand-ins >/dev/null || return 1
  choice=$(forge:stand-in drafts) || return 1
  case "${choice}" in
    ('') forge:changing pr-create "$@" ;;
    (*'"label"'*)
      number=$(forge:changing pr-create "$1" "$2" "$3" false) || return $?
      forge:changing pr-label-add "${number}" "$(jq --raw-output .label <<<"${choice}")" \
        || warn "${number} could not be labelled a draft; label it by hand"
      printf '%s\n' "${number}"
      ;;
    (*) forge:changing pr-create "$1" "$2" "$(jq --raw-output .title_prefix <<<"${choice}")$3" false ;;
  esac
}

# forge:pr-ready <n> makes the draft ready for review: the forge's own draft, or the stand-in's label or prefix taken
# off. A forge without drafts, and no stand-in, answers unsupported.
forge:pr-ready() {
  local choice title prefix
  forge:stand-ins >/dev/null || return 1
  choice=$(forge:stand-in drafts) || return 1
  case "${choice}" in
    ('') forge:changing pr-ready "$@" ;;
    (*'"label"'*) forge:changing pr-label-remove "$1" "$(jq --raw-output .label <<<"${choice}")" ;;
    (*)
      prefix=$(jq --raw-output .title_prefix <<<"${choice}")
      title=$(forge:call pr-title "$1") || return $?
      [[ "${title}" == "${prefix}"* ]] || return 0
      forge:changing pr-set-title "$1" "${title#"${prefix}"}"
      ;;
  esac
}

# forge:pr-title <n> prints the change's title, without the prefix a title-prefix stand-in for drafts puts on it, so a
# policy judges the title a person wrote.
forge:pr-title() {
  local choice title prefix
  forge:stand-ins >/dev/null || return 1
  choice=$(forge:stand-in drafts) || return 1
  title=$(forge:call pr-title "$@") || return $?
  if [[ "${choice}" == *'"title_prefix"'* ]]
  then
    prefix=$(jq --raw-output .title_prefix <<<"${choice}")
    title=${title#"${prefix}"}
  fi
  printf '%s\n' "${title}"
}
forge:pr-set-title()        { forge:changing pr-set-title "$@"; }
forge:pr-label-add()        { forge:changing pr-label-add "$@"; }
forge:pr-label-remove()     { forge:changing pr-label-remove "$@"; }
forge:pr-body()             { forge:call pr-body "$@"; }
forge:pr-set-body()         { forge:changing pr-set-body "$@"; }
forge:pr-author()           { forge:call pr-author "$@"; }
forge:pr-last-for-branch()  { forge:call pr-last-for-branch "$@"; }
forge:pr-state()            { forge:call pr-state "$@"; }
forge:pr-state-for-branch() { forge:call pr-state-for-branch "$@"; }
forge:pr-head-for-branch()  { forge:call pr-head-for-branch "$@"; }
forge:pr-here()             { forge:call pr-here "$@"; }
# forge:pr-closing-issues <n> prints the issues the change closes, as the forge says, or, with the closing-issues stand-in,
# as its body says: each issue of the forge's own a closing keyword names, with its state, as the repository's.
forge:pr-closing-issues() {
  local choice body repo numbers number state
  forge:stand-ins >/dev/null || return 1
  choice=$(forge:stand-in closing-issues) || return 1
  if [[ -z "${choice}" ]]
  then
    forge:call pr-closing-issues "$@"
    return
  fi
  body=$(forge:pr-body "$1") || return 1
  repo=$(forge:call repo) || return 1
  numbers=$(grep --only-matching --ignore-case --extended-regexp '(close[sd]?|fix(e[sd])?|resolve[sd]?):? #[0-9]+' \
    <<<"${body}" | grep --only-matching --extended-regexp '[0-9]+$' | LC_ALL=C sort --unique --numeric-sort) \
    || numbers=''  # grep's exit 1 is a body that names no issue of the forge's own
  for number in ${numbers}
  do
    state=$(forge:issue-state "${number}") || return 1
    printf '%s %s %s\n' "${number}" "${state}" "${repo}"
  done
}

# forge:open-prs prints the open changes, as the forge's snapshot keeps them, each drawn as a draft while the drafts
# stand-in says it is one: it has the label, or its title the prefix, which the document then leaves off.
forge:open-prs() {
  local choice docs
  forge:stand-ins >/dev/null || return 1
  choice=$(forge:stand-in drafts) || return 1
  if [[ -z "${choice}" ]]
  then
    forge:snapshot open-prs
    return
  fi
  docs=$(forge:snapshot open-prs) || return $?
  [[ -z "${docs}" ]] || jq --compact-output --argjson choice "${choice}" '
    if $choice.label != null and any(.labels[]?; .name == $choice.label) then .state = "draft"
    elif $choice.title_prefix != null and (.title | startswith($choice.title_prefix))
    then .state = "draft" | .title |= ltrimstr($choice.title_prefix)
    else . end' <<<"${docs}"
}
forge:open-issues()         { forge:snapshot open-issues; }
forge:releases()            { forge:call releases "$@"; }
forge:runs()                { use git; forge:call runs "$@"; }
forge:milestones() { forge:call milestones "$@"; }
forge:iterations() { forge:call iterations "$@"; }
forge:merged-prs() { forge:call merged-prs "$@"; }
forge:artifacts() { forge:call artifacts "$@"; }
forge:environments() { forge:call environments "$@"; }
forge:deployments() { forge:call deployments "$@"; }
forge:collaborators() { forge:call collaborators "$@"; }
forge:teams() { forge:call teams "$@"; }
forge:bots() { forge:call bots "$@"; }
forge:attestations() { forge:call attestations "$@"; }
forge:code-scanning-alerts() { forge:call code-scanning-alerts "$@"; }
forge:dependabot-alerts() { forge:call dependabot-alerts "$@"; }
forge:secret-scanning-alerts() { forge:call secret-scanning-alerts "$@"; }
forge:rulesets() { forge:call rulesets "$@"; }
forge:bypasses() { forge:call bypasses "$@"; }

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
