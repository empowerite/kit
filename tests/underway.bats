#!/usr/bin/env bats
# underway.bats - kit flow underway lists each open issue an open pull request closes: who works on it, and where
# ----------------------------------------------------------------------------------------------------------------------
# The world, seen through a fake gh: the caller ada; issue 7, assigned to bob, closed by two pull requests, 3, a draft
# by ada on 7-fix-the-thing, and 5, ready, by cy on 11-third; issue 11, assigned to ada, closed by 5 alone. A linked
# worktree on this desk has 7-fix-the-thing checked out, so pull request 3 is here, and 5 is on cy's desk. Neither
# comes from a fork, and the forge tells neither's last push, so those columns take no room until a test gives them.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  mkdir .kit && printf 'forge: github\n' > kit.yml && printf 'worktrees/\n' > .gitignore
  git add --all && git commit --quiet --message '#1 start: the first'
  git push --quiet origin main main:7-fix-the-thing main:11-third
  git remote set-head origin main
  git worktree add --quiet -b 7-fix-the-thing worktrees/7-fix-the-thing origin/7-fix-the-thing
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture open-prs.json <<'JSON'
{"data": {"repository": {"pullRequests": {"nodes": [
 {"number": 3, "title": "Fix the thing", "isDraft": true, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"},
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing", "createdAt": "2026-09-20T09:00:00Z",
                                         "assignees": {"nodes": [{"login": "bob"}]}}]},
  "reviewThreads": {"nodes": []},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}},
 {"number": 5, "title": "Third and more", "isDraft": false, "headRefName": "11-third",
  "createdAt": "2026-10-02T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": "APPROVED",
  "author": {"login": "cy"},
  "closingIssuesReferences": {"nodes": [
    {"number": 11, "title": "Third thing", "createdAt": "2026-09-30T09:00:00Z", "assignees": {"nodes": [{"login": "ada"}]}},
    {"number": 7, "title": "Fix the thing", "createdAt": "2026-09-20T09:00:00Z", "assignees": {"nodes": [{"login": "bob"}]}}]},
  "reviewThreads": {"nodes": []},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": {"contexts": {"nodes": [
    {"__typename": "CheckRun", "name": "ci", "status": "COMPLETED", "conclusion": "SUCCESS",
     "startedAt": "2026-10-02T09:00:00Z", "completedAt": "2026-10-02T09:03:00Z"}]}}}}]}}]}}}}
JSON
  route 'api user*' user.json
  route 'repo view*' repo.json
  route 'api graphql*' open-prs.json
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

@test "kit flow underway draws a row per issue and pull request, the oldest issue first: here, or on the author's desk" {
  run kit flow underway
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 3 ]
  [[ "${lines[0]}" == " #7  👤 bob  "*"d  🔀 #3 🔨 draft  🌿 here  🎫 Fix the thing" ]]
  [[ "${lines[1]}" == " #7  👤 bob  "*"d  🔀 #5 🏁 ready  👤 cy    🎫 Fix the thing" ]]
  [[ "${lines[2]}" == "#11  👤 ada  "*"d  🔀 #5 🏁 ready  👤 cy    🎫 Third thing" ]]
  # the cells sit in columns, so every title starts at the same column
  [ "${#lines[0]}" -eq "${#lines[1]}" ]
}

@test "kit flow underway paints the issue's number cyan, the pull request's magenta, and its state in its family" {
  local esc=$'\x1b'
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" flow underway
  [[ "${lines[2]}" == "${esc}[36m#11${esc}[0m  👤 ${esc}[90mada${esc}[0m  ${esc}[90m"* ]]
  [[ "${lines[2]}" == *"🔀 ${esc}[35m#5${esc}[0m 🏁 ${esc}[32mready${esc}[0m  👤 ${esc}[90mcy  ${esc}[0m  🎫 Third thing" ]]
}

