# shellcheck shell=bash
# toolchain.sh - the seam over the tool managers that install the gate's tools (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# kit finds the toolchains where it runs, never in kit.yml. Each tool manager is a provider: toolchain/<name>.sh
# beside it, whose functions are named toolchain/<name>:<call>.
#
# A toolchain fills one role or both:
#
#   - project: it pins the repository's tools
#   - system: it supplies what kit itself stands on, such as bash, git, jq, and yq
#
# toolchain:detect asks every provider about each role it can fill:
#
#   - usable: its tool is installed and on PATH, and its file is on disk here, committed or not
#   - active: the run is already inside it, and its file is here
#
# A probe stats files and runs a tool for its version. It never touches the network, and never calls git.
# The answers are cached per checkout in .kit/cache/toolchains.tsv. The cache holds while its key, in
# .kit/cache/toolchains.key, still matches: PATH, where each tool resolves, each file's modification time, and
# each activation marker.
#
# toolchain:chosen gives the one toolchain for a role, or ambient for none. brew is a project candidate only where no
# other is usable; otherwise it fills in, for the tools the chosen one does not pin (toolchain:filler). toolchain:choose
# records a choice:
#
#   - system: toolchain: <name> in ${KIT_HOME:-$HOME}/.kit/settings.yml
#   - project, at the desk: <git common dir>/kit/toolchain, shared with every worktree of the clone
#   - project, in a linked worktree: <its git dir>/kit/toolchain, kept in that worktree alone
#
# Every provider implements the calls below, except where noted.
#
#   active                  succeeds when the run is inside the toolchain, and its file is here
#   file                    the file it reads, relative to the repository root
#   inputs                  what its answers depend on, one per line: file, env, or tool, a tab, then the name
#   install                 installs every pinned tool, from the file and its lock
#   outdated                one line per tool that is behind its source: "<name>\t<installed>\t<latest>"; when it
#                           cannot read a source, it says which, prints the rest, and fails
#   path                    the directories to put first on PATH for its tools, one per line
#   pins                    one line per pinned unit, "<name>\t<pinned>\t<installed>"; never reaches the network
#   require                 succeeds when its own tool is present; otherwise fails with a remedy
#   remedy <tool>           the command that installs one tool; a system provider may implement it, and kit
#                           prints it for the person, never running it on its own
#   roles                   the roles it can fill, one per line: project, system, or both
#   update                  moves every tool to its latest version, and rewrites the pins and the lock
#   usable <role>           succeeds when its tool is installed and on PATH, and it is configured for the role
#   run <command>…          runs the command with its tools; only a provider that isolates, as docker, implements it,
#                           and toolchain:run takes it only while the provider is not active
#
# see: docs/toolchains.md

use file

# toolchain:providers prints the providers on the module path, space-separated.
toolchain:providers() {
  use:modules 'toolchain/*' | sed 's|^toolchain/||' | paste -s -d ' ' -
}

# toolchain:detect prints one line per provider and role it can fill: "<provider>\t<role>\t<usable>\t<active>",
# each flag yes or no, providers in toolchain:providers order. It answers from the cache while the key matches.
# Otherwise it probes, and rewrites the cache where .kit/.gitignore lists /cache/, so git never sees it.
toolchain:detect() {
  local key found provider role usable active
  local -a roles
  key=$(toolchain:key) || return 1
  if [[ -f .kit/cache/toolchains.tsv && "$(cat .kit/cache/toolchains.key 2>/dev/null)" == "${key}" ]]
  then
    cat .kit/cache/toolchains.tsv
    return 0
  fi
  found=$(
    for provider in $(toolchain:providers)
    do
      active=no
      toolchain:call "${provider}" active && active=yes
      list:read roles toolchain:call "${provider}" roles || exit 1
      for role in "${roles[@]}"
      do
        usable=no
        toolchain:call "${provider}" usable "${role}" && usable=yes
        printf '%s\t%s\t%s\t%s\n' "${provider}" "${role}" "${usable}" "${active}"
      done
    done
  ) || return 1
  printf '%s\n' "${found}"
  grep --quiet --line-regexp /cache/ .kit/.gitignore 2>/dev/null || return 0
  # The key goes first and comes back last, so a cache half written never matches. Where the cache cannot be
  # written, detection still answers, and probes again next time.
  { rm -f .kit/cache/toolchains.key \
    && file:replace .kit/cache/toolchains.tsv printf '%s\n' "${found}" \
    && file:replace .kit/cache/toolchains.key printf '%s\n' "${key}"; } 2>/dev/null || true
}

