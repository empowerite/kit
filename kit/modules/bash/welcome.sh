# shellcheck shell=bash
# welcome.sh - kit welcome's checks, their fixes, and the pages of its steps (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# kit welcome walks a newcomer through five steps, each a page: the introduction, toolchains, setup, the terms, and
# customize. Steps 2 and 3 hold checks, run here. Each check is one JSON object:
#
#   id            stable, in kebab-case: tool-yq, toolchain-project, linter-shellcheck, forge-login-github.com
#   step          the step it belongs to: toolchains or setup
#   ok            true when it holds
#   line          one short sentence: what holds, or what is wrong
#   fix           the command that fixes it, or null when it holds or kit knows none
#   interactive   true when the fix needs the person at the terminal, as gh auth login does
#   more          where to read more, or the other commands beside the fix, or null
#   runs          false when kit only shows the fix and never runs it, as for a line that changes the person's own
#                 shell; true for every other check. kit welcome --fix refuses a check whose runs is false.
#
# The checks run in this order:
#
#   - toolchains: bash, git, jq, and yq, which kit runs on; then, only where brew is installed at a Homebrew prefix but
#     is not on PATH, brew-path, whose fix kit shows and never runs; then one check per role, project then system
#   - setup, tools: lefthook when hooks/lefthook is active; then each dependency provider whose manifest is here,
#     installed from its lock; then each linter the active packages' rules and styles name under lint:
#   - setup, forges: for each distinct forge host among the remotes, its tool installed, then logged in to that host
#   - setup, this repository: a git repository; then, where it is one, kit installed; then, where kit is, .kit/
#     hydrated, the git hooks installed when hooks/lefthook is active, and kit check clean
#   - setup, where you are: at the desk or in a worktree; it always holds
#
# kit finds the toolchains as kit toolchains does, and never assumes one. The checks only report: they never ask
# which toolchain to use, and never record one. A missing tool's fix:
#
#   - toolchains, for a tool kit runs on: the system toolchain's command, such as its install of the tool, else none,
#     and more names the tool's site
#   - setup: kit tools install where a project toolchain is settled; ambient, the system toolchain's command; else
#     none, and more says where to read
#
# Only welcome:fix runs a fix, and only the one it is asked for.
#
# The pages are markdown files under .kit/welcome/, which the kit package delivers. A page's name is its order, a
# dash, then its step's id: 2-toolchains.md. A house package replaces one by placing a file of the same name, or adds
# one; the later package wins.
#
# see: bin/kit-welcome, modules/bash/toolchain.sh, issue #51

# The yq check says what yaml.sh says when it loads without yq, so its own line is not printed twice.
use yaml 2>/dev/null
use settings forge materialize render place toolchain dependencies

# welcome_ids holds the checks' ids, in order, and the maps beside it what judging one needs: the rule that names
# each linter, and the forge of each host. welcome:plan fills them.
declare -ga welcome_ids=()
declare -gA welcome_linter_rule=() welcome_host_forge=()

# welcome:record <id> <ok> <line> [<fix> [<interactive> [<more> [<runs>]]]] prints one check as JSON, on one line, in
# the step its id belongs to. An empty fix or more is null. runs is true unless the check says kit only shows its fix.
welcome:record() {
  jq --null-input --compact-output --arg id "$1" --arg step "$(welcome:step "$1")" --argjson ok "$2" --arg line "$3" \
    --arg fix "${4:-}" --argjson interactive "${5:-false}" --arg more "${6:-}" --argjson runs "${7:-true}" \
    '{id: $id, step: $step, ok: $ok, line: $line, fix: ($fix | if . == "" then null else . end),
      interactive: $interactive, more: ($more | if . == "" then null else . end), runs: $runs}'
}

