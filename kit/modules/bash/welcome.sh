# shellcheck shell=bash
# welcome.sh - kit welcome's checks, their fixes, and the pages of its steps (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root and kit_jq are exported by modules/bash/1st.sh
#
# kit welcome walks a newcomer through five steps, each a page: the introduction, toolchains, setup, customize, and
# the terms. Steps 2 and 3 hold checks, run here. A check is the machine's, judged once, or a folder's, judged inside
# each folder of the workspace. Each check is one JSON object:
#
#   id            stable, in kebab-case: tool-yq, toolchain-project, linter-shellcheck, forge-login-github.com; a
#                 folder's check has the same id in every folder, and its folder tells them apart
#   step          the step it belongs to: toolchains or setup
#   ok            true when it holds
#   line          one short sentence: what holds, or what is wrong
#   fix           the command that fixes it, or null when it holds or kit knows none
#   interactive   true when the fix needs the person at the terminal, as gh auth login does
#   more          where to read more, or the other commands beside the fix, or null
#   runs          false when kit only shows the fix and never runs it, as for a line that changes the person's own
#                 shell; true for every other check. kit welcome --fix refuses a check whose runs is false.
#   folder        the folder's absolute path, where its fix runs; null for a machine check
#   repository    the folder's repository: its member, host and path, or, with no remote, its desk's folder name
#   desk          true for a repository's desk, false for one of its worktrees
#   branch        the folder's branch, or null where none is checked out
#   kit           the kit version the folder's kit.yml declares, or null where it declares none
#   findings      kit-check's alone, where kit check finds more than one problem: each problem as kit check names
#                 it, without the command after "run:" when every problem names the same one, which is then the fix
#
# A machine check's repository, desk, branch, and kit are null too.
#
# The folders are the workspace's, as kit workspaces --current names it, or kit --workspace=<workspace>: for a
# repository, its desk, then each of its worktrees, as git worktree list gives them; for a group, each member's, in
# the group's order, for every checkout of it under the roots. --workspace=none names no group, as for kit flow: the
# repository here. With no workspace, outside a repository and with no group used, there is no folder, and only the
# machine checks run.
#
# The checks run in this order:
#
#   - the machine, toolchains: bash, git, jq, and yq, which kit runs on; then, only where brew is installed at a
#     Homebrew prefix but is not on PATH, brew-path, whose fix kit shows and never runs; then the system toolchain
#   - the machine, setup: for each distinct forge host among every folder's remotes, its tool installed, then logged
#     in to that host, once per host; then, where KIT_TERMINAL_PROBES holds what the terminal answered, the terminal
#     kit runs in: its heading, each finding of its settings, and each probe, as modules/bash/terminal.sh judges them,
#     each fix shown and never run
#   - each folder, setup: a git repository; kit installed, with the version kit.yml declares; then, where kit is,
#     .kit/ hydrated and the git hooks installed when hooks/lefthook is active; the project toolchain; then, where kit
#     is, lefthook when hooks/lefthook is active, each dependency provider whose manifest is there, installed from its
#     lock, each linter the active packages' rules and styles name under lint:, and kit check clean
#
# The folders are judged at once, as many at a time as the machine has processors, and each prints its checks in
# the folders' order. Nothing is judged twice: a machine check once, each folder's checks once in it, and kit check
# once per folder, by the kit release that folder's kit.yml pins.
#
# kit finds the toolchains as kit toolchains does, and never assumes one. The checks only report: they never ask
# which toolchain to use, and never record one. A missing tool's fix:
#
#   - the machine's: the system toolchain's command, such as its install of the tool, else none, and more names the
#     tool's site
#   - a folder's: kit tools install where that folder's project toolchain is settled; ambient, the system toolchain's
#     command; else none, and more says where to read
#
# Only welcome:fix runs a fix, and only the one it is asked for: a folder's check's in that folder.
#
# The pages are markdown files under .kit/welcome/, which the kit package delivers. A page's name is its order, a
# dash, then its step's id: 2-toolchains.md. A house package replaces one by placing a file of the same name, or adds
# one; the later package wins.
#
# see: bin/kit-welcome, modules/bash/toolchain.sh, modules/bash/workspaces.sh, issue #51

# The yq check says what yaml.sh says when it loads without yq, so its own line is not printed twice.
use yaml 2>/dev/null
use settings forge materialize render place toolchain dependencies workspaces file terminal markdown bots

# welcome_ids holds the machine checks' ids, in order, and welcome_folder_ids the checks of the folder here, in order;
# the maps beside them hold what judging one needs: the rule that names each linter, and the forge of each host.
# welcome:plan-machine and welcome:plan-folder fill them.
declare -ga welcome_ids=() welcome_folder_ids=()
declare -gA welcome_linter_rule=() welcome_host_forge=()
# welcome_terminal holds each terminal check by its id, as terminal:checks judged them once, in welcome:plan-machine.
declare -gA welcome_terminal=()

# welcome_folders holds the workspace's folders, in order, each "<repository>\x1f<desk>\x1f<folder>": its repository,
# true for a desk and false for a worktree, and its absolute path. welcome:folders fills it.
declare -ga welcome_folders=()

# welcome_progress is true when kit welcome --checks --progress asked to say on stderr what is being checked as it goes;
# welcome:progress says it.
declare -g welcome_progress=false

# welcome_where is what a check adds to say where it holds: {folder, repository, desk, branch, kit}, each null for a
# machine check. welcome:folder-checks sets it for the folder it judges.
declare -g welcome_where='{"folder":null,"repository":null,"desk":null,"branch":null,"kit":null}'

# welcome:record <id> <ok> <line> [<fix> [<interactive> [<more> [<runs>]]]] prints one check as JSON, on one line, in
# the step its id belongs to, with welcome_where's folder. An empty fix or more is null. runs is true unless the check
# says kit only shows its fix. A check that holds and still has a fix is an offer: nothing is wrong, and its fix is a
# step kit suggests, which counts as no failure.
welcome:record() {
  jq --null-input --compact-output --arg id "$1" --arg step "$(welcome:step "$1")" --argjson ok "$2" --arg line "$3" \
    --arg fix "${4:-}" --argjson interactive "${5:-false}" --arg more "${6:-}" --argjson runs "${7:-true}" \
    --argjson where "${welcome_where}" \
    '{id: $id, step: $step, ok: $ok, line: $line, fix: ($fix | if . == "" then null else . end),
      interactive: $interactive, more: ($more | if . == "" then null else . end), runs: $runs} + $where'
}

# welcome:step <id> prints the step a check belongs to: toolchains for what kit runs on and the system toolchain, and
# setup for the rest, a folder's project toolchain among them.
welcome:step() {
  case "$1" in
    (tool-bash | tool-git | tool-jq | tool-yq | brew-path | toolchain-system) printf 'toolchains\n' ;;
    (*) printf 'setup\n' ;;
  esac
}

# welcome:machine <id> succeeds for the id of a machine check, which holds for the whole machine: what kit runs on,
# brew on PATH, the system toolchain, each forge host's tool and login, and the terminal's. Every other check is a
# folder's.
welcome:machine() {
  case "$1" in
    (tool-bash | tool-git | tool-jq | tool-yq | brew-path | toolchain-system) ;;
    (forge-cli-* | forge-login-* | terminal-*) ;;
    (*) return 1
  esac
}

