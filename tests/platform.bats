#!/usr/bin/env bats
# platform.bats - kit platform: the services, toolchains, tools, and settings kit runs with, each a stage with a mark
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them, so the
# workspace file starts missing. A repository's origin is a GitHub URL nothing fetches; the fake gh answers for it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
}

# repo-at <path> <member> makes a repository at <path> with one commit, its origin <member> on GitHub.
repo-at() {
  mkdir -p "$1"
  git init --quiet --initial-branch=main "$1"
  printf 'one\n' > "$1/a"
  git -C "$1" add a && git -C "$1" commit --quiet --message one
  git -C "$1" remote add origin "https://$2.git"
}

# signed-in puts the fake gh first on PATH, answering the caller's login as a signed-in gh does.
signed-in() {
  fake-tools gh
  fixture user.json <<<'{"login": "ada"}'
  route 'api user*' user.json
}

@test "kit platform --stages prints the four stages, in order" {
  run "${kit_root}/bin/kit" platform --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'services\ntoolchains\ntools\nsettings' ]
}

@test "toolchains is kit toolchains --json, each row keyed by its provider and role, with no links" {
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  printf '[tools]\n' > mise.toml
  run --separate-stderr "${kit_root}/bin/kit" platform toolchains --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.rows[] | select(.key == "mise project") | [.provider, .role, .links]' <<<"$output")" \
    = '["mise","project",{}]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "services names the forge a github origin infers, ready when gh is signed in, and each capability kit fills" {
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  signed-in
  run --separate-stderr "${kit_root}/bin/kit" platform services --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.rows[] | select(.key == "hosting") | {capability, service, chosen, ready}' <<<"$output")" \
    = '{"capability":"hosting","service":"github","chosen":"inferred","ready":true}' ]
  [ "$(jq -c '[.rows[].capability]' <<<"$output")" \
    = '["hosting","review","ci","releases","tracking #","toolchain","dependencies","policy"]' ]
  [ "$(jq -c '.rows[] | select(.key == "tracking #") | [.service, .provider, .ready]' <<<"$output")" \
    = '["github","tracker/forge",true]' ]
  [ "$(jq -c '.rows[] | select(.key == "policy") | [.service, .provider, .chosen, .ready]' <<<"$output")" \
    = '["patterns","policy/patterns","inferred",true]' ]
}

@test "a forge gh is not signed in to is not ready, and the stage still draws" {
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  fake-tools gh
  route 'api user*' '!1'
  printf 'forge: github\n' > kit.yml
  run --separate-stderr "${kit_root}/bin/kit" platform services --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.rows[] | select(.key == "hosting") | [.service, .chosen, .ready]' <<<"$output")" \
    = '["github","set",false]' ]
  [ -z "$stderr" ]
}

@test "in a repository with no remote, the forge's services say service null, and the rest still draw" {
  mkdir -p "${BATS_TEST_TMPDIR}/r" && git init --quiet "${BATS_TEST_TMPDIR}/r" && cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" platform services --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | select(.capability | IN("hosting", "review", "ci", "releases")) | .service] | unique' \
    <<<"$output")" = '[null]' ]
  [ "$(jq -r '.rows[] | select(.key == "policy") | .service' <<<"$output")" = patterns ]
}

@test "settings lists every key the settings schema names: the value kit.yml sets, else the schema's default" {
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  printf 'primary: upstream\nflow: [backlog, changes]\n' > kit.yml
  run --separate-stderr "${kit_root}/bin/kit" platform settings --json
  [ "$status" -eq 0 ]
  expected='[["primary","upstream","set"],["gate",null,"default"],["log_days",14,"default"],'
  expected+='["flow",["backlog","changes"],"set"]]'
  [ "$(jq -c '[.rows[] | select(.key | IN("primary", "flow", "log_days", "gate")) | [.setting, .value, .from]]' \
    <<<"$output")" = "${expected}" ]
  [ "$(jq '.rows | length' <<<"$output")" \
    -eq "$(jq '.properties | length' "${kit_root}/packages/kit/files/repo/.kit/kit.schema.json")" ]
}

@test "tools that kit tools refuses is its line and exit 1, and toolchains still draws" {
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  KIT_TOOLCHAIN=nonesuch run --separate-stderr "${kit_root}/bin/kit" platform tools --json
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"❌ the toolchain named for this run, nonesuch, is not usable"* ]]
  KIT_TOOLCHAIN=nonesuch run --separate-stderr "${kit_root}/bin/kit" platform toolchains --json
  [ "$status" -eq 0 ]
  KIT_TOOLCHAIN=nonesuch run --separate-stderr "${kit_root}/bin/kit" platform --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.stages, .tools, (.toolchains.rows | length > 0)]' <<<"$output")" \
    = '[["services","toolchains","tools","settings"],null,true]' ]
}

