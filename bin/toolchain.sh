# shellcheck shell=bash
# toolchain.sh - the seam over how a checkout gets the tools its gate runs: mise, brew, or docker (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Which tool manager puts a repository's tools on a desk or a runner is the repository's decision. A repository
# declares its toolchains in .kit/settings.yml, or the kit detects them from the files each one reads:
#
#   toolchains:
#     - provider: mise                       detected from mise.toml or .mise.toml
#     - provider: brew                       detected from Brewfile
#       file: Brewfile
#     - provider: docker                     detected from toolchain.Dockerfile
#       file: toolchain.Dockerfile
#       image: example/toolchain
#
# A provider is toolchain-<name>.sh beside this file, its functions named toolchain-<name>:<call>, each reading its
# record's fields through toolchain:field. A provider answers:
#
#   file                    the file it reads, relative to the repository root
#   require                 its own tool is present, or a remedy
#   install                 every pinned tool installed, from the file and its lock
#   outdated                one line per tool behind its source: "<name>\t<installed>\t<latest>"
#   update                  every tool moved to its latest, and the pins and the lock rewritten
#   path                    the directories that put its tools first on PATH, one per line
#   run <command>…          the command run with its tools; only a provider that isolates, docker, answers
#
# A command runs with every toolchain's path first on PATH, or inside the container where docker is declared, so
# the gate sees the same tools on a desk and on a runner.
#
# see: docs/toolchains.md

use release yaml

# toolchain:providers prints the providers this release carries, space-separated.
toolchain:providers() {
  # shellcheck disable=SC2154  # kit_bin_dir is declared by bin/1st.sh
  (cd "${kit_bin_dir}" && find . -maxdepth 1 -name 'toolchain-*.sh' | sed 's@^\./toolchain-@@; s@\.sh$@@' \
    | LC_ALL=C sort | paste -s -d ' ' -)
}

# toolchain:declared prints the declared providers, one per line, in the order the settings name them; with none
# declared, the ones whose file is here, in the order mise, brew, docker. A provider the kit does not carry is
# refused by name.
toolchain:declared() {
  local provider found=''
  local -a declared=()
  while IFS=$'\x1f' read -r provider
  do
    [[ -n "${provider}" ]] && declared+=("${provider}")
  done < <(yaml:records "$(release:settings)" toolchains provider)
  if (( ${#declared[@]} > 0 ))
  then
    for provider in "${declared[@]}"
    do
      if [[ ! -f "${kit_bin_dir}/toolchain-${provider}.sh" ]]
      then
        error "'${provider}' is not a toolchain the kit knows; the toolchains are: $(toolchain:providers)" \
          "(toolchains in .kit/settings.yml)"
        return 1
      fi
    done
    printf '%s\n' "${declared[@]}"
    return 0
  fi
  [[ -f mise.toml || -f .mise.toml ]] && found+=$'mise\n'
  [[ -f Brewfile ]] && found+=$'brew\n'
  [[ -f toolchain.Dockerfile ]] && found+=$'docker\n'
  printf '%s' "${found}"
}

# toolchain:field <provider> <field> prints a field of the provider's record, or nothing. The loop reads to the
# end, so the reader is never left writing into a pipe nobody reads.
toolchain:field() {
  local provider value found=''
  while IFS=$'\x1f' read -r provider value
  do
    [[ -z "${found}" && "${provider}" == "$1" ]] && found=${value}
  done < <(yaml:records "$(release:settings)" toolchains provider "$2")
  [[ -z "${found}" ]] || printf '%s\n' "${found}"
}

# toolchain:call <provider> <call> <argument>… runs the provider's function for the call.
toolchain:call() {
  local provider=$1 call=$2
  shift 2
  use "toolchain-${provider}" || return 1
  "toolchain-${provider}:${call}" "$@"
}

# toolchain:each <call> runs the call on every declared toolchain in order, stopping at the first that fails.
toolchain:each() {
  local provider providers
  providers=$(toolchain:declared) || return 1
  for provider in ${providers}
  do
    toolchain:call "${provider}" "$@" || return 1
  done
}

# toolchain:path prints the directories every declared toolchain puts first on PATH, in order, one per line.
toolchain:path() {
  toolchain:each path
}

# toolchain:isolating prints the declared toolchain that runs a command in a place of its own, docker, or nothing.
toolchain:isolating() {
  local providers
  providers=$(toolchain:declared) || return 1
  grep --line-regexp docker <<<"${providers}" || true
}

# toolchain:run <command>… runs the command with the declared toolchains' tools: inside the container where docker
# is declared, else with every toolchain's path first on PATH. With none declared the command runs as it is.
toolchain:run() {
  local isolating dirs
  isolating=$(toolchain:isolating) || return 1
  if [[ -n "${isolating}" ]]
  then
    toolchain:call "${isolating}" run "$@"
    return
  fi
  dirs=$(toolchain:path | paste -s -d ':' -) || return 1
  PATH="${dirs:+${dirs}:}${PATH}" "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
