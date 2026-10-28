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
  export KIT_HOME="${repo}.home" XDG_CACHE_HOME="${repo}.cache"
  mkdir "${KIT_HOME}" "${XDG_CACHE_HOME}"
  cd "${repo}"
}

teardown() {
  rm -rf "${repo}" "${repo}.later" "${repo}.home" "${repo}.cache" "${repo}.shims"
}

# shims <tool>… puts a failing stand-in for each tool first on PATH, so a test proves a path is not taken.
shims() {
  local tool
  mkdir -p "${repo}.shims"
  for tool in "$@"
  do
    printf '#!/usr/bin/env bash\nexit 1\n' > "${repo}.shims/${tool}" && chmod +x "${repo}.shims/${tool}"
  done
  printf '%s:%s\n' "${repo}.shims" "${PATH}"
}

# later-release <dir> copies this checkout's release to <dir> as version 9.9.9 with one changed rule line.
later-release() {
  mkdir -p "$1"
  cp -R "${kit_checkout}/bin" "${kit_checkout}/modules" "$1/"
  printf '9.9.9\n' > "$1/version"
  printf '\n## Added in 9.9.9\n\nA new rule.\n' >> "$1/modules/kit/files/repo/.kit/rules/json.md"
}

@test "install with no module writes the kit module and nothing else" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ -f .kit/rules/bash.md ]
  [ ! -e .kit/lefthook.yml ]
  [ ! -e .github ]
  [ ! -e .claude ]
  grep --quiet '^version unreleased$' .kit/manifest
  grep --quiet '^modules kit$' .kit/manifest
  run "${kit_checkout}/bin/kit" install ci-github
  [ "$status" -eq 1 ]
  run "${kit_checkout}/bin/kit" module add ci-github
  [ "$status" -eq 0 ]
  grep --quiet '^    runs-on: "ubuntu-latest"$' .github/workflows/kit-ci.yml
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
  grep --quiet '^modules kit claude$' .kit/manifest
  jq --exit-status '.permissions.deny | index("Bash(git rebase *)")' .claude/settings.json >/dev/null
  jq --exit-status '.hooks.PreToolUse[0].hooks[0].command == "kit context --claude"' .claude/settings.json >/dev/null
  grep --quiet --line-regexp '@../AGENTS.md' .claude/CLAUDE.md
  grep --quiet --line-regexp '<!-- kit:generate .claude/CLAUDE.md -->' .claude/CLAUDE.md
  [ ! -e .claude/rules ]
}

@test "install refuses a module this release does not know and names the known ones" {
  run "${kit_checkout}/bin/kit" install gemini
  [ "$status" -eq 2 ]
  [[ "$output" == *"the modules are: ci-github claude kit lefthook"* ]]
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
  grep --quiet '^modules kit claude$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" module remove claude
  [ "$status" -eq 0 ]
  [ ! -e .claude/CLAUDE.md ]
  [ "$(jq --raw-output .model .claude/settings.json)" = "x" ]
  jq --exit-status '.permissions.deny == ["Bash(rm -rf /)"]' .claude/settings.json >/dev/null
  jq --exit-status '.hooks == {"PreToolUse": []}' .claude/settings.json >/dev/null
  grep --quiet '^modules kit$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "module remove refuses a module another active module depends on, and removes a leaf" {
  "${kit_checkout}/bin/kit" install claude ci-github
  run "${kit_checkout}/bin/kit" module remove kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit cannot be removed while"* ]]
  [ -f .kit/1st.md ]
  run "${kit_checkout}/bin/kit" module remove ci-github
  [ "$status" -eq 0 ]
  [ ! -e .github/workflows/kit-ci.yml ]
  grep --quiet '^modules kit claude$' .kit/manifest
}

