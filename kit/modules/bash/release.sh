# shellcheck shell=bash
# release.sh - a release: its version, where it is fetched from, and the bases it delivers to (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# A release is bin/, modules/, and packages/, with a version file the release workflow writes; a checkout has none
# and is `unreleased`. kit update fetches the previous release here, checked against the digest published beside it.
# What a release delivers, and how, is the work of the modules beside this one:
#
#   resolve.sh      the packages in reach, their metadata, and the active set from the names, their dependencies,
#                   and the bundles among them
#   settings.sh     kit.yml, read and validated, and the declaration the kit writes into it
#   render.sh       a package's templates and regions, rendered once per process with the repository's answers
#   region.sh       the kit's region inside a file the repository owns
#   place.sh        a delivered path, written, merged by key, compared with the release, and taken out again
#   materialize.sh  what kit keeps under .kit/: a link or a read-only copy per path, and .kit/.gitignore
#   manifest.sh     .kit/manifest and ~/.kit/manifest, the digest of every delivered path
#
# Every function in these modules reads a release from a root directory: kit_root by default, which
# modules/bash/1st.sh exports. kit:enter changes to the repository root, so every repo path is relative to it. Every
# sort and comm uses the C locale, so an order written on one machine reads the same on another. Every file written
# goes through modules/bash/file.sh, whole or not at all.
#
# see: docs/packages.md, docs/modules.md

use file yaml resolve

# release:unreleased prints `unreleased`, the version of a kit checkout. The release workflow
# (.github/workflows/release.yml) writes a version file into each release; a checkout has none. kit update never
# fetches this version.
release:unreleased() { printf 'unreleased\n'; }

# release:version prints this release's version.
release:version() {
  cat "${kit_root}/version" 2>/dev/null || release:unreleased
}

# release:bases prints the bases a package can deliver to. Each base is a subdirectory name under files/,
# templates/, and regions/.
release:bases() { printf 'repo\nhome\n'; }

# release:base-root <base> prints the directory a base's files are copied to: `.` (the repository root, where the
# verbs run) for repo, and the home directory for home. KIT_HOME, when set, replaces the home directory, so a test
# can use a stand-in.
release:base-root() {
  case "$1" in
    (repo) printf '.\n' ;;
    (home) printf '%s\n' "${KIT_HOME:-${HOME}}" ;;
    (*)
      error "'$1' is not a base; the bases are: $(release:bases | paste -s -d ' ' -)"
      return 2
  esac
}

# release:dest <base> <path> prints where a delivered path is written.
release:dest() {
  if [[ "$1" == repo ]]
  then printf '%s\n' "$2"
  else printf '%s/%s\n' "$(release:base-root "$1")" "$2"
  fi
}

# release:target <path> prints the file that a write to <path> changes: the path itself, or the target of a
# symlink there. A repository that keeps AGENTS.md or a settings file as a link into another directory keeps the
# link.
release:target() {
  if [[ -L "$1" ]]
  then readlink -f "$1"
  else printf '%s\n' "$1"
  fi
}

# release:fetch <version> prints the root of that release, fetching it if needed. It tries, in order:
#
#   1. the kit's cache
#   2. a copy installed by mise, when mise is present
#   3. the release's public tarball, downloaded with curl beside the .sha256 the release publishes, and unpacked into
#      the cache only when its bytes have that digest; a release that publishes no digest is refused
#
# The version comes from a manifest, so it must have a version's shape before it is used in a path.
release:fetch() {
  local version=$1 root cache fresh tarball published have
  if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]
  then
    error "'${version}' is not a kit version"
    return 1
  fi
  cache="${XDG_CACHE_HOME:-${HOME}/.cache}/kit/kit-${version}"
  if [[ -x "${cache}/bin/kit" ]]
  then
    printf '%s\n' "${cache}"
    return 0
  fi
  if command -v mise >/dev/null 2>&1
  then
    root=$(mise where "github:empowerite/kit@${version}" 2>/dev/null || true)
    if [[ -x "${root}/bin/kit" ]]
    then
      printf '%s\n' "${root}"
      return 0
    fi
  fi
  step "Fetching kit ${version} from its release"
  file:scratch fresh || return 1
  tarball="https://github.com/empowerite/kit/releases/download/release/${version}/kit-${version}.tar.gz"
  if ! curl --fail --silent --location --output "${fresh}/kit.tar.gz" "${tarball}" 2>/dev/null
  then
    rm -rf "${fresh}"
    error "kit ${version} could not be fetched from its release; set KIT_BASE to a copy of it"
    return 1
  fi
  if ! curl --fail --silent --location --output "${fresh}/kit.tar.gz.sha256" "${tarball}.sha256" 2>/dev/null
  then
    rm -rf "${fresh}"
    error "kit ${version} publishes no digest beside its tarball, so it is not trusted; set KIT_BASE to a copy of it"
    return 1
  fi
  published=$(cut -d ' ' -f 1 "${fresh}/kit.tar.gz.sha256")
  have=$(file:digest < "${fresh}/kit.tar.gz")
  if [[ "${have}" != "${published}" ]]
  then
    rm -rf "${fresh}"
    error "kit ${version}'s tarball has digest ${have}, not the ${published} its release publishes; it is not unpacked"
    return 1
  fi
  mkdir "${fresh}/tree"
  if ! tar --extract --gzip --file "${fresh}/kit.tar.gz" --directory "${fresh}/tree" --strip-components 1 2>/dev/null
  then
    rm -rf "${fresh}"
    error "kit ${version}'s tarball could not be unpacked; set KIT_BASE to a copy of it"
    return 1
  fi
  mkdir -p "$(dirname "${cache}")"
  mv "${fresh}/tree" "${cache}"
  rm -rf "${fresh}"
  printf '%s\n' "${cache}"
}

# release:finish <package>… reports a finished install or update.
release:finish() {
  ok "kit $(release:version) is installed, packages: ${*:-none}; kit.yml declares them."
  release:say-unmet "$@"
  release:say-notices "$@"
  step "Commit kit.yml and what kit changed outside .kit/; what it placed under .kit/ is ignored. The" \
    "workflow is in .kit/1st.md."
}

# release:say-notices <package>… prints each notice of the packages whose path this process placed afresh, or merged
# a new entry into: what happens next, such as a harness asking once whether to use a server the file names.
release:say-notices() {
  local package path text placed
  for package in "$@"
  do
    while IFS=$'\x1f' read -r path text
    do
      [[ -n "${path}" ]] || continue
      for placed in "${place_placed[@]}"
      do
        [[ "${placed}" == "${path}" ]] && step "${path}: ${text}"
      done
    done < <(yaml:records "$(resolve:meta "${kit_root}" "${package}")" notices path text)
  done
  return 0
}

# release:say-unmet <package>… warns about each unmet expectation of the packages, as a step left to do.
release:say-unmet() {
  local line
  while IFS= read -r line
  do
    [[ -n "${line}" ]] && warn "left to do in ${line}"
  done < <(resolve:unmet "${kit_root}" "$@")
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
