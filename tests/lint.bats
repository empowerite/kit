#!/usr/bin/env bats
# lint.bats - kit lint runs each rule's linter over the paths it governs, staged, named, or every tracked one
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  shellcheck --version >/dev/null 2>&1 || skip "shellcheck is the bash rule's linter, and it is not on PATH"
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  "${kit_checkout}/bin/kit" install >/dev/null
  script bad.sh 'echo $x'
  script good.sh 'echo "${x:-}"'
  script tool 'echo $y'
}

# script <path> <line> writes an executed bash script in the bash template's shape, its body the one line, so the
# only thing a test's linter can refuse is that line.
script() {
  local rule
  printf -v rule '# %s' "$(printf '%.0s-' $(seq 1 118))"
  printf '#!/usr/bin/env bash\n# %s - a fixture\n%s\n\nset -o errexit -o nounset -o pipefail\n\n%s\n\n%s\n' \
    "$1" "${rule}" "$2" "${rule}" > "$1"
}

teardown() {
  rm -rf "${repo}"
}

@test "lint refuses a staged bash file shellcheck faults, and passes a clean one" {
  git add bad.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 1 ]
  [[ "$output" == *"SC2086"* ]]
  [[ "$output" == *".kit/rules/bash.md: shellcheck"* ]]
  git reset --quiet && git add good.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
  [[ "$output" == *"every governed path passes"* ]]
}

@test "a named path is linted by the rule its first line names, and --all-files reads every tracked path" {
  run "${kit_checkout}/bin/kit" lint tool
  [ "$status" -eq 1 ]
  [[ "$output" == *"SC2086"* ]]
  git add good.sh && git commit --quiet --no-verify --message good
  git add bad.sh && git commit --quiet --no-verify --message bad
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
  [[ "$output" == *"nothing to lint"* ]]
  run "${kit_checkout}/bin/kit" lint --all-files
  [ "$status" -eq 1 ]
  [[ "$output" == *"bad.sh"* ]]
}

@test "a named directory stands for the tracked paths under it" {
  mkdir sub && cp bad.sh sub/bad.sh && cp good.sh sub/good.sh && git add sub
  cp tool sub/untracked
  run "${kit_checkout}/bin/kit" lint sub
  [ "$status" -eq 1 ]
  [[ "$output" == *"sub/bad.sh"* ]]
  [[ "$output" != *"untracked"* ]]
}

@test "a named path is taken from the directory the command was typed in, and the staged list from the root" {
  mkdir sub && cp bad.sh sub/bad.sh && git add sub/bad.sh
  cd sub
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 1 ]
  [[ "$output" == *"sub/bad.sh"* ]]
  run "${kit_checkout}/bin/kit" lint bad.sh
  [ "$status" -eq 1 ]
  [[ "$output" == *"SC2086"* ]]
}

@test "the delivered markdown passes the delivered markdown linter in a fresh repository" {
  command -v markdownlint-cli2 >/dev/null || skip "markdownlint-cli2, the markdown style's linter, is not on PATH"
  "${kit_checkout}/bin/kit" package add style/markdown >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" lint .kit/1st.md .kit/rules/bash.md .kit/style/markdown.md AGENTS.md
  [ "$status" -eq 0 ]
  [[ "$output" == *".kit/markdownlint.jsonc"* ]]
}

@test "the workflows rule's pinned check refuses a uses: without a commit SHA and its version, and accepts local and docker actions" {
  command -v actionlint >/dev/null || skip "actionlint, the workflows rule's linter, is not on PATH"
  mkdir -p .github/workflows .github/actions/local
  printf 'name: local\ndescription: a local action\nruns:\n  using: composite\n  steps:\n    - run: true\n      shell: bash\n' \
    > .github/actions/local/action.yml
  cat > .github/workflows/w.yml <<'YAML'
on: push
jobs:
  a:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1  # v7.0.1
      - uses: actions/checkout@v7
      - uses: "jdx/mise-action@c2a87611a18de5b3828c5652fe268e992400cb5c"
      - uses: ./.github/actions/local
      - uses: docker://alpine:3.20
YAML
  run "${kit_checkout}/bin/kit" lint .github/workflows/w.yml
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/workflows.md: the pinned check over 1 path(s)"* ]]
  [[ "$output" == *"w.yml:7: uses: actions/checkout@v7 is not pinned to a commit SHA"* ]]
  [[ "$output" == *"w.yml:8: uses: jdx/mise-action@c2a87611a18de5b3828c5652fe268e992400cb5c has no version comment"* ]]
  [[ "$output" != *"w.yml:6:"* && "$output" != *"w.yml:9:"* && "$output" != *"w.yml:10:"* ]]
  grep -v -e '@v7$' -e 'mise-action' .github/workflows/w.yml > w.ok && mv w.ok .github/workflows/w.yml
  run "${kit_checkout}/bin/kit" lint .github/workflows/w.yml
  [ "$status" -eq 0 ]
}

@test "a linter a rule names that is not on PATH is a refusal, named" {
  mkdir -p "${BATS_TEST_TMPDIR}/path" && ln -s "$(command -v git)" "${BATS_TEST_TMPDIR}/path/git"
  for tool in bash jq awk sed grep sort cut paste tr find head tail cat mkdir rm mv cp dirname basename date env uname \
    readlink comm
  do
    p=$(command -v "${tool}") && ln -s "${p}" "${BATS_TEST_TMPDIR}/path/${tool}" || true
  done
  git add good.sh
  run env PATH="${BATS_TEST_TMPDIR}/path" "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 1 ]
  [[ "$output" == *"shellcheck is not on PATH"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
