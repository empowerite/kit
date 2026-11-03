#!/usr/bin/env bats
# toolchains.bats - kit finds the toolchains where it runs, chooses one for each role, and kit toolchains lists them
# ----------------------------------------------------------------------------------------------------------------------
# tests/fakes/fake-tool stands in for each provider's tool, so detection and choosing run without the real ones.
# What each provider does with its tool's answers, installing, listing, and updating, is in tests/tools.bats.

load helper

setup() {
  hide-system-toolchains
  fake-tools mise nix brew docker devcontainer
  # a probe runs mise and docker once, to know each is installed; brew is never run, since it calls git
  route mise '--version' '!0'
  route docker '--version' '!0'
  # brew reads a Brewfile on macOS alone, so every run is on macOS unless a test says otherwise
  export OSTYPE=darwin24
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

@test "brew is a system toolchain anywhere it is installed, and a project one with a Brewfile on macOS" {
  run detect
  [ "$status" -eq 0 ]
  [ "$(grep '^brew' <<<"$output")" = $'brew\tsystem\tyes\tno\nbrew\tproject\tno\tno' ]
  touch Brewfile
  run detect
  [ "$(grep '^brew' <<<"$output")" = $'brew\tsystem\tyes\tno\nbrew\tproject\tyes\tno' ]
  # elsewhere, as on a Linux runner, the Brewfile is not read
  OSTYPE=linux-gnu run detect
  [ "$(grep '^brew' <<<"$output")" = $'brew\tsystem\tyes\tno\nbrew\tproject\tno\tno' ]
}

@test "brew fills in for the chosen toolchain, for the Brewfile's tools it does not pin" {
  touch mise.toml Brewfile toolchain.Dockerfile
  export MISE_SHELL=zsh
  run kit-bash 'use toolchain && toolchain:filler mise'
  [ "$output" = brew ]
  # docker runs commands in its container, which brew's tools never reach; brew alone fills in for none
  run kit-bash 'use toolchain && toolchain:filler docker && toolchain:filler brew && toolchain:filler ambient'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  OSTYPE=linux-gnu run kit-bash 'use toolchain && toolchain:filler mise'
  [ -z "$output" ]
  # a tool's name is compared without its backend, path, or version
  fixture ls.json <<<'{"jq": [{"requested_version": "1.8", "version": "1.8.2", "installed": true}],
    "npm:markdownlint-cli2": [{"version": "0.24.0", "installed": true}],
    "aqua:mikefarah/yq": [{"version": "4.54.1", "installed": true}]}'
  route mise 'ls --json --current' ls.json
  run kit-bash 'use toolchain && toolchain:pinned-names mise'
  [ "$output" = $'jq\nmarkdownlint-cli2\nyq' ]
}

@test "brew is usable only when it is on PATH, not when it is only at a Homebrew prefix" {
  touch Brewfile
  # a brew at a prefix, off PATH
  rm "${FAKE_DIR}/bin/brew"
  path-without brew
  mkdir -p "${BATS_TEST_TMPDIR}/prefix/bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${BATS_TEST_TMPDIR}/prefix/bin/brew"
  export KIT_BREW_PREFIXES="${BATS_TEST_TMPDIR}/prefix/bin/brew"
  run detect
  [ "$(grep '^brew' <<<"$output")" = $'brew\tsystem\tno\tno\nbrew\tproject\tno\tno' ]
  # on PATH, it is usable for both roles
  export PATH="${BATS_TEST_TMPDIR}/prefix/bin:${PATH}"
  run detect
  [ "$(grep '^brew' <<<"$output")" = $'brew\tsystem\tyes\tno\nbrew\tproject\tyes\tno' ]
}

@test "apt, dnf, and pacman are system toolchains where each is on PATH, and never project ones" {
  run detect
  [ "$(grep -e '^apt' -e '^dnf' -e '^pacman' <<<"$output")" = \
    $'apt\tsystem\tno\tno\ndnf\tsystem\tno\tno\npacman\tsystem\tno\tno' ]
  fake-tools apt-get dnf pacman
  run detect
  [ "$(grep -e '^apt' -e '^dnf' -e '^pacman' <<<"$output")" = \
    $'apt\tsystem\tyes\tno\ndnf\tsystem\tyes\tno\npacman\tsystem\tyes\tno' ]
  # none is ever run to find it
  [ ! -e "${FAKE_DIR}/apt-get.calls" ] && [ ! -e "${FAKE_DIR}/dnf.calls" ] && [ ! -e "${FAKE_DIR}/pacman.calls" ]
  # beside brew, brew is offered first
  run kit-bash 'use toolchain && toolchain:usable system'
  [ "$output" = $'brew\napt\ndnf\npacman' ]
  # a project call refuses, naming the role it fills
  run kit-bash 'use toolchain && toolchain:call apt install'
  [ "$status" -eq 1 ]
  [[ "$output" == *"apt fills the system role only"* ]]
}

