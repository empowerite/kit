# shellcheck shell=bash
# devcontainer.sh - the dev container toolchain provider: devcontainer.json builds the environment (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# devcontainer fills the project role, and the system role too while the run is inside the container. The container
# is configured by .devcontainer/devcontainer.json, or .devcontainer.json, at the repository root.
#
# From outside, it is usable with the devcontainer CLI on PATH and the file here. It runs a command in the container
# instead of putting tools on PATH, as docker does. Inside, the file alone makes it usable. It is active there, when
# REMOTE_CONTAINERS, CODESPACES, or DEVCONTAINER is set and the file is here: a marker left over from another project
# does not count. Inside, a run needs no container to enter, so toolchain:run never uses its run.
#
# The file is JSON with comments, which neither jq nor yq reads, so kit never parses it. pins reads the lock beside
# it, devcontainer-lock.json (or .devcontainer-lock.json for .devcontainer.json). A feature not yet locked is not
# listed. kit tools install brings the container up, and kit tools run needs it up.

toolchain/devcontainer:active() {
  [[ -n "${REMOTE_CONTAINERS:-}${CODESPACES:-}${DEVCONTAINER:-}" ]] && toolchain/devcontainer:present
}

# toolchain/devcontainer:file prints the configuration here, or the usual path when there is none.
toolchain/devcontainer:file() {
  if [[ ! -f .devcontainer/devcontainer.json && -f .devcontainer.json ]]
  then printf '.devcontainer.json\n'
  else printf '.devcontainer/devcontainer.json\n'
  fi
}

toolchain/devcontainer:inputs() {
  printf 'file\t.devcontainer/devcontainer.json\nfile\t.devcontainer.json\n'
  printf 'file\t.devcontainer/devcontainer-lock.json\nfile\t.devcontainer-lock.json\n'
  printf 'env\tREMOTE_CONTAINERS\nenv\tCODESPACES\nenv\tDEVCONTAINER\ntool\tdevcontainer\n'
}

# toolchain/devcontainer:install brings the container up, building it when need be. Inside, it is already up.
toolchain/devcontainer:install() {
  toolchain/devcontainer:active && return 0
  toolchain/devcontainer:require || return 1
  devcontainer up --workspace-folder .
}

# toolchain/devcontainer:outdated asks the CLI, which reaches the registry, for each feature's current and latest
# versions.
toolchain/devcontainer:outdated() {
  local doc
  toolchain/devcontainer:require || return 1
  doc=$(devcontainer outdated --output-format json --workspace-folder .) || return 1
  toolchain/devcontainer:jq devcontainer_outdated <<<"${doc}"
}

# The tools are in the container, so path prints nothing. A command reaches them through run.
toolchain/devcontainer:path() { return 0; }

# toolchain/devcontainer:pins reads the lock alone: each locked feature and its version.
toolchain/devcontainer:pins() {
  local lock
  lock=$(toolchain/devcontainer:lock)
  [[ -f "${lock}" ]] || return 0
  toolchain/devcontainer:jq devcontainer_pins < "${lock}"
}

toolchain/devcontainer:require() {
  command -v devcontainer >/dev/null 2>&1 && return 0
  error "install the devcontainer CLI: https://github.com/devcontainers/cli (the devcontainer toolchain needs it)"
  return 1
}

toolchain/devcontainer:roles() {
  printf 'project\n'
  ! toolchain/devcontainer:active || printf 'system\n'
}

toolchain/devcontainer:update() {
  toolchain/devcontainer:require || return 1
  devcontainer upgrade --workspace-folder .
}

# toolchain/devcontainer:usable needs the file. Outside it needs the CLI too, and fills only the project. Inside,
# the file alone is enough, for the project and for the system.
toolchain/devcontainer:usable() {
  toolchain/devcontainer:present || return 1
  if toolchain/devcontainer:active
  then
    [[ "$1" == project || "$1" == system ]]
  else
    [[ "$1" == project ]] && command -v devcontainer >/dev/null 2>&1
  fi
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/devcontainer:present succeeds when this repository has a dev container configuration.
toolchain/devcontainer:present() {
  [[ -f .devcontainer/devcontainer.json || -f .devcontainer.json ]]
}

# toolchain/devcontainer:lock prints the lock's path: beside the configuration, hidden as it is when the file is.
toolchain/devcontainer:lock() {
  if [[ "$(toolchain/devcontainer:file)" == .devcontainer.json ]]
  then printf '.devcontainer-lock.json\n'
  else printf '.devcontainer/devcontainer-lock.json\n'
  fi
}

# toolchain/devcontainer:jq <function> runs a function of the toolchain jq module over stdin.
toolchain/devcontainer:jq() {
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --raw-output "import \"toolchain\" as toolchain; toolchain::$1"
}

# toolchain/devcontainer:run runs the command in the container, from outside. Inside, toolchain:run does not call it.
toolchain/devcontainer:run() {
  toolchain/devcontainer:require || return 1
  devcontainer exec --workspace-folder . "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
