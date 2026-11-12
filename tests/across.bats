#!/usr/bin/env bats
# across.bats - kit flow across a workspace: its members and their checkouts, each member's kit flow merged into one
# document per stage, every row tagged with its member, and kit workspace's repos stage
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them, so the
# workspace file starts missing.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  kit_bash_modules=across
}

# clone-at <path> <url> makes a repository at <path> whose origin is <url>; nothing is fetched.
clone-at() {
  git init --quiet "$1"
  git -C "$1" remote add origin "$2"
}

# workspaces-file writes what it reads to the workspace file.
workspaces-file() {
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  cat > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
}

# desk-at <path> <member> makes a desk at <path>, cloned from a bare origin elsewhere with one commit pushed, and then
# points origin at <member> on its forge, so workspaces finds its member; nothing fetches it again.
desk-at() {
  local origin
  origin="${BATS_TEST_TMPDIR}/origins/${2//\//_}.git"
  mkdir -p "${BATS_TEST_TMPDIR}/origins" "$(dirname "$1")"
  git init --quiet --bare --initial-branch=main "${origin}"
  git clone --quiet "${origin}" "$1" 2>/dev/null
  printf 'one\n' > "$1/a"
  git -C "$1" add a && git -C "$1" commit --quiet --message one && git -C "$1" push --quiet origin main
  git -C "$1" remote set-head origin main
  git -C "$1" remote set-url origin "https://$2.git"
}

# two-desks makes the desks of github.com/o/a and github.com/o/b, the second's path with a space, and the group w of
# a, b, and github.com/o/c, which has none.
two-desks() {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  desk-at "${BATS_TEST_TMPDIR}/code/b b" github.com/o/b
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, github.com/o/b, github.com/o/c]}
EOF
}

# fake-tui puts the fake go and the fake kittui first on PATH, as flow.bats does: kit builds kittui from projects/tui/
# with the fake go, which writes a script that prints "kittui native-native" and its arguments.
fake-tui() {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
}

@test "across:members gives each member and its checkout, in the group's order, empty for one not cloned" {
  clone-at "${BATS_TEST_TMPDIR}/code/b" https://github.com/o/b.git
  clone-at "${BATS_TEST_TMPDIR}/code/a dir/a" https://github.com/o/a.git
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, github.com/o/c, github.com/o/b]}
EOF
  run kit-bash "across:members w"
  [ "$status" -eq 0 ]
  [ "$output" = "$(printf '%s\t%s\n' github.com/o/a "${BATS_TEST_TMPDIR}/code/a dir/a" github.com/o/c '' \
    github.com/o/b "${BATS_TEST_TMPDIR}/code/b")" ]
}

@test "across:members refuses a workspace that is neither a group nor a repository, and none, with 2" {
  run kit-bash "across:members nosuch"
  [ "$status" -eq 2 ]
  [[ "$output" == *"'nosuch' is no workspace"* ]]
  run kit-bash "across:members none"
  [ "$status" -eq 2 ]
  [[ "$output" == *"'none' is no workspace"* ]]
}

@test "across:merge tags each row with its member, in order, and turns a failed member into its line" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '[{"n":1},{"n":2}]' > "$d/0.out"; printf 0 > "$d/0.status"
  : > "$d/0.err"
  printf 'github.com/o/b' > "$d/1.member"; : > "$d/1.out"; printf 1 > "$d/1.status"
  printf 'a warning\n❌ gh cannot read o/b\n' > "$d/1.err"
  printf 'github.com/o/c' > "$d/2.member"; printf '[{"n":3}]' > "$d/2.out"; printf 0 > "$d/2.status"; : > "$d/2.err"
  run kit-bash "across:merge '$d' array"
  [ "$status" -eq 1 ]
  [ "$output" = '{"rows":[{"n":1,"member":"github.com/o/a"},{"n":2,"member":"github.com/o/a"},{"n":3,"member":"github.com/o/c"}],"errors":[{"member":"github.com/o/b","error":"gh cannot read o/b"}]}' ]
}

@test "across:merge tags an object stage's object with its member" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '{"branch":"main"}' > "$d/0.out"; printf 0 > "$d/0.status"
  : > "$d/0.err"
  run kit-bash "across:merge '$d' object"
  [ "$output" = '{"rows":[{"branch":"main","member":"github.com/o/a"}],"errors":[]}' ]
}