# toolchain:key prints the cache key: a digest of PATH, and of every provider's inputs. A tool counts by where it
# resolves on PATH, a file by its modification time or absent, and a variable by its value.
toolchain:key() {
  local provider kind name row
  local -a inputs
  {
    printf 'PATH\t%s\n' "${PATH}"
    for provider in $(toolchain:providers)
    do
      list:read inputs toolchain:call "${provider}" inputs || exit 1
      for row in "${inputs[@]}"
      do
        IFS=$'\t' read -r kind name <<<"${row}"
        case "${kind}" in
          (tool) printf 'tool\t%s\t%s\n' "${name}" "$(command -v "${name}" 2>/dev/null)" ;;
          (env) printf 'env\t%s\t%s\n' "${name}" "${!name:-}" ;;
          (file) printf 'file\t%s\t%s\n' "${name}" "$(toolchain:modified "${name}")" ;;
          (*) ;;
        esac
      done
    done
  } | file:digest
}

# toolchain:modified <path> prints the path's modification time, to the fraction of a second where stat gives it,
# or absent. GNU stat takes -c; BSD stat refuses it and takes -f.
toolchain:modified() {
  [[ -e "$1" || -L "$1" ]] || { printf 'absent\n'; return 0; }
  stat -c %y -- "$1" 2>/dev/null || stat -f %Fm -- "$1"
}

# toolchain:call <provider> <call> <argument>… runs the provider's function for the call.
toolchain:call() {
  local provider=$1 call=$2
  shift 2
  use "toolchain/${provider}" || return 1
  "toolchain/${provider}:${call}" "$@"
}

# toolchain:remedy <provider> <tool> prints the provider's command that installs the tool. It fails for ambient, and
# for a provider that implements no remedy or names no command.
toolchain:remedy() {
  local provider=$1 fix
  [[ -n "${provider}" && "${provider}" != ambient ]] || return 1
  use "toolchain/${provider}" 2>/dev/null && declare -F "toolchain/${provider}:remedy" >/dev/null || return 1
  fix=$(toolchain:call "${provider}" remedy "$2") && [[ -n "${fix}" ]] || return 1
  printf '%s\n' "${fix}"
}

# toolchain:system-only <provider> answers a project call for a provider that fills the system role alone, as apt.
# kit never makes one; it refuses, so a call made by mistake says why.
toolchain:system-only() {
  error "$1 fills the system role only: it installs for the whole machine, and pins no repository's tools"
  return 1
}

# toolchain:sudo prints "sudo " for a remedy that installs for the whole machine, and nothing when kit already runs as
# root, as in a container that may have no sudo.
toolchain:sudo() {
  (( EUID == 0 )) || printf 'sudo '
}

# toolchain:usable <role> prints the providers usable for the role, one per line, in precedence order: mise, nix,
# brew, docker, then any other, as apt or devcontainer, in toolchain:providers order. The order is how candidates are
# offered; a run uses one.
toolchain:usable() {
  toolchain:detect | toolchain:ranked "$1" usable
}

# toolchain:ranked <role> (usable | active) reads toolchain:detect's lines on stdin. It prints the providers that
# are usable for the role, or usable and active, one per line, in toolchain:usable's precedence order. For the
# project, brew is left out where another is usable, since it fills in for that one instead.
toolchain:ranked() {
  awk -F '\t' -v role="$1" -v active="$2" '
    $2 == role && $3 == "yes" && (active != "active" || $4 == "yes") {
      rank = ($1 == "mise") ? 1 : ($1 == "nix") ? 2 : ($1 == "brew") ? 3 : ($1 == "docker") ? 4 : 5
      found[$1] = rank "\t" NR
      if ($1 != "brew") others++
    }
    END {
      for (name in found)
        if (!(role == "project" && name == "brew" && others)) print found[name] "\t" name
    }' | sort -n -k 1,1 -k 2,2 | cut -f 3
}

