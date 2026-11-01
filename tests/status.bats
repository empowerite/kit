#!/usr/bin/env bats
# status.bats - kit status draws the picture: its stages, their headings and counts, one stage alone, the JSON, the
# hand-off to kittui on a terminal, and the cells every stage shares
# ----------------------------------------------------------------------------------------------------------------------
#
# origin is a bare repository beside the desk, so the fetch is real git. The forge, where a test needs one, is a fake
# gh: forge-world sets the caller ada, issues 7, 9, and 11, change 3 closing 7, and three releases, the world of
# backlog.bats and changes.bats.

bats_require_minimum_version 1.5.0

load helper

setup() {
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  printf 'one\n' > a && git add a && git commit --quiet --message one && git push --quiet origin main
  git remote set-head origin main
  mkdir worktrees
}

# in_status <command>… runs a command with the status module loaded, from this directory, as a verb would.
in_status() {
  kit_invoked_in=$(pwd -P) bash -c 'source "${kit_root}/modules/bash/1st.sh" && use status && "$@"' _ "$@"
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

# commit-settings <line>… writes the lines as the desk's kit.yml, ignores worktrees/, and commits and pushes both, so
# every checkout reads the settings and the desk stays even with origin.
commit-settings() {
  printf '%s\n' "$@" > kit.yml
  printf 'worktrees/\n' > .gitignore
  git add kit.yml .gitignore && git commit --quiet --message settings && git push --quiet origin main
}

# stages <list> sets the status setting to the stages named.
stages() {
  commit-settings "status: [$1]"
}

# forge-world puts a fake gh on PATH that answers for the forge: the caller ada; issue 7, assigned to bob and closed
# by change 3, a draft by ada on 7-fix-the-thing; issue 9, claimed by the branch 9-other on origin; issue 11; and
# three releases. The settings name the forge.
forge-world() {
  fake-tools gh
  commit-settings 'forge: github'
  git push --quiet origin main:7-fix-the-thing main:9-other
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture issues.json <<'JSON'
[{"number": 11, "title": "Third thing", "assignees": [{"login": "ada"}], "createdAt": "2026-09-30T09:00:00Z"},
 {"number": 9, "title": "Other thing", "assignees": [], "createdAt": "2026-09-25T09:00:00Z"},
 {"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  fixture open-prs.json <<'JSON'
{"data": {"repository": {"pullRequests": {"nodes": [{
  "number": 3, "title": "Fix the thing", "isDraft": true, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"},
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing", "createdAt": "2026-09-20T09:00:00Z",
                                         "assignees": {"nodes": [{"login": "bob"}]}}]},
  "reviewThreads": {"nodes": []},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}}]}}}}
JSON
  fixture releases.json <<'JSON'
[{"tagName": "release/0.1.0-rc.6", "name": "kit 0.1.0-rc.6", "isPrerelease": true, "isDraft": false, "isLatest": false,
  "publishedAt": "2026-10-27T07:30:14Z", "createdAt": "2026-10-27T07:30:14Z"},
 {"tagName": "release/0.0.9", "name": "kit 0.0.9", "isPrerelease": false, "isDraft": false, "isLatest": true,
  "publishedAt": "2026-10-20T07:30:14Z", "createdAt": "2026-10-20T07:30:14Z"},
 {"tagName": "release/0.1.0-rc.7", "name": "kit 0.1.0-rc.7", "isPrerelease": true, "isDraft": true, "isLatest": false,
  "publishedAt": null, "createdAt": "2026-10-30T07:30:14Z"}]
JSON
  route 'api user*' user.json
  route 'repo view*' repo.json
  route 'issue list --state open --limit 1000 --json *' issues.json
  route 'api graphql*' open-prs.json
  route 'release list --limit 1000 --json *' releases.json
}

# empty-forge makes the fake forge answer with no issue, no change, and no release.
empty-forge() {
  fixture issues.json <<<'[]'
  fixture open-prs.json <<<'{"data": {"repository": {"pullRequests": {"nodes": []}}}}'
  fixture releases.json <<<'[]'
}