@test "kit --workspace= flow desk prints each cloned member's desk, tagged, in the group's order" {
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow desk --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.member, .branch]]' <<<"$output")" = '[["github.com/o/a","main"],["github.com/o/b","main"]]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "a member whose checkout is gone since the roots were walked is its line in errors, and the others still draw" {
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run kit-bash "across:members() { printf '%s\t%s\n' github.com/o/a '${BATS_TEST_TMPDIR}/code/a' \
    github.com/o/b '${BATS_TEST_TMPDIR}/gone'; }; across:stage w desk --json"
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.rows[].member]' <<<"$output")" = '["github.com/o/a"]' ]
  [ "$(jq -r '.errors[0].member' <<<"$output")" = github.com/o/b ]
  [ "$(jq -r '.errors[0].error' <<<"$output")" = "${BATS_TEST_TMPDIR}/gone is gone; run: kit workspaces --show=w" ]
}

@test "a group with no member cloned prints no rows and no errors, and exits 0" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow desk --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "kit --workspace= flow refuses a workspace it does not know, and a pull request's number, with 2" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=nosuch flow desk --json
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" --workspace=w flow changes --json 7
  [ "$status" -eq 2 ]
  [[ "$output" == *"run kit flow changes <number> in it"* ]]
}

@test "a workspace file that does not parse stops kit flow and kit workspace over a workspace with 1, not 2" {
  workspaces-file <<<'workspaces: [x'
  cd "${BATS_TEST_TMPDIR}"
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w flow desk --json
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w flow --stages
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace repos --json
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace --json
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "a file kit workspaces refuses is refused by kit flow and kit workspace, with 1 and no document" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a], exclude: [nonsense]}}'
  for words in "flow desk" "workspace repos" "workspace branches"
  do
    # shellcheck disable=SC2086  # each holds its words
    run --separate-stderr "${kit_root}/bin/kit" --workspace=w ${words} --json
    [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"'nonsense' in w's exclude names no repository"* ]]
  done
}

@test "without --workspace=, kit flow desk --json prints the desk's object as before" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  cd "${BATS_TEST_TMPDIR}/code/a"
  run "${kit_root}/bin/kit" flow desk --json
  [ "$status" -eq 0 ]
  [ "$(jq -r 'type' <<<"$output")" = object ]
}

@test "kit --workspace= workspace repos gives a row per member, cloned or not, with its desk's branch" {
  two-desks
  git -C "${BATS_TEST_TMPDIR}/code/a" commit --quiet --allow-empty --message ahead
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w workspace repos --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.repository, .by, .branch, .ahead, .checkout != null, .error]]' <<<"$output")" = \
    '[["github.com/o/a","fixed","main",1,true,null],["github.com/o/b","fixed","main",0,true,null],["github.com/o/c","fixed",null,null,false,null]]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "kit workspace repos in a repository lists it alone; outside one, without --workspace=, it refuses with 2" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  cd "${BATS_TEST_TMPDIR}/code/a"
  run "${kit_root}/bin/kit" workspace repos --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.repository, .by, .branch]]' <<<"$output")" = '[["github.com/o/a","here","main"]]' ]
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" workspace repos --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit workspace reads a repository: run it in one, or name a workspace: kit --workspace=<workspace> workspace"* ]]
}

@test "kit workspace repos in a linked worktree gives the clone as the checkout, as its branch is the clone's" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  git -C "${BATS_TEST_TMPDIR}/code/a" worktree add --quiet -b topic "${BATS_TEST_TMPDIR}/code/a/worktrees/topic"
  cd "${BATS_TEST_TMPDIR}/code/a/worktrees/topic"
  run "${kit_root}/bin/kit" workspace repos --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.checkout, .checkouts, .branch]]' <<<"$output")" = \
    "$(jq -nc --arg a "$(cd ../.. && pwd -P)" '[[$a, [$a], "main"]]')" ]
}

@test "kit workspace repos lists a member's linked worktrees, and its local branches no checkout has; none, not cloned" {
  two-desks
  local a="${BATS_TEST_TMPDIR}/code/a"
  git -C "${a}" worktree add --quiet -b topic "${a}/worktrees/topic"
  git -C "${a}" worktree add --quiet --detach "${a}/worktrees/loose"
  git -C "${a}" branch idle
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w workspace repos --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.worktrees, .branches]]' <<<"$output")" = "$(jq -nc --arg a "$(cd "${a}" && pwd -P)" \
    '[[[{branch: null, path: "\($a)/worktrees/loose"}, {branch: "topic", path: "\($a)/worktrees/topic"}], ["idle"]],
      [[], []], [null, null]]')" ]
}

