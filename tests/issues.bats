#!/usr/bin/env bats
# issues.bats - kit issues: the forge's milestones and iterations, each a stage with a mark; the other tabs are kit flow's
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them. A repository's
# origin is a GitHub URL, or an Azure DevOps one, that nothing fetches: the fake gh or az answers for it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
}

# github-world makes a repository whose origin is GitHub's, and the fake gh's answers: one milestone, number 2, and a
# project with Sprint 1, which ended, and Sprint 2, which runs from before now to after it.
github-world() {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://github.com/o/r.git
  fake-tools gh
  fixture user.json <<<'{"login": "ada"}'
  fixture milestones.json <<'JSON'
[[{"number": 2, "title": "v0.1", "state": "open", "due_on": "2026-12-01T00:00:00Z", "open_issues": 3,
   "closed_issues": 2, "html_url": "https://github.com/o/r/milestone/2"}]]
JSON
  local from to
  from=$(date -u -v-3d +%Y-%m-%d 2>/dev/null || date -u -d '3 days ago' +%Y-%m-%d)
  fixture iterations.json <<JSON
{"data": {"repository": {"projectsV2": {"nodes": [{"url": "https://github.com/orgs/o/projects/4",
  "fields": {"nodes": [{"configuration": {
    "iterations": [{"id": "PVTI_2", "title": "Sprint 2", "startDate": "${from}", "duration": 14}],
    "completedIterations": [{"id": "PVTI_1", "title": "Sprint 1", "startDate": "2020-01-06", "duration": 14}]}}]}}]}}}}
JSON
  printf 'HTTP/2.0 200 OK\r\nETag: W/"m1"\r\n\r\n[]\n' > "${FAKE_DIR}/milestones-200.txt"
  fixture repo.json <<<'{"nameWithOwner": "o/r"}'
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'api --paginate --slurp repos/{owner}/{repo}/milestones?state=all&per_page=100' milestones.json
  route gh 'api --include*milestones*' milestones-200.txt
  route gh 'api --include*' milestones-200.txt
  route gh 'api graphql*projectsV2*' iterations.json
}

@test "kit issues --stages prints milestones then iterations, and --marks has a key for each" {
  run "${kit_root}/bin/kit" issues --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'milestones\niterations' ]
  github-world
  run --separate-stderr "${kit_root}/bin/kit" issues --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"milestones":"W/\"m1\"","iterations":null}' ]
}

@test "milestones is the forge's, a row keyed by its number, with its page as the title's link" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" issues milestones --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = \
    '[{"key":2,"number":2,"title":"v0.1","state":"open","due":1796083200,"open":3,"closed":2,"links":{"title":"https://github.com/o/r/milestone/2"}}]' ]
  [ "$(jq -c '.errors' <<<"$output")" = '[]' ]
}

@test "iterations is the forge's by start, each keyed by its id, and current for the one that contains now" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" issues iterations --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.key, .title, .current, .links.title]]' <<<"$output")" = \
    '[["PVTI_1","Sprint 1",false,"https://github.com/orgs/o/projects/4"],["PVTI_2","Sprint 2",true,"https://github.com/orgs/o/projects/4"]]' ]
}

@test "kit issues with no --json off a terminal prints a stage as text" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" issues milestones --text
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "number  title  state  due"* ]]
  [[ "${lines[1]}" == "2  "*"v0.1"* ]]
}

@test "a project that is not linked to GitHub's projects has no iterations, and no error" {
  github-world
  fixture none.json <<<'{"data": {"repository": {"projectsV2": {"nodes": []}}}}'
  : > "${FAKE_DIR}/gh.routes"
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'api graphql*projectsV2*' none.json
  run --separate-stderr "${kit_root}/bin/kit" issues iterations --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "a GitLab tier without iterations answers 403, and iterations is the error {feature, off}, with no failure" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://gitlab.com/o/r.git
  fake-tools glab
  fixture forbidden.txt <<<'glab: 403 Forbidden'
  route glab 'api user' user.json
  route glab 'api projects/o%2Fr/iterations?*' '!1 forbidden.txt'
  run --separate-stderr "${kit_root}/bin/kit" issues iterations --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[{"feature":"iterations","off":true}]}' ]
}

@test "Issues' first tab is kit flow's: kit issues issues, backlog, or underway --json refuses with 2, naming kit flow issues" {
  github-world
  for tab in issues backlog underway
  do
    run --separate-stderr "${kit_root}/bin/kit" issues "${tab}" --json
    [ "$status" -eq 2 ]
    [ -z "$output" ]
    [[ "$stderr" == *"run: kit flow issues --json"* ]]
  done
  run --separate-stderr "${kit_root}/bin/kit" issues --stages
  [ "$output" = $'milestones\niterations' ]
}

@test "on Azure DevOps, which has no milestones, milestones is no rows and no errors, and exits 0" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://ada@dev.azure.com/acme/empowerite/_git/example
  fake-tools az
  run --separate-stderr "${kit_root}/bin/kit" issues milestones --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"rows":[],"errors":[]}' ]
}

@test "kit issues --json draws both stages, and a repository with no forge draws none" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" issues --json
  [ "$status" -eq 0 ]
  [ "$output" = '{"stages":["milestones","iterations"],"milestones":{"rows":[],"errors":[]},"iterations":{"rows":[],"errors":[]}}' ]
}
