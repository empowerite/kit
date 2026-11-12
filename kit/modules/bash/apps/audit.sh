# shellcheck shell=bash
# audit.sh - kit audit's stages: the forge's attestations, alerts, controls, and waivers, the policy, and what binds
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is set by modules/bash/1st.sh
#
# Trust is the app whose tabs are all its own. Four stages ask the forge through the seam of modules/bash/forge.sh,
# policy reads kit's own policy, and enforcement reads both, with the agent packages' files and the git hooks:
#
#   evidence  a row per attestation of the newest releases' assets and artifacts: its kind and who signed it
#   findings  a row per alert of the code scanning, Dependabot, and secret scanning alerts, open or dismissed, each
#             row its scanner's, the tool; a scanner off here, or one that refuses the caller, is its line under
#             errors, and the others still draw
#   controls  a row per ruleset or protection that guards the default branch: its rules, and how it is enforced
#   waivers   a row per bypass of a control, and per alert dismissed, by whom and why: the bypasses the forge says, and
#             the dismissed among the alerts findings reads, each scanner read once a process for both; a scanner
#             off here, or one that refuses the caller, is under errors as in findings, and the rest still draw
#   policy    a row per rule kit policy judges by in each checkout with a branch, as kit policy --rules --json lists
#             them there, placed at its source, organization, repository, and branch, the rows that agree merged
#   enforcement  a row per mechanism that holds an agent to the rules, per harness and for the repository, in its
#             tier: advisory, best-effort, local floor, or authoritative, each with its status and evidence
#
# evidence's mark holds the forge's marks of its runs and its releases, as a run or a release makes an attestation;
# findings', its runs' and its alerts', as a scan is a run and a person dismisses an alert without one. controls and
# waivers have none, and are read again on Refresh only. policy's is each checkout's hash of kit.yml and
# .kit/policy, and on a branch past the default, of the base's, which judge it. enforcement's is the hash of the files
# its own rows read; its forge's rows, read as controls and waivers are, move with none of them.
#
# see: bin/kit-audit, modules/bash/apps.sh, modules/bash/forge.sh, docs/apps.md
use apps forge policy settings file workspaces git region resolve

# audit_stages are kit audit's stages, in Trust's tab order.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
audit_stages=(evidence findings controls waivers policy enforcement)

# apps/audit:evidence prints the attestations document, through apps::attestation, limited when the forge holds more
# than it read.
apps/audit:evidence() { apps:forge attestations attestations attestation; }

# apps_audit_reads is a directory this process keeps each scanner's document in, once read, so the findings and the
# waivers of one picture share one read of each; the stages run in subshells of the shell that loads this module.
apps_audit_reads=''
file:scratch apps_audit_reads || apps_audit_reads=''

# apps_audit_scanners are the three alerts calls: each one's feature and call, the feature being the rows' tool.
apps_audit_scanners=('code-scanning code-scanning-alerts' 'dependabot dependabot-alerts'
  'secret-scanning secret-scanning-alerts')

# apps/audit:alerts <feature> <call> prints the document of the scanner's call, its alerts as the call prints them,
# through apps::kept, read once a process. One whose read fails is {feature, error}, its line, under errors, with no
# rows, and it returns 1 after it, as it does each time it is asked.
apps/audit:alerts() {
  local feature=$1 call=$2 kept said doc line status=0
  kept=${apps_audit_reads:+${apps_audit_reads}/${feature}}
  if [[ -n "${kept}" && -f "${kept}.json" && -f "${kept}.status" ]]
  then
    cat "${kept}.json"
    return "$(<"${kept}.status")"
  fi
  file:temp said audit || return 1
  if ! doc=$(apps:forge "${feature}" "${call}" kept 2>"${said}")
  then
    status=1
    line=$(grep -v '^[[:space:]]*$' "${said}" | tail -n 1 || true)  # grep's exit 1: the read said nothing
    doc=$(jq --null-input --compact-output --arg feature "${feature}" --arg error "${line#❌ }" \
      '{rows: [], errors: [{feature: $feature, error: $error}]}') || return 1
  fi
  # A document that cannot be kept costs only another read of it.
  if [[ -n "${kept}" ]] && printf '%s\n' "${doc}" > "${kept}.json"
  then
    printf '%s\n' "${status}" > "${kept}.status" || rm -f "${kept}.json"
  fi
  printf '%s\n' "${doc}"
  return "${status}"
}

