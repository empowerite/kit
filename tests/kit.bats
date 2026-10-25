#!/usr/bin/env bats
# kit.bats - bin/kit install, module, check, and update against a fresh repository
# ----------------------------------------------------------------------------------------------------------------------
#
# Each test gets an empty git repository under a temporary directory and runs the kit from this checkout against
# it. The update tests build a second, later release from the first so the three-way merge has a base.

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  repo=$(mktemp -d)
  git -C "${repo}" init --quiet --initial-branch=main
  # a home directory of the test's own, so no test ever writes under the real one
  export KIT_HOME="${repo}.home"
  mkdir "${KIT_HOME}"
  cd "${repo}"
}

teardown() {
  rm -rf "${repo}" "${repo}.later" "${repo}.home"
}

# later-release <dir> copies this checkout's release to <dir> as version 9.9.9 with one changed rule line.
later-release() {
  mkdir -p "$1"
  cp -R "${kit_checkout}/bin" "${kit_checkout}/modules" "$1/"
  printf '9.9.9\n' > "$1/version"
  printf '\n## Added in 9.9.9\n\nA new rule.\n' >> "$1/modules/kit/files/repo/.kit/rules/json.md"
}

@test "install with no module writes the default modules and nothing under .claude" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ -f .kit/rules/bash.md ]
  [ -f .kit/lefthook.yml ]
  [ -f .github/workflows/kit-ci.yml ]
  grep --quiet '^    runs-on: "ubuntu-latest"$' .github/workflows/kit-ci.yml
  [ ! -e .claude ]
  grep --quiet '^version unreleased$' .kit/manifest
  grep --quiet '^modules kit ci lefthook$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  # a stray file under a module not active is nobody's business
  mkdir -p .claude && printf '{"model":"x"}\n' > .claude/settings.json
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "install with claude adds the deny list and the context hook, and copies no rule" {
  run "${kit_checkout}/bin/kit" install claude
  [ "$status" -eq 0 ]
  grep --quiet '^modules kit ci claude lefthook$' .kit/manifest
  jq --exit-status '.permissions.deny | index("Bash(git rebase *)")' .claude/settings.json >/dev/null
  jq --exit-status '.hooks.PreToolUse[0].hooks[0].command == "kit context --claude"' .claude/settings.json >/dev/null
  [ "$(tail -n 1 .claude/CLAUDE.md)" = "@../AGENTS.md" ]
  [ ! -e .claude/rules ]
}

@test "install refuses a module this release does not know and names the known ones" {
  run "${kit_checkout}/bin/kit" install gemini
  [ "$status" -eq 2 ]
  [[ "$output" == *"the modules are: ci claude kit lefthook"* ]]
  [ ! -e .kit/manifest ]
}

@test "module add places a module's files, and module remove takes them and the kit's entries back out" {
  mkdir .claude
  printf '{"model":"x","permissions":{"deny":["Bash(rm -rf /)"]}}\n' > .claude/settings.json
  "${kit_checkout}/bin/kit" install
  run "${kit_checkout}/bin/kit" module list
  [[ "$output" == *"✅ kit "* ]]
  [[ "$output" == *"   claude "* ]]
  run "${kit_checkout}/bin/kit" module add claude
  [ "$status" -eq 0 ]
  jq --exit-status '.permissions.deny | index("Bash(git switch main *)")' .claude/settings.json >/dev/null
  grep --quiet '^modules kit ci claude lefthook$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" module remove claude
  [ "$status" -eq 0 ]
  [ ! -e .claude/CLAUDE.md ]
  [ "$(jq --raw-output .model .claude/settings.json)" = "x" ]
  jq --exit-status '.permissions.deny == ["Bash(rm -rf /)"]' .claude/settings.json >/dev/null
  jq --exit-status '.hooks == {"PreToolUse": []}' .claude/settings.json >/dev/null
  grep --quiet '^modules kit ci lefthook$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "module remove refuses a module another active module depends on, and removes a leaf" {
  "${kit_checkout}/bin/kit" install claude
  run "${kit_checkout}/bin/kit" module remove kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit cannot be removed while"* ]]
  [ -f .kit/1st.md ]
  run "${kit_checkout}/bin/kit" module remove ci
  [ "$status" -eq 0 ]
  [ ! -e .github/workflows/kit-ci.yml ]
  grep --quiet '^modules kit claude lefthook$' .kit/manifest
}

@test "a template takes the repository's answer over the module's default, and refuses a key with neither" {
  mkdir .kit && printf 'ci_runner blacksmith-2vcpu\n' > .kit/answers
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  grep --quiet '^    runs-on: "blacksmith-2vcpu"$' .github/workflows/kit-ci.yml
  grep --quiet 'uses: actions/checkout@' .github/workflows/kit-ci.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  later-release "${repo}.later"
  printf 'runner @ci_runner@ owner @owner@\n' > "${repo}.later/modules/ci/templates/repo/OWNER"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *"OWNER needs an answer for 'owner'"* ]]
  [ ! -e OWNER ]
}

@test "a module whose metadata has a key the schema does not know is refused, naming the key" {
  later-release "${repo}.later"
  printf 'description: "x"\ndependz:\n  - kit\n' > "${repo}.later/modules/lefthook/module.yml"
  run "${repo}.later/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"modules/lefthook/module.yml has a key the kit does not know: dependz"* ]]
  [ ! -e .kit/manifest ]
  printf 'description: "x"\ndefault: yes\n' > "${repo}.later/modules/lefthook/module.yml"
  run "${repo}.later/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"default must be true or false"* ]]
}

