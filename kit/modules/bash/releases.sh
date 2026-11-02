# shellcheck shell=bash
# releases.sh - the repository's releases: a release tag's rules, the tagging and its push, the list (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
#
# A release is a tag, release/<version>, where <version> is <major>.<minor>.<patch>, with -<suffix> for a
# pre-release. kit release makes one and pushes it, which is how a release starts: the repository's release
# workflow, if it has one, does the rest. The releases stage of kit flow lists what the forge made of them.
#
# see: kit/bin/kit-release, kit/bin/kit-flow, modules/jq/version.jq, docs/features.md

use git forge flow settings

# releases:valid <version> says whether the version has the shape a release tag needs.
releases:valid() {
  [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]
}

# releases:on-origin prints the version of every release tag on the primary remote, one per line, those of a valid
# shape alone.
releases:on-origin() {
  local ref version primary refs
  primary=$(forge:primary) || return $?
  refs=$(git ls-remote --tags --end-of-options "${primary}" 'refs/tags/release/*' 2>/dev/null || true)
  while read -r _ ref
  do
    version=${ref#refs/tags/release/}
    version=${version%^\{\}}
    releases:valid "${version}" || continue
    printf '%s\n' "${version}"
  done <<<"${refs}" | LC_ALL=C sort --unique
}

# releases:release <version> [lower] tags HEAD as release/<version> and pushes that tag alone, to the primary remote,
# after the rules: the version's shape (2 when it is wrong), a tag made once, a release on the default branch at the
# primary, and a version above every release tag there unless lower is given. It returns 1 on any other refusal,
# and leaves no tag behind when the push fails.
releases:release() {
  local version=$1 lower=${2:-} tag default highest sends subject primary text
  local -a sign=() versions=()
  if ! releases:valid "${version}"
  then
    error "'${version}' is not a version: <major>.<minor>.<patch>, with -<suffix> for a pre-release;" \
      "run: kit help release"
    return 2
  fi
  tag=release/${version}
  primary=$(forge:primary) || return $?
  if git rev-parse --quiet --verify "refs/tags/${tag}" >/dev/null \
    || [[ -n "$(git ls-remote --tags --end-of-options "${primary}" "refs/tags/${tag}")" ]]
  then
    error "${tag} exists already; a version is tagged once"
    return 1
  fi
  if [[ "${version}" != *-* ]]
  then
    default=$(git:default-branch)
    git fetch --quiet --end-of-options "${primary}" "${default}" 2>/dev/null || true
    if ! git merge-base --is-ancestor HEAD "${primary}/${default}" 2>/dev/null
    then
      error "${version} is a release, and HEAD is not on ${default} at ${primary}; tag a release there, or give a" \
        "pre-release a suffix"
      return 1
    fi
  fi
  text=$(releases:on-origin) || return $?
  [[ -z "${text}" ]] || readarray -t versions <<<"${text}"
  if [[ -z "${lower}" ]] && ! jq --null-input --exit-status -L "${kit_jq}" --arg v "${version}" \
      'import "version" as version; $v | version::higher($ARGS.positional)' --args "${versions[@]}" >/dev/null
  then
    highest=$(jq --null-input --raw-output -L "${kit_jq}" \
      'import "version" as version; $ARGS.positional | version::highest' \
      --args "${versions[@]}")
    error "${version} is not higher than ${highest}, the highest release; run: kit release ${version} --lower"
    return 1
  fi
  subject=$(git log -1 --format=%s HEAD)
  step "tagging $(git rev-parse --short HEAD) ${subject} as ${tag}"
  [[ "$(settings:value require_signing)" == true ]] && sign=(--sign)
  git tag --annotate "${sign[@]}" --message "release ${version}" "${tag}" || return 1
  sends=$(git rev-list --count HEAD --not --remotes="${primary}")
  (( sends == 0 )) || step "the push sends ${sends} commit(s) no branch on ${primary} has, reachable from ${tag} alone"
  if ! git push --quiet --end-of-options "${primary}" "refs/tags/${tag}"
  then
    git tag --delete "${tag}" >/dev/null
    error "${tag} could not be pushed, and is deleted here again; see above"
    return 1
  fi
  ok "pushed ${tag}"
}

# releases:documents prints the forge's releases, newest first, as one JSON array: version, tag, name, kind, latest,
# at, and age in seconds, each.
releases:documents() {
  local docs
  forge:require || return 1
  docs=$(forge:releases) || return 1
  jq --slurp --compact-output . <<<"${docs}"
}

# releases:draw <releases-json> prints the releases, one row each, to stderr: the version, the kind with its glyph
# and the age, and the latest marked. With none it says so in one dim line.
releases:draw() {
  local doc=$1 width kind_width age_width version kind latest age glyph
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no release yet'
    return 0
  fi
  width=$(jq '[.[].version | length] | max' <<<"${doc}")
  kind_width=$(jq '[.[].kind | length] | max' <<<"${doc}")
  age_width=$(while read -r age; do flow:age "${age}"; echo; done < <(jq --raw-output '.[].age' <<<"${doc}") \
    | awk 'length > w { w = length } END { print w + 0 }')
  while IFS=$'\t' read -r version kind latest age
  do
    case "${kind}" in
      (draft) glyph='📝' ;;
      (pre-release) glyph='🧪' ;;
      (*) glyph='📦' ;;
    esac
    # the age is a number, so it is right-aligned, as kit flow draws one
    printf '%s  %s %s %s%s\n' "$(flow:cell "${version}" "${width}")" "${glyph}" \
      "$(flow:cell "${kind}" "${kind_width}")" \
      "$(flow:paint 2 "$(flow:cell "$(flow:age "${age}")" "${age_width}" right)")" \
      "$([[ "${latest}" == true ]] && printf '  ⭐ latest')" >&2
  done < <(jq --raw-output '.[] | [.version, .kind, .latest, .age] | @tsv' <<<"${doc}")
}

# releases:render [--json] prints the releases as releases:draw does, or with --json as releases:documents does, to
# stdout.
releases:render() {
  local doc
  doc=$(releases:documents) || return 1
  if [[ " $* " == *" --json "* ]]
  then
    printf '%s\n' "${doc}"
    return 0
  fi
  releases:draw "${doc}"
}

# ----------------------------------------------------------------------------------------------------------------------
