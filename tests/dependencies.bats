#!/usr/bin/env bats
# dependencies.bats - npm and uv pin a language's packages on top of the chosen toolchain, and kit tools acts on both
# ----------------------------------------------------------------------------------------------------------------------
# tests/fakes/fake-tool stands in for npm, uv, and mise. npm's pins are read from its files, so they are fixtures;
# uv's installed versions come from uv pip list, so that is a route.

load helper

setup() {
  hide-system-toolchains
  fake-tools mise nix brew docker devcontainer npm uv
  route mise '--version' '!0'
  route docker '--version' '!0'
  fresh-repo
}

# npm-here writes a package.json naming two packages, its lock, and node_modules' own lock, which has one installed.
npm-here() {
  printf '{"dependencies": {"left-pad": "^1.3.0"}, "devDependencies": {"markdownlint-cli2": "^0.18.0"}}\n' \
    > package.json
  printf '{"packages": {"": {}, "node_modules/left-pad": {"version": "1.3.0"},
    "node_modules/markdownlint-cli2": {"version": "0.18.1"}}}\n' > package-lock.json
  mkdir -p node_modules/.bin
  printf '{"packages": {"node_modules/left-pad": {"version": "1.3.0"}}}\n' > node_modules/.package-lock.json
}

# without-on-path <tool> takes <tool> off PATH, so a test decides where it is found: a directory that holds it is
# replaced by a copy of its links without it, as hide-system-toolchains does.
without-on-path() {
  local dir kept='' copy n=0 IFS=:
  for dir in ${PATH}
  do
    if [[ -x "${dir}/$1" ]]
    then
      copy="${BATS_TEST_TMPDIR}/without-$1-$(( n++ ))"
      mkdir -p "${copy}"
      ln -s "${dir}"/* "${copy}/" 2>/dev/null || true
      rm -f "${copy}/$1"
      dir=${copy}
    fi
    kept+="${kept:+:}${dir}"
  done
  export PATH=${kept}
}

# uv-here writes a pyproject.toml and a uv.lock: the project itself, then ruff and PyYAML; only PyYAML is installed.
uv-here() {
  printf '[project]\nname = "demo"\nversion = "0.1.0"\n' > pyproject.toml
  cat > uv.lock <<'TOML'
version = 1

[[package]]
name = "demo"
version = "0.1.0"
source = { virtual = "." }

[[package]]
name = "ruff"
version = "0.14.3"
source = { registry = "https://pypi.org/simple" }

[[package]]
name = "pyyaml"
version = "6.0.3"
source = { registry = "https://pypi.org/simple" }
TOML
  mkdir -p .venv/bin
  printf '#!/bin/sh\n' > .venv/bin/python && chmod +x .venv/bin/python
  fixture pip-list.json <<<'[{"name": "PyYAML", "version": "6.0.3"}]'
  route uv 'pip list --format json --python .venv/bin/python' pip-list.json
}

@test "npm and uv are usable where each is on PATH, and its manifest and lock are here" {
  run kit-bash 'use dependencies && dependencies:usable'
  [ -z "$output" ]
  touch package.json pyproject.toml
  run kit-bash 'use dependencies && dependencies:usable && echo --- && dependencies:unlocked'
  [ "$output" = $'---\nnpm\nuv' ]
  touch package-lock.json uv.lock
  run kit-bash 'use dependencies && dependencies:usable && echo --- && dependencies:unlocked'
  [ "$output" = $'npm\nuv\n---' ]
  # neither is run to find it
  [ ! -e "${FAKE_DIR}/npm.calls" ] && [ ! -e "${FAKE_DIR}/uv.calls" ]
  rm "${FAKE_DIR}/bin/npm"
  path-without npm
  run kit-bash 'use dependencies && dependencies:usable'
  [ "$output" = uv ]
}

@test "a dependency provider is looked for with the chosen toolchain's directories first on PATH" {
  touch package.json package-lock.json mise.toml
  rm "${FAKE_DIR}/bin/npm"
  path-without npm
  mkdir -p "${BATS_TEST_TMPDIR}/node/bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${BATS_TEST_TMPDIR}/node/bin/npm"
  printf '%s\n' "${BATS_TEST_TMPDIR}/node/bin" > "${FAKE_DIR}/bin-paths"
  route mise 'bin-paths' bin-paths
  run kit-bash 'use dependencies && dependencies:usable'
  [ "$output" = npm ]
  run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [ "$output" = "${BATS_TEST_TMPDIR}/node/bin
$(pwd -P)/node_modules/.bin" ]
}

@test "npm pins what package.json names at the lock's versions, and reads what is installed without running" {
  npm-here
  run kit-bash 'use dependencies && dependencies:call npm pins'
  [ "$status" -eq 0 ]
  [ "$output" = $'left-pad\t1.3.0\t1.3.0\nmarkdownlint-cli2\t0.18.1\t' ]
  [ ! -e "${FAKE_DIR}/npm.calls" ]
  run kit-bash 'use dependencies && dependencies:installed npm'
  [ "$status" -eq 1 ]
  rm -r node_modules
  run kit-bash 'use dependencies && dependencies:call npm pins'
  [ "$output" = $'left-pad\t1.3.0\t\nmarkdownlint-cli2\t0.18.1\t' ]
}

@test "uv pins every locked package but the project, each by its normal name" {
  uv-here
  run kit-bash 'use dependencies && dependencies:call uv pins'
  [ "$status" -eq 0 ]
  [ "$output" = $'ruff\t0.14.3\t\npyyaml\t6.0.3\t6.0.3' ]
  # with no .venv, nothing is installed, and uv is not asked
  rm -r .venv
  : > "${FAKE_DIR}/uv.calls"
  run kit-bash 'use dependencies && dependencies:call uv pins'
  [ "$output" = $'ruff\t0.14.3\t\npyyaml\t6.0.3\t' ]
  [ ! -s "${FAKE_DIR}/uv.calls" ]
}

@test "kit tools lists the toolchain's pins, then each provider's, each table headed by its name" {
  touch mise.toml
  fixture ls.json <<<'{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}]}'
  route mise 'ls --json --current' ls.json
  route mise 'bin-paths' '!0'
  npm-here
  export COLUMNS=200
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E '^mise · mise.toml$' <<<"$output"
  grep --quiet -E '^npm · package-lock.json$' <<<"$output"
  grep --quiet -E '^✅ jq +· 1\.8 +· 1\.8\.0$' <<<"$output"
  grep --quiet -E '^❌ markdownlint-cli2 +· 0\.18\.1$' <<<"$output"
  # mise's table comes first
  [ "$(grep -n -E '^(mise|npm) ·' <<<"$output" | cut -d: -f2 | cut -c1-3 | paste -s -d ' ' -)" = 'mis npm' ]
  run "${kit_root}/bin/kit" tools --json
  [ "$(jq --compact-output --slurp 'map([.name, .from])' <<<"$output")" \
    = '[["jq","mise"],["left-pad","npm"],["markdownlint-cli2","npm"]]' ]
}

@test "ambient, kit tools draws the rules' tools and then each provider's pins, and path is the providers' alone" {
  npm-here
  mkdir -p .kit/rules
  printf -- '---\nlint: markdownlint-cli2\n---\n# Markdown\n' > .kit/rules/markdown.md
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" node_modules/.bin/markdownlint-cli2
  export COLUMNS=500
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E "^ambient · the rules' tools, on PATH$" <<<"$output"
  # the rules' linter is found where npm installed it
  grep --quiet -E "^✅ markdownlint-cli2 +· $(pwd -P)/node_modules/.bin/markdownlint-cli2$" <<<"$output"
  run "${kit_root}/bin/kit" tools path
  [ "$output" = "$(pwd -P)/node_modules/.bin" ]
  run "${kit_root}/bin/kit" tools run bash -c 'command -v markdownlint-cli2'
  [ "$output" = "$(pwd -P)/node_modules/.bin/markdownlint-cli2" ]
}

@test "kit tools install installs the toolchain, then each provider from its lock" {
  touch mise.toml
  route mise 'install*' '!0'
  route mise 'bin-paths' '!0'
  npm-here
  uv-here
  route npm 'ci --no-audit --no-fund' '!0'
  route uv 'sync --frozen' '!0'
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"mise installed the tools mise.toml pins"*"npm installed the tools package-lock.json pins"*"uv installed the tools uv.lock pins"* ]]
  grep --quiet --line-regexp 'ci --no-audit --no-fund' "${FAKE_DIR}/npm.calls"
  grep --quiet --line-regexp 'sync --frozen' "${FAKE_DIR}/uv.calls"
  # a provider that fails stops the run, named
  route npm 'ci --no-audit --no-fund' '!1'
  sed -i.bak '1d' "${FAKE_DIR}/npm.routes" && rm "${FAKE_DIR}/npm.routes.bak"
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 1 ]
  [[ "$output" == *"npm: the tools could not be installed"* ]]
  [[ "$output" != *"uv installed"* ]]
}

@test "the first kit tools install finds npm where the toolchain put it, and refuses a lock whose tool is not on PATH" {
  local node="${BATS_TEST_TMPDIR}/node/bin"
  touch mise.toml
  npm-here
  # npm is on no PATH until mise installs it: then mise's bin-paths names the directory that holds it
  mkdir -p "${node}" "${FAKE_DIR}/real"
  rm "${FAKE_DIR}/bin/npm" "${FAKE_DIR}/bin/uv" "${FAKE_DIR}/bin/mise"
  without-on-path npm
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${node}/npm"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/real/mise"
  printf '#!/usr/bin/env bash\n[[ "$1" != install ]] || printf "%%s\\n" %q > %q\nexec %q "$@"\n' \
    "${node}" "${FAKE_DIR}/bin-paths.txt" "${FAKE_DIR}/real/mise" > "${FAKE_DIR}/bin/mise"
  chmod +x "${FAKE_DIR}/bin/mise"
  : > "${FAKE_DIR}/bin-paths.txt"
  route mise 'install*' '!0'
  route mise 'bin-paths' bin-paths.txt
  route npm 'ci --no-audit --no-fund' '!0'
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"npm installed the tools package-lock.json pins"* ]]
  grep --quiet --line-regexp 'ci --no-audit --no-fund' "${FAKE_DIR}/npm.calls"
  # a toolchain that leaves npm off PATH: the lock is here, so install fails, naming npm, and npm never runs
  rm "${FAKE_DIR}/npm.calls" "${node}/npm"
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 1 ]
  [[ "$output" == *"package-lock.json is here, but npm is not on PATH"* ]]
  [ ! -e "${FAKE_DIR}/npm.calls" ]
}

@test "--outdated asks each layer, and update moves each behind package to its latest" {
  npm-here
  uv-here
  fixture npm-outdated.json <<<'{"markdownlint-cli2": {"current": "0.18.1", "wanted": "0.18.1", "latest": "0.19.0"}}'
  route npm 'outdated --json' npm-outdated.json
  fixture uv-outdated.json <<<'[{"name": "PyYAML", "version": "6.0.3", "latest_version": "6.0.4"}]'
  route uv 'pip list --outdated --format json --python .venv/bin/python' uv-outdated.json
  export COLUMNS=200
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 0 ]
  # the installed version is what pins read: markdownlint-cli2 is locked, but not installed
  grep --quiet -E '^🔺 markdownlint-cli2 +· 0\.18\.1 +· +· 0\.19\.0$' <<<"$output"
  grep --quiet -E '^🔺 pyyaml +· 6\.0\.3 +· 6\.0\.3 +· 6\.0\.4$' <<<"$output"
  [[ "$output" == *"2 tool(s) are behind; run: kit tools update"* ]]
  run --separate-stderr "${kit_root}/bin/kit" tools --outdated --json
  [ "$(jq --compact-output --slurp 'map([.name, .latest, .from])' <<<"$output")" \
    = '[["markdownlint-cli2","0.19.0","npm"],["pyyaml","6.0.4","uv"]]' ]
  route npm 'install --no-audit --no-fund markdownlint-cli2@latest' '!0'
  route uv 'lock --upgrade' '!0'
  route uv 'sync --frozen' '!0'
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'install --no-audit --no-fund markdownlint-cli2@latest' "${FAKE_DIR}/npm.calls"
  grep --quiet --line-regexp 'lock --upgrade' "${FAKE_DIR}/uv.calls"
}

@test "npm outdated's answer counts, whatever its status, and an error in it fails the run, naming npm" {
  npm-here
  fixture npm-error.json <<<'{"error": {"code": "ENOTFOUND", "summary": "request failed"}}'
  route npm 'outdated --json' npm-error.json
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 1 ]
  [[ "$output" == *"npm outdated could not ask the registry"* ]]
  [[ "$output" == *"npm: could not tell whether every tool is behind"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