# welcome:site <tool> prints the page that says how to install the tool, or nothing for a tool kit does not know.
welcome:site() {
  case "$1" in
    (bash) printf 'https://www.gnu.org/software/bash/\n' ;;
    (git) printf 'https://git-scm.com/downloads\n' ;;
    (jq) printf 'https://jqlang.org/download/\n' ;;
    (yq) printf 'https://github.com/mikefarah/yq#install\n' ;;
    (lefthook) printf 'https://lefthook.dev/installation/\n' ;;
    (shellcheck) printf 'https://github.com/koalaman/shellcheck#installing\n' ;;
    (actionlint) printf 'https://github.com/rhysd/actionlint/blob/main/docs/install.md\n' ;;
    (markdownlint-cli2) printf 'https://github.com/DavidAnson/markdownlint-cli2#install\n' ;;
    (gh) printf 'https://cli.github.com\n' ;;
    (glab) printf 'https://gitlab.com/gitlab-org/cli#installation\n' ;;
    (*) ;;
  esac
}

# welcome_chosen holds each role's toolchain, as welcome:find found it. welcome:plan fills it, and a fix fills it
# again, in the main shell, so the checks that run in subshells read it rather than finding it each time.
declare -gA welcome_chosen=()

# welcome:find <role> sets welcome_chosen[<role>] to the toolchain for the role, as kit toolchains reports it: a
# provider, ambient, or nothing when several are usable and none is chosen. It never asks and never records. Where
# kit cannot read the system choice, as without yq, the one usable system toolchain stands in for it.
welcome:find() {
  local role=$1 chosen
  local -a usable
  if ! chosen=$(toolchain:chosen "${role}" list 2>/dev/null)
  then
    chosen=''
    # A list kit cannot read gives no stand-in; the role's toolchain check then says kit could not look.
    if list:read usable toolchain:usable "${role}" 2>/dev/null
    then
      [[ "${role}" != system || ${#usable[@]} -ne 1 ]] || chosen=${usable[0]}
    fi
  fi
  welcome_chosen[${role}]=${chosen}
}

# welcome:chosen <role> prints the toolchain for the role, as welcome:find sets it, finding it first where it has not.
welcome:chosen() {
  [[ -n "${welcome_chosen[$1]+x}" ]] || welcome:find "$1"
  printf '%s\n' "${welcome_chosen[$1]}"
}

# welcome:remedy <tool> prints the system toolchain's command that installs the tool, and fails where there is no
# system toolchain, or it names no command.
welcome:remedy() {
  toolchain:remedy "$(welcome:chosen system)" "$1"
}

# welcome:toolchain-has <tool> succeeds when the project toolchain has installed the tool: it is in a directory the
# toolchain puts on PATH.
welcome:toolchain-has() {
  local project dir
  local -a dirs=() more
  project=$(welcome:chosen project)
  # Directories kit cannot read hold no tool, so the check names the install fix.
  if [[ -n "${project}" && "${project}" != ambient ]]
  then
    list:read more toolchain:call "${project}" path 2>/dev/null || return 1
    dirs+=("${more[@]}")
  fi
  list:read more dependencies:path 2>/dev/null || return 1
  for dir in "${dirs[@]}" "${more[@]}"
  do
    [[ -n "${dir}" && -x "${dir}/$1" ]] && return 0
  done
  return 1
}

# welcome:install <scope> <tool> prints the fix that installs the tool and where to read more, separated by \x1f:
#
#   - machine: the system toolchain's command, else no fix, and the tool's site
#   - folder: kit tools install where the folder's project toolchain is settled; ambient, the system toolchain's
#     command, else no fix and the tool's site; with several usable and none chosen, no fix, and the toolchains doc
welcome:install() {
  local scope=$1 tool=$2 project='' fix
  [[ "${scope}" != folder ]] || project=$(welcome:chosen project)
  if [[ "${scope}" == folder && -n "${project}" && "${project}" != ambient ]]
  then
    printf 'kit tools install\x1f\n'
  elif [[ "${scope}" == folder && -z "${project}" ]]
  then
    printf '\x1f%s\n' "${welcome_toolchains_doc}"
  elif fix=$(welcome:remedy "${tool}") && [[ -n "${fix}" ]]
  then
    printf '%s\x1f\n' "${fix}"
  else
    printf '\x1f%s\n' "$(welcome:site "${tool}")"
  fi
}

# welcome_toolchains_doc is where to read about choosing a toolchain.
declare -gr welcome_toolchains_doc=https://github.com/empowerite/kit/blob/main/docs/toolchains.md

# welcome:missing <id> <tool> <line> [<more>] prints the check of a tool that does not run: its install fix, and where
# to read more, <more> when kit knows no site for it. In a folder, a tool its project toolchain installed, in a
# directory not on PATH, needs its directory on PATH rather than another install.
welcome:missing() {
  local id=$1 tool=$2 line=$3 scope=folder fix more
  ! welcome:machine "${id}" || scope=machine
  if [[ "${scope}" == folder ]] && welcome:toolchain-has "${tool}"
  then
    welcome:record "${id}" false "${tool} is installed, but not on PATH; kit tools path names its directory" '' \
      false "${welcome_toolchains_doc}"
    return 0
  fi
  IFS=$'\x1f' read -r fix more < <(welcome:install "${scope}" "${tool}")
  [[ -n "${fix}" ]] || more=${more:-${4:-}}
  welcome:record "${id}" false "${line}" "${fix}" false "${more}"
}

# welcome:toolchain <role> prints the check of the role's toolchain. It holds when one is chosen, when one alone is
# usable, and when none is, which is ambient. With several usable and none chosen, it fails: its fix chooses the
# first, and more names the command for each of the others.
welcome:toolchain() {
  local role=$1 chosen flag candidate names='' others=''
  local -a usable
  chosen=$(welcome:chosen "${role}")
  if ! list:read usable toolchain:usable "${role}" 2>/dev/null
  then
    welcome:record "toolchain-${role}" false "kit could not look for a toolchain for the ${role} here" '' false \
      "${welcome_toolchains_doc}"
    return 0
  fi
  flag=$(toolchain:flag "${role}")
  case "${chosen}" in
    (ambient)
      if (( ${#usable[@]} ))
      then
        welcome:record "toolchain-${role}" true "the ${role} toolchain is ambient, as chosen: kit uses what is on PATH"
      elif [[ "${role}" == project ]]
      then
        welcome:record "toolchain-${role}" true \
          "no toolchain is usable for the project, so its tools are ambient: kit uses what is on PATH"
      else
        welcome:record "toolchain-${role}" true \
          "no toolchain is usable for the system, so it is ambient: kit uses what is on PATH"
      fi
      ;;
    ('')
      if [[ "${role}" == project && -n "${KIT_TOOLCHAIN:-}" ]]
      then
        welcome:record "toolchain-${role}" false \
          "KIT_TOOLCHAIN names ${KIT_TOOLCHAIN}, which is not usable for the project here" '' false \
          "${welcome_toolchains_doc}"
        return 0
      fi
      if (( ${#usable[@]} < 2 ))
      then
        welcome:record "toolchain-${role}" false "kit could not look for a toolchain for the ${role} here" '' false \
          "${welcome_toolchains_doc}"
        return 0
      fi
      for candidate in "${usable[@]}"
      do
        names+="${names:+, }${candidate}"
        [[ "${candidate}" == "${usable[0]}" ]] || others+="${others:+; }kit toolchains use ${candidate}${flag}"
      done
      welcome:record "toolchain-${role}" false "several toolchains are usable for the ${role}: ${names}; choose one" \
        "kit toolchains use ${usable[0]}${flag}" false "or: ${others}"
      ;;
    (*)
      if [[ "${role}" == project ]]
      then
        welcome:record "toolchain-${role}" true "${chosen} is the project toolchain; kit tools lists what it pins"
      else
        welcome:record "toolchain-${role}" true "${chosen} is the system toolchain; it names the fix for a missing tool"
      fi
      ;;
  esac
}

# welcome:dependencies <provider> prints the check of a dependency provider whose manifest is here. It holds when
# every package its lock pins is installed. Its fix is kit tools install. A manifest without its lock, or a provider
# whose tool is not on PATH, has no fix kit can run.
welcome:dependencies() {
  local provider=$1 file lock
  local -a usable
  file=$(dependencies:call "${provider}" file)
  lock=$(dependencies:call "${provider}" lock)
  if [[ ! -f "${lock}" ]]
  then
    welcome:record "dependencies-${provider}" false \
      "${file} is here without ${lock}, so ${provider} pins nothing; write the lock with ${provider}, and commit it" \
      '' false "${welcome_toolchains_doc}"
    return 0
  fi
  list:read usable dependencies:usable || return 1
  if ! list:among "${provider}" "${usable[@]}"
  then
    welcome:record "dependencies-${provider}" false "${lock} is here, but ${provider} is not on PATH" '' false \
      "${welcome_toolchains_doc}"
  elif dependencies:installed "${provider}"
  then
    welcome:record "dependencies-${provider}" true "${provider} installed every package ${lock} pins"
  else
    welcome:record "dependencies-${provider}" false "${provider} has not installed every package ${lock} pins" \
      'kit tools install'
  fi
}

# welcome:version <tool> prints the first line the tool prints for --version, and fails when it does not run: a
# tool manager's shim with no version behind it is on PATH, and fails.
welcome:version() {
  local out
  command -v "$1" >/dev/null 2>&1 || return 1
  out=$("$1" --version 2>/dev/null) || return 1
  printf '%s\n' "${out%%$'\n'*}"
}

# welcome:tool <id> <tool> <what it does> [<more>] prints the check that a tool runs, for a tool any version of which
# will do.
welcome:tool() {
  if welcome:version "$2" >/dev/null
  then
    welcome:record "$1" true "$2 is on PATH; $3"
  else
    welcome:missing "$1" "$2" "$2 is not installed; $3" "${4:-}"
  fi
}

# welcome:linter <id> <tool> <rule> prints the check of a linter the rule names. kit lint looks in the project
# toolchain's directories and each installed dependency provider's before PATH, so a linter there holds, wherever PATH
# leads.
welcome:linter() {
  local id=$1 tool=$2 rule=$3 dir
  local -a dirs=()
  # Directories kit cannot read hold no linter, so the check falls back to PATH.
  list:read dirs dependencies:installed-path 2>/dev/null || dirs=()
  for dir in "${dirs[@]}"
  do
    if [[ -n "${dir}" && -x "${dir}/${tool}" ]]
    then
      welcome:record "${id}" true "${tool} is installed in ${dir#"${PWD}/"}; kit lint runs it for ${rule}"
      return 0
    fi
  done
  if welcome:toolchain-has "${tool}"
  then
    welcome:record "${id}" true "${tool} is installed by $(welcome:chosen project); kit lint runs it for ${rule}"
    return 0
  fi
  welcome:tool "${id}" "${tool}" "kit lint runs it for ${rule}" "${rule}"
}

# welcome:bash prints the check of the bash that runs kit. The loader refuses one before 4.4, so it always holds.
welcome:bash() {
  welcome:record tool-bash true "bash ${BASH_VERSION%%(*} runs kit; it needs 4.4 or later"
}

# welcome:jq prints the check of jq: 1.7 or later, which reads every verb's arguments.
welcome:jq() {
  local version
  if ! version=$(welcome:version jq)
  then
    welcome:missing tool-jq jq "jq is not installed; kit reads its JSON through it"
  elif [[ "${version}" =~ ^jq-([0-9]+)\.([0-9]+) ]] \
    && (( BASH_REMATCH[1] > 1 || (BASH_REMATCH[1] == 1 && BASH_REMATCH[2] >= 7) ))
  then
    welcome:record tool-jq true "${version/-/ } reads kit's JSON; it needs 1.7 or later"
  else
    welcome:missing tool-jq jq "${version/-/ } is too old; kit needs jq 1.7 or later"
  fi
}

# welcome:yq prints the check of yq: mikefarah's yq 4, which reads kit.yml and every other YAML file.
welcome:yq() {
  local version
  if ! version=$(welcome:version yq)
  then
    welcome:missing tool-yq yq "yq is not installed; kit reads its YAML through it"
  elif [[ "${version}" == *mikefarah* && "${version}" =~ version\ v(4[.0-9]*) ]]
  then
    welcome:record tool-yq true "yq ${BASH_REMATCH[1]} reads kit.yml; kit needs yq 4"
  else
    welcome:missing tool-yq yq "the yq on PATH is not yq 4, mikefarah's, which kit needs"
  fi
}

# welcome:delivered prints "<path>\t<source>" for each rule and style the active packages deliver. Where .kit/ is
# hydrated, they are the files there, as kit lint reads them; otherwise they are read from the packages.
welcome:delivered() {
  local path
  local -a packages
  if materialize:hydrated
  then
    for path in .kit/rules/*.md .kit/styles/*.md
    do
      [[ ! -f "${path}" ]] || printf '%s\t%s\n' "${path}" "${path}"
    done
    return 0
  fi
  # welcome plans the checks it can read: settings it cannot read, or a release it cannot render, plan no linter.
  list:read packages settings:packages 2>/dev/null || return 0
  (( ${#packages[@]} )) || return 0
  render:all "${kit_root}" "${packages[@]}" >/dev/null 2>&1 || return 0
  place:delivered "${kit_root}" repo "${packages[@]}" 2>/dev/null
}

# welcome:linters prints "<linter>\t<rule>" for each linter the active packages' rules and styles name under lint:,
# once each, in the order of the files that name them. kit lint runs the same commands.
welcome:linters() {
  local path src command
  local -A seen=()
  while IFS=$'\t' read -r path src
  do
    [[ "${path}" =~ ^\.kit/(rules|styles)/[^/]+\.md$ ]] || continue
    command=$(yaml:value "${src}" lint 2>/dev/null) || continue
    command=${command%% *}
    [[ -n "${command}" && -z "${seen[${command}]+x}" ]] || continue
    seen[${command}]=1
    printf '%s\t%s\n' "${command}" "${path}"
  done < <(welcome:delivered)
}

# welcome:forges prints "<forge>\t<host>" for each distinct host among the remotes whose forge is github or gitlab,
# the primary first. A host on github.com or gitlab.com under another name is that host, where its tool logs in.
welcome:forges() {
  local remote forge host
  local -a remotes
  local -A seen=()
  # welcome plans the checks it can read: remotes it cannot read plan no forge check.
  list:read remotes forge:remotes 2>/dev/null || return 0
  for remote in "${remotes[@]}"
  do
    forge=$(forge:of "${remote}" 2>/dev/null) || continue
    host=$(forge:host "${remote}")
    case "${forge}:${host}" in
      (github:*.github.com) host=github.com ;;
      (gitlab:*.gitlab.com) host=gitlab.com ;;
      (github:* | gitlab:*) ;;
      (*) continue
    esac
    # A host reaches a command line, so it is a host name and nothing more.
    [[ "${host}" =~ ^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?$ && -z "${seen[${host}]+x}" ]] || continue
    seen[${host}]=1
    printf '%s\t%s\n' "${forge}" "${host}"
  done
}

# welcome:repository succeeds inside a git repository.
welcome:repository() {
  git rev-parse --git-dir >/dev/null 2>&1
}

# welcome:scratch succeeds in a linked worktree that stands outside its desk's tree, in a temporary folder: /tmp,
# /var/tmp, macOS's /var/folders, or where TMPDIR names, each as its physical path. It is one a person or an agent made for a task and left
# behind, so it is no checkout of the repository, and a person finds it by its path alone. It fails in the desk, in a
# worktree in the desk's own tree, and in a folder that is gone.
welcome:scratch() {
  local here desk dir
  here=$(pwd -P 2>/dev/null) || return 1
  [[ "$(jq --raw-output .desk <<<"${welcome_where}")" == false ]] || return 1
  desk=$(workspaces:desk "${here}")
  [[ "${here}/" != "${desk}/"* ]] || return 1
  for dir in /tmp /var/tmp /var/folders "${TMPDIR:-}"
  do
    [[ -n "${dir}" ]] && dir=$(welcome:physical "${dir}") && [[ "${dir}" != / && "${here}/" == "${dir}/"* ]] && return 0
  done
  return 1
}

# welcome:installed succeeds where kit.yml declares a kit version.
welcome:installed() {
  [[ "${yaml_ready}" == true && -n "$(settings:version 2>/dev/null)" ]]
}

# welcome:hooks-active succeeds where the hooks/lefthook package is active.
welcome:hooks-active() {
  local -a packages
  # welcome plans the checks it can read: settings it cannot read activate no package.
  list:read packages settings:packages 2>/dev/null || return 1
  list:among hooks/lefthook "${packages[@]}"
}

# welcome:hosts prints "<forge>\t<host>" for each distinct forge host among the remotes of every folder's desk, as
# welcome:forges reads them there, in the folders' order, each host once. A desk's worktrees share its remotes, so
# each desk is read once.
welcome:hosts() {
  local record repository desk folder forge host
  local -A read=() seen=()
  for record in "${welcome_folders[@]}"
  do
    IFS=$'\x1f' read -r repository desk folder <<<"${record}"
    [[ "${desk}" == true && -z "${read[${folder}]+x}" ]] || continue
    read[${folder}]=1
    while IFS=$'\t' read -r forge host
    do
      [[ -z "${seen[${host}]+x}" ]] || continue
      seen[${host}]=1
      printf '%s\t%s\n' "${forge}" "${host}"
    done < <(welcome:forges-in "${folder}")
  done
}

# welcome:forges-in <dir> prints what welcome:forges prints in <dir>, and nothing where kit cannot go there.
welcome:forges-in() (
  cd "$1" 2>/dev/null || exit 0
  welcome:forges
)

# welcome:plan-machine fills welcome_ids with the machine checks, in order, and the forge of each host, and finds the
# system toolchain. It reads the hosts from welcome_folders, which welcome:folders fills first. Then it judges the
# terminal, as welcome:plan-terminal does; with <id>, a check that is not the terminal's, as for a fix of that one, it
# leaves the terminal unjudged.
welcome:plan-machine() {
  local forge host
  welcome_ids=(tool-bash tool-git tool-jq tool-yq) welcome_host_forge=() welcome_terminal=()
  ! welcome:brew-off-path || welcome_ids+=(brew-path)
  welcome_ids+=(toolchain-system)
  welcome:find system
  while IFS=$'\t' read -r forge host
  do
    welcome_ids+=("forge-cli-${host}" "forge-login-${host}")
    welcome_host_forge[${host}]=${forge}
  done < <(welcome:hosts)
  [[ -z "${1:-}" || "$1" == terminal-* ]] || return 0
  welcome:plan-terminal
}

# welcome:plan-terminal judges the terminal once, where KIT_TERMINAL_PROBES says what it answered, into
# welcome_terminal, and adds its checks to welcome_ids; without it, kit runs in no terminal it asked, and the terminal
# has no checks.
welcome:plan-terminal() {
  local check id
  local -a terminal_checks=()
  welcome_terminal=()
  [[ -n "${KIT_TERMINAL_PROBES:-}" ]] || return 0
  # terminal:checks names a bug in terminals.json in the terminal's heading; were it to fail all the same, the
  # terminal has no checks, and the rest of welcome goes on
  list:read terminal_checks terminal:checks || return 0
  for check in "${terminal_checks[@]}"
  do
    id=$(jq --raw-output .id <<<"${check}")
    welcome_terminal[${id}]=${check}
    welcome_ids+=("${id}")
  done
}

# welcome:plan-folder fills welcome_folder_ids with the checks of the folder here, in order, and the rule of each
# linter, and finds the folder's project toolchain.
welcome:plan-folder() {
  local linter rule provider
  welcome_folder_ids=(repo-git kit-installed) welcome_linter_rule=()
  ! welcome:scratch || welcome_folder_ids=(repo-git worktree-scratch kit-installed)
  welcome:find project
  if welcome:installed
  then
    welcome_folder_ids+=(kit-hydrated)
    ! welcome:hooks-active || welcome_folder_ids+=(hooks-installed)
  fi
  welcome_folder_ids+=(toolchain-project)
  ! welcome:github || welcome_folder_ids+=(bot)
  welcome:installed || return 0
  ! welcome:hooks-active || welcome_folder_ids+=(tool-lefthook)
  for provider in $(dependencies:providers)
  do
    [[ ! -f "$(dependencies:call "${provider}" file)" ]] || welcome_folder_ids+=("dependencies-${provider}")
  done
  while IFS=$'\t' read -r linter rule
  do
    welcome_folder_ids+=("linter-${linter##*/}")
    welcome_linter_rule[${linter##*/}]=${rule}
  done < <(welcome:linters)
  welcome_folder_ids+=(kit-check)
}

# welcome:physical <dir> prints the physical path of a directory, and fails where it is not there.
welcome:physical() {
  (cd "$1" 2>/dev/null && pwd -P)
}

# welcome:repository-folders <member> <checkout> prints the folders of the repository a checkout is in, each as
# welcome_folders holds it: its desk, as workspaces:desk finds it, then each worktree git worktree list names that is
# still there. The repository is the member, or, with none, the desk's folder name. A bare desk has no folder of its
# own.
welcome:repository-folders() {
  local member=$1 desk line path first=true
  desk=$(workspaces:desk "$2")
  desk=$(welcome:physical "${desk}") || return 0
  member=${member:-${desk##*/}}
  while IFS= read -r line
  do
    case "${line}" in
      ('worktree '*) path=${line#worktree } ;;
      (bare) path='' ;;
      ('')
        welcome:repository-folder "${member}" "${desk}" "${path:-}" "${first}"
        first=false path=''
        ;;
    esac
  done < <(welcome:worktrees "${desk}")
}

# welcome:worktrees <desk> prints git worktree list --porcelain for the desk, each entry ended by an empty line, and an
# empty line more, so the last entry ends too. A desk git cannot list prints that empty line alone: no folder.
welcome:worktrees() {
  git -C "$1" worktree list --porcelain 2>/dev/null || true  # as above
  printf '\n'
}

# welcome:repository-folder <repository> <desk> <path> <first> prints one folder of git worktree list, as
# welcome:repository-folders reads it, when it is there: a desk where it is the desk, or git's first entry; a worktree
# otherwise.
welcome:repository-folder() {
  local repository=$1 desk=$2 path=$3 first=$4 physical
  [[ -n "${path}" ]] || return 0
  physical=$(welcome:physical "${path}") || return 0
  if [[ "${physical}" == "${desk}" || "${first}" == true ]]
  then
    printf '%s\x1ftrue\x1f%s\n' "${repository}" "${physical}"
  else
    printf '%s\x1ffalse\x1f%s\n' "${repository}" "${physical}"
  fi
}

# welcome:folders fills welcome_folders with the folders of the workspace, as workspaces:current names it, or the
# KIT_WORKSPACE that kit --workspace= sets: a repository, its desk and worktrees, where it has a checkout; a group,
# those of every checkout of each member, in the group's order; none, as for kit flow, no group: the repository here,
# and outside one, no folder. Each folder is there once. It returns what workspaces:current and workspaces:resolve
# return for a workspace they refuse.
welcome:folders() {
  local given=${KIT_WORKSPACE:-} current kind name member checkout resolved record top
  local -a records=()
  local -A seen=()
  welcome_folders=()
  if [[ "${given}" == none ]] && top=$(workspaces:in-repository)
  then
    given=${top}
  fi
  current=$(workspaces:current "${given}") || return $?
  IFS=$'\x1f' read -r kind name member checkout < <(jq --raw-output \
    '[.kind, (.name // ""), (.member // ""), (.checkout // "")] | join("\u001f")' <<<"${current}")
  case "${kind}" in
    (repository)
      [[ -z "${checkout}" ]] || readarray -t records < <(welcome:repository-folders "${member}" "${checkout}")
      ;;
    (group)
      resolved=$(workspaces:resolve "${name}") || return $?
      while IFS=$'\x1f' read -r member checkout
      do
        readarray -t -O "${#records[@]}" records < <(welcome:repository-folders "${member}" "${checkout}")
      done < <(jq --raw-output '.members[] | .repository as $r | .checkouts[] | [$r, .] | join("\u001f")' \
        <<<"${resolved}")
      ;;
  esac
  for record in "${records[@]}"
  do
    [[ -n "${record}" && -z "${seen[${record##*$'\x1f'}]+x}" ]] || continue
    seen[${record##*$'\x1f'}]=1
    welcome_folders+=("${record}")
  done
}

# welcome:folder <path> fills welcome_folders with the one folder whose working tree holds <path>, read from where kit
# was run, as kit welcome --folder= names it: its repository, and whether it is the desk. It returns 2, after a line
# that says so, for a path in no working tree.
welcome:folder() {
  local top desk record
  top=$(workspaces:tree "$1") || return $?
  top=$(welcome:physical "${top}") || return 2
  desk=$(workspaces:desk "${top}")
  welcome_folders=()
  while IFS= read -r record
  do
    [[ "${record##*$'\x1f'}" != "${top}" ]] || welcome_folders=("${record}")
  done < <(welcome:repository-folders "$(workspaces:member-of "${desk}")" "${desk}")
  (( ${#welcome_folders[@]} )) && return 0
  error "git worktree list does not name ${top}, so kit cannot judge it as a folder; run: git worktree prune"
  return 2
}

# welcome:here <record> sets welcome_where for a folder, as welcome_folders holds it: its path, repository, and
# whether it is the desk, with the branch git has checked out there and the kit version its kit.yml declares. Call it
# in the folder.
welcome:here() {
  local repository desk folder branch version=''
  IFS=$'\x1f' read -r repository desk folder <<<"$1"
  branch=$(git branch --show-current 2>/dev/null) || branch=''
  [[ "${yaml_ready}" != true ]] || version=$(settings:version 2>/dev/null) || version=''
  welcome_where=$(jq --null-input --compact-output --arg folder "${folder}" --arg repository "${repository}" \
    --argjson desk "${desk}" --arg branch "${branch}" --arg kit "${version}" '
    def some: if . == "" then null else . end;
    {folder: $folder, repository: $repository, desk: $desk, branch: ($branch | some), kit: ($kit | some)}')
}

# welcome:progress checking|checked [<id>] [<folder>] says on stderr, where welcome_progress asks it to, that a check
# starts, ⏳ checking <id>, with " in <folder>" for a folder's; or that the machine's checks or a folder's end, ⏳
# checked, with " <folder>" for a folder. Each is one short write, so the folders judged at once never mix their lines.
welcome:progress() {
  [[ "${welcome_progress}" == true ]] || return 0
  case "$1" in
    (checking) printf '⏳ checking %s%s\n' "$2" "${3:+ in $3}" >&2 ;;
    (checked) printf '⏳ checked%s\n' "${2:+ $2}" >&2 ;;
  esac
}

# welcome:folder-checks <record> judges the checks of one folder, as welcome_folders holds it, in the folder, and
# prints each as JSON, in order. It runs in a subshell of its own, so nothing it finds there reaches another folder.
welcome:folder-checks() (
  local id
  IFS=$'\x1f' read -r _ _ folder <<<"$1"
  cd "${folder}" || exit 0
  welcome:here "$1"
  welcome:plan-folder
  for id in "${welcome_folder_ids[@]}"
  do
    welcome:progress checking "${id}" "${folder}"
    welcome:judge "${id}"
  done
  welcome:progress checked "${folder}"
)

# welcome:brew-off-path succeeds when brew is installed at a Homebrew prefix but is not on PATH.
welcome:brew-off-path() {
  ! command -v brew >/dev/null 2>&1 && toolchain:call brew prefixed >/dev/null
}

# welcome:brew-path prints the check that brew is on PATH, for brew installed at a prefix and left off it. Its fix is
# the line that puts brew on PATH. A subshell cannot change the person's shell, so kit shows the line and never runs
# it: runs is false.
welcome:brew-path() {
  local brew prefix
  brew=$(toolchain:call brew prefixed) || return 0
  prefix=${brew%/bin/brew}
  welcome:record brew-path false "brew is installed at ${prefix} but not on your PATH" "eval \"\$(${brew} shellenv)\"" \
    false "run it in your shell, and add it to your shell's profile so it holds in every shell" false
}

# welcome:forge-cli <host> prints the check that the host's forge tool, gh or glab, runs.
welcome:forge-cli() {
  local host=$1 tool=gh
  [[ "${welcome_host_forge[${host}]}" == github ]] || tool=glab
  welcome:tool "forge-cli-${host}" "${tool}" "kit reaches ${host} through it"
}

# welcome:forge-login <host> prints the check that the host's forge tool is logged in to it. Logging in needs the
# person, so its fix is interactive.
welcome:forge-login() {
  local host=$1 tool=gh
  [[ "${welcome_host_forge[${host}]}" == github ]] || tool=glab
  if ! welcome:version "${tool}" >/dev/null
  then
    welcome:record "forge-login-${host}" false "${tool} must be installed before it can log in to ${host}" '' false \
      "$(welcome:site "${tool}")"
  elif CLICOLOR_FORCE=0 GH_FORCE_TTY='' "${tool}" auth status --hostname "${host}" >/dev/null 2>&1
  then
    welcome:record "forge-login-${host}" true "${tool} is logged in to ${host}"
  else
    welcome:record "forge-login-${host}" false "${tool} is not logged in to ${host}" \
      "${tool} auth login --hostname ${host}" true
  fi
}

# welcome:kit-check prints the check that kit check finds nothing. It runs kit check and reads each finding, the ❌
# lines above its count. A single finding is the check's line; more than one are the check's findings, under a line
# that counts them. The fix is the command after "run:" when every finding names the same one, and each finding is
# then named without it. It runs only when it is a kit command. A finding without a command does not get a fix.
welcome:kit-check() {
  local out line command fix='' runs=false
  local -a findings=() commands=() named
  if out=$(NO_COLOR=1 run kit check 2>&1)
  then
    welcome:record kit-check true "kit check finds nothing wrong"
    return 0
  fi
  # the counts kit check and kit generate --check print are not findings of their own
  readarray -t findings < <(sed -n 's/^❌ //p' <<<"${out}" \
    | grep --invert-match --extended-regexp '^[0-9]+ finding\(s\), each named above|^generate: [0-9]+ of [0-9]+ ')
  for line in "${findings[@]}"
  do
    command=''
    if [[ "${line}" == *'run: '* ]]
    then
      command=${line##*run: }
      # the check's fix runs in its folder, which the check names; the cd kit check wrote in front is not the fix's
      [[ "${command}" != 'cd '*' && '* ]] || command=${command#* && }
      command=${command%% (*}
      command=${command%%;*}
      command=${command%%,*}
    fi
    commands+=("${command}")
  done
  if (( ${#findings[@]} > 0 )) && [[ "$(printf '%s\n' "${commands[@]}" | sort --unique | wc -l)" -eq 1 ]]
  then
    fix=${commands[0]}
    [[ "${fix}" != 'kit '* ]] || runs=true
  fi
  case "${#findings[@]}" in
    (0) line="kit check fails; run kit check to see why" ;;
    (1) line=${findings[0]} ;;
    (*) line="kit check finds ${#findings[@]} problems"
  esac
  if (( ${#findings[@]} < 2 ))
  then
    welcome:record kit-check false "${line}" "${fix}" false '' "${runs}"
    return 0
  fi
  named=("${findings[@]}")
  [[ -z "${fix}" ]] || named=("${findings[@]%; run: *}")
  welcome:record kit-check false "${line}" "${fix}" false '' "${runs}" \
    | jq --compact-output '. + {findings: $ARGS.positional}' --args "${named[@]}"
}

# welcome:terminal <id> prints the terminal check <id>, as welcome:plan-machine judged it: its fix is the person's to
# make in their terminal's settings, so kit shows it and never runs it.
welcome:terminal() {
  local ok line fix more
  IFS=$'\x1f' read -r ok line fix more < <(jq --raw-output \
    '[(.ok | tostring), .line, (.fix // ""), (.more // "")] | join("\u001f")' <<<"${welcome_terminal[$1]}")
  welcome:record "$1" "${ok}" "${line}" "${fix}" false "${more}" false
}

# welcome:folded <check-json> succeeds for a terminal's finding or probe that holds and is not kept: the text guide,
# as kittui, shows it under its terminal's heading by leaving it out. With welcome_fold false, as for the one check
# kit welcome --keep= says, nothing is folded.
declare -g welcome_fold=true
welcome:folded() {
  [[ "${welcome_fold}" == true ]] || return 1
  jq --exit-status '(.id | test("^terminal-[a-z]+-")) and .ok and (.line | startswith("kept: ") | not)' <<<"$1" \
    >/dev/null
}

# welcome:keep keep|unkeep <check> keeps the terminal's finding or probe <check>, so it holds from now on, or undoes
# the keep, then prints the check as the terminal is now judged, where KIT_TERMINAL_PROBES says what it answered. It
# returns 2 for any other check: one of another kind, or a terminal's heading, which always holds.
welcome:keep() {
  local action=$1 id=$2
  if ! jq --exit-status --arg id "${id}" '
      [(.probes | keys[] | "terminal-probe-\(.)"),
       (.terminals | to_entries[] | .key as $key | .value.findings[]? | "terminal-\($key)-\(.id)")]
      | index($id) != null' "${kit_jq}/terminals.json" >/dev/null
  then
    error "${id} is not a finding of a terminal, which is all kit welcome --${action}= takes; see: kit welcome --checks"
    return 2
  fi
  "terminal:${action}" "${id}" || return $?
  welcome:plan-terminal
  [[ -z "${welcome_terminal[${id}]+x}" ]] || welcome:terminal "${id}"
}

# welcome:worktree-scratch prints the check that the worktree here is no scratch worktree: it fails for one in a
# temporary folder, outside its desk, whose fix removes it as git does, never one with changes, and holds for one that
# is gone. A path with a space in it, which a fix cannot carry, names the command in more instead of the fix.
welcome:worktree-scratch() {
  local here desk
  if ! welcome:scratch
  then
    welcome:record worktree-scratch true "this worktree is no scratch worktree"
    return 0
  fi
  here=$(pwd -P)
  desk=$(workspaces:desk "${here}")
  if [[ "${here}${desk}" == *[[:space:]]* ]]
  then
    welcome:record worktree-scratch false \
      "this worktree is in a temporary folder, outside ${desk}: a scratch worktree, left behind" '' false \
      "remove it from ${desk}: git worktree remove, then its path"
  else
    welcome:record worktree-scratch false \
      "this worktree is in a temporary folder, outside ${desk}: a scratch worktree, left behind" \
      "git -C ${desk} worktree remove ${here}"
  fi
}

# welcome:github succeeds where the primary remote's forge is github.
welcome:github() {
  local remote
  remote=$(forge:primary 2>/dev/null) || return 1
  [[ "$(forge:of "${remote}" 2>/dev/null)" == github ]]
}

# welcome:bot-add <name> <app> prints the command that adds a bot's key.
welcome:bot-add() { printf 'kit bots --add=%s --app=%s --key=<file>\n' "$1" "$2"; }

# welcome:bot-reason <file> prints the first line of what a command said on stderr, less its marker, or <fallback>.
welcome:bot-reason() {
  local line=''
  IFS= read -r line < "$1" || true  # an empty file has no line
  line=${line#❌ }
  printf '%s\n' "${line:-$2}"
}

# welcome:bot prints the check of the bot that acts for the person in this folder. No bot on the primary remote's
# host is an offer to add one. It fails for a key the secret store lacks, for an installation that does not reach this
# repository, for a job that lacks a permission, and for two bots with none chosen or a forge that cannot be asked:
# each says why, and names the fix. The forge is asked once, for the App, its installation, and its permissions.
welcome:bot() {
  local errors bot name app check status=0 jobs lacks pem
  local more='https://docs.github.com/en/apps/creating-github-apps'
  file:temp errors welcome-bot-errors || return 1
  bot=$(bots:choose 2> "${errors}") || status=$?
  case "${status}" in
    (0) ;;
    (2)
      case "$(cat "${errors}")" in
        (*[0-9]' bots are on '*)
          welcome:record bot false "$(welcome:bot-reason "${errors}" 'two bots are on this host')" \
            'kit bots --check'
          ;;
        (*'no bot is named '*)
          name=$(settings:value bot)
          welcome:record bot false "kit.yml names the bot ${name}, which the bots file does not have" \
            "kit bots --add=${name} --app=<id> --key=<file>" false '' false
          ;;
        (*)
          welcome:record bot true 'no bot acts for you here; set one up to hand it issues' \
            'kit bots --add=<name> --app=<id> --key=<file>' false "${more}" false
      esac
      rm -f "${errors}"
      return 0
      ;;
    (*)
      welcome:record bot false "$(welcome:bot-reason "${errors}" 'the bots file cannot be read')" 'kit bots --check'
      rm -f "${errors}"
      return 0
  esac
  name=$(jq --raw-output .name <<<"${bot}")
  app=$(jq --raw-output .app <<<"${bot}")
  file:temp pem welcome-bot-key || return 1
  if ! bots:key "${bot}" "${pem}" 2> /dev/null
  then
    rm -f "${pem}" "${errors}"
    welcome:record bot false "the key of ${name} is not in the secret store" "$(welcome:bot-add "${name}" "${app}")" \
      false '' false
    return 0
  fi
  rm -f "${pem}"
  check=$(bots:check "${bot}" 2> "${errors}") || status=$?
  if (( status != 0 ))
  then
    welcome:record bot false "$(welcome:bot-reason "${errors}" "the forge could not be asked about ${name}")" \
      'kit bots --check'
  elif [[ "$(jq --raw-output .reached <<<"${check}")" == false ]]
  then
    welcome:record bot false \
      "the installation of ${name} does not reach $(jq --raw-output .repository <<<"${check}")" 'kit bots --check'
  elif lacks=$(jq --raw-output '.lacks | to_entries | map("\(.key): \(.value | join(", "))") | join("; ")' \
    <<<"${check}") && [[ -n "${lacks}" ]]
  then
    welcome:record bot false "${name} lacks permissions its jobs need; the ${lacks}" 'kit bots --check'
  else
    jobs=$(jq --raw-output '.jobs | join(", ")' <<<"${bot}")
    welcome:record bot true "${name}[bot] acts for you here, for ${jobs}"
  fi
  rm -f "${errors}"
}

# welcome:judge <id> runs one check, which welcome:plan-machine or, in its folder, welcome:plan-folder named, and
# prints it as JSON. It returns 2 for an id neither plan names.
welcome:judge() {
  local id=$1
  list:among "${id}" "${welcome_ids[@]}" "${welcome_folder_ids[@]}" || return 2
  case "${id}" in
    (tool-bash) welcome:bash ;;
    (tool-git) welcome:tool tool-git git "kit works through it" ;;
    (tool-jq) welcome:jq ;;
    (tool-yq) welcome:yq ;;
    (brew-path) welcome:brew-path ;;
    (toolchain-project) welcome:toolchain project ;;
    (toolchain-system) welcome:toolchain system ;;
    (tool-lefthook) welcome:tool tool-lefthook lefthook "it runs the git hooks" ;;
    (dependencies-*) welcome:dependencies "${id#dependencies-}" ;;
    (linter-*) welcome:linter "${id}" "${id#linter-}" "${welcome_linter_rule[${id#linter-}]}" ;;
    (forge-cli-*) welcome:forge-cli "${id#forge-cli-}" ;;
    (forge-login-*) welcome:forge-login "${id#forge-login-}" ;;
    (terminal-*) welcome:terminal "${id}" ;;
    (bot) welcome:bot ;;
    (repo-git)
      if welcome:repository
      then
        welcome:record repo-git true "this folder is a git repository"
      else
        welcome:record repo-git false "this folder is not a git repository; kit works inside one" '' false \
          'https://git-scm.com/book/en/v2/Git-Basics-Getting-a-Git-Repository'
      fi
      ;;
    (worktree-scratch) welcome:worktree-scratch ;;
    (kit-installed)
      if [[ "${yaml_ready}" != true ]]
      then
        welcome:record kit-installed false "kit cannot read kit.yml without yq; fix tool-yq first" '' false \
          "$(welcome:site yq)"
      elif welcome:installed
      then
        welcome:record kit-installed true "kit.yml declares kit $(settings:version)"
      else
        # kit runs here all the same, on what it infers, so not installing it is a choice: kit offers the install
        welcome:record kit-installed true "kit is not installed here; it runs on what it infers" 'kit install'
      fi
      ;;
    (kit-hydrated)
      if materialize:hydrated
      then
        welcome:record kit-hydrated true ".kit/ is in place: every file kit placed there is present"
      else
        welcome:record kit-hydrated false \
          ".kit/ is not in place: a file kit placed there is missing or changed, or a package's file is newer" \
          'kit hydrate'
      fi
      ;;
    (hooks-installed)
      if grep --quiet lefthook "$(git rev-parse --git-path hooks/pre-commit)" 2>/dev/null
      then
        welcome:record hooks-installed true "the git hooks run lefthook before each commit"
      else
        welcome:record hooks-installed false "the git hooks are not installed; lefthook runs none" 'lefthook install'
      fi
      ;;
    (kit-check) welcome:kit-check ;;
  esac
}

# welcome:checks [--folders] runs every check, in order, and prints one JSON object per line: the machine checks, then
# each folder's of welcome_folders, which welcome:folders or welcome:folder fills first. With --folders, the folders'
# alone. The folders are judged at once, as many at a time as there are processors, each into a file of its own, and
# printed in their order.
welcome:checks() {
  local id dir n=0 i most
  if [[ "${1:-}" != --folders ]]
  then
    welcome:plan-machine
    for id in "${welcome_ids[@]}"
    do
      welcome:progress checking "${id}"
      welcome:judge "${id}"
    done
    welcome:progress checked
  else
    welcome:find system
  fi
  (( ${#welcome_folders[@]} )) || return 0
  file:scratch dir
  most=$(getconf _NPROCESSORS_ONLN 2>/dev/null) || most=4
  [[ "${most}" =~ ^[1-9][0-9]*$ ]] || most=4
  for (( n = 0; n < ${#welcome_folders[@]}; n++ ))
  do
    while (( $(jobs -r -p | wc -l) >= most ))
    do
      wait -n || true  # a folder that fails prints what it judged; its file is read below
    done
    welcome:folder-checks "${welcome_folders[n]}" > "${dir}/${n}" &
  done
  wait
  for (( i = 0; i < n; i++ ))
  do
    cat "${dir}/${i}"
  done
}

# welcome:show-all <check-json>… prints the checks as welcome:show does, with a line before each folder's, the first
# time one comes: its repository, the desk or a worktree, its branch, the kit its kit.yml declares, and its path.
welcome:show-all() {
  local check folder last='' head
  for check in "$@"
  do
    folder=$(jq --raw-output '.folder // ""' <<<"${check}")
    if [[ -n "${folder}" && "${folder}" != "${last}" ]]
    then
      head=$(jq --raw-output '"\(.repository), " + (if .desk then "the desk" else "a worktree" end)
        + (if .branch then " on \(.branch)" else "" end) + ", " + (if .kit then "kit \(.kit)" else "no kit" end)
        + ": \(.folder)"' <<<"${check}")
      printf '%s%s%s\n' "$(out:sgr 1)" "${head}" "$(out:sgr 0)"
    fi
    last=${folder}
    welcome:show "${check}"
  done
}

# welcome:show <check-json>… prints each check as text: ✅, 💡 for an offer, or ❌, and its line; under an offer, dim, its
# fix; and under a failed one, each of its findings, then, dim, its fix and what more says beside it, or, with no fix,
# where to read more.
welcome:show() {
  local check ok line fix more finding
  for check in "$@"
  do
    ! welcome:folded "${check}" || continue
    IFS=$'\x1f' read -r ok line fix more < <(jq --raw-output \
      '[(.ok | tostring), .line, (.fix // ""), (.more // "")] | join("\u001f")' <<<"${check}")
    fix=$(welcome:fix-line "${check}" "${fix}")
    if [[ "${ok}" == true && -n "${fix}" ]]
    then
      printf '💡 %s\n   %s%s%s\n' "${line}" "$(out:sgr 90)" "${fix}" "$(out:sgr 0)"
      continue
    elif [[ "${ok}" == true ]]
    then
      printf '✅ %s\n' "${line}"
      continue
    fi
    printf '❌ %s\n' "${line}"
    while IFS= read -r finding
    do
      printf '   %s\n' "${finding}"
    done < <(jq --raw-output '.findings // [] | .[]' <<<"${check}")
    if [[ -n "${fix}" ]]
    then
      printf '   %s%s%s\n' "$(out:sgr 90)" "${fix}" "$(out:sgr 0)"
      [[ -z "${more}" ]] || printf '   %s%s%s\n' "$(out:sgr 90)" "${more}" "$(out:sgr 0)"
    elif [[ -n "${more}" ]]
    then
      printf '   %sread: %s%s\n' "$(out:sgr 90)" "${more}" "$(out:sgr 0)"
    fi
  done
}

# welcome:fix-line <check-json> <fix> prints a check's fix as kit shows it: a folder's with cd and its folder in front,
# as out:remedy writes a remedy, so it runs from anywhere; a machine's as it is; nothing for none.
welcome:fix-line() {
  local folder
  [[ -n "$2" ]] || return 0
  folder=$(jq --raw-output '.folder // ""' <<<"$1")
  if [[ -n "${folder}" ]]
  then
    printf 'cd %s && %s\n' "$(out:folder "${folder}")" "$2"
  else
    printf '%s\n' "$2"
  fi
}

# welcome:left <check-json>… prints what is left: each failed check's id, a folder's with its folder, and its fix, as
# welcome:fix-line shows it, or where to read more. A fix keeps what more says beside it, as the toolchains check's
# other choices. With nothing left, it says so.
welcome:left() {
  local rows='' check fix
  for check in "$@"
  do
    [[ "$(jq '.ok | not' <<<"${check}")" == true ]] || continue
    fix=$(welcome:fix-line "${check}" "$(jq --raw-output '.fix // ""' <<<"${check}")")
    rows+=$(jq --raw-output --arg fix "${fix}" '"- \(.id)" + (if .folder then " in \(.folder)" else "" end) + ": "
      + (if $fix != "" then $fix + (if .more then "; " + .more else "" end)
        + (if .runs == false then "; kit does not run it" else "" end)
      else "read " + (.more // "the line above") end)' <<<"${check}")$'\n'
  done
  rows=${rows%$'\n'}
  printf '# What is left\n\n'
  if [[ -z "${rows}" ]]
  then
    printf 'Nothing is left: every check holds.\n'
    return 0
  fi
  printf '%s\n\nRun a command yourself, or have kit run one and check again: kit welcome --fix=<id>, and for a %s\n' \
    "${rows}" "folder's check, --folder=<path>."
}

# welcome:pages prints the pages of kit welcome, one path per line: the files under .kit/welcome/, as the active
# packages placed them, else the kit package's own, where nothing is placed yet. They are in the order of the number
# a name starts with, so 10-ours.md comes after 9-theirs.md; a name with no number comes last, and a tie goes by name.
welcome:pages() {
  local dir=.kit/welcome
  compgen -G "${dir}/*.md" >/dev/null || dir=${kit_root}/packages/kit/files/repo/.kit/welcome
  compgen -G "${dir}/*.md" \
    | awk -F/ '{ n = $NF; k = match(n, /^[0-9]+/) ? substr(n, 1, RLENGTH) + 0 : 1e9; printf "%d\t%s\n", k, $0 }' \
    | LC_ALL=C sort -t $'\t' -k1,1n -k2,2 | cut -f2-
}

# welcome:page-step <path> prints the id of the step a page is for: its file name without the order and the dash
# before it, and without .md. 2-toolchains.md is for toolchains.
welcome:page-step() {
  local name=${1##*/}
  name=${name%.md}
  printf '%s\n' "${name#*-}"
}

# welcome:page <path> prints a page without the kit:generated and see: comments it opens with, each link drawn as
# text, markdown:unlink's.
welcome:page() {
  awk 'started || !/^<!--.*-->$/ { started = 1; print }' "$1" | markdown:unlink
}

# welcome:fix <id> [<folder>] runs the fix of one check, with its output on the terminal, then judges the check again
# and prints it as text. A machine check takes no folder. A folder's check runs in the folder whose working tree holds
# <folder>, as kit welcome --folder= names it, else in the repository here: its fix runs there, and only that folder is
# judged. An offer's fix runs as a failed check's does. It returns 0 when the check now holds with nothing left to
# offer, 1 when it does not, and 2 for an id that names no check, a machine check given a folder, a folder's check with
# no folder here or given, a check that holds already with no offer, one kit knows no fix for, or one whose fix kit
# only shows (runs is false).
welcome:fix() {
  local id=$1 folder=${2:-} top
  if welcome:machine "${id}"
  then
    if [[ -n "${folder}" ]]
    then
      error "${id} is the machine's check, not a folder's, so it takes no --folder=; run: kit welcome --fix=${id}"
      return 2
    fi
    # Only a forge's checks read the folders, for their hosts.
    [[ "${id}" != forge-* ]] || welcome:folders || return $?
    welcome:plan-machine "${id}"
    if ! list:among "${id}" "${welcome_ids[@]}"
    then
      error "no check is named '${id}'; the checks are: ${welcome_ids[*]}"
      return 2
    fi
    welcome:fix-one "${id}"
    return
  fi
  if [[ -z "${folder}" ]] && ! folder=$(workspaces:in-repository)
  then
    error "${id} is a folder's check, and kit runs outside a repository; name the folder:" \
      "kit welcome --fix=${id} --folder=<path>"
    return 2
  fi
  welcome:folder "${folder}" || return $?
  welcome:plan-machine "${id}"
  top=${welcome_folders[0]##*$'\x1f'}
  (
    cd "${top}" || exit 2
    welcome:here "${welcome_folders[0]}"
    welcome:plan-folder
    if ! list:among "${id}" "${welcome_folder_ids[@]}"
    then
      error "no check is named '${id}'; the checks are: ${welcome_ids[*]}, and in ${top}: ${welcome_folder_ids[*]}"
      exit 2
    fi
    welcome:fix-one "${id}"
  )
}

# welcome:fix-one <id> runs the fix of one check, which the plans here name, and judges it again, as welcome:fix says.
welcome:fix-one() {
  local id=$1 check fix more
  local -a command
  check=$(welcome:judge "${id}")
  if jq --exit-status '.ok and .fix == null' <<<"${check}" >/dev/null
  then
    error "${id} holds already, so there is nothing to fix: $(jq --raw-output .line <<<"${check}")"
    return 2
  fi
  fix=$(jq --raw-output '.fix // ""' <<<"${check}")
  if [[ -z "${fix}" ]]
  then
    more=$(jq --raw-output '.more // ""' <<<"${check}")
    error "kit knows no fix for ${id}${more:+; read: ${more}}"
    return 2
  fi
  if [[ "$(jq --raw-output .runs <<<"${check}")" == false ]]
  then
    error "${id}'s fix must run in your own shell, so kit does not run it; run it there: ${fix}"
    return 2
  fi
  step "${fix}"
  read -r -a command <<<"${fix}"
  # A fix that fails shows in the check judged again below, which then fails.
  if [[ "${command[0]}" == kit ]]
  then
    run "${command[@]}" || true  # as above
  else
    "${command[@]}" || true  # as above
  fi
  # A fix can choose a toolchain, so the toolchains are found again: a folder's project toolchain, in its folder.
  (( ${#welcome_folder_ids[@]} == 0 )) || welcome:find project
  welcome:find system
  check=$(welcome:judge "${id}")
  welcome:show "${check}"
  # an offer still made, as an install that failed, is a fix that did not take
  jq --exit-status '.ok and .fix == null' <<<"${check}" >/dev/null || return 1
}

# ----------------------------------------------------------------------------------------------------------------------
