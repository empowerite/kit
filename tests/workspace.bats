#!/usr/bin/env bats
# workspace.bats - kit workspace: a repository's git, or a workspace's, as rows per stage, in text, as JSON, and in
# kittui
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them, so the
# workspace file starts missing. A repository's origin is a bare one under the test's directory, pointed at a GitHub
# URL once everything is pushed, so the forge is GitHub's and nothing reaches it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
}

# workspaces-file writes what it reads to the workspace file.
workspaces-file() {
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  cat > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
}

# history-at <path> <member> makes a desk at <path> with a history: main pushed, origin's HEAD on it, its worktrees/
# ignored; 7-fix in a linked worktree, its first commit pushed and its second not; old, whose upstream is gone; scratch, which has none; and the
# tags v1, annotated, and wip, lightweight. Then origin points at <member> on GitHub, which nothing fetches.
history-at() {
  local desk=$1 origin
  origin="${BATS_TEST_TMPDIR}/origins/${2//\//_}.git"
  mkdir -p "${BATS_TEST_TMPDIR}/origins" "$(dirname "${desk}")"
  git init --quiet --bare --initial-branch=main "${origin}"
  git clone --quiet "${origin}" "${desk}" 2>/dev/null
  printf 'one\n' > "${desk}/a"
  printf '/worktrees/\n' > "${desk}/.gitignore"
  git -C "${desk}" add a .gitignore && git -C "${desk}" commit --quiet --message one
  git -C "${desk}" push --quiet --set-upstream origin main 2>/dev/null
  git -C "${desk}" remote set-head origin main
  git -C "${desk}" tag --annotate --message 'release 1' v1
  git -C "${desk}" worktree add --quiet -b 7-fix "${desk}/worktrees/7-fix"
  printf 'fix\n' > "${desk}/worktrees/7-fix/b"
  git -C "${desk}/worktrees/7-fix" add b && git -C "${desk}/worktrees/7-fix" commit --quiet --message '#7 fix the thing'
  git -C "${desk}/worktrees/7-fix" push --quiet --set-upstream origin 7-fix 2>/dev/null
  printf 'more\n' >> "${desk}/worktrees/7-fix/b"
  git -C "${desk}/worktrees/7-fix" commit --quiet --all --message 'more of it'
  printf 'loose\n' > "${desk}/worktrees/7-fix/c"
  git -C "${desk}" branch --quiet old
  git -C "${desk}" push --quiet --set-upstream origin old 2>/dev/null
  git -C "${desk}" push --quiet origin --delete old 2>/dev/null
  git -C "${desk}" fetch --quiet --prune
  git -C "${desk}" branch --quiet scratch
  git -C "${desk}" tag wip 7-fix
  git -C "${desk}" remote set-url origin "https://$2.git"
}

# fake-tui puts the fake go and the fake kittui first on PATH, as flow.bats does: kit builds kittui from projects/tui/
# with the fake go, which writes a script that prints "kittui native-native" and its arguments.
fake-tui() {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
}

@test "kit workspace --stages prints the six stages, in order" {
  run "${kit_root}/bin/kit" workspace --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'repos\nremotes\nbranches\nworktrees\ncommits\ntags' ]
}

@test "remotes gives each remote: its forge, its repository past the host, its default branch, and its branches" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run "${kit_root}/bin/kit" workspace remotes --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.rows | map({key, forge, repository, host, push, default, branches, primary, links})' <<<"$output")" \
    = '[{"key":"origin","forge":"github","repository":"o/r","host":"github.com","push":null,"default":"main","branches":2,"primary":"primary","links":{"repository":"https://github.com/o/r","default":"row:branches/origin/main"}}]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
}

@test "branches pairs each branch on a remote with the local ones that pull from it, and gives a gone one and one with no upstream rows of their own" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run "${kit_root}/bin/kit" workspace branches --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.key, .kind, .local, .upstream, .default]]' <<<"$output")" \
    = '[["origin/7-fix","remote",["7-fix"],[1,0],null],["origin/main","remote",["main"],[0,0],"default"],["origin/old","gone",["old"],null,null],["scratch","local",["scratch"],null,null]]' ]
  [ "$(jq -r '.rows[] | select(.key == "origin/7-fix") | .worktree[0]' <<<"$output")" \
    = "$(cd "${BATS_TEST_TMPDIR}/r/worktrees/7-fix" && pwd -P)" ]
  [ "$(jq -r '.rows[] | select(.key == "origin/main") | .links.branch' <<<"$output")" = https://github.com/o/r/tree/main ]
}

@test "worktrees gives the desk first, then each linked worktree: its branch, its ahead and behind, and its paths" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run "${kit_root}/bin/kit" workspace worktrees --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.kind, .branch, .upstream, .track, .base, .changes, .untracked, .links.branch]]' <<<"$output")" \
    = '[["desk","main","origin/main",[0,0],[0,0],0,0,"row:branches/origin/main"],["linked","7-fix","origin/7-fix",[1,0],[2,0],0,1,"row:branches/origin/7-fix"]]' ]
}

