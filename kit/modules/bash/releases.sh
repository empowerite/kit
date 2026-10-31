# shellcheck shell=bash
# releases.sh - the repository's releases: the rules a release tag keeps, the tagging and its push, and the list (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# A release is a tag, release/<version>, where <version> is <major>.<minor>.<patch>, with -<suffix> for a
# pre-release. kit release makes one and pushes it, which is how a release starts: the repository's release
# workflow, if it has one, does the rest. kit releases lists what the forge made of them.
#
# see: modules/jq/version.jq, docs/features.md

use git release forge status

# releases:valid <version> says whether the version has the shape a release tag needs.
releases:valid() {
  [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]
}

# releases:on-origin prints the version of every release tag on origin, one per line, those of a valid shape alone.
releases:on-origin() {
  local ref version
  while read -r _ ref
  do
    version=${ref#refs/tags/release/}
    version=${version%^\{\}}
    releases:valid "${version}" && printf '%s\n' "${version}"
  done < <(git ls-remote --tags origin 'refs/tags/release/*' 2>/dev/null) | LC_ALL=C sort --unique
}

# releases:release <version> [lower] tags HEAD as release/<version> and pushes that tag alone, after the rules: the
# version's shape (2 when it is wrong), a tag made once, a release on the default branch at origin, and a version
# above every release tag on origin unless lower is given. It returns 1 on any other refusal, and leaves no tag
# behind when the push fails.
releases:release() {
  local version=$1 lower=${2:-} tag default highest sends subject
  local -a sign=() versions=()
  if ! releases:valid "${version}"
  then
    error "'${version}' is not a version: <major>.<minor>.<patch>, with -<suffix> for a pre-release; run: kit help release"
    return 2
  fi
  tag=release/${version}
  if git rev-parse --quiet --verify "refs/tags/${tag}" >/dev/null || [[ -n "$(git ls-remote --tags origin "refs/tags/${tag}")" ]]
  then
    error "${tag} exists already; a version is tagged once"
    return 1
  fi
  if [[ "${version}" != *-* ]]
  then
    default=$(git:default-branch)
    git fetch --quiet origin "${default}" 2>/dev/null || true
    if ! git merge-base --is-ancestor HEAD "origin/${default}" 2>/dev/null
    then
      error "${version} is a release, and HEAD is not on ${default} at origin; tag a release there, or give a" \
        "pre-release a suffix"
      return 1
    fi
  fi
  readarray -t versions < <(releases:on-origin)
  if [[ -z "${lower}" ]] && ! jq --null-input --exit-status -L "${kit_jq}" --arg v "${version}" \
      'import "version" as version; $v | version::higher($ARGS.positional)' --args "${versions[@]}" >/dev/null
  then
    highest=$(jq --null-input --raw-output -L "${kit_jq}" 'import "version" as version; $ARGS.positional | version::highest' \
      --args "${versions[@]}")
    error "${version} is not higher than ${highest}, the highest release; run: kit release ${version} --lower"
    return 1
  fi
  [[ "$(release:setting require_signing)" == true ]] && sign=(--sign)
  git tag --annotate "${sign[@]}" --message "release ${version}" "${tag}" || return 1
  sends=$(git rev-list --count HEAD --not --remotes=origin)
  (( sends == 0 )) || step "the push sends ${sends} commit(s) no branch on origin has, reachable from ${tag} alone"
  if ! git push --quiet origin "refs/tags/${tag}"
  then
    git tag --delete "${tag}" >/dev/null
    error "${tag} could not be pushed, and is deleted here again; see above"
    return 1
  fi
  subject=$(git log -1 --format=%s HEAD)
  ok "pushed ${tag}, at $(git rev-parse --short HEAD) ${subject}"
}

# releases:render [--json] prints the forge's releases, newest first, one row each: the version, the kind with its
# glyph and the age, and the latest marked. With --json it prints the documents as one JSON array.
releases:render() {
  local json=false arg docs width version kind latest age glyph
  for arg in "$@"
  do
    [[ "${arg}" == --json ]] && json=true
  done
  forge:require || return 1
  docs=$(forge:releases) || return 1
  if [[ "${json}" == true ]]
  then
    jq --slurp --compact-output . <<<"${docs}"
    return 0
  fi
  if [[ -z "${docs}" ]]
  then
    step "no release yet; run: kit release <version>"
    return 0
  fi
  width=$(jq --slurp '[.[].version | length] | max' <<<"${docs}")
  while IFS=$'\t' read -r version kind latest age
  do
    case "${kind}" in
      (draft) glyph='📝' ;;
      (pre-release) glyph='🧪' ;;
      (*) glyph='📦' ;;
    esac
    printf '%s  %s %s %s%s\n' "$(status:cell "${version}" "${width}")" "${glyph}" "${kind}" \
      "$(status:paint 2 "$(status:age "${age}")")" "$([[ "${latest}" == true ]] && printf '  ⭐ latest')" >&2
  done < <(jq --raw-output '[.version, .kind, .latest, .age] | @tsv' <<<"${docs}")
}

# ----------------------------------------------------------------------------------------------------------------------
