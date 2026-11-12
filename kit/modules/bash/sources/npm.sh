# shellcheck shell=bash
# npm.sh - the npm source: a registry that serves each package's versions as tarballs (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# An npm: coordinate, npm:<package>@<version>, is the tarball the package's document at <url>/<package> names for the
# version, under dist.tarball, a scoped package's / written %2f. Its hash is npm's integrity, sha512- and the sha512
# of the tarball in base64. The versions are the document's.
#
# It takes scope:, as "@corp", for a registry that serves that scope alone: it serves no package outside it, and
# settings writes it as the scope's registry, npm_config_@corp:registry, where any other is npm_config_registry.

sources/npm:fetch() {
  local package version doc tarball status=0
  sources/npm:split "$2" package version || return 4
  doc=$(sources/npm:document "$1" "${package}") || return $?
  # a document that is not JSON names no tarball, which the next line takes as not served
  tarball=$(jq --raw-output --arg v "${version}" '.versions[$v].dist.tarball // empty' <<<"${doc}" 2>/dev/null) || true
  [[ -n "${tarball}" ]] || return 4
  sources:get "${tarball}" "$3" || status=$?
  return "${status}"
}

sources/npm:hash() {
  printf 'sha512-%s\n' "$(shasum -a 512 "$1" | cut -d ' ' -f 1 \
    | perl -MMIME::Base64 -ne 'chomp; print encode_base64(pack("H*", $_), "")')"
}

sources/npm:home() { printf 'https://registry.npmjs.org\n'; }

sources/npm:keys() { printf 'scope\n'; }

sources/npm:serves() { [[ "$1" == npm:* ]]; }

# sources/npm:settings writes nothing for off: npm has no word for a path that ends, so a registry before it is the
# last npm asks.
sources/npm:settings() {
  local scope
  [[ "$1" != off ]] || return 0
  scope=$(jq --raw-output '.scope // ""' <<<"${2:-{\}}")
  if [[ -n "${scope}" ]]
  then printf 'npm_config_%s:registry=%s\n' "${scope}" "$1"
  else printf 'npm_config_registry=%s\n' "$1"
  fi
}

sources/npm:versions() {
  local package version doc
  sources/npm:split "$2" package version || return 4
  doc=$(sources/npm:document "$1" "${package}") || return $?
  jq --raw-output '(.versions // {}) | keys_unsorted[]' <<<"${doc}" 2>/dev/null || return 1
}

# The seam's calls end here; what follows is this provider's own.

# sources/npm:split <coordinate> <package> <version> sets the two variables to the package and the version an npm:
# coordinate names, the version empty where it names none, and returns 1 for a coordinate of another form.
sources/npm:split() {
  local rest=${1#npm:}
  local -n sources_npm_package=$2 sources_npm_version=$3
  [[ "$1" == npm:?* ]] || return 1
  # shellcheck disable=SC2034  # the caller's variables, through the names
  if [[ "${rest:1}" == *@* ]]
  then
    sources_npm_version=${rest##*@}
    sources_npm_package=${rest%@*}
  else
    sources_npm_version=''
    sources_npm_package=${rest}
  fi
}

# sources/npm:document <url> <package> prints the package's document from the registry. A package outside the scope
# the entry names, sources_entry's, is not served: it returns 4.
sources/npm:document() {
  local scope status=0
  scope=$(jq --raw-output '.scope // ""' <<<"${sources_entry:-{\}}")
  [[ -z "${scope}" || "$2" == "${scope}"/* ]] || return 4
  sources:get "$1/${2//\//%2f}" - || status=$?
  return "${status}"
}

# ----------------------------------------------------------------------------------------------------------------------
