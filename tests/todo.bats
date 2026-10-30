#!/usr/bin/env bats
# todo.bats - kit todo lists the forge's open issues, oldest first, and marks the ones already in flight
# ----------------------------------------------------------------------------------------------------------------------
#
# The world is forge-conformance.bats's, seen through a fake gh: the caller ada; issue 7 "Fix the thing", assigned to
# bob, with change 3 open on branch 7-fix-the-thing and closing it; and issue 9 "Other thing", assigned to nobody. The
# branch 9-other is on origin with no change, so 9 is in flight by its branch alone. Issue 11 has neither.

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

@test "kit todo lists every open issue oldest first, with its assignees, and marks the change or the branch in flight" {
  run kit todo
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "#7 "*"Fix the thing"*"bob"*"#3"*"7-fix-the-thing"* ]]
  [[ "${lines[1]}" == "#9 "*"Other thing"*"9-other"* ]]
  [[ "${lines[1]}" != *"#3"* ]]
  [[ "${lines[2]}" == "#11 "*"Third thing"*"ada"* ]]
  [[ "${lines[2]}" != *"in flight"* ]]
  [ "${#lines[@]}" -eq 3 ]
}

@test "kit todo --mine lists the caller's, --owner the named assignees', and --json says the same as data" {
  run kit todo --mine
  [ "$status" -eq 0 ]
  [[ "$output" == "#11 "* ]]
  [ "${#lines[@]}" -eq 1 ]
  run kit todo --owner=bob
  [ "$status" -eq 0 ]
  [[ "$output" == "#7 "* ]]
  [ "${#lines[@]}" -eq 1 ]
  run kit todo --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "7 9 11" ]
  [ "$(jq --compact-output '.[0].inflight' <<<"$output")" = '{"change":3,"branch":"7-fix-the-thing"}' ]
  [ "$(jq --compact-output '.[1].inflight' <<<"$output")" = '{"change":null,"branch":"9-other"}' ]
  [ "$(jq --compact-output '.[2].inflight' <<<"$output")" = 'null' ]
}

@test "kit todo refuses an unknown flag with a pointer to its page" {
  run kit todo --help
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--help'; run: kit help todo"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
