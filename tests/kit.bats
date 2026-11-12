#!/usr/bin/env bats
# kit.bats - bin/kit install, package, check, and update against a fresh repository
# ----------------------------------------------------------------------------------------------------------------------
# Each test gets an empty git repository under a temporary directory and runs the kit from this checkout against
# it. The update tests build a second, later release from the first so the three-way merge has a base.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo
}

# tree-digest prints each file and link in the repository, with a digest of what it holds: a link's target, a file's
# bytes. .git and the log every kit command appends to are left out.
tree-digest() {
  local path
  while IFS= read -r path
  do
    if [[ -L "${path}" ]]
    then
      printf '%s -> %s\n' "${path}" "$(readlink "${path}")"
    else
      printf '%s %s\n' "${path}" "$(cksum < "${path}")"
    fi
  done < <(find . \( -path ./.git -o -path ./.kit/logs \) -prune -o \( -type f -o -type l \) -print | LC_ALL=C sort)
}

@test "install with no package writes the kit package and nothing else" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ -f .kit/rules/bash.md ]
  [ ! -e .kit/lefthook.yml ]
  [ ! -e .github ]
  [ ! -e .claude ]
  [ "$(declared version)" = "unreleased" ]
  [ "$(declared packages)" = "kit" ]
  run "${kit_checkout}/bin/kit" install github/ci
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: cd "*" && kit update"* ]]
  run "${kit_checkout}/bin/kit" packages add github/ci
  [ "$status" -eq 0 ]
  grep --quiet '^    runs-on: "ubuntu-latest"$' .github/workflows/kit-ci.yml
  grep --quiet "sed -n 's/^version: \*//p' kit.yml" .github/workflows/kit-ci.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  # a stray file under a package not active is nobody's business
  mkdir -p .claude && printf '{"model":"x"}\n' > .claude/settings.json
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "AGENTS.md comes from agents/generic, which every agents package depends on, and kit alone writes none" {
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ ! -e AGENTS.md ]
  grep --quiet --line-regexp '/worktrees/' .gitignore
  for package in claude cursor copilot gemini
  do
    grep --quiet --line-regexp '  - agents/generic' "${kit_checkout}/packages/agents/${package}/package.yml"
  done
  run "${kit_checkout}/bin/kit" packages add agents/generic
  [ "$status" -eq 0 ]
  grep --quiet 'Read .kit/1st.md at the start of every session' AGENTS.md
  run "${kit_checkout}/bin/kit" packages remove agents/generic
  [ "$status" -eq 0 ]
  [ ! -e AGENTS.md ]
}

@test "a host template makes the file, places the region in one without it, refuses one it cannot place, and holds the shape" {
  printf 'Our rules, and no title.\n' > AGENTS.md
  run "${kit_checkout}/bin/kit" install agents/generic
  [ "$status" -eq 1 ]
  [[ "$output" == *"AGENTS.md does not open as the agents/generic package's host template has it"* ]]
  [ "$(cat AGENTS.md)" = "Our rules, and no title." ]
  printf '# Ours\nOur rules.\n' > AGENTS.md
  run "${kit_checkout}/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  [ "$(paste -s -d '|' - < AGENTS.md | sed 's/|<!-- kit:generate.*kit:end AGENTS.md -->|/|REGION|/')" = "# Ours||REGION||Our rules." ]
  [ "$(sed -n '1,3p' .claude/CLAUDE.md | paste -s -d '|' -)" = "# CLAUDE.md||<!-- kit:generate .claude/CLAUDE.md -->" ]
  [[ "$output" == *".claude/CLAUDE.md: Claude Code reads it when a session starts"* ]]
  [[ "$output" == *".claude/settings.json: Claude Code reloads its settings when they change"* ]]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  sed -i.bak '1d' AGENTS.md && rm AGENTS.md.bak
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"AGENTS.md does not have the shape of the agents/generic package's host template; it leaves AGENTS.md at line 1"* ]]
}

@test "install with claude adds the deny list and the context hook, and copies no rule" {
  run "${kit_checkout}/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  [ "$(declared packages)" = "kit agents/generic agents/claude" ]
  jq --exit-status '.permissions.deny | index("Bash(git rebase *)")' .claude/settings.json >/dev/null
  jq --exit-status '.hooks.PreToolUse[0].hooks[0].command == "kit context --claude"' .claude/settings.json >/dev/null
  grep --quiet --line-regexp '@../AGENTS.md' .claude/CLAUDE.md
  grep --quiet --line-regexp '<!-- kit:generate .claude/CLAUDE.md -->' .claude/CLAUDE.md
  [ ! -e .claude/rules ]
  grep --quiet 'Read .kit/1st.md at the start of every session' AGENTS.md
}

@test "install refuses a package this release does not know, in the words the form uses, and the choices list the known ones" {
  run "${kit_checkout}/bin/kit" install gemini
  [ "$status" -eq 2 ]
  [[ "$output" == *"<package> 'gemini' is not one of the choices; run: cd "*" && kit help install"* ]]
  run "${kit_checkout}/bin/kit" syntax install --choices '<package>'
  [[ "$(cut -f 1 <<<"$output" | paste -s -d ' ' -)" == "agents/claude agents/copilot agents/cursor agents/gemini agents/generic empowerite github/ci hooks/lefthook kit policy/opa style/bash"* ]]
  [ ! -e .kit/manifest ]
}

