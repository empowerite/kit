#!/usr/bin/env bash
# kit-release - verify a release tag, pack a checkout as a release, publish it, and retire pre-releases (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The kit releasing itself, as verbs a workflow step or a desk calls alike:
#
#   verify <tag>              the tag is v<major>.<minor>.<patch>, on the default branch, or the same with a
#                             -<suffix>, a pre-release, on a commit off it; refused otherwise, naming why
#   pack <tag> [<directory>]  kit-<version>.tar.gz and its .sha256 in <directory>, the current one by default:
#                             one directory kit-<version>/ with bin/, modules/, and the version stamped in
#   publish <tag>             pack, then a GitHub release carrying both files, marked pre-release for a suffix
#   retire [<branch>]         delete every pre-release whose commit no remaining branch reaches, after <branch>
#                             was deleted; the branch is the runner's delete event when none is given
#
# A release is permanent and immutable; only a pre-release is ever retired.
#
# see: .github/workflows/release.yml, .github/workflows/retire-pre-releases.yml

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use git release

usage() { echo "usage: kit release verify <tag> | pack <tag> [<directory>] | publish <tag> | retire [<branch>]" >&2; }

# release-tag:refuse <message> is the one refusal: the message, the runner's error line, the summary, exit 1.
release-tag:refuse() {
  error "$1"
  summary "### ❌ $1"
  exit 1
}

# release-tag:version <tag> prints the version a tag names, refusing a tag that is not a version.
release-tag:version() {
  local version=${1#v}
  if [[ "$1" != v* || ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]
  then
    release-tag:refuse "$1 is not a version tag: v<major>.<minor>.<patch>, with -<suffix> for a pre-release"
  fi
  printf '%s\n' "${version}"
}

verb=${1:-}
case "${verb}" in
  (--help | '') usage; exit 0 ;;
  (verify | pack | publish | retire) ;;
  (*) usage; exit 2
esac
shift

case "${verb}" in
  (verify)
    (( $# == 1 )) || { usage; exit 2; }
    tag=$1
    version=$(release-tag:version "${tag}")
    default_branch=$(git:default-branch)
    on_default=false
    git merge-base --is-ancestor "${tag}" "origin/${default_branch}" 2>/dev/null && on_default=true
    if [[ "${version}" == *-* && "${on_default}" == true ]]
    then
      release-tag:refuse "${tag} is a pre-release tag on ${default_branch}; tag a pre-release on a commit off it"
    fi
    if [[ "${version}" != *-* && "${on_default}" == false ]]
    then
      release-tag:refuse "${tag} is not on ${default_branch}; a release is tagged there, a pre-release has a suffix"
    fi
    ok "${tag} is a version tag on the right side of ${default_branch} (on it: ${on_default})"
    summary "✅ ${tag} is a version tag on the right side of ${default_branch}."
    ;;
  (pack)
    (( $# >= 1 && $# <= 2 )) || { usage; exit 2; }
    version=$(release-tag:version "$1")
    into=${2:-.}
    staging=$(mktemp -d)
    # shellcheck disable=SC2064  # expanded now on purpose: the path is fixed from here on
    trap "rm -rf '${staging}'" EXIT
    mkdir -p "${into}" "${staging}/kit-${version}"
    cp -R "${kit_root}/bin" "${kit_root}/modules" "${staging}/kit-${version}/"
    printf '%s\n' "${version}" > "${staging}/kit-${version}/version"
    tar --create --gzip --file "${into}/kit-${version}.tar.gz" --directory "${staging}" "kit-${version}"
    printf '%s  kit-%s.tar.gz\n' "$(release:digest < "${into}/kit-${version}.tar.gz")" "${version}" \
      > "${into}/kit-${version}.tar.gz.sha256"
    ok "packed kit ${version}: ${into}/kit-${version}.tar.gz and its .sha256"
    ;;
  (publish)
    (( $# == 1 )) || { usage; exit 2; }
    tag=$1
    version=$(release-tag:version "${tag}")
    guard:require tool:gh
    into=$(mktemp -d)
    # shellcheck disable=SC2064  # expanded now on purpose: the path is fixed from here on
    trap "rm -rf '${into}'" EXIT
    "${kit_root}/bin/kit" release pack "${tag}" "${into}"
    notes=$(printf 'Pin it in mise.toml:\n\n    [tools]\n    "github:empowerite/kit" = "%s"\n\n' "${version}"
      printf 'Then, in the repository: kit install\n')
    flags=()
    [[ "${version}" == *-* ]] && flags+=(--prerelease)
    gh release create "${tag}" "${into}/kit-${version}.tar.gz" "${into}/kit-${version}.tar.gz.sha256" \
      --title "kit ${version}" --notes "${notes}" "${flags[@]}"
    ok "released kit ${version}"
    summary "✅ Released kit ${version}."
    ;;
  (retire)
    (( $# <= 1 )) || { usage; exit 2; }
    deleted=${1:-}
    [[ -n "${deleted}" || -z "${GITHUB_EVENT_PATH:-}" ]] \
      || deleted=$(jq --raw-output '.ref // empty' "${GITHUB_EVENT_PATH}")
    guard:require tool:gh
    git fetch --quiet --tags origin
    retired=0
    while IFS= read -r tag
    do
      [[ -n "${tag}" ]] || continue
      commit=$(git rev-parse --verify --quiet "${tag}^{commit}" || true)
      if [[ -z "${commit}" ]]
      then
        warn "${tag} has no commit here; leaving it"
        continue
      fi
      if git branch --remotes --contains "${commit}" | grep --quiet --invert-match 'origin/HEAD'
      then
        step "keeping ${tag}: a branch still reaches ${commit:0:7}"
        continue
      fi
      gh release delete "${tag}" --cleanup-tag --yes
      step "retired ${tag}: no branch reaches ${commit:0:7}${deleted:+ since ${deleted} was deleted}"
      retired=$(( retired + 1 ))
    done < <(gh release list --limit 1000 --json tagName,isPrerelease --jq '.[] | select(.isPrerelease) | .tagName')
    ok "retired ${retired} pre-release(s)"
    summary "✅ Retired ${retired} pre-release(s)."
esac

# ----------------------------------------------------------------------------------------------------------------------
