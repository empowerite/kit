# shellcheck shell=bash
# dnf.sh - the DNF toolchain provider: Fedora's packages, for the system role (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# dnf fills the system role wherever dnf is on PATH. It installs for the whole machine, so it never fills the project
# role, and the project calls refuse. dnf is never active: it is not a shell one enters.
#
# kit never runs dnf itself. remedy names the command, and kit welcome runs it only when the person says now, in the
# terminal, so sudo can ask for a password.
#
# A package's name can differ from the tool's: Fedora names shellcheck ShellCheck.

toolchain/dnf:active() { return 1; }

toolchain/dnf:file() { toolchain:system-only dnf; }

toolchain/dnf:inputs() { printf 'tool\tdnf\n'; }

toolchain/dnf:install() { toolchain:system-only dnf; }

toolchain/dnf:outdated() { toolchain:system-only dnf; }

toolchain/dnf:path() { toolchain:system-only dnf; }

toolchain/dnf:pins() { toolchain:system-only dnf; }

toolchain/dnf:require() { command -v dnf >/dev/null 2>&1; }

toolchain/dnf:roles() { printf 'system\n'; }

toolchain/dnf:update() { toolchain:system-only dnf; }

# toolchain/dnf:usable never runs dnf. Finding it on PATH is enough.
toolchain/dnf:usable() {
  [[ "$1" == system ]] && command -v dnf >/dev/null 2>&1
}

# toolchain/dnf:remedy prints the command that installs one tool, and fails for a tool Fedora has no package for.
toolchain/dnf:remedy() {
  local package
  package=$(toolchain/dnf:package "$1") || return 1
  printf '%sdnf install %s\n' "$(toolchain:sudo)" "${package}"
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/dnf:package <tool> prints the package that installs the tool, and fails when there is none. Fedora's yq
# is mikefarah's.
toolchain/dnf:package() {
  case "$1" in
    (bash | git | jq | yq | gh | glab) printf '%s\n' "$1" ;;
    (shellcheck) printf 'ShellCheck\n' ;;
    (*) return 1 ;;
  esac
}

# ----------------------------------------------------------------------------------------------------------------------