@test "package add places a package's files, and package remove takes them and the kit's entries back out" {
  mkdir .claude
  printf '{"model":"x","permissions":{"deny":["Bash(rm -rf /)"]}}\n' > .claude/settings.json
  "${kit_checkout}/bin/kit" install
  run "${kit_checkout}/bin/kit" packages
  [[ "$output" == *"✅ kit "* ]]
  [[ "$output" =~ "   agents/claude " ]]
  run "${kit_checkout}/bin/kit" packages add agents/claude
  [ "$status" -eq 0 ]
  jq --exit-status '.permissions.deny | index("Bash(git switch main *)")' .claude/settings.json >/dev/null
  [ "$(declared packages)" = "kit agents/generic agents/claude" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages remove agents/claude
  [ "$status" -eq 0 ]
  [ ! -e .claude/CLAUDE.md ]
  [ "$(jq --raw-output .model .claude/settings.json)" = "x" ]
  jq --exit-status '.permissions.deny == ["Bash(rm -rf /)"]' .claude/settings.json >/dev/null
  jq --exit-status 'has("hooks") | not' .claude/settings.json >/dev/null
  [ "$(declared packages)" = "kit agents/generic" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "package remove refuses a package another active package depends on, and removes a leaf" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci
  run "${kit_checkout}/bin/kit" packages remove kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit cannot be removed while"* ]]
  [ -f .kit/1st.md ]
  run "${kit_checkout}/bin/kit" packages remove github/ci
  [ "$status" -eq 0 ]
  [ ! -e .github/workflows/kit-ci.yml ]
  [ "$(declared packages)" = "kit agents/generic agents/claude" ]
}

@test "a template takes the repository's answer over the package's default, and refuses a key with neither" {
  mkdir .kit && printf 'answers:\n  ci_runner: blacksmith-2vcpu\n' > kit.yml
  run "${kit_checkout}/bin/kit" install github/ci
  [ "$status" -eq 0 ]
  grep --quiet '^    runs-on: "blacksmith-2vcpu"$' .github/workflows/kit-ci.yml
  grep --quiet 'uses: actions/checkout@' .github/workflows/kit-ci.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  later-release "${repo}.later"
  printf 'runner @ci_runner@ owner @owner@\n' > "${repo}.later/packages/github/ci/templates/repo/OWNER"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *"OWNER needs an answer for 'owner'"* ]]
  [ ! -e OWNER ]
}

@test "a package whose metadata has a key the schema does not know is refused, naming the key" {
  later-release "${repo}.later"
  printf 'description: "x"\ndependz:\n  - kit\n' > "${repo}.later/packages/hooks/lefthook/package.yml"
  run "${repo}.later/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"packages/hooks/lefthook/package.yml has a key the kit does not know: dependz"* ]]
  [ ! -e .kit/manifest ]
  printf 'description: "x"\ndefault: yes\n' > "${repo}.later/packages/hooks/lefthook/package.yml"
  run "${repo}.later/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"default must be true or false"* ]]
}

@test "a package delivering to a base the kit does not know is refused, naming the bases" {
  later-release "${repo}.later"
  mkdir -p "${repo}.later/packages/kit/files/etc" && printf 'x\n' > "${repo}.later/packages/kit/files/etc/.x"
  run "${repo}.later/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"packages/kit/files/etc names a base the kit does not deliver to; the bases are: repo home"* ]]
  [ ! -e .kit/manifest ]
}

@test "a package delivering to home lands under it with a manifest of its own, and leaves with it" {
  later-release "${repo}.later"
  mkdir -p "${repo}.later/packages/agents/claude/files/home/.kit"
  printf 'user level\n' > "${repo}.later/packages/agents/claude/files/home/.kit/claude"
  run "${repo}.later/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  [ "$(cat "${KIT_HOME}/.kit/claude")" = "user level" ]
  grep --quiet '^version 9.9.9$' "${KIT_HOME}/.kit/manifest"
  grep --quiet '  .kit/claude$' "${KIT_HOME}/.kit/manifest"
  run ! grep --quiet '  .kit/claude$' .kit/manifest
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'edited\n' >> "${KIT_HOME}/.kit/claude"
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"/.kit/claude differs"* ]]
  printf 'user level\n' > "${KIT_HOME}/.kit/claude"
  run "${repo}.later/bin/kit" packages remove agents/claude
  [ "$status" -eq 0 ]
  [ ! -e "${KIT_HOME}/.kit/claude" ]
  [ ! -e "${KIT_HOME}/.kit/manifest" ]
}

@test "kit packages remove names a home-base copy the person edited, and leaves it" {
  later-release "${repo}.later"
  mkdir -p "${repo}.later/packages/agents/claude/files/home/.kit"
  printf 'user level\n' > "${repo}.later/packages/agents/claude/files/home/.kit/claude"
  run "${repo}.later/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  printf 'edited\n' >> "${KIT_HOME}/.kit/claude"
  run "${repo}.later/bin/kit" packages remove agents/claude
  [ "$status" -eq 0 ]
  [[ "$output" == *"${KIT_HOME}/.kit/claude: edited since the kit placed it; it is left as it is"* ]]
  [ "$(cat "${KIT_HOME}/.kit/claude")" = "$(printf 'user level\nedited')" ]
}

@test "check skips the home base where nothing was installed there, and says so" {
  later-release "${repo}.later"
  mkdir -p "${repo}.later/packages/kit/files/home/.kit" && printf 'x\n' > "${repo}.later/packages/kit/files/home/.kit/x"
  "${repo}.later/bin/kit" install
  rm -rf "${KIT_HOME}/.kit"
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
  [[ "$output" == *"home: nothing is installed under ${KIT_HOME} on this machine"* ]]
}