@test "--marks is an object of the four stages' marks; settings', services', and toolchains' move when kit.yml does" {
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" platform --marks
  [ "$status" -eq 0 ]
  [ "$(jq -c 'keys' <<<"$output")" = '["services","settings","toolchains","tools"]' ]
  before=$output
  printf 'log_days: 3\n' > kit.yml
  run --separate-stderr "${kit_root}/bin/kit" platform --marks
  [ "$status" -eq 0 ]
  [ "$(jq -r .settings <<<"$output")" != "$(jq -r .settings <<<"${before}")" ]
  [ "$(jq -r .services <<<"$output")" != "$(jq -r .services <<<"${before}")" ]
  # the kit toolchain reads tools: in kit.yml, so what is usable can move with it
  [ "$(jq -r .toolchains <<<"$output")" != "$(jq -r .toolchains <<<"${before}")" ]
  before=$output
  printf '[tools]\n' > mise.toml
  run --separate-stderr "${kit_root}/bin/kit" platform --marks
  [ "$(jq -r .toolchains <<<"$output")" != "$(jq -r .toolchains <<<"${before}")" ]
  [ "$(jq -r .tools <<<"$output")" = "$(jq -r .toolchains <<<"$output")" ]
}

@test "services' mark moves when a remote is added, and settings' does not" {
  mkdir -p "${BATS_TEST_TMPDIR}/r" && git init --quiet "${BATS_TEST_TMPDIR}/r" && cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" platform --marks
  [ "$status" -eq 0 ]
  before=$output
  git remote add origin https://github.com/o/r.git
  run --separate-stderr "${kit_root}/bin/kit" platform --marks
  [ "$status" -eq 0 ]
  [ "$(jq -r .services <<<"$output")" != "$(jq -r .services <<<"${before}")" ]
  [ "$(jq -r .settings <<<"$output")" = "$(jq -r .settings <<<"${before}")" ]
}

@test "a workspace whose members none is cloned draws no rows and no errors, and exits 0" {
  mkdir -p "${XDG_CONFIG_HOME}/kit" "${BATS_TEST_TMPDIR}/code"
  printf 'roots: [%s]\nworkspaces: {w: {repos: [github.com/o/a]}}\n' "${BATS_TEST_TMPDIR}/code" \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  cd "${BATS_TEST_TMPDIR}"
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w platform settings --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "--workspace= reads each cloned member's settings, each row tagged with its member, in the group's order" {
  repo-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  repo-at "${BATS_TEST_TMPDIR}/code/b" github.com/o/b
  printf 'log_days: 3\n' > "${BATS_TEST_TMPDIR}/code/b/kit.yml"
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  printf 'roots: [%s]\nworkspaces: {w: {repos: [github.com/o/a, github.com/o/b]}}\n' "${BATS_TEST_TMPDIR}/code" \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  cd "${BATS_TEST_TMPDIR}"
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w platform settings --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | select(.key == "log_days") | [.member, .value, .from]]' <<<"$output")" \
    = '[["github.com/o/a",14,"default"],["github.com/o/b",3,"set"]]' ]
  [ "$(jq '[.rows[] | select(.member == null)] | length' <<<"$output")" -eq 0 ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "as text, a stage is a line of its fields' names and a line per row, and the picture each stage under its name" {
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  printf 'primary: origin\n' > kit.yml
  run --separate-stderr "${kit_root}/bin/kit" platform settings --text
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" =~ ^setting\ +value\ +from$ ]]
  [[ "$output" =~ $'\n'primary\ +origin\ +set$'\n' ]]
  KIT_TOOLCHAIN=ambient run --separate-stderr "${kit_root}/bin/kit" platform
  [ "$status" -eq 0 ]
  [[ "$output" == *$'services:\ncapability '* && "$output" == *$'\n\nsettings:\nsetting '* ]]
}

@test "outside a repository, and with no workspace, kit platform refuses with 2; --stages still answers" {
  mkdir -p "${BATS_TEST_TMPDIR}/nowhere" && cd "${BATS_TEST_TMPDIR}/nowhere"
  run "${kit_root}/bin/kit" platform settings --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit platform reads a repository: run it in one, or name a workspace: kit --workspace="* ]]
  run "${kit_root}/bin/kit" platform --stages
  [ "$status" -eq 0 ]
}

@test "on a terminal, kit platform hands off to kittui with its stage; a form stays in text" {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
  repo-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run pty "'${kit_root}/bin/kit' platform settings"
  [[ "$output" == *"kittui native-native platform settings"* ]]
  run pty "'${kit_root}/bin/kit' platform settings --json"
  [[ "$output" != *"kittui"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