@test "kit workspace repos draws a line per member, its branch with what it is ahead, and not cloned" {
  two-desks
  git -C "${BATS_TEST_TMPDIR}/code/a" commit --quiet --allow-empty --message ahead
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w workspace repos --text
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "repository "*" forge "*" branch "*" ahead "*" checkout "* ]]
  [[ "${lines[1]}" =~ ^github\.com/o/a\ +github\ +main\ +1\ +0\ +${BATS_TEST_TMPDIR}/code/a ]]
  [[ "${lines[3]}" =~ ^github\.com/o/c\ +github\ +not\ cloned ]]
}

@test "kit --workspace= flow --stages lists kit's seven stages, and repos is kit workspace's first" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow --stages
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = issues ]
  [ "${#lines[@]}" -eq 7 ]
  [[ "$output" != *repos* ]]
  run "${kit_root}/bin/kit" workspace --stages
  [ "${lines[0]}" = repos ]
  run "${kit_root}/bin/kit" flow repos --json
  [ "$status" -eq 2 ]
}

# local-desks makes two-desks' desks draw only desk and worktrees, which read no forge.
local-desks() {
  two-desks
  local d
  for d in "${BATS_TEST_TMPDIR}/code/a" "${BATS_TEST_TMPDIR}/code/b b"
  do
    printf 'flow: [desk, worktrees]\n' > "${d}/kit.yml"
    git -C "${d}" add kit.yml && git -C "${d}" commit --quiet --message settings
  done
}

@test "kit --workspace= flow --json is one object: the stages, and each stage across the members" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -r '.stages[0]' <<<"$output")" = issues ]
  [ "$(jq -r 'has("repos")' <<<"$output")" = false ]
  [ "$(jq -c '[.desk.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  [ "$(jq -c '[.worktrees.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  [ "$(jq -c '.issues.rows' <<<"$output")" = '[]' ]
}

@test "kit --workspace= flow as text draws each cloned member's picture under its name, and no member not cloned" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow --no-fetch --text
  [ "$status" -eq 0 ]
  [ "$(grep -c "^📚 github.com/o/[ab]$" <<<"$output")" -eq 2 ]
  [[ "$output" != *"github.com/o/c"* ]]
}

@test "kit --workspace= flow as text with no member cloned says so once" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow --no-fetch --text
  [ "$status" -eq 0 ]
  [ "$(grep -c "none of w's members is cloned" <<<"$output")" -eq 1 ]
  [[ "$output" == *"none of w's members is cloned under the roots; run: kit workspaces --show=w"* ]]
}

@test "kit --workspace= flow --marks changes a stage's mark when one member's changes" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow --marks
  [ "$status" -eq 0 ]
  before=$(jq -r .desk <<<"$output")
  [ "${before}" != null ]
  [ "$(jq -r 'has("repos")' <<<"$output")" = false ]
  run "${kit_root}/bin/kit" --workspace=w workspace --marks
  repos=$(jq -r .repos <<<"$output")
  printf 'x\n' > "${BATS_TEST_TMPDIR}/code/b b/x"
  git -C "${BATS_TEST_TMPDIR}/code/a" switch --quiet --create other
  run "${kit_root}/bin/kit" --workspace=w flow --marks
  [ "$(jq -r .desk <<<"$output")" != "${before}" ]
  run "${kit_root}/bin/kit" --workspace=w workspace --marks
  [ "$(jq -r .repos <<<"$output")" != "${repos}" ]
}

@test "across:quota-merge lists every forge the members touch once" {
  d=${BATS_TEST_TMPDIR}/q && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '[{"forge":"github","limits":[]}]' > "$d/0.out"
  printf 0 > "$d/0.status"; : > "$d/0.err"
  printf 'gitlab.com/o/b' > "$d/1.member"
  printf '[{"forge":"github","limits":[]},{"forge":"gitlab","limits":[]}]' > "$d/1.out"; printf 0 > "$d/1.status"
  : > "$d/1.err"
  run kit-bash "across:quota-merge '$d'"
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.[].forge]' <<<"$output")" = '["github","gitlab"]' ]
}

