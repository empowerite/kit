# shellcheck shell=bash
# toolchain-mise.sh - the mise toolchain provider: mise.toml pins the tools, mise.lock records them (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# bin/toolchain.sh loads this provider for `provider: mise`, or when mise.toml or .mise.toml is at the repository
# root. The provider needs mise. When the caller sets MISE_ENV, mise layers mise.<env>.toml on top, as it always
# does.

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

# toolchain-mise:outdated reads mise outdated --json: an object keyed by tool name, with each tool's current and
# latest version.
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
