# shellcheck shell=bash
# welcome.sh - the checks of kit welcome's setup step, their fixes, and the pages of the steps after it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# kit welcome walks a newcomer through four steps: setup, the lay of the land, the terms, and making it yours. Setup
# is a list of checks, run here. Each check is one JSON object:
#
#   id            stable, in kebab-case: tool-yq, linter-shellcheck, forge-login-github.com
#   step          the step it belongs to: setup
#   ok            true when it holds
#   line          one short sentence: what holds, or what is wrong
#   fix           the command that fixes it, or null when it holds or kit knows none
#   interactive   true when the fix needs the person at the terminal, as gh auth login does
#   more          where to read more when kit knows no fix, or null
#
# The checks run in this order:
#
#   - tools: bash, git, jq, yq; lefthook when hooks/lefthook is active; then each linter the active packages' rules
#     and styles name under lint:
#   - forges: for each distinct forge host among the remotes, its tool installed, then logged in to that host
#   - this repository: a git repository; then, where it is one, kit installed; then, where kit is, .kit/ hydrated,
#     the git hooks installed when hooks/lefthook is active, and kit check clean
#   - where you are: at the desk or in a worktree; it always holds
#
# kit detects the repository's toolchain, as kit toolchains does, and never assumes one. A missing tool's fix is
# kit toolchains install where a toolchain is detected. Where none is, the tools are ambient: the fix is brew install
# on macOS with Homebrew, else none, and more names the tool's site. Only welcome:fix runs a fix, and only the one it
# is asked for.
#
# The pages of steps 2 to 4 are markdown files under .kit/welcome/, which the kit package delivers. A house package
# replaces one by placing a file of the same name, or adds one; the later package wins.
#
# see: bin/kit-welcome, issue #51

# The yq check says what yaml.sh says when it loads without yq, so its own line is not printed twice.
use yaml 2>/dev/null
use settings forge materialize render place toolchain

# welcome_ids holds the checks' ids, in order, and the maps beside it what judging one needs: the rule that names
# each linter, and the forge of each host. welcome:plan fills them.
declare -ga welcome_ids=()
declare -gA welcome_linter_rule=() welcome_host_forge=()

# welcome:record <id> <ok> <line> [<fix> [<interactive> [<more>]]] prints one check as JSON, on one line. An empty
# fix or more is null.
welcome:record() {
  jq --null-input --compact-output --arg id "$1" --argjson ok "$2" --arg line "$3" --arg fix "${4:-}" \
    --argjson interactive "${5:-false}" --arg more "${6:-}" \
    '{id: $id, step: "setup", ok: $ok, line: $line, fix: ($fix | if . == "" then null else . end),
      interactive: $interactive, more: ($more | if . == "" then null else . end)}'
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

# welcome:toolchains prints the toolchains kit detects here, one per line, as kit toolchains lists them. Nothing
# means the tools are ambient: whatever is on PATH.
welcome:toolchains() {
  toolchain:declared 2>/dev/null || true
}

# welcome:toolchain-files prints the files the detected toolchains read, comma-separated: where a tool is pinned.
welcome:toolchain-files() {
  local provider files=''
  for provider in $(welcome:toolchains)
  do
    files+="${files:+, }$(toolchain:call "${provider}" file 2>/dev/null)"
  done
  printf '%s\n' "${files}"
}

# welcome:toolchain-has <tool> succeeds when a detected toolchain has installed the tool: it is in a directory the
# toolchains put on PATH.
welcome:toolchain-has() {
  local dir
  while IFS= read -r dir
  do
    [[ -n "${dir}" && -x "${dir}/$1" ]] && return 0
  done < <(toolchain:path 2>/dev/null)
  return 1
}

# welcome:brew succeeds on macOS with brew on PATH, where brew install is the fix kit offers.
welcome:brew() {
  [[ "$(uname -s)" == Darwin ]] && command -v brew >/dev/null 2>&1
}

# welcome:install <tool> prints the fix that installs the tool and where to read more, separated by \x1f:
#
#   - kit toolchains install where kit detects a toolchain, and the files that pin the tools
#   - else, with the tools ambient, brew install <tool> on macOS with Homebrew
#   - else no fix, and the tool's site
welcome:install() {
  if [[ -n "$(welcome:toolchains)" ]]
  then
    printf 'kit toolchains install\x1f%s\n' "$(welcome:toolchain-files)"
  elif welcome:brew
  then
    printf 'brew install %s\x1f\n' "$1"
  else
    printf '\x1f%s\n' "$(welcome:site "$1")"
  fi
}

