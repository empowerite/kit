#!/usr/bin/env bats
# apps.bats - the app verbs' runner: a forge call's document, documents merged, hashes, marks, and another app's tab
# ----------------------------------------------------------------------------------------------------------------------
# The runner's helpers run in a repository whose origin is on GitHub, through kit-bash, with a forge call stood in by a
# function where a test needs an answer the fake gh cannot give. apps:main runs through a verb written for the tests,
# kit demo, in a copy of the kit.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo >/dev/null
  git commit --quiet --allow-empty --message one
  git remote add origin https://github.com/o/r.git
  fake-tools gh
  kit_bash_modules=apps
}

@test "apps:forge on a primary with no forge, or a forge that lacks the feature, is a document with no rows" {
  run --separate-stderr kit-bash 'apps:forge milestones milestones milestone'
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
  git remote remove origin
  run --separate-stderr kit-bash 'apps:forge releases releases release'
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "apps:forge on a forge that has the feature off here says so under errors, and exits 0" {
  run --separate-stderr kit-bash 'forge:releases() { return 3; }; apps:forge releases releases release'
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[{"feature":"releases","off":true}]}' ]
}

@test "apps:forge on a call that fails is the forge's line and 1, with nothing printed" {
  run --separate-stderr kit-bash 'forge:releases() { error "run: gh auth status (x)"; return 1; }
    apps:forge releases releases release'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "$stderr" = '❌ run: gh auth status (x)' ]
}

@test "apps:forge folds each object a call prints through its def, and a capped call's last line into limited" {
  mkdir -p "${BATS_TEST_TMPDIR}/jq"
  { cat "${kit_root}/modules/jq/apps.jq"; printf 'def probe: {key: .n, n, links: {}};\n'; } \
    > "${BATS_TEST_TMPDIR}/jq/apps.jq"
  run --separate-stderr kit-bash "kit_jq=${BATS_TEST_TMPDIR}/jq"'
    forge:releases() { printf "%s\n" "{\"n\": 2}" "{\"n\": 1}" "{\"limited\": 2}"; }
    apps:forge releases releases probe'
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[{"key":2,"n":2,"links":{}},{"key":1,"n":1,"links":{}}],"errors":[],"limited":2}' ]
  run --separate-stderr kit-bash 'apps:forge releases releases "probe; 1"'
  [ "$status" -eq 2 ]
}

@test "apps:merge joins the rows and the errors in order, limited the least of those that say one" {
  run kit-bash 'apps:merge "{\"rows\":[{\"key\":1}],\"errors\":[],\"limited\":50}" \
    "{\"rows\":[{\"key\":2}],\"errors\":[{\"feature\":\"x\",\"off\":true}]}" \
    "{\"rows\":[],\"errors\":[],\"limited\":20}"'
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[{"key":1},{"key":2}],"errors":[{"feature":"x","off":true}],"limited":20}' ]
  run kit-bash 'apps:merge'
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "apps:hash is nothing for paths that do not exist, and moves with the contents of those that do" {
  run kit-bash 'apps:hash nope also-nope'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  mkdir -p d && printf 'one\n' > d/x
  run kit-bash 'apps:hash nope d'
  [[ "$output" =~ ^[0-9a-f]{64}$ ]]
  before=$output
  printf 'two\n' > d/x
  run kit-bash 'apps:hash nope d'
  [ "$output" != "${before}" ]
}

@test "apps:forge-mark joins the forge's marks of the kinds, and is nothing when one is null or there is no forge" {
  run kit-bash 'forge:marks() { printf "%s\n" "{\"issues\":\"i\",\"runs\":\"r\",\"releases\":null}"; }
    apps:forge-mark issues runs; printf "|"; apps:forge-mark issues releases; printf "|"'
  [ "$status" -eq 0 ]
  [ "$output" = $'i r\n||' ]
  run --separate-stderr kit-bash 'forge:marks() { error "run: gh auth status (the marks could not be read)"; return 1; }
    apps:forge-mark issues'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  git remote remove origin
  run --separate-stderr kit-bash 'apps:forge-mark issues'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "a verb's tab of another app is refused with --json, naming the command that prints it; its own stage prints" {
  root="${BATS_TEST_TMPDIR}/kit"
  mkdir -p "${root}"
  cp -R "${kit_root}/bin" "${kit_root}/libexec" "${kit_root}/modules" "${kit_root}/packages" "${root}/"
  cat > "${root}/bin/kit-demo" <<'EOF'
#!/usr/bin/env bash
# kit-demo - a verb written for the tests (kit verb: plumbing)
# ----------------------------------------------------------------------------------------------------------------------
# kit demo [--text | --json | --jq]
#
# kit demo --stages
#
# kit demo --marks
#
# kit demo one [--text | --json | --jq]
#
# kit demo backlog [--text | --json | --jq]
#
#   --text    text
#   --json    JSON
#   --jq      formatted
#   --stages  the stages
#   --marks   the marks
#   one       its own stage
#   backlog   another app's tab
set -o errexit -o nounset -o pipefail
source "$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/../modules/bash/1st.sh"
kit:enter
use apps
guard:arguments demo "$@" || exit $?
apps:main demo "$@"
EOF
  chmod +x "${root}/bin/kit-demo"
  cat > "${root}/modules/bash/apps/demo.sh" <<'EOF'
# shellcheck shell=bash
demo_stages=(one)
declare -gA demo_elsewhere=([backlog]='kit flow backlog')
apps/demo:one() { printf '{"rows":[{"key":"a","links":{}}],"errors":[]}\n'; }
EOF
  run --separate-stderr "${root}/bin/kit" demo backlog --json
  [ "$status" -eq 2 ]
  [ -z "$output" ]
  [[ "$stderr" == *"run: kit flow backlog --json"* ]]
  run --separate-stderr "${root}/bin/kit" demo one --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[{"key":"a","links":{}}],"errors":[]}' ]
  run --separate-stderr "${root}/bin/kit" demo --marks
  [ "$output" = '{"one":null}' ]
  run --separate-stderr "${root}/bin/kit" demo --json
  [ "$output" = '{"stages":["one"],"one":{"rows":[{"key":"a","links":{}}],"errors":[]}}' ]
}

# ----------------------------------------------------------------------------------------------------------------------