@test "a package the repository defines in .kit/packages installs, checks, updates, and removes like a delivered one" {
  mkdir -p .kit/packages/python/files/repo/.kit/rules .kit/packages/python/files/repo/.kit/kinds \
    .kit/packages/python/templates/repo
  printf 'description: "The Python rule and the tool pins"\ndepends:\n  - kit\nanswers:\n  python: "3.13"\n' \
    > .kit/packages/python/package.yml
  printf '# The Python kind\nname: python\npaths:\n  - "**/*.py"\n' > .kit/packages/python/files/repo/.kit/kinds/python.yml
  printf -- '---\nkind:\n  - python\n---\n# Python\n\nType everything.\n' \
    > .kit/packages/python/files/repo/.kit/rules/python.md
  printf 'python @python@\n' > .kit/packages/python/templates/repo/.python-version
  run "${kit_checkout}/bin/kit" install python
  [ "$status" -eq 0 ]
  [ "$(declared packages)" = "kit python" ]
  [ "$(cat .python-version)" = "python 3.13" ]
  run "${kit_checkout}/bin/kit" packages
  [[ "$output" =~ python\ +repository ]]
  [[ "$output" =~ kit\ +release ]]
  run "${kit_checkout}/bin/kit" context app.py
  [ "$output" = ".kit/rules/python.md" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  # the rule is a link into the package, so an edit through it lands in the source and check stays green
  [ -L .kit/rules/python.md ]
  printf 'edited\n' >> .kit/rules/python.md
  grep --quiet '^edited$' .kit/packages/python/files/repo/.kit/rules/python.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  # the file now carries the declaration, so the repository's own keys are added to it
  printf 'answers:\n  python: "3.14"\n' >> kit.yml
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(cat .python-version)" = "python 3.14" ]
  grep --quiet '^edited$' .kit/rules/python.md
  run "${kit_checkout}/bin/kit" packages remove python
  [ "$status" -eq 0 ]
  [ ! -e .python-version ]
  [ ! -e .kit/rules/python.md ]
  [ -f .kit/packages/python/package.yml ]
}

@test "the settings file is held to its schema: an unknown key, status for flow, and a non-boolean require_signing are refused" {
  mkdir .kit && printf 'forgery: github\n' > kit.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml has a key the kit does not know: forgery (see .kit/kit.schema.json)"* ]]
  printf 'status: [desk]\n' > kit.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml has a key the kit does not know: status; the stages kit flow draws are named under flow:"* ]]
  printf 'require_signing: yes\n' > kit.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"require_signing must be true or false"* ]]
  printf 'linked_exempt: dependabot[bot]\n' > kit.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"linked_exempt must be a list of logins, one per line under it"* ]]
  printf 'require_signing: false\nanswers:\n  ci_runner: x\nlinked_exempt:\n  - dependabot[bot]\n' > kit.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  [ -f .kit/kit.schema.json ]
}

@test "a package the repository defines is validated like a delivered one" {
  mkdir -p .kit/packages/odd && printf 'description: "x"\ndependz:\n  - kit\n' > .kit/packages/odd/package.yml
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/packages/odd/package.yml has a key the kit does not know: dependz"* ]]
}

@test "a region is placed inside the repository's own file, held to the release, and taken out whole" {
  printf '# Ours\n\nOur own instructions.\n' > AGENTS.md
  printf 'build/\n' > .gitignore
  run "${kit_checkout}/bin/kit" install agents/generic
  [ "$status" -eq 0 ]
  # the host template puts the region after the title, with a blank line on either side
  [ "$(sed -n '1,3p' AGENTS.md | paste -s -d '|' -)" = "# Ours||<!-- kit:generate AGENTS.md -->" ]
  [ "$(tail -n 2 AGENTS.md | paste -s -d '|' -)" = "|Our own instructions." ]
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
  sed -i.bak 's/Read .kit\/1st.md/Skip .kit\/1st.md/' AGENTS.md && rm AGENTS.md.bak
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"AGENTS.md: stale; run: cd "*" && kit generate"* ]]
  run "${kit_checkout}/bin/kit" generate
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: regenerated"* ]]
  grep --quiet 'Read .kit/1st.md' AGENTS.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  # a region is what its command makes it: update regenerates an edited one rather than merging it
  sed -i.bak 's/Read .kit\/1st.md/Skip .kit\/1st.md/' AGENTS.md && rm AGENTS.md.bak
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: the region regenerated"* ]]
  grep --quiet 'Read .kit/1st.md' AGENTS.md
  run "${kit_checkout}/bin/kit" packages remove kit agents/generic
  [ "$status" -eq 0 ]
  [ "$(paste -s -d '|' - < AGENTS.md)" = "# Ours||Our own instructions.|## More of ours" ]
  [ "$(cat .gitignore)" = "build/" ]
}

@test "a symlinked destination keeps its link, and a directory at a destination is refused" {
  mkdir docs && printf '# Ours\n' > docs/AGENTS.md && ln -s docs/AGENTS.md AGENTS.md
  run "${kit_checkout}/bin/kit" install agents/generic
  [ "$status" -eq 0 ]
  [ -L AGENTS.md ]
  grep --quiet --line-regexp '<!-- kit:generate AGENTS.md -->' docs/AGENTS.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages remove kit agents/generic
  [ "$status" -eq 0 ]
  mkdir -p .kit/1st.md
  run "${kit_checkout}/bin/kit" packages add kit
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/1st.md: a directory, or something that is not a file, is there"* ]]
}

