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
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[{"n":1,"member":"github.com/o/a"},{"n":2,"member":"github.com/o/a"},{"n":3,"member":"github.com/o/c"}],"errors":[{"member":"github.com/o/b","error":"gh cannot read o/b"}]}' ]
}

@test "across:merge tags an object stage's object with its member" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '{"branch":"main"}' > "$d/0.out"; printf 0 > "$d/0.status"
  : > "$d/0.err"
  run kit-bash "across:merge '$d' object"
  [ "$output" = '{"rows":[{"branch":"main","member":"github.com/o/a"}],"errors":[]}' ]
}

@test "kit flow desk --workspace prints each cloned member's desk, tagged, in the group's order" {
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow desk --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.member, .branch]]' <<<"$output")" = '[["github.com/o/a","main"],["github.com/o/b","main"]]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "a member whose checkout is gone since the roots were walked is its line in errors, and the others still draw" {
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run kit-bash "across:members() { printf '%s\t%s\n' github.com/o/a '${BATS_TEST_TMPDIR}/code/a' \
    github.com/o/b '${BATS_TEST_TMPDIR}/gone'; }; across:stage w desk --json"
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[].member]' <<<"$output")" = '["github.com/o/a"]' ]
  [ "$(jq -r '.errors[0].member' <<<"$output")" = github.com/o/b ]
  [ "$(jq -r '.errors[0].error' <<<"$output")" = "${BATS_TEST_TMPDIR}/gone is gone; run: kit workspaces --show w" ]
}

@test "a group with no member cloned prints no rows and no errors, and exits 0" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow desk --workspace w --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "kit flow --workspace refuses a workspace it does not know, and a pull request's number, with 2" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow desk --workspace nosuch --json
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" flow changes --workspace w --json 7
  [ "$status" -eq 2 ]
  [[ "$output" == *"run kit flow changes <number> in it"* ]]
}

@test "without --workspace, kit flow desk --json prints the desk's object as before" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  cd "${BATS_TEST_TMPDIR}/code/a"
  run "${kit_root}/bin/kit" flow desk --json
  [ "$status" -eq 0 ]
  [ "$(jq -r 'type' <<<"$output")" = object ]
}

@test "kit workspace repos --workspace gives a row per member, cloned or not, with its desk's branch" {
  two-desks
  git -C "${BATS_TEST_TMPDIR}/code/a" commit --quiet --allow-empty --message ahead
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" workspace repos --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.repository, .by, .branch, .ahead, .checkout != null, .error]]' <<<"$output")" = \
    '[["github.com/o/a","fixed","main",1,true,null],["github.com/o/b","fixed","main",0,true,null],["github.com/o/c","fixed",null,null,false,null]]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "kit workspace repos in a repository lists it alone; outside one, without --workspace, it refuses with 2" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  cd "${BATS_TEST_TMPDIR}/code/a"
  run "${kit_root}/bin/kit" workspace repos --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.repository, .by, .branch]]' <<<"$output")" = '[["github.com/o/a","here","main"]]' ]
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" workspace repos --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit workspace reads a repository: run it in one, or give --workspace"* ]]
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
  run "${kit_root}/bin/kit" workspace repos --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.worktrees, .branches]]' <<<"$output")" = "$(jq -nc --arg a "$(cd "${a}" && pwd -P)" \
    '[[[{branch: null, path: "\($a)/worktrees/loose"}, {branch: "topic", path: "\($a)/worktrees/topic"}], ["idle"]],
      [[], []], [null, null]]')" ]
}

@test "kit workspace repos draws a line per member, its branch with what it is ahead, and not cloned" {
  two-desks
  git -C "${BATS_TEST_TMPDIR}/code/a" commit --quiet --allow-empty --message ahead
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" workspace repos --workspace w --text
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "repository "*" forge "*" branch "*" ahead "*" checkout "* ]]
  [[ "${lines[1]}" =~ ^github\.com/o/a\ +github\ +main\ +1\ +0\ +${BATS_TEST_TMPDIR}/code/a ]]
  [[ "${lines[3]}" =~ ^github\.com/o/c\ +github\ +not\ cloned ]]
}

@test "kit flow --stages --workspace lists kit's eight stages, and repos is kit workspace's first" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --stages --workspace w
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = backlog ]
  [ "${#lines[@]}" -eq 8 ]
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

@test "kit flow --workspace --json is one object: the stages, and each stage across the members" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --workspace w --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -r '.stages[0]' <<<"$output")" = backlog ]
  [ "$(jq -r 'has("repos")' <<<"$output")" = false ]
  [ "$(jq -c '[.desk.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  [ "$(jq -c '[.worktrees.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  [ "$(jq -c '.backlog.rows' <<<"$output")" = '[]' ]
}

@test "kit flow --workspace as text draws each cloned member's picture under its name, and no member not cloned" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --workspace w --no-fetch --text
  [ "$status" -eq 0 ]
  [ "$(grep -c "^📚 github.com/o/[ab]$" <<<"$output")" -eq 2 ]
  [[ "$output" != *"github.com/o/c"* ]]
}

