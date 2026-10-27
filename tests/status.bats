#!/usr/bin/env bats
# status.bats - the parts of the picture that need no GitHub: a cell, a checkout's measures and block, its label,
# and its path from the current directory
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
  mkdir worktrees
}

teardown() {
  rm -rf "${work}"
}

# in_status <command>… runs a command with the status module loaded, from this directory, as a verb would.
in_status() {
  kit_invoked_in=$(pwd -P) bash -c 'source "${kit_root}/bin/1st.sh" && use status && "$@"' _ "$@"
}

# commits-in <path> <n> makes <n> commits in the checkout at <path>.
commits-in() {
  local i
  for (( i = 1; i <= $2; i++ ))
  do
    printf '%s\n' "${i}" > "$1/f${i}"
    git -C "$1" add . && git -C "$1" commit --quiet --message "c${i}"
  done
}

@test "a cell is padded to its width, or clipped with an ellipsis" {
  run in_status status:cell abc 5
  [ "$output" = "abc  " ]
  run in_status status:cell abcdefgh 5
  [ "$output" = "abcd…" ]
}

@test "a checkout is measured from where its branch left origin's default branch" {
  git worktree add --quiet -b topic worktrees/topic main
  printf 'two\nthree\n' > worktrees/topic/b
  git -C worktrees/topic add b && git -C worktrees/topic commit --quiet --message two
  printf 'four\n' > c && git add c && git commit --quiet --message four && git push --quiet origin main
  printf 'x\n' > worktrees/topic/x
  run in_status status:checkout "${PWD}/worktrees/topic" topic main
  [ "$status" -eq 0 ]
  fields='[.relative, .branch, .ahead, .behind, .added, .removed, .uncommitted, (.commits | length)] | @tsv'
  [ "$(jq -r "${fields}" <<<"$output")" = "$(printf 'worktrees/topic\ttopic\t1\t1\t2\t0\t1\t1')" ]
}

@test "a block shows the newest commits, counts the hidden ones, and ends with the oldest" {
  git worktree add --quiet -b topic worktrees/topic main
  commits-in worktrees/topic 15
  doc=$(in_status status:checkout "${PWD}/worktrees/topic" topic main)
  run in_status status:checkout-block 🌿 "${doc}" 15
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🌿 worktrees/topic   ↑15 ↓0 · +15 -0" ]
  [ "${#lines[@]}" -eq 12 ]
  [[ "${lines[1]}" == *" c15" ]]
  [[ "${lines[9]}" == *" c7" ]]
  [[ "${lines[10]}" == *"… and 5 more" ]]
  [[ "${lines[11]}" == *" c1" ]]
}

@test "the desk block is its branch, its unpushed commits, and its uncommitted paths" {
  printf 'two\n' > b && git add b && git commit --quiet --message two
  printf 'x\n' > c
  doc=$(in_status status:checkout "${PWD}" main main)
  run in_status status:checkout-block 🏠 "${doc}" 4
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🏠 main   ↑1 ↓0 · +1 -0 · 1 uncommitted" ]
  [[ "${lines[1]}" == *" two" ]]
}

@test "a detached checkout is marked and lists no commits" {
  git worktree add --quiet --detach worktrees/loose main
  doc=$(in_status status:checkout "${PWD}/worktrees/loose" '' main)
  run in_status status:checkout-block 🌿 "${doc}" 15
  [ "$output" = "🌿 worktrees/loose   ❗ detached HEAD   ↑0 ↓0 · +0 -0" ]
}

@test "the counts of every checkout sit in columns, ahead and added right-aligned, behind and removed left-aligned" {
  one=$(jq -nc '{ahead: 1, behind: 22, added: 3, removed: 4444, uncommitted: 0}')
  two=$(jq -nc '{ahead: 100, behind: 2, added: 30000, removed: 4, uncommitted: 2}')
  widths=$(in_status status:count-widths "${one}" "${two}")
  [ "${widths}" = "4 3 6 5" ]
  run in_status status:measures "${one}" "${widths}"
  [ "$output" = "  ↑1 ↓22 ·     +3 -4444" ]
  run in_status status:measures "${two}" "${widths}"
  [ "$output" = "↑100 ↓2  · +30000 -4    · 2 uncommitted" ]
}

@test "a label is the path when the branch ends it, the branch at the current directory, or branch on path" {
  run in_status status:label worktrees/topic topic
  [ "$output" = "$(printf 'worktrees/topic\t15')" ]
  run in_status status:label . main
  [ "$output" = "$(printf 'main\t4')" ]
  run in_status status:label ../other topic
  [ "$output" = "$(printf 'topic on ../other\t17')" ]
}

@test "a pull request's row ends with its worktree, and the issues it closes follow" {
  doc='{"number":2,"title":"The first","state":"queued","age":259200,"unresolved":0,'
  doc+='"checks":{"state":"PENDING","passed":0,"total":1,"failing":0,"took":36},'
  doc+='"closes":[{"number":1,"title":"One"},{"number":4,"title":"Four"}]}'
  run in_status status:pr-block "${doc}" 12 6 2 worktrees/topic
  [ "${lines[0]}" = "🔀 #2 The first   🚦 queued   3d   🏃 0 / 1 · 36s   🌿 worktrees/topic" ]
  [ "${lines[1]}" = "   closes #1 One" ]
  [ "${lines[2]}" = "   closes #4 Four" ]
  [ "${#lines[@]}" -eq 3 ]
  run in_status status:pr-block "${doc}" 12 6 2 ''
  [ "${lines[0]}" = "🔀 #2 The first   🚦 queued   3d   🏃 0 / 1 · 36s" ]
}

@test "a path is printed relative to the current directory" {
  run in_status status:relative "${PWD}"
  [ "$output" = "." ]
  run in_status status:relative "${PWD}/worktrees/topic"
  [ "$output" = "worktrees/topic" ]
  cd worktrees
  run in_status status:relative "${PWD%/worktrees}"
  [ "$output" = ".." ]
}
