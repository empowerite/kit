# shellcheck shell=bash
# github-releases.sh - the github-releases source: files attached to a GitHub release, by its tag (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A release: coordinate, release:<owner>/<repo>@<tag>/<asset>, is served at <url>/<owner>/<repo>/releases/download/
# <tag>/<asset>: github.com itself, a GitHub Enterprise Server, or a mirror laid out as GitHub's downloads are. An
# archive: coordinate, archive:<owner>/<repo>@<tag>, is the tag's source tarball, at <url>/<owner>/<repo>/archive/
# refs/tags/<tag>.tar.gz, whose bytes GitHub keeps the same. Its
# hash is the sha256 of the file. The versions are the release's tags, read through gh where gh is installed and the
# source is github.com, and through the API otherwise: api.github.com for github.com, and <url>/api/v3 for another
# host. It takes no key of its own.

sources/github-releases:fetch() {
  local release
  if [[ "$2" == archive:* ]]
  then
    [[ "$2" =~ ^archive:([^/@]+/[^/@]+)@(.+)$ ]] || return 4
    sources:get "$1/${BASH_REMATCH[1]}/archive/refs/tags/${BASH_REMATCH[2]}.tar.gz" "$3"
    return
  fi
  release=$(sources/github-releases:split "$2") || return 4
  sources:get "$1/${release%%$'\t'*}/releases/download/${release#*$'\t'}" "$3"
}

sources/github-releases:hash() { printf 'sha256:%s\n' "$(file:digest "$1")"; }

sources/github-releases:home() { printf 'https://github.com\n'; }

sources/github-releases:keys() { :; }

sources/github-releases:serves() { [[ "$1" == release:* || "$1" == archive:* ]]; }

# sources/github-releases:settings writes nothing: no language's manager reads GitHub's releases.
sources/github-releases:settings() { :; }

# sources/github-releases:versions prints the tags of the repository's releases, a draft left out, the newest first,
# as GitHub lists them.
sources/github-releases:versions() {
  local repo doc status=0 api
  repo=${2#release:}
  repo=${repo#archive:}
  repo=${repo%%@*}
  if [[ "$1" == https://github.com ]] && command -v gh >/dev/null 2>&1
  then
    doc=$(gh api "repos/${repo}/releases?per_page=100") || return 1
  else
    api="$1/api/v3"
    [[ "$1" != https://github.com ]] || api=https://api.github.com
    doc=$(sources:get "${api}/repos/${repo}/releases?per_page=100" -) || status=$?
    (( status == 0 )) || return "${status}"
  fi
  jq --raw-output '.[] | select(.draft | not) | .tag_name' <<<"${doc}" 2>/dev/null || return 1
}

# The seam's calls end here; what follows is this provider's own.

# sources/github-releases:split <coordinate> prints "<owner>/<repo>\t<tag>/<asset>" for a release: coordinate, and
# returns 1 for one of another shape.
sources/github-releases:split() {
  local rest=${1#release:}
  [[ "${rest}" =~ ^([^/@]+/[^/@]+)@([^/]+)/(.+)$ ]] || return 1
  printf '%s\t%s/%s\n' "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}"
}

# ----------------------------------------------------------------------------------------------------------------------