@test "a template takes the repository's answer over the module's default, and refuses a key with neither" {
  mkdir .kit && printf 'answers:\n  ci_runner: blacksmith-2vcpu\n' > .kit/settings.yml
  run "${kit_checkout}/bin/kit" install ci-github
  [ "$status" -eq 0 ]
  grep --quiet '^    runs-on: "blacksmith-2vcpu"$' .github/workflows/kit-ci.yml
  grep --quiet 'uses: actions/checkout@' .github/workflows/kit-ci.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  later-release "${repo}.later"
  printf 'runner @ci_runner@ owner @owner@\n' > "${repo}.later/modules/ci-github/templates/repo/OWNER"
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
  mkdir -p "${repo}.later/modules/claude/files/home/.kit"
  printf 'user level\n' > "${repo}.later/modules/claude/files/home/.kit/claude"
  run "${repo}.later/bin/kit" install claude
  [ "$status" -eq 0 ]
  [ "$(cat "${KIT_HOME}/.kit/claude")" = "user level" ]
  grep --quiet '^version 9.9.9$' "${KIT_HOME}/.kit/manifest"
  grep --quiet '  .kit/claude$' "${KIT_HOME}/.kit/manifest"
  ! grep --quiet 'claude$' .kit/manifest
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

@test "a module the repository defines under .kit/modules installs, checks, updates, and removes like a delivered one" {
  mkdir -p .kit/modules/python/files/repo/.kit/rules .kit/modules/python/templates/repo
  printf 'description: "The Python rule and the tool pins"\ndepends:\n  - kit\nanswers:\n  python: "3.13"\n' \
    > .kit/modules/python/module.yml
  printf -- '---\npaths:\n  - "**/*.py"\n---\n# Python\n\nType everything.\n' \
    > .kit/modules/python/files/repo/.kit/rules/python.md
  printf 'python @python@\n' > .kit/modules/python/templates/repo/.python-version
  run "${kit_checkout}/bin/kit" install python
  [ "$status" -eq 0 ]
  grep --quiet '^modules kit python$' .kit/manifest
  [ "$(cat .python-version)" = "python 3.13" ]
  run "${kit_checkout}/bin/kit" module list
  [[ "$output" == *"python     repository"* ]]
  [[ "$output" == *"kit        release"* ]]
  run "${kit_checkout}/bin/kit" context app.py
  [ "$output" = ".kit/rules/python.md" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'edited\n' >> .kit/rules/python.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/python.md differs"* ]]
  printf 'answers:\n  python: "3.14"\n' > .kit/settings.yml
  run env PATH="$(shims mise gh)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(cat .python-version)" = "python 3.14" ]
  grep --quiet '^edited$' .kit/rules/python.md
  run "${kit_checkout}/bin/kit" module remove python
  [ "$status" -eq 0 ]
  [ ! -e .python-version ]
  [ ! -e .kit/rules/python.md ]
  [ -f .kit/modules/python/module.yml ]
}

@test "the settings file is held to its schema: an unknown key and a non-boolean require_signing are refused" {
  mkdir .kit && printf 'forgery: github\n' > .kit/settings.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/settings.yml has a key the kit does not know: forgery (see .kit/settings.schema.json)"* ]]
  printf 'require_signing: yes\n' > .kit/settings.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"require_signing must be true or false"* ]]
  printf 'require_signing: false\nanswers:\n  ci_runner: x\n' > .kit/settings.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -f .kit/settings.schema.json ]
}

@test "a module the repository defines is validated like a delivered one" {
  mkdir -p .kit/modules/odd && printf 'description: "x"\ndependz:\n  - kit\n' > .kit/modules/odd/module.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/modules/odd/module.yml has a key the kit does not know: dependz"* ]]
}

@test "a region is placed inside the repository's own file, held to the release, and taken out whole" {
  printf '# Ours\n\nOur own instructions.\n' > AGENTS.md
  printf 'build/\n' > .gitignore
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ "$(sed -n '1,3p' AGENTS.md | paste -s -d '|' -)" = "# Ours||Our own instructions." ]
  grep --quiet --line-regexp '<!-- kit:generate AGENTS.md -->' AGENTS.md
  grep --quiet --line-regexp '<!-- kit:end AGENTS.md -->' AGENTS.md
  grep --quiet 'Read .kit/1st.md at the start of every session' AGENTS.md
  [ "$(sed -n '1p' .gitignore)" = "build/" ]
  grep --quiet --line-regexp '/worktrees/' .gitignore
  grep --quiet '  AGENTS.md$' .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  printf '## More of ours\n' >> AGENTS.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  sed -i.bak 's/^Read .kit\/1st.md/Skip .kit\/1st.md/' AGENTS.md && rm AGENTS.md.bak
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"AGENTS.md: stale; run: kit generate"* ]]
  run "${kit_checkout}/bin/kit" generate
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: regenerated"* ]]
  grep --quiet 'Read .kit/1st.md' AGENTS.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  # a region is what its command makes it: update regenerates an edited one rather than merging it
  sed -i.bak 's/^Read .kit\/1st.md/Skip .kit\/1st.md/' AGENTS.md && rm AGENTS.md.bak
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: the region regenerated"* ]]
  grep --quiet 'Read .kit/1st.md' AGENTS.md
  run "${kit_checkout}/bin/kit" module remove kit
  [ "$status" -eq 0 ]
  [ "$(cat AGENTS.md | paste -s -d '|' -)" = "# Ours||Our own instructions.|## More of ours" ]
  [ "$(cat .gitignore)" = "build/" ]
}

