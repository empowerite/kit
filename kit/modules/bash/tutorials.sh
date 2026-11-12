# shellcheck shell=bash
# tutorials.sh - kit's tutorials: the catalogue, each version's base, and the JSON of kit tutorials (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# The tutorials are of two kinds: online, a site, one MDX page per tutorial; and in the kit, a markdown page that ships
# with the release, under modules/tutorials/, read on this machine. Their catalogue is modules/tutorials.yml, which kit
# generate writes from the pages' front matter: the audiences, each its id, label, and color, and for each tutorial its
# id, title, summary, glyph, audiences, and order, with its path on the site when it is online, or its file, from the
# catalogue's directory, when it is in the kit. KIT_TUTORIALS_CATALOGUE, when set, names another file in its place.
#
# A tutorial's link is <base><path>?kit=<version>, and the base is <site><line>/, one site for every version. The line
# follows this kit's version:
#
#   - unreleased:   main
#   - with a suffix, as 0.5.0-rc.1:   the version, 0.5.0-rc.1
#   - X.Y.Z:   X.Y
#
# KIT_TUTORIALS_BASE, when set, replaces the base. Nothing here runs a tutorial's commands.
#
use release yaml markdown

# tutorials:catalogue prints the path of the catalogue.
tutorials:catalogue() { printf '%s\n' "${KIT_TUTORIALS_CATALOGUE:-${kit_root}/modules/tutorials.yml}"; }

# tutorials:site prints the one site the tutorials are published at, ending in a slash.
tutorials:site() { printf '%s\n' https://empowerite.github.io/kit/tutorials/; }

# tutorials:line <version> prints the line of the site a version's tutorials are on: main for unreleased, the version
# for one with a suffix, and <major>.<minor> for a release.
tutorials:line() {
  local version=$1
  if [[ "${version}" == "$(release:unreleased)" ]]
  then
    printf '%s\n' main
  elif [[ "${version}" =~ ^([0-9]+\.[0-9]+)\.[0-9]+$ ]]
  then
    printf '%s\n' "${BASH_REMATCH[1]}"
  else
    printf '%s\n' "${version}"
  fi
}

# tutorials:base <version> prints the base the tutorials' links start from, ending in a slash: KIT_TUTORIALS_BASE, or
# the site and the version's line.
tutorials:base() {
  local base
  if [[ -n "${KIT_TUTORIALS_BASE:-}" ]]
  then
    base=${KIT_TUTORIALS_BASE}
  else
    base=$(tutorials:site)$(tutorials:line "$1")
  fi
  printf '%s\n' "${base%/}/"
}

# tutorials:json [<tag>…] prints one JSON object: the version, the base, the audiences, each with its color, empty
# where the catalogue gives none, and the tutorials in order, each with where it is, kit or online, and its link when
# online or the absolute path of its file when in the kit, the other empty; with tags, only the tutorials for any of
# those audiences. It returns 1 where the catalogue is missing.
tutorials:json() {
  local file version base dir
  file=$(tutorials:catalogue)
  if [[ ! -f "${file}" ]]
  then
    error "no catalogue of tutorials at ${file}"
    return 1
  fi
  version=$(release:version)
  base=$(tutorials:base "${version}")
  dir=$(cd "$(dirname "${file}")" && pwd)
  yaml:json "${file}" | jq --compact-output --arg version "${version}" --arg base "${base}" --arg dir "${dir}" --args '
    ($ARGS.positional) as $tags
    | {version: $version, base: $base,
       audiences: [(.audiences // [])[] | {id, label, color: (.color // "")}],
       tutorials: [(.tutorials // []) | sort_by(.order)[]
         | select(($tags | length) == 0 or any(.audience[]?; . as $a | $tags | index($a)))
         | {id, title, summary, glyph, audience: (.audience // []),
            where: (if .file then "kit" else "online" end),
            url: (if .file then "" else $base + .path + "?kit=" + $version end),
            file: (if .file then $dir + "/" + .file else "" end)}]}' \
    -- "$@"
}

# tutorials:page <file> prints a tutorial's page that ships with the kit, its front matter left out and each link drawn
# as text, markdown:unlink's. It returns 1 where the page is missing.
tutorials:page() {
  if [[ ! -f "$1" ]]
  then
    error "no page of the tutorial at $1"
    return 1
  fi
  awk 'NR == 1 && $0 == "---" { front = 1; next } front && $0 == "---" { front = 0; next } !front' "$1" \
    | markdown:unlink
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
  [[ -f "${file}" ]] || return 1
  if [[ "${key}" == '<tutorial>' ]]
  then
    yaml:json "${file}" | jq --raw-output '(.tutorials // []) | sort_by(.order)[] | [.id, .title, "", ""] | @tsv'
  else
    yaml:json "${file}" | jq --raw-output '(.audiences // [])[] | [.id, .label, "", ""] | @tsv'
  fi
}

# tutorials:text <json> prints the tutorials of the JSON tutorials:json made, a block each: the glyph and title, the
# summary, the audiences, and the link, or, for a tutorial in the kit, the command that reads it. With none, it says so
# in one dim line.
tutorials:text() {
  if [[ "$(jq '.tutorials | length' <<<"$1")" -eq 0 ]]
  then
    out:emit '•' '90' "no tutorials here; the catalogue lists none$([[ $# -gt 1 ]] && printf ' for %s' "${*:2}")"
    return 0
  fi
  jq --raw-output '.tutorials
    | map("\(.glyph) \(.title)\n   \(.summary)\n   for \(.audience | join(", "))\n   "
      + (if .where == "kit" then "in the kit: kit tutorials \(.id)" else .url end)) | join("\n\n")' <<<"$1"
}

# ----------------------------------------------------------------------------------------------------------------------
