#!/usr/bin/env bats
# flow.bats - kit flow draws the picture: its stages, their headings and counts, one stage alone, the JSON, the
# hand-off to kittui on a terminal, and the cells every stage shares
# ----------------------------------------------------------------------------------------------------------------------
# origin is a bare repository beside the desk, so the fetch is real git. The forge, where a test needs one, is a fake
# gh: forge-world sets the caller ada, issues 7, 9, and 11, change 3 closing 7, two runs, and three releases, the world
# of backlog.bats and changes.bats.

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

# in_flow <command>… runs a command with the flow module loaded, from this directory, as a verb would.
in_flow() {
  kit_invoked_in=$(pwd -P) bash -c 'source "${kit_root}/modules/bash/1st.sh" && use flow && "$@"' _ "$@"
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

# stages <list> sets the flow setting to the stages named.
stages() {
  commit-settings "flow: [$1]"
}

# forge-world puts a fake gh on PATH that answers for the forge: the caller ada; issue 7, assigned to bob and closed
# by change 3, a draft by ada on 7-fix-the-thing; issue 9, claimed by the branch 9-other on origin; issue 11; a run
# on main and one for change 3; and three releases. The settings name the forge.
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
  fixture runs.json <<'JSON'
[{"databaseId": 12, "workflowName": "kit", "headBranch": "7-fix-the-thing", "event": "pull_request",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T11:00:00Z",
  "startedAt": "2026-10-01T11:00:00Z", "updatedAt": "2026-10-01T11:03:00Z", "url": "https://github.com/r/12"},
 {"databaseId": 11, "workflowName": "kit", "headBranch": "main", "event": "push",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T10:00:00Z",
  "startedAt": "2026-10-01T10:00:00Z", "updatedAt": "2026-10-01T10:03:36Z", "url": "https://github.com/r/11"}]
JSON
  route 'release list --limit 1000 --json *' releases.json
  route 'run list --limit 50 --json *' runs.json
}

# empty-forge makes the fake forge answer with no issue, no change, no run, and no release.
empty-forge() {
  fixture issues.json <<<'[]'
  fixture open-prs.json <<<'{"data": {"repository": {"pullRequests": {"nodes": []}}}}'
  fixture releases.json <<<'[]'
  fixture runs.json <<<'[]'
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

# fake-tui puts the fake go and the fake kittui first on PATH, as tui.bats does: kit builds kittui from projects/tui/
# with the fake go, which writes a script that prints "kittui native-native" and its arguments.
fake-tui() {
  mkdir -p "${BATS_TEST_TMPDIR}/fake"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-kittui" "${BATS_TEST_TMPDIR}/fake/kittui"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-go" "${BATS_TEST_TMPDIR}/fake/go"
  export FAKES="${BATS_TEST_TMPDIR}/fake" PATH="${BATS_TEST_TMPDIR}/fake:${PATH}"
}

@test "a cell is padded to its width, or clipped with an ellipsis" {
  run in_flow flow:cell abc 5
  [ "$output" = "abc  " ]
  run in_flow flow:cell abcdefgh 5
  [ "$output" = "abcd…" ]
}

@test "a cell is right-aligned on request, for a number" {
  run in_flow flow:cell 3d 4 right
  [ "$output" = "  3d" ]
  run in_flow flow:cell abcdefgh 5 right
  [ "$output" = "abcd…" ]
}

@test "a checkout is measured from where its branch left origin's default branch" {
  git worktree add --quiet -b topic worktrees/topic main
  printf 'two\nthree\n' > worktrees/topic/b
  git -C worktrees/topic add b && git -C worktrees/topic commit --quiet --message two
  printf 'four\n' > c && git add c && git commit --quiet --message four && git push --quiet origin main
  printf 'x\n' > worktrees/topic/x
  run in_flow flow:checkout "${PWD}/worktrees/topic" topic main
  [ "$status" -eq 0 ]
  fields='[.relative, .branch, .ahead, .behind, .added, .removed, .uncommitted, (.commits | length)] | @tsv'
  [ "$(jq -r "${fields}" <<<"$output")" = "$(printf 'worktrees/topic\ttopic\t1\t1\t2\t0\t1\t1')" ]
  [ "$(jq --compact-output .changes <<<"$output")" = '[{"status":"??","path":"x"}]' ]
}

@test "a block shows the newest commits, counts the hidden ones, and ends with the oldest" {
  git worktree add --quiet -b topic worktrees/topic main
  commits-in worktrees/topic 15
  doc=$(in_flow flow:checkout "${PWD}/worktrees/topic" topic main)
  run in_flow flow:checkout-block 🌿 "${doc}" 15
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
  doc=$(in_flow flow:checkout "${PWD}" main main)
  run in_flow flow:checkout-block 💻 "${doc}" 4
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "💻 main · ↑1 ↓0 · +1 -0 · 1 uncommitted" ]
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
  doc=$(in_flow flow:checkout "${PWD}" main main)
  run in_flow flow:checkout-block 💻 "${doc}" 4
  [ "${lines[0]}" = "💻 main · ↑0 ↓0 · +1 -1 · 13 uncommitted" ]
  [ "${lines[1]}" = "    M a" ]
  [ "${lines[2]}" = "   ?? n1" ]
  [ "${lines[11]}" = "   … and 3 more" ]
  [ "${#lines[@]}" -eq 12 ]
}