@test "across:quota-merge says each member whose rate limits could not be read, and exits 1 after the rest" {
  d=${BATS_TEST_TMPDIR}/q && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '[{"forge":"github","limits":[]}]' > "$d/0.out"
  printf 0 > "$d/0.status"; : > "$d/0.err"
  printf 'gitlab.com/o/b' > "$d/1.member"; : > "$d/1.out"; printf 1 > "$d/1.status"
  printf '❌ run: glab auth status (the rate limits could not be read)\n' > "$d/1.err"
  run --separate-stderr kit-bash "across:quota-merge '$d'"
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.[].forge]' <<<"$output")" = '["github"]' ]
  [ "$stderr" = "❌ gitlab.com/o/b: run: glab auth status (the rate limits could not be read)" ]
  printf 1 > "$d/0.status"; : > "$d/0.out"; printf '❌ run: gh auth status\n' > "$d/0.err"
  run --separate-stderr kit-bash "across:quota-merge '$d'"
  [ "$status" -eq 1 ]
  [ "$output" = '' ]
  [ "${#stderr_lines[@]}" -eq 2 ]
}

@test "a member whose limits cannot be read, or whose kit.yml does not parse, says so where it is drawn" {
  two-desks
  PATH="$(shims gh)" run --separate-stderr "${kit_root}/bin/kit" --workspace=w flow --quota
  [ "$status" -eq 1 ]
  [ "$output" = '' ]
  [[ "$stderr" == *"❌ github.com/o/a: "*"❌ github.com/o/b: "* ]]
  printf 'primary: [\n' > "${BATS_TEST_TMPDIR}/code/a/kit.yml"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show github.com/o/c --json
  [ "$status" -eq 0 ]
  [ -z "$stderr" ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show=github.com/o/a --json
  [ "$status" -eq 0 ]
  [[ "$stderr" == *"${BATS_TEST_TMPDIR}/code/a/kit.yml does not parse, so its primary is read as origin"* ]]
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace repos --json
  [ "$status" -eq 1 ]
  [[ "$(jq -r '.rows[0].error' <<<"$output")" == *"kit.yml does not parse, so its primary is read as origin"* ]]
  [ "$(jq -r '.rows[0].repository' <<<"$output")" = github.com/o/a ]
}

@test "a member on a host kit has no forge for draws its forge's stages as nothing to read, and the picture exits 0" {
  local-desks
  d="${BATS_TEST_TMPDIR}/code/b b"
  printf 'flow: [desk, worktrees, issues]\n' > "$d/kit.yml"
  git -C "$d" commit --quiet --all --message settings
  git -C "$d" remote set-url origin https://bitbucket.org/o/b.git
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, bitbucket.org/o/b]}
EOF
  cd "${BATS_TEST_TMPDIR}"
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w flow --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.issues' <<<"$output")" = '{"rows":[],"errors":[]}' ]
  [ "$(jq -c '[.[] | objects | .errors[]]' <<<"$output")" = '[]' ]
  [ "$(jq -c '[.desk.rows[].member]' <<<"$output")" = '["github.com/o/a","bitbucket.org/o/b"]' ]
}

@test "kit workspace over a workspace prints a stage read in part and exits 1, and repos a desk it could not read" {
  local-desks
  printf 'bash_path: [nosuch:dir]\n' > "${BATS_TEST_TMPDIR}/code/b b/kit.yml"
  cd "${BATS_TEST_TMPDIR}"
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace remotes --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.rows[].member] | unique' <<<"$output")" = '["github.com/o/a"]' ]
  [ "$(jq -r '.errors[0].member' <<<"$output")" = github.com/o/b ]
  [[ "$(jq -r '.errors[0].error' <<<"$output")" == "bash_path must start with kit:modules/bash/"* ]]
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace repos --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.rows[] | [.repository, .branch]]' <<<"$output")" = \
    '[["github.com/o/a","main"],["github.com/o/b",null],["github.com/o/c",null]]' ]
  [ "$(jq -r '.rows[0].error' <<<"$output")" = null ]
  [[ "$(jq -r '.rows[1].error' <<<"$output")" == "bash_path must start with kit:modules/bash/"* ]]
  git -C "${BATS_TEST_TMPDIR}/code/b b" checkout --quiet kit.yml
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace repos --json
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace remotes --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.errors' <<<"$output")" = '[]' ]
}

@test "kit workspace --marks over a workspace names a member it could not read, says its line, and exits 1" {
  local-desks
  printf 'bash_path: [nosuch:dir]\n' > "${BATS_TEST_TMPDIR}/code/b b/kit.yml"
  cd "${BATS_TEST_TMPDIR}"
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace --marks
  [ "$status" -eq 1 ]
  [ "$(jq -c '.errors | map(.member)' <<<"$output")" = '["github.com/o/b"]' ]
  [[ "$(jq -r '.errors[0].error' <<<"$output")" == "bash_path must start with kit:modules/bash/"* ]]
  [ "$(jq -r '.tags | type' <<<"$output")" = string ]
  [ "$(jq -r '.repos | type' <<<"$output")" = string ]
  [[ "$stderr" == "❌ github.com/o/b: bash_path must start with kit:modules/bash/"* ]]
  git -C "${BATS_TEST_TMPDIR}/code/b b" checkout --quiet kit.yml
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspace --marks
  [ "$status" -eq 0 ]
  [ "$(jq -r 'has("errors")' <<<"$output")" = false ]
}

