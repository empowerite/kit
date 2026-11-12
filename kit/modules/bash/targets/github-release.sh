# shellcheck shell=bash
# github-release.sh - the github-release target: a GitHub release of the version's tag, made with gh (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# The release is of the tag release/<version>, which kit release pushed, on the repository gh resolves here, or the
# one repo: names, as owner/repo. Its files are the artifact's, its title the artifact's name and the version, and a
# version with a suffix is marked a pre-release. retire deletes the release, and a pre-release's tag with it.

targets/github-release:connector() { printf 'gh\n'; }

targets/github-release:keys() { printf 'repo\n'; }

targets/github-release:push() {
  local version=$1 target=$2 name
  shift 2
  local -a flags=()
  name=$(jq --raw-output '.artifact.name // "release"' <<<"${target}")
  [[ "${version}" != *-* ]] || flags+=(--prerelease)
  targets/github-release:repo "${target}" flags
  gh release create "release/${version}" "$@" --verify-tag --title "${name} ${version}" \
    --notes "${name} ${version}" "${flags[@]}"
}

targets/github-release:pushed() {
  local -a flags=()
  targets/github-release:repo "$2" flags
  gh release view "release/$1" "${flags[@]}" --json tagName >/dev/null 2>&1
}

targets/github-release:retire() {
  local -a flags=(--yes)
  [[ "$1" != *-* ]] || flags+=(--cleanup-tag)
  targets/github-release:repo "$2" flags
  gh release delete "release/$1" "${flags[@]}"
}

# The seam's calls end here; what follows is this provider's own.

# targets/github-release:repo <target> <array> adds --repo=<repo> to the array where the target names a repo.
targets/github-release:repo() {
  local repo
  local -n targets_github_flags=$2
  repo=$(jq --raw-output '.repo // ""' <<<"$1")
  [[ -z "${repo}" ]] || targets_github_flags+=("--repo=${repo}")
}

# ----------------------------------------------------------------------------------------------------------------------