@test "a region that opens and never closes is refused by name, and two packages cannot share a region's file" {
  printf '<!-- kit:generate AGENTS.md -->\nhalf\n' > AGENTS.md
  run "${kit_checkout}/bin/kit" install agents/generic
  [ "$status" -eq 1 ]
  [[ "$output" == *"AGENTS.md: the kit's region opens with '<!-- kit:generate AGENTS.md -->' and never closes"* ]]
  rm AGENTS.md
  mkdir -p .kit/packages/extra/regions/repo
  printf 'description: "x"\ndepends:\n  - kit\n' > .kit/packages/extra/package.yml
  printf 'extra\n' > .kit/packages/extra/regions/repo/AGENTS.md
  run "${kit_checkout}/bin/kit" install agents/generic extra
  [ "$status" -eq 1 ]
  [[ "$output" == *"packages agents/generic and extra both deliver a region to AGENTS.md"* ]]
  printf 'x = 1\n' > .kit/packages/extra/regions/repo/config.toml
  rm .kit/packages/extra/regions/repo/AGENTS.md
  run "${kit_checkout}/bin/kit" install agents/generic extra
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' config.toml)" = '# kit:generate config.toml' ]
  [ "$("${kit_checkout}/bin/kit" region config.toml)" = "x = 1" ]
  printf 'const x = 1;\n' > .kit/packages/extra/regions/repo/lib.js
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' lib.js)" = '// kit:generate lib.js' ]
  [ "$(sed -n '3p' lib.js)" = '// kit:end lib.js' ]
  run "${kit_checkout}/bin/kit" region AGENTS.md
  [ "$status" -eq 0 ]
  [[ "$output" == "## Follow rules defined by kit"*"Read .kit/1st.md"* ]]
  run env KIT_GENERATE_REGION=AGENTS.md "${kit_checkout}/bin/kit" region
  [ "$status" -eq 0 ]
  [[ "$output" == "## Follow rules defined by kit"*"Read .kit/1st.md"* ]]
  run "${kit_checkout}/bin/kit" region
  [ "$status" -eq 2 ]
  [[ "$output" == *"run: cd "*" && kit help region"* ]]
  run "${kit_checkout}/bin/kit" region README.md
  [ "$status" -eq 1 ]
  [[ "$output" == *"no active package delivers a region at README.md"* ]]
}

@test "a region in a linked file is found under the link's name, so an update never adds a second" {
  mkdir shared && printf 'build/\n' > shared/gitignore && ln -s shared/gitignore .gitignore
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [ -L .gitignore ]
  [ "$(grep --count --line-regexp --fixed-strings '# kit:generate .gitignore' shared/gitignore)" -eq 1 ]
  grep --quiet --line-regexp 'build/' shared/gitignore
}

@test "markers out of order, or doubled, refuse the write and leave the file; a marker that starts with -- is text" {
  local before sound
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 0 ]
  sound=$(cat .gitignore)
  printf 'top\n# kit:end .gitignore\nmiddle\n# kit:generate .gitignore\nold\nbottom-1\nbottom-2\n' > .gitignore
  before=$(cat .gitignore)
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 1 ]
  [ "$(cat .gitignore)" = "${before}" ]
  [[ "$output" == *"'# kit:generate .gitignore' and '# kit:end .gitignore' must each be there once, in that order"* ]]
  printf '# kit:generate .gitignore\n/worktrees/\n# kit:end .gitignore\nmine\n# kit:generate .gitignore\ntail\n' \
    > .gitignore
  before=$(cat .gitignore)
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 1 ]
  [ "$(cat .gitignore)" = "${before}" ]
  [[ "$output" == *"'# kit:generate .gitignore' and '# kit:end .gitignore' must each be there once, in that order"* ]]
  grep --quiet --line-regexp 'tail' .gitignore
  printf '%s\n' "${sound}" > .gitignore
  mkdir -p .kit/packages/extra/regions/repo
  printf 'description: "x"\ndepends:\n  - kit\n' > .kit/packages/extra/package.yml
  printf 'create table t (id int);\n' > .kit/packages/extra/regions/repo/schema.sql
  run "${kit_checkout}/bin/kit" packages add extra
  [ "$status" -eq 0 ]
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [[ "$output" != *"grep:"* ]]
  [ "$(grep --count --line-regexp --fixed-strings -e '-- kit:generate schema.sql' schema.sql)" -eq 1 ]
}

@test "update refuses a region marked wrongly before it writes anything: every other file and the manifest stay" {
  local before
  run "${kit_checkout}/bin/kit" install github/ci agents/claude
  [ "$status" -eq 0 ]
  # .gitignore's markers are out of order, and two delivered files are missing, so an update would write them
  printf 'top\n# kit:end .gitignore\nmiddle\n# kit:generate .gitignore\nold\n' > .gitignore
  rm .github/workflows/kit-ci.yml .claude/settings.json
  before=$(tree-digest)
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *"'# kit:generate .gitignore' and '# kit:end .gitignore' must each be there once, in that order"* ]]
  [ "$(tree-digest)" = "${before}" ]
  [ ! -e .github/workflows/kit-ci.yml ]
  [ ! -e .claude/settings.json ]
  [[ "$output" == *"1 region(s) cannot be written, each named above"* ]]
  [[ "$output" != *"the region regenerated"* ]]
}

@test "a region makes the file where there is none, and one holding different content is regenerated" {
  run "${kit_checkout}/bin/kit" install agents/generic
  [ "$status" -eq 0 ]
  [ "$(sed -n '1,3p' AGENTS.md | paste -s -d '|' -)" = '# AGENTS.md||<!-- kit:generate AGENTS.md -->' ]
  run "${kit_checkout}/bin/kit" packages remove kit agents/generic
  [ ! -e AGENTS.md ]
  [ ! -e .gitignore ]
  printf '# Ours\n\n<!-- kit:generate AGENTS.md -->\nsomething else\n<!-- kit:end AGENTS.md -->\n' > AGENTS.md
  run "${kit_checkout}/bin/kit" packages add agents/generic
  [ "$status" -eq 0 ]
  [[ "$output" == *"AGENTS.md: placed, the region regenerated"* ]]
  [ "$(sed -n '1p' AGENTS.md)" = "# Ours" ]
  grep --quiet 'Read .kit/1st.md' AGENTS.md
  run ! grep --quiet 'something else' AGENTS.md
}