@test "kit flow incoming prints the fetch's line, and --json what the fetch brought: commits, branches new and gone" {
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
  run "${kit_root}/bin/kit" flow incoming --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.fetched, .primary, (.remotes | length)]' <<<"$output")" = '[true,"origin",1]' ]
  [ "$(jq -c '.remotes[0] | [.name, .forge, .host, .default, (.commits | map(.subject)), .branches, .tags]' \
    <<<"$output")" = '["origin",null,null,"main",["c","b"],{"new":["fresh"],"updated":["stays"],"gone":["old"]},["v1"]]' ]
  [ "$(jq -r '.remotes[0].commits[0].sha | length' <<<"$output")" -ge 7 ]
  run "${kit_root}/bin/kit" flow incoming --json
  [ "$(jq -c '[.fetched, .remotes[0].commits, .remotes[0].branches.new]' <<<"$output")" = '[true,[],[]]' ]
  cd ../twin
  run "${kit_root}/bin/kit" flow incoming
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "📡 origin · no forge · ${work}/origin.git" ]
  expected='📥 fetched 2 new commits on main · 1 branch updated · 1 new branch · 1 branch pruned'
  [ "${lines[1]}" = "${expected} · 1 new tag" ]
  [ "${#lines[@]}" -eq 2 ]
  run "${kit_root}/bin/kit" flow incoming
  [ "${lines[1]}" = "📥 unchanged" ]
  run "${kit_root}/bin/kit" flow incoming --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"fetched":false,"primary":"origin","remotes":[{"name":"origin","forge":null,"host":null}]}' ]
  run "${kit_root}/bin/kit" flow incoming --no-fetch
  [ "$output" = "📡 origin · no forge · ${work}/origin.git"$'\n'"📥 not fetched" ]
}

@test "kit flow --stages prints the configured stages in order, every stage without the setting, and refuses a name that is none" {
  run "${kit_root}/bin/kit" flow --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'backlog\nunderway\ndesk\nworktrees\nchanges\nincoming\nbuilds\nreleases' ]
  stages 'releases, worktrees, desk'
  run "${kit_root}/bin/kit" flow --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'releases\nworktrees\ndesk' ]
  printf 'flow: [worktrees, nope]\n' > kit.yml
  run "${kit_root}/bin/kit" flow --stages
  [ "$status" -eq 2 ]
  [[ "$output" == *"'nope' is not a stage; the stages are: backlog underway desk worktrees changes incoming builds releases (flow in kit.yml)"* ]]
  run "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 2 ]
  [[ "$output" == *"'nope' is not a stage"* ]]
}

@test "kit flow prints each configured stage in order under a dim heading with its count, a blank line between stages" {
  local esc=$'\x1b'
  stages 'worktrees, desk, incoming'
  git worktree add --quiet -b 7-topic worktrees/7-topic main
  printf 'x\n' > c
  run --keep-empty-lines "${kit_root}/bin/kit" flow
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🌿 WORKTREES 1" ]
  [ "${lines[1]}" = "🌿 worktrees/7-topic · ↑0 ↓0 · +0 -0" ]
  [ "${lines[2]}" = "" ]
  [ "${lines[3]}" = "💻 DESK 1" ]
  [ "${lines[4]}" = "💻 main · ↑0 ↓0 · +0 -0 · 1 uncommitted" ]
  [ "${lines[5]}" = "   ?? c" ]
  [ "${lines[6]}" = "" ]
  [ "${lines[7]}" = "📥 INCOMING" ]
  [ "${lines[8]}" = "📡 origin · no forge · ${work}/origin.git" ]
  [ "${lines[9]}" = "📥 unchanged" ]
  [ "${#lines[@]}" -eq 10 ]
  # the heading is dim only when effects are on; --text prints the same picture
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" flow --text --no-fetch
  [[ "${lines[0]}" == "🌿 ${esc}[1mWORKTREES${esc}[0m ${esc}[90m1${esc}[0m" ]]
  [[ "$output" == *"💻 ${esc}[1mDESK${esc}[0m ${esc}[90m1${esc}[0m"* ]]
  [[ "$output" == *"📥 ${esc}[1mINCOMING${esc}[0m"$'\n'"📡 origin${esc}[90m · no forge · ${work}/origin.git${esc}[0m"$'\n'"${esc}[90m📥 not fetched${esc}[0m"* ]]
  run --keep-empty-lines "${kit_root}/bin/kit" flow --text --no-fetch
  [[ "$output" != *"${esc}["* ]]
  [ "${lines[7]}" = "📥 INCOMING" ]
  [ "${lines[8]}" = "📡 origin · no forge · ${work}/origin.git" ]
  [ "${lines[9]}" = "📥 not fetched" ]
}

