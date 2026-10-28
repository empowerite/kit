# shellcheck shell=bash
# toolchain-mise.sh - mise as a toolchain: mise.toml pins the tools, mise.lock records them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The provider bin/toolchain.sh loads for `provider: mise`, or for a mise.toml or .mise.toml at the root. mise is
# its requirement; MISE_ENV, where the caller sets it, layers mise.<env>.toml on top as mise does.

toolchain-mise:file() {
  if [[ -f .mise.toml && ! -f mise.toml ]]
  then printf '.mise.toml\n'
  else printf 'mise.toml\n'
  fi
}

toolchain-mise:require() {
  command -v mise >/dev/null 2>&1 && return 0
  error "install mise: https://mise.jdx.dev/installing-mise.html (the mise toolchain needs it)"
  return 1
}

toolchain-mise:install() {
  toolchain-mise:require || return 1
  mise trust --quiet "$(toolchain-mise:file)" >/dev/null 2>&1 || true
  mise install --quiet
}

# mise outdated --json is an object keyed by tool, each with its current and latest version.
toolchain-mise:outdated() {
  local doc
  toolchain-mise:require || return 1
  doc=$(mise outdated --json 2>/dev/null) || { error "mise outdated could not run; see: mise doctor"; return 1; }
  jq --raw-output 'to_entries[] | "\(.key)\t\(.value.current // "none")\t\(.value.latest)"' <<<"${doc:-{\}}"
}

toolchain-mise:update() {
  toolchain-mise:require || return 1
  mise upgrade --quiet
}

toolchain-mise:path() {
  toolchain-mise:require || return 1
  mise bin-paths 2>/dev/null
}

# ----------------------------------------------------------------------------------------------------------------------