@test "a package's expected include is named at install, refused by check until it is there, and never written" {
  run "${kit_checkout}/bin/kit" install hooks/lefthook
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

@test "the include hint says what a new file needs, so the file it describes passes lint and check" {
  run "${kit_checkout}/bin/kit" install hooks/lefthook style/yaml
  [ "$status" -eq 0 ]
  [[ "$output" == *"A new lefthook.yml starts with a # line that says what it is"* ]]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  # the summary names no delivered file and no region when neither is the finding
  [[ "$output" == *"1 finding(s), each named above"* ]]
  [[ "$output" != *"a generated region by kit generate"* ]]
  printf '# lefthook.yml - the git hooks\nextends: [.kit/lefthook.yml]\n' > lefthook.yml
  run "${kit_checkout}/bin/kit" lint lefthook.yml
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "install keeps the repository's own settings keys and deny entries" {
  mkdir .claude
  printf '{"model":"x","permissions":{"deny":["Bash(rm -rf /)"]}}\n' > .claude/settings.json
  "${kit_checkout}/bin/kit" install agents/claude
  [ "$(jq --raw-output .model .claude/settings.json)" = "x" ]
  jq --exit-status '.permissions.deny | index("Bash(rm -rf /)")' .claude/settings.json >/dev/null
  jq --exit-status '.permissions.deny | index("Bash(git switch main *)")' .claude/settings.json >/dev/null
}

@test "install refuses a second time and names update" {
  "${kit_checkout}/bin/kit" install
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: cd "*" && kit update"* ]]
}

@test "check passes after install and fails on a drifted, a missing, and a lost deny entry" {
  "${kit_checkout}/bin/kit" install agents/claude hooks/lefthook
  printf 'extends:\n  - .kit/lefthook.yml\n' > lefthook.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  chmod u+w .kit/rules/markdown.md && printf 'edited\n' >> .kit/rules/markdown.md
  rm .kit/lefthook.yml
  jq '.permissions.deny -= ["Bash(git reset --hard *)"]' .claude/settings.json > s && mv s .claude/settings.json
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/markdown.md was changed in place"* ]]
  [[ "$output" == *".kit/lefthook.yml is missing"* ]]
  [[ "$output" == *"missing an entry the kit writes"* ]]
  [[ "$output" == *"3 finding(s)"* ]]
}

@test "check refuses in a repository that declares no kit version, and names install" {
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: cd "*" && kit install"* ]]
  [[ "$output" == *"kit.yml"* ]]
}

@test "install writes the declaration into kit.yml, keeps the repository's own lines, and the manifest records digests only" {
  mkdir .kit
  printf '# yaml-language-server: $schema=.kit/kit.schema.json\nrequire_signing: true\n' > kit.yml
  run "${kit_checkout}/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' kit.yml)" = '# yaml-language-server: $schema=.kit/kit.schema.json' ]
  grep --quiet '^require_signing: true$' kit.yml
  [ "$(declared version)" = unreleased ]
  [ "$(declared packages)" = "kit agents/generic agents/claude" ]
  run ! grep --quiet '^excludes' kit.yml
  run ! grep --quiet '^version' .kit/manifest
  run ! grep --quiet '^packages' .kit/manifest
  grep --quiet '  .kit/1st.md$' .kit/manifest
  # a later write replaces the declaration in place, once
  run "${kit_checkout}/bin/kit" packages add hooks/lefthook
  [ "$status" -eq 0 ]
  [ "$(declared packages)" = "kit agents/generic agents/claude hooks/lefthook" ]
  [ "$(grep -c '^version:' kit.yml)" -eq 1 ]
  [ "$(grep -c '^require_signing: true$' kit.yml)" -eq 1 ]
}

@test "update replaces an edited copy under .kit/, merges an edited file outside it, and records the release's digests" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci
  later-release "${repo}.later"
  printf '# later\n' >> "${repo}.later/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml"
  # an edit under .kit/ is replaced; an edit outside it, to a file the later release also changes elsewhere, is merged
  chmod u+w .kit/rules/json.md
  sed -i.bak 's/^# JSON$/# JSON (ours)/' .kit/rules/json.md && rm .kit/rules/json.md.bak
  sed -i.bak 's/^name: kit$/name: ours/' .github/workflows/kit-ci.yml && rm .github/workflows/kit-ci.yml.bak
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(declared version)" = "9.9.9" ]
  run ! grep --quiet '^# JSON (ours)$' .kit/rules/json.md
  grep --quiet '^## Added in 9.9.9$' .kit/rules/json.md
  [ ! -w .kit/rules/json.md ]
  grep --quiet '^name: ours$' .github/workflows/kit-ci.yml
  grep --quiet '^# later$' .github/workflows/kit-ci.yml
  release_digest=$(shasum -a 256 "${repo}.later/packages/kit/files/repo/.kit/rules/json.md" | cut -d ' ' -f 1)
  grep --quiet "^${release_digest}  .kit/rules/json.md$" .kit/manifest
  # the merged workflow still reads as an edit
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".github/workflows/kit-ci.yml differs"* ]]
  [[ "$output" == *"1 finding(s)"* ]]
}

@test "update without a manifest takes the base release's digest for a path, so an edited copy outside .kit/ is merged, not refused" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci
  later-release "${repo}.later"
  printf '# later\n' >> "${repo}.later/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml"
  sed -i.bak 's/^name: kit$/name: ours/' .github/workflows/kit-ci.yml && rm .github/workflows/kit-ci.yml.bak
  # a fresh clone, or bin/refresh's export: the manifest is not committed
  rm .kit/manifest
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  grep --quiet '^name: ours$' .github/workflows/kit-ci.yml
  grep --quiet '^# later$' .github/workflows/kit-ci.yml
  [ -f .kit/manifest ]
}