# toolchain:filler <chosen> prints brew where it fills in for the chosen project toolchain, and nothing otherwise.
# brew fills in where it is usable for the project and another is chosen that runs commands on this machine. docker
# and devcontainer run them in a container, which brew's tools never reach.
toolchain:filler() {
  local chosen=$1
  [[ -n "${chosen}" && "${chosen}" != ambient && "${chosen}" != brew ]] || return 0
  use "toolchain/${chosen}" || return 1
  ! declare -F "toolchain/${chosen}:run" >/dev/null || return 0
  toolchain:detect | awk -F '\t' '$1 == "brew" && $2 == "project" && $3 == "yes" { print "brew" }'
}

# toolchain:pinned-names <provider> prints the name of each tool the provider pins, one per line, as the filler
# compares them: without a backend, as npm: or aqua:, a path, or an @version. brew filling in reads them from
# toolchain_pinned.
toolchain:pinned-names() {
  toolchain:call "$1" pins | cut -f 1 | awk '{ sub(/^.*:/, ""); sub(/^.*\//, ""); sub(/@.*$/, ""); print }'
}

# toolchain:chosen <role> prints the toolchain for the role, project or system, or ambient for none. The first
# that answers is the one:
#
#   1. a toolchain active here; naming another for the run is refused
#   2. for the project, the one named for the run, by --toolchain or KIT_TOOLCHAIN; one not usable here is
#      refused, never replaced
#   3. the one recorded earlier, while it is still usable here
#   4. the only usable one; none is ambient
#   5. on a terminal, the person's answer, which is recorded
#
# Otherwise, with several usable and no terminal to ask, it refuses. The refusal lists each candidate with the
# command that picks it. Every refusal returns 1.
#
# With list as the second word, it only reports, for kit toolchains to draw. It never asks, never refuses, and
# never records. Where nothing is chosen, as with several usable, it prints nothing and returns 0.
toolchain:chosen() {
  local role=$1 mode=${2:-} named='' found active recorded choice answer
  local -a usable=()
  toolchain:role "${role}" || return 1
  # The run names one toolchain, the project's: CI names its own, and the system one is found or chosen.
  [[ "${role}" != project ]] || named=${KIT_TOOLCHAIN:-}
  found=$(toolchain:detect) || return 1
  active=$(toolchain:ranked "${role}" active <<<"${found}" | head -n 1)
  list:read usable toolchain:ranked "${role}" usable <<<"${found}" || return 1
  if [[ -n "${active}" ]]
  then
    if [[ -n "${named}" && "${named}" != "${active}" && "${mode}" != list ]]
    then
      error "${active} is active here, so this run cannot use ${named}; leave ${active}'s shell, or name ${active}"
      return 1
    fi
    printf '%s\n' "${active}"
    return 0
  fi
  if [[ -n "${named}" ]]
  then
    if [[ "${named}" == ambient ]] || toolchain:among "${named}" "${usable[@]}"
    then
      printf '%s\n' "${named}"
      return 0
    fi
    [[ "${mode}" != list ]] || return 0
    error "the toolchain named for this run, ${named}, is not usable for the ${role} here;" \
      "usable: ${usable[*]:-none, so name ambient}"
    return 1
  fi
  # A worktree's own choice comes first, then the clone's; each holds only while it is usable here.
  recorded=$(toolchain:recorded "${role}") || return 1
  while IFS= read -r choice
  do
    if [[ "${choice}" == ambient ]] || { [[ -n "${choice}" ]] && toolchain:among "${choice}" "${usable[@]}"; }
    then
      printf '%s\n' "${choice}"
      return 0
    fi
  done <<<"${recorded}"
  case "${#usable[@]}" in
    (0) printf 'ambient\n'; return 0 ;;
    (1) printf '%s\n' "${usable[0]}"; return 0 ;;
    (*) ;;
  esac
  [[ "${mode}" != list ]] || return 0
  if [[ -t 0 && -t 2 ]]
  then
    answer=$(toolchain:ask "${role}" "${usable[@]}") || return 1
    toolchain:choose "${role}" "${answer}" || return 1
    info "kit keeps ${answer} for the ${role}; change it with: kit toolchains use <name>$(toolchain:flag "${role}")"
    printf '%s\n' "${answer}"
    return 0
  fi
  toolchain:refuse "${role}" "${usable[@]}"
  return 1
}

