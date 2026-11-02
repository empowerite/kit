# shellcheck shell=bash
# toolchain.sh - the seam over the tool managers (mise, brew, docker) that install the gate's tools (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
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
# Every provider implements the calls below, except where noted.
#
#   active                  succeeds when the run is inside the toolchain, and its file is here
#   file                    the file it reads, relative to the repository root
#   inputs                  what its answers depend on, one per line: file, env, or tool, a tab, then the name
#   install                 installs every pinned tool, from the file and its lock
#   outdated                one line per tool that is behind its source: "<name>\t<installed>\t<latest>"
#   path                    the directories to put first on PATH for its tools, one per line
#   require                 succeeds when its own tool is present; otherwise fails with a remedy
#   roles                   the roles it can fill, one per line: project, system, or both
#   update                  moves every tool to its latest version, and rewrites the pins and the lock
#   usable <role>           succeeds when its tool is installed and on PATH, and it is configured for the role
#   run <command>…          runs the command with its tools; only docker implements it, because it isolates
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
      while IFS= read -r role
      do
        usable=no
        toolchain:call "${provider}" usable "${role}" && usable=yes
        printf '%s\t%s\t%s\t%s\n' "${provider}" "${role}" "${usable}" "${active}"
      done < <(toolchain:call "${provider}" roles)
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
  local provider kind name
  {
    printf 'PATH\t%s\n' "${PATH}"
    for provider in $(toolchain:providers)
    do
      while IFS=$'\t' read -r kind name
      do
        case "${kind}" in
          (tool) printf 'tool\t%s\t%s\n' "${name}" "$(command -v "${name}" 2>/dev/null)" ;;
          (env) printf 'env\t%s\t%s\n' "${name}" "${!name:-}" ;;
          (file) printf 'file\t%s\t%s\n' "${name}" "$(toolchain:modified "${name}")" ;;
          (*) ;;
        esac
      done < <(toolchain:call "${provider}" inputs)
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

# toolchain:path prints the directories that every toolchain usable for the project puts first on PATH, in order,
# one per line.
toolchain:path() {
  local provider
  for provider in $(toolchain:detect | awk -F '\t' '$2 == "project" && $3 == "yes" { print $1 }')
  do
    toolchain:call "${provider}" path || return 1
  done
}

# toolchain:run <command>… runs the command with the project's tools: inside the container when docker is usable
# for the project, otherwise with every usable toolchain's path first on PATH. With none, the command runs as is.
toolchain:run() {
  local dirs
  if toolchain:detect | awk -F '\t' '$1 == "docker" && $2 == "project" && $3 == "yes" { found = 1 } END { exit !found }'
  then
    toolchain:call docker run "$@"
    return
  fi
  dirs=$(toolchain:path | paste -s -d ':' -) || return 1
  PATH="${dirs:+${dirs}:}${PATH}" "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