@test "kit --workspace= flow --me refuses with 2" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow --me
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit flow --me and --capabilities read the primary's forge alone, so they take no --workspace="* ]]
  run "${kit_root}/bin/kit" --workspace=none flow --capabilities
  [ "$status" -eq 2 ]
}


@test "--workspace= is a kit option, before the verb; an empty one, one whose workspace is the next word, one after the verb, and one beside --version= are refused with 2" {
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=w flow desk --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  run "${kit_root}/bin/kit" --workspace='' flow desk --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"--workspace= needs a workspace"* ]]
  run "${kit_root}/bin/kit" --workspace w flow desk --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"--workspace takes its workspace after =: kit --workspace=<workspace> <verb>"* ]]
  for words in "flow desk --workspace=w --json" "workspace repos --workspace w" "workspaces --current --workspace=w"
  do
    # shellcheck disable=SC2086  # each holds its words
    run "${kit_root}/bin/kit" ${words}
    [ "$status" -eq 2 ]
    [[ "$output" == *"--workspace is a kit option, before the verb: kit --workspace=<workspace> ${words%% *}"* ]]
  done
  run "${kit_root}/bin/kit" --workspace=w --version=9.9.9 flow desk --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"--workspace= runs each member on the release it pins, so it takes no --version= or KIT_VERSION"* ]]
}

@test "a command over a workspace walks the roots once, and the JSON picture reads no member's desk twice" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  calls=${BATS_TEST_TMPDIR}/calls
  for words in "--marks" "--no-fetch --json" "--no-fetch --text"
  do
    : > "${calls}"
    kit-bash "use across; workspaces:checkouts() { printf walk >> '${calls}'; printf '\\n' >> '${calls}'
      printf '%s\\t%s\\n' github.com/o/a '${BATS_TEST_TMPDIR}/code/a' github.com/o/b '${BATS_TEST_TMPDIR}/code/b b'; }
      run() { printf '%s\\n' \"\$*\" >> '${calls}'; \"\${kit_root}/bin/\$1\" \"\${@:2}\"; }
      across:flow w ${words}" >/dev/null 2>&1 || true
    [ "$(grep -c '^walk$' "${calls}")" -eq 1 ]
    [[ "${words}" != *--json* ]] || [ "$(grep -c '^kit flow desk' "${calls}")" -eq 0 ]
  done
}

@test "a stage as text over a workspace kit does not know exits 2" {
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" --workspace=bogus flow desk
  [ "$status" -eq 2 ]
  [[ "$output" != *"none of bogus"* ]]
}

@test "across:merge keeps what a member printed when it exited 1, and names an exit with nothing said" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '{"fetched":true}' > "$d/0.out"; printf 1 > "$d/0.status"
  : > "$d/0.err"
  printf 'github.com/o/b' > "$d/1.member"; : > "$d/1.out"; printf 3 > "$d/1.status"; : > "$d/1.err"
  run kit-bash "across:merge '$d' object incoming"
  [ "$status" -eq 1 ]
  [ "$output" = '{"rows":[{"fetched":true,"member":"github.com/o/a"}],"errors":[{"member":"github.com/o/b","error":"kit flow incoming exited 3"}]}' ]
}

@test "over a workspace, the stages are kit's seven, whatever the kit.yml where kit runs says" {
  local-desks
  cd "${BATS_TEST_TMPDIR}/code/a"
  run "${kit_root}/bin/kit" --workspace=w flow --stages
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = issues ]
  [ "${#lines[@]}" -eq 7 ]
}

