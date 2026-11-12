# shellcheck shell=bash
# artifactory-local.sh - the artifactory-local target: files uploaded to an Artifactory local repository, by jf
# ----------------------------------------------------------------------------------------------------------------------
# jfrog-cli, jf, uploads each file to repository:, under path:, <artifact>/<version>/ by default, {name} and {version}
# filled in, with the server jf is configured for: its own config, or url: as --url. The runner carries the
# credentials, as JFROG_CLI's environment or jf config. pushed asks jf rt search for anything under the path, and
# retire deletes the path.

targets/artifactory-local:connector() { printf 'jf\n'; }

targets/artifactory-local:keys() { printf 'repository\npath\nurl\n'; }

targets/artifactory-local:push() {
  local version=$1 target=$2 file where
  shift 2
  local -a flags=()
  where=$(targets/artifactory-local:where "${version}" "${target}" flags) || return 1
  for file in "$@"
  do
    jf rt upload "${flags[@]}" "${file}" "${where}" || return 1
  done
}

targets/artifactory-local:pushed() {
  local where found
  local -a flags=()
  where=$(targets/artifactory-local:where "$1" "$2" flags) || return 1
  found=$(jf rt search "${flags[@]}" "${where}*") || return 1
  [[ "$(jq 'length' <<<"${found:-[]}" 2>/dev/null)" -gt 0 ]]
}

targets/artifactory-local:retire() {
  local where
  local -a flags=()
  where=$(targets/artifactory-local:where "$1" "$2" flags) || return 1
  jf rt delete "${flags[@]}" --quiet "${where}"
}

# The seam's calls end here; what follows is this provider's own.

# targets/artifactory-local:where <version> <target> <array> prints <repository>/<path> for the version, and adds
# --url=<url> to the array where the target names one. It refuses a target with no repository.
targets/artifactory-local:where() {
  local repository path url
  local -n targets_artifactory_flags=$3
  repository=$(jq --raw-output '.repository // ""' <<<"$2")
  if [[ -z "${repository}" ]]
  then
    error "an artifactory-local target needs repository:, the local repository it uploads to"
    return 1
  fi
  path=$(jq --raw-output --arg v "$1" '(.artifact.name // "") as $n | (.path // "{name}/{version}/")
    | gsub("\\{version\\}"; $v) | gsub("\\{name\\}"; $n)' <<<"$2") || return 1
  url=$(jq --raw-output '.url // ""' <<<"$2")
  # shellcheck disable=SC2034  # the caller's array, through the name
  [[ -z "${url}" ]] || targets_artifactory_flags+=("--url=${url}")
  printf '%s/%s\n' "${repository}" "${path}"
}

# ----------------------------------------------------------------------------------------------------------------------
