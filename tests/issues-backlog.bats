#!/usr/bin/env bats
# issues-backlog.bats - kit flow issues lists the open issues no open pull request closes as the backlog, oldest first,
# beside those underway
# ----------------------------------------------------------------------------------------------------------------------
# The world is forge-conformance.bats's, seen through a fake gh: the caller ada; issue 7 "Fix the thing", assigned to
# bob, with change 3 open on branch 7-fix-the-thing and closing it, so 7 is underway; issue 9 "Other thing", assigned
# to nobody, with the branch 9-other on origin and no change, so it is in the backlog with its branch; and issue 11,
# with neither. A test that draws the backlog alone opens no change, as no-changes does.

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

# no-changes leaves the forge with no open pull request, so every open issue is in the backlog.
no-changes() {
  fixture open-prs.json <<<'{"data": {"repository": {"issues": {"nodes": []}, "pullRequests": {"nodes": []}}}}'
}

# title-of <row> prints the title cell of a row, the last: what follows the issue glyph.
title-of() {
  printf '%s\n' "${1##*🎫 }"
}

@test "kit flow issues draws an issue in the backlog: number, assignees, age, 💤, the branch if any, then title" {
  run kit flow issues
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == " #7  👤 bob  "*"d  🛫  🔀 #3 🔨 draft  👤 ada      🎫 Fix the thing" ]]
  [[ "${lines[1]}" == " #9          "*"d  💤                  🌿 9-other  🎫 Other thing" ]]
  [[ "${lines[2]}" == "#11  👤 ada  "*"d  💤                              🎫 Third thing" ]]
  [ "${#lines[@]}" -eq 3 ]
}

@test "kit flow issues right-aligns the age, a number, in its column, and leaves the branch column out when no issue has one" {
  # The dates give ages of four and three digits, widths that hold until 2028.
  no-changes
  fixture issues.json <<'JSON'
[{"number": 11, "title": "Third thing", "assignees": [], "createdAt": "2025-06-01T09:00:00Z"},
 {"number": 5, "title": "Old thing", "assignees": [], "createdAt": "2020-01-01T09:00:00Z"}]
JSON
  run kit flow issues
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == " #5  "[0-9][0-9][0-9][0-9]"d  💤  🎫 Old thing" ]]
  [[ "${lines[1]}" == "#11   "[0-9][0-9][0-9]"d  💤  🎫 Third thing" ]]
}

@test "kit flow issues --mine drops the assignee column, as kit flow --mine drops the author; --owner keeps it" {
  run kit flow issues --mine
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#11' '💤' 'Third thing'
  [[ "${lines[0]}" != *"ada"* ]]
  [ "${#lines[@]}" -eq 1 ]
  fixture issues.json <<'JSON'
[{"number": 11, "title": "Third thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-30T09:00:00Z"}]
JSON
  forge-forget
  run kit flow issues --owner=bob
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#7' 'bob' 'Fix the thing'
  in-order "${lines[1]}" '#11' 'bob' 'Third thing'
  [ "${#lines[@]}" -eq 2 ]
}

@test "kit flow issues --others is every issue --mine leaves out, unassigned ones included, with the assignee column" {
  run kit flow issues --others --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map("\(.number) \(.status)") | join(", ")' <<<"$output")" = "7 underway, 9 backlog" ]
  run kit flow issues --others
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#7' 'bob' 'Fix the thing'
  in-order "${lines[1]}" '#9' '9-other' 'Other thing'
  [ "${#lines[@]}" -eq 2 ]
}

@test "kit flow issues --json says the same as data, each issue in the backlog with its branch on origin or null" {
  run kit flow issues --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(select(.status == "backlog") | .number) | join(" ")' <<<"$output")" = "9 11" ]
  [ "$(jq --raw-output '.[1].branch' <<<"$output")" = "9-other" ]
  [ "$(jq --compact-output '.[2] | [.branch, .pr, .here, .local, .worktree]' <<<"$output")" \
    = '[null,null,null,null,null]' ]
  [ "$(jq --compact-output '.[2] | [.milestone, .iteration, .swimlane]' <<<"$output")" = '[null,null,null]' ]
}

@test "kit flow issues paints the number cyan and the detail dim, and no color without a terminal" {
  local esc=$'\x1b'
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_checkout}/bin/kit" flow issues
  [ "$status" -eq 0 ]
  [[ "$output" == *"${esc}[36m #9${esc}[0m"* ]]
  [[ "$output" == *"${esc}[90mada${esc}[0m"* ]]
  [[ "$output" == *"🌿 ${esc}[90m9-other${esc}[0m"* ]]
  [[ "$output" == *"🎫 Other thing"* ]]
  run env -u NO_COLOR "${kit_checkout}/bin/kit" flow issues
  [ "$status" -eq 0 ]
  [[ "$output" != *"${esc}["* ]]
}