# pty <command> runs the command in a pseudo-terminal, so its stdin, stdout, and stderr are terminals, as at a desk.
pty() {
  if script --version >/dev/null 2>&1
  then
    script --quiet --return --command "$1" /dev/null </dev/null
  else
    script -q /dev/null bash -c "$1" </dev/null
  fi
}

# fake-tui puts the fake go and the fake kittui first on PATH, as tui.bats does: kit builds kittui from tui/ with the
# fake go, which writes a script that prints "kittui native-native" and its arguments.
fake-tui() {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
}

# without-kittui prints a PATH with the tools kit status needs and neither go nor kittui.
without-kittui() {
  local dir="${BATS_TEST_TMPDIR}/nokittui" tool
  mkdir -p "${dir}"
  for tool in bash git jq yq
  do
    ln -sf "$(command -v "${tool}")" "${dir}/${tool}"
  done
  printf '%s:/usr/bin:/bin\n' "${dir}"
}

@test "a cell is padded to its width, or clipped with an ellipsis" {
  run in_status status:cell abc 5
  [ "$output" = "abc  " ]
  run in_status status:cell abcdefgh 5
  [ "$output" = "abcd…" ]
}

@test "a cell is right-aligned on request, for a number" {
  run in_status status:cell 3d 4 right
  [ "$output" = "  3d" ]
  run in_status status:cell abcdefgh 5 right
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
  for i in $(seq 1 12)
  do
    printf '%s\n' "${i}" > "n${i}"
  done
  doc=$(in_status status:checkout "${PWD}" main main)
  run in_status status:checkout-block 🏠 "${doc}" 4
  [ "${lines[0]}" = "🏠 main · ↑0 ↓0 · +1 -1 · 13 uncommitted" ]
  [ "${lines[1]}" = "    M a" ]
  [ "${lines[2]}" = "   ?? n1" ]
  [ "${lines[11]}" = "   … and 3 more" ]
  [ "${#lines[@]}" -eq 12 ]
}

@test "kit status incoming prints the fetch's line, and --json what the fetch brought: commits, branches new and gone" {
  git push --quiet origin main:old main:stays
  git fetch --quiet origin
  git clone --quiet ../origin.git ../other 2>/dev/null
  (
    cd ../other
    printf 'b\n' > b && git add b && git commit --quiet --message b
    printf 'c\n' > c && git add c && git commit --quiet --message c
    git push --quiet origin main main:fresh main:stays HEAD:refs/tags/v1
    git push --quiet origin --delete old
  )
  cp -R . ../twin
  run "${kit_root}/bin/kit" status incoming --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.fetched, .default, (.commits | map(.subject)), .branches, .tags]' <<<"$output")" \
    = '[true,"main",["c","b"],{"new":["fresh"],"updated":["stays"],"gone":["old"]},["v1"]]' ]
  [ "$(jq -r '.commits[0].sha | length' <<<"$output")" -ge 7 ]
  run "${kit_root}/bin/kit" status incoming --json
  [ "$(jq -c '[.fetched, .commits, .branches.new]' <<<"$output")" = '[true,[],[]]' ]
  cd ../twin
  run "${kit_root}/bin/kit" status incoming
  [ "$status" -eq 0 ]
  expected='📥 origin: fetched 2 new commits on main · 1 branch updated · 1 new branch · 1 branch pruned'
  [ "$output" = "${expected} · 1 new tag" ]
  run "${kit_root}/bin/kit" status incoming
  [ "$output" = "📥 origin: unchanged" ]
  run "${kit_root}/bin/kit" status incoming --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"fetched":false}' ]
  run "${kit_root}/bin/kit" status incoming --no-fetch
  [ "$output" = "📥 origin: not fetched" ]
}

@test "kit status --stages prints the configured stages in order, every stage without the setting, and refuses a name that is none" {
  run "${kit_root}/bin/kit" status --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'backlog\nincoming\nworktrees\nchanges\nreleases' ]
  stages 'releases, worktrees'
  run "${kit_root}/bin/kit" status --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'releases\nworktrees' ]
  printf 'status: [worktrees, nope]\n' > kit.yml
  run "${kit_root}/bin/kit" status --stages
  [ "$status" -eq 2 ]
  [[ "$output" == *"'nope' is not a stage; the stages are: backlog incoming worktrees changes releases (status in kit.yml)"* ]]
  run "${kit_root}/bin/kit" status --no-fetch
  [ "$status" -eq 2 ]
  [[ "$output" == *"'nope' is not a stage"* ]]
}

