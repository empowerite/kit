# shellcheck shell=bash
# dependencies.sh - the seam over the package managers that pin a language's packages (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# A toolchain pins tools, node and python among them. A language's package manager pins that language's packages on
# top: npm reads package.json, and uv reads pyproject.toml. Each is a provider: dependencies/<name>.sh beside it,
# whose functions are named dependencies/<name>:<call>.
#
# A run uses one toolchain, but every usable dependency provider at once. Each is looked for with the chosen
# toolchain's directories first on PATH, so a node the toolchain pins counts. A provider reads the manifest at the
# repository root only, and needs its lock: without one, nothing is pinned. Finding one stats two files and looks up
# one name on PATH, so nothing is cached.
#
# Every provider implements the calls below.
#
#   file                    the manifest it reads, relative to the repository root
#   lock                    the lock beside it, which pins every package's version
#   usable                  succeeds when its tool is on PATH, and the manifest and its lock are here
#   install                 installs what the lock pins, exactly, and never rewrites the lock
#   path                    the directory that holds the packages' commands, absolute
#   pins                    one line per pinned package, "<name>\t<pinned>\t<installed>"; never reaches the network
#   outdated                one line per package behind its latest: "<name>\t<installed>\t<latest>"; fails when it
#                           cannot ask
#   update                  moves every package to its latest, and rewrites the lock
#
# see: docs/toolchains.md, issue #54

use toolchain

# dependencies:providers prints the providers on the module path, space-separated.
dependencies:providers() {
  use:modules 'dependencies/*' | sed 's|^dependencies/||' | paste -s -d ' ' -
}

# dependencies:call <provider> <call> <argument>… runs the provider's function for the call.
dependencies:call() {
  local provider=$1 call=$2
  shift 2
  use "dependencies/${provider}" || return 1
  "dependencies/${provider}:${call}" "$@"
}

# dependencies:usable [<dirs>] prints the providers usable here, one per line, in dependencies:providers order. Each
# is looked for with <dirs>, directories joined by colons, first on PATH. Without <dirs>, they are the chosen
# toolchain's, found only where some manifest and its lock are here, since a toolchain's path can be slow to find.
dependencies:usable() {
  local provider dirs
  local -a here=()
  for provider in $(dependencies:providers)
  do
    ! dependencies:locked "${provider}" || here+=("${provider}")
  done
  (( ${#here[@]} )) || return 0
  if (( $# ))
  then dirs=$1
  else dirs=$(dependencies:toolchain-dirs)
  fi
  for provider in "${here[@]}"
  do
    if PATH="${dirs:+${dirs}:}${PATH}" dependencies:call "${provider}" usable
    then printf '%s\n' "${provider}"
    fi
  done
}

# dependencies:locked <provider> succeeds when the provider's manifest and its lock are both here.
dependencies:locked() {
  [[ -f "$(dependencies:call "$1" file)" && -f "$(dependencies:call "$1" lock)" ]]
}

# dependencies:unlocked prints each provider whose manifest is here without its lock, one per line. It pins nothing,
# so it is not usable.
dependencies:unlocked() {
  local provider
  for provider in $(dependencies:providers)
  do
    if [[ -f "$(dependencies:call "${provider}" file)" && ! -f "$(dependencies:call "${provider}" lock)" ]]
    then printf '%s\n' "${provider}"
    fi
  done
}

# dependencies:toolchain-dirs prints the chosen project toolchain's directories, joined by colons. It never asks
# which toolchain to use: with none settled, or ambient, it prints nothing.
dependencies:toolchain-dirs() {
  local chosen
  chosen=$(toolchain:chosen project list 2>/dev/null) || return 0
  [[ -n "${chosen}" && "${chosen}" != ambient ]] || return 0
  toolchain:call "${chosen}" path 2>/dev/null | paste -s -d ':' -
}

# dependencies:path [<dirs>] prints the directory of each usable provider, one per line, after <dirs>, as
# dependencies:usable takes them.
dependencies:path() {
  local provider
  local -a usable
  list:read usable dependencies:usable "$@" || return 1
  for provider in "${usable[@]}"
  do
    dependencies:call "${provider}" path
  done
}

# dependencies:installed <provider> succeeds when every package the provider pins is installed at its pinned version.
dependencies:installed() {
  local pins
  pins=$(dependencies:call "$1" pins 2>/dev/null) || return 1
  awk -F '\t' 'NF && $2 != $3 { behind = 1 } END { exit behind }' <<<"${pins}"
}

# ----------------------------------------------------------------------------------------------------------------------
