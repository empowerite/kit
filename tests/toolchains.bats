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

# chosen [<role>] prints the toolchain toolchain:chosen gives for the role, project unless named.
chosen() {
  kit-bash "use toolchain && toolchain:chosen ${1:-project}"
}

# choose <provider> [<role>] records the provider for the role, project unless named.
choose() {
  kit-bash "use toolchain && toolchain:choose ${2:-project} $1"
}

@test "one usable toolchain is the one, none is ambient" {
  run chosen
  [ "$status" -eq 0 ]
  [ "$output" = ambient ]
  touch mise.toml
  run chosen
  [ "$status" -eq 0 ]
  [ "$output" = mise ]
  KIT_TOOLCHAIN=ambient run chosen
  [ "$status" -eq 0 ]
  [ "$output" = ambient ]
}

@test "an active toolchain is the one, and naming another for the run is refused" {
  export MISE_SHELL=zsh
  touch mise.toml Brewfile
  run chosen
  [ "$status" -eq 0 ]
  [ "$output" = mise ]
  run choose brew
  [ "$status" -eq 0 ]
  run chosen
  [ "$output" = mise ]
  KIT_TOOLCHAIN=mise run chosen
  [ "$status" -eq 0 ]
  [ "$output" = mise ]
  KIT_TOOLCHAIN=brew run chosen
  [ "$status" -eq 1 ]
  [[ "$output" == *"mise"*"active"* ]]
  [ "$(grep --count --line-regexp brew <<<"$output")" -eq 0 ]
}

@test "a toolchain named for the run that is not usable here is refused, never replaced" {
  touch mise.toml
  KIT_TOOLCHAIN=brew run chosen
  [ "$status" -eq 1 ]
  [[ "$output" == *"brew"* ]]
  [ "$(grep --count --line-regexp -e mise -e brew -e ambient <<<"$output")" -eq 0 ]
  KIT_TOOLCHAIN=nix run chosen
  [ "$status" -eq 1 ]
  [[ "$output" == *"nix"* ]]
  [ "$(grep --count --line-regexp -e mise -e ambient <<<"$output")" -eq 0 ]
}

@test "with several usable and no terminal, kit refuses and lists each with the command that picks it" {
  # setup() stands fakes in for mise and brew; with each one's file here, both are usable
  touch mise.toml Brewfile
  run chosen </dev/null
  [ "$status" -eq 1 ]
  grep --quiet 'kit toolchains choose mise' <<<"$output"
  grep --quiet 'kit toolchains choose brew' <<<"$output"
  [ ! -e .git/kit/toolchain ]
}

@test "a project choice made at the desk holds in a worktree while usable there; one made there stays there" {
  touch mise.toml Brewfile
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  git add mise.toml Brewfile toolchain.Dockerfile && git commit --quiet --message 'toolchains'
  run choose mise
  [ "$status" -eq 0 ]
  [ "$(cat .git/kit/toolchain)" = mise ]
  git worktree add --quiet "${BATS_TEST_TMPDIR}/feature" -b feature
  cd "${BATS_TEST_TMPDIR}/feature"
  run chosen </dev/null
  [ "$status" -eq 0 ]
  [ "$output" = mise ]
  rm mise.toml
  run chosen </dev/null
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit toolchains choose brew"* ]]
  [ "$(grep --count --line-regexp mise <<<"$output")" -eq 0 ]
  run choose brew
  [ "$status" -eq 0 ]
  run chosen </dev/null
  [ "$output" = brew ]
  [ "$(cat .kit/cache/toolchain)" = brew ]
  [ "$(cat "${repo}/.git/kit/toolchain")" = mise ]
  cd "${repo}"
  run chosen </dev/null
  [ "$output" = mise ]
}

@test "in a worktree, its own choice that is no longer usable gives way to the clone's that is" {
  touch mise.toml Brewfile
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  git add mise.toml Brewfile toolchain.Dockerfile && git commit --quiet --message 'toolchains'
  run choose mise
  [ "$status" -eq 0 ]
  git worktree add --quiet "${BATS_TEST_TMPDIR}/feature" -b feature
  cd "${BATS_TEST_TMPDIR}/feature"
  rm mise.toml
  run choose brew
  [ "$status" -eq 0 ]
  [ "$(cat .kit/cache/toolchain)" = brew ]
  # the branch gains mise.toml and drops the Brewfile; mise and docker are usable, brew is not
  touch mise.toml
  rm Brewfile
  run chosen </dev/null
  [ "$status" -eq 0 ]
  [ "$output" = mise ]
  cd "${repo}"
}

@test "a recorded toolchain that is no longer usable is not used" {
  touch mise.toml Brewfile
  run choose mise
  [ "$status" -eq 0 ]
  run chosen </dev/null
  [ "$output" = mise ]
  # mise is uninstalled: a stand-in that fails is first on PATH, ahead of the fake and any mise on the desk
  PATH=$(shims mise)
  export PATH
  run chosen </dev/null
  [ "$status" -eq 0 ]
  [ "$output" = brew ]
  run choose mise
  [ "$status" -eq 1 ]
  [[ "$output" == *"mise"*"brew"* ]]
  [ "$(cat .git/kit/toolchain)" = mise ]
  # a detection that fails is that failure, not a toolchain that is not usable
  run kit-bash 'use toolchain && toolchain:detect() { return 1; } && toolchain:choose project brew'
  [ "$status" -eq 1 ]
  [[ "$output" != *"not usable"* ]]
  [ "$(cat .git/kit/toolchain)" = mise ]
}