@test "kit status prints each configured stage in order under a dim heading with its count, a blank line between stages" {
  local esc=$'\x1b'
  stages 'worktrees, incoming'
  git worktree add --quiet -b 7-topic worktrees/7-topic main
  printf 'x\n' > c
  run --keep-empty-lines "${kit_root}/bin/kit" status
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "WORKTREES 2" ]
  [ "${lines[1]}" = "🏠 main              · ↑0 ↓0 · +0 -0 · 1 uncommitted" ]
  [ "${lines[2]}" = "   ?? c" ]
  [ "${lines[3]}" = "" ]
  [ "${lines[4]}" = "🌿 worktrees/7-topic · ↑0 ↓0 · +0 -0" ]
  [ "${lines[5]}" = "" ]
  [ "${lines[6]}" = "INCOMING" ]
  [ "${lines[7]}" = "📥 origin: unchanged" ]
  [ "${#lines[@]}" -eq 8 ]
  # the heading is dim only when effects are on; --text prints the same picture
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" status --text --no-fetch
  [[ "${lines[0]}" == "${esc}[90mWORKTREES 2${esc}[0m" ]]
  [[ "$output" == *"${esc}[90mINCOMING${esc}[0m"$'\n'"${esc}[90m📥 origin: not fetched${esc}[0m"* ]]
  run --keep-empty-lines "${kit_root}/bin/kit" status --text --no-fetch
  [[ "$output" != *"${esc}["* ]]
  [ "${lines[6]}" = "INCOMING" ]
  [ "${lines[7]}" = "📥 origin: not fetched" ]
}

@test "kit status --json is one object: stages in the configured order, and under each name what the stage alone prints" {
  stages 'incoming, worktrees'
  git worktree add --quiet -b 7-topic worktrees/7-topic main
  run "${kit_root}/bin/kit" status --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'keys_unsorted' <<<"$output")" = '["stages","incoming","worktrees"]' ]
  [ "$(jq -c '.stages' <<<"$output")" = '["incoming","worktrees"]' ]
  [ "$(jq -c '[.incoming.fetched, .worktrees.desk.branch, [.worktrees.worktrees[].branch], .worktrees.goneBranches]' <<<"$output")" \
    = '[true,"main",["7-topic"],[]]' ]
  [ "$(jq -c .worktrees <<<"$output")" = "$("${kit_root}/bin/kit" status worktrees --json)" ]
  run "${kit_root}/bin/kit" status --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.incoming' <<<"$output")" = '{"fetched":false}' ]
  run "${kit_root}/bin/kit" status --jq
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "{" ]
}