@test "kit flow --json is one object: stages in the configured order, and under each name what the stage alone prints" {
  stages 'incoming, desk, worktrees'
  git worktree add --quiet -b 7-topic worktrees/7-topic main
  run "${kit_root}/bin/kit" flow --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'keys_unsorted' <<<"$output")" = '["stages","incoming","desk","worktrees"]' ]
  [ "$(jq -c '.stages' <<<"$output")" = '["incoming","desk","worktrees"]' ]
  [ "$(jq -c '[.incoming.fetched, .incoming.primary, .desk.branch, [.worktrees.worktrees[].branch],
               .worktrees.goneBranches]' <<<"$output")" = '[true,"origin","main",["7-topic"],[]]' ]
  [ "$(jq -c '.worktrees | keys_unsorted' <<<"$output")" = '["worktrees","goneBranches"]' ]
  [ "$(jq -c .desk <<<"$output")" = "$("${kit_root}/bin/kit" flow desk --json)" ]
  [ "$(jq -c .worktrees <<<"$output")" = "$("${kit_root}/bin/kit" flow worktrees --json)" ]
  run "${kit_root}/bin/kit" flow --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.incoming' <<<"$output")" = '{"fetched":false,"primary":"origin","remotes":[{"name":"origin","forge":null,"host":null}]}' ]
  run "${kit_root}/bin/kit" flow --jq
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "{" ]
}

@test "with a forge, the picture counts the backlog, the changes, the builds, and the releases, a blank line between stages" {
  forge-world
  run --keep-empty-lines "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🎫 BACKLOG 2" ]
  [[ "${lines[1]}" == " #9        "*"d  🌿 9-other  🎫 Other thing" ]]
  [[ "${lines[2]}" == "#11  👤 ada  "*"d              🎫 Third thing" ]]
  [ "${lines[3]}" = "" ]
  [ "${lines[4]}" = "🛫 UNDERWAY 1" ]
  [[ "${lines[5]}" == "#7  👤 bob  🛫 "*"d  🔀 #3 🔨 draft  👤 ada   🎫 Fix the thing" ]]
  [ "${lines[6]}" = "" ]
  [ "${lines[7]}" = "💻 DESK 1" ]
  [ "${lines[8]}" = "💻 main · ↑0 ↓0 · +0 -0" ]
  [ "${lines[9]}" = "" ]
  [ "${lines[10]}" = "🌿 WORKTREES 0" ]
  [ "${lines[11]}" = "no worktree" ]
  [ "${lines[12]}" = "" ]
  [ "${lines[13]}" = "🔀 CHANGES 1" ]
  [[ "${lines[14]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  [[ "${lines[15]}" == "   #7  👤 bob  🛫 in flight "*"d  🎫 Fix the thing" ]]
  [ "${lines[16]}" = "" ]
  [ "${lines[17]}" = "📥 INCOMING" ]
  [ "${lines[18]}" = "📡 origin · github · ${work}/origin.git" ]
  [ "${lines[19]}" = "📥 not fetched" ]
  [ "${lines[20]}" = "" ]
  [ "${lines[21]}" = "🏭 BUILDS 1" ]
  [[ "${lines[22]}" == "✅  kit  push  main  "*"d  3m 36s" ]]
  [ "${lines[23]}" = "" ]
  [ "${lines[24]}" = "📦 RELEASES 3" ]
  [[ "${lines[25]}" == "0.1.0-rc.7  📝 draft "* ]]
  [[ "${lines[27]}" == "0.0.9       📦 release "*"  ⭐ latest" ]]
  [ "${#lines[@]}" -eq 28 ]
  run "${kit_root}/bin/kit" flow --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.stages' <<<"$output")" \
    = '["backlog","underway","desk","worktrees","changes","incoming","builds","releases"]' ]
  [ "$(jq -c 'keys_unsorted' <<<"$output")" \
    = '["stages","backlog","underway","desk","worktrees","changes","incoming","builds","releases"]' ]
  [ "$(jq -c '[(.backlog | map(.number)), (.underway | map([.number, .pr.number, .here])), (.changes | map(.number)),
               (.builds | map(.id)), (.releases | map(.version))]' <<<"$output")" \
    = '[[9,11],[[7,3,false]],[3],[11],["0.1.0-rc.7","0.1.0-rc.6","0.0.9"]]' ]
  [ "$(jq -c '.underway | del(.. | .age?)' <<<"$output")" \
    = "$("${kit_root}/bin/kit" flow underway --json | jq -c 'del(.. | .age?)')" ]
  # the ages count seconds from now, so they are left out of the comparison
  [ "$(jq -c '.changes | del(.. | .age?)' <<<"$output")" \
    = "$("${kit_root}/bin/kit" flow changes --json | jq -c 'del(.. | .age?)')" ]
  # the filters reach the backlog and the changes
  run "${kit_root}/bin/kit" flow --no-fetch --mine --json
  [ "$(jq -c '[(.backlog | map(.number)), (.underway | map(.number)), (.changes | map(.number))]' <<<"$output")" \
    = '[[11],[7],[3]]' ]
  run "${kit_root}/bin/kit" flow --no-fetch --owner=nobody --json
  [ "$(jq -c '[.backlog, .underway, .changes]' <<<"$output")" = '[[],[],[]]' ]
}

@test "an empty stage says so in one dim line, alone or in the picture, and prints [] with --json" {
  local esc=$'\x1b'
  forge-world
  empty-forge
  run --keep-empty-lines "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🎫 BACKLOG 0" ]
  [ "${lines[1]}" = "no open issue without a pull request" ]
  [ "${lines[3]}" = "🛫 UNDERWAY 0" ]
  [ "${lines[4]}" = "no issue is underway" ]
  [ "${lines[12]}" = "🔀 CHANGES 0" ]
  [ "${lines[13]}" = "no open pull request" ]
  [ "${lines[19]}" = "🏭 BUILDS 0" ]
  [ "${lines[20]}" = "no build since the last 20 runs" ]
  [ "${lines[22]}" = "📦 RELEASES 0" ]
  [ "${lines[23]}" = "no release yet" ]
  [ "${#lines[@]}" -eq 24 ]
  run "${kit_root}/bin/kit" flow backlog
  [ "$status" -eq 0 ]
  [ "$output" = "no open issue without a pull request" ]
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" flow backlog
  [ "$output" = "${esc}[90mno open issue without a pull request${esc}[0m" ]
  run "${kit_root}/bin/kit" flow releases
  [ "$output" = "no release yet" ]
  run "${kit_root}/bin/kit" flow changes --json
  [ "$status" -eq 0 ]
  [ "$output" = "[]" ]
  run "${kit_root}/bin/kit" flow releases --json
  [ "$output" = "[]" ]
  run "${kit_root}/bin/kit" flow builds
  [ "$output" = "no build since the last 20 runs" ]
  run "${kit_root}/bin/kit" flow builds --json
  [ "$output" = "[]" ]
}