@test "a symlinked destination keeps its link, and a directory at a destination is refused" {
  mkdir docs && printf '# Ours\n' > docs/AGENTS.md && ln -s docs/AGENTS.md AGENTS.md
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -L AGENTS.md ]
  grep --quiet --line-regexp '<!-- kit:generate AGENTS.md -->' docs/AGENTS.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" module remove kit
  [ "$status" -eq 0 ]
  mkdir -p .kit/1st.md
  run "${kit_checkout}/bin/kit" module add kit
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/1st.md: a directory, or something that is not a file, is there"* ]]
}

@test "a region that opens and never closes is refused by name, and two modules cannot share a region's file" {
  printf '<!-- kit:generate AGENTS.md -->\nhalf\n' > AGENTS.md
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"AGENTS.md: the kit's region opens with '<!-- kit:generate AGENTS.md -->' and never closes"* ]]
  rm AGENTS.md
  mkdir -p .kit/modules/extra/regions/repo
  printf 'description: "x"\ndepends:\n  - kit\n' > .kit/modules/extra/module.yml
  printf 'extra\n' > .kit/modules/extra/regions/repo/AGENTS.md
  run "${kit_checkout}/bin/kit" install extra
  [ "$status" -eq 1 ]
  [[ "$output" == *"modules kit and extra both deliver a region to AGENTS.md"* ]]
  printf 'x = 1\n' > .kit/modules/extra/regions/repo/config.toml
  rm .kit/modules/extra/regions/repo/AGENTS.md
  run "${kit_checkout}/bin/kit" install extra
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' config.toml)" = '# kit:generate config.toml' ]
  [ "$("${kit_checkout}/bin/kit" region config.toml)" = "x = 1" ]
  printf 'const x = 1;\n' > .kit/modules/extra/regions/repo/lib.js
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' lib.js)" = '// kit:generate lib.js' ]
  [ "$(sed -n '3p' lib.js)" = '// kit:end lib.js' ]
  run "${kit_checkout}/bin/kit" region AGENTS.md
  [ "$status" -eq 0 ]
  [[ "$output" == "Read .kit/1st.md"* ]]
  run env KIT_GENERATE_REGION=AGENTS.md "${kit_checkout}/bin/kit" region
  [ "$status" -eq 0 ]
  [[ "$output" == "Read .kit/1st.md"* ]]
  run "${kit_checkout}/bin/kit" region
  [ "$status" -eq 2 ]
  [[ "$output" == *"usage: kit region"* ]]
  run "${kit_checkout}/bin/kit" region README.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"no active module delivers a region at README.md"* ]]
}

@test "a region makes the file where there is none, and one holding different content is regenerated" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' AGENTS.md)" = '<!-- kit:generate AGENTS.md -->' ]
  run "${kit_checkout}/bin/kit" module remove kit
  [ ! -e AGENTS.md ]
  [ ! -e .gitignore ]
  printf '# Ours\n\n<!-- kit:generate AGENTS.md -->\nsomething else\n<!-- kit:end AGENTS.md -->\n' > AGENTS.md
  run "${kit_checkout}/bin/kit" module add kit
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: placed, the region regenerated"* ]]
  [ "$(sed -n '1p' AGENTS.md)" = "# Ours" ]
  grep --quiet 'Read .kit/1st.md' AGENTS.md
  ! grep --quiet 'something else' AGENTS.md
}

@test "a module's expected include is named at install, refused by check until it is there, and never written" {
  run "${kit_checkout}/bin/kit" install lefthook
  [ "$status" -eq 0 ]
  [[ "$output" == *"left to do in lefthook.yml: put extends: [.kit/lefthook.yml] in lefthook.yml"* ]]
  [ ! -e lefthook.yml ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"lefthook.yml: put extends: [.kit/lefthook.yml] in lefthook.yml"* ]]
  printf 'pre-commit:\n  commands: {}\n' > lefthook.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  printf 'extends:\n  - .kit/lefthook.yml\n' > lefthook.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  [ "$(cat lefthook.yml)" = "$(printf 'extends:\n  - .kit/lefthook.yml')" ]
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
  "${kit_checkout}/bin/kit" install claude lefthook
  printf 'extends:\n  - .kit/lefthook.yml\n' > lefthook.yml
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
  [[ "$output" == *".kit/rules/toml.md: a different file is there already"* ]]
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
  theirs="${repo}.later/modules/kit/files/repo/.kit/rules/json.md"
  sed -i.bak 's/^# JSON$/# JSON (theirs)/' "${theirs}" && rm "${theirs}.bak"
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
  ! grep --quiet 'rules/yaml.md' .kit/manifest
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "update without the base release refuses an edited file, names the base, and leaves the manifest's version" {
  later-release "${repo}.later"
  "${repo}.later/bin/kit" install claude
  grep --quiet '^version 9.9.9$' .kit/manifest
  mkdir -p "${repo}.newer" && cp -R "${repo}.later/bin" "${repo}.later/modules" "${repo}.newer/"
  printf '9.9.10\n' > "${repo}.newer/version"
  printf 'edited\n' >> .kit/rules/bash.md
  # neither the cache, nor mise, nor the release can produce 9.9.9
  run env PATH="$(shims mise curl)" "${repo}.newer/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *"could not be fetched"* ]]
  [[ "$output" == *".kit/rules/bash.md cannot be updated without the base release 9.9.9"* ]]
  grep --quiet '^version 9.9.9$' .kit/manifest
  rm -rf "${repo}.newer"
}

