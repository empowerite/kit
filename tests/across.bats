#!/usr/bin/env bats
# across.bats - kit flow across a workspace: its members and their checkouts, each member's kit flow merged into one
# document per stage, every row tagged with its member, and the repos stage
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

@test "kit flow repos --workspace gives a row per member, cloned or not, with its desk's branch" {
  two-desks
  git -C "${BATS_TEST_TMPDIR}/code/a" commit --quiet --allow-empty --message ahead
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow repos --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.repository, .by, .branch, .ahead, .checkout != null, .error]]' <<<"$output")" = \
    '[["github.com/o/a","fixed","main",1,true,null],["github.com/o/b","fixed","main",0,true,null],["github.com/o/c","fixed",null,null,false,null]]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "kit flow repos in a repository lists it alone; outside one, without --workspace, it refuses with 2" {
  desk-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  cd "${BATS_TEST_TMPDIR}/code/a"
  run "${kit_root}/bin/kit" flow repos --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.repository, .by, .branch]]' <<<"$output")" = '[["github.com/o/a","here","main"]]' ]
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow repos --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"repos lists a workspace's repositories"* ]]
}

@test "kit flow repos draws a line per member, its branch with what it is ahead, and not cloned" {
  two-desks
  git -C "${BATS_TEST_TMPDIR}/code/a" commit --quiet --allow-empty --message ahead
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow repos --workspace w
  [ "$status" -eq 0 ]
  [[ "$output" == *"github.com/o/a"*"main ↑1"* ]]
  [[ "$output" == *"github.com/o/c"*"not cloned"* ]]
}

@test "kit flow --stages --workspace puts repos first" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/o/a]}}'
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --stages --workspace w
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = repos ]
  [ "${#lines[@]}" -eq 9 ]
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

@test "kit flow --workspace --json is one object: the stages, repos, and each stage across the members" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --workspace w --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -r '.stages[0]' <<<"$output")" = repos ]
  [ "$(jq -c '[.desk.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  [ "$(jq -c '[.worktrees.rows[].member]' <<<"$output")" = '["github.com/o/a","github.com/o/b"]' ]
  [ "$(jq -c '[.repos.rows[].repository]' <<<"$output")" = '["github.com/o/a","github.com/o/b","github.com/o/c"]' ]
  [ "$(jq -c '.backlog.rows' <<<"$output")" = '[]' ]
}

@test "kit flow --workspace as text draws repos, then each cloned member's picture under it" {
  local-desks
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" flow --workspace w --no-fetch --text
  [ "$status" -eq 0 ]
  [ "$(grep -c "^📚 github.com/o/[ab]$" <<<"$output")" -eq 2 ]
  [[ "$output" == *"github.com/o/c"*"not cloned"* ]]
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
  repos=$(jq -r .repos <<<"$output")
  [ "${before}" != null ]
  printf 'x\n' > "${BATS_TEST_TMPDIR}/code/b b/x"
  git -C "${BATS_TEST_TMPDIR}/code/a" switch --quiet --create other
  run "${kit_root}/bin/kit" flow --marks --workspace w
  [ "$(jq -r .desk <<<"$output")" != "${before}" ]
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
