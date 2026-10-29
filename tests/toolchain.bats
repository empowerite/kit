#!/usr/bin/env bats
# toolchain.bats - mise, brew, and docker each answer the toolchain seam, through a fake of their tool
# ----------------------------------------------------------------------------------------------------------------------
#
# Each provider is driven by tests/fakes/fake-tool standing in for its tool, so what is proven is how the provider
# reads its tool's answers and what it asks of it. What only the real tool can prove, that it answers as the
# fixtures do, is proven by running kit toolchain on a desk that has it.

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}" "${FAKE_DIR}/bin"
  for tool in mise brew docker
  do
    ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/${tool}"
    : > "${FAKE_DIR}/${tool}.routes"
  done
  export PATH="${FAKE_DIR}/bin:${PATH}"
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
}

teardown() {
  rm -rf "${repo}"
}

fixture() { cat > "${FAKE_DIR}/$1"; }
route() { printf '%s\t%s\n' "$2" "$3" >> "${FAKE_DIR}/$1.routes"; }

@test "the toolchains are detected from their files, in order, and a declared one wins" {
  run "${kit_root}/bin/kit" toolchain list
  [ -z "$output" ]
  run "${kit_root}/bin/kit" toolchain install
  [ "$status" -eq 0 ]
  [[ "$output" == *"no toolchain is declared or detected"* ]]
  touch mise.toml Brewfile toolchain.Dockerfile
  run "${kit_root}/bin/kit" toolchain list
  detected=$(awk '{ print $1 "=" $2 }' <<<"$output" | paste -s -d ' ' -)
  [ "${detected}" = "mise=mise.toml brew=Brewfile docker=toolchain.Dockerfile" ]
  mkdir .kit && printf 'toolchains:\n  - provider: brew\n    file: tools/Brewfile\n' > .kit/settings.yml
  run "${kit_root}/bin/kit" toolchain list
  [ "$output" = "brew     tools/Brewfile" ]
  printf 'toolchains:\n  - provider: nix\n' > .kit/settings.yml
  run "${kit_root}/bin/kit" toolchain list
  [ "$status" -ne 0 ]
  [[ "$output" == *"'nix' is not a toolchain the kit knows; the toolchains are: brew docker mise"* ]]
}

@test "mise installs, reports, updates, and puts its bin paths first" {
  touch mise.toml
  fixture outdated.json <<<'{"jq": {"requested": "1.8", "current": "1.8.0", "latest": "1.8.1"}}'
  fixture paths.txt <<<$'/m/jq/bin\n/m/just/bin'
  route mise 'trust*' '!0'
  route mise 'install --quiet' '!0'
  route mise 'outdated --json' outdated.json
  route mise 'upgrade --quiet' '!0'
  route mise 'bin-paths' paths.txt
  run "${kit_root}/bin/kit" toolchain install
  [ "$status" -eq 0 ]
  [[ "$output" == *"mise: installing from mise.toml"* ]]
  run "${kit_root}/bin/kit" toolchain outdated
  [ "$status" -eq 0 ]
  [[ "$output" == *"mise     jq                               1.8.0            → 1.8.1"* ]]
  run "${kit_root}/bin/kit" toolchain outdated --check
  [ "$status" -eq 1 ]
  run "${kit_root}/bin/kit" toolchain update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade --quiet' "${FAKE_DIR}/mise.calls"
  run "${kit_root}/bin/kit" toolchain path
  [ "$output" = $'/m/jq/bin\n/m/just/bin' ]
  run "${kit_root}/bin/kit" toolchain run bash -c 'printf "%s" "${PATH%%:*}"'
  [ "$output" = "/m/jq/bin" ]
}

@test "brew installs from the Brewfile without upgrading, and reports and moves only the Brewfile's tools" {
  touch Brewfile
  fixture names.txt <<<$'jq\nshellcheck'
  fixture outdated.json <<'JSON'
{"formulae": [{"name": "jq", "installed_versions": ["1.7.1"], "current_version": "1.8.1"},
              {"name": "wget", "installed_versions": ["1.24"], "current_version": "1.25"}],
 "casks": []}
JSON
  fixture prefix.txt <<<'/opt/brew'
  route brew 'bundle install --file Brewfile --no-upgrade --quiet' '!0'
  route brew 'bundle list --file Brewfile --formula --cask' names.txt
  route brew 'outdated --json=v2' outdated.json
  route brew 'upgrade jq' '!0'
  route brew '--prefix' prefix.txt
  run "${kit_root}/bin/kit" toolchain install
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" toolchain outdated
  [[ "$output" == *"brew     jq                               1.7.1            → 1.8.1"* ]]
  [[ "$output" != *"wget"* ]]
  run "${kit_root}/bin/kit" toolchain update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade jq' "${FAKE_DIR}/brew.calls"
  run "${kit_root}/bin/kit" toolchain path
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
  run "${kit_root}/bin/kit" toolchain install
  [ "$status" -eq 0 ]
  tag=$(sed -n 's/^build --quiet --file toolchain.Dockerfile --tag \(kit-toolchain:[0-9a-f]*\) \.$/\1/p' \
    "${FAKE_DIR}/docker.calls")
  [ "${#tag}" -eq 26 ]
  run "${kit_root}/bin/kit" toolchain run just ci
  [ "$status" -eq 0 ]
  call="run --rm --volume $(pwd -P):/work --workdir /work --user $(id -u):$(id -g) --env HOME=/tmp ${tag} just ci"
  grep --quiet -- "${call}" "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" toolchain outdated
  [[ "$output" == *"alpine:3.20"*"sha256:old"*"→ sha256:new"* ]]
  [[ "$output" == *"debian:12"*"unpinned"*"→ sha256:deb"* ]]
  [[ "$output" != *"tools"*"→"* ]]
  run "${kit_root}/bin/kit" toolchain update
  [ "$status" -eq 0 ]
  pinned=$'FROM alpine:3.20@sha256:new AS tools\nRUN true\nFROM tools\nFROM debian:12@sha256:deb'
  [ "$(cat toolchain.Dockerfile)" = "${pinned}" ]
}

@test "the gate runs inside the docker toolchain, and with mise's path first otherwise" {
  mkdir .kit && printf 'gate: echo gated\ntoolchains:\n  - provider: docker\n' > .kit/settings.yml
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  route docker 'image inspect *' '!0'
  route docker 'run --rm *' '!0'
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  grep --quiet -- ' bash -c echo gated$' "${FAKE_DIR}/docker.calls"
  printf 'gate: echo "${PATH%%%%:*}"\n' > .kit/settings.yml
  mv toolchain.Dockerfile elsewhere.Dockerfile
  touch mise.toml
  fixture paths.txt <<<'/m/tools/bin'
  route mise 'bin-paths' paths.txt
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"/m/tools/bin"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
