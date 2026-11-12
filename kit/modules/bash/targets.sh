# shellcheck shell=bash
# targets.sh - the seam over push targets: where a release of the repository's own artifacts goes (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A seam is a contract plus the point that picks who answers it; docs/architecture.md says why not "interface".
#
# A push target is where a release goes, named for each artifact under publish: in kit.yml, and never searched: the
# pull path is what consumers search. Each kind is a provider, targets/<kind>.sh beside it, whose functions are named
# targets/<kind>:<call>. A kind is its own, never the push side of a pull kind: a Go module is published by its tag,
# vcs-tag, and the proxy fetches it. kit never speaks a service's upload API itself: each provider runs its connector,
# the tool its service ships, with the runner's credentials.
#
#   publish:
#     kit:
#       files: ["dist/kit-{version}.tar.gz"]
#       release: empowerite/kit
#       asset: kit-{version}.tar.gz
#       targets:
#         release: [github-release]
#         prerelease: [github-release]
#
# A target is a kind's name, or a map with its kind and the keys it takes, as {kind: artifactory-local, url: …,
# repository: …}. A version is never overwritten: a target that holds it already is refused before any is pushed to.
#
# Every provider implements the calls below.
#
#   connector               the command it runs, which must be on PATH, as gh or jf
#   keys                    the keys a target of this kind takes beyond kind, one per line
#   pushed <version> <target>
#                           succeeds when the target holds the version; the target is its JSON
#   push <version> <target> <file>…
#                           pushes the files as the version; a pre-release where the version has a suffix
#   retire <version> <target>
#                           takes the version back, where the target can: delete is retire
#
# see: bin/kit-publish, docs/superpowers/specs/2026-11-12-artifact-sources-design.md, issue #182

use yaml

# targets:kinds prints the kinds on the module path, one per line.
targets:kinds() {
  use:modules 'targets/*' | sed 's|^targets/||'
}

# targets:call <kind> <call> <argument>… runs the kind's provider's function for the call.
targets:call() {
  local kind=$1 call=$2
  shift 2
  use "targets/${kind}" || return 1
  "targets/${kind}:${call}" "$@"
}

# targets:runner prints the name of the runner kit runs on, from its own marker, GITHUB_ACTIONS, GITLAB_CI, or TF_BUILD,
# and returns 1 at a desk, where none is set.
targets:runner() {
  if [[ -n "${GITHUB_ACTIONS:-}" ]]
  then printf 'GitHub Actions\n'
  elif [[ -n "${GITLAB_CI:-}" ]]
  then printf 'GitLab CI\n'
  elif [[ -n "${TF_BUILD:-}" ]]
  then printf 'Azure Pipelines\n'
  else return 1
  fi
}

# targets:artifacts prints each artifact under publish: in kit.yml as one line of JSON, in order: {name, files,
# release, asset, tag, module, module_version, npm, url, targets: {release, prerelease}, verify}, each target as {kind,
# …}, the strings as strings, and the lists empty where kit.yml leaves them out.
targets:artifacts() {
  local doc
  doc=$(yaml:eval kit.yml 'publish artifacts' '.publish // {}' --output-format=json --indent=0) || return 1
  jq --compact-output '
    def target: if type == "string" then {kind: .} else . end;
    def strings: [(. // [])[] | tostring];
    to_entries[] | .key as $name | (.value // {}) | {
      name: $name, files: (.files | strings),
      release: (.release // "" | tostring), asset: (.asset // "" | tostring), tag: (.tag // "" | tostring),
      module: (.module // "" | tostring), module_version: (.module_version // "" | tostring),
      npm: (.npm // "" | tostring), url: (.url // "" | tostring),
      targets: {release: [(.targets.release // [])[] | target], prerelease: [(.targets.prerelease // [])[] | target]},
      verify: (.verify | strings), os: {}, arch: {}, ext: {}}' <<<"${doc:-{\}}"
}

# targets:check <artifact>… checks the artifacts, each a line targets:artifacts prints, and says what is wrong with
# the first that is wrong, and returns 1: an artifact with no files, a target of a kind no provider answers, and a
# key a kind does not take.
targets:check() {
  local line kinds kind target keys wrong name
  local -a targets
  kinds=$(targets:kinds | paste -s -d ' ' -)
  for line in "$@"
  do
    [[ -n "${line}" ]] || continue
    name=$(jq --raw-output .name <<<"${line}")
    if [[ "$(jq '.files | length' <<<"${line}")" == 0 ]]
    then
      error "kit.yml: the artifact ${name} under publish: needs files:, what the runner built, as" \
        "dist/${name}-{version}.tar.gz"
      return 1
    fi
    list:read targets jq --compact-output '.targets[][]' <<<"${line}" || return 1
    for target in "${targets[@]}"
    do
      [[ -n "${target}" ]] || continue
      kind=$(jq --raw-output '.kind // ""' <<<"${target}")
      if [[ " ${kinds} " != *" ${kind} "* ]]
      then
        error "kit.yml: the artifact ${name} has a target of the kind ${kind:-none}, which no provider answers;" \
          "the kinds are ${kinds}"
        return 1
      fi
      keys=$(targets:call "${kind}" keys | paste -s -d ' ' -) || return 1
      wrong=$(jq --raw-output --arg allowed "kind ${keys}" \
        'first(keys[] | select(. as $key | $allowed | split(" ") | index([$key]) | not)) // empty' \
        <<<"${target}")
      if [[ -n "${wrong}" ]]
      then
        error "kit.yml: the artifact ${name} has a ${kind} target with the key ${wrong}, which a ${kind} target does" \
          "not take"
        return 1
      fi
    done
  done
}

# ----------------------------------------------------------------------------------------------------------------------
