#!/usr/bin/env bats
# kit.bats - bin/kit install, check, and update against a fresh repository
# ----------------------------------------------------------------------------------------------------------------------
#
# Each test gets an empty git repository under a temporary directory and runs the kit from this checkout against
# it. The update tests build a second, later payload from the first so the three-way merge has a base.

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  repo=$(mktemp -d)
  git -C "${repo}" init --quiet --initial-branch=main
  cd "${repo}"
}

teardown() {
  rm -rf "${repo}"
}

# later-release <dir> copies this checkout's release to <dir> as version 9.9.9 with one changed rule line.
later-release() {
  mkdir -p "$1"
  cp -R "${kit_checkout}/bin" "${kit_checkout}/tree" "${kit_checkout}/targets" "${kit_checkout}/bash" "$1/"
  printf '9.9.9\n' > "$1/version"
  printf '\n## Added in 9.9.9\n\nA new rule.\n' >> "$1/tree/.kit/rules/json.md"
}

@test "install with the claude target writes the tree, the mirrors, the workflow, the settings, and the manifest" {
  run "${kit_checkout}/bin/kit" install claude
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ -f .kit/rules/bash.md ]
  [ ! -e .kit/bash ]
  [ -f .kit/lefthook.yml ]
  [ -f .claude/rules/kit-markdown.md ]
  [ ! -L .claude/rules/kit-markdown.md ]
  cmp --silent .claude/rules/kit-markdown.md .kit/rules/markdown.md
  head -n 1 .claude/rules/kit-bash.md | grep --quiet --line-regexp -- '---'
  grep --quiet '"\*\*/\*\.sh"' .claude/rules/kit-bash.md
  [ -f .github/workflows/kit-ci.yml ]
  jq --exit-status '.permissions.deny | index("Bash(git rebase *)")' .claude/settings.json >/dev/null
  grep --quiet '^version 0.1.0$' .kit/manifest
  grep --quiet '^targets claude$' .kit/manifest
}

@test "install with no target writes the neutral tree and nothing under .claude" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ -f .github/workflows/kit-ci.yml ]
  [ ! -e .claude ]
  grep --quiet '^targets$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  # a stray file under a target not taken is nobody's business
  mkdir -p .claude/rules && printf 'theirs\n' > .claude/rules/kit-json.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "install refuses a target this release does not know and names the known ones" {
  run "${kit_checkout}/bin/kit" install gemini
  [ "$status" -eq 2 ]
  [[ "$output" == *"the targets are: claude"* ]]
  [ ! -e .kit/manifest ]
}

@test "target add places a target's files, and target remove takes them and the kit's entries back out" {
  mkdir .claude
  printf '{"model":"x","permissions":{"deny":["Bash(rm -rf /)"]}}\n' > .claude/settings.json
  "${kit_checkout}/bin/kit" install
  [ "$("${kit_checkout}/bin/kit" target list)" = "" ]
  run "${kit_checkout}/bin/kit" target add claude
  [ "$status" -eq 0 ]
  [ "$("${kit_checkout}/bin/kit" target list)" = "claude" ]
  [ -f .claude/rules/kit-bash.md ]
  jq --exit-status '.permissions.deny | index("Bash(git switch main *)")' .claude/settings.json >/dev/null
  grep --quiet '^targets claude$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" target remove claude
  [ "$status" -eq 0 ]
  [ ! -e .claude/rules/kit-bash.md ]
  [ "$(jq --raw-output .model .claude/settings.json)" = "x" ]
  jq --exit-status '.permissions.deny == ["Bash(rm -rf /)"]' .claude/settings.json >/dev/null
  grep --quiet '^targets$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "install keeps the repository's own settings keys and deny entries" {
  mkdir .claude
  printf '{"model":"x","permissions":{"deny":["Bash(rm -rf /)"]}}\n' > .claude/settings.json
  "${kit_checkout}/bin/kit" install claude
  [ "$(jq --raw-output .model .claude/settings.json)" = "x" ]
  jq --exit-status '.permissions.deny | index("Bash(rm -rf /)")' .claude/settings.json >/dev/null
  jq --exit-status '.permissions.deny | index("Bash(git switch main *)")' .claude/settings.json >/dev/null
}

@test "install refuses a second time and names update" {
  "${kit_checkout}/bin/kit" install
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: kit update"* ]]
}

@test "check passes after install and fails on a drifted, a missing, and a lost deny entry" {
  "${kit_checkout}/bin/kit" install claude
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'edited\n' >> .kit/rules/yaml.md
  rm .claude/rules/kit-json.md
  jq '.permissions.deny -= ["Bash(git reset --hard *)"]' .claude/settings.json > s && mv s .claude/settings.json
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/yaml.md differs"* ]]
  [[ "$output" == *".claude/rules/kit-json.md is missing"* ]]
  [[ "$output" == *"missing an entry the kit writes"* ]]
  [[ "$output" == *"3 finding(s)"* ]]
}

@test "check refuses without a manifest and names install" {
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: kit install"* ]]
}

@test "update replaces unchanged files, merges an edited one, and records the release's digests" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  # a consumer edit to a file the later release also changes, on a different line
  sed -i.bak 's/^# JSON$/# JSON (ours)/' .kit/rules/json.md && rm .kit/rules/json.md.bak
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  grep --quiet '^version 9.9.9$' .kit/manifest
  grep --quiet '^# JSON (ours)$' .kit/rules/json.md
  grep --quiet '^## Added in 9.9.9$' .kit/rules/json.md
  grep --quiet '^## Added in 9.9.9$' .claude/rules/kit-json.md
  # the manifest carries the release's digest, so the edit still reads as an edit
  release_digest=$(shasum -a 256 "${repo}.later/tree/.kit/rules/json.md" | cut -d ' ' -f 1)
  grep --quiet "^${release_digest}  .kit/rules/json.md$" .kit/manifest
  # and check names the one file that still differs from the release
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/json.md differs"* ]]
  [[ "$output" == *"1 finding(s)"* ]]
  rm -rf "${repo}.later"
}

@test "update of an untouched install leaves check green" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
  rm -rf "${repo}.later"
}

@test "update leaves conflict markers and fails when the same lines changed" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  sed -i.bak 's/^# JSON$/# JSON (theirs)/' "${repo}.later/tree/.kit/rules/json.md" && rm "${repo}.later/tree/.kit/rules/json.md.bak"
  sed -i.bak 's/^# JSON$/# JSON (ours)/' .kit/rules/json.md && rm .kit/rules/json.md.bak
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  grep --quiet '^<<<<<<<' .kit/rules/json.md
  [[ "$output" == *"conflict markers"* ]]
  rm -rf "${repo}.later"
}

@test "update without the base release refuses an edited file and names the base" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  printf 'edited\n' >> .kit/rules/bash.md
  # a mise that answers nothing, ahead of the real one
  mkdir shims && printf '#!/usr/bin/env bash\nexit 1\n' > shims/mise && chmod +x shims/mise
  run env PATH="${PWD}/shims:${PATH}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *"base release 0.1.0 is not installed"* ]]
  rm -rf "${repo}.later"
}

# ----------------------------------------------------------------------------------------------------------------------