@test "kit flow issues clips the title to what the terminal's width leaves after the other cells" {
  no-changes
  fixture issues.json <<'JSON'
[{"number": 11, "title": "A very long title that goes on and on, far past what a narrow terminal shows",
  "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  run env COLUMNS=60 "${kit_checkout}/bin/kit" flow issues
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "#11  👤 bob  "*"d  💤  🎫 A very long"*"…" ]]
  narrow=$(title-of "${lines[0]}")
  run env COLUMNS=80 "${kit_checkout}/bin/kit" flow issues
  [ "$status" -eq 0 ]
  wide=$(title-of "${lines[0]}")
  [ "${#wide}" -eq $(( ${#narrow} + 20 )) ]
  run env COLUMNS=160 "${kit_checkout}/bin/kit" flow issues
  [ "$status" -eq 0 ]
  [ "$(title-of "${lines[0]}")" = "A very long title that goes on and on, far past what a narrow terminal shows" ]
}

@test "kit flow issues --today lists the issues opened in the last 24 hours, and combines with --mine and --json" {
  now=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  fixture issues.json <<JSON
[{"number": 13, "title": "Fresh thing", "assignees": [{"login": "ada"}], "createdAt": "${now}"},
 {"number": 14, "title": "Fresh other", "assignees": [], "createdAt": "${now}"},
 {"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  run kit flow issues --today
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#13' 'ada' 'Fresh thing'
  in-order "${lines[1]}" '#14' 'Fresh other'
  [ "${#lines[@]}" -eq 2 ]
  run kit flow issues --today --mine
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#13' 'Fresh thing'
  [[ "${lines[0]}" != *"ada"* ]]
  [ "${#lines[@]}" -eq 1 ]
  run kit flow issues --today --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map(.number)' <<<"$output")" = "[13,14]" ]
  run kit flow issues --today --owner=bob
  [ "$status" -eq 0 ]
  [ "$output" = "no open issue" ]
}

@test "kit flow issues refuses an unknown flag with a pointer to its page, and kit backlog and kit todo are no verbs" {
  run kit flow issues --bogus
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--bogus'; run: cd "*" && kit help flow"* ]]
  run kit backlog
  [ "$status" -eq 2 ]
  [[ "$output" == *"backlog"* ]]
  run kit todo
  [ "$status" -eq 2 ]
  [[ "$output" == *"todo"* ]]
}

@test "kit flow issues --mine combines with --owner and --others as a union, keeping the assignee column; --all stands alone" {
  run kit flow issues --mine --owner=bob --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "7 11" ]
  run kit flow issues --mine --owner=bob
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#7' 'bob' 'Fix the thing'
  in-order "${lines[1]}" '#11' 'ada' 'Third thing'
  [ "${#lines[@]}" -eq 2 ]
  run kit flow issues --mine --others --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "7 9 11" ]
  run kit flow issues --all --mine
  [ "$status" -eq 2 ]
  [[ "$output" == *"--all cannot be combined with --mine; run: cd "*" && kit help flow"* ]]
}

@test "kit flow backlog and kit flow underway are no stages: off a terminal, or with a form, they name kit flow issues" {
  for word in backlog underway
  do
    for form in '' --json --jq --text
    do
      run kit flow "${word}" ${form:+"${form}"}
      [ "$status" -eq 2 ]
      [[ "$output" == *"${word} is no stage of kit flow: issues lists every open issue, its status backlog or underway; run: cd "*" && kit flow issues"* ]]
    done
  done
  run kit flow backlog --mine
  [ "$status" -eq 2 ]
}

@test "a flow setting that names backlog or underway is refused, naming issues" {
  printf 'forge: github\nflow: [backlog, changes]\n' > kit.yml
  run kit flow --stages
  [ "$status" -eq 1 ]
  [[ "$output" == *"'backlog' is no stage of kit flow: issues lists every open issue, backlog and underway alike; name issues (flow in "*"kit.yml)"* ]]
  printf 'forge: github\nflow: [changes, underway]\n' > kit.yml
  run kit flow --stages
  [ "$status" -eq 1 ]
  [[ "$output" == *"'underway' is no stage of kit flow"* ]]
  printf 'forge: github\nflow: [issues, changes]\n' > kit.yml
  run kit flow --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'issues\nchanges' ]
}

@test "a swimlane comes from a label kit.yml's stand-in names, else its default; a forge's milestone is its own" {
  printf 'forge: github\nforge_settings:\n  github:\n    stand_ins:\n      swimlanes: {label_prefix: "lane:", default: none}\n' \
    > kit.yml
  fixture issues.json <<'JSON'
[{"number": 11, "title": "Third thing", "assignees": [], "createdAt": "2026-09-30T09:00:00Z",
  "labels": [{"name": "lane:ops"}], "milestone": {"title": "v0.1"}},
 {"number": 9, "title": "Other thing", "assignees": [], "createdAt": "2026-09-25T09:00:00Z"},
 {"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z",
  "labels": [{"name": "lane:dev"}]}]
JSON
  run kit flow issues --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map([.number, .status, .milestone, .swimlane])' <<<"$output")" \
    = '[[7,"underway",null,"dev"],[9,"backlog",null,"none"],[11,"backlog","v0.1","ops"]]' ]
  run kit flow issues
  [[ "${lines[2]}" == "#11  "*"⛳ v0.1  🏊 ops   🎫 Third thing" ]]
}

# ----------------------------------------------------------------------------------------------------------------------