@test "with a forge, the picture counts the backlog, the changes, and the releases, and the blank lines separate the stages" {
  forge-world
  run --keep-empty-lines "${kit_root}/bin/kit" status --no-fetch
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "BACKLOG 2" ]
  [[ "${lines[1]}" == " #9        "*"d  🌿 9-other  🎫 Other thing" ]]
  [[ "${lines[2]}" == "#11  👤 ada  "*"d              🎫 Third thing" ]]
  [ "${lines[3]}" = "" ]
  [ "${lines[4]}" = "INCOMING" ]
  [ "${lines[5]}" = "📥 origin: not fetched" ]
  [ "${lines[6]}" = "" ]
  [ "${lines[7]}" = "WORKTREES 1" ]
  [ "${lines[8]}" = "🏠 main · ↑0 ↓0 · +0 -0" ]
  [ "${lines[9]}" = "" ]
  [ "${lines[10]}" = "CHANGES 1" ]
  [[ "${lines[11]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  [[ "${lines[12]}" == "   #7  👤 bob  🛫 in flight "*"d  🎫 Fix the thing" ]]
  [ "${lines[13]}" = "" ]
  [ "${lines[14]}" = "RELEASES 3" ]
  [[ "${lines[15]}" == "0.1.0-rc.7  📝 draft "* ]]
  [[ "${lines[17]}" == "0.0.9       📦 release "*"  ⭐ latest" ]]
  [ "${#lines[@]}" -eq 18 ]
  run "${kit_root}/bin/kit" status --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.stages' <<<"$output")" = '["backlog","incoming","worktrees","changes","releases"]' ]
  [ "$(jq -c '[(.backlog | map(.number)), (.changes | map(.number)), (.releases | map(.version))]' <<<"$output")" \
    = '[[9,11],[3],["0.1.0-rc.7","0.1.0-rc.6","0.0.9"]]' ]
  # the ages count seconds from now, so they are left out of the comparison
  [ "$(jq -c '.changes | del(.. | .age?)' <<<"$output")" \
    = "$("${kit_root}/bin/kit" status changes --json | jq -c 'del(.. | .age?)')" ]
  # the filters reach the backlog and the changes
  run "${kit_root}/bin/kit" status --no-fetch --mine --json
  [ "$(jq -c '[(.backlog | map(.number)), (.changes | map(.number))]' <<<"$output")" = '[[11],[3]]' ]
}

@test "an empty stage says so in one dim line, alone or in the picture, and prints [] with --json" {
  local esc=$'\x1b'
  forge-world
  empty-forge
  run --keep-empty-lines "${kit_root}/bin/kit" status --no-fetch
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "BACKLOG 0" ]
  [ "${lines[1]}" = "no open issue without a pull request" ]
  [ "${lines[9]}" = "CHANGES 0" ]
  [ "${lines[10]}" = "no open pull request" ]
  [ "${lines[12]}" = "RELEASES 0" ]
  [ "${lines[13]}" = "no release yet" ]
  [ "${#lines[@]}" -eq 14 ]
  run "${kit_root}/bin/kit" status backlog
  [ "$status" -eq 0 ]
  [ "$output" = "no open issue without a pull request" ]
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" status backlog
  [ "$output" = "${esc}[90mno open issue without a pull request${esc}[0m" ]
  run "${kit_root}/bin/kit" status releases
  [ "$output" = "no release yet" ]
  run "${kit_root}/bin/kit" status changes --json
  [ "$status" -eq 0 ]
  [ "$output" = "[]" ]
  run "${kit_root}/bin/kit" status releases --json
  [ "$output" = "[]" ]
}

@test "a stage the forge cannot answer is drawn as its refusal under its heading, and the picture exits 1" {
  run "${kit_root}/bin/kit" status --no-fetch
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = "BACKLOG" ]
  [[ "${lines[1]}" == *"set forge in kit.yml"* ]]
  [[ "$output" == *$'\nWORKTREES 1\n🏠 main'* ]]
  run "${kit_root}/bin/kit" status --no-fetch --json
  [ "$status" -eq 1 ]
  [[ "${lines[0]}" == *"set forge in kit.yml"* ]]
  [ "$(jq -c '[.backlog, .changes, .releases, .worktrees.desk.branch]' <<<"${lines[${#lines[@]} - 1]}")" \
    = '[null,null,null,"main"]' ]
  run "${kit_root}/bin/kit" status changes
  [ "$status" -eq 1 ]
  [[ "$output" == *"set forge in kit.yml"* ]]
}

@test "a repository with no origin draws its checkouts, says so once, and exits 0; a forge stage alone still refuses" {
  fresh-repo alone >/dev/null
  printf 'one\n' > a && git add a && git commit --quiet --message one
  run --keep-empty-lines "${kit_root}/bin/kit" status
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == *"no origin: nothing fetched, no pull request shown" ]]
  [ "${lines[1]}" = "" ]
  [ "${lines[2]}" = "WORKTREES 1" ]
  [ "${lines[3]}" = "🏠 main · ↑0 ↓0 · +0 -0" ]
  [ "${#lines[@]}" -eq 4 ]
  run "${kit_root}/bin/kit" status --json
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == *"no origin: nothing fetched, no pull request shown" ]]
  [ "$(jq -c '[.backlog, .incoming, .changes, .releases, .worktrees.desk.branch]' <<<"${lines[1]}")" \
    = '[null,{"fetched":false},null,null,"main"]' ]
  run "${kit_root}/bin/kit" status changes
  [ "$status" -eq 1 ]
  [[ "$output" == *"origin's host 'none' names no forge"* ]]
}

