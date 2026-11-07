# shellcheck shell=bash
# brew.sh - the Homebrew toolchain provider: a Brewfile lists the formulae and casks (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# brew fills the system role wherever brew is installed. On macOS, a Brewfile at the repository root makes it usable
# for the project role too. Elsewhere, as on a Linux runner, the Brewfile is not read. brew is never active: it is not
# a shell one enters.
#
# brew never competes with another project toolchain. Where one is usable, brew fills in: it acts on the Brewfile's
# tools that toolchain does not pin, named in toolchain_pinned (see toolchain:filler). Alone, it acts on them all.
#
# A Brewfile pins a formula by name only, so its version moves with Homebrew unless the name includes it
# (node@22). The kit can only report which formulae are behind and upgrade them.
#
# The provider needs brew on PATH. A brew only at one of Homebrew's own prefixes is not usable: kit welcome offers the
# line that puts it on PATH (toolchain/brew:prefixed names it).

toolchain/brew:active() { return 1; }

toolchain/brew:file() { printf 'Brewfile\n'; }

toolchain/brew:inputs() { printf 'tool\tbrew\nfile\tBrewfile\nenv\tOSTYPE\n'; }

# toolchain/brew:install installs from the Brewfile without upgrading. Filling in, it installs only the tools it acts
# on that are missing, by name, so a tool the other toolchain pins is left to it.
toolchain/brew:install() {
  local brew
  local -a missing
  brew=$(toolchain/brew:brew) || return 1
  if [[ -z "${toolchain_pinned+x}" ]]
  then
    "${brew}" bundle install --file "$(toolchain/brew:file)" --no-upgrade --quiet
    return
  fi
  readarray -t missing < <(toolchain/brew:pins | awk -F '\t' '$3 == "" { print $1 }') || return 1
  (( ${#missing[@]} == 0 )) || "${brew}" install --quiet "${missing[@]}"
}

# toolchain/brew:outdated prints the Brewfile's formulae and casks that brew reports as outdated.
toolchain/brew:outdated() {
  local brew names doc
  brew=$(toolchain/brew:brew) || return 1
  names=$(toolchain/brew:names) || return 1
  doc=$("${brew}" outdated --json=v2 2>/dev/null) || return 1
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --raw-output --arg names "${names}" \
    'import "toolchain" as toolchain; toolchain::brew_outdated($names)' <<<"${doc}"
}

toolchain/brew:path() {
  local brew prefix
  brew=$(toolchain/brew:brew) || return 1
  prefix=$("${brew}" --prefix)
  printf '%s/bin\n%s/sbin\n' "${prefix}" "${prefix}"
}

# toolchain/brew:pins reads the Brewfile's names and the versions brew has installed. Neither updates Homebrew.
toolchain/brew:pins() {
  local brew names versions
  brew=$(toolchain/brew:brew) || return 1
  names=$(toolchain/brew:names) || return 1
  versions=$(HOMEBREW_NO_AUTO_UPDATE=1 "${brew}" list --versions 2>/dev/null) || return 1
  jq -L "${kit_jq}" --raw-input --slurp --raw-output --arg names "${names}" \
    'import "toolchain" as toolchain; toolchain::brew_pins($names)' <<<"${versions}"
}

toolchain/brew:require() { toolchain/brew:brew >/dev/null; }

toolchain/brew:roles() { printf 'system\nproject\n'; }

toolchain/brew:update() {
  local brew outdated
  brew=$(toolchain/brew:brew) || return 1
  outdated=$(toolchain/brew:outdated | cut -f 1) || return 1
  [[ -z "${outdated}" ]] && return 0
  # shellcheck disable=SC2086  # one formula or cask per word
  "${brew}" upgrade ${outdated}
}

# toolchain/brew:usable never runs brew, which calls git on every start. Finding it on PATH is enough. The project
# role needs macOS and a Brewfile as well.
toolchain/brew:usable() {
  [[ "$1" == system || "$1" == project ]] || return 1
  command -v brew >/dev/null 2>&1 || return 1
  [[ "$1" == system || ( "${OSTYPE:-}" == darwin* && -f Brewfile ) ]]
}

# toolchain/brew:remedy prints the command that installs one tool for the system role. Homebrew's formulae are named
# for the tools kit checks.
toolchain/brew:remedy() { printf 'brew install %s\n' "$1"; }

# The seam's calls end here; what follows is this provider's own.

# toolchain_brew_prefixes holds where Homebrew installs brew itself. KIT_BREW_PREFIXES, a space-separated list of brew
# paths, replaces them, as a test does.
toolchain_brew_default="/opt/homebrew/bin/brew /usr/local/bin/brew /home/linuxbrew/.linuxbrew/bin/brew"
read -r -a toolchain_brew_prefixes <<<"${KIT_BREW_PREFIXES:-${toolchain_brew_default}}"
declare -ga toolchain_brew_prefixes

# toolchain/brew:prefixed prints the first brew executable at a Homebrew prefix, found or not on PATH, and fails when
# none is there.
toolchain/brew:prefixed() {
  local brew
  for brew in "${toolchain_brew_prefixes[@]}"
  do
    [[ -x "${brew}" ]] && { printf '%s\n' "${brew}"; return 0; }
  done
  return 1
}

# toolchain/brew:names prints the Brewfile's formulae and casks, one per line. Filling in, it leaves out each one
# toolchain_pinned names, read as toolchain:pinned-names reads them. It never updates Homebrew.
toolchain/brew:names() {
  local brew names
  brew=$(toolchain/brew:brew) || return 1
  names=$(HOMEBREW_NO_AUTO_UPDATE=1 "${brew}" bundle list --file "$(toolchain/brew:file)" --formula --cask \
    2>/dev/null) || return 1
  if [[ -z "${toolchain_pinned+x}" ]]
  then
    printf '%s\n' "${names}"
    return 0
  fi
  awk 'NR == FNR { pinned[$0]; next }
    { name = $0; sub(/^.*:/, "", name); sub(/^.*\//, "", name); sub(/@.*$/, "", name); if (!(name in pinned)) print }' \
    <(printf '%s\n' "${toolchain_pinned}") <(printf '%s\n' "${names}")
}

# toolchain/brew:brew prints the brew executable on PATH.
toolchain/brew:brew() {
  command -v brew 2>/dev/null && return 0
  error "put Homebrew on PATH, or install it: https://brew.sh (the brew toolchain needs it)"
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