@test "kit flow underway --json is one object per issue and pull request: the issue, its pull request, and where" {
  run kit flow underway --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'map([.number, .pr.number, .here])' <<<"$output")" = '[[7,3,true],[7,5,false],[11,5,false]]' ]
  [ "$(jq -c '.[0] | keys_unsorted' <<<"$output")" \
    = '["number","title","assignees","age","pr","here","local","worktree"]' ]
  [ "$(jq -c '.[0].pr' <<<"$output")" = '{"number":3,"title":"Fix the thing","author":"ada","state":"draft",'\
'"branch":"7-fix-the-thing","fork":null,"pushed":null}' ]
  [ "$(jq -c 'map(.local)' <<<"$output")" = '[false,false,false]' ]
  [ "$(jq -c '[.[0].assignees, .[1].worktree, (.[0].age | type)]' <<<"$output")" = '[["bob"],null,"number"]' ]
  [ "$(jq -r '.[0].worktree' <<<"$output")" -ef worktrees/7-fix-the-thing ]
}

@test "the desk's own branch is here too, drawn 💻 here, and a branch no checkout has is not" {
  git -C worktrees/7-fix-the-thing checkout --quiet --detach
  git checkout --quiet -b 11-third origin/11-third
  run kit flow underway --json
  [ "$(jq -c 'map([.pr.number, .here])' <<<"$output")" = '[[3,false],[5,true],[5,true]]' ]
  [ "$(jq -r '.[1].worktree' <<<"$output")" -ef . ]
  run kit flow underway
  [[ "${lines[1]}" == " #7  👤 bob  "*"d  🔀 #5 🏁 ready  💻 here   🎫 Fix the thing" ]]
}

@test "a branch this desk has and no checkout has is local, drawn 🌱 local; one the desk lacks is the author's" {
  git -C worktrees/7-fix-the-thing checkout --quiet --detach
  run kit flow underway --json
  [ "$(jq -c 'map([.pr.number, .here, .local, .worktree])' <<<"$output")" \
    = '[[3,false,true,null],[5,false,false,null],[5,false,false,null]]' ]
  run kit flow underway
  [[ "${lines[0]}" == " #7  👤 bob  "*"d  🔀 #3 🔨 draft  🌱 local  🎫 Fix the thing" ]]
  [[ "${lines[1]}" == " #7  👤 bob  "*"d  🔀 #5 🏁 ready  👤 cy     🎫 Fix the thing" ]]
}

@test "a fork's owner, the last push's age, and who pushed it when not the author each take a column" {
  local esc=$'\x1b' hours days
  hours=$(jq --null-input --raw-output 'now - 7200 | todate')
  days=$(jq --null-input --raw-output 'now - 3 * 86400 | todate')
  jq --arg hours "${hours}" --arg days "${days}" '.data.repository.pullRequests.nodes |= [
      (.[0] | .commits.nodes[0].commit += {committedDate: $days, author: {user: {login: "ada"}}}),
      (.[1] | .isCrossRepository = true | .headRepositoryOwner = {login: "cy"}
        | .commits.nodes[0].commit += {committedDate: $hours, author: {user: {login: "dee"}}})]' \
    "${FAKE_DIR}/open-prs.json" > "${FAKE_DIR}/fork.json"
  mv "${FAKE_DIR}/fork.json" "${FAKE_DIR}/open-prs.json"
  run kit flow underway --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'map([.pr.number, .pr.fork, .pr.pushed.by, (.pr.pushed.age / 3600 | floor)])' <<<"$output")" \
    = '[[3,null,"ada",72],[5,"cy","dee",2],[5,"cy","dee",2]]' ]
  # ada pushed her own pull request last, so no second 👤 follows its push
  run kit flow underway
  [[ "${lines[0]}" == " #7  👤 bob  "*"d  🔀 #3 🔨 draft  🌿 here         📤 3d          🎫 Fix the thing" ]]
  [[ "${lines[1]}" == " #7  👤 bob  "*"d  🔀 #5 🏁 ready  👤 cy    🍴 cy  📤 2h  👤 dee  🎫 Fix the thing" ]]
  [[ "${lines[2]}" == "#11  👤 ada  "*"d  🔀 #5 🏁 ready  👤 cy    🍴 cy  📤 2h  👤 dee  🎫 Third thing" ]]
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" flow underway
  [[ "${lines[1]}" == *"🍴 ${esc}[90mcy${esc}[0m  📤 ${esc}[90m2h${esc}[0m  👤 ${esc}[90mdee${esc}[0m  🎫 "* ]]
  # a push the forge does not tell leaves its cell blank, and the column stays for the rows that have one
  jq '.data.repository.pullRequests.nodes[0].commits.nodes[0].commit |= del(.committedDate)' \
    "${FAKE_DIR}/open-prs.json" > "${FAKE_DIR}/unknown.json"
  mv "${FAKE_DIR}/unknown.json" "${FAKE_DIR}/open-prs.json"
  run kit flow underway --json
  [ "$(jq -c 'map(.pr.pushed == null)' <<<"$output")" = '[true,false,false]' ]
  run kit flow underway
  [[ "${lines[0]}" == " #7  👤 bob  "*"d  🔀 #3 🔨 draft  🌿 here                        🎫 Fix the thing" ]]
}

