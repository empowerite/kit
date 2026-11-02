# shellcheck shell=bash
# apt.sh - the APT toolchain provider: Debian's and Ubuntu's packages, for the system role (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# apt fills the system role wherever apt-get is on PATH. It installs for the whole machine, so it never fills the
# project role, and the project calls refuse. apt is never active: it is not a shell one enters.
#
# kit never runs apt-get itself. remedy names the command, and kit welcome runs it only when the person says now,
# in the terminal, so sudo can ask for a password.
#
# A package's name can differ from the tool's. Debian's yq is not mikefarah's, which kit needs, so yq has none here.

toolchain/apt:active() { return 1; }

toolchain/apt:file() { toolchain:system-only apt; }

toolchain/apt:inputs() { printf 'tool\tapt-get\n'; }

toolchain/apt:install() { toolchain:system-only apt; }

toolchain/apt:outdated() { toolchain:system-only apt; }

toolchain/apt:path() { toolchain:system-only apt; }

toolchain/apt:pins() { toolchain:system-only apt; }

toolchain/apt:require() { command -v apt-get >/dev/null 2>&1; }

toolchain/apt:roles() { printf 'system\n'; }

toolchain/apt:update() { toolchain:system-only apt; }

# toolchain/apt:usable never runs apt-get. Finding it on PATH is enough.
toolchain/apt:usable() {
  [[ "$1" == system ]] && command -v apt-get >/dev/null 2>&1
}

# toolchain/apt:remedy prints the command that installs one tool, and fails for a tool Debian and Ubuntu have no
# package for.
toolchain/apt:remedy() {
  local package
  package=$(toolchain/apt:package "$1") || return 1
  printf '%sapt-get install %s\n' "$(toolchain:sudo)" "${package}"
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/apt:package <tool> prints the package that installs the tool, and fails when there is none.
toolchain/apt:package() {
  case "$1" in
    (bash | git | jq | shellcheck | gh | glab) printf '%s\n' "$1" ;;
    (*) return 1 ;;
  esac
}

# ----------------------------------------------------------------------------------------------------------------------