@test "a repository from a release whose manifest still declared the version and packages is taken as declared, and update moves the declaration into kit.yml" {
  "${kit_checkout}/bin/kit" install agents/claude
  # the shape rc.3 to rc.6 left behind: no declaration in kit.yml, the manifest's version and packages lines
  sed -i.bak -e '/^version:/d' -e '/^packages:/d' -e '/^  - /d' kit.yml && rm kit.yml.bak
  { printf '# .kit/manifest - written by kit, not by hand\nversion unreleased\npackages kit agents/generic agents/claude\n'
    grep '  ' .kit/manifest; } > m && mv m .kit/manifest
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: cd "*" && kit update"* ]]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run env KIT_BASE="${kit_checkout}" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(declared version)" = unreleased ]
  [ "$(declared packages)" = "kit agents/generic agents/claude" ]
  run ! grep --quiet '^version ' .kit/manifest
  git add --force .kit/manifest
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/manifest is tracked by git"* ]]
}

@test "a repository whose settings are still in .kit/settings.yml has them moved to kit.yml by the next verb" {
  "${kit_checkout}/bin/kit" install agents/claude
  # the shape a release before kit.yml left behind: the settings under .kit/, with the modeline beside the schema
  sed 's#\$schema=\.kit/kit\.schema\.json#$schema=settings.schema.json#' kit.yml > .kit/settings.yml && rm kit.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  [[ "$output" == *"kit.yml: the settings moved here from .kit/settings.yml"* ]]
  [ ! -e .kit/settings.yml ]
  [ "$(sed -n '1p' kit.yml)" = '# yaml-language-server: $schema=.kit/kit.schema.json' ]
  [ "$(declared packages)" = "kit agents/generic agents/claude" ]
}

@test "a kit.yml already there is kept, and .kit/settings.yml beside it is left alone" {
  "${kit_checkout}/bin/kit" install
  printf 'version: 0.0.1\n' > .kit/settings.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  [ -f .kit/settings.yml ]
  [ "$(declared version)" = unreleased ]
}

@test "the declaration is rewritten where it stands, so the repository's own keys and comments keep their places" {
  mkdir .kit
  printf 'require_signing: true\n\n# the kit\nversion: unreleased\npackages:\n  - kit\n\ngate: bin/ci\n' > kit.yml
  run "${kit_checkout}/bin/kit" packages add hooks/lefthook
  [ "$status" -eq 0 ]
  expected='require_signing: true||# the kit|version: unreleased|packages:|  - kit|  - hooks/lefthook||gate: bin/ci'
  [ "$(paste -s -d '|' - < kit.yml)" = "${expected}" ]
}

@test "the delivered workflow's version read strips quotes and a trailing comment" {
  "${kit_checkout}/bin/kit" install github/ci
  printf 'version: "9.9.9"  # pinned\n' > s.yml
  line=$(grep 'version=\$(sed' .github/workflows/kit-ci.yml | sed 's/^ *//; s#kit\.yml#s.yml#')
  eval "${line}"
  [ "${version}" = "9.9.9" ]
}

@test "update refuses a path new in the release where the repository already has a different file" {
  "${kit_checkout}/bin/kit" install agents/claude
  later-release "${repo}.later"
  printf 'new rule\n' > "${repo}.later/packages/kit/files/repo/.kit/rules/toml.md"
  printf 'mine\n' > .kit/rules/toml.md
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/toml.md: a different file is there already"* ]]
  [ "$(cat .kit/rules/toml.md)" = "mine" ]
}

@test "update of an untouched install leaves check green" {
  "${kit_checkout}/bin/kit" install agents/claude
  later-release "${repo}.later"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "update leaves conflict markers and fails when the same lines changed, in a file outside .kit/" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci
  later-release "${repo}.later"
  theirs="${repo}.later/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml"
  sed -i.bak 's/^name: kit$/name: theirs/' "${theirs}" && rm "${theirs}.bak"
  sed -i.bak 's/^name: kit$/name: ours/' .github/workflows/kit-ci.yml && rm .github/workflows/kit-ci.yml.bak
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 1 ]
  grep --quiet '^<<<<<<<' .github/workflows/kit-ci.yml
  [[ "$output" == *"conflict markers"* ]]
}

@test "a glob names every package it matches, and a glob that matches nothing is refused" {
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" packages add 'style/*'
  [ "$status" -eq 0 ]
  styles='style/bash style/gitignore style/json style/just style/markdown style/prose style/workflows style/yaml'
  [ "$(declared packages)" = "kit ${styles}" ]
  [ -f .kit/styles/bash.md ]
  run "${kit_checkout}/bin/kit" packages remove 'style/j*'
  [ "$status" -eq 0 ]
  [ ! -e .kit/styles/json.md ]
  [ ! -e .kit/styles/just.md ]
  [ -f .kit/styles/yaml.md ]
  run "${kit_checkout}/bin/kit" packages add 'nothing/*'
  [ "$status" -eq 2 ]
  [[ "$output" == *"<package> 'nothing/*' is not one of the choices; run: cd "*" && kit help packages"* ]]
}

@test "a bundle brings its members in, a member leaves on its own, and the exclusion holds through update" {
  run "${kit_checkout}/bin/kit" install empowerite
  [ "$status" -eq 0 ]
  [ -f .kit/styles/just.md ]
  [ -f .github/workflows/kit-ci.yml ]
  members='agents/generic agents/claude github/ci hooks/lefthook style/bash style/gitignore style/json style/just style/markdown'
  members+=' style/prose style/workflows style/yaml workflow'
  [ "$(declared packages)" = "kit ${members} empowerite" ]
  run "${kit_checkout}/bin/kit" packages remove style/just
  [ "$status" -eq 0 ]
  [ ! -e .kit/styles/just.md ]
  [ "$(declared excludes)" = "style/just" ]
  [[ " $(declared packages) " != *style/just* ]]
  run "${kit_checkout}/bin/kit" packages
  [[ "$output" == *"❌ style/just "* ]]
  run env KIT_BASE="${kit_checkout}" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [ ! -e .kit/styles/just.md ]
  [ "$(declared excludes)" = "style/just" ]
}

