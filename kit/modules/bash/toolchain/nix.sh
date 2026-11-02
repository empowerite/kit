# shellcheck shell=bash
# nix.sh - the nix toolchain provider: flake.nix declares a dev shell, flake.lock pins its inputs (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# nix fills the project role. It is configured here by a flake.nix at the repository root that declares devShells or
# devShell, which the provider reads as text and never evaluates. It is usable only where flakes are on. It is active
# inside nix develop, which sets IN_NIX_SHELL, and only where this repository has a flake.nix: a marker left over from
# another project does not count.
#
# A probe never evaluates the flake and never reaches the network. Asking nix whether flakes are on runs only when
# detection misses its cache. Evaluating the flake is for install, path, and update. Outdated reaches the network.
# Nix itself ignores a flake.nix that git does not track, so add the file before installing.

toolchain/nix:active() {
  [[ -n "${IN_NIX_SHELL:-}" && -f flake.nix ]]
}

toolchain/nix:file() {
  printf 'flake.nix\n'
}

toolchain/nix:inputs() {
  printf 'file\tflake.nix\nfile\tflake.lock\nenv\tIN_NIX_SHELL\ntool\tnix\n'
}

toolchain/nix:install() {
  toolchain/nix:require || return 1
  nix develop --command true
}

# toolchain/nix:outdated asks nix for the latest revision of each input's original reference. It prints each input
# whose locked revision differs, with the locked revision as installed. An input pinned to a revision in flake.nix
# is not asked.
toolchain/nix:outdated() {
  local name ref locked doc latest
  toolchain/nix:require || return 1
  [[ -f flake.lock ]] || return 0
  while IFS=$'\t' read -r name ref locked
  do
    if ! doc=$(nix flake metadata --json "${ref}" 2>/dev/null)
    then
      error "nix could not read ${ref}, the source of ${name}; see: nix flake metadata ${ref}"
      return 1
    fi
    latest=$(jq --raw-output '.locked.rev // empty' <<<"${doc}")
    [[ -z "${latest}" || "${latest}" == "${locked}" ]] && continue
    printf '%s\t%s\t%s\n' "${name}" "${locked:0:7}" "${latest:0:7}"
  done < <(toolchain/nix:lock nix_refs)
}

# toolchain/nix:path prints the dev shell's PATH, one directory per line. This evaluates the flake.
toolchain/nix:path() {
  local doc
  toolchain/nix:require || return 1
  if ! doc=$(nix print-dev-env --json 2>/dev/null)
  then
    error "nix could not build the dev shell; see: nix develop"
    return 1
  fi
  jq --raw-output '.variables.PATH.value // empty' <<<"${doc}" | tr ':' '\n'
}

# toolchain/nix:pins reads flake.lock alone: each flake input by its locked revision, short. Nix keeps no separate
# record of what is in the store, so the pinned and installed columns are the same revision.
toolchain/nix:pins() {
  [[ -f flake.lock ]] || return 0
  toolchain/nix:lock nix_pins
}

toolchain/nix:require() {
  command -v nix >/dev/null 2>&1 && return 0
  error "install nix: https://nixos.org/download (the nix toolchain needs it)"
  return 1
}

toolchain/nix:roles() { printf 'project\n'; }

toolchain/nix:update() {
  toolchain/nix:require || return 1
  nix flake update
}

# toolchain/nix:usable needs nix on PATH, a dev shell named in flake.nix, and flakes on. The file is read first, so
# nix runs only where a flake could use it.
toolchain/nix:usable() {
  [[ "$1" == project ]] || return 1
  command -v nix >/dev/null 2>&1 || return 1
  grep --quiet --extended-regexp 'devShells?([^[:alnum:]_]|$)' flake.nix 2>/dev/null || return 1
  nix config show experimental-features 2>/dev/null | grep --quiet --word-regexp flakes
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/nix:lock <function> runs a function of the toolchain jq module over flake.lock.
toolchain/nix:lock() {
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --raw-output "import \"toolchain\" as toolchain; toolchain::$1" flake.lock
}

# ----------------------------------------------------------------------------------------------------------------------
