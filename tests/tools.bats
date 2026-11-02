#!/usr/bin/env bats
# tools.bats - kit tools lists, installs, updates, and runs with what the chosen project toolchain pins
# ----------------------------------------------------------------------------------------------------------------------
#
# Each provider is driven by tests/fakes/fake-tool standing in for its tool, so what is proven is how kit tools reads
# the tool's answers and what it asks of it. What only the real tool can prove, that it answers as the fixtures do, is
# proven by running kit tools on a desk that has it.

load helper

setup() {
  fake-tools mise brew docker
  # a probe runs mise and docker once, to know each is installed; brew is never run, since it calls git
  route mise '--version' '!0'
  route docker '--version' '!0'
  fresh-repo
}

# reached prints every call the fakes saw that could reach the network: outdated, imagetools, or upgrade.
reached() {
  cat "${FAKE_DIR}"/*.calls 2>/dev/null | grep -E 'outdated|imagetools|upgrade' || true
}

@test "kit tools lists what the chosen toolchain pins, with no network" {
  touch mise.toml
  fixture ls.json <<'JSON'
{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}],
 "just": [{"version": "1.40.0", "requested_version": "1.40", "installed": false}]}
JSON
  route mise 'ls --json --current' ls.json
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E 'name +· pinned +· installed$' <<<"$output"
  grep --quiet -E '^✅ jq +· 1\.8 +· 1\.8\.0$' <<<"$output"
  # a tool not installed has no glyph of an installed one, and nothing in its installed cell
  grep --quiet -E '^❌ just +· 1\.40$' <<<"$output"
  run "${kit_root}/bin/kit" tools --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --slurp . <<<"$output")" \
    = '[{"name":"jq","pinned":"1.8","installed":"1.8.0"},{"name":"just","pinned":"1.40","installed":null}]' ]
  [ -z "$(reached)" ]
}

@test "kit tools --outdated --check fails when one is behind, naming kit tools update" {
  touch mise.toml
  fixture ls.json <<<'{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}]}'
  fixture outdated.json <<<'{"jq": {"requested": "1.8", "current": "1.8.0", "latest": "1.8.1"}}'
  route mise 'ls --json --current' ls.json
  route mise 'outdated --json' outdated.json
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 0 ]
  grep --quiet -E 'name +· pinned +· installed +· latest$' <<<"$output"
  grep --quiet -E '^🔺 jq +· 1\.8 +· 1\.8\.0 +· 1\.8\.1$' <<<"$output"
  [[ "$output" == *"run: kit tools update"* ]]
  run "${kit_root}/bin/kit" tools --outdated --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"1 tool(s) are behind; run: kit tools update"* ]]
  run "${kit_root}/bin/kit" tools --outdated --json
  [ "$(jq --compact-output . <<<"$output")" = '{"name":"jq","pinned":"1.8","installed":"1.8.0","latest":"1.8.1"}' ]
  run "${kit_root}/bin/kit" tools --check
  [ "$status" -eq 2 ]
}

@test "ambient, kit tools lists the rules' tools and where each is found, and install refuses" {
  mkdir -p .kit/rules .kit/styles
  printf -- '---\nlint: shellcheck --shell=bash\n---\n# Bash\n' > .kit/rules/bash.md
  printf -- '---\nlint: kit-test-absent-linter --strict\n---\n# Absent\n' > .kit/rules/absent.md
  printf -- '---\nlint: shellcheck\n---\n# Twice\n' > .kit/styles/twice.md
  printf -- '---\nkind:\n  - text\n---\n# No linter\n' > .kit/rules/text.md
  fake-tools shellcheck
  # a row is clipped to the terminal's width, and the test's path is long
  export COLUMNS=500
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E 'name +· found$' <<<"$output"
  grep --quiet -E "^✅ shellcheck +· ${FAKE_DIR}/bin/shellcheck$" <<<"$output"
  grep --quiet -E '^❌ kit-test-absent-linter +· missing$' <<<"$output"
  [ "$(grep --count shellcheck <<<"$output")" -eq 1 ]
  run "${kit_root}/bin/kit" tools --json
  [ "$(jq --compact-output --slurp 'sort_by(.name) | map([.name, .path])' <<<"$output")" \
    = "[[\"kit-test-absent-linter\",null],[\"shellcheck\",\"${FAKE_DIR}/bin/shellcheck\"]]" ]
  [ "$(jq --compact-output --slurp 'map(keys_unsorted) | unique' <<<"$output")" = '[["name","path"]]' ]
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit-test-absent-linter"* ]]
  [[ "$output" != *"install by hand: shellcheck"* ]]
  # ambient, the path is empty and a command runs as is
  run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_root}/bin/kit" tools run printf ran
  [ "$status" -eq 0 ]
  [ "$output" = ran ]
}

@test "ambient, with every tool the rules need on PATH, install has nothing to do" {
  mkdir -p .kit/rules
  printf -- '---\nlint: shellcheck --shell=bash\n---\n# Bash\n' > .kit/rules/bash.md
  fake-tools shellcheck
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"every tool the rules need is on PATH"* ]]
}

@test "kit tools acts on the toolchain named for the run, and refuses one not usable here" {
  touch mise.toml Brewfile
  fixture paths.txt <<<'/m/jq/bin'
  fixture prefix.txt <<<'/opt/brew'
  route mise 'bin-paths' paths.txt
  route brew '--prefix' prefix.txt
  # with two usable and no terminal, kit refuses and names the command that picks each
  run "${kit_root}/bin/kit" tools path </dev/null
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit toolchains choose mise"* ]]
  run "${kit_root}/bin/kit" tools path --toolchain brew
  [ "$status" -eq 0 ]
  [ "$output" = $'/opt/brew/bin\n/opt/brew/sbin' ]
  KIT_TOOLCHAIN=mise run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [ "$output" = '/m/jq/bin' ]
  run "${kit_root}/bin/kit" tools path --toolchain docker
  [ "$status" -eq 1 ]
  [[ "$output" == *"docker"*"not usable"* ]]
}

@test "kit tools path prints only the chosen toolchain's directories, never two toolchains' combined" {
  touch mise.toml Brewfile
  fixture paths.txt <<<'/m/jq/bin'
  fixture prefix.txt <<<'/opt/brew'
  route mise 'bin-paths' paths.txt
  route brew '--prefix' prefix.txt
  run "${kit_root}/bin/kit" toolchains choose mise
  [ "$status" -eq 0 ]
  run kit-bash 'use toolchain && toolchain:path'
  [ "$status" -eq 0 ]
  [ "$output" = '/m/jq/bin' ]
  run "${kit_root}/bin/kit" toolchains choose brew
  run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [ "$output" = $'/opt/brew/bin\n/opt/brew/sbin' ]
}

@test "mise installs, reports, updates, and puts its bin paths first" {
  touch mise.toml
  fixture ls.json <<<'{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}]}'
  fixture outdated.json <<<'{"jq": {"requested": "1.8", "current": "1.8.0", "latest": "1.8.1"}}'
  fixture paths.txt <<<$'/m/jq/bin\n/m/just/bin'
  route mise 'trust*' '!0'
  route mise 'install --quiet' '!0'
  route mise 'ls --json --current' ls.json
  route mise 'outdated --json' outdated.json
  route mise 'upgrade --quiet' '!0'
  route mise 'bin-paths' paths.txt
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"mise: installing from mise.toml"* ]]
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 0 ]
  grep --quiet -E '^🔺 jq +· 1\.8 +· 1\.8\.0 +· 1\.8\.1$' <<<"$output"
  run "${kit_root}/bin/kit" tools --outdated --check
  [ "$status" -eq 1 ]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade --quiet' "${FAKE_DIR}/mise.calls"
  run "${kit_root}/bin/kit" tools path
  [ "$output" = $'/m/jq/bin\n/m/just/bin' ]
  run "${kit_root}/bin/kit" tools run bash -c 'printf "%s" "${PATH%%:*}"'
  [ "$output" = "/m/jq/bin" ]
}

@test "brew installs from the Brewfile without upgrading, and reports and moves only the Brewfile's tools" {
  touch Brewfile
  fixture names.txt <<<$'jq\nshellcheck\nnode@22'
  fixture versions.txt <<<$'jq 1.7.1\nwget 1.24\nnode@22 22.11.0'
  fixture outdated.json <<'JSON'
{"formulae": [{"name": "jq", "installed_versions": ["1.7.1"], "current_version": "1.8.1"},
              {"name": "wget", "installed_versions": ["1.24"], "current_version": "1.25"}],
 "casks": []}
JSON
  fixture prefix.txt <<<'/opt/brew'
  route brew 'bundle install --file Brewfile --no-upgrade --quiet' '!0'
  route brew 'bundle list --file Brewfile --formula --cask' names.txt
  route brew 'list --versions' versions.txt
  route brew 'outdated --json=v2' outdated.json
  route brew 'upgrade jq' '!0'
  route brew '--prefix' prefix.txt
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  # a Brewfile pins by name; a name with @ pins that version
  grep --quiet -E '^✅ jq +· +· 1\.7\.1$' <<<"$output"
  grep --quiet -E '^❌ shellcheck$' <<<"$output"
  grep --quiet -E '^✅ node@22 +· 22 +· 22\.11\.0$' <<<"$output"
  [[ "$output" != *"wget"* ]]
  run "${kit_root}/bin/kit" tools --outdated
  grep --quiet -E '^🔺 jq +· +· 1\.7\.1 +· 1\.8\.1$' <<<"$output"
  [[ "$output" != *"wget"* ]]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade jq' "${FAKE_DIR}/brew.calls"
  run "${kit_root}/bin/kit" tools path
  [ "$output" = $'/opt/brew/bin\n/opt/brew/sbin' ]
}

@test "docker builds its image once per file digest, runs a command in it, and pins its bases" {
  printf 'FROM alpine:3.20@sha256:old AS tools\nRUN true\nFROM tools\nFROM debian:12\n' > toolchain.Dockerfile
  fixture alpine.json <<<'{"digest": "sha256:new"}'
  fixture debian.json <<<'{"digest": "sha256:deb"}'
  route docker 'image inspect *' '!1'
  route docker 'build --quiet --file toolchain.Dockerfile --tag kit-toolchain:* .' '!0'
  route docker 'run --rm *' '!0'
  route docker 'buildx imagetools inspect alpine:3.20 *' alpine.json
  route docker 'buildx imagetools inspect debian:12 *' debian.json
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E '^❌ alpine:3\.20 +· sha256:old$' <<<"$output"
  grep --quiet -E '^❌ debian:12$' <<<"$output"
  [ -z "$(reached)" ]
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  tag=$(sed -n 's/^build --quiet --file toolchain.Dockerfile --tag \(kit-toolchain:[0-9a-f]*\) \.$/\1/p' \
    "${FAKE_DIR}/docker.calls")
  [ "${#tag}" -eq 26 ]
  run "${kit_root}/bin/kit" tools run just ci
  [ "$status" -eq 0 ]
  call="run --rm --volume $(pwd -P):/work --workdir /work --user $(id -u):$(id -g) --env HOME=/tmp ${tag} just ci"
  grep --quiet -- "${call}" "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" tools --outdated
  grep --quiet -E '^🔺 alpine:3\.20 +· sha256:old +· +· sha256:new$' <<<"$output"
  grep --quiet -E '^🔺 debian:12 +· +· +· sha256:deb$' <<<"$output"
  [[ "$output" != *"🔺 tools"* ]]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  pinned=$'FROM alpine:3.20@sha256:new AS tools\nRUN true\nFROM tools\nFROM debian:12@sha256:deb'
  [ "$(cat toolchain.Dockerfile)" = "${pinned}" ]
}

@test "kit gate runs through the chosen toolchain, inside docker when docker is chosen" {
  mkdir .kit && printf 'gate: echo "${PATH%%%%:*}"\n' > kit.yml
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  touch mise.toml
  fixture paths.txt <<<'/m/tools/bin'
  route mise 'bin-paths' paths.txt
  route docker 'image inspect *' '!0'
  route docker 'run --rm *' '!0'
  # with mise and docker both usable and none chosen, the gate cannot run, and that is not the gate failing
  run "${kit_root}/bin/kit" gate </dev/null
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit toolchains choose docker"* ]]
  [[ "$output" != *"the gate failed"* ]]
  run "${kit_root}/bin/kit" toolchains choose docker
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  grep --quiet --fixed-strings -- ' bash -c echo "${PATH%%:*}"' "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" toolchains choose mise
  [ "$status" -eq 0 ]
  : > "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"/m/tools/bin"* ]]
  if grep --quiet '^run ' "${FAKE_DIR}/docker.calls"; then false; fi
}

@test "kit toolchains no longer takes install, outdated, update, path, or run" {
  for verb in install outdated update path run
  do
    run "${kit_root}/bin/kit" toolchains "${verb}"
    [ "$status" -eq 2 ]
  done
}

# ----------------------------------------------------------------------------------------------------------------------