# The bundle's tests stand apart, each with three of its renders at most, so none holds more than a loaded gate gives
# one test.
@test "a member removed from a bundle comes back when it is added, and is no longer an exclusion" {
  run "${kit_checkout}/bin/kit" install empowerite
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages remove style/just
  [ "$status" -eq 0 ]
  [ "$(declared excludes)" = "style/just" ]
  run "${kit_checkout}/bin/kit" packages add style/just
  [ "$status" -eq 0 ]
  [ -f .kit/styles/just.md ]
  [ -z "$(declared excludes)" ]
}

@test "a package a bundle depends on is refused, the bundle leaves without its members, and a member then is no exclusion" {
  run "${kit_checkout}/bin/kit" install empowerite
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages remove kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit cannot be removed while"* ]]
  run "${kit_checkout}/bin/kit" packages remove empowerite
  [ "$status" -eq 0 ]
  [ -f .kit/styles/just.md ]
  # with no active package including it, a removed package is not an exclusion
  run "${kit_checkout}/bin/kit" packages remove style/just
  [ "$status" -eq 0 ]
  [ -z "$(declared excludes)" ]
}

@test "update resolves the active set from the new release: a new member arrives, and a new dependency brings an excluded package back, saying so" {
  "${kit_checkout}/bin/kit" install empowerite >/dev/null 2>&1
  "${kit_checkout}/bin/kit" packages remove style/json style/yaml >/dev/null 2>&1
  [ "$(declared excludes)" = "style/json style/yaml" ]
  later-release "${repo}.later"
  mkdir -p "${repo}.later/packages/extra/files/repo/.kit"
  printf 'description: "A package new in 9.9.9"\ndepends:\n  - kit\n' > "${repo}.later/packages/extra/package.yml"
  printf 'extra\n' > "${repo}.later/packages/extra/files/repo/.kit/extra"
  printf '  - extra\n' >> "${repo}.later/packages/empowerite/package.yml"
  printf '  - style/json\n' >> "${repo}.later/packages/style/markdown/package.yml"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(cat .kit/extra)" = "extra" ]
  [ -f .kit/styles/json.md ]
  [ ! -e .kit/styles/yaml.md ]
  [[ "$output" == *"style/json was excluded, but style/markdown depends on it"* ]]
  [ "$(declared excludes)" = "style/yaml" ]
  [[ " $(declared packages) " == *" extra "* ]]
  printf 'extends: [.kit/lefthook.yml]\n' > lefthook.yml
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "a package including a member the release does not have is refused, naming it" {
  later-release "${repo}.later"
  printf 'description: "x"\nincludes:\n  - nothing\n' > "${repo}.later/packages/hooks/lefthook/package.yml"
  run "${repo}.later/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *"packages/hooks/lefthook/package.yml includes 'nothing', which is not in reach"* ]]
  [ ! -e .kit/manifest ]
}

@test "update reads a base release that predates an active package: that release delivered nothing for it" {
  "${kit_checkout}/bin/kit" install
  later-release "${repo}.later"
  mkdir -p "${repo}.later/packages/extra/files/repo/.kit"
  printf 'description: "A package new in 9.9.9"\ndepends:\n  - kit\n' > "${repo}.later/packages/extra/package.yml"
  printf 'extra\n' > "${repo}.later/packages/extra/files/repo/.kit/extra"
  "${repo}.later/bin/kit" packages add extra >/dev/null 2>&1
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  [ "$(cat .kit/extra)" = "extra" ]
  grep --quiet '^## Added in 9.9.9$' .kit/rules/json.md
}

@test "update retires a path the new release no longer delivers, unless it was edited outside .kit/" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci
  later-release "${repo}.later"
  rm "${repo}.later/packages/kit/files/repo/.kit/rules/gitignore.md" \
    "${repo}.later/packages/github/ci/templates/repo/.github/workflows/kit-ci.yml"
  printf '# mine too\n' >> .github/workflows/kit-ci.yml
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  [ ! -e .kit/rules/gitignore.md ]
  [ -e .github/workflows/kit-ci.yml ]
  [[ "$output" == *".kit/rules/gitignore.md: retired in 9.9.9, removed"* ]]
  [[ "$output" == *".github/workflows/kit-ci.yml: retired in 9.9.9, but edited"* ]]
  run ! grep --quiet 'rules/gitignore.md' .kit/manifest
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "update retires nothing when the paths the release delivers cannot be read" {
  "${kit_checkout}/bin/kit" install
  local before kit_bash_modules=place
  before=$(tree-digest)
  # kit update runs place:retire under pipefail, as every verb does
  run kit-bash 'set -o pipefail && place:delivered() { return 1; } && place:retire "" repo unreleased 9.9.9 kit'
  [ "$status" -eq 1 ]
  [[ "$output" != *"retired in"* ]]
  [ "$(tree-digest)" = "${before}" ]
}

@test "a removal counts no refusal when the paths the release delivers cannot be read: it fails" {
  "${kit_checkout}/bin/kit" install
  local kit_bash_modules=place
  run kit-bash 'set -o pipefail && place:delivered() { return 1; } && printf ".kit/1st.md\n" | place:refusals repo kit'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "update without the base release refuses an edited file outside .kit/, names the base, and leaves the declared version" {
  later-release "${repo}.later"
  "${repo}.later/bin/kit" install agents/claude github/ci
  [ "$(declared version)" = "9.9.9" ]
  later-release "${repo}.newer" 9.9.10
  printf '# edited\n' >> .github/workflows/kit-ci.yml
  # neither the cache nor the release can produce 9.9.9
  run env PATH="$(shims mise curl)" "${repo}.newer/bin/kit" --version="${repo}.newer" update
  [ "$status" -eq 1 ]
  [[ "$output" == *"could not be fetched"* ]]
  [[ "$output" == *".github/workflows/kit-ci.yml cannot be updated without the base release 9.9.9"* ]]
  [ "$(declared version)" = "9.9.9" ]
  # whole or not at all: the file that needed no base was not written either
  run ! grep --quiet 'Added in 9.9.10' .kit/rules/json.md
}

@test "update without the base release replaces every file still as it was placed, where none is merged by key" {
  later-release "${repo}.later"
  "${repo}.later/bin/kit" install
  later-release "${repo}.newer" 9.9.10
  run env PATH="$(shims mise curl)" "${repo}.newer/bin/kit" --version="${repo}.newer" update
  [ "$status" -eq 0 ]
  grep --quiet 'Added in 9.9.10' .kit/rules/json.md
  [ "$(declared version)" = "9.9.10" ]
  run "${repo}.newer/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "update at the same version uses the release as its own base, and a second run changes nothing" {
  "${kit_checkout}/bin/kit" install agents/claude github/ci
  printf '# edited\n' >> .github/workflows/kit-ci.yml
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  grep --quiet '^# edited$' .github/workflows/kit-ci.yml
  run env PATH="$(shims mise curl)" "${kit_checkout}/bin/kit" update
  [ "$status" -eq 0 ]
  [[ "$output" != *"conflict"* ]]
}

@test "the verbs run from a subdirectory as from the root" {
  "${kit_checkout}/bin/kit" install
  mkdir -p docs/deep && cd docs/deep
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" packages add agents/claude
  [ "$status" -eq 0 ]
  [ -f "${repo}/.claude/settings.json" ]
  [ ! -e .claude ]
}

@test "install and package add refuse a different file at a delivered path, and accept the same one" {
  mkdir .kit && printf 'mine\n' > .kit/1st.md
  run "${kit_checkout}/bin/kit" install
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/1st.md: a different file is there already"* ]]
  [ ! -e .kit/manifest ]
  [ "$(cat .kit/1st.md)" = "mine" ]
  rm .kit/1st.md
  "${kit_checkout}/bin/kit" install
  printf 'theirs\n' > .kit/lefthook.yml
  run "${kit_checkout}/bin/kit" packages add hooks/lefthook
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/lefthook.yml: a different file is there already"* ]]
  [ "$(declared packages)" = "kit" ]
  rm .kit/lefthook.yml
  mkdir .claude && printf '# Ours\n' > .claude/CLAUDE.md
  run "${kit_checkout}/bin/kit" packages add agents/claude hooks/lefthook
  [ "$status" -eq 0 ]
  [ "$(sed -n '1p' .claude/CLAUDE.md)" = "# Ours" ]
  grep --quiet --line-regexp '@../AGENTS.md' .claude/CLAUDE.md
}

@test "package remove judges dependents against what stays, in any order" {
  "${kit_checkout}/bin/kit" install agents/claude
  run "${kit_checkout}/bin/kit" packages remove kit hooks/lefthook agents/claude agents/generic
  [ "$status" -eq 0 ]
  [ "$(declared packages)" = "" ]
  [ ! -e .kit/1st.md ]
}

@test "a home manifest another repository wrote is left alone, and its surviving lines are kept" {
  mkdir -p "${KIT_HOME}/.kit" && printf 'theirs\n' > "${KIT_HOME}/.kit/theirs"
  digest=$(shasum -a 256 "${KIT_HOME}/.kit/theirs" | cut -d ' ' -f 1)
  printf '# manifest\nversion 0.0.1\npackages other\n%s  .kit/theirs\n' "${digest}" > "${KIT_HOME}/.kit/manifest"
  "${kit_checkout}/bin/kit" install
  [ -f "${KIT_HOME}/.kit/theirs" ]
  grep --quiet '  .kit/theirs$' "${KIT_HOME}/.kit/manifest"
  grep --quiet '^version 0.0.1$' "${KIT_HOME}/.kit/manifest"
}

@test "a region is a template: @key@ takes the repository's answer, the package's default, or a settings key, and refuses a key with none" {
  mkdir -p .kit/packages/extra/regions/repo
  printf 'description: "x"\ndepends:\n  - kit\nanswers:\n  ci_runner: ours\n' > .kit/packages/extra/package.yml
  printf 'runner = "@ci_runner@"\nsigning = @require_signing@\nagain = "@answers.ci_runner@"\n' \
    > .kit/packages/extra/regions/repo/config.toml
  printf 'require_signing: false\n' > kit.yml
  run "${kit_checkout}/bin/kit" install extra
  [ "$status" -eq 0 ]
  rendered='runner = "ours"|signing = false|again = "ours"'
  [ "$(sed -n '2,4p' config.toml | paste -s -d '|' -)" = "${rendered}" ]
  [ "$("${kit_checkout}/bin/kit" region config.toml | paste -s -d '|' -)" = "${rendered}" ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'answers:\n  ci_runner: theirs\n' >> kit.yml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"config.toml: stale"* ]]
  run "${kit_checkout}/bin/kit" generate
  [ "$status" -eq 0 ]
  grep --quiet '^runner = "theirs"$' config.toml
  grep --quiet '^again = "theirs"$' config.toml
  printf 'x = @missing@\n' > .kit/packages/extra/regions/repo/other.toml
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  remedy="other.toml needs an answer for 'missing'; add 'missing: <value>' under answers: in kit.yml"
  [[ "$output" == *"${remedy} (package extra)"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