@test "a stage the forge cannot answer is drawn as its refusal under its heading, and the picture exits 1" {
  # the forge is named, and its tool fails every call
  commit-settings 'forge: github'
  export PATH="$(shims gh)"
  run "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 1 ]
  [ "${lines[0]}" = "🎫 BACKLOG" ]
  [[ "${lines[1]}" == "❌ "* ]]
  [[ "$output" == *$'\n💻 DESK 1\n💻 main'* ]]
  run "${kit_root}/bin/kit" flow --no-fetch --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.backlog, .underway, .changes, .builds, .releases, .desk.branch]' <<<"${lines[${#lines[@]} - 1]}")" \
    = '[null,null,null,null,null,"main"]' ]
  run "${kit_root}/bin/kit" flow changes
  [ "$status" -eq 1 ]
}

@test "a repository with no origin draws its checkouts, says so once, and exits 0; a forge stage alone still refuses" {
  fresh-repo alone >/dev/null
  printf 'one\n' > a && git add a && git commit --quiet --message one
  run --keep-empty-lines "${kit_root}/bin/kit" flow
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == *"no origin: nothing fetched, no pull request shown" ]]
  [ "${lines[1]}" = "" ]
  [ "${lines[2]}" = "💻 DESK 1" ]
  [ "${lines[3]}" = "💻 main · ↑0 ↓0 · +0 -0" ]
  [ "${lines[4]}" = "" ]
  [ "${lines[5]}" = "🌿 WORKTREES 0" ]
  [ "${lines[6]}" = "no worktree" ]
  [ "${#lines[@]}" -eq 7 ]
  run "${kit_root}/bin/kit" flow --json
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == *"no origin: nothing fetched, no pull request shown" ]]
  [ "$(jq -c '[.backlog, .incoming, .changes, .builds, .releases, .desk.branch, .worktrees.worktrees]' \
    <<<"${lines[1]}")" = '[null,{"fetched":false,"primary":"origin","remotes":[]},null,null,null,"main",[]]' ]
  run "${kit_root}/bin/kit" flow changes
  [ "$status" -eq 1 ]
  [[ "$output" == *"does not have a remote named origin"* ]]
  run "${kit_root}/bin/kit" flow builds
  [ "$status" -eq 1 ]
  [[ "$output" == *"does not have a remote named origin"* ]]
}

@test "an origin that does not name a forge is fetched, said once, and its checkouts drawn; kit flow exits 0" {
  fresh-repo alone >/dev/null
  printf 'one\n' > a && git add a && git commit --quiet --message one
  git init --quiet --bare "${BATS_TEST_TMPDIR}/origin.git"
  git remote add origin "${BATS_TEST_TMPDIR}/origin.git"
  git push --quiet origin main
  run --keep-empty-lines "${kit_root}/bin/kit" flow
  [ "$status" -eq 0 ]
  [ "$(grep --count 'does not name a forge' <<<"${output}")" -eq 1 ]
  [[ "${lines[0]}" == *"origin is a local path, so it does not name a forge; kit does not show any pull request"* ]]
  [[ "$output" == *"💻 DESK 1"* ]]
  [[ "$output" == *"🌿 WORKTREES 0"* ]]
  [[ "$output" != *"BACKLOG"* ]]
  run "${kit_root}/bin/kit" flow --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.backlog, .changes, .builds, .releases, .desk.branch]' <<<"${lines[${#lines[@]} - 1]}")" = \
    '[null,null,null,null,"main"]' ]
}

