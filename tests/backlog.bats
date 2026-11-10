#!/usr/bin/env bats
# backlog.bats - kit flow backlog lists the open issues nothing is in flight for, oldest first, as every stage draws
# ----------------------------------------------------------------------------------------------------------------------
# The world is forge-conformance.bats's, seen through a fake gh: the caller ada; issue 7 "Fix the thing", assigned to
# bob, with change 3 open on branch 7-fix-the-thing and closing it, so 7 is the changes stage's and not the backlog's;
# issue 9 "Other thing", assigned to nobody, with the branch 9-other on origin and no change, so it is in the backlog
# with its branch; and issue 11, with neither.

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
{"data": {"repository": {"nameWithOwner": "empowerite/example",
  "issues": {"nodes": [{"number": 7, "assignees": {"nodes": [{"login": "bob"}]}}]},
  "pullRequests": {"nodes": [{
  "number": 3, "title": "Fix the thing", "isDraft": true, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"},
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing", "createdAt": "2026-09-20T09:00:00Z",
                                         "repository": {"nameWithOwner": "empowerite/example"}}]},
  "reviewThreads": {"nodes": []},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}}]}}}}
JSON
  route 'api user*' user.json
  route 'repo view*' repo.json
  route 'issue list --state open --limit 10000 --json *' issues.json
  route 'api graphql*' open-prs.json
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

# title-of <row> prints the title cell of a row, the last: what follows the issue glyph.
title-of() {
  printf '%s\n' "${1##*🎫 }"
}

@test "kit flow backlog draws each issue nothing is in flight for: number, assignees, age, the branch if any, then title" {
  run kit flow backlog
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == " #9        "*"d  🌿 9-other  🎫 Other thing" ]]
  [[ "${lines[1]}" == "#11  👤 ada  "*"d              🎫 Third thing" ]]
  [[ "$output" != *"#7"* && "$output" != *"in flight"* && "$output" != *"idle"* ]]
  [ "${#lines[@]}" -eq 2 ]
}

@test "kit flow backlog right-aligns the age, a number, in its column, and leaves the branch column out when no issue has one" {
  # The dates give ages of four and three digits, widths that hold until 2028.
  fixture issues.json <<'JSON'
[{"number": 11, "title": "Third thing", "assignees": [], "createdAt": "2025-06-01T09:00:00Z"},
 {"number": 5, "title": "Old thing", "assignees": [], "createdAt": "2020-01-01T09:00:00Z"}]
JSON
  run kit flow backlog
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == " #5  "[0-9][0-9][0-9][0-9]"d  🎫 Old thing" ]]
  [[ "${lines[1]}" == "#11   "[0-9][0-9][0-9]"d  🎫 Third thing" ]]
}

@test "kit flow backlog --mine drops the assignee column, as kit flow --mine drops the author; --owner keeps it" {
  run kit flow backlog --mine
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#11' 'Third thing'
  [[ "${lines[0]}" != *"ada"* ]]
  [ "${#lines[@]}" -eq 1 ]
  fixture issues.json <<'JSON'
[{"number": 11, "title": "Third thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-30T09:00:00Z"}]
JSON
  forge-forget
  run kit flow backlog --owner=bob
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#11' 'bob' 'Third thing'
  [ "${#lines[@]}" -eq 1 ]
}

@test "kit flow backlog --others is every issue --mine leaves out, unassigned ones included, with the assignee column" {
  run kit flow backlog --others --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "9" ]
  run kit flow backlog --others
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#9' '9-other' 'Other thing'
  [ "${#lines[@]}" -eq 1 ]
}

@test "kit flow backlog --json says the same as data, each issue with its branch on origin or null" {
  run kit flow backlog --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "9 11" ]
  [ "$(jq --raw-output '.[0].branch' <<<"$output")" = "9-other" ]
  [ "$(jq --compact-output '.[1].branch' <<<"$output")" = 'null' ]
  [ "$(jq --compact-output '.[0] | has("inflight")' <<<"$output")" = 'false' ]
}

@test "kit flow backlog paints the number cyan and the detail dim, and no color without a terminal" {
  local esc=$'\x1b'
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_checkout}/bin/kit" flow backlog
  [ "$status" -eq 0 ]
  [[ "$output" == *"${esc}[36m #9${esc}[0m"* ]]
  [[ "$output" == *"${esc}[90mada${esc}[0m"* ]]
  [[ "$output" == *"${esc}[90m🌿 9-other${esc}[0m"* ]]
  [[ "$output" == *"🎫 Other thing"* ]]
  run env -u NO_COLOR "${kit_checkout}/bin/kit" flow backlog
  [ "$status" -eq 0 ]
  [[ "$output" != *"${esc}["* ]]
}

@test "kit flow backlog clips the title to what the terminal's width leaves after the other cells" {
  fixture issues.json <<'JSON'
[{"number": 11, "title": "A very long title that goes on and on, far past what a narrow terminal shows",
  "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  run env COLUMNS=60 "${kit_checkout}/bin/kit" flow backlog
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "#11  👤 bob  "*"d  🎫 A very long"*"…" ]]
  narrow=$(title-of "${lines[0]}")
  run env COLUMNS=80 "${kit_checkout}/bin/kit" flow backlog
  [ "$status" -eq 0 ]
  wide=$(title-of "${lines[0]}")
  [ "${#wide}" -eq $(( ${#narrow} + 20 )) ]
  run env COLUMNS=160 "${kit_checkout}/bin/kit" flow backlog
  [ "$status" -eq 0 ]
  [ "$(title-of "${lines[0]}")" = "A very long title that goes on and on, far past what a narrow terminal shows" ]
}

@test "kit flow backlog --today lists the issues opened in the last 24 hours, and combines with --mine and --json" {
  now=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  fixture issues.json <<JSON
[{"number": 13, "title": "Fresh thing", "assignees": [{"login": "ada"}], "createdAt": "${now}"},
 {"number": 14, "title": "Fresh other", "assignees": [], "createdAt": "${now}"},
 {"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  run kit flow backlog --today
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#13' 'ada' 'Fresh thing'
  in-order "${lines[1]}" '#14' 'Fresh other'
  [ "${#lines[@]}" -eq 2 ]
  run kit flow backlog --today --mine
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#13' 'Fresh thing'
  [[ "${lines[0]}" != *"ada"* ]]
  [ "${#lines[@]}" -eq 1 ]
  run kit flow backlog --today --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map(.number)' <<<"$output")" = "[13,14]" ]
  run kit flow backlog --today --owner=bob
  [ "$status" -eq 0 ]
  [ "$output" = "no open issue without a pull request" ]
}

@test "kit flow backlog refuses an unknown flag with a pointer to its page, and kit backlog and kit todo are no verbs" {
  run kit flow backlog --bogus
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--bogus'; run: kit help flow"* ]]
  run kit backlog
  [ "$status" -eq 2 ]
  [[ "$output" == *"backlog"* ]]
  run kit todo
  [ "$status" -eq 2 ]
  [[ "$output" == *"todo"* ]]
}

@test "kit flow backlog --mine combines with --owner and --others as a union, keeping the assignee column; --all stands alone" {
  # #7 is in flight: change #3 closes it, so the backlog never lists it
  run kit flow backlog --mine --owner=bob --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "11" ]
  run kit flow backlog --mine --owner=bob
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#11' 'ada' 'Third thing'
  [ "${#lines[@]}" -eq 1 ]
  run kit flow backlog --mine --others --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "9 11" ]
  run kit flow backlog --all --mine
  [ "$status" -eq 2 ]
  [[ "$output" == *"--all cannot be combined with --mine; run: kit help flow"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
