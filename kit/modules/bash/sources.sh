# shellcheck shell=bash
# sources.sh - the sources packages come from beyond the release: pinned by digest, fetched once, cached (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A repository takes packages from the kit's release and from its own .kit/packages/. It can name more sources, in
# order, under sources: in .kit/settings.yml, each a tarball pinned by the sha256 digest of its bytes:
#
#   sources:
#     - name: team
#       url: git:empowerite/kit-team@v1.2.0
#       digest: sha256:3b1f…
#
# A url is git:<owner>/<repo>@<tag>, the forge's tarball of that tag (github.com, or a host named first, as
# git:gitlab.com/<owner>/<repo>@<tag>), or any URL curl fetches, such as https://…/team-1.2.0.tar.gz. A source is
# fetched once into the kit's cache, under its digest, and refused when its bytes have any other digest; nothing from
# a source runs before that check. Its packages are under packages/ in the tarball, or under kit/packages/ for a
# repository laid out as the kit's own is.
#
# A package name is looked up in the sources, the last first, then in the release, then in the repository's own
# .kit/packages/, so a later source replaces a package of the same name. A bash_path entry <name>:<dir> names a
# directory in a source.
#
# see: modules/bash/release.sh, docs/packages.md

use yaml

# sources:list prints "<name>\x1f<url>\x1f<digest>" for each source in .kit/settings.yml, in order.
sources:list() {
  [[ -f .kit/settings.yml ]] || return 0
  yaml:records .kit/settings.yml sources name url digest
}

# sources:cache <digest> prints the directory a source with that digest is unpacked into.
sources:cache() {
  printf '%s/kit/sources/%s\n' "${XDG_CACHE_HOME:-${HOME}/.cache}" "${1#sha256:}"
}

# sources:tarball <url> prints the address of the tarball a url names: a git: url as its forge's tarball of the tag.
sources:tarball() {
  local spec host repo tag
  case "$1" in
    (git:*)
      spec=${1#git:}
      tag=${spec##*@}
      spec=${spec%@*}
      host=github.com
      [[ "${spec%%/*}" == *.* ]] && host=${spec%%/*} spec=${spec#*/}
      repo=${spec##*/}
      case "${host}" in
        (github.com) printf 'https://codeload.github.com/%s/tar.gz/refs/tags/%s\n' "${spec}" "${tag}" ;;
        (*) printf 'https://%s/%s/-/archive/%s/%s-%s.tar.gz\n' "${host}" "${spec}" "${tag}" "${repo}" "${tag}"
      esac
      ;;
    (*) printf '%s\n' "$1"
  esac
}

# sources:digest <file> prints the file's sha256 digest as sha256:<hex>.
sources:digest() {
  local hex
  if command -v sha256sum >/dev/null 2>&1
  then hex=$(sha256sum "$1" | cut -d ' ' -f 1)
  else hex=$(shasum -a 256 "$1" | cut -d ' ' -f 1)
  fi
  printf 'sha256:%s\n' "${hex}"
}

# sources:fetch <name> <url> <digest> unpacks the source into its cache directory, unless it is there already, and
# prints the directory. It refuses a source whose bytes have another digest, naming the digest they have, and a
# record without a sha256 digest.
sources:fetch() {
  local name=$1 url=$2 digest=$3 cache fresh have
  if [[ ! "${digest}" =~ ^sha256:[0-9a-f]{64}$ ]]
  then
    error "the source ${name} has no digest: pin it as digest: sha256:<hex> under sources: in .kit/settings.yml"
    return 1
  fi
  cache=$(sources:cache "${digest}")
  if [[ -d "${cache}" ]]
  then
    printf '%s\n' "${cache}"
    return 0
  fi
  fresh=$(mktemp -d)
  step "Fetching the source ${name} from ${url}"
  if ! curl --fail --silent --location --output "${fresh}/source.tar.gz" "$(sources:tarball "${url}")"
  then
    rm -rf "${fresh}"
    error "the source ${name} could not be fetched from ${url}"
    return 1
  fi
  have=$(sources:digest "${fresh}/source.tar.gz")
  if [[ "${have}" != "${digest}" ]]
  then
    rm -rf "${fresh}"
    error "the source ${name} has digest ${have}, not the ${digest} .kit/settings.yml pins; check it, then pin it"
    return 1
  fi
  mkdir -p "${fresh}/tree"
  if ! tar --extract --gzip --file "${fresh}/source.tar.gz" --directory "${fresh}/tree" --strip-components 1
  then
    rm -rf "${fresh}"
    error "the source ${name} is not a gzipped tarball: ${url}"
    return 1
  fi
  mkdir -p "$(dirname "${cache}")"
  mv "${fresh}/tree" "${cache}"
  rm -rf "${fresh}"
  printf '%s\n' "${cache}"
}

# sources:fetch-all fetches every source the settings name that is not in the cache yet. It returns 1 when one fails.
sources:fetch-all() {
  local name url digest
  while IFS=$'\x1f' read -r name url digest
  do
    [[ -n "${name}" ]] || continue
    sources:fetch "${name}" "${url}" "${digest}" >/dev/null || return 1
  done < <(sources:list)
}

# sources:roots prints "<name>\t<root>" for each source in the cache, in the settings' order: <root> is the directory
# that holds its packages/. A source not fetched yet is left out; sources:fetch-all fetches it.
sources:roots() {
  local name url digest cache
  while IFS=$'\x1f' read -r name url digest
  do
    [[ -n "${name}" ]] || continue
    cache=$(sources:cache "${digest}")
    if [[ -d "${cache}/packages" ]]
    then
      printf '%s\t%s\n' "${name}" "${cache}"
    elif [[ -d "${cache}/kit/packages" ]]
    then
      printf '%s\t%s\n' "${name}" "${cache}/kit"
    fi
  done < <(sources:list)
}

# ----------------------------------------------------------------------------------------------------------------------
