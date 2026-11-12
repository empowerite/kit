# shellcheck shell=bash
# artifactory.sh - the artifactory source: a repository of an Artifactory server, by the protocol it speaks
# ----------------------------------------------------------------------------------------------------------------------
# An Artifactory repository speaks one of the protocols kit has a kind for, its type:, and this source asks it as that
# kind does, at the repository's own address on the server url: names:
#
#   github-releases   <url>/artifactory/<repository>, a remote of github.com laid out as GitHub's downloads are
#   goproxy           <url>/artifactory/api/go/<repository>, a Go remote or virtual repository
#   npm               <url>/artifactory/api/npm/<repository>, an npm remote or virtual repository
#   dist              <url>/artifactory/<repository>, a generic remote of a project's download directory
#
# It serves what its type serves, and its hashes and its versions are its type's. A download Artifactory refuses with
# 403, as Xray's blocking policy or a refused approval refuses one, is blocked: Artifactory says it never will serve
# it. It has no public home, so direct never stands for it, and its settings are its type's, for go and npm. It takes
# repository: and type:, both needed.

sources/artifactory:fetch() {
  local type base status=0
  sources/artifactory:resolve "$1" type base || return 1
  sources:call "${type}" fetch "${base}" "$2" "$3" || status=$?
  if (( status == 4 )) && [[ "${sources_http:-}" == 403 ]]
  then
    return 5
  fi
  return "${status}"
}

sources/artifactory:hash() {
  local type
  type=$(jq --raw-output '.type // ""' <<<"${sources_entry:-{\}}")
  sources:call "${type:-github-releases}" hash "$@"
}

sources/artifactory:home() { :; }

sources/artifactory:keys() { printf 'repository\ntype\n'; }

# sources/artifactory:serves takes every form its types serve, and fetch passes by one its own type does not: the
# entry is not known here.
sources/artifactory:serves() {
  case "$1" in
    (release:* | archive:* | module:* | npm:* | dist:*) return 0 ;;
    (*) return 1
  esac
}

sources/artifactory:settings() {
  local type base
  [[ "$1" != off ]] || return 0
  sources_entry=$2 sources/artifactory:resolve "$1" type base || return 1
  sources:call "${type}" settings "${base}" "$2"
}

sources/artifactory:versions() {
  local type base
  sources/artifactory:resolve "$1" type base || return 1
  sources:call "${type}" versions "${base}" "$2"
}

# The seam's calls end here; what follows is this provider's own.

# sources/artifactory:resolve <url> <type> <base> sets the two variables to the entry's type, sources_entry's, and
# the address its repository is asked at as that type, and refuses an entry with no repository or with a type kit
# has no kind for.
sources/artifactory:resolve() {
  local repository kind
  local -n sources_artifactory_type=$2 sources_artifactory_base=$3
  repository=$(jq --raw-output '.repository // ""' <<<"${sources_entry:-{\}}")
  kind=$(jq --raw-output '.type // ""' <<<"${sources_entry:-{\}}")
  if [[ -z "${repository}" ]]
  then
    error "an artifactory source needs repository:, the repository it asks"
    return 1
  fi
  # shellcheck disable=SC2034  # the caller's variables, through the names
  case "${kind}" in
    (github-releases | dist) sources_artifactory_base="${1%/}/artifactory/${repository}" ;;
    (goproxy) sources_artifactory_base="${1%/}/artifactory/api/go/${repository}" ;;
    (npm) sources_artifactory_base="${1%/}/artifactory/api/npm/${repository}" ;;
    (*)
      error "an artifactory source's type: is github-releases, goproxy, npm, or dist, not '${kind}'"
      return 1
  esac
  # shellcheck disable=SC2034  # the caller's variable, through the name
  sources_artifactory_type=${kind}
}

# ----------------------------------------------------------------------------------------------------------------------
