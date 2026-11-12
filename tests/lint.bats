#!/usr/bin/env bats
# lint.bats - kit lint runs each rule's linter over the paths it governs, staged, named, or every tracked one
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup_file() {
  world-repo installed
}

setup() {
  shellcheck --version >/dev/null 2>&1 || skip "shellcheck is the bash rule's linter, and it is not on PATH"
  world-copy-repo installed
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

@test "a repository's own bin/kit-* script is a script, held to no verb template, which is the kit's own source's" {
  mkdir bin
  (cd bin && script kit-bootstrap 'echo "${x:-}"')
  run "${kit_checkout}/bin/kit" lint bin/kit-bootstrap
  [ "$status" -eq 0 ]
}

@test "in the kit's own source, a verb in the old shape is refused by the verb template's Usage: line" {
  mkdir -p kit/bin
  printf '%s\n' '#!/usr/bin/env bash' '# kit-old - a verb in the old shape (kit verb: help)' '# kit old' '#' '# Does it.' \
    '' 'set -o errexit -o nounset -o pipefail' > kit/bin/kit-old
  chmod +x kit/bin/kit-old
  run "${kit_checkout}/bin/kit" lint kit/bin/kit-old
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit/bin/kit-old does not have the shape of the bash template (verb); it leaves .kit/templates/bash/verb line 3: # Usage:"* ]]
}

@test "a verb whose body holds a line of the verb template's own is refused, naming the file's line that holds it" {
  mkdir -p kit/bin
  printf '%s\n' '#!/usr/bin/env bash' '# kit-heredoc - a verb that prints a label (kit verb: help)' '# Usage:' \
    '#   kit heredoc' '#' '# Does it.' '#' '# Arguments:' '#' '# Exits 0.' '#' '# see: bin/kit' '' \
    'set -o errexit -o nounset -o pipefail' "cat <<'TXT'" '# Usage:' 'TXT' > kit/bin/kit-heredoc
  chmod +x kit/bin/kit-heredoc
  run "${kit_checkout}/bin/kit" lint kit/bin/kit-heredoc
  [ "$status" -eq 1 ]
  [[ "$output" == *"the file's line 16, # Usage:, is a line of the template's own, which no hole may hold"* ]]
}

@test "a kit.yml yq cannot read stops kit lint and kit hydrate, never as a repository with no kit" {
  local placed
  rm -rf .kit/rules
  placed=$(find .kit | LC_ALL=C sort)
  printf 'packages: [kit\n' > kit.yml
  run "${kit_checkout}/bin/kit" lint good.sh
  [ "$status" -eq 1 ]
  [[ "$output" != *"✅"* ]]
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 1 ]
  [[ "$output" != *"✅"* ]]
  [[ "$output" != *"run: kit install"* ]]
  run "${kit_checkout}/bin/kit" hydrate --quick
  [ "$status" -eq 1 ]
  [[ "$output" != *"✅"* ]]
  # nothing is placed: the rules taken out before stay out
  [ "$(find .kit | LC_ALL=C sort)" = "${placed}" ]
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

@test "a path that is not there is refused by name" {
  run "${kit_checkout}/bin/kit" lint nosuch.sh
  [ "$status" -eq 2 ]
  [[ "$output" == *"'nosuch.sh' is not a file or a directory here"* ]]
  [[ "$output" != *"✅"* ]]
  run "${kit_checkout}/bin/kit" lint good.sh nosuch.sh
  [ "$status" -eq 2 ]
  [[ "$output" != *"shellcheck"* ]]
}

@test "a path staged and then taken off the disk is refused by name; a staged deletion is left out" {
  git add bad.sh good.sh
  rm bad.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 2 ]
  said="'bad.sh' is staged but is not on the disk; run: git restore -- 'bad.sh', or git rm --cached -- 'bad.sh'"
  [[ "$output" == *"${said}"* ]]
  [[ "$output" != *"shellcheck"* ]]
  [[ "$output" != *"✅"* ]]
  git rm --quiet --cached bad.sh
  git commit --quiet --message '#1 good: a script'
  git rm --quiet good.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
  [[ "$output" == *"nothing to lint"* ]]
}

@test "a path no rule governs is said to be linted by nothing" {
  printf 'png\n' > logo.png
  run "${kit_checkout}/bin/kit" lint logo.png
  [ "$status" -eq 0 ]
  [[ "$output" == *"no rule governs the 1 path(s); nothing was linted"* ]]
  [[ "$output" != *"every governed path passes"* ]]
}

