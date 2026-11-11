#!/usr/bin/env bats
# house-policy.bats - this repository's five tree rules, in Rego, from the house package, judged on fixture documents
# ----------------------------------------------------------------------------------------------------------------------

load helper

repo_root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)

setup() {
  command -v opa >/dev/null || skip "opa, the opa policy engine, is not on PATH"
  fake-tools gh
  work="${BATS_TEST_TMPDIR}/desk"
  mkdir -p "${work}/.kit/policy"
  cd "${work}"
  git init --quiet --initial-branch=main
  cp "${kit_root}/packages/policy/opa/files/repo/.kit/policy/kit.rego" .kit/policy/
  cp "${repo_root}/.kit/packages/house/files/repo/.kit/policy/house.rego" .kit/policy/
}

# entry <path> [executable] adds one file to the fixture tree; it is executable when the second word is "x".
# shellcheck disable=SC2154  # the facts document is built in the test's scratch directory
entry() {
  local executable=false
  [[ "${2:-}" == x ]] && executable=true
  jq --arg path "$1" --argjson executable "${executable}" \
    '.tree[$path] = {type: "file", executable: $executable, size: 1, blob: "0", kinds: ["any"]}' \
    "${work}/doc.json" > "${work}/doc.tmp" && mv "${work}/doc.tmp" "${work}/doc.json"
}

# fixture starts a document that passes: a README.md, and the kit command in kit/bin/. kit policy --facts takes
# only a document with a policy key.
fixture() {
  printf '{"tree": {}, "policy": {}}\n' > "${work}/doc.json"
  entry README.md
  entry kit/bin/kit x
}

# judge runs the policy over the fixture document.
judge() {
  run "${kit_root}/bin/kit" policy --facts="${work}/doc.json"
}

@test "the fixture that passes passes" {
  fixture
  entry kit/bin/kit-flow x
  entry kit/modules/bash/policy.sh
  entry kit/packages/house/package.yml
  judge
  [ "$status" -eq 0 ]
}

@test "the kit's own tree passes" {
  # the kit's own tree, gathered in a copy so nothing lands in the checkout, judged without the branch's names
  (checkout-copy && "${kit_root}/bin/kit" facts --moment=commit > "${BATS_TEST_TMPDIR}/own.json")
  jq '.branch = null | .commits = [] | .change = null' "${BATS_TEST_TMPDIR}/own.json" > "${work}/doc.json"
  [ "$(jq '.tree | length' "${work}/doc.json")" -gt 0 ]
  judge
  [ "$status" -eq 0 ]
}

@test "a non-executable file in kit/bin/ is refused" {
  fixture
  entry kit/bin/kit-flow
  judge
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/bin/kit-flow is not executable"* ]]
}

@test "a file in kit/bin/ named other than kit or kit-* is refused" {
  fixture
  entry kit/bin/helper x
  entry kit/bin/kit_flow x
  entry kit/bin/kit-Flow x
  judge
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/bin/helper: a file in kit/bin/ is named kit or kit-*"* ]]
  [[ "$output" == *"kit/bin/kit_flow: a file"* ]]
  [[ "$output" == *"kit/bin/kit-Flow: a file"* ]]
}

@test "a folder in kit/bin/ is refused" {
  fixture
  entry kit/bin/extras/kit-more x
  judge
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/bin/extras/kit-more: kit/bin/ has no subfolders"* ]]
}

@test "kit/libexec/, which holds what the loader runs, is a folder of the release, as bin, modules, and packages are" {
  fixture
  entry kit/libexec/kit-call
  judge
  [ "$status" -eq 0 ]
}

@test "a folder in kit/ other than bin, libexec, modules, or packages is refused once" {
  fixture
  entry kit/lib/a.sh
  entry kit/lib/b.sh
  entry kit/lib/deep/c.sh
  judge
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/lib: kit/ holds only bin/, libexec/, modules/, and packages/"* ]]
  [ "$(grep --count 'kit/lib:' <<<"$output")" -eq 1 ]
}

@test "a file directly in kit/ is refused" {
  fixture
  entry kit/stray.sh
  judge
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/stray.sh: kit/ holds only bin/, libexec/, modules/, and packages/"* ]]
}

@test "a missing README.md is refused" {
  fixture
  jq 'del(.tree["README.md"])' "${work}/doc.json" > "${work}/doc.tmp" && mv "${work}/doc.tmp" "${work}/doc.json"
  judge
  [ "$status" -eq 1 ]
  [[ "$output" == *"README.md is missing"* ]]
}

# readme <text> puts the text in the fixture's README.md entry, and judges.
readme() {
  fixture
  jq --arg text "$1" '.tree["README.md"].text = $text' "${work}/doc.json" > "${work}/doc.tmp" \
    && mv "${work}/doc.tmp" "${work}/doc.json"
  judge
}

@test "a template README.md is refused: GitHub's, with a heading only" {
  readme $'# demo\n'
  [ "$status" -eq 1 ]
  [[ "$output" == *"README.md is still a template"* ]]
}

@test "a template README.md is refused: GitHub's, with a heading and a description" {
  readme $'# demo\nA demo repository.\n'
  [ "$status" -eq 1 ]
  [[ "$output" == *"README.md is still a template"* ]]
}

@test "a template README.md is refused: GitLab's" {
  readme $'# demo\n\nSome text here.\n\n## Suggestions for a good README\n'
  [ "$status" -eq 1 ]
  [[ "$output" == *"README.md is still a template"* ]]
}

@test "a template README.md is refused: the kit's, left unfilled" {
  readme $'# «title»\n«body»\n'
  [ "$status" -eq 1 ]
  [[ "$output" == *"README.md is still a template"* ]]
}

@test "a README.md of a heading and two lines of text is not a template" {
  readme $'# demo\nThis does one thing.\nIt does it well.\n'
  [ "$status" -eq 0 ]
}
