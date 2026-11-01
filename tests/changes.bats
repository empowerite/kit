#!/usr/bin/env bats
# changes.bats - kit status changes lists the open changes, each with the issues it closes under it
# ----------------------------------------------------------------------------------------------------------------------
#
# The world is forge-conformance.bats's, seen through a fake gh: the caller ada; issue 7 "Fix the thing", assigned to
# bob, with change 3, a draft by ada on branch 7-fix-the-thing, closing it. Issues 9 and 11 are in no change, so they are
# the backlog stage's and not drawn here.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  mkdir .kit && printf 'forge: github\n' > kit.yml
  git add --all && git commit --quiet --message '#1 start: the first'
  git push --quiet origin main main:7-fix-the-thing main:9-other
  git remote set-head origin main
  world
}

world() {
  : > "${FAKE_DIR}/gh.routes"
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
  route 'api user*' user.json
  route 'repo view*' repo.json
  route 'issue list --state open --limit 1000 --json *' issues.json
  route 'api graphql*' open-prs.json
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

@test "kit status changes draws each open change as kit status does, and under it the issues it closes as the backlog stage draws one, in flight" {
  run kit status changes
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  [[ "${lines[1]}" == "   #7  👤 bob  🛫 in flight "*"d  🎫 Fix the thing" ]]
  [ "${#lines[@]}" -eq 2 ]
  [[ "$output" != *"closes"* ]]
}

@test "kit status changes --mine, --others, and --owner choose by the change's author, as kit status does" {
  run kit status changes --mine
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  [[ "${lines[1]}" == "   #7  👤 bob  🛫 in flight "*"d  🎫 Fix the thing" ]]
  run kit status changes --others
  [ "$status" -eq 0 ]
  [ "$output" = "no open pull request" ]
  run kit status changes --owner=bob
  [ "$status" -eq 0 ]
  [ "$output" = "no open pull request" ]
}

@test "kit status changes --json prints the change documents, each issue with its assignees and age" {
  run kit status changes --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "3" ]
  [ "$(jq --compact-output '.[0].closes[0] | {number, title, assignees}' <<<"$output")" \
    = '{"number":7,"title":"Fix the thing","assignees":["bob"]}' ]
  [ "$(jq --raw-output '.[0].closes[0].age > 0' <<<"$output")" = "true" ]
}

@test "kit status changes refuses an unknown flag with a pointer to its page" {
  run kit status changes --bogus
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--bogus'; run: kit help status"* ]]
}

@test "kit status changes --mine combines with --owner and --others as a union, keeping the author column; --all stands alone" {
  run kit status changes --mine --owner=bob
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  run kit status changes --others --owner=bob
  [ "$status" -eq 0 ]
  [ "$output" = "no open pull request" ]
  run kit status changes --mine --others
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  run kit status changes --all --owner=bob
  [ "$status" -eq 2 ]
  [[ "$output" == *"--all cannot be combined with --owner; run: kit help status"* ]]
}

@test "kit status changes <number>… lists only the changes numbered so or closing an issue so numbered, with or without the #" {
  run kit status changes 3
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  run kit status changes 7
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  [[ "${lines[1]}" == "   #7  👤 bob  🛫 in flight "*"d  🎫 Fix the thing" ]]
  run kit status changes '#7' 11 --json
  [ "$status" -eq 0 ]
  [ "$(jq -r 'map(.number) | join(" ")' <<<"$output")" = "3" ]
  run kit status changes 007 --json
  [ "$(jq -r 'map(.number) | join(" ")' <<<"$output")" = "3" ]
  run kit status changes 11
  [ "$status" -eq 0 ]
  [ "$output" = "no open pull request" ]
  run kit status changes 9 --mine
  [ "$status" -eq 0 ]
  [ "$output" = "no open pull request" ]
  run kit status changes x7
  [ "$status" -eq 2 ]
  [[ "$output" == *"'x7' is not a number: 7 or #7; run: kit help status"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