@test "commits lists what the local branches hold and the default branch does not: its issue, branches, and whether a remote has it" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run "${kit_root}/bin/kit" workspace commits --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.subject, .issue, .branches, .pushed, (.worktrees | length)]]' <<<"$output")" \
    = '[["more of it",null,["7-fix"],"local",1],["#7 fix the thing","#7",["7-fix"],"pushed",0]]' ]
  [ "$(jq -r '.rows[1].links.issue' <<<"$output")" = https://github.com/o/r/issues/7 ]
  [ "$(jq -r '.rows[0].links.commit' <<<"$output")" = null ]
}

@test "tags tells an annotated tag from a lightweight one, each with its commit" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run "${kit_root}/bin/kit" workspace tags --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.tag, .kind, .tagger]] | sort' <<<"$output")" = '[["v1","annotated","t"],["wip","lightweight",null]]' ]
  [ "$(jq -r '.rows[] | select(.tag == "wip") | .commit' <<<"$output")" = "$(git rev-parse --short 7-fix)" ]
}

@test "from a linked worktree, every stage reads the main checkout, as the desk does" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  desk=$("${kit_root}/bin/kit" workspace branches --json)
  cd worktrees/7-fix
  run "${kit_root}/bin/kit" workspace branches --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | del(.age)]' <<<"$output")" = "$(jq -c '[.rows[] | del(.age)]' <<<"${desk}")" ]
}

@test "as text, a stage is a line of its columns' names and a line per row; the picture is every stage under its name" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run "${kit_root}/bin/kit" workspace remotes --text
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = 'remote  repository  forge   url                         default  branches' ]
  [ "${lines[1]}" = 'origin  o/r         github  https://github.com/o/r.git  main     2' ]
  run "${kit_root}/bin/kit" workspace --text
  [ "$status" -eq 0 ]
  [ "$(grep -c ':$' <<<"$output")" -eq 6 ]
  [[ "$output" == *$'repos:\nrepository '* && "$output" == *$'\n\nremotes:\nremote '* ]]
  [[ "$output" == *$'\n\ntags:\ntag '* ]]
  run "${kit_root}/bin/kit" workspace --json
  [ "$(jq -c '[.stages, (.stages | map(length > 0))]' <<<"$output")" \
    = '[["repos","remotes","branches","worktrees","commits","tags"],[true,true,true,true,true,true]]' ]
  [ "$(jq '.tags.rows | length' <<<"$output")" -eq 2 ]
}

@test "outside a repository, and with no workspace, kit workspace refuses with 2" {
  mkdir -p "${BATS_TEST_TMPDIR}/nowhere" && cd "${BATS_TEST_TMPDIR}/nowhere"
  run "${kit_root}/bin/kit" workspace remotes --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit workspace reads a repository: run it in one, or give --workspace"* ]]
  run "${kit_root}/bin/kit" workspace --stages
  [ "$status" -eq 0 ]
}

@test "--workspace reads every cloned member's checkout, each row tagged with its member, in the group's order" {
  history-at "${BATS_TEST_TMPDIR}/code/a" github.com/o/a
  history-at "${BATS_TEST_TMPDIR}/code/b b" github.com/o/b
  workspaces-file <<EOF
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w: {repos: [github.com/o/a, github.com/o/b, github.com/o/c]}
EOF
  cd "${BATS_TEST_TMPDIR}"
  run "${kit_root}/bin/kit" workspace remotes --workspace w --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.member, .repository]]' <<<"$output")" = '[["github.com/o/a","o/a"],["github.com/o/b","o/b"]]' ]
  [ "$(jq -c .errors <<<"$output")" = '[]' ]
  run "${kit_root}/bin/kit" workspace branches --workspace w --text
  [ "$status" -eq 0 ]
  [[ "$output" == *$'📚 github.com/o/a\nbranch '* && "$output" == *$'📚 github.com/o/b\nbranch '* ]]
  run "${kit_root}/bin/kit" workspace --workspace nosuch --json
  [ "$status" -eq 2 ]
}

@test "--marks gives a mark per stage, and a stage's mark moves when what it draws does, and no other's" {
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run "${kit_root}/bin/kit" workspace --marks
  [ "$status" -eq 0 ]
  [ "$(jq -c 'keys' <<<"$output")" = '["branches","commits","remotes","repos","tags","worktrees"]' ]
  before=$output
  git tag v2
  run "${kit_root}/bin/kit" workspace --marks
  [ "$(jq -r .tags <<<"$output")" != "$(jq -r .tags <<<"${before}")" ]
  [ "$(jq -r .branches <<<"$output")" = "$(jq -r .branches <<<"${before}")" ]
}

@test "on a terminal, kit workspace hands off to kittui with its stage and workspace; a form stays in text" {
  fake-tui
  history-at "${BATS_TEST_TMPDIR}/r" github.com/o/r
  cd "${BATS_TEST_TMPDIR}/r"
  run pty "'${kit_root}/bin/kit' workspace branches"
  [[ "$output" == *"kittui native-native workspace branches"* ]]
  run pty "'${kit_root}/bin/kit' workspace --json"
  [[ "$output" != *"kittui"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