@test "update at the same version uses the release as its own base, and a second run changes nothing" {
  "${kit_checkout}/bin/kit" install claude
  printf 'edited\n' >> .kit/rules/bash.md
  run env PATH="$(shims mise gh)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  grep --quiet '^edited$' .kit/rules/bash.md
  run env PATH="$(shims mise gh)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [[ "$output" != *"conflict"* ]]
}

@test "the verbs run from a subdirectory as from the root" {
  "${kit_checkout}/bin/kit" install
  mkdir -p docs/deep && cd docs/deep
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" module add claude
  [ "$status" -eq 0 ]
  [ -f "${repo}/.claude/settings.json" ]
  [ ! -e .claude ]
}

@test "install and module add refuse a different file at a delivered path, and accept the same one" {
  mkdir .kit && printf 'mine\n' > .kit/1st.md
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/1st.md: a different file is there already"* ]]
  [ ! -e .kit/manifest ]
  [ "$(cat .kit/1st.md)" = "mine" ]
  rm .kit/1st.md
  "${kit_checkout}/bin/kit" install
  printf 'theirs\n' > .kit/lefthook.yml
  run "${kit_checkout}/bin/kit" module add lefthook
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/lefthook.yml: a different file is there already"* ]]
  grep --quiet '^modules kit$' .kit/manifest
  rm .kit/lefthook.yml
  mkdir .claude && printf '# Ours\n' > .claude/CLAUDE.md
  run "${kit_checkout}/bin/kit" module add claude lefthook
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' .claude/CLAUDE.md)" = "# Ours" ]
  grep --quiet --line-regexp '@../AGENTS.md' .claude/CLAUDE.md
}

@test "module remove judges dependents against what stays, in any order" {
  "${kit_checkout}/bin/kit" install claude
  run "${kit_checkout}/bin/kit" module remove kit lefthook claude
  [ "$status" -eq 0 ]
  grep --quiet '^modules$' .kit/manifest
  [ ! -e .kit/1st.md ]
}

@test "a home manifest another repository wrote is left alone, and its surviving lines are kept" {
  mkdir -p "${KIT_HOME}/.kit" && printf 'theirs\n' > "${KIT_HOME}/.kit/theirs"
  digest=$(shasum -a 256 "${KIT_HOME}/.kit/theirs" | cut -d ' ' -f 1)
  printf '# manifest\nversion 0.0.1\nmodules other\n%s  .kit/theirs\n' "${digest}" > "${KIT_HOME}/.kit/manifest"
  "${kit_checkout}/bin/kit" install
  [ -f "${KIT_HOME}/.kit/theirs" ]
  grep --quiet '  .kit/theirs$' "${KIT_HOME}/.kit/manifest"
  grep --quiet '^version 0.0.1$' "${KIT_HOME}/.kit/manifest"
}

@test "a region is a template: @key@ takes the repository's answer, the module's default, or a settings key, and refuses a key with none" {
  mkdir -p .kit/modules/extra/regions/repo
  printf 'description: "x"\ndepends:\n  - kit\nanswers:\n  ci_runner: ours\n' > .kit/modules/extra/module.yml
  printf 'runner = "@ci_runner@"\nsigning = @require_signing@\nagain = "@answers.ci_runner@"\n' > .kit/modules/extra/regions/repo/config.toml
  printf 'require_signing: false\n' > .kit/settings.yml
  run "${kit_checkout}/bin/kit" install extra
  [ "$status" -eq 0 ]
  [ "$(sed -n '2,4p' config.toml | paste -s -d '|' -)" = 'runner = "ours"|signing = false|again = "ours"' ]
  [ "$("${kit_checkout}/bin/kit" region config.toml | paste -s -d '|' -)" = 'runner = "ours"|signing = false|again = "ours"' ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'require_signing: false\nanswers:\n  ci_runner: theirs\n' > .kit/settings.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"config.toml: stale"* ]]
  run "${kit_checkout}/bin/kit" generate
  [ "$status" -eq 0 ]
  grep --quiet '^runner = "theirs"$' config.toml
  grep --quiet '^again = "theirs"$' config.toml
  printf 'x = @missing@\n' > .kit/modules/extra/regions/repo/other.toml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"other.toml needs an answer for 'missing'; add 'missing: <value>' under answers: in .kit/settings.yml (module extra)"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