@test "on a terminal, kit flow hands off to kittui with its arguments; a stage, --text, --json, or no terminal stays in text" {
  fake-tui
  stages 'desk'
  run pty "'${kit_root}/bin/kit' flow --no-fetch --mine"
  [[ "$output" == *"building kittui from projects/tui/"* ]]
  [[ "$output" == *"kittui native-native flow --no-fetch --mine"* ]]
  [[ "$output" != *"DESK"* ]]
  run pty "'${kit_root}/bin/kit' flow --text --no-fetch"
  [[ "$output" == *"DESK 1"* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow --json --no-fetch"
  [[ "$output" == *'"stages":["desk"]'* && "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow desk"
  [[ "$output" == *"💻 main"* && "$output" != *"kittui"* && "$output" != *"DESK"* ]]
  run "${kit_root}/bin/kit" flow --no-fetch </dev/null
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "💻 DESK 1" ]
  [[ "$output" != *"kittui"* ]]
  run pty "'${kit_root}/bin/kit' flow --bogus"
  [[ "$output" == *"unknown flag '--bogus'; run: kit help flow"* && "$output" != *"kittui"* ]]
}

@test "on a terminal with no kittui to be had, kit flow prints the text picture and says nothing of kittui" {
  stages 'desk'
  run pty "env PATH='$(without-kittui)' '${kit_root}/bin/kit' flow --no-fetch"
  [[ "$output" == *"DESK 1"* ]]
  [[ "$output" != *"kittui"* && "$output" != *"mise install"* ]]
}

@test "kittui runs the kit that handed off, named in KIT" {
  fake-tui
  export FAKE_KITTUI_PRINT_KIT=1
  stages 'worktrees'
  run pty "'${kit_root}/bin/kit' flow --no-fetch"
  [[ "$output" == *"kittui native-native flow --no-fetch"* ]]
}

@test "in a worktree, worktrees draws that worktree alone and changes its pull request; at the desk, every linked worktree" {
  forge-world
  git worktree add --quiet -b 7-fix-the-thing worktrees/7-fix-the-thing origin/7-fix-the-thing
  git worktree add --quiet -b elsewhere ../elsewhere main
  cd worktrees/7-fix-the-thing
  run "${kit_root}/bin/kit" flow worktrees
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "🌿 7-fix-the-thing · ↑0 ↓0 · +0 -0" ]
  [ "${#lines[@]}" -eq 1 ]
  run "${kit_root}/bin/kit" flow worktrees --json
  [ "$(jq -c '[has("desk"), [.worktrees[].branch]]' <<<"$output")" = '[false,["7-fix-the-thing"]]' ]
  run "${kit_root}/bin/kit" flow changes --others
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  run "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\n💻 DESK 1\n💻 main on ../.. · '*$'\n🌿 WORKTREES 1\n🌿 7-fix-the-thing · '* ]]
  [[ "$output" == *$'\n🔀 CHANGES 1\n🔀 #3 Fix the thing'* ]]
  cd ../../../elsewhere
  run "${kit_root}/bin/kit" flow changes
  [ "$output" = "no open pull request" ]
  cd ../desk
  run "${kit_root}/bin/kit" flow worktrees --json
  [ "$(jq -c '[has("desk"), [.worktrees[].branch]]' <<<"$output")" = '[false,["elsewhere","7-fix-the-thing"]]' ]
  run "${kit_root}/bin/kit" flow worktrees
  [[ "${lines[0]}" == "🌿 ../elsewhere "* ]]
  [[ "$output" == *$'\n\n🌿 worktrees/7-fix-the-thing'* && "$output" != *"💻"* ]]
}

@test "kit flow desk is the root checkout, at the desk and from a linked worktree, in text and as its object" {
  git worktree add --quiet -b 7-topic worktrees/7-topic main
  printf 'two\n' > b && git add b && git commit --quiet --message two
  printf 'x\n' > c
  run "${kit_root}/bin/kit" flow desk
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "💻 main · ↑1 ↓0 · +1 -0 · 2 uncommitted" ]
  [[ "$output" == *"?? c"* && "$output" == *" two" && "$output" != *"7-topic"* ]]
  run "${kit_root}/bin/kit" flow desk --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'keys' <<<"$output")" \
    = '["added","ahead","behind","branch","changes","commits","detached","path","relative","removed","uncommitted"]' ]
  [ "$(jq -c '[.path, .relative, .branch, .ahead, [.commits[].subject]]' <<<"$output")" \
    = "$(jq -nc --arg path "$(pwd -P)" '[$path, ".", "main", 1, ["two"]]')" ]
  desk=$(pwd -P)
  cd worktrees/7-topic
  run "${kit_root}/bin/kit" flow desk
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "💻 main on ../.. · ↑1 ↓0 · +1 -0 · 2 uncommitted" ]
  run "${kit_root}/bin/kit" flow desk --json
  [ "$(jq -c '[.path, .relative, .branch, .uncommitted]' <<<"$output")" \
    = "$(jq -nc --arg path "${desk}" '[$path, "../..", "main", 2]')" ]
}