@test "a kind whose pattern is no regular expression fails the lint, and names the kind" {
  "${kit_checkout}/bin/kit" new kind odd --content='^#!.*(odd'
  "${kit_checkout}/bin/kit" packages add local
  run "${kit_checkout}/bin/kit" lint bad.sh
  [ "$status" -eq 1 ]
  [[ "$output" == *"the kind odd's content pattern"* ]]
  [[ "$output" != *"✅ every governed"* ]]
  # bad.sh fails shellcheck too, so the status alone proves nothing: no linter ran
  [[ "$output" != *".kit/rules/bash.md: shellcheck"* ]]
  run "${kit_checkout}/bin/kit" context bad.sh
  [ "$status" -eq 1 ]
  [[ "$output" == *"the kind odd's content pattern '^#!.*(odd' is not a regular expression"* ]]
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

@test "an embedded kind's rule reaches the reader through kit context, and never its linter through kit lint" {
  printf '[script]\nbuild:\n  echo $x\n' > justfile
  run "${kit_checkout}/bin/kit" context justfile
  [ "${lines[0]}" = ".kit/rules/just.md" ]
  [ "${lines[1]}" = ".kit/rules/bash.md" ]
  run "${kit_checkout}/bin/kit" lint justfile
  [ "$status" -eq 0 ]
  [[ "$output" != *"bash.md"* ]]
  [[ "$output" != *"shellcheck"* ]]
}

@test "the delivered markdown passes the delivered markdown linter in a fresh repository" {
  command -v markdownlint-cli2 >/dev/null || skip "markdownlint-cli2, the markdown style's linter, is not on PATH"
  "${kit_checkout}/bin/kit" packages add style/markdown agents/generic >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" lint .kit/1st.md .kit/rules/bash.md .kit/styles/markdown.md AGENTS.md
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
  for tool in bash jq yq awk sed grep sort cut paste tr find head tail cat mkdir rm mv cp dirname basename date env \
    uname readlink comm cksum
  do
    p=$(command -v "${tool}") && ln -s "${p}" "${BATS_TEST_TMPDIR}/path/${tool}" || true
  done
  git add good.sh
  run env PATH="${BATS_TEST_TMPDIR}/path" "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 1 ]
  [[ "$output" == *"shellcheck is not on PATH"* ]]
}

@test "a linter a dependency provider installed is run from its directory, though PATH lacks it" {
  mkdir -p "${BATS_TEST_TMPDIR}/path" node_modules/.bin
  for tool in git bash jq yq awk sed grep sort cut paste tr find head tail cat mkdir rm mv cp dirname basename date \
    env uname readlink comm cksum
  do
    p=$(command -v "${tool}") && ln -s "${p}" "${BATS_TEST_TMPDIR}/path/${tool}" || true
  done
  printf '{}\n' > package.json && printf '{}\n' > package-lock.json
  printf '#!/usr/bin/env bash\necho "installed shellcheck over $#"\n' > node_modules/.bin/shellcheck
  chmod +x node_modules/.bin/shellcheck
  git add good.sh
  run env PATH="${BATS_TEST_TMPDIR}/path" "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
  [[ "$output" == *"installed shellcheck over"* ]]
  # a lock with no manifest pins nothing, so its directory is not looked in
  rm package.json
  run env PATH="${BATS_TEST_TMPDIR}/path" "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 1 ]
  [[ "$output" == *"shellcheck is not on PATH"* ]]
}

@test "a linter the chosen toolchain installed is run from its directory, though PATH lacks it" {
  platform=$(kit-bash 'use toolchain/kit && toolchain/kit:platform')
  printf 'shellcheck\t1\t%s\tboth\thttps://example.com/shellcheck\t%064d\t.\n' "${platform}" 0 > tools.lock
  dir="${KIT_HOME}/.kit/tools/shellcheck/1/${platform}"
  mkdir -p "${dir}"
  printf '#!/usr/bin/env bash\necho "the toolchain'"'"'s shellcheck over $#"\n' > "${dir}/shellcheck"
  chmod +x "${dir}/shellcheck"
  path-without shellcheck
  git add good.sh
  KIT_TOOLCHAIN=kit run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
  [[ "$output" == *"the toolchain's shellcheck over"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