@test "the system choice is kept in ~/.kit/settings.yml" {
  KIT_HOME=$(mktemp -d)
  export KIT_HOME
  mkdir "${KIT_HOME}/.kit"
  printf 'theme: dark\n' > "${KIT_HOME}/.kit/settings.yml"
  run choose brew system
  [ "$status" -eq 0 ]
  [ "$(yq .toolchain "${KIT_HOME}/.kit/settings.yml")" = brew ]
  [ "$(yq .theme "${KIT_HOME}/.kit/settings.yml")" = dark ]
  run kit-bash 'use toolchain && toolchain:recorded system'
  [ "$output" = brew ]
  run chosen system
  [ "$output" = brew ]
  # a toolchain named for the run is the project's, and leaves the system one as it is
  KIT_TOOLCHAIN=ambient run chosen system
  [ "$output" = brew ]
  # brew alone fills the system role today, so the refusal for several is asked for directly
  run kit-bash 'use toolchain && toolchain:refuse system brew nix'
  [[ "$output" == *"with brew: kit toolchains choose brew --system"* ]]
  [[ "$output" == *"with nix: kit toolchains choose nix --system"* ]]
  [[ "$output" != *KIT_TOOLCHAIN* ]]
  run choose mise system
  [ "$status" -eq 1 ]
  [ "$(yq .toolchain "${KIT_HOME}/.kit/settings.yml")" = brew ]
  [ ! -e .git/kit/toolchain ]
}

@test "a stale marker does not make a toolchain active" {
  export MISE_SHELL=zsh
  touch Brewfile
  run chosen
  [ "$status" -eq 0 ]
  [ "$output" = brew ]
  KIT_TOOLCHAIN=brew run chosen
  [ "$status" -eq 0 ]
  [ "$output" = brew ]
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
  run kit-bash 'use toolchain && toolchain:usable project'
  [ "$output" = $'mise\nbrew' ]
  run kit-bash 'use toolchain && toolchain:path'
  [ "$status" -eq 0 ]
  [ "$output" = $'/m/jq/bin\n/opt/brew/bin\n/opt/brew/sbin' ]
}

@test "mise installs, reports, updates, and puts its bin paths first" {
  skip "kit tools takes install, outdated, update, path, and run in the next task"
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
  skip "kit tools takes install, outdated, update, path, and run in the next task"
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
  skip "kit tools takes install, outdated, update, path, and run in the next task"
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

@test "kit toolchains lists every provider and role, and which is usable, active, and chosen" {
  touch mise.toml Brewfile
  export MISE_SHELL=zsh
  run "${kit_root}/bin/kit" toolchains
  [ "$status" -eq 0 ]
  grep --quiet 'provider *role *usable *active *chosen' <<<"$output"
  # mise is active, so it is the project's choice; brew fills the system role alone among the usable
  grep --quiet -E '^mise +project +✅ +⚡ +📌' <<<"$output"
  grep --quiet -E '^brew +project +✅ *$' <<<"$output"
  grep --quiet -E '^brew +system +✅ +📌' <<<"$output"
  grep --quiet -E '^docker +project *$' <<<"$output"
  [ "$(grep --count 'project' <<<"$output")" -ge 3 ]
}

@test "kit toolchains marks none chosen with several usable, exits 0, and never asks" {
  touch mise.toml Brewfile
  run "${kit_root}/bin/kit" toolchains </dev/null
  [ "$status" -eq 0 ]
  if grep --quiet '📌.*project\|project.*📌' <<<"$output"; then false; fi
  run "${kit_root}/bin/kit" toolchains --json
  [ "$status" -eq 0 ]
  [ "$(jq --slurp '[.[] | select(.role == "project" and .chosen)] | length' <<<"$output")" -eq 0 ]
  [ ! -e .git/kit/toolchain ]
}

@test "kit toolchains --json prints one object per provider and role" {
  touch mise.toml
  run "${kit_root}/bin/kit" toolchains --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --slurp 'map(keys_unsorted) | unique' <<<"$output")" \
    = '[["provider","role","usable","active","chosen"]]' ]
  [ "$(jq --slurp 'map(select(.provider == "mise" and .role == "project")) | .[0] | [.usable, .active, .chosen]
    | map(type) | unique' --compact-output <<<"$output")" = '["boolean"]' ]
  [ "$(jq --slurp 'map(select(.provider == "mise" and .role == "project")) | .[0].chosen' <<<"$output")" = true ]
  [ "$(jq --slurp 'length' <<<"$output")" -eq "$(detect | wc -l | tr -d ' ')" ]
}

@test "kit toolchains choose records the project choice, and --system the system one" {
  touch mise.toml Brewfile
  run "${kit_root}/bin/kit" toolchains choose brew
  [ "$status" -eq 0 ]
  [ "$(cat .git/kit/toolchain)" = brew ]
  run "${kit_root}/bin/kit" toolchains --json
  [ "$(jq --slurp 'map(select(.chosen and .role == "project") | .provider) | .[]' --raw-output <<<"$output")" = brew ]
  run "${kit_root}/bin/kit" toolchains choose brew --system
  [ "$status" -eq 0 ]
  [ "$(yq .toolchain "${KIT_HOME}/.kit/settings.yml")" = brew ]
}

@test "kit toolchains choose refuses a toolchain not usable here, naming the usable ones" {
  touch mise.toml
  run "${kit_root}/bin/kit" toolchains choose brew
  [ "$status" -eq 1 ]
  [[ "$output" == *"brew is not usable for the project here; usable: mise"* ]]
  [ ! -e .git/kit/toolchain ]
  run "${kit_root}/bin/kit" toolchains choose
  [ "$status" -eq 2 ]
}