@test "on a terminal, kit --workspace= flow hands off to kittui; --json, --text, and a stage stay across the members" {
  fake-tui
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run pty "'${kit_root}/bin/kit' --workspace=w flow --no-fetch"
  [[ "$output" == *"kittui native-native flow --workspace=w --no-fetch"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow"
  [[ "$output" == *"kittui native-native flow --workspace=w"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow --json --no-fetch"
  [[ "$output" == *'"stages":'* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow --text --no-fetch"
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow desk"
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow backlog"
  [[ "$output" == *"kittui native-native flow --workspace=w backlog"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow underway --json"
  [[ "$output" == *"underway is no stage of kit flow"* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow --stages"
  [[ "$output" == *"issues"* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow --marks"
  [[ "$output" == *'"desk":'* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow --quota"
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' --workspace=w flow --me"
  [[ "$output" != *"kittui"* ]]
}

@test "outside a repository the last group used is drawn; --workspace=none draws none; inside one, nothing changes" {
  local-desks
  # with no group used, the file is not read, so one kit refuses changes nothing
  cp "${XDG_CONFIG_HOME}/kit/workspaces.yml" "${BATS_TEST_TMPDIR}/kept.yml"
  workspaces-file <<<'workspaces: [unclosed'
  run --separate-stderr "${kit_root}/bin/kit" flow --json --no-fetch
  [ "$status" -eq 0 ]
  [ "$(jq -r type <<<"$output")" = array ]
  run "${kit_root}/bin/kit" workspace repos --json
  [ "$status" -eq 2 ]
  workspaces-file < "${BATS_TEST_TMPDIR}/kept.yml"
  "${kit_root}/bin/kit" workspaces --use w
  run --separate-stderr "${kit_root}/bin/kit" flow --json --no-fetch
  [ "$(jq -c '[.desk.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  run --separate-stderr "${kit_root}/bin/kit" workspace repos --json
  [ "$(jq -c '[.rows[].by]' <<<"$output")" = '["fixed","fixed","fixed"]' ]
  run --separate-stderr "${kit_root}/bin/kit" --workspace=none flow --json --no-fetch
  [ "$(jq -r type <<<"$output")" = array ]
  run "${kit_root}/bin/kit" --workspace=none workspace repos --json
  [[ "$status" -eq 2 && "$output" == *"kit workspace reads a repository"* ]]
  run "${kit_root}/bin/kit" workspaces --new none github.com/o/a
  [[ "$status" -eq 2 && "$output" == *"--workspace=none names no workspace"* ]]
  cd code/a && workspaces-file <<<'workspaces: [unclosed'
  run --separate-stderr "${kit_root}/bin/kit" flow desk --json
  [[ "$status" -eq 0 && -z "$stderr" && "$(jq -r .branch <<<"$output")" == main ]]
  # outside a repository, its .git directory too, with a group used, the group must be read
  for dir in .git "${BATS_TEST_TMPDIR}"
  do
    cd "${dir}"
    run --separate-stderr "${kit_root}/bin/kit" flow --json --no-fetch
    [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"workspaces.yml"* ]]
  done
}

@test "a bare repository and a .git directory are outside a repository to kit workspaces, kit flow, and kit workspace" {
  local-desks
  git init --quiet --bare "${BATS_TEST_TMPDIR}/bare.git"
  for used in no yes
  do
    [[ "${used}" == no ]] || "${kit_root}/bin/kit" workspaces --use=w
    for dir in "${BATS_TEST_TMPDIR}/bare.git" "${BATS_TEST_TMPDIR}/code/a/.git"
    do
      cd "${dir}"
      run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
      [[ "$status" -eq 0 && -z "$stderr" ]]
      current=$(jq --compact-output '[.kind, .name, .why]' <<<"$output")
      run --separate-stderr "${kit_root}/bin/kit" flow --json --no-fetch
      flow_status=$status flow_output=$output flow_stderr=$stderr
      run --separate-stderr "${kit_root}/bin/kit" workspace repos --json
      [[ "${flow_stderr}" != *fatal:* && "$stderr" != *fatal:* ]]
      if [[ "${used}" == no ]]
      then
        # above desks, where there are none below, and kit workspace's refusal
        [ "${current}" = '["none",null,"none"]' ]
        [[ "${flow_status}" -eq 1 && "${flow_stderr}" == *"no repository at or below $(pwd -P)"* ]]
        [[ "$status" -eq 2 && "$stderr" == *"kit workspace reads a repository"* ]]
      else
        [ "${current}" = '["group","w","last"]' ]
        [ "${flow_status}" -eq 0 ]
        [ "$(jq -c '[.desk.rows[].member]' <<<"${flow_output}")" = '["github.com/o/a","github.com/o/b"]' ]
        [ "$status" -eq 0 ]
        [ "$(jq -c '[.rows[].repository]' <<<"$output")" = '["github.com/o/a","github.com/o/b","github.com/o/c"]' ]
      fi
    done
  done
}
