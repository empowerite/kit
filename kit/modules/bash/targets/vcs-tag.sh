# shellcheck shell=bash
# vcs-tag.sh - the vcs-tag target: a tag of the version pushed to the primary remote, as a Go module is published
# ----------------------------------------------------------------------------------------------------------------------
# A Go module is published by its tag, and a proxy fetches it from there. The tag is prefix: and the version, v1.2.0 by
# default, or a module in a subdirectory's own, as prefix: tools/v. It is made on the commit release/<version> tags,
# annotated, and pushed to the primary remote alone. A pushed tag is never taken back, so retire refuses.

targets/vcs-tag:connector() { printf 'git\n'; }

targets/vcs-tag:keys() { printf 'prefix\n'; }

targets/vcs-tag:push() {
  local version=$1 tag primary
  tag=$(targets/vcs-tag:tag "$1" "$2")
  primary=$(forge:primary) || return 1
  git tag --annotate --message "${tag}" "${tag}" "release/${version}^{commit}" || return 1
  git push --quiet --end-of-options "${primary}" "refs/tags/${tag}"
}

targets/vcs-tag:pushed() {
  local tag primary refs
  tag=$(targets/vcs-tag:tag "$1" "$2")
  primary=$(forge:primary) || return 1
  refs=$(git ls-remote --tags --end-of-options "${primary}" "refs/tags/${tag}") || return 1
  [[ -n "${refs}" ]]
}

targets/vcs-tag:retire() {
  error "$(targets/vcs-tag:tag "$1" "$2") is a pushed tag, which a proxy may serve already; a tag is never taken back"
  return 1
}

# The seam's calls end here; what follows is this provider's own.

# targets/vcs-tag:tag <version> <target> prints the tag the target makes of the version: its prefix, v by default,
# and the version.
targets/vcs-tag:tag() {
  printf '%s%s\n' "$(jq --raw-output '.prefix // "v"' <<<"$2")" "$1"
}

# ----------------------------------------------------------------------------------------------------------------------
