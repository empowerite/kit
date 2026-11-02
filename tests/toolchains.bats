#!/usr/bin/env bats
# toolchain.bats - mise, brew, and docker each answer the toolchain seam, through a fake of their tool
# ----------------------------------------------------------------------------------------------------------------------
#
# Each provider is driven by tests/fakes/fake-tool standing in for its tool, so what is proven is how the provider
# reads its tool's answers and what it asks of it. What only the real tool can prove, that it answers as the
# fixtures do, is proven by running kit toolchains on a desk that has it.

load helper

setup() {
  fake-tools mise brew docker
  # a probe runs mise and docker once, to know each is installed; brew is never run, since it calls git
  route mise '--version' '!0'
  route docker '--version' '!0'
  fresh-repo
}

# detect prints what toolchain:detect finds here.
detect() {
  kit-bash 'use toolchain && toolchain:detect'
}

# probes prints how many times the fake mise has run.
probes() {
  if [[ -f "${FAKE_DIR}/mise.calls" ]]
  then wc -l < "${FAKE_DIR}/mise.calls" | tr -d ' '
  else echo 0
  fi
}

@test "a toolchain is usable when its tool is on PATH and its file is here" {
  run detect
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp $'mise\tproject\tno\tno' <<<"$output"
  touch mise.toml
  run detect
  grep --quiet --line-regexp $'mise\tproject\tyes\tno' <<<"$output"
  fresh-repo local
  touch mise.local.toml
  run detect
  grep --quiet --line-regexp $'mise\tproject\tyes\tno' <<<"$output"
}

@test "brew is a system toolchain anywhere it is installed, and a project one with a Brewfile" {
  run detect
  [ "$status" -eq 0 ]
  [ "$(grep '^brew' <<<"$output")" = $'brew\tsystem\tyes\tno\nbrew\tproject\tno\tno' ]
  touch Brewfile
  run detect
  [ "$(grep '^brew' <<<"$output")" = $'brew\tsystem\tyes\tno\nbrew\tproject\tyes\tno' ]
}

@test "mise is active only when activated and configured here" {
  export MISE_SHELL=zsh
  run detect
  grep --quiet --line-regexp $'mise\tproject\tno\tno' <<<"$output"
  touch mise.toml
  run detect
  grep --quiet --line-regexp $'mise\tproject\tyes\tyes' <<<"$output"
}

@test "the cache answers until PATH, a tool, a file, or a marker changes" {
  # with nowhere git ignores, nothing is cached, and each run probes
  mkdir .kit
  run detect
  run detect
  [ "$(probes)" -eq 2 ]
  [ ! -e .kit/cache ]
  printf '/cache/\n' > .kit/.gitignore
  run detect
  [ "$(probes)" -eq 3 ]
  found=$output
  run detect
  [ "$(probes)" -eq 3 ]
  [ "$output" = "${found}" ]
  [ "$(cat .kit/cache/toolchains.tsv)" = "${found}" ]
  [ -s .kit/cache/toolchains.key ]
  mkdir "${BATS_TEST_TMPDIR}/elsewhere"
  export PATH="${BATS_TEST_TMPDIR}/elsewhere:${PATH}"
  run detect
  [ "$(probes)" -eq 4 ]
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${BATS_TEST_TMPDIR}/elsewhere/mise"
  run detect
  [ "$(probes)" -eq 5 ]
  touch mise.toml
  run detect
  [ "$(probes)" -eq 6 ]
  grep --quiet --line-regexp $'mise\tproject\tyes\tno' <<<"$output"
  export MISE_SHELL=zsh
  run detect
  [ "$(probes)" -eq 7 ]
  grep --quiet --line-regexp $'mise\tproject\tyes\tyes' <<<"$output"
  run detect
  [ "$(probes)" -eq 7 ]
}

@test "a toolchains key in kit.yml is refused" {
  printf 'toolchains:\n  - provider: mise\n' > kit.yml
  run "${kit_root}/bin/kit" toolchains
  [ "$status" -eq 1 ]
  [[ "$output" == *"a key the kit does not know: toolchains"* ]]
}