@test "kit flow --workspace as text with no member cloned says so once" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --workspace w --no-fetch --text
  [ "$status" -eq 0 ]
  [ "$(grep -c "none of w's members is cloned" <<<"$output")" -eq 1 ]
}

@test "kit flow --marks --workspace changes a stage's mark when one member's changes" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --marks --workspace w
  [ "$status" -eq 0 ]
  before=$(jq -r .desk <<<"$output")
  [ "${before}" != null ]
  [ "$(jq -r 'has("repos")' <<<"$output")" = false ]
  run "${kit_root}/bin/kit" workspace --marks --workspace w
  repos=$(jq -r .repos <<<"$output")
  printf 'x\n' > "${BATS_TEST_TMPDIR}/code/b b/x"
  git -C "${BATS_TEST_TMPDIR}/code/a" switch --quiet --create other
  run "${kit_root}/bin/kit" flow --marks --workspace w
  [ "$(jq -r .desk <<<"$output")" != "${before}" ]
  run "${kit_root}/bin/kit" workspace --marks --workspace w
  [ "$(jq -r .repos <<<"$output")" != "${repos}" ]
}

@test "across:quota-merge lists every forge the members touch once" {
  d=${BATS_TEST_TMPDIR}/q && mkdir -p "$d"
  printf '[{"forge":"github","limits":[]}]' > "$d/0.out"; printf 0 > "$d/0.status"
  printf '[{"forge":"github","limits":[]},{"forge":"gitlab","limits":[]}]' > "$d/1.out"; printf 0 > "$d/1.status"
  : > "$d/2.out"; printf 1 > "$d/2.status"
  run kit-bash "across:quota-merge '$d'"
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.[].forge]' <<<"$output")" = '["github","gitlab"]' ]
}

@test "kit flow --me --workspace refuses with 2" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --me --workspace w
  [ "$status" -eq 2 ]
}


@test "--workspace=w is --workspace w, and an empty workspace is refused with 2" {
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow desk --workspace=w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  run "${kit_root}/bin/kit" flow desk --workspace '' --json
  [ "$status" -eq 2 ]
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
  run "${kit_root}/bin/kit" flow desk --workspace bogus
  [ "$status" -eq 2 ]
  [[ "$output" != *"none of bogus"* ]]
}

@test "across:merge keeps what a member printed when it exited 1, and names an exit with nothing said" {
  d=${BATS_TEST_TMPDIR}/m && mkdir -p "$d"
  printf 'github.com/o/a' > "$d/0.member"; printf '{"fetched":true}' > "$d/0.out"; printf 1 > "$d/0.status"
  : > "$d/0.err"
  printf 'github.com/o/b' > "$d/1.member"; : > "$d/1.out"; printf 3 > "$d/1.status"; : > "$d/1.err"
  run kit-bash "across:merge '$d' object incoming"
  [ "$output" = '{"rows":[{"fetched":true,"member":"github.com/o/a"}],"errors":[{"member":"github.com/o/b","error":"kit flow incoming exited 3"}]}' ]
}

@test "over a workspace, the stages are kit's eight, whatever the kit.yml where kit runs says" {
  local-desks
  cd "${BATS_TEST_TMPDIR}/code/a"
  run "${kit_root}/bin/kit" flow --stages --workspace w
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = backlog ]
  [ "${#lines[@]}" -eq 8 ]
}

@test "on a terminal, kit flow --workspace hands off to kittui; --json, --text, and a stage stay across the members" {
  fake-tui
  two-desks
  cd "${BATS_TEST_TMPDIR}"
  run pty "'${kit_root}/bin/kit' flow --workspace w --no-fetch"
  [[ "$output" == *"kittui native-native flow --workspace w --no-fetch"* ]]
  run pty "'${kit_root}/bin/kit' flow --workspace=w"
  [[ "$output" == *"kittui native-native flow --workspace w"* ]]
  run pty "'${kit_root}/bin/kit' flow --workspace w --json --no-fetch"
  [[ "$output" == *'"stages":'* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow --workspace w --text --no-fetch"
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow desk --workspace w"
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow --stages --workspace w"
  [[ "$output" == *"backlog"* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow --marks --workspace w"
  [[ "$output" == *'"desk":'* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow --quota --workspace w"
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow --me --workspace w"
  [[ "$output" != *"kittui"* ]]
}