# welcome:missing <id> <tool> <line> [<more>] prints the check of a tool that does not run: its install fix, and where
# to read more, <more> when kit knows no site for it. A tool the toolchain installed, in a directory not on PATH,
# needs its directory on PATH rather than another install.
welcome:missing() {
  local id=$1 tool=$2 line=$3 fix more
  if welcome:toolchain-has "${tool}"
  then
    welcome:record "${id}" false "${tool} is installed, but not on PATH; kit toolchains path names its directory" '' \
      false 'https://github.com/empowerite/kit/blob/main/docs/toolchains.md'
    return 0
  fi
  IFS=$'\x1f' read -r fix more < <(welcome:install "${tool}")
  welcome:record "${id}" false "${line}" "${fix}" false "${more:-${4:-}}"
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
  readarray -t packages < <(settings:packages)
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
  local -A seen=()
  while IFS= read -r remote
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
  done < <(forge:remotes 2>/dev/null)
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
  readarray -t packages < <(settings:packages 2>/dev/null)
  list:among hooks/lefthook "${packages[@]}"
}

# welcome:plan fills welcome_ids with the checks to run, in order, and the maps that judging them needs.
welcome:plan() {
  local linter rule forge host
  welcome_ids=(tool-bash tool-git tool-jq tool-yq) welcome_linter_rule=() welcome_host_forge=()
  if welcome:repository && welcome:installed
  then
    ! welcome:hooks-active || welcome_ids+=(tool-lefthook)
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

# welcome:kit-check prints the check that kit check finds nothing: it runs kit check and counts its findings. Its fix
# is kit update, which brings every delivered file back to the release.
welcome:kit-check() {
  local out count
  if out=$(NO_COLOR=1 run kit check 2>&1)
  then
    welcome:record kit-check true "kit check finds nothing wrong"
  else
    count=$(grep --count '^❌' <<<"${out}" || true)
    (( count > 0 )) || count=1
    welcome:record kit-check false "kit check finds ${count} problem(s); run kit check to see them" 'kit update'
  fi
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
  branch=$(git branch --show-current 2>/dev/null || true)
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
    (tool-lefthook) welcome:tool tool-lefthook lefthook "it runs the git hooks" ;;
    (linter-*)
      welcome:tool "${id}" "${id#linter-}" "kit lint runs it for ${welcome_linter_rule[${id#linter-}]}" \
        "${welcome_linter_rule[${id#linter-}]}"
      ;;
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
        welcome:record kit-hydrated false ".kit/ is not in place: a file kit placed there is missing or changed" \
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

# welcome:show <check-json>… prints each check as text: ✅ or ❌ and its line, and under a failed one, dim, its fix,
# or where to read more.
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
    elif [[ -n "${more}" ]]
    then
      printf '   %sread: %s%s\n' "$(out:sgr 90)" "${more}" "$(out:sgr 0)"
    fi
  done
}

# welcome:left <check-json>… prints what is left: each failed check's id, and its fix or where to read more. With
# nothing left, it says so.
welcome:left() {
  local rows
  rows=$(printf '%s\n' "$@" \
    | jq --raw-output 'select(.ok | not) | "- \(.id): \(.fix // ("read " + (.more // "the line above")))"')
  printf '## What is left\n\n'
  if [[ -z "${rows}" ]]
  then
    printf 'Nothing is left: every check holds.\n'
    return 0
  fi
  printf '%s\n\nRun a command yourself, or have kit run one and check again: kit welcome --fix <id>.\n' "${rows}"
}

# welcome:pages prints the pages of kit welcome, one path per line, in order: the files under .kit/welcome/, as the
# active packages placed them, else the kit package's own, where nothing is placed yet.
welcome:pages() {
  local dir=.kit/welcome
  compgen -G "${dir}/*.md" >/dev/null || dir=${kit_root}/packages/kit/files/repo/.kit/welcome
  compgen -G "${dir}/*.md" | LC_ALL=C sort
}

# welcome:page <path> prints a page without the kit:generated and see: comments it opens with.
welcome:page() {
  awk 'started || !/^<!--.*-->$/ { started = 1; print }' "$1"
}

# welcome:fix <id> runs the fix of one check, with its output on the terminal, then judges the check again and prints
# it as text. It returns 0 when the check now holds, 1 when it does not, and 2 for an id that names no check, a
# check that holds already, or one kit knows no fix for.
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
  step "${fix}"
  read -r -a command <<<"${fix}"
  if [[ "${command[0]}" == kit ]]
  then
    run "${command[@]}" || true
  else
    "${command[@]}" || true
  fi
  check=$(welcome:judge "${id}")
  welcome:show "${check}"
  [[ "$(jq --raw-output .ok <<<"${check}")" == true ]] || return 1
}

# ----------------------------------------------------------------------------------------------------------------------