# apps/audit:scanned <def> prints one document of the three scanners' alerts, each row through apps::<def>, given the
# scanner's feature, its tool. A scanner off here is {feature, off: true} under errors; one whose read fails is
# {feature, error}, its line, and the others still draw; it returns 1 after the document then.
apps/audit:scanned() {
  local def=$1 each feature call doc unread=0
  local -a docs=()
  for each in "${apps_audit_scanners[@]}"
  do
    read -r feature call <<<"${each}"
    doc=$(apps/audit:alerts "${feature}" "${call}") || unread=1
    doc=$(jq --compact-output -L "${kit_jq}" --arg tool "${feature}" \
      "import \"apps\" as apps; .rows |= map(apps::${def}(\$tool))" <<<"${doc}") || return 1
    docs+=("${doc}")
  done
  apps:merge "${docs[@]}" || return 1
  return "${unread}"
}

# apps/audit:findings prints one document of the three scanners' alerts, each row through apps::alert, which gives it
# its tool, as apps/audit:scanned does.
apps/audit:findings() { apps/audit:scanned alert; }

# apps/audit:controls prints the controls document, through apps::control.
apps/audit:controls() { apps:forge rulesets rulesets control; }

# apps/audit:waivers prints one document of the forge's bypasses, through apps::waiver, and of the dismissed alerts,
# through apps::dismissal, as apps/audit:scanned reads them. A bypasses read that fails is {feature, error}, its line,
# under errors, as a scanner's is, and the rest still draw; it returns 1 after the document then.
apps/audit:waivers() {
  local bypasses dismissed said line unread=0
  file:temp said audit || return 1
  if ! bypasses=$(apps:forge bypasses bypasses waiver 2>"${said}")
  then
    unread=1
    line=$(grep -v '^[[:space:]]*$' "${said}" | tail -n 1 || true)  # grep's exit 1: the read said nothing
    bypasses=$(jq --null-input --compact-output --arg error "${line#❌ }" \
      '{rows: [], errors: [{feature: "bypasses", error: $error}]}') || return 1
  fi
  dismissed=$(apps/audit:scanned dismissal) || unread=1
  [[ -n "${dismissed}" ]] || return 1
  apps:merge "${bypasses}" "${dismissed}" || return 1
  return "${unread}"
}

