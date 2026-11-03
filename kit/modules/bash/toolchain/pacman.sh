# shellcheck shell=bash
# pacman.sh - the pacman toolchain provider: Arch Linux's packages, for the system role (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# pacman fills the system role wherever pacman is on PATH. It installs for the whole machine, so it never fills the
# project role, and the project calls refuse. pacman is never active: it is not a shell one enters.
#
# kit never runs pacman itself. remedy names the command, and kit welcome runs it only when the person says now, in
# the terminal, so sudo can ask for a password.
#
# A package's name can differ from the tool's. Arch's yq is kislyuk's; mikefarah's, which kit needs, is go-yq.

toolchain/pacman:active() { return 1; }

toolchain/pacman:file() { toolchain:system-only pacman; }

toolchain/pacman:inputs() { printf 'tool\tpacman\n'; }

toolchain/pacman:install() { toolchain:system-only pacman; }

toolchain/pacman:outdated() { toolchain:system-only pacman; }

toolchain/pacman:path() { toolchain:system-only pacman; }

toolchain/pacman:pins() { toolchain:system-only pacman; }

toolchain/pacman:require() { command -v pacman >/dev/null 2>&1; }

toolchain/pacman:roles() { printf 'system\n'; }

toolchain/pacman:update() { toolchain:system-only pacman; }

# toolchain/pacman:usable never runs pacman. Finding it on PATH is enough.
toolchain/pacman:usable() {
  [[ "$1" == system ]] && command -v pacman >/dev/null 2>&1
}

# toolchain/pacman:remedy prints the command that installs one tool, and fails for a tool Arch has no package for.
toolchain/pacman:remedy() {
  local package
  package=$(toolchain/pacman:package "$1") || return 1
  printf '%spacman -S %s\n' "$(toolchain:sudo)" "${package}"
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/pacman:package <tool> prints the package that installs the tool, and fails when there is none.
toolchain/pacman:package() {
  case "$1" in
    (bash | git | jq | shellcheck | glab | actionlint) printf '%s\n' "$1" ;;
    (yq) printf 'go-yq\n' ;;
    (gh) printf 'github-cli\n' ;;
    (*) return 1 ;;
  esac
}

# ----------------------------------------------------------------------------------------------------------------------
