# shellcheck shell=bash
# toolchain-brew.sh - the Homebrew toolchain provider: a Brewfile lists the formulae and casks (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# bash/toolchain.sh loads this provider for `provider: brew`, or when a Brewfile is at the repository root. The
# `file:` field names a different file.
#
# A Brewfile pins a formula by name only, so its version moves with Homebrew unless the name includes it
# (node@22). The kit can only report which formulae are behind and upgrade them.
#
# The provider needs brew, on PATH or at one of Homebrew's own prefixes. That covers a Linux runner where brew is
# installed but not on PATH.

toolchain-brew:file() {
  local file
  file=$(toolchain:field brew file)
  printf '%s\n' "${file:-Brewfile}"
}

# toolchain-brew:brew prints the brew executable, on PATH or at one of Homebrew's prefixes.
toolchain-brew:brew() {
  local brew
  for brew in "$(command -v brew 2>/dev/null)" /opt/homebrew/bin/brew /usr/local/bin/brew \
    /home/linuxbrew/.linuxbrew/bin/brew
  do
    [[ -n "${brew}" && -x "${brew}" ]] && { printf '%s\n' "${brew}"; return 0; }
  done
  error "install Homebrew: https://brew.sh (the brew toolchain needs it)"
  return 1
}

toolchain-brew:require() { toolchain-brew:brew >/dev/null; }

toolchain-brew:install() {
  local brew
  brew=$(toolchain-brew:brew) || return 1
  "${brew}" bundle install --file "$(toolchain-brew:file)" --no-upgrade --quiet
}

# toolchain-brew:outdated prints the Brewfile's formulae and casks that brew reports as outdated.
toolchain-brew:outdated() {
  local brew names doc
  brew=$(toolchain-brew:brew) || return 1
  names=$("${brew}" bundle list --file "$(toolchain-brew:file)" --formula --cask 2>/dev/null) || return 1
  doc=$("${brew}" outdated --json=v2 2>/dev/null) || return 1
  jq --raw-output --arg names "${names}" '
    ($names | split("\n") | map(select(. != ""))) as $wanted
    | (.formulae + .casks)[]
    | select(.name as $n | $wanted | index($n))
    | "\(.name)\t\(.installed_versions | last)\t\(.current_version)"' <<<"${doc}"
}

toolchain-brew:update() {
  local brew outdated
  brew=$(toolchain-brew:brew) || return 1
  outdated=$(toolchain-brew:outdated | cut -f 1) || return 1
  [[ -z "${outdated}" ]] && return 0
  # shellcheck disable=SC2086  # one formula or cask per word
  "${brew}" upgrade ${outdated}
}

toolchain-brew:path() {
  local brew prefix
  brew=$(toolchain-brew:brew) || return 1
  prefix=$("${brew}" --prefix)
  printf '%s/bin\n%s/sbin\n' "${prefix}" "${prefix}"
}

# ----------------------------------------------------------------------------------------------------------------------
