# shellcheck shell=bash
# tutorials.sh - kit's tutorials: the catalogue, each version's base, and the JSON of kit tutorials (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# The tutorials are a site, one MDX page per tutorial. Its catalogue is modules/tutorials.yml, which kit generate writes
# from the pages' front matter: the audiences, and for each tutorial its id, path, title, summary, glyph, audiences,
# and order. KIT_TUTORIALS_CATALOGUE, when set, names another file in its place.
#
# A tutorial's link is <base><path>?kit=<version>, and the base follows this kit's version:
#
#   - unreleased:   https://empowerite.github.io/kit/tutorials/main/
#   - with a suffix, as 0.5.0-rc.1:   https://empowerite.github.io/kit/tutorials/0.5.0-rc.1/
#   - X.Y.Z:   https://kit.empowerite.dev/tutorials/X.Y/
#
# KIT_TUTORIALS_BASE, when set, replaces the base. Nothing here runs a tutorial's commands.
#
use release yaml

# tutorials:catalogue prints the path of the catalogue.
tutorials:catalogue() { printf '%s\n' "${KIT_TUTORIALS_CATALOGUE:-${kit_root}/modules/tutorials.yml}"; }

# tutorials:base <version> prints the base the tutorials' links start from, ending in a slash.
tutorials:base() {
  local version=$1 base
  if [[ -n "${KIT_TUTORIALS_BASE:-}" ]]
  then
    base=${KIT_TUTORIALS_BASE}
  elif [[ "${version}" == "$(release:unreleased)" ]]
  then
    base=https://empowerite.github.io/kit/tutorials/main/
  elif [[ "${version}" =~ ^([0-9]+\.[0-9]+)\.[0-9]+$ ]]
  then
    base=https://kit.empowerite.dev/tutorials/${BASH_REMATCH[1]}/
  else
    base=https://empowerite.github.io/kit/tutorials/${version}/
  fi
  printf '%s\n' "${base%/}/"
}

# tutorials:json [<tag>…] prints one JSON object: the version, the base, the audiences, and the tutorials in order,
# each with its link; with tags, only the tutorials for any of those audiences. It returns 1 where the catalogue
# is missing.
tutorials:json() {
  local file version base
  file=$(tutorials:catalogue)
  if [[ ! -f "${file}" ]]
  then
    error "no catalogue of tutorials at ${file}"
    return 1
  fi
  version=$(release:version)
  base=$(tutorials:base "${version}")
  yaml:json "${file}" | jq --compact-output --arg version "${version}" --arg base "${base}" --args '
    ($ARGS.positional) as $tags
    | {version: $version, base: $base,
       audiences: [(.audiences // [])[] | {id, label}],
       tutorials: [(.tutorials // []) | sort_by(.order)[]
         | select(($tags | length) == 0 or any(.audience[]?; . as $a | $tags | index($a)))
         | {id, title, summary, glyph, audience: (.audience // []),
            url: ($base + .path + "?kit=" + $version)}]}' \
    -- "$@"
}

# tutorials:choices <key> [--kind | <answer>…] answers kit syntax --choices for <tutorial> and <audience>, which
# --tag takes: closed sets, each value with its title or label.
tutorials:choices() {
  local key=$1 file
  shift
  [[ "${key}" == '<tutorial>' || "${key}" == '<audience>' || "${key}" == --tag ]] || return 1
  if [[ "${1:-}" == --kind ]]
  then
    printf 'closed\n'
    return 0
  fi
  file=$(tutorials:catalogue)
  [[ -f "${file}" ]] || return 0
  if [[ "${key}" == '<tutorial>' ]]
  then
    yaml:json "${file}" | jq --raw-output '(.tutorials // []) | sort_by(.order)[] | [.id, .title, "", ""] | @tsv'
  else
    yaml:json "${file}" | jq --raw-output '(.audiences // [])[] | [.id, .label, "", ""] | @tsv'
  fi
}

# tutorials:text <json> prints the tutorials of the JSON tutorials:json made, a block each: the glyph and title, the
# summary, the audiences, and the link. With none, it says so in one dim line.
tutorials:text() {
  if [[ "$(jq '.tutorials | length' <<<"$1")" -eq 0 ]]
  then
    out:emit '•' '90' "no tutorials here; the catalogue lists none$([[ $# -gt 1 ]] && printf ' for %s' "${*:2}")"
    return 0
  fi
  jq --raw-output '.tutorials
    | map("\(.glyph) \(.title)\n   \(.summary)\n   for \(.audience | join(", "))\n   \(.url)") | join("\n\n")' <<<"$1"
}

# ----------------------------------------------------------------------------------------------------------------------
