# shellcheck shell=bash
# npm-publish.sh - the npm-publish target: the package's tarball published to a registry by npm (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# The artifact names its package with npm:, and its file is the tarball npm pack made. npm publishes it to registry:,
# npm's own registry by default, a version with a suffix under the dist-tag next, so latest is never a pre-release.
# The runner carries the credentials, as NODE_AUTH_TOKEN or .npmrc. retire deprecates the version: a registry keeps
# what it served.

targets/npm-publish:connector() { printf 'npm\n'; }

targets/npm-publish:keys() { printf 'registry\n'; }

targets/npm-publish:push() {
  local version=$1 target=$2 file
  shift 2
  local -a flags=()
  targets/npm-publish:registry "${target}" flags
  [[ "${version}" != *-* ]] || flags+=(--tag=next)
  for file in "$@"
  do
    npm publish "${file}" "${flags[@]}" || return 1
  done
}

targets/npm-publish:pushed() {
  local package served
  local -a flags=()
  package=$(targets/npm-publish:package "$2") || return 1
  targets/npm-publish:registry "$2" flags
  served=$(npm view "${package}@$1" version "${flags[@]}" 2>/dev/null) || return 1
  [[ "${served}" == "$1" ]]
}

targets/npm-publish:retire() {
  local package
  local -a flags=()
  package=$(targets/npm-publish:package "$2") || return 1
  targets/npm-publish:registry "$2" flags
  npm deprecate "${package}@$1" "retired" "${flags[@]}"
}

# The seam's calls end here; what follows is this provider's own.

# targets/npm-publish:package <target> prints the package the artifact names with npm:, and refuses one that names
# none.
targets/npm-publish:package() {
  local package
  package=$(jq --raw-output '.artifact.npm // ""' <<<"$1")
  if [[ -z "${package}" ]]
  then
    error "an npm-publish target needs its artifact to name its package with npm: under publish: in kit.yml"
    return 1
  fi
  printf '%s\n' "${package}"
}

# targets/npm-publish:registry <target> <array> adds --registry=<url> to the array where the target names one.
targets/npm-publish:registry() {
  local registry
  local -n targets_npm_flags=$2
  registry=$(jq --raw-output '.registry // ""' <<<"$1")
  # shellcheck disable=SC2034  # the caller's array, through the name
  [[ -z "${registry}" ]] || targets_npm_flags+=("--registry=${registry}")
}

# ----------------------------------------------------------------------------------------------------------------------
