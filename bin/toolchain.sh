# shellcheck shell=bash
# toolchain.sh - one interface to the tool managers (mise, brew, docker) that install the gate's tools (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# Each repository picks the tool managers that install the tools its gate runs, on the root checkout (the desk)
# and on a CI runner. Each tool manager is a provider: toolchain-<name>.sh in this directory, whose functions are
# named toolchain-<name>:<call>. A provider reads the fields of its settings record with toolchain:field.
#
# A repository can use more than one provider. They are chosen by:
#
#   1. the `toolchains` list in .kit/settings.yml, in the order listed
#   2. otherwise every provider whose file is at the repository root, in this order:
#        - mise: mise.toml or .mise.toml
#        - brew: Brewfile
#        - docker: toolchain.Dockerfile
#
# A `toolchains` setting looks like this:
#
#   toolchains:
#     - provider: mise
#     - provider: brew
#       file: Brewfile
#     - provider: docker
#       file: toolchain.Dockerfile
#       image: example/toolchain
#
# Every provider implements the calls below, except where noted.
#
#   file                    the file it reads, relative to the repository root
#   require                 succeeds when its own tool is present; otherwise fails with a remedy
#   install                 installs every pinned tool, from the file and its lock
#   outdated                one line per tool that is behind its source: "<name>\t<installed>\t<latest>"
#   update                  moves every tool to its latest version, and rewrites the pins and the lock
#   path                    the directories to put first on PATH for its tools, one per line
#   run <command>…          runs the command with its tools; only docker implements it, because it isolates
#
# toolchain:run runs a command inside the container when docker is declared, and otherwise with every
# toolchain's path first on PATH. Either way the gate sees the same tools on the desk and on a runner.
#
# see: docs/toolchains.md

use release yaml

# toolchain:providers prints the providers this release carries, space-separated.
toolchain:providers() {
  # shellcheck disable=SC2154  # kit_bin_dir is declared by bin/1st.sh
  (cd "${kit_bin_dir}" && find . -maxdepth 1 -name 'toolchain-*.sh' | sed 's@^\./toolchain-@@; s@\.sh$@@' \
    | LC_ALL=C sort | paste -s -d ' ' -)
}

# toolchain:declared prints the chosen providers, one per line, as the file header describes. It fails, naming
# the provider, when a declared provider is not one the kit has.
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

# toolchain:field <provider> <field> prints a field of the provider's record, or nothing. The while loop reads
# every record, even after a match, so yaml:records never writes into a pipe that nobody reads.
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

# toolchain:isolating prints docker when docker is declared, otherwise nothing. docker is the only provider that
# runs a command in its own container.
toolchain:isolating() {
  local providers
  providers=$(toolchain:declared) || return 1
  grep --line-regexp docker <<<"${providers}" || true
}

# toolchain:run <command>… runs the command with the declared toolchains' tools: inside the container when docker
# is declared, otherwise with every toolchain's path first on PATH. With no toolchain, the command runs as is.
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