@test "with mise.toml and a Brewfile, mise's pinned tools come first on PATH, before brew's" {
  touch mise.toml Brewfile
  fixture paths.txt <<<'/m/jq/bin'
  fixture prefix.txt <<<'/opt/brew'
  route mise 'bin-paths' paths.txt
  route brew '--prefix' prefix.txt
  run "${kit_root}/bin/kit" toolchains
  [ "$(awk '{ print $1 }' <<<"$output" | paste -s -d ' ' -)" = "mise brew" ]
  run "${kit_root}/bin/kit" toolchains path
  [ "$status" -eq 0 ]
  [ "$output" = $'/m/jq/bin\n/opt/brew/bin\n/opt/brew/sbin' ]
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
  run "${kit_root}/bin/kit" toolchains install
  [ "$status" -eq 0 ]
  [[ "$output" == *"mise: installing from mise.toml"* ]]
  run "${kit_root}/bin/kit" toolchains outdated
  [ "$status" -eq 0 ]
  [[ "$output" == *"mise     jq                               1.8.0            → 1.8.1"* ]]
  run "${kit_root}/bin/kit" toolchains outdated --check
  [ "$status" -eq 1 ]
  run "${kit_root}/bin/kit" toolchains update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade --quiet' "${FAKE_DIR}/mise.calls"
  run "${kit_root}/bin/kit" toolchains path
  [ "$output" = $'/m/jq/bin\n/m/just/bin' ]
  run "${kit_root}/bin/kit" toolchains run bash -c 'printf "%s" "${PATH%%:*}"'
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
  run "${kit_root}/bin/kit" toolchains install
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" toolchains outdated
  [[ "$output" == *"brew     jq                               1.7.1            → 1.8.1"* ]]
  [[ "$output" != *"wget"* ]]
  run "${kit_root}/bin/kit" toolchains update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade jq' "${FAKE_DIR}/brew.calls"
  run "${kit_root}/bin/kit" toolchains path
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
  run "${kit_root}/bin/kit" toolchains install
  [ "$status" -eq 0 ]
  tag=$(sed -n 's/^build --quiet --file toolchain.Dockerfile --tag \(kit-toolchain:[0-9a-f]*\) \.$/\1/p' \
    "${FAKE_DIR}/docker.calls")
  [ "${#tag}" -eq 26 ]
  run "${kit_root}/bin/kit" toolchains run just ci
  [ "$status" -eq 0 ]
  call="run --rm --volume $(pwd -P):/work --workdir /work --user $(id -u):$(id -g) --env HOME=/tmp ${tag} just ci"
  grep --quiet -- "${call}" "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" toolchains outdated
  [[ "$output" == *"alpine:3.20"*"sha256:old"*"→ sha256:new"* ]]
  [[ "$output" == *"debian:12"*"unpinned"*"→ sha256:deb"* ]]
  [[ "$output" != *"tools"*"→"* ]]
  run "${kit_root}/bin/kit" toolchains update
  [ "$status" -eq 0 ]
  pinned=$'FROM alpine:3.20@sha256:new AS tools\nRUN true\nFROM tools\nFROM debian:12@sha256:deb'
  [ "$(cat toolchain.Dockerfile)" = "${pinned}" ]
}

@test "the gate runs inside the docker toolchain, and with mise's path first otherwise" {
  mkdir .kit && printf 'gate: echo gated\n' > kit.yml
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  route docker 'image inspect *' '!0'
  route docker 'run --rm *' '!0'
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  grep --quiet -- ' bash -c echo gated$' "${FAKE_DIR}/docker.calls"
  printf 'gate: echo "${PATH%%%%:*}"\n' > kit.yml
  mv toolchain.Dockerfile elsewhere.Dockerfile
  touch mise.toml
  fixture paths.txt <<<'/m/tools/bin'
  route mise 'bin-paths' paths.txt
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"/m/tools/bin"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
