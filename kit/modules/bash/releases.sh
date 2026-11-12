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

use git forge flow settings workspaces

# releases:valid <version> says whether the version has the shape a release tag needs.
releases:valid() {
  [[ "$1" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]
}

# releases:on-origin prints the version of every release tag on the primary remote, one per line, those of a valid
# shape alone.
releases:on-origin() {
  local ref version primary refs
  primary=$(forge:primary) || return $?
  if ! refs=$(git ls-remote --tags --end-of-options "${primary}" 'refs/tags/release/*')
  then
    error "run: git ls-remote --tags ${primary} (the release tags on ${primary} could not be read)"
    return 1
  fi
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
    default=$(git:default-branch) || return 1
    # A fetch that fails leaves the default branch as last fetched, and a HEAD past it is refused below.
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

# releases:documents prints the forge's releases, newest first, as one JSON array of rows, one a release, flat, as
# kittui's catalog draws them: key, its tag; version, tag, name, kind, at, age in seconds, url, and notes, as the
# forge seam gives them; latest, "latest" for the forge's latest release, else null; assets, the name of each file it
# offers, and downloads, where each downloads; and links, where a value leads, by its field: the kind and the url to
# the release's page, and each asset and each download to where that one downloads. In a repository whose kit.yml
# names artifacts under publish:, each of the releases_visible newest that is no draft carries visibility, its state
# on each pull source that checks it, as visibility:release prints them, and every other row null.
releases:documents() {
  local docs rows
  forge:require || return 1
  # A forge without releases has none to list.
  docs=$(forge:releases) || (( $? == 3 )) || return 1
  rows=$(jq --slurp --compact-output 'map({key: .tag} + . + {
      latest: (if .latest then "latest" else null end),
      assets: [(.assets // [])[].name], downloads: [(.assets // [])[].url],
      links: {kind: .url, url: .url, assets: [(.assets // [])[].url], downloads: [(.assets // [])[].url]}})' \
    <<<"${docs}") || return 1
  releases:visible "${rows}"
}

# releases_visible is how many of the newest releases releases:documents reads the visibility of.
declare -g releases_visible=5

# releases:visible <rows> prints the rows with visibility added, as releases:documents says, null on every row where
# kit.yml names no artifact under publish:.
releases:visible() {
  local rows=$1 version age states path='' pair seen='[]'
  local -a pairs
  if ! grep --quiet '^publish:' kit.yml 2>/dev/null
  then
    jq --compact-output 'map(. + {visibility: null})' <<<"${rows}"
    return
  fi
  use visibility
  # shellcheck disable=SC2016  # the $ names are jq's own, which jq binds
  list:read pairs jq --raw-output --argjson n "${releases_visible}" \
    '[.[] | select(.kind != "draft")][:$n][] | "\(.version)\t\(.age // 0)"' <<<"${rows}" || return 1
  for pair in "${pairs[@]}"
  do
    [[ -n "${pair}" ]] || continue
    IFS=$'\t' read -r version age <<<"${pair}"
    [[ -n "${path}" ]] || path=$(sources:path) || return 1
    states=$(visibility:release "${version}" "${age}" "${path}") || return 1
    seen=$(jq --slurp --compact-output --arg v "${version}" --argjson seen "${seen}" \
      '$seen + [{version: $v, states: .}]' <<<"${states}") || return 1
  done
  jq --compact-output --argjson seen "${seen}" '
    ($seen | map({(.version): .states}) | add // {}) as $by
    | map(. + {visibility: ($by[.version] // null | if . == [] then null else . end)})' <<<"${rows}"
}

# releases:draw <releases-json> prints the releases, one row each, to stderr: the version, the kind with its glyph
# and the age, and the latest marked. With none it says so in one dim line.
releases:draw() {
  local doc=$1 width kind_width age_width version kind latest age glyph seen line
  local -a stuck
  if [[ "$(jq 'length' <<<"${doc}")" == 0 ]]
  then
    flow:none 'no release yet'
    return 0
  fi
  width=$(jq '[.[].version | length] | max' <<<"${doc}")
  kind_width=$(jq '[.[].kind | length] | max' <<<"${doc}")
  age_width=$(while read -r age; do flow:age "${age}"; echo; done < <(jq --raw-output '.[].age' <<<"${doc}") \
    | awk 'length > w { w = length } END { print w + 0 }')
  # each row's last field is its pull path's state, empty where it has none, so the fields are apart by \x1f
  while IFS=$'\x1f' read -r version kind latest age seen
  do
    case "${kind}" in
      (draft) glyph=${flow_glyphs[Draft]} ;;
      (pre-release) glyph=${flow_glyphs[Pre]} ;;
      (*) glyph=${flow_glyphs[Release]} ;;
    esac
    # the age is a number, so it is right-aligned, as kit flow draws one
    printf '%s  %s %s %s%s%s\n' "$(flow:cell "${version}" "${width}")" "${glyph}" \
      "$(flow:cell "${kind}" "${kind_width}")" \
      "$(flow:paint 2 "$(flow:cell "$(flow:age "${age}")" "${age_width}" right)")" \
      "$([[ "${latest}" == latest ]] && printf '  %s latest' "${flow_glyphs[Latest]}")" \
      "$([[ -z "${seen}" ]] || printf '  %s' "$(flow:paint 2 "${seen}")")" >&2
  done < <(jq --raw-output '.[] | [.version, .kind, .latest, .age, (.visibility // [] | if length == 0 then ""
      elif all(.state == "visible") then "pull: visible"
      else "pull: " + (map(select(.state != "visible") | "\(.state) on \(.source)") | join(", ")) end)]
    | map(tostring) | join("\u001f")' <<<"${doc}")
  # shellcheck disable=SC2016  # the $ names are jq's own, which jq binds
  list:read stuck jq --raw-output '.[] | .version as $v | (.visibility // [])[] | select(.state == "stuck")
    | "\($v): \(.artifact) is stuck on \(.source), \(.detail)"' <<<"${doc}" || return 1
  for line in "${stuck[@]}"
  do
    [[ -z "${line}" ]] || warn "${line}"
  done
}

# releases:render [--json] prints the releases as releases:draw does, or with --json as releases:documents does, to
# stdout, each saying where it is from, the forge, as workspaces:placed says.
releases:render() {
  local doc
  doc=$(releases:documents) || return 1
  if [[ " $* " == *" --json "* ]]
  then
    workspaces:placed forge "${doc}"
    return
  fi
  releases:draw "${doc}"
}

# releases:wait <version> <seconds> waits for visibility: until each pull source that checks an artifact serves the
# version, one blocks it or serves other bytes, or the seconds pass, asking every KIT_RELEASE_POLL seconds, 30 by
# default, never from what is kept. It returns 0 when every one serves it, 1 when one blocks it or serves other bytes,
# or there is nothing to watch, and 3 when the time passes with one still pending.
releases:wait() {
  local version=$1 seconds=$2 poll=${KIT_RELEASE_POLL:-30} start now states summary path tagged
  use visibility
  # shellcheck disable=SC2034  # read by visibility:kept: a wait asks again every time
  visibility_pending_for=0
  path=$(sources:path) || return 1
  start=$(date +%s)
  # the age a source's expect: is held to runs from the tag, where it is here
  tagged=$(git for-each-ref --format='%(creatordate:unix)' "refs/tags/release/${version}")
  [[ -n "${tagged}" ]] || tagged=${start}
  while :
  do
    now=$(date +%s)
    states=$(visibility:release "${version}" $(( now - tagged )) "${path}") || return 1
    if [[ -z "${states}" ]]
    then
      error "kit.yml names no artifact under publish: that the pull path can serve, so there is nothing to wait for"
      return 1
    fi
    summary=$(visibility:summary "${states}")
    if jq --slurp --exit-status 'all(.state == "visible")' <<<"${states}" >/dev/null
    then
      ok "${version} is ${summary}"
      return 0
    fi
    if jq --slurp --exit-status 'any(.state == "blocked" or .state == "mismatch")' <<<"${states}" >/dev/null
    then
      error "${version}: ${summary}"
      return 1
    fi
    if (( now - start + poll > seconds ))
    then
      error "${version} is not served everywhere after $(visibility:age "${seconds}"): ${summary}"
      return 3
    fi
    step "${version}: ${summary}; asking again in ${poll}s"
    sleep "${poll}"
  done
}

# ----------------------------------------------------------------------------------------------------------------------
