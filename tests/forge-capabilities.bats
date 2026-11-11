#!/usr/bin/env bats
# forge-capabilities.bats - a forge provider declares the contract it implements and the features its forge lacks
# ----------------------------------------------------------------------------------------------------------------------
# Each test's forge is a provider the repository adds on its module path, so the seam's own reading of a declaration
# is what is proven: a call of a feature the forge lacks answers unsupported, 3, and runs nothing of the provider's,
# and each verb that meets one answers as the file header of modules/bash/forge.sh says it should.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh glab
  fresh-repo
  git remote add origin https://code.example/team/example.git
  mkdir -p .kit bash/forge
}

# call <call> <argument>… runs one call of the seam, as a verb would.
call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && "forge:$0" "$@"' "$@"
}

# provider <name> <contract> <lacks>… writes a provider under bash/forge/ and makes it the repository's forge. Each of
# its calls appends its name to calls.log, so a test sees which ran. <contract> - writes no declaration of a version,
# and no <lacks> writes no declaration of what it lacks.
provider() {
  local name=$1 contract=$2 call
  shift 2
  printf 'forge: %s\nbash_path:\n  - kit:modules/bash/\n  - bash/\n' "${name}" > kit.yml
  {
    [[ "${contract}" == - ]] || printf 'forge/%s:contract() { printf "%%s\\n" %s; }\n' "${name}" "${contract}"
    (( $# == 0 )) || printf 'forge/%s:lacks() { printf "%%s\\n" %s; }\n' "${name}" "$*"
    printf 'forge/%s:require() { echo require >> "%s/calls.log"; }\n' "${name}" "${repo}"
    for call in quota marks releases runs open-issues pr-closing-issues issue-title
    do
      printf 'forge/%s:%s() { echo %s >> "%s/calls.log"; echo answered; }\n' "${name}" "${call}" "${call}" "${repo}"
    done
  } > "bash/forge/${name}.sh"
}

# added is what contract 4 adds, as JSON strings: a provider that has written none of it lacks all of it.
added='"milestones","iterations","reviewers","merged","artifacts","environments","deployments","collaborators",'
added+='"teams","bots","attestations","code-scanning","dependabot","secret-scanning","rulesets","bypasses"'

@test "GitHub and GitLab implement contract 4, and lack the features it adds until their calls are written" {
  git remote set-url origin https://github.com/empowerite/example.git
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$status" -eq 0 ]
  [ "$(jq -c '{contract, forge, provider}' <<<"$output")" = '{"contract":4,"forge":"github","provider":4}' ]
  [ "$(jq -c '[.features | to_entries[] | select(.value == "native") | .key] | length' <<<"$output")" = 28 ]
  [ "$(jq -c '.features.milestones' <<<"$output")" = '"native"' ]
  [ "$(jq -r '.features | keys_unsorted | join(" ")' <<<"$output")" \
    = 'issues sub-issues linked-branches drafts closing-issues review-threads checks quota marks releases runs labels milestones iterations reviewers merged artifacts environments deployments collaborators teams bots attestations code-scanning dependabot secret-scanning rulesets bypasses' ]
  git remote set-url origin https://gitlab.com/empowerite/example.git
  run "${kit_root}/bin/kit" flow --capabilities
  not_built=${added#'"milestones","iterations","reviewers","merged","artifacts","environments","deployments",'}
  not_built=${not_built#'"collaborators","teams","bots",'}
  not_built=${not_built/'"code-scanning","dependabot","secret-scanning","rulesets",'/}
  [ "$(jq -c '{forge, provider, none: [.features | to_entries[] | select(.value == "none") | .key]}' <<<"$output")" \
    = "{\"forge\":\"gitlab\",\"provider\":4,\"none\":[${not_built}]}" ]
}

@test "a provider that declares nothing implements contract 1: every feature of it, none added since" {
  provider plain -
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$status" -eq 0 ]
  [ "$(jq -c '{forge, provider, none: [.features | to_entries[] | select(.value == "none") | .key]}' <<<"$output")" \
    = "{\"forge\":\"plain\",\"provider\":1,\"none\":[\"labels\",${added}]}" ]
  run call quota
  [ "$status" -eq 0 ]
  [ "$output" = answered ]
}

@test "a call of a feature the forge lacks answers 3, prints nothing, and runs nothing of the provider's" {
  provider sparse 2 quota marks releases runs issues closing-issues
  for c in quota marks releases runs open-issues pr-closing-issues issue-title
  do
    run --separate-stderr call "${c}" 7
    [ "$status" -eq 3 ]
    [ -z "$output" ]
    [ -z "$stderr" ]
  done
  [ ! -e calls.log ]
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$(jq -c '[.features | to_entries[] | select(.value == "none") | .key]' <<<"$output")" \
    = "[\"issues\",\"closing-issues\",\"quota\",\"marks\",\"releases\",\"runs\",\"labels\",${added}]" ]
}

@test "a forge without rate limits gives kit flow --quota none, and asks it nothing" {
  provider sparse 2 quota
  run --separate-stderr "${kit_root}/bin/kit" flow --quota
  [ "$status" -eq 0 ]
  [ "$output" = '[]' ]
  [ ! -e calls.log ]
}

@test "a forge without releases, runs, or issues lists none of them, and a stage of them still draws" {
  provider sparse 2 releases runs issues
  run bash -c 'source "${kit_root}/modules/bash/1st.sh" && use releases && releases:documents'
  [ "$status" -eq 0 ]
  [ "$output" = '[]' ]
  run bash -c 'source "${kit_root}/modules/bash/1st.sh" && use builds && builds:documents'
  [ "$status" -eq 0 ]
  [ "$output" = '[]' ]
  [ "$(cat calls.log)" = $'require\nrequire' ]
}

@test "a forge without marks gives its stages none, and that is no failure" {
  provider sparse 2 marks
  printf 'flow: [releases, builds]\n' >> kit.yml
  run --separate-stderr "${kit_root}/bin/kit" flow --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"releases":null,"builds":null}' ]
  [ -z "$stderr" ]
}

@test "the forge's own tracker refuses on a forge without issues, and names the remedy" {
  provider sparse 2 issues
  run bash -c 'source "${kit_root}/modules/bash/1st.sh" && use tracker && tracker:of 7 title'
  [ "$status" -eq 1 ]
  [[ "$output" == *'the sparse forge has no issues of its own; claim "#" for another tracker under trackers in kit.yml'* ]]
  [ ! -e calls.log ]
}

@test "a declaration of a contract above the kit's, or of a feature the contract does not have, is refused" {
  provider ahead 5
  run call quota
  [ "$status" -eq 1 ]
  [[ "$output" == *"the ahead forge implements contract '5', and this kit knows 1 to 4; move to a later kit: kit --version=<version> update"* ]]
  provider odd 2 telepathy
  run call quota
  [ "$status" -eq 1 ]
  [[ "$output" == *"the odd forge lacks 'telepathy', which is no feature of the forge contract"* ]]
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$status" -eq 1 ]
  [ ! -e calls.log ]
}

@test "a primary with no forge has every feature none" {
  git remote set-url origin /nowhere/example
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$status" -eq 0 ]
  [ "$(jq -c '{forge, provider, native: [.features[] | select(. == "native")]}' <<<"$output")" \
    = '{"forge":null,"provider":0,"native":[]}' ]
}

# ----------------------------------------------------------------------------------------------------------------------