@test "the worktrees stage ends with the branches whose upstream is gone, in text and as goneBranches" {
  git push --quiet origin main:gone-soon
  git fetch --quiet origin
  git branch --quiet --track gone-soon origin/gone-soon
  git push --quiet origin --delete gone-soon
  git fetch --quiet --prune origin
  run --keep-empty-lines "${kit_root}/bin/kit" flow worktrees
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "no worktree" ]
  [ "${lines[1]}" = "" ]
  [ "${lines[2]}" = "🔖 gone-soon · upstream gone · kit tidy removes it" ]
  run "${kit_root}/bin/kit" flow worktrees --json
  [ "$(jq -c .goneBranches <<<"$output")" = '["gone-soon"]' ]
}

@test "above desks, kit flow finds each desk below, stops at it, skips hidden directories, and draws each under its path" {
  stages 'desk'
  cd "${work}"
  mkdir -p group/.hidden group/deep
  git init --quiet --initial-branch=main group/.hidden/secret
  git clone --quiet origin.git group/deep/two 2>/dev/null
  git -C group/deep/two remote set-head origin main
  git init --quiet --initial-branch=main desk/nested
  run "${kit_root}/bin/kit" flow --no-fetch
  [[ "$output" == *"📂 desk"$'\n'"💻 DESK 1"$'\n'"💻 main"* ]]
  [[ "$output" == *"📂 group/deep/two"$'\n'"💻 DESK 1"$'\n'"💻 main"* ]]
  [[ "$output" != *"📂 group/.hidden"* && "$output" != *"📂 desk/nested"* ]]
  output=$("${kit_root}/bin/kit" flow --no-fetch --json 2>/dev/null)
  [ "$(jq -c 'map(.path)' <<<"$output")" = '["desk","group/deep/two"]' ]
  [ "$(jq -r '.[0].status.desk.branch' <<<"$output")" = main ]
  run "${kit_root}/bin/kit" flow desk
  [[ "$output" == *"📂 desk"$'\n'"💻 main"* ]]
  output=$("${kit_root}/bin/kit" flow desk --json 2>/dev/null)
  [ "$(jq -c 'map([.path, .status.branch])' <<<"$output")" = '[["desk","main"],["group/deep/two","main"]]' ]
  mkdir "${BATS_TEST_TMPDIR}/nowhere" && cd "${BATS_TEST_TMPDIR}/nowhere"
  run "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 1 ]
  [[ "$output" == *"no repository at or below"* ]]
}

@test "a detached checkout is marked and lists no commits" {
  git worktree add --quiet --detach worktrees/loose main
  doc=$(in_flow flow:checkout "${PWD}/worktrees/loose" '' main)
  run in_flow flow:checkout-block 🌿 "${doc}" 15
  [ "$output" = "🌿 worktrees/loose  ❗ detached HEAD · ↑0 ↓0 · +0 -0" ]
}

@test "the counts of every checkout sit in columns, ahead and added right-aligned, behind and removed left-aligned" {
  one=$(jq -nc '{ahead: 1, behind: 22, added: 3, removed: 4444, uncommitted: 0}')
  two=$(jq -nc '{ahead: 100, behind: 2, added: 30000, removed: 4, uncommitted: 2}')
  widths=$(in_flow flow:count-widths "${one}" "${two}")
  [ "${widths}" = "4 3 6 5" ]
  run in_flow flow:measures "${one}" "${widths}"
  [ "$output" = "  ↑1 ↓22 ·     +3 -4444" ]
  run in_flow flow:measures "${two}" "${widths}"
  [ "$output" = "↑100 ↓2  · +30000 -4    · 2 uncommitted" ]
}

@test "a label is the path when the branch ends it, the branch at the current directory, or branch on path" {
  run in_flow flow:label worktrees/topic topic
  [ "$output" = "$(printf 'worktrees/topic\t15')" ]
  run in_flow flow:label . main
  [ "$output" = "$(printf 'main\t4')" ]
  run in_flow flow:label ../other topic
  [ "$output" = "$(printf 'topic on ../other\t17')" ]
}

@test "a pull request's row carries its author when asked and ends with its branch, and the issues it closes follow as rows" {
  doc='{"number":2,"title":"The first","author":"ada","branch":"2-the-first","state":"queued","age":259200,'
  doc+='"unresolved":0,"checks":{"state":"PENDING","passed":0,"total":1,"failing":0,"took":36},'
  doc+='"closes":[{"number":1,"title":"One","assignees":["ada"],"age":172800},{"number":4,"title":"Four","assignees":[],"age":7200}]}'
  widths=$(in_flow flow:issue-widths "$(jq --compact-output .closes <<<"${doc}")")
  [ "${widths}" = "2 3 2 0" ]
  run in_flow flow:pr-block "${doc}" 12 0 6 2 worktrees/2-the-first "${widths}"
  [ "${lines[0]}" = "🔀 #2 The first  🚦 queued 3d  🏃 0 / 1 36s  🌿 2-the-first" ]
  [ "${lines[1]}" = "   #1  👤 ada  🛫 in flight 2d  🎫 One" ]
  [ "${lines[2]}" = "   #4          🛫 in flight 2h  🎫 Four" ]
  [ "${#lines[@]}" -eq 3 ]
  run in_flow flow:pr-block "${doc}" 12 5 6 2 '' "${widths}"
  [ "${lines[0]}" = "🔀 #2 The first  👤 ada    🚦 queued 3d  🏃 0 / 1 36s" ]
}

