#!/usr/bin/env bats
# changes.bats - kit changes lists the open changes, each as kit status draws it, with the issues it closes under it
# ----------------------------------------------------------------------------------------------------------------------
#
# The world is forge-conformance.bats's, seen through a fake gh: the caller ada; issue 7 "Fix the thing", assigned to
# bob, with change 3, a draft by ada on branch 7-fix-the-thing, closing it. Issues 9 and 11 are in no change, so they are
# kit backlog's and not drawn here.

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

@test "kit changes draws each open change as kit status does, and under it the issues it closes as kit backlog draws one, in flight" {
  run kit changes
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  [[ "${lines[1]}" == "   #7  👤 bob  🛫 in flight "*"d  🎫 Fix the thing" ]]
  [ "${#lines[@]}" -eq 2 ]
  [[ "$output" != *"closes"* ]]
}

@test "kit changes --mine, --others, and --owner choose by the change's author, as kit status does" {
  run kit changes --mine
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  [[ "${lines[1]}" == "   #7  👤 bob  🛫 in flight "*"d  🎫 Fix the thing" ]]
  run kit changes --others
  [ "$status" -eq 0 ]
  [[ "$output" == *"no open change by anyone but ada"* ]]
  run kit changes --owner=bob
  [ "$status" -eq 0 ]
  [[ "$output" == *"no open change by bob"* ]]
}

@test "kit changes --json prints the change documents, each issue with its assignees and age" {
  run kit changes --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.number) | join(" ")' <<<"$output")" = "3" ]
  [ "$(jq --compact-output '.[0].closes[0] | {number, title, assignees}' <<<"$output")" \
    = '{"number":7,"title":"Fix the thing","assignees":["bob"]}' ]
  [ "$(jq --raw-output '.[0].closes[0].age > 0' <<<"$output")" = "true" ]
}

@test "kit changes refuses an unknown flag with a pointer to its page" {
  run kit changes --help
  [ "$status" -eq 2 ]
  [[ "$output" == *"unknown flag '--help'; run: kit help changes"* ]]
}

@test "kit changes --mine combines with --owner and --others as a union, keeping the author column; --all stands alone" {
  run kit changes --mine --owner=bob
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  run kit changes --others --owner=bob
  [ "$status" -eq 0 ]
  [[ "$output" == *"no open change by anyone but ada"* ]]
  run kit changes --mine --others
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "🔀 #3 Fix the thing  👤 ada  🔨 draft "*"d  🌿 7-fix-the-thing" ]]
  run kit changes --all --owner=bob
  [ "$status" -eq 2 ]
  [[ "$output" == *"--all cannot be combined with --owner; run: kit help changes"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
