# shellcheck shell=bash
# packages.sh - kit packages' stage: the packages in reach, what each needs, and what needs it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is set by modules/bash/1st.sh
#
# The one stage, packages, is a row per package in reach, as kit packages lists them: its name, its state (active,
# inactive, or excluded), its origin (release, repository, or a source's name), what it depends on and includes, the
# active packages that depend on it and the active bundles that include it, and its description. It changes nothing;
# kit packages select, add, and remove change the active set, and kittui's actions run them.
#
# The stage's mark is the hash of kit.yml, which declares the active set, the excluded members, the release, and the
# sources, and of .kit/packages/, the repository's own packages.
#
# see: bin/kit-packages, modules/bash/apps.sh, modules/bash/package.sh, docs/apps.md
use package resolve settings apps workspaces

# packages_stages are kit packages' stages, in the Packages app's tab order.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
packages_stages=(packages)

# apps/packages:packages prints the packages document, as the file header says. Where kit was never installed it
# refuses as kit packages does, and returns 1.
apps/packages:packages() {
  local listed package
  local -a active
  settings:require || return 1
  listed=$(package:list --json) || return 1
  list:read active settings:packages || return 1
  # A line per package in reach: its name, then what it depends on and what it includes, each apart by spaces.
  for package in $(jq --raw-output .name <<<"${listed}")
  do
    printf '%s\t%s\t%s\n' "${package}" "$(resolve:depends "${kit_root}" "${package}" | paste -s -d ' ' -)" \
      "$(resolve:includes "${kit_root}" "${package}" | paste -s -d ' ' -)"
  done | jq --raw-input --slurp --compact-output --argjson listed "$(jq --slurp --compact-output . <<<"${listed}")" \
    --argjson active "$(printf '%s\n' "${active[@]}" | workspaces:lines-json)" '
    def words: split(" ") | map(select(. != ""));
    (split("\n") | map(select(. != "") | split("\t")
      | {key: .[0], value: {depends: (.[1] // "" | words), includes: (.[2] // "" | words)}}) | from_entries) as $graph
    | {rows: [$listed[] | .name as $name
        | {key: $name, name: $name, state, origin, depends: $graph[$name].depends, includes: $graph[$name].includes,
           dependents: [$active[] | select(any($graph[.].depends[]?; . == $name))],
           bundles: [$active[] | select(any($graph[.].includes[]?; . == $name))],
           description, links: {}}],
       errors: []}'
}

# apps/packages:mark-packages prints the packages stage's mark: the hash of kit.yml and of .kit/packages/.
apps/packages:mark-packages() { apps:hash kit.yml .kit/packages; }

# ----------------------------------------------------------------------------------------------------------------------