@test "the owner flags match the pull request's author or the issue's assignees" {
  run kit flow underway --mine --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'map([.number, .pr.number])' <<<"$output")" = '[[7,3],[11,5]]' ]
  run kit flow underway --others --json
  [ "$(jq -c 'map([.number, .pr.number])' <<<"$output")" = '[[7,5]]' ]
  run kit flow underway --owner=bob --json
  [ "$(jq -c 'map([.number, .pr.number])' <<<"$output")" = '[[7,3],[7,5]]' ]
  run kit flow underway --owner=cy --json
  [ "$(jq -c 'map([.number, .pr.number])' <<<"$output")" = '[[7,5],[11,5]]' ]
  run kit flow underway --all --json
  [ "$(jq 'length' <<<"$output")" -eq 3 ]
  # --mine alone leaves the assignee column out, as the backlog does
  run kit flow underway --mine
  [[ "${lines[0]}" == " #7  "*"d  🔀 #3 🔨 draft  🌿 here  🎫 Fix the thing" ]]
  run kit flow underway --mine --all
  [ "$status" -eq 2 ]
}

@test "with no open pull request closing an issue, the stage says so in one dim line, and --json prints []" {
  local esc=$'\x1b'
  fixture open-prs.json <<<'{"data": {"repository": {"pullRequests": {"nodes": []}}}}'
  run kit flow underway
  [ "$status" -eq 0 ]
  [ "$output" = "no issue is underway" ]
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" flow underway
  [ "$output" = "${esc}[90mno issue is underway${esc}[0m" ]
  run kit flow underway --json
  [ "$output" = "[]" ]
}

@test "a narrow terminal clips the title and keeps every other cell whole" {
  COLUMNS=60 run kit flow underway
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == " #7  👤 bob  "*"d  🔀 #3 🔨 draft  🌿 here  🎫 Fix the thing" ]]
  jq '.data.repository.pullRequests.nodes[1].closingIssuesReferences.nodes[0].title
      = "a title long enough to run past the edge of any terminal this narrow"' "${FAKE_DIR}/open-prs.json" \
    > "${FAKE_DIR}/long.json"
  mv "${FAKE_DIR}/long.json" "${FAKE_DIR}/open-prs.json"
  COLUMNS=60 run kit flow underway
  [[ "${lines[2]}" == "#11  👤 ada  "*"d  🔀 #5 🏁 ready  👤 cy    🎫 a title long"*"…" ]]
}

@test "a forge that cannot be read is refused, and the stage exits 1" {
  printf 'forge: nowhere\n' > kit.yml
  run kit flow underway
  [ "$status" -eq 1 ]
  [[ "$output" == *"'nowhere' is not a forge the kit knows"* ]]
}

@test "kit help flow holds the underway legend, and backlog points to it" {
  run kit help flow
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\n  underway:\n\n    🎫  an issue, and its title\n'* ]]
  [[ "$output" == *$'\n    💻  here: the desk, the root checkout, has the pull request\'s branch\n'* ]]
  [[ "$output" == *$'\n    🌱  local: a local branch with no worktree;'* ]]
  [[ "$output" == *"    backlog             the open issues not closed by any open pull request,"$'\n'*"oldest first; underway has the rest"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