@test "each system toolchain names the package that installs a tool, by its own name, or none" {
  local sudo='sudo '
  (( EUID != 0 )) || sudo=''
  run kit-bash 'use toolchain && toolchain:remedy apt shellcheck && toolchain:remedy dnf shellcheck &&
    toolchain:remedy pacman shellcheck && toolchain:remedy dnf yq && toolchain:remedy pacman yq &&
    toolchain:remedy pacman gh'
  [ "$status" -eq 0 ]
  [ "$output" = "${sudo}apt-get install shellcheck
${sudo}dnf install ShellCheck
${sudo}pacman -S shellcheck
${sudo}dnf install yq
${sudo}pacman -S go-yq
${sudo}pacman -S github-cli" ]
  # Debian's yq is not mikefarah's, and no archive has lefthook
  run kit-bash 'use toolchain && toolchain:remedy apt yq'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  run kit-bash 'use toolchain && toolchain:remedy dnf lefthook || toolchain:remedy pacman lefthook'
  [ "$status" -eq 1 ]
}

@test "mise is active only when activated and configured here" {
  export MISE_SHELL=zsh
  run detect
  grep --quiet --line-regexp $'mise\tproject\tno\tno' <<<"$output"
  touch mise.toml
  run detect
  grep --quiet --line-regexp $'mise\tproject\tyes\tyes' <<<"$output"
}

@test "mise is active where every directory it puts on PATH is there, as direnv puts them" {
  touch mise.toml
  mkdir -p "${BATS_TEST_TMPDIR}/installs/jq" "${BATS_TEST_TMPDIR}/installs/yq"
  printf '%s\n' "${BATS_TEST_TMPDIR}/installs/jq" "${BATS_TEST_TMPDIR}/installs/yq" | fixture bin-paths.txt
  route mise 'bin-paths' bin-paths.txt
  run detect
  grep --quiet --line-regexp $'mise\tproject\tyes\tno' <<<"$output"
  # one directory of two is not enough
  export PATH="${BATS_TEST_TMPDIR}/installs/jq:${PATH}"
  run detect
  grep --quiet --line-regexp $'mise\tproject\tyes\tno' <<<"$output"
  export PATH="${BATS_TEST_TMPDIR}/installs/yq:${PATH}"
  run detect
  grep --quiet --line-regexp $'mise\tproject\tyes\tyes' <<<"$output"
  run chosen
  [ "$output" = mise ]
  # with the tools on PATH but no mise file here, mise is neither usable nor active
  rm mise.toml
  run detect
  grep --quiet --line-regexp $'mise\tproject\tno\tno' <<<"$output"
}

# a flake that holds a dev shell, and nix with flakes on: builtins.getFlake exists
nix-flake() {
  printf '{ outputs = { self }: { devShells.x86_64-linux.default = {}; }; }\n' > flake.nix
  fixture flakes.txt <<<'true'
  route nix 'eval --expr builtins ? getFlake' flakes.txt
}

@test "nix is usable with flakes on and a dev shell in flake.nix" {
  run detect
  grep --quiet --line-regexp $'nix\tproject\tno\tno' <<<"$output"
  nix-flake
  run detect
  grep --quiet --line-regexp $'nix\tproject\tyes\tno' <<<"$output"
  [ "$(grep '^nix' <<<"$output" | wc -l | tr -d ' ')" -eq 1 ]
  printf '{ outputs = { self }: { devShell.x86_64-linux = {}; }; }\n' > flake.nix
  run detect
  grep --quiet --line-regexp $'nix\tproject\tyes\tno' <<<"$output"
}

@test "nix without flakes, or a flake without a dev shell, is not usable" {
  nix-flake
  fixture flakes.txt <<<'false'
  run detect
  grep --quiet --line-regexp $'nix\tproject\tno\tno' <<<"$output"
  nix-flake
  printf '{ outputs = { self }: { packages.x86_64-linux.default = {}; }; }\n' > flake.nix
  run detect
  grep --quiet --line-regexp $'nix\tproject\tno\tno' <<<"$output"
  # a name that only ends in devShell is not a dev shell, and a nix that cannot eval, as without nix-command, has no
  # flakes
  nix-flake
  printf '{ outputs = { self }: { mydevShell = {}; }; }\n' > flake.nix
  run detect
  grep --quiet --line-regexp $'nix\tproject\tno\tno' <<<"$output"
  nix-flake
  : > "${FAKE_DIR}/nix.routes"
  route nix 'eval --expr builtins ? getFlake' '!1'
  run detect
  grep --quiet --line-regexp $'nix\tproject\tno\tno' <<<"$output"
  # no flake.nix, and nix is never asked
  rm flake.nix
  : > "${FAKE_DIR}/nix.calls"
  run detect
  [ ! -s "${FAKE_DIR}/nix.calls" ]
}

@test "nix is active in nix develop only where this repository has a flake" {
  export IN_NIX_SHELL=impure
  run detect
  grep --quiet --line-regexp $'nix\tproject\tno\tno' <<<"$output"
  nix-flake
  run detect
  grep --quiet --line-regexp $'nix\tproject\tyes\tyes' <<<"$output"
  unset IN_NIX_SHELL
  run detect
  grep --quiet --line-regexp $'nix\tproject\tyes\tno' <<<"$output"
}

@test "nix is offered after mise and before docker, and brew is offered only alone" {
  touch mise.toml Brewfile toolchain.Dockerfile
  nix-flake
  run kit-bash 'use toolchain && toolchain:usable project'
  [ "$output" = $'mise\nnix\ndocker' ]
  rm mise.toml flake.nix toolchain.Dockerfile
  run kit-bash 'use toolchain && toolchain:usable project'
  [ "$output" = brew ]
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
  # mise configured here runs twice: once for the directories it puts on PATH, once for its version
  touch mise.toml
  run detect
  [ "$(probes)" -eq 7 ]
  grep --quiet --line-regexp $'mise\tproject\tyes\tno' <<<"$output"
  export MISE_SHELL=zsh
  run detect
  [ "$(probes)" -eq 8 ]
  grep --quiet --line-regexp $'mise\tproject\tyes\tyes' <<<"$output"
  run detect
  [ "$(probes)" -eq 8 ]
  # mise's lock, its environment, and that environment's file each change what it puts on PATH
  touch mise.lock
  run detect
  [ "$(probes)" -eq 9 ]
  export MISE_ENV=desk
  run detect
  [ "$(probes)" -eq 10 ]
  touch mise.desk.toml
  run detect
  [ "$(probes)" -eq 11 ]
  run detect
  [ "$(probes)" -eq 11 ]
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
  touch mise.toml toolchain.Dockerfile
  run chosen
  [ "$status" -eq 0 ]
  [ "$output" = mise ]
  run choose docker
  [ "$status" -eq 0 ]
  run chosen
  [ "$output" = mise ]
  KIT_TOOLCHAIN=mise run chosen
  [ "$status" -eq 0 ]
  [ "$output" = mise ]
  KIT_TOOLCHAIN=docker run chosen
  [ "$status" -eq 1 ]
  [[ "$output" == *"mise"*"active"* ]]
  [ "$(grep --count --line-regexp docker <<<"$output")" -eq 0 ]
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
  # setup() stands fakes in for mise and docker; with each one's file here, both are usable
  touch mise.toml toolchain.Dockerfile
  run chosen </dev/null
  [ "$status" -eq 1 ]
  grep --quiet 'kit toolchains choose mise' <<<"$output"
  grep --quiet 'kit toolchains choose docker' <<<"$output"
  [ ! -e .git/kit/toolchain ]
}

@test "a project choice made at the desk holds in a worktree while usable there; one made there stays there" {
  touch mise.toml
  nix-flake
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  git add mise.toml flake.nix toolchain.Dockerfile && git commit --quiet --message 'toolchains'
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
  [[ "$output" == *"kit toolchains choose docker"* ]]
  [ "$(grep --count --line-regexp mise <<<"$output")" -eq 0 ]
  run choose docker
  [ "$status" -eq 0 ]
  run chosen </dev/null
  [ "$output" = docker ]
  [ "$(cat "${repo}/.git/worktrees/feature/kit/toolchain")" = docker ]
  [ ! -e .kit/cache/toolchain ]
  [ "$(cat "${repo}/.git/kit/toolchain")" = mise ]
  cd "${repo}"
  run chosen </dev/null
  [ "$output" = mise ]
}

@test "in a worktree, its own choice that is no longer usable gives way to the clone's that is" {
  touch mise.toml
  nix-flake
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  git add mise.toml flake.nix toolchain.Dockerfile && git commit --quiet --message 'toolchains'
  run choose mise
  [ "$status" -eq 0 ]
  git worktree add --quiet "${BATS_TEST_TMPDIR}/feature" -b feature
  cd "${BATS_TEST_TMPDIR}/feature"
  rm mise.toml
  run choose docker
  [ "$status" -eq 0 ]
  [ "$(cat "${repo}/.git/worktrees/feature/kit/toolchain")" = docker ]
  [ ! -e .kit/cache/toolchain ]
  # the branch gains mise.toml and drops toolchain.Dockerfile; mise and nix are usable, docker is not
  touch mise.toml
  rm toolchain.Dockerfile
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
  # with apt usable too, the choice still holds; forgotten, the system role is refused with no terminal to ask
  fake-tools apt-get
  run chosen system
  [ "$output" = brew ]
  rm "${KIT_HOME}/.kit/settings.yml"
  run chosen system
  [ "$status" -eq 1 ]
  [[ "$output" == *"with brew: kit toolchains choose brew --system"* ]]
  [[ "$output" == *"with apt: kit toolchains choose apt --system"* ]]
  [[ "$output" != *KIT_TOOLCHAIN* ]]
  printf 'theme: dark\ntoolchain: brew\n' > "${KIT_HOME}/.kit/settings.yml"
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

# ----------------------------------------------------------------------------------------------------------------------

@test "kit toolchains lists every provider and role, and which is usable, active, and chosen" {
  touch mise.toml Brewfile
  export MISE_SHELL=zsh
  run "${kit_root}/bin/kit" toolchains
  [ "$status" -eq 0 ]
  grep --quiet 'provider *role *usable *active *chosen' <<<"$output"
  # mise is active, so it is the project's choice; brew fills in for what mise does not pin, and fills the system role
  # alone among the usable
  grep --quiet -E '^mise +project +✅ +⚡ +📌' <<<"$output"
  grep --quiet -E '^brew +project +✅ +➕' <<<"$output"
  grep --quiet -E '^brew +system +✅ +📌' <<<"$output"
  grep --quiet -E '^docker +project *$' <<<"$output"
  [ "$(grep --count 'project' <<<"$output")" -ge 3 ]
}

@test "kit toolchains marks none chosen with several usable, exits 0, and never asks" {
  touch mise.toml toolchain.Dockerfile
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
    = '[["provider","role","usable","active","chosen","fills"]]' ]
  [ "$(jq --slurp 'map(select(.provider == "mise" and .role == "project")) | .[0] | [.usable, .active, .chosen, .fills]
    | map(type) | unique' --compact-output <<<"$output")" = '["boolean"]' ]
  [ "$(jq --slurp 'map(select(.provider == "mise" and .role == "project")) | .[0].chosen' <<<"$output")" = true ]
  [ "$(jq --slurp 'length' <<<"$output")" -eq "$(detect | wc -l | tr -d ' ')" ]
}

@test "kit toolchains choose records the project choice, and --system the system one" {
  touch mise.toml toolchain.Dockerfile
  run "${kit_root}/bin/kit" toolchains choose docker
  [ "$status" -eq 0 ]
  [ "$(cat .git/kit/toolchain)" = docker ]
  run "${kit_root}/bin/kit" toolchains --json
  [ "$(jq --slurp 'map(select(.chosen and .role == "project") | .provider) | .[]' --raw-output <<<"$output")" = docker ]
  run "${kit_root}/bin/kit" toolchains choose brew --system
  [ "$status" -eq 0 ]
  [ "$(yq .toolchain "${KIT_HOME}/.kit/settings.yml")" = brew ]
}

@test "a choice made in a fresh worktree with no .kit/ leaves git nothing to see" {
  touch mise.toml toolchain.Dockerfile
  git add mise.toml toolchain.Dockerfile && git commit --quiet --message 'toolchains'
  git worktree add --quiet "${BATS_TEST_TMPDIR}/feature" -b feature
  cd "${BATS_TEST_TMPDIR}/feature"
  [ ! -e .kit ]
  run "${kit_root}/bin/kit" toolchains choose docker
  [ "$status" -eq 0 ]
  [ "$(cat "${repo}/.git/worktrees/feature/kit/toolchain")" = docker ]
  [ ! -e .kit ]
  [ -z "$(git status --porcelain)" ]
  cd "${repo}"
}

@test "a choice made in a kit repository's fresh worktree is kept in its git directory, and git sees nothing" {
  "${kit_checkout}/bin/kit" install
  touch mise.toml toolchain.Dockerfile
  git add --all && git commit --quiet --message init
  git worktree add --quiet "${BATS_TEST_TMPDIR}/feature" -b feature
  cd "${BATS_TEST_TMPDIR}/feature"
  run "${kit_root}/bin/kit" toolchains choose docker
  [ "$status" -eq 0 ]
  [ "$(cat "${repo}/.git/worktrees/feature/kit/toolchain")" = docker ]
  [ ! -e .kit/cache/toolchain ]
  [ -z "$(git status --porcelain)" ]
  cd "${repo}"
}

@test "kit toolchains choose refuses a toolchain not usable here, naming the usable ones" {
  touch mise.toml
  run "${kit_root}/bin/kit" toolchains choose brew
  [ "$status" -eq 1 ]
  [[ "$output" == *"brew is not usable for the project here; usable: mise"* ]]
  [ ! -e .git/kit/toolchain ]
  # with a Brewfile, brew fills in for mise, and is never chosen over it
  touch Brewfile
  run "${kit_root}/bin/kit" toolchains choose brew
  [ "$status" -eq 1 ]
  [[ "$output" == *"brew fills in for the tools mise does not pin; choose mise"* ]]
  [ ! -e .git/kit/toolchain ]
  run "${kit_root}/bin/kit" toolchains choose
  [ "$status" -eq 2 ]
}

# a devcontainer configuration, and the devcontainer CLI (the fake) on PATH
devcontainer-file() {
  mkdir -p .devcontainer
  printf '{ // a comment\n  "image": "debian",\n}\n' > .devcontainer/devcontainer.json
}

@test "a devcontainer is usable from outside with its CLI and file, and active inside with its file" {
  run detect
  grep --quiet --line-regexp $'devcontainer\tproject\tno\tno' <<<"$output"
  devcontainer-file
  run detect
  grep --quiet --line-regexp $'devcontainer\tproject\tyes\tno' <<<"$output"
  [[ "$output" != *$'devcontainer\tsystem'* ]]
  # the other spelling of the file counts
  rm -r .devcontainer
  printf '{}\n' > .devcontainer.json
  run detect
  grep --quiet --line-regexp $'devcontainer\tproject\tyes\tno' <<<"$output"
  # without its CLI it is not usable from outside
  rm "${FAKE_DIR}/bin/devcontainer"
  run detect
  grep --quiet --line-regexp $'devcontainer\tproject\tno\tno' <<<"$output"
  # inside, the file alone is enough, and it fills the system role too
  for marker in REMOTE_CONTAINERS CODESPACES DEVCONTAINER
  do
    export "${marker}=true"
    run detect
    grep --quiet --line-regexp $'devcontainer\tproject\tyes\tyes' <<<"$output"
    grep --quiet --line-regexp $'devcontainer\tsystem\tyes\tyes' <<<"$output"
    unset "${marker}"
  done
  # a marker left over from another project, where this one has no file, is not active
  rm .devcontainer.json
  REMOTE_CONTAINERS=true run detect
  grep --quiet --line-regexp $'devcontainer\tproject\tno\tno' <<<"$output"
  [[ "$output" != *$'devcontainer\tsystem'* ]]
}

@test "a devcontainer's answers follow its markers through the cache, and it is offered last" {
  devcontainer-file
  touch mise.toml Brewfile toolchain.Dockerfile
  nix-flake
  run kit-bash 'use toolchain && toolchain:usable project'
  [ "$output" = $'mise\nnix\ndocker\ndevcontainer' ]
  mkdir -p .kit
  printf '/cache/\n' > .kit/.gitignore
  run detect
  grep --quiet --line-regexp $'devcontainer\tproject\tyes\tno' <<<"$output"
  [ -s .kit/cache/toolchains.tsv ]
  CODESPACES=true run detect
  grep --quiet --line-regexp $'devcontainer\tsystem\tyes\tyes' <<<"$output"
  # inside, it is the one chosen for both roles, ahead of every other
  CODESPACES=true run kit-bash 'use toolchain && toolchain:chosen system && toolchain:chosen project'
  [ "$output" = $'devcontainer\ndevcontainer' ]
}
