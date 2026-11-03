# shellcheck shell=bash
# mise.sh - the mise toolchain provider: mise.toml pins the tools, mise.lock records them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# mise fills the project role. It is configured here by mise.toml, .mise.toml, or mise.local.toml at the
# repository root. The provider needs mise. When the caller sets MISE_ENV, mise layers mise.<env>.toml on top, as it
# always does.
#
# It is active where it is configured here and its tools are already on PATH, however they got there:
#
#   - mise activate sets MISE_SHELL
#   - every directory mise bin-paths prints is on PATH, as direnv, a harness, or a runner's setup puts them
#
# Shims alone do not count: they sit on PATH in every directory, set up once per machine, not for this repository.
#
# Detection's cache keeps the answer while PATH, MISE_ENV, and mise's files are unchanged.

toolchain/mise:active() {
  toolchain/mise:configured || return 1
  [[ -n "${MISE_SHELL:-}" ]] && return 0
  toolchain/mise:on-path
}

toolchain/mise:file() {
  if [[ -f .mise.toml && ! -f mise.toml ]]
  then printf '.mise.toml\n'
  else printf 'mise.toml\n'
  fi
}

toolchain/mise:inputs() {
  local env
  printf 'tool\tmise\nenv\tMISE_SHELL\nenv\tMISE_ENV\n'
  printf 'file\t%s\n' mise.toml .mise.toml mise.local.toml mise.lock
  # MISE_ENV names one environment or several, comma-separated, each with its own file and lock
  for env in ${MISE_ENV:+${MISE_ENV//,/ }}
  do
    printf 'file\t%s\n' "mise.${env}.toml" "mise.${env}.lock"
  done
}

toolchain/mise:install() {
  toolchain/mise:require || return 1
  mise trust --quiet "$(toolchain/mise:file)" >/dev/null 2>&1 || true
  mise install --quiet
}

# toolchain/mise:outdated reads mise outdated --json: an object keyed by tool name, with each tool's current and
# latest version.
toolchain/mise:outdated() {
  local doc
  toolchain/mise:require || return 1
  doc=$(mise outdated --json 2>/dev/null) || { error "mise outdated could not run; see: mise doctor"; return 1; }
  jq --raw-output 'to_entries[] | "\(.key)\t\(.value.current // "none")\t\(.value.latest)"' <<<"${doc:-{\}}"
}

toolchain/mise:path() {
  toolchain/mise:require || return 1
  mise bin-paths 2>/dev/null
}

# toolchain/mise:pins reads mise ls offline, so the listing never reaches the network, even for a pin of latest.
toolchain/mise:pins() {
  local doc
  toolchain/mise:require || return 1
  if ! doc=$(MISE_OFFLINE=1 mise ls --json --current 2>/dev/null)
  then
    error "mise ls could not run; see: mise doctor"
    return 1
  fi
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --raw-output 'import "toolchain" as toolchain; toolchain::mise_pins' <<<"${doc:-{\}}"
}

toolchain/mise:require() {
  command -v mise >/dev/null 2>&1 && return 0
  error "install mise: https://mise.jdx.dev/installing-mise.html (the mise toolchain needs it)"
  return 1
}

toolchain/mise:roles() { printf 'project\n'; }

toolchain/mise:update() {
  toolchain/mise:require || return 1
  mise upgrade --quiet
}

# toolchain/mise:usable runs mise offline, so a stale shim or a broken install is not usable.
toolchain/mise:usable() {
  [[ "$1" == project ]] || return 1
  command -v mise >/dev/null 2>&1 && MISE_OFFLINE=1 mise --version >/dev/null 2>&1 && toolchain/mise:configured
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/mise:on-path succeeds when every directory mise bin-paths prints is on PATH. It runs mise offline, and
# only where mise is installed.
toolchain/mise:on-path() {
  local dirs dir
  command -v mise >/dev/null 2>&1 || return 1
  dirs=$(MISE_OFFLINE=1 mise bin-paths 2>/dev/null) && [[ -n "${dirs}" ]] || return 1
  while IFS= read -r dir
  do
    [[ ":${PATH}:" == *":${dir}:"* ]] || return 1
  done <<<"${dirs}"
}

# toolchain/mise:configured succeeds when one of mise's files is at the repository root.
toolchain/mise:configured() {
  [[ -f mise.toml || -f .mise.toml || -f mise.local.toml ]]
}

# ----------------------------------------------------------------------------------------------------------------------
