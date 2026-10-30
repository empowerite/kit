#!/usr/bin/env bats
# todo.bats - kit todo lists the forge's open issues, oldest first, drawn as kit status draws a change
# ----------------------------------------------------------------------------------------------------------------------
#
# The world is forge-conformance.bats's, seen through a fake gh: the caller ada; issue 7 "Fix the thing", assigned to
# bob, with change 3 open on branch 7-fix-the-thing and closing it; and issue 9 "Other thing", assigned to nobody. The
# branch 9-other is on origin with no change, so 9 is in flight by its branch alone. Issue 11 has neither: it is idle.

bats_require_minimum_version 1.5.0

setup() {
  export NO_COLOR=1
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}" "${FAKE_DIR}/bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/gh"
  export PATH="${FAKE_DIR}/bin:${PATH}"
  unset GITHUB_EVENT_PATH CI_MERGE_REQUEST_IID
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  mkdir .kit && printf 'forge: github\n' > .kit/settings.yml
  git add --all && git commit --quiet --message '#1 start: the first'
  git push --quiet origin main main:7-fix-the-thing main:9-other
  git remote set-head origin main
  world
}

teardown() {
  rm -rf "${work}"
}

fixture() { cat > "${FAKE_DIR}/$1"; }
route() { printf '%s\t%s\n' "$1" "$2" >> "${FAKE_DIR}/gh.routes"; }

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
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing"}]},
  "reviewThreads": {"nodes": []},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}}]}}}}
JSON
  route 'api user*' user.json
  route 'repo view*' repo.json
  route 'issue list --state open --limit 1000 --json *' issues.json
  route 'api graphql*' open-prs.json
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

# title-of <row> prints the title cell of a row: what is between the issue glyph and the first separator.
title-of() {
  local row=${1#🎫 }
  printf '%s\n' "${row%% · *}"
}

@test "kit todo draws each issue as kit status draws a change: title, assignees, state, age, and what is in flight" {
  run kit todo
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #7 Fix the thing · 👤 bob · 🛫 in flight "*"d · 🔀 #3 · 🌿 7-fix-the-thing" ]]
  [[ "${lines[1]}" == "🎫 #9 Other thing   ·        · 🛫 in flight "*"d · 🌿 9-other" ]]
  [[ "${lines[2]}" == "🎫 #11 Third thing  · 👤 ada · ⏳ idle      "*"d" ]]
  [[ "${lines[2]}" != *"🔀"* && "${lines[2]}" != *"🌿"* ]]
  [ "${#lines[@]}" -eq 3 ]
}

@test "kit todo right-aligns the age, a number, in its column" {
  # The dates give ages of four and three digits, widths that hold until 2028.
  fixture issues.json <<'JSON'
[{"number": 7, "title": "Fix the thing", "assignees": [], "createdAt": "2025-06-01T09:00:00Z"},
 {"number": 5, "title": "Old thing", "assignees": [], "createdAt": "2020-01-01T09:00:00Z"}]
JSON
  run kit todo
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #5 Old thing     · ⏳ idle      "[0-9][0-9][0-9][0-9]"d" ]]
  expected="🎫 #7 Fix the thing · 🛫 in flight  "
  [[ "${lines[1]}" == "${expected}"[0-9][0-9][0-9]"d · 🔀 #3 · 🌿 7-fix-the-thing" ]]
}

@test "kit todo --mine drops the assignee column, as kit status --mine drops the author; --owner keeps it" {
  run kit todo --mine
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #11 Third thing · ⏳ idle "*"d" ]]
  [ "${#lines[@]}" -eq 1 ]
  run kit todo --owner=bob
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #7 Fix the thing · 👤 bob · 🛫 in flight "*"d · 🔀 #3 · 🌿 7-fix-the-thing" ]]
  [ "${#lines[@]}" -eq 1 ]
}

@test "kit todo --json says the same as data, with inflight: {change, branch} or null" {
  run kit todo --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "7 9 11" ]
  [ "$(jq --compact-output '.[0].inflight' <<<"$output")" = '{"change":3,"branch":"7-fix-the-thing"}' ]
  [ "$(jq --compact-output '.[1].inflight' <<<"$output")" = '{"change":null,"branch":"9-other"}' ]
  [ "$(jq --compact-output '.[2].inflight' <<<"$output")" = 'null' ]
}

@test "kit todo paints the state by its family, idle yellow and in flight blue, and the detail dim" {
  local esc=$'\x1b'
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_checkout}/bin/kit" todo
  [ "$status" -eq 0 ]
  [[ "$output" == *"${esc}[34min flight${esc}[0m"* ]]
  [[ "$output" == *"${esc}[33midle     ${esc}[0m"* ]]
  [[ "$output" == *"${esc}[90mbob${esc}[0m"* ]]
  [[ "$output" == *"${esc}[90m🔀 #3${esc}[0m"* ]]
  [[ "$output" == *"${esc}[90m🌿 7-fix-the-thing${esc}[0m"* ]]
  [[ "$output" != *"${esc}[90m#7 Fix the thing"* ]]
  run env -u NO_COLOR "${kit_checkout}/bin/kit" todo
  [ "$status" -eq 0 ]
  [[ "$output" != *"${esc}["* ]]
  run env -u NO_COLOR CLICOLOR_FORCE=0 "${kit_checkout}/bin/kit" todo
  [ "$status" -eq 0 ]
  [[ "$output" != *"${esc}["* ]]
}

@test "kit todo clips the title to what the terminal's width leaves after the other cells" {
  fixture issues.json <<'JSON'
[{"number": 7, "title": "A very long title that goes on and on, far past what a narrow terminal shows",
  "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  run env COLUMNS=80 "${kit_checkout}/bin/kit" todo
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #7 A very long"*"… · 👤 bob · 🛫 in flight "* ]]
  [[ "${lines[0]}" == *"d · 🔀 #3 · 🌿 7-fix-the-thing" ]]
  narrow=$(title-of "${lines[0]}")
  run env COLUMNS=100 "${kit_checkout}/bin/kit" todo
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #7 A very long"*"… · 👤 bob · "* ]]
  wide=$(title-of "${lines[0]}")
  [ "${#wide}" -eq $(( ${#narrow} + 20 )) ]
  run env COLUMNS=140 "${kit_checkout}/bin/kit" todo
  [ "$status" -eq 0 ]
  [ "$(title-of "${lines[0]}")" = "#7 A very long title that goes on and on, far past what a narrow terminal shows" ]
}

@test "kit todo --today lists the issues opened in the last 24 hours, and combines with --mine and --json" {
  now=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  fixture issues.json <<JSON
[{"number": 13, "title": "Fresh thing", "assignees": [{"login": "ada"}], "createdAt": "${now}"},
 {"number": 14, "title": "Fresh other", "assignees": [], "createdAt": "${now}"},
 {"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  run kit todo --today
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #13 Fresh thing · 👤 ada · ⏳ idle "* ]]
  [[ "${lines[1]}" == "🎫 #14 Fresh other ·        · ⏳ idle "* ]]
  [ "${#lines[@]}" -eq 2 ]
  run kit todo --today --mine
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🎫 #13 Fresh thing · ⏳ idle "* ]]
  [ "${#lines[@]}" -eq 1 ]
  run kit todo --today --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map(.number)' <<<"$output")" = "[13,14]" ]
  run kit todo --today --owner=bob
  [ "$status" -eq 0 ]
  [[ "$output" == *"nothing to pick up: no open issue opened in the last 24 hours assigned to bob"* ]]
}

@test "kit todo refuses an unknown flag with a pointer to its page" {
  run kit todo --help
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--help'; run: kit help todo"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