# welcome:step <id> prints the step a check belongs to: toolchains for what kit runs on and each role's toolchain,
# and setup for the rest.
welcome:step() {
  case "$1" in
    (tool-bash | tool-git | tool-jq | tool-yq | brew-path | toolchain-*) printf 'toolchains\n' ;;
    (*) printf 'setup\n' ;;
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

# welcome:install <step> <tool> prints the fix that installs the tool and where to read more, separated by \x1f:
#
#   - toolchains: the system toolchain's command, else no fix, and the tool's site
#   - setup: kit tools install where a project toolchain is settled; ambient, the system toolchain's command, else
#     no fix and the tool's site; with several usable and none chosen, no fix, and the toolchains doc
welcome:install() {
  local step=$1 tool=$2 project fix
  project=$(welcome:chosen project)
  if [[ "${step}" == setup && -n "${project}" && "${project}" != ambient ]]
  then
    printf 'kit tools install\x1f\n'
  elif [[ "${step}" == setup && -z "${project}" ]]
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
# to read more, <more> when kit knows no site for it. A tool the project toolchain installed, in a directory not on
# PATH, needs its directory on PATH rather than another install.
welcome:missing() {
  local id=$1 tool=$2 line=$3 step fix more
  step=$(welcome:step "${id}")
  if [[ "${step}" == setup ]] && welcome:toolchain-has "${tool}"
  then
    welcome:record "${id}" false "${tool} is installed, but not on PATH; kit tools path names its directory" '' \
      false "${welcome_toolchains_doc}"
    return 0
  fi
  IFS=$'\x1f' read -r fix more < <(welcome:install "${step}" "${tool}")
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
        [[ "${candidate}" == "${usable[0]}" ]] || others+="${others:+; }kit toolchains choose ${candidate}${flag}"
      done
      welcome:record "toolchain-${role}" false "several toolchains are usable for the ${role}: ${names}; choose one" \
        "kit toolchains choose ${usable[0]}${flag}" false "or: ${others}"
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

# welcome:plan fills welcome_ids with the checks to run, in order, and the maps that judging them needs, the
# toolchains among them.
welcome:plan() {
  local linter rule forge host provider
  welcome_ids=(tool-bash tool-git tool-jq tool-yq) welcome_linter_rule=() welcome_host_forge=()
  ! welcome:brew-off-path || welcome_ids+=(brew-path)
  welcome_ids+=(toolchain-project toolchain-system)
  welcome:find project
  welcome:find system
  if welcome:repository && welcome:installed
  then
    ! welcome:hooks-active || welcome_ids+=(tool-lefthook)
    for provider in $(dependencies:providers)
    do
      [[ ! -f "$(dependencies:call "${provider}" file)" ]] || welcome_ids+=("dependencies-${provider}")
    done
    while IFS=$'\t' read -r linter rule
    do
      welcome_ids+=("linter-${linter##*/}")
      welcome_linter_rule[${linter##*/}]=${rule}
    done < <(welcome:linters)
  fi
  if welcome:repository
  then
    while IFS=$'\t' read -r forge host
    do
      welcome_ids+=("forge-cli-${host}" "forge-login-${host}")
      welcome_host_forge[${host}]=${forge}
    done < <(welcome:forges)
  fi
  welcome_ids+=(repo-git)
  if welcome:repository
  then
    welcome_ids+=(kit-installed)
    if welcome:installed
    then
      welcome_ids+=(kit-hydrated)
      ! welcome:hooks-active || welcome_ids+=(hooks-installed)
      welcome_ids+=(kit-check)
    fi
  fi
  welcome_ids+=(where)
}

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
# lines above its count. A single finding is the check's line. The fix is the command after "run:" when every finding
# names the same one. It runs only when it is a kit command. A finding without a command does not get a fix.
welcome:kit-check() {
  local out line command fix='' runs=false
  local -a findings=() commands=()
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
    (*) line="kit check finds ${#findings[@]} problem(s); run kit check to see each one"
  esac
  welcome:record kit-check false "${line}" "${fix}" false '' "${runs}"
}

# welcome:where prints where this shell is: at the desk, in a worktree, or outside any repository. It always holds;
# its line says what the place is for.
welcome:where() {
  local dirs branch
  if ! welcome:repository
  then
    welcome:record where true "you are outside any repository; kit works inside one"
    return 0
  fi
  dirs=$(git rev-parse --path-format=absolute --git-dir --git-common-dir)
  branch=$(git branch --show-current 2>/dev/null || true)  # the line names no branch where git gives none
  if [[ "${dirs%%$'\n'*}" == "${dirs#*$'\n'}" ]]
  then
    welcome:record where true "you are at the desk, your own checkout; work on an issue goes in a worktree"
  else
    welcome:record where true "you are in a worktree${branch:+ on ${branch}}; one issue's work happens here"
  fi
}

# welcome:judge <id> runs one check, which welcome:plan named, and prints it as JSON. It returns 2 for an id the plan
# does not name.
welcome:judge() {
  local id=$1
  list:among "${id}" "${welcome_ids[@]}" || return 2
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
    (repo-git)
      if welcome:repository
      then
        welcome:record repo-git true "this is a git repository"
      else
        welcome:record repo-git false "this is not a git repository; run kit welcome inside one" '' false \
          'https://git-scm.com/book/en/v2/Git-Basics-Getting-a-Git-Repository'
      fi
      ;;
    (kit-installed)
      if [[ "${yaml_ready}" != true ]]
      then
        welcome:record kit-installed false "kit cannot read kit.yml without yq; fix tool-yq first" '' false \
          "$(welcome:site yq)"
      elif welcome:installed
      then
        welcome:record kit-installed true "kit.yml declares kit $(settings:version)"
      else
        welcome:record kit-installed false "kit.yml declares no kit version; kit is not installed here" \
          'kit install'
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
    (where) welcome:where ;;
  esac
}

# welcome:checks runs every check, in order, and prints one JSON object per line.
welcome:checks() {
  local id
  welcome:plan
  for id in "${welcome_ids[@]}"
  do
    welcome:judge "${id}"
  done
}

# welcome:show <check-json>… prints each check as text: ✅ or ❌ and its line, and under a failed one, dim, its fix and
# what more says beside it, or, with no fix, where to read more.
welcome:show() {
  local check ok line fix more
  for check in "$@"
  do
    IFS=$'\x1f' read -r ok line fix more < <(jq --raw-output \
      '[(.ok | tostring), .line, (.fix // ""), (.more // "")] | join("\u001f")' <<<"${check}")
    if [[ "${ok}" == true ]]
    then
      printf '✅ %s\n' "${line}"
      continue
    fi
    printf '❌ %s\n' "${line}"
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

# welcome:left <check-json>… prints what is left: each failed check's id, and its fix or where to read more. A fix
# keeps what more says beside it, as the toolchains check's other choices. With nothing left, it says so.
welcome:left() {
  local rows
  rows=$(printf '%s\n' "$@" | jq --raw-output 'select(.ok | not)
    | "- \(.id): " + (if .fix then .fix + (if .more then "; " + .more else "" end)
        + (if .runs == false then "; kit does not run it" else "" end)
      else "read " + (.more // "the line above") end)')
  printf '# What is left\n\n'
  if [[ -z "${rows}" ]]
  then
    printf 'Nothing is left: every check holds.\n'
    return 0
  fi
  printf '%s\n\nRun a command yourself, or have kit run one and check again: kit welcome --fix <id>.\n' "${rows}"
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

# welcome:page <path> prints a page without the kit:generated and see: comments it opens with.
welcome:page() {
  awk 'started || !/^<!--.*-->$/ { started = 1; print }' "$1"
}

# welcome:fix <id> runs the fix of one check, with its output on the terminal, then judges the check again and prints
# it as text. It returns 0 when the check now holds, 1 when it does not, and 2 for an id that names no check, a
# check that holds already, one kit knows no fix for, or one whose fix kit only shows (runs is false).
welcome:fix() {
  local id=$1 check fix more
  local -a command
  welcome:plan
  if ! list:among "${id}" "${welcome_ids[@]}"
  then
    error "no check is named '${id}'; the checks are: ${welcome_ids[*]}"
    return 2
  fi
  check=$(welcome:judge "${id}")
  if [[ "$(jq --raw-output .ok <<<"${check}")" == true ]]
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
  # A fix can choose a toolchain, so the toolchains are found again.
  welcome:find project
  welcome:find system
  check=$(welcome:judge "${id}")
  welcome:show "${check}"
  [[ "$(jq --raw-output .ok <<<"${check}")" == true ]] || return 1
}

# ----------------------------------------------------------------------------------------------------------------------