@test "an issue row without assignees or age, as an older forge answer gives it, keeps its number and title" {
  doc='{"number":2,"title":"The first","author":"ada","branch":"2-the-first","state":"queued","age":60,'
  doc+='"unresolved":0,"checks":{"state":"NONE"},"closes":[{"number":1,"title":"One"}]}'
  run in_flow flow:pr-block "${doc}" 12 0 6 2 '' "$(in_flow flow:issue-widths '[{"number":1,"title":"One"}]')"
  [ "${lines[1]}" = "   #1  🛫 in flight  🎫 One" ]
}

@test "a change's number is magenta and the number of an issue it closes cyan, as the backlog draws them" {
  local esc=$'\x1b'
  doc='{"number":2,"title":"The first","author":"ada","branch":"2-the-first","state":"queued","age":60,'
  doc+='"unresolved":0,"checks":{"state":"NONE"},"closes":[{"number":1,"title":"One","assignees":[],"age":120}]}'
  run env -u NO_COLOR CLICOLOR_FORCE=1 \
    bash -c 'source "${kit_root}/modules/bash/1st.sh" && use flow && flow:pr-block "$@"' _ "${doc}" 12 0 6 2 '' '2 0 2 0'
  [[ "${lines[0]}" == "🔀 ${esc}[35m#2${esc}[0m The first "* ]]
  [[ "${lines[1]}" == "   ${esc}[36m#1${esc}[0m  🛫 ${esc}[34min flight${esc}[0m ${esc}[90m2m${esc}[0m  🎫 One" ]]
}

@test "a failed checks cell names the failing count, then passed of total, then the time taken" {
  run in_flow flow:check-cell '{"state":"FAILURE","passed":3,"total":5,"failing":2,"took":31}'
  [ "$output" = "❌ 2 failing 3 / 5 31s" ]
}

@test "the pull requests shown are everyone's, the named authors' alone, or everyone's but theirs" {
  docs=$(printf '{"number":1,"author":"ada"}\n{"number":2,"author":"bob"}\n{"number":3,"author":"cy"}\n')
  run in_flow flow:select <<<"${docs}"
  [ "$(jq --slurp --compact-output 'map(.number)' <<<"$output")" = "[1,2,3]" ]
  run in_flow flow:select ada cy <<<"${docs}"
  [ "$(jq --slurp 'map(.number) | join(",")' --raw-output <<<"$output")" = "1,3" ]
  run in_flow flow:select nobody <<<"${docs}"
  [ -z "$output" ]
  run in_flow flow:select --except ada <<<"${docs}"
  [ "$(jq --slurp 'map(.number) | join(",")' --raw-output <<<"$output")" = "2,3" ]
  run in_flow flow:select --except nobody <<<"${docs}"
  [ "$(jq --slurp 'map(.number) | join(",")' --raw-output <<<"$output")" = "1,2,3" ]
}

@test "a path is printed relative to the current directory" {
  run in_flow flow:relative "${PWD}"
  [ "$output" = "." ]
  run in_flow flow:relative "${PWD}/worktrees/topic"
  [ "$output" = "worktrees/topic" ]
  cd worktrees
  run in_flow flow:relative "${PWD%/worktrees}"
  [ "$output" = ".." ]
}

@test "flow:filter turns the owner flags into flow:select's arguments, and kit flow refuses --all beside --mine" {
  run in_flow flow:filter false false bob cy
  [ "$status" -eq 0 ]
  [ "$output" = "bob cy" ]
  run in_flow flow:filter false false
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_root}/bin/kit" flow --all --mine
  [ "$status" -eq 2 ]
  [[ "$output" == *"--all cannot be combined with --mine; run: kit help flow"* ]]
  run "${kit_root}/bin/kit" flow --today
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" flow incoming --mine
  [ "$status" -eq 2 ]
  [[ "$output" == *"--mine cannot be combined with incoming; run: kit help flow"* ]]
}

@test "kit syntax flow lists the eight stages as a closed set, each described, and the removed verbs are no verbs" {
  run "${kit_root}/bin/kit" syntax flow --choices '<stage>'
  [ "$status" -eq 0 ]
  [ "$(cut -f 1 <<<"$output" | paste -s -d ' ' -)" = "backlog underway desk worktrees changes incoming builds releases" ]
  [[ "${lines[2]}" == $'desk\tthe root checkout: '* ]]
  [[ "${lines[0]}" == $'backlog\tthe open issues not closed by any open pull request, oldest first\t\t' ]]
  [[ "${lines[1]}" == $'underway\tthe open issues closed by an open pull request: who works on each, and where\t\t' ]]
  local verb
  for verb in backlog changes incoming worktrees releases
  do
    run "${kit_root}/bin/kit" "${verb}"
    [ "$status" -eq 2 ]
    [[ "$output" == *"${verb}"* ]]
    [ ! -e "${kit_root}/bin/kit-${verb}" ]
  done
}