# toolchain:ask <role> <candidate>… asks on the terminal which candidate to use for the role, and prints it. It
# asks again after an answer it cannot read, and fails when the input ends.
toolchain:ask() {
  local role=$1 answer i
  shift
  {
    case "${role}" in
      (system) printf 'Several toolchains can supply what kit runs on: bash, git, jq, and yq.\n' ;;
      (*) printf 'Several toolchains can install the tools this repository pins.\n' ;;
    esac
    printf 'Which one should kit use?\n'
    for (( i = 1; i <= $#; i++ ))
    do
      printf '  %d) %s\n' "${i}" "${!i}"
    done
  } >&2
  while true
  do
    printf 'Type a number from 1 to %d, then Enter: ' "$#" >&2
    if ! IFS= read -r answer
    then
      printf '\n' >&2
      error "no toolchain was chosen"
      return 1
    fi
    if [[ "${answer}" =~ ^[0-9]+$ ]] && (( 10#${answer} >= 1 && 10#${answer} <= $# ))
    then
      printf '%s\n' "${@:10#${answer}:1}"
      return 0
    fi
    if toolchain:among "${answer}" "$@"
    then
      printf '%s\n' "${answer}"
      return 0
    fi
  done
}

# toolchain:refuse <role> <candidate>… says that several toolchains are usable for the role and none can be asked
# for, and lists each candidate with the command that picks it.
toolchain:refuse() {
  local role=$1 candidate
  shift
  error "several toolchains are usable for the ${role} here, and there is no terminal to ask which; choose one:"
  for candidate in "$@"
  do
    printf '    with %s: kit toolchains use %s%s\n' "${candidate}" "${candidate}" "$(toolchain:flag "${role}")" >&2
  done
  # The run names only the project toolchain; the system one is chosen and kept.
  [[ "${role}" != project ]] || printf '  or name one for a single run: KIT_TOOLCHAIN=<name>\n' >&2
}

# toolchain:choose <role> <provider> records the provider as the role's choice, or ambient for none. It fails,
# naming the usable ones, when the provider is not usable for the role here.
#
#   - system: toolchain: <provider> in ${KIT_HOME:-$HOME}/.kit/settings.yml, written with yq
#   - project, at the desk: <git common dir>/kit/toolchain, which every worktree of the clone reads
#   - project, in a linked worktree: <its git dir>/kit/toolchain, which only that worktree reads
toolchain:choose() {
  local role=$1 provider=$2 settings found kept
  local -a usable=()
  toolchain:role "${role}" || return 1
  found=$(toolchain:detect) || return 1
  list:read usable toolchain:ranked "${role}" usable <<<"${found}" || return 1
  if [[ "${role}" == project && "${provider}" == brew && ${#usable[@]} -gt 0 ]] \
    && grep --quiet --line-regexp $'brew\tproject\tyes\t.*' <<<"${found}"
  then
    error "brew fills in for the tools ${usable[*]// / or } does not pin; choose ${usable[*]// / or }"
    return 1
  fi
  if [[ "${provider}" != ambient ]] && ! toolchain:among "${provider}" "${usable[@]}"
  then
    error "${provider} is not usable for the ${role} here; usable: ${usable[*]:-none, so choose ambient}"
    return 1
  fi
  case "${role}" in
    (system)
      settings="$(toolchain:settings)"
      if [[ -f "${settings}" ]]
      then
        kit_toolchain=${provider} file:replace "${settings}" yq '.toolchain = strenv(kit_toolchain)' "${settings}"
      else
        kit_toolchain=${provider} file:replace "${settings}" yq --null-input '.toolchain = strenv(kit_toolchain)'
      fi
      ;;
    (project)
      kept=$(toolchain:kept) || return 1
      file:replace "${kept}" printf '%s\n' "${provider}"
      ;;
  esac
}

# toolchain:recorded <role> prints the choices recorded for the role, one per line, or nothing. A linked worktree
# prints its own choice first, then the clone's; toolchain:chosen takes the first that is usable. Outside a git
# repository, nothing is recorded for the project.
toolchain:recorded() {
  local role=$1 kept common
  toolchain:role "${role}" || return 1
  if [[ "${role}" == system ]]
  then
    use yaml || return 1
    yaml:value "$(toolchain:settings)" toolchain
    return
  fi
  common=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || return 0
  kept=$(toolchain:kept) || return 1
  [[ "${kept}" == "${common}/kit/toolchain" || ! -s "${kept}" ]] || head -n 1 "${kept}"
  [[ ! -s "${common}/kit/toolchain" ]] || head -n 1 "${common}/kit/toolchain"
}

# toolchain:kept prints the file that keeps the project choice made here: kit/toolchain in this checkout's git
# directory. At the desk, that is the common one. A linked worktree has its own, .git/worktrees/<name>/ in the
# clone, so its choice is never in the working tree, and git worktree remove takes it away.
toolchain:kept() {
  local dir
  dir=$(git rev-parse --path-format=absolute --git-dir) || return 1
  printf '%s/kit/toolchain\n' "${dir}"
}

# toolchain:settings prints the file that keeps the system choice: ~/.kit/settings.yml, under KIT_HOME when set.
toolchain:settings() {
  printf '%s/.kit/settings.yml\n' "${KIT_HOME:-${HOME}}"
}

# toolchain:role <role> succeeds for project and system, and otherwise fails, saying so.
toolchain:role() {
  [[ "$1" == project || "$1" == system ]] && return 0
  error "no such toolchain role: $1; the roles are project and system"
  return 1
}

# toolchain:flag <role> prints what kit toolchains use adds for the role: --system for system, else nothing.
toolchain:flag() {
  [[ "$1" != system ]] || printf ' --system'
}

# toolchain:among <name> <candidate>… succeeds when the name is one of the candidates.
toolchain:among() {
  local name=$1 candidate
  shift
  for candidate in "$@"
  do
    [[ "${candidate}" == "${name}" ]] && return 0
  done
  return 1
}

# toolchain:path prints the directories to put first on PATH, one per line: the chosen project toolchain's, then its
# filler's, then each usable dependency provider's, as node_modules/.bin. Ambient, only the providers'.
toolchain:path() {
  local chosen filler dirs=''
  chosen=$(toolchain:chosen project) || return 1
  [[ "${chosen}" == ambient ]] || dirs=$(toolchain:call "${chosen}" path) || return 1
  filler=$(toolchain:filler "${chosen}") || return 1
  [[ -z "${filler}" ]] || dirs+=${dirs:+$'\n'}$(toolchain:call "${filler}" path) || return 1
  [[ -z "${dirs}" ]] || printf '%s\n' "${dirs}"
  use dependencies || return 1
  dependencies:path "$(paste -s -d ':' - <<<"${dirs}")"
}

# toolchain:run <command>… runs the command with the chosen project toolchain's tools:
#
#   - through the provider's run, where it has one and is not active, as docker does to run in its container; a
#     provider that is active is already inside its container, so the command runs there as is
#   - otherwise, with toolchain:path first on PATH: the provider's directories, then the dependencies'
#   - ambient, with the dependencies' directories first
toolchain:run() {
  local chosen dirs
  chosen=$(toolchain:chosen project) || return 1
  if [[ "${chosen}" != ambient ]]
  then
    use "toolchain/${chosen}" || return 1
    if declare -F "toolchain/${chosen}:run" >/dev/null && ! toolchain:call "${chosen}" active
    then
      toolchain:call "${chosen}" run "$@"
      return
    fi
  fi
  dirs=$(toolchain:path | paste -s -d ':' -) || return 1
  PATH="${dirs:+${dirs}:}${PATH}" "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