@test "on a terminal, kit status hands off to kittui with its arguments; a stage, --text, --json, or no terminal stays in text" {
  fake-tui
  stages 'worktrees'
  run pty "'${kit_root}/bin/kit' status --no-fetch --mine"
  [[ "$output" == *"building kittui from tui/"* ]]
  [[ "$output" == *"kittui native-native status --no-fetch --mine"* ]]
  [[ "$output" != *"WORKTREES"* ]]
  run pty "'${kit_root}/bin/kit' status --text --no-fetch"
  [[ "$output" == *"WORKTREES 1"* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' status --json --no-fetch"
  [[ "$output" == *'"stages":["worktrees"]'* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' status worktrees"
  [[ "$output" == *"🏠 main"* && "$output" != *"kittui"* && "$output" != *"WORKTREES"* ]]
  run "${kit_root}/bin/kit" status --no-fetch </dev/null
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "WORKTREES 1" ]
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' status --bogus"
  [[ "$output" == *"unknown flag '--bogus'; run: kit help status"* && "$output" != *"kittui"* ]]
}

@test "on a terminal with no kittui to be had, kit status prints the text picture and says nothing of kittui" {
  stages 'worktrees'
  run pty "env PATH='$(without-kittui)' '${kit_root}/bin/kit' status --no-fetch"
  [[ "$output" == *"WORKTREES 1"* ]]
  [[ "$output" != *"kittui"* && "$output" != *"mise install"* ]]
}

@test "kittui runs the kit that handed off, named in KIT" {
  fake-tui
  export FAKE_KITTUI_PRINT_KIT=1
  stages 'worktrees'
  run pty "'${kit_root}/bin/kit' status --no-fetch"
  [[ "$output" == *"kittui native-native status --no-fetch"* ]]
}

@test "in a worktree, worktrees draws that worktree alone and changes its pull request; at the desk, every linked worktree" {
  forge-world
  git worktree add --quiet -b 7-fix-the-thing worktrees/7-fix-the-thing origin/7-fix-the-thing
  git worktree add --quiet -b elsewhere ../elsewhere main
  cd worktrees/7-fix-the-thing
  run "${kit_root}/bin/kit" status worktrees
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🌿 7-fix-the-thing · ↑0 ↓0 · +0 -0" ]
  [ "${#lines[@]}" -eq 1 ]
  run "${kit_root}/bin/kit" status worktrees --json
  [ "$(jq -c '[.desk, [.worktrees[].branch]]' <<<"$output")" = '[null,["7-fix-the-thing"]]' ]
  run "${kit_root}/bin/kit" status changes --others
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  run "${kit_root}/bin/kit" status --no-fetch
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\nWORKTREES 1\n🌿 7-fix-the-thing · '* ]]
  [[ "$output" == *$'\nCHANGES 1\n🔀 #3 Fix the thing'* ]]
  cd ../../../elsewhere
  run "${kit_root}/bin/kit" status changes
  [ "$output" = "no open pull request" ]
  cd ../desk
  run "${kit_root}/bin/kit" status worktrees --json
  [ "$(jq -c '[.desk.branch, [.worktrees[].branch]]' <<<"$output")" = '["main",["elsewhere","7-fix-the-thing"]]' ]
  run "${kit_root}/bin/kit" status worktrees
  [[ "${lines[0]}" == "🏠 main "* ]]
  [[ "$output" == *$'\n\n🌿 '*"elsewhere"* && "$output" == *$'\n\n🌿 worktrees/7-fix-the-thing'* ]]
}

@test "the worktrees stage ends with the branches whose upstream is gone, in text and as goneBranches" {
  git push --quiet origin main:gone-soon
  git fetch --quiet origin
  git branch --quiet --track gone-soon origin/gone-soon
  git push --quiet origin --delete gone-soon
  git fetch --quiet --prune origin
  run --keep-empty-lines "${kit_root}/bin/kit" status worktrees
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🏠 main · ↑0 ↓0 · +0 -0" ]
  [ "${lines[1]}" = "" ]
  [ "${lines[2]}" = "🔖 gone-soon · upstream gone · kit tidy removes it" ]
  run "${kit_root}/bin/kit" status worktrees --json
  [ "$(jq -c .goneBranches <<<"$output")" = '["gone-soon"]' ]
}

@test "above desks, kit status finds each desk below, stops at it, skips hidden directories, and draws each under its path" {
  stages 'worktrees'
  cd "${work}"
  mkdir -p group/.hidden group/deep
  git init --quiet --initial-branch=main group/.hidden/secret
  git clone --quiet origin.git group/deep/two 2>/dev/null
  git -C group/deep/two remote set-head origin main
  git init --quiet --initial-branch=main desk/nested
  run "${kit_root}/bin/kit" status --no-fetch
  [[ "$output" == *"📂 desk"$'\n'"WORKTREES 1"$'\n'"🏠 main"* ]]
  [[ "$output" == *"📂 group/deep/two"$'\n'"WORKTREES 1"$'\n'"🏠 main"* ]]
  [[ "$output" != *"📂 group/.hidden"* && "$output" != *"📂 desk/nested"* ]]
  output=$("${kit_root}/bin/kit" status --no-fetch --json 2>/dev/null)
  [ "$(jq -c 'map(.path)' <<<"$output")" = '["desk","group/deep/two"]' ]
  [ "$(jq -r '.[0].status.worktrees.desk.branch' <<<"$output")" = main ]
  run "${kit_root}/bin/kit" status worktrees
  [[ "$output" == *"📂 desk"$'\n'"🏠 main"* ]]
  mkdir "${BATS_TEST_TMPDIR}/nowhere" && cd "${BATS_TEST_TMPDIR}/nowhere"
  run "${kit_root}/bin/kit" status --no-fetch
  [ "$status" -eq 1 ]
  [[ "$output" == *"no repository at or below"* ]]
}

@test "a detached checkout is marked and lists no commits" {
  git worktree add --quiet --detach worktrees/loose main
  doc=$(in_status status:checkout "${PWD}/worktrees/loose" '' main)
  run in_status status:checkout-block 🌿 "${doc}" 15
  [ "$output" = "🌿 worktrees/loose  ❗ detached HEAD · ↑0 ↓0 · +0 -0" ]
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

@test "a pull request's row carries its author when asked and ends with its branch, and the issues it closes follow as rows" {
  doc='{"number":2,"title":"The first","author":"ada","branch":"2-the-first","state":"queued","age":259200,'
  doc+='"unresolved":0,"checks":{"state":"PENDING","passed":0,"total":1,"failing":0,"took":36},'
  doc+='"closes":[{"number":1,"title":"One","assignees":["ada"],"age":172800},{"number":4,"title":"Four","assignees":[],"age":7200}]}'
  widths=$(in_status status:issue-widths "$(jq --compact-output .closes <<<"${doc}")")
  [ "${widths}" = "2 3 2 0" ]
  run in_status status:pr-block "${doc}" 12 0 6 2 worktrees/2-the-first "${widths}"
  [ "${lines[0]}" = "🔀 #2 The first  🚦 queued 3d  🏃 0 / 1 36s  🌿 2-the-first" ]
  [ "${lines[1]}" = "   #1  👤 ada  🛫 in flight 2d  🎫 One" ]
  [ "${lines[2]}" = "   #4          🛫 in flight 2h  🎫 Four" ]
  [ "${#lines[@]}" -eq 3 ]
  run in_status status:pr-block "${doc}" 12 5 6 2 '' "${widths}"
  [ "${lines[0]}" = "🔀 #2 The first  👤 ada    🚦 queued 3d  🏃 0 / 1 36s" ]
}

@test "an issue row without assignees or age, as an older forge answer gives it, keeps its number and title" {
  doc='{"number":2,"title":"The first","author":"ada","branch":"2-the-first","state":"queued","age":60,'
  doc+='"unresolved":0,"checks":{"state":"NONE"},"closes":[{"number":1,"title":"One"}]}'
  run in_status status:pr-block "${doc}" 12 0 6 2 '' "$(in_status status:issue-widths '[{"number":1,"title":"One"}]')"
  [ "${lines[1]}" = "   #1  🛫 in flight  🎫 One" ]
}

@test "a change's number is magenta and the number of an issue it closes cyan, as the backlog draws them" {
  local esc=$'\x1b'
  doc='{"number":2,"title":"The first","author":"ada","branch":"2-the-first","state":"queued","age":60,'
  doc+='"unresolved":0,"checks":{"state":"NONE"},"closes":[{"number":1,"title":"One","assignees":[],"age":120}]}'
  run env -u NO_COLOR CLICOLOR_FORCE=1 \
    bash -c 'source "${kit_root}/modules/bash/1st.sh" && use status && status:pr-block "$@"' _ "${doc}" 12 0 6 2 '' '2 0 2 0'
  [[ "${lines[0]}" == "🔀 ${esc}[35m#2${esc}[0m The first "* ]]
  [[ "${lines[1]}" == "   ${esc}[36m#1${esc}[0m  🛫 ${esc}[34min flight${esc}[0m ${esc}[90m2m${esc}[0m  🎫 One" ]]
}

@test "a failed checks cell names the failing count, then passed of total, then the time taken" {
  run in_status status:check-cell '{"state":"FAILURE","passed":3,"total":5,"failing":2,"took":31}'
  [ "$output" = "❌ 2 failing 3 / 5 31s" ]
}

@test "the pull requests shown are everyone's, the named authors' alone, or everyone's but theirs" {
  docs=$(printf '{"number":1,"author":"ada"}\n{"number":2,"author":"bob"}\n{"number":3,"author":"cy"}\n')
  run in_status status:select <<<"${docs}"
  [ "$(jq --slurp --compact-output 'map(.number)' <<<"$output")" = "[1,2,3]" ]
  run in_status status:select ada cy <<<"${docs}"
  [ "$(jq --slurp 'map(.number) | join(",")' --raw-output <<<"$output")" = "1,3" ]
  run in_status status:select nobody <<<"${docs}"
  [ -z "$output" ]
  run in_status status:select --except ada <<<"${docs}"
  [ "$(jq --slurp 'map(.number) | join(",")' --raw-output <<<"$output")" = "2,3" ]
  run in_status status:select --except nobody <<<"${docs}"
  [ "$(jq --slurp 'map(.number) | join(",")' --raw-output <<<"$output")" = "1,2,3" ]
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

@test "status:filter turns the owner flags into status:select's arguments, and kit status refuses --all beside --mine" {
  run in_status status:filter false false bob cy
  [ "$status" -eq 0 ]
  [ "$output" = "bob cy" ]
  run in_status status:filter false false
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_root}/bin/kit" status --all --mine
  [ "$status" -eq 2 ]
  [[ "$output" == *"--all cannot be combined with --mine; run: kit help status"* ]]
  run "${kit_root}/bin/kit" status --today
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" status incoming --mine
  [ "$status" -eq 2 ]
  [[ "$output" == *"--mine cannot be combined with incoming; run: kit help status"* ]]
}

@test "kit syntax status lists the five stages as a closed set, each described, and the removed verbs are no verbs" {
  run "${kit_root}/bin/kit" syntax status --choices '<stage>'
  [ "$status" -eq 0 ]
  [ "$(cut -f 1 <<<"$output" | paste -s -d ' ' -)" = "backlog incoming worktrees changes releases" ]
  [[ "${lines[0]}" == $'backlog\tthe open issues no pull request closes, oldest first\t\t' ]]
  local verb
  for verb in backlog changes incoming worktrees releases
  do
    run "${kit_root}/bin/kit" "${verb}"
    [ "$status" -eq 2 ]
    [[ "$output" == *"${verb}"* ]]
    [ ! -e "${kit_root}/bin/kit-${verb}" ]
  done
}

# ----------------------------------------------------------------------------------------------------------------------