# apps/audit:checkouts prints each checkout of the clone here that has a branch, as "<path>\t<branch>" a line: the
# one here first, its branch the one kit policy judges, which on a runner is the change's and in a detached checkout
# none; then each other one git lists with a branch, while its directory is there. A detached checkout is no branch's,
# so only the one here is listed without one. It returns 1 when git cannot list the worktrees.
apps/audit:checkouts() {
  local here line path='' branch=''
  local -a lines
  here=$(pwd -P)
  printf '%s\t%s\n' "${here}" "$(policy:branch)"
  list:read lines git worktree list --porcelain || return 1
  # a blank line ends each worktree's record, the last one's too, which list:read drops
  for line in "${lines[@]}" ''
  do
    case "${line}" in
      ('worktree '*) path=${line#worktree } branch='' ;;
      ('branch refs/heads/'*) branch=${line#branch refs/heads/} ;;
      ('')
        [[ -z "${branch}" || ! -d "${path}" || "$(cd "${path}" && pwd -P)" == "${here}" ]] \
          || printf '%s\t%s\n' "${path}" "${branch}"
        path='' branch=''
    esac
  done
}

# apps/audit:where prints where the clone here is, as JSON: {source, organization, repository}: source the primary's
# forge, github, gitlab, or ado, as forge:of names it; and from the member the primary's URL names, the organization
# and the repository as place in modules/jq/forge.jq splits its path, a group and its subgroups on GitLab. A clone with
# no remote on a forge is its main checkout's directory. A place the clone has none of, the organization of one with no
# remote or the source of a remote on no forge, is "", as null is the place a merged row spans.
apps/audit:where() {
  local member common source primary
  primary=$(forge:primary 2>/dev/null) && source=$(forge:of "${primary}" 2>/dev/null) || source=''
  member=$(workspaces:member-of "$(pwd -P)")
  if [[ -n "${member}" ]]
  then
    jq --null-input --compact-output -L "${kit_jq}" --arg source "${source}" --arg path "${member#*/}" '
      import "forge" as forge;
      {source: $source} + forge::place($path)'
    return
  fi
  common=$(git rev-parse --path-format=absolute --git-common-dir) || return 1
  common=${common%/.git}
  jq --null-input --compact-output --arg source "${source}" --arg repository "${common##*/}" \
    '{source: $source, organization: "", repository: $repository}'
}

# apps/audit:policy prints the policy's document: a row per rule kit policy judges by in each checkout of the clone
# that has a branch, as kit policy --rules --json lists them there, each row placed at its source, organization,
# repository, and branch, and the rows that agree merged, as apps::scopes merges them. Another checkout is read as it
# is, nothing placed in it: its branch past the default is judged by the base's policy, which kit policy reads
# wherever it runs. It returns 1 when kit policy fails here, its line said; another checkout that fails is
# {branch, error}, its line, under errors, and the rest still draw.
apps/audit:policy() {
  local where path branch out said line here checkout status=0
  local -a docs=() checkouts
  where=$(apps/audit:where) || return 1
  here=$(pwd -P)
  file:temp said audit || return 1
  list:read checkouts apps/audit:checkouts || return 1
  for checkout in "${checkouts[@]}"
  do
    IFS=$'\t' read -r path branch <<<"${checkout}"
    if [[ "${path}" == "${here}" ]]
    then
      out=$(run kit policy --rules --json) || return 1
    # kit_serve_hydrated, as under kit serve, keeps kit policy from placing this release's .kit/ in the checkout
    elif ! out=$(cd "${path}" && export kit_serve_hydrated=1 && run kit policy --rules --json 2>"${said}")
    then
      status=1
      line=$(grep -v '^[[:space:]]*$' "${said}" | tail -n 1 || true)  # grep's exit 1: kit policy said nothing
      docs+=("$(jq --null-input --compact-output --arg branch "${branch}" --arg error "${line#❌ }" \
        '{rows: [], errors: [{branch: $branch, error: $error}]}')")
      continue
    fi
    docs+=("$(jq --slurp --compact-output -L "${kit_jq}" --argjson where "${where}" --arg branch "${branch}" '
      import "apps" as apps;
      {rows: map(apps::placed($where + {branch: (if $branch == "" then null else $branch end)})), errors: []}' \
      <<<"${out}")")
  done
  out=$(apps:merge "${docs[@]}") || return 1
  apps/audit:merged-policy <<<"${out}" || return 1
  return "${status}"
}

# audit_ci_check is the check kit's CI job reports, the job's name in the github/ci package's kit-ci.yml, which a
# control requires to bind a merge to kit ci.
audit_ci_check=CI

# audit_harnesses are the agent packages the enforcement stage reads, each a harness, in its rows' order: generic for
# any agent that reads AGENTS.md, then each harness kit has a package for.
audit_harnesses=(generic claude copilot cursor gemini)

# apps/audit:enforcement prints the enforcement document: a row per mechanism that holds an agent to the rules, for
# each harness whose agent package is active and for the repository, each in its tier, advisory, best-effort, local
# floor, or authoritative, with its status, active, missing, or unknown, and its evidence, probed where kit read what
# decides it, documented where it rests on what the harness or the forge documents, unverified where it rests on
# neither. A mechanism merely registered, a deny list or a hook, is unknown at best: kit cannot probe that a harness
# runs it. passes names what gets past it, null where kit cannot tell. A forge read that fails leaves its rows
# unknown, its line under errors, {feature, error}, and the rest still draw; it returns 1 after the document then.
apps/audit:enforcement() {
  local harness default rows forge status=0
  local -a packages
  list:read packages settings:packages || return 1
  default=$(git:default-branch 2>/dev/null) || default=main
  rows=$(
    for harness in "${audit_harnesses[@]}"
    do
      list:among "agents/${harness}" "${packages[@]}" || continue
      "apps/audit:enforcement-${harness}" "${default}" || exit 1
    done
    apps/audit:enforcement-floor "${packages[@]}" || exit 1
    apps/audit:enforcement-ci "${packages[@]}" || exit 1
  ) || return 1
  forge=$(apps/audit:enforcement-forge) || status=1
  [[ -n "${forge}" ]] || return 1
  rows=$(jq --slurp --compact-output . <<<"${rows}") || return 1
  jq --null-input --compact-output -L "${kit_jq}" --argjson forge "${forge}" --arg check "${audit_ci_check}" \
    --argjson rows "${rows}" '
    import "apps" as apps;
    {rows: ($rows + [apps::required($forge; $check)]),
     errors: [$forge | (select(.state == "failed") | {feature: "rulesets", error}),
                       (select(.bypasses == null and .bypassed_error != null) | {feature: "bypasses",
                                                                                 error: .bypassed_error})]}' \
    || return 1
  return "${status}"
}

# apps/audit:enforced <harness> <tier> <mechanism> <status> <evidence> <detail> [<pass>…] prints a row of the
# enforcement document, as apps::enforced gives it: passes the words after the detail, or null for the one word -.
apps/audit:enforced() {
  jq --null-input --compact-output -L "${kit_jq}" --arg harness "$1" --arg tier "$2" --arg mechanism "$3" \
    --arg status "$4" --arg evidence "$5" --arg detail "$6" '
    import "apps" as apps;
    {harness: $harness, tier: $tier, mechanism: $mechanism, status: $status, evidence: $evidence, detail: $detail,
     passes: (if $ARGS.positional == ["-"] then null else $ARGS.positional end), control: null} | apps::enforced' \
    --args "${@:7}"
}

# apps/audit:deny-passes <default> prints, one a line, what passes a deny list of the raw git commands: the forms its
# literal patterns do not name, git -C and git merge, and where the default branch is not main, a checkout of it, as
# the list names main.
apps/audit:deny-passes() {
  printf '%s\n' 'git -C <dir> commit' 'git merge'
  [[ "$1" == main ]] || printf '%s\n' "git checkout $1" "git switch $1"
}

# apps/audit:package-file <package> <path> prints the path of a file the agent package places at <path>, in the
# release kit runs from or the source that holds the package.
apps/audit:package-file() {
  local home
  home=$(resolve:home "${kit_root}" "agents/$1") || return 1
  printf '%s\n' "${home}/packages/agents/$1/files/repo/$2"
}

# apps/audit:region-holds <path> <text> [<flag>] succeeds where the file's kit region holds the text, a whole line of
# it with the flag --line-regexp. A region kit cannot read holds nothing.
apps/audit:region-holds() {
  local region
  region=$(region:read "$1" 2>/dev/null) || return 1
  grep --quiet --fixed-strings ${3:+"$3"} -- "$2" <<<"${region}"
}

# apps/audit:placed <harness> <dir> prints the first file the agent package places under <dir> that is not here, or
# nothing when every one is.
apps/audit:placed() {
  local root file
  local -a files
  root=$(apps/audit:package-file "$1" "$2") || return 1
  [[ -d "${root}" ]] || return 0
  list:read files apps/audit:files "${root}" || return 1
  for file in "${files[@]}"
  do
    [[ -f "$2/${file}" ]] || { printf '%s\n' "$2/${file}"; return 0; }
  done
}

# apps/audit:files <dir> prints each file under the directory, by its path there, sorted.
apps/audit:files() {
  (cd "$1" && find . -type f -print) | cut -c 3- | LC_ALL=C sort
}

# apps/audit:entries <harness> <path> <filter> prints how many of the entries the package's file at <path> holds, as
# <filter> lists them, the file here lacks, or nothing when kit cannot read the file here as JSON. A file that is not
# here lacks every one.
apps/audit:entries() {
  local ours
  ours=$(apps/audit:package-file "$1" "$2") || return 1
  if [[ ! -f "$2" ]]
  then
    jq --raw-output "[$3] | length" "${ours}"
    return
  fi
  jq --null-input --raw-output --slurpfile ours "${ours}" --slurpfile here "$2" \
    "[\$ours[0] | $3] - [\$here[0] | $3] | length" 2>/dev/null || true  # the caller says a file jq cannot read
}

# audit_filters gives each list of entries a package merges, by name, as apps/audit:entries reads it: a value
# that is not there, or not a list, holds none, and Gemini's context files may be one name alone.
declare -grA audit_filters=(
  [deny]='.permissions.deny? // [] | arrays | .[]'
  [hook]='.hooks.PreToolUse? // [] | arrays | .[] | .hooks? // [] | arrays | .[] | .command?'
  [context]='.context.fileName? // [] | if type == "string" then . else arrays | .[] end'
  [exclude]='.tools.exclude? // [] | arrays | .[]'
)

# apps/audit:enforcement-generic <default> prints the rows of any agent that reads AGENTS.md: kit's region there,
# advice, and no deny list kit can read, which is unknown.
apps/audit:enforcement-generic() {
  if apps/audit:region-holds AGENTS.md .kit/1st.md
  then
    apps/audit:enforced generic advisory AGENTS.md active documented \
      "AGENTS.md carries kit's region, which sends an agent to .kit/1st.md; an agent that reads AGENTS.md reads it" \
      'an agent that does not follow it'
  else
    apps/audit:enforced generic advisory AGENTS.md missing probed \
      "AGENTS.md carries no kit region that sends an agent to .kit/1st.md: run kit generate" 'every agent'
  fi
  apps/audit:enforced generic best-effort 'deny list' unknown unverified \
    "kit delivers no deny list for a harness it has no package for, and reads no harness's own" -
}

# apps/audit:enforcement-claude <default> prints Claude Code's rows: .claude/CLAUDE.md's import of AGENTS.md, advice;
# and the deny list and the context hook merged into .claude/settings.json, each unknown where it is there.
apps/audit:enforcement-claude() {
  local lacked file=.claude/settings.json
  local -a passes
  list:read passes apps/audit:deny-passes "$1" || return 1
  if apps/audit:region-holds .claude/CLAUDE.md @../AGENTS.md --line-regexp
  then
    apps/audit:enforced claude advisory .claude/CLAUDE.md active documented \
      "imports AGENTS.md, which Claude Code reads when a session starts" 'an agent that does not follow it'
  else
    apps/audit:enforced claude advisory .claude/CLAUDE.md missing probed \
      ".claude/CLAUDE.md does not import AGENTS.md in kit's region, and Claude Code then skips it: run kit generate" \
      'every agent'
  fi
  lacked=$(apps/audit:entries claude "${file}" "${audit_filters[deny]}") || return 1
  apps/audit:listed claude 'deny list' "${file}" "${lacked}" "permissions.deny" \
    "Claude Code refuses a command these literal patterns match" "${passes[@]}" || return 1
  lacked=$(apps/audit:entries claude "${file}" "${audit_filters[hook]}") || return 1
  apps/audit:listed claude 'context hook' "${file}" "${lacked}" hooks.PreToolUse \
    "Claude Code runs kit context --claude before each Write, Edit, and MultiEdit, which shows the rules" \
    'an edit through Bash: sed -i, or a redirect' 'an edit made with the rules unread: it refuses none' || return 1
}

# apps/audit:listed <harness> <mechanism> <file> <lacked> <key> <documented> <pass>… prints the best-effort row of
# entries a package merges into a harness's file: unknown, as documented, where the file holds every one, since kit
# cannot probe that the harness honors them; missing, probed, where it lacks any; unknown, unverified, where kit cannot
# read it.
apps/audit:listed() {
  local harness=$1 mechanism=$2 file=$3 lacked=$4 key=$5 documented=$6
  shift 6
  if [[ -z "${lacked}" ]]
  then
    apps/audit:enforced "${harness}" best-effort "${mechanism}" unknown unverified \
      "kit cannot read ${file} as JSON, so cannot tell what the harness reads there" -
  elif (( lacked == 0 ))
  then
    apps/audit:enforced "${harness}" best-effort "${mechanism}" unknown documented \
      "merged into ${file}'s ${key}: ${documented}; kit cannot probe that it does in a session" "$@"
  else
    apps/audit:enforced "${harness}" best-effort "${mechanism}" missing probed \
      "${file}'s ${key} lacks ${lacked} of the package's entries: run kit hydrate, and kit check names them" \
      'everything it lacks'
  fi
}

# apps/audit:enforcement-gemini <default> prints Gemini CLI's rows: AGENTS.md among the context files in
# .gemini/settings.json, advice; and the deny list, tools.exclude, unknown where it is there.
apps/audit:enforcement-gemini() {
  local lacked file=.gemini/settings.json
  local -a passes
  list:read passes apps/audit:deny-passes "$1" || return 1
  # Gemini CLI reads AGENTS.md where its context files name it, whatever else they name.
  lacked=$(jq --raw-output "if [${audit_filters[context]}] | index([\"AGENTS.md\"]) then 0 else 1 end" "${file}" \
    2>/dev/null) || lacked=''  # a file that is not here, or not JSON, says so below
  [[ -f "${file}" ]] || lacked=1
  if [[ "${lacked}" == 0 ]]
  then
    apps/audit:enforced gemini advisory "${file}" active documented \
      "names AGENTS.md among the context files, which Gemini CLI reads when a session starts" \
      'an agent that does not follow it'
  elif [[ -z "${lacked}" ]]
  then
    apps/audit:enforced gemini advisory "${file}" unknown unverified \
      "kit cannot read ${file} as JSON, so cannot tell what Gemini CLI reads" -
  else
    apps/audit:enforced gemini advisory "${file}" missing probed \
      "${file} does not name AGENTS.md among the context files, and Gemini CLI then skips it: run kit hydrate" \
      'every agent'
  fi
  lacked=$(apps/audit:entries gemini "${file}" "${audit_filters[exclude]}") || return 1
  apps/audit:listed gemini 'deny list' "${file}" "${lacked}" tools.exclude \
    "Gemini CLI refuses a shell command that starts with one, each part of a chained one" "${passes[@]}" || return 1
}

# apps/audit:enforcement-mirrored <harness> <dir> <name> prints the rows of a harness that reads AGENTS.md on its own
# and the package's files under <dir>, applied by path, advice; it offers no deny list a repository can commit.
apps/audit:enforcement-mirrored() {
  local harness=$1 dir=$2 name=$3 lacking
  lacking=$(apps/audit:placed "${harness}" "${dir}") || return 1
  if [[ -z "${lacking}" ]]
  then
    apps/audit:enforced "${harness}" advisory "${dir}" active documented \
      "${name} reads AGENTS.md, and applies the package's files under ${dir} by path" \
      'an agent that does not follow it'
  else
    apps/audit:enforced "${harness}" advisory "${dir}" missing probed \
      "${lacking} is not here: run kit hydrate" 'every agent'
  fi
  apps/audit:enforced "${harness}" best-effort 'deny list' missing documented \
    "${name} offers no deny list a repository can commit, so nothing on its side refuses a raw git command" \
    'every command'
}

# apps/audit:enforcement-copilot <default> prints GitHub Copilot's rows, as apps/audit:enforcement-mirrored does.
apps/audit:enforcement-copilot() { apps/audit:enforcement-mirrored copilot .github/instructions 'GitHub Copilot'; }

# apps/audit:enforcement-cursor <default> prints Cursor's rows, as apps/audit:enforcement-mirrored does.
apps/audit:enforcement-cursor() { apps/audit:enforcement-mirrored cursor .cursor/rules Cursor; }

# apps/audit:enforcement-floor <package>… prints the repository's local floor: the pre-commit hook, active where git
# runs lefthook before each commit in this clone and lefthook is on PATH, and lefthook.yml extending
# .kit/lefthook.yml, as the hooks/lefthook package expects, where the active packages, the words, hold it. A commit
# made with --no-verify, or with LEFTHOOK=0, passes both.
apps/audit:enforcement-floor() {
  local hook
  local -a passes=('git commit --no-verify' 'LEFTHOOK=0 git commit')
  hook=$(git rev-parse --path-format=absolute --git-path hooks/pre-commit) || return 1
  if [[ ! -x "${hook}" ]] || ! grep --quiet lefthook "${hook}" 2>/dev/null
  then
    apps/audit:enforced repository 'local floor' 'pre-commit hook' missing probed \
      "git runs no lefthook before a commit in this clone: run lefthook install" 'every commit'
  elif ! command -v lefthook >/dev/null
  then
    apps/audit:enforced repository 'local floor' 'pre-commit hook' unknown unverified \
      "${hook} runs lefthook, which is not on PATH here, so kit cannot tell what a commit from here runs" -
  else
    apps/audit:enforced repository 'local floor' 'pre-commit hook' active probed \
      "git runs ${hook}, which runs lefthook, before each commit in this clone" "${passes[@]}"
  fi
  if ! list:among hooks/lefthook "$@"
  then
    apps/audit:enforced repository 'local floor' lefthook.yml missing probed \
      "the hooks/lefthook package is not active, so lefthook runs none of kit's commands: run kit packages add\
 hooks/lefthook" 'every commit'
  elif ! grep --quiet '\.kit/lefthook\.yml' lefthook.yml 2>/dev/null
  then
    apps/audit:enforced repository 'local floor' lefthook.yml missing probed \
      "lefthook.yml does not extend .kit/lefthook.yml: put extends: [.kit/lefthook.yml] in it" 'every commit'
  elif [[ ! -f .kit/lefthook.yml ]]
  then
    apps/audit:enforced repository 'local floor' lefthook.yml missing probed \
      "lefthook.yml extends .kit/lefthook.yml, which is not here: run kit hydrate" 'every commit'
  else
    apps/audit:enforced repository 'local floor' lefthook.yml active probed \
      "lefthook.yml extends .kit/lefthook.yml, so lefthook runs kit lint and kit generate --check" "${passes[@]}"
  fi
}

# apps/audit:enforcement-ci <package>… prints the row of kit's CI job: active, as GitHub documents that Actions runs a
# workflow on each pull request, where the github/ci package, among the active packages, the words, placed it here.
# The job binds a merge only where a control requires its check, the required check row.
apps/audit:enforcement-ci() {
  local file=.github/workflows/kit-ci.yml
  if ! list:among github/ci "$@"
  then
    apps/audit:enforced repository authoritative "kit's CI job" missing probed \
      "the github/ci package is not active, so no CI job runs kit ci: kit packages add github/ci" 'every merge'
  elif [[ -f "${file}" ]]
  then
    apps/audit:enforced repository authoritative "kit's CI job" active documented \
      "${file} runs kit ci on each pull request into the default branch, its check ${audit_ci_check}" \
      "a merge no control requires ${audit_ci_check} for"
  else
    apps/audit:enforced repository authoritative "kit's CI job" missing probed \
      "the github/ci package is active, and ${file} is not here: run kit hydrate" 'every merge'
  fi
}

# apps/audit:enforcement-forge prints what kit read of the forge, as apps::required takes it: {state, error,
# controls, bypasses, bypassed, bypassed_error}: the controls on the default branch, and who may bypass them, which a
# forge without bypasses does not say. It returns 1, after it, where a read failed, its line the error.
apps/audit:enforcement-forge() {
  local said line controls bypasses failed=false status=0
  if forge:none
  then
    printf '{"state":"none","controls":[],"bypasses":[]}\n'
    return 0
  fi
  file:temp said audit || return 1
  if ! forge:lacks >/dev/null 2>"${said}"
  then
    status=1
  elif ! forge:has rulesets
  then
    printf '{"state":"lacked","controls":[],"bypasses":[]}\n'
    return 0
  else
    controls=$(forge:rulesets 2>"${said}") || status=$?
  fi
  if (( status == 3 ))
  then
    printf '{"state":"off","controls":[],"bypasses":[]}\n'
    return 0
  elif (( status != 0 ))
  then
    line=$(grep -v '^[[:space:]]*$' "${said}" | tail -n 1 || true)  # grep's exit 1: the read said nothing
    jq --null-input --compact-output --arg error "${line#❌ }" \
      '{state: "failed", error: $error, controls: [], bypasses: []}' || return 1
    return 1
  fi
  controls=$(jq --slurp --compact-output . <<<"${controls}") || return 1
  bypasses=$(forge:bypasses 2>"${said}") || status=$?
  if (( status == 0 ))
  then
    bypasses=$(jq --slurp --compact-output . <<<"${bypasses}") || return 1
    jq --null-input --compact-output --argjson controls "${controls}" --argjson bypasses "${bypasses}" \
      '{state: "read", controls: $controls, bypasses: $bypasses}' || return 1
    return 0
  fi
  (( status == 3 )) || failed=true
  line=$(grep -v '^[[:space:]]*$' "${said}" | tail -n 1 || true)  # grep's exit 1: the read said nothing
  jq --null-input --compact-output --argjson controls "${controls}" --arg error "${line#❌ }" \
    --argjson failed "${failed}" '
    {state: "read", controls: $controls, bypasses: null,
     bypassed: (if $failed then "who may bypass it could not be read" else "the forge says not who may bypass it" end),
     bypassed_error: (if $failed then $error else null end)}' || return 1
  [[ "${failed}" == false ]]
}

# apps/audit:merged-policy reads the policy's document, one checkout's or a workspace's merged, and prints it with
# its rows folded, as apps::scopes folds them; apps:across calls it on the members' document.
apps/audit:merged-policy() {
  jq --compact-output -L "${kit_jq}" 'import "apps" as apps; .rows |= apps::scopes'
}

# apps/audit:mark-evidence prints the forge's marks of its runs and its releases.
apps/audit:mark-evidence() { apps:forge-mark runs releases; }

# apps/audit:mark-findings prints the forge's marks of its runs and its alerts.
apps/audit:mark-findings() { apps:forge-mark runs alerts; }

# apps/audit:mark-enforcement prints the hash of the files the enforcement stage's own rows read: kit.yml, which says
# the active packages, each harness's, the git hooks' pre-commit, lefthook.yml and .kit/lefthook.yml, and kit's CI
# job. The forge's rows move with none of them, and are read again on Refresh.
apps/audit:mark-enforcement() {
  local hook
  hook=$(git rev-parse --path-format=absolute --git-path hooks/pre-commit) || return 1
  apps:hash "$(settings:file)" AGENTS.md .claude/CLAUDE.md .claude/settings.json .gemini/settings.json \
    .github/instructions .cursor/rules "${hook}" lefthook.yml .kit/lefthook.yml .github/workflows/kit-ci.yml
}

# apps/audit:mark-policy prints the marks of the policy of each checkout apps/audit:checkouts lists, as
# apps/audit:mark-checkout prints them, each after its branch; nothing when the clone has the one checkout and none is
# there. A checkout added or removed changes it, as the rows it draws change.
apps/audit:mark-policy() {
  local path branch mark checkout any=false
  local -a marks=() checkouts
  list:read checkouts apps/audit:checkouts || return 1
  for checkout in "${checkouts[@]}"
  do
    IFS=$'\t' read -r path branch <<<"${checkout}"
    mark=$(cd "${path}" && apps/audit:mark-checkout) || return 1
    [[ -z "${mark}" ]] || any=true
    marks+=("${branch}=${mark}")
  done
  [[ "${any}" == true ]] || (( ${#checkouts[@]} > 1 )) || return 0
  printf '%s\n' "${marks[*]}"
}

# apps/audit:mark-checkout prints the hash of the checkout's kit.yml and .kit/policy, and on a branch past the
# default, whose policy is its base's, the base's kit.yml and .kit/policy as git names them, which a fetch moves;
# nothing when none is there. A base that cannot be named leaves the base out: kit policy --rules then says why.
apps/audit:mark-checkout() {
  local hash past base='' object
  local -a marks=()
  hash=$(apps:hash "$(settings:file)" .kit/policy) || return 1
  [[ -z "${hash}" ]] || marks+=("${hash}")
  if past=$(policy:branch-past 2>/dev/null) && [[ -n "${past}" ]] && base=$(policy:base 2>/dev/null)
  then
    for object in kit.yml .kit/policy
    do
      object=$(git rev-parse --quiet --verify "${base}:${object}" 2>/dev/null) || continue
      marks+=("${object}")
    done
  fi
  (( ${#marks[@]} == 0 )) || printf '%s\n' "${marks[*]}"
}

# ----------------------------------------------------------------------------------------------------------------------
