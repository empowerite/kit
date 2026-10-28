#!/usr/bin/env bats
# status.bats - the parts of the picture that need no GitHub: a cell, a checkout's measures and block, its label,
# and its path from the current directory
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
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
  kit_invoked_in=$(pwd -P) bash -c 'source "${kit_root}/bash/1st.sh" && use status && "$@"' _ "$@"
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
  [ "$(jq --compact-output .changes <<<"$output")" = '[{"status":"??","path":"x"}]' ]
}

@test "a block shows the newest commits, counts the hidden ones, and ends with the oldest" {
  git worktree add --quiet -b topic worktrees/topic main
  commits-in worktrees/topic 15
  doc=$(in_status status:checkout "${PWD}/worktrees/topic" topic main)
  run in_status status:checkout-block 🌿 "${doc}" 15
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🌿 worktrees/topic · ↑15 ↓0 · +15 -0" ]
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
  [ "${lines[0]}" = "🏠 main · ↑1 ↓0 · +1 -0 · 1 uncommitted" ]
  [ "${lines[1]}" = "   ?? c" ]
  [[ "${lines[2]}" == *" two" ]]
  [ "${#lines[@]}" -eq 3 ]
}

@test "a block lists its uncommitted paths with git's status, the first ten and a count of the rest" {
  printf 'changed\n' > a
  for i in $(seq 1 12); do printf '%s\n' "${i}" > "n${i}"; done
  doc=$(in_status status:checkout "${PWD}" main main)
  run in_status status:checkout-block 🏠 "${doc}" 4
  [ "${lines[0]}" = "🏠 main · ↑0 ↓0 · +1 -1 · 13 uncommitted" ]
  [ "${lines[1]}" = "    M a" ]
  [ "${lines[2]}" = "   ?? n1" ]
  [ "${lines[11]}" = "   … and 3 more" ]
  [ "${#lines[@]}" -eq 12 ]
}

@test "the fetch's line names the commits new on the default branch and the branches and tags it moved" {
  git push --quiet origin main:old main:stays
  git fetch --quiet origin
  git clone --quiet ../origin.git ../other 2>/dev/null
  (
    cd ../other
    git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
    printf 'b\n' > b && git add b && git commit --quiet --message b
    printf 'c\n' > c && git add c && git commit --quiet --message c
    git push --quiet origin main main:fresh main:stays HEAD:refs/tags/v1
    git push --quiet origin --delete old
  )
  run in_status status:fetch main
  [ "$status" -eq 0 ]
  expected='📥 fetched origin: 2 new commits on main · 1 branch updated · 1 new branch · 1 branch pruned'
  [ "${lines[0]}" = "${expected} · 1 new tag" ]
  run in_status status:fetch main
  [ "${lines[0]}" = "📥 fetched origin: nothing new" ]
}

@test "kit status prints the fetch first, then the checkouts, then what the forge said, held until its place" {
  printf 'x\n' > c
  run "${kit_root}/bin/kit" status
  [ "${lines[0]}" = "📥 fetched origin: nothing new" ]
  [ "${lines[1]}" = "🏠 main · ↑0 ↓0 · +0 -0 · 1 uncommitted" ]
  [ "${lines[2]}" = "   ?? c" ]
  [[ "${lines[3]}" == *"forge"* ]]
}

@test "a detached checkout is marked and lists no commits" {
  git worktree add --quiet --detach worktrees/loose main
  doc=$(in_status status:checkout "${PWD}/worktrees/loose" '' main)
  run in_status status:checkout-block 🌿 "${doc}" 15
  [ "$output" = "🌿 worktrees/loose · ❗ detached HEAD · ↑0 ↓0 · +0 -0" ]
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

@test "a pull request's row carries its author when asked and ends with its branch, and the issues it closes follow" {
  doc='{"number":2,"title":"The first","author":"ada","branch":"2-the-first","state":"queued","age":259200,'
  doc+='"unresolved":0,"checks":{"state":"PENDING","passed":0,"total":1,"failing":0,"took":36},'
  doc+='"closes":[{"number":1,"title":"One"},{"number":4,"title":"Four"}]}'
  run in_status status:pr-block "${doc}" 12 0 6 2 worktrees/2-the-first
  [ "${lines[0]}" = "🔀 #2 The first · 🚦 queued 3d · 🏃 0 / 1 36s · 🌿 2-the-first" ]
  [ "${lines[1]}" = "   closes #1 One" ]
  [ "${lines[2]}" = "   closes #4 Four" ]
  [ "${#lines[@]}" -eq 3 ]
  run in_status status:pr-block "${doc}" 12 5 6 2 ''
  [ "${lines[0]}" = "🔀 #2 The first · 👤 ada   · 🚦 queued 3d · 🏃 0 / 1 36s" ]
}

@test "a failed checks cell names the failing count, then passed of total, then the time taken" {
  run in_status status:check-cell '{"state":"FAILURE","passed":3,"total":5,"failing":2,"took":31}'
  [ "$output" = "❌ 2 failing 3 / 5 31s" ]
}

@test "the pull requests shown are everyone's, or the named authors' alone" {
  docs=$(printf '{"number":1,"author":"ada"}\n{"number":2,"author":"bob"}\n{"number":3,"author":"cy"}\n')
  run in_status status:select <<<"${docs}"
  [ "$(jq --slurp --compact-output 'map(.number)' <<<"$output")" = "[1,2,3]" ]
  run in_status status:select ada cy <<<"${docs}"
  [ "$(jq --slurp 'map(.number) | join(",")' --raw-output <<<"$output")" = "1,3" ]
  run in_status status:select nobody <<<"${docs}"
  [ -z "$output" ]
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