@test "above desks, kit flow on a terminal prints the text picture and never hands off to kittui" {
  fake-tui
  stages 'desk'
  cd "${work}"
  run pty "'${kit_root}/bin/kit' flow --no-fetch"
  # the pseudo-terminal ends every line with CR LF
  [[ "${output//$'\r'/}" == *"📂 desk"$'\n'"💻 DESK 1"$'\n'"💻 main"* ]]
  [[ "$output" != *"kittui"* ]]
}

@test "flow in kit.yml must be a list of stages: a scalar, an empty list, and a repeated name are refused, and null means every stage" {
  printf 'flow: backlog\n' > kit.yml
  run "${kit_root}/bin/kit" flow --stages
  [ "$status" -eq 2 ]
  [[ "$output" == *"flow in kit.yml must be a list of stages, not 'backlog'"* ]]
  run "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 2 ]
  [[ "$output" == *"must be a list of stages, not 'backlog'"* && "$output" != *"WORKTREES"* ]]
  printf 'flow: []\n' > kit.yml
  run "${kit_root}/bin/kit" flow --stages
  [ "$status" -eq 2 ]
  [[ "$output" == *"flow in kit.yml names no stage; the stages are: backlog underway desk worktrees changes incoming builds releases"* ]]
  run "${kit_root}/bin/kit" flow --no-fetch --json
  [ "$status" -eq 2 ]
  [[ "$output" == *"names no stage"* && "$output" != *'"stages"'* ]]
  printf 'flow: [worktrees, worktrees]\n' > kit.yml
  run "${kit_root}/bin/kit" flow --stages
  [ "$status" -eq 2 ]
  [[ "$output" == *"'worktrees' is named twice in flow in kit.yml"* ]]
  run "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 2 ]
  [[ "$output" != *"WORKTREES"* ]]
  printf 'flow:\n' > kit.yml
  run "${kit_root}/bin/kit" flow --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'backlog\nunderway\ndesk\nworktrees\nchanges\nincoming\nbuilds\nreleases' ]
}

@test "a primary or a forges key that is no remote is refused with 2 by every stage, before git sees it" {
  local setting args
  for setting in 'primary: --upload-pack=touch pwned' 'primary: nowhere' \
    'forges: {"--upload-pack=touch pwned": github}'
  do
    printf '%s\n' "${setting}" > kit.yml
    for args in '' '--json' 'incoming' 'releases' 'backlog' 'underway' 'changes' 'desk' 'worktrees'
    do
      # shellcheck disable=SC2086  # args is zero or one word, a stage, and empty means none
      run "${kit_root}/bin/kit" flow ${args}
      [ "$status" -eq 2 ] || { echo "kit flow ${args}, with ${setting}: exit ${status}: ${output}"; return 1; }
    done
    run "${kit_root}/bin/kit" release 9.9.9
    [ "$status" -eq 2 ]
    run "${kit_root}/bin/kit" tidy --dry-run
    [ "$status" -eq 1 ]
  done
  printf 'primary: --upload-pack=touch pwned\n' > kit.yml
  run "${kit_root}/bin/kit" flow incoming
  [[ "$output" == *"primary '--upload-pack=touch pwned' in kit.yml is not a remote; the remotes are: origin"* ]]
  printf 'forges: {"--upload-pack=touch pwned": github}\n' > kit.yml
  run "${kit_root}/bin/kit" flow backlog
  [[ "$output" == *"'--upload-pack=touch pwned' in forges in kit.yml is not a remote name"* ]]
  [ -z "$(find "${work}" "${BATS_TEST_TMPDIR}" -name pwned)" ]
  # a forges key may name a remote this desk lacks: it is only compared with a remote's name
  printf 'forges: {upstream: gitlab}\n' > kit.yml
  run "${kit_root}/bin/kit" flow desk
  [ "$status" -eq 0 ]
}

@test "a fetch that fails warns once and the picture exits 1, without incoming in the picture, in text and as JSON" {
  stages 'desk'
  git remote set-url origin "${work}/nowhere.git"
  run "${kit_root}/bin/kit" flow
  [ "$status" -eq 1 ]
  [[ "${lines[0]}" == *"git fetch --prune failed; ahead and behind counts may be stale" ]]
  [ "$(grep -c 'fetch --prune failed' <<<"$output")" -eq 1 ]
  [[ "$output" == *$'💻 DESK 1\n💻 main'* ]]
  run "${kit_root}/bin/kit" flow --json
  [ "$status" -eq 1 ]
  [[ "$output" == *"git fetch --prune failed"* ]]
  [ "$(jq -r '[.stages[], .desk.branch] | join(" ")' <<<"${lines[${#lines[@]} - 1]}")" = "desk main" ]
  run "${kit_root}/bin/kit" flow --no-fetch
  [ "$status" -eq 0 ]
  [[ "$output" != *"fetch --prune failed"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