@test "a module delivering to a base the kit does not know is refused, naming the bases" {
  later-release "${repo}.later"
  mkdir -p "${repo}.later/modules/kit/files/etc" && printf 'x\n' > "${repo}.later/modules/kit/files/etc/.x"
  run "${repo}.later/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"modules/kit/files/etc names a base the kit does not deliver to; the bases are: repo home"* ]]
  [ ! -e .kit/manifest ]
}

@test "a module delivering to home lands under it with a manifest of its own, and leaves with it" {
  later-release "${repo}.later"
  mkdir -p "${repo}.later/modules/claude/files/home/.kit" && printf 'user level\n' > "${repo}.later/modules/claude/files/home/.kit/claude"
  run "${repo}.later/bin/kit" install claude
  [ "$status" -eq 0 ]
  [ "$(cat "${KIT_HOME}/.kit/claude")" = "user level" ]
  grep --quiet '^version 9.9.9$' "${KIT_HOME}/.kit/manifest"
  grep --quiet '  .kit/claude$' "${KIT_HOME}/.kit/manifest"
  grep --quiet --invert-match 'claude$' .kit/manifest
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'edited\n' >> "${KIT_HOME}/.kit/claude"
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"/.kit/claude differs"* ]]
  run "${repo}.later/bin/kit" module remove claude
  [ "$status" -eq 0 ]
  [ ! -e "${KIT_HOME}/.kit/claude" ]
  [ ! -e "${KIT_HOME}/.kit/manifest" ]
}

@test "check skips the home base where nothing was installed there, and says so" {
  later-release "${repo}.later"
  mkdir -p "${repo}.later/modules/kit/files/home/.kit" && printf 'x\n' > "${repo}.later/modules/kit/files/home/.kit/x"
  "${repo}.later/bin/kit" install
  rm -rf "${KIT_HOME}/.kit"
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
  [[ "$output" == *"home: nothing is installed under ${KIT_HOME} on this machine"* ]]
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
  rm .kit/lefthook.yml
  jq '.permissions.deny -= ["Bash(git reset --hard *)"]' .claude/settings.json > s && mv s .claude/settings.json
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/yaml.md differs"* ]]
  [[ "$output" == *".kit/lefthook.yml is missing"* ]]
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
  # the manifest carries the release's digest, so the edit still reads as an edit
  release_digest=$(shasum -a 256 "${repo}.later/modules/kit/files/repo/.kit/rules/json.md" | cut -d ' ' -f 1)
  grep --quiet "^${release_digest}  .kit/rules/json.md$" .kit/manifest
  # and check names the one file that still differs from the release
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/json.md differs"* ]]
  [[ "$output" == *"1 finding(s)"* ]]
}

@test "update refuses a path new in the release where the repository already has a different file" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  printf 'new rule\n' > "${repo}.later/modules/kit/files/repo/.kit/rules/toml.md"
  printf 'mine\n' > .kit/rules/toml.md
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/toml.md is new in 9.9.9"* ]]
  [ "$(cat .kit/rules/toml.md)" = "mine" ]
}

@test "update of an untouched install leaves check green" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "update leaves conflict markers and fails when the same lines changed" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  sed -i.bak 's/^# JSON$/# JSON (theirs)/' "${repo}.later/modules/kit/files/repo/.kit/rules/json.md" && rm "${repo}.later/modules/kit/files/repo/.kit/rules/json.md.bak"
  sed -i.bak 's/^# JSON$/# JSON (ours)/' .kit/rules/json.md && rm .kit/rules/json.md.bak
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  grep --quiet '^<<<<<<<' .kit/rules/json.md
  [[ "$output" == *"conflict markers"* ]]
}

@test "update retires a path the new release no longer delivers, unless it was edited" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  rm "${repo}.later/modules/kit/files/repo/.kit/rules/yaml.md" "${repo}.later/modules/kit/files/repo/.kit/rules/json.md"
  printf 'mine too\n' >> .kit/rules/json.md
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  [ ! -e .kit/rules/yaml.md ]
  [ -e .kit/rules/json.md ]
  [[ "$output" == *".kit/rules/yaml.md: retired in 9.9.9, removed"* ]]
  [[ "$output" == *".kit/rules/json.md: retired in 9.9.9, but edited"* ]]
  grep --quiet --invert-match 'rules/yaml.md' .kit/manifest
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "update reads a manifest written before modules existed as the default modules plus its targets" {
  "${kit_checkout}/bin/kit" install claude
  sed -i.bak 's/^modules .*$/targets claude/' .kit/manifest && rm .kit/manifest.bak
  grep --quiet '^targets claude$' .kit/manifest
  later-release "${repo}.later"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  grep --quiet '^modules kit ci claude lefthook$' .kit/manifest
  jq --exit-status '.hooks.PreToolUse[0].hooks[0].command == "kit context --claude"' .claude/settings.json >/dev/null
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "update without the base release refuses an edited file and names the base" {
  "${kit_checkout}/bin/kit" install claude
  later-release "${repo}.later"
  printf 'edited\n' >> .kit/rules/bash.md
  # a mise that answers nothing, ahead of the real one
  mkdir shims && printf '#!/usr/bin/env bash\nexit 1\n' > shims/mise && chmod +x shims/mise
  run env PATH="${PWD}/shims:${PATH}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *"cannot be updated without the base release unreleased"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
