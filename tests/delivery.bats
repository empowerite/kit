#!/usr/bin/env bats
# delivery.bats - kit delivery: the artifacts, environments, and deployments the forge holds, each with a mark
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them. A repository's
# origin is a GitHub URL that nothing fetches: the fake gh answers for it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
}

# github-world makes a repository whose origin is GitHub's, and the fake gh's answers: run 103 kept the artifact
# kit-0.1.0.tar.gz; the environment production asks ada to review and lets main deploy; and deployment 9 put
# release/0.1.0 on production, by ada, and succeeded.
github-world() {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://github.com/o/r.git
  fake-tools gh
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "o/r", "url": "https://github.com/o/r"}'
  fixture artifacts.json <<'JSON'
{"total_count": 1, "artifacts": [{"id": 13, "name": "kit-0.1.0.tar.gz", "size_in_bytes": 2048,
  "digest": "sha256:ab12", "expired": false, "created_at": "2026-10-01T10:00:00Z",
  "expires_at": "2026-12-30T10:00:00Z", "workflow_run": {"id": 103}}]}
JSON
  fixture environments.json <<'JSON'
[{"total_count": 1, "environments": [{"id": 1, "name": "production",
  "html_url": "https://github.com/o/r/deployments/activity_log?environments_filter=production",
  "protection_rules": [
    {"id": 5, "type": "required_reviewers", "reviewers": [{"type": "User", "reviewer": {"login": "ada"}}]},
    {"id": 6, "type": "branch_policy"}],
  "deployment_branch_policy": {"protected_branches": false, "custom_branch_policies": true}}]}]
JSON
  fixture branch-policies.json <<<'{"total_count": 1, "branch_policies": [{"name": "main", "type": "branch"}]}'
  fixture deployments.json <<'JSON'
{"data": {"repository": {"url": "https://github.com/o/r", "deployments": {"totalCount": 1, "nodes": [
  {"databaseId": 9, "environment": "production", "ref": {"name": "release/0.1.0"}, "commitOid": "8e817bf0",
   "state": "ACTIVE", "createdAt": "2026-10-01T10:00:00Z", "creator": {"login": "ada"},
   "commit": {"url": "https://github.com/o/r/commit/8e817bf0"},
   "latestStatus": {"state": "SUCCESS", "logUrl": "https://github.com/o/r/actions/runs/103/job/1"}}]}}}}
JSON
  printf 'HTTP/2.0 200 OK\r\nETag: W/"m1"\r\n\r\n[]\n' > "${FAKE_DIR}/marks-200.txt"
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'api --include*' marks-200.txt
  route gh 'api repos/{owner}/{repo}/actions/artifacts?per_page=100' artifacts.json
  route gh 'api --paginate --slurp repos/{owner}/{repo}/environments?per_page=100' environments.json
  route gh 'api repos/{owner}/{repo}/environments/production/deployment-branch-policies?per_page=100' \
    branch-policies.json
  route gh 'api graphql*deployments(first: 50*' deployments.json
}

@test "kit delivery --stages prints its three stages, and --marks gives each the runs mark, with its own kind's" {
  run "${kit_root}/bin/kit" delivery --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'artifacts\nenvironments\ndeployments' ]
  github-world
  run --separate-stderr "${kit_root}/bin/kit" delivery --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"artifacts":"W/\"m1\"","environments":"W/\"m1\" W/\"m1\"","deployments":"W/\"m1\" W/\"m1\""}' ]
  grep --quiet 'api --include repos/{owner}/{repo}/deployments?per_page=50' "${FAKE_DIR}/gh.calls"
  grep --quiet 'api --include repos/{owner}/{repo}/environments?per_page=100' "${FAKE_DIR}/gh.calls"
}

@test "artifacts is a row per artifact, keyed by its id, its page the name's link" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" delivery artifacts --json
  [ "$status" -eq 0 ]
  expected='[{"key":13,"id":13,"name":"kit-0.1.0.tar.gz","digest":"sha256:ab12","size":2048,"run":103,'
  expected+='"expires":1798624800,"links":{"name":"https://github.com/o/r/actions/runs/103/artifacts/13"}}]'
  [ "$(jq -c '.rows | map(del(.age, .organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c '.errors' <<<"$output")" = '[]' ]
}

@test "environments is a row per environment, keyed by its name: its protection, approvers, and branches" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" delivery environments --json
  [ "$status" -eq 0 ]
  expected='[{"key":"production","name":"production","protection":["branches","reviewers"],"approvers":["ada"],'
  expected+='"branches":["main"],'
  expected+='"links":{"name":"https://github.com/o/r/deployments/activity_log?environments_filter=production"}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
}

@test "deployments is a row per deployment, its environment linked to that row and its commit to the commit's page" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" delivery deployments --json
  [ "$status" -eq 0 ]
  expected='[{"key":9,"id":9,"environment":"production","ref":"release/0.1.0","sha":"8e817bf0","state":"success",'
  expected+='"actor":"ada","links":{"environment":"row:environments/production",'
  expected+='"sha":"https://github.com/o/r/commit/8e817bf0","state":"https://github.com/o/r/actions/runs/103/job/1"}}]'
  [ "$(jq -c '.rows | map(del(.age, .organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c 'has("limited")' <<<"$output")" = false ]
  # 60 deployments, of which GitHub's read holds 50, give 50 rows and say so
  jq '.data.repository.deployments |= (.totalCount = 60
       | .nodes = [range(60) as $i | .nodes[0] | .databaseId = (100 - $i)])' "${FAKE_DIR}/deployments.json" \
    > "${FAKE_DIR}/deployments.new" && mv "${FAKE_DIR}/deployments.new" "${FAKE_DIR}/deployments.json"
  run --separate-stderr "${kit_root}/bin/kit" delivery deployments --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[(.rows | length), .rows[0].key, .limited]' <<<"$output")" = '[50,100,50]' ]
}

@test "kit delivery with --text prints a stage as text, and says where kit stopped reading" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" delivery deployments --text
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "id  environment  ref  "*"sha  "*"state  "*"actor  "*"age" ]]
  [[ "${lines[1]}" == "9  "*"production  "*"release/0.1.0  "*"success  "*"ada  "* ]]
  [ "${#lines[@]}" -eq 2 ]
  jq '.data.repository.deployments.totalCount = 60' "${FAKE_DIR}/deployments.json" > "${FAKE_DIR}/deployments.new" \
    && mv "${FAKE_DIR}/deployments.new" "${FAKE_DIR}/deployments.json"
  run --separate-stderr "${kit_root}/bin/kit" delivery deployments --text
  [ "$status" -eq 0 ]
  [ "${lines[-1]}" = 'kit read the first 1; the forge holds more' ]
}

@test "the other Delivery tabs are kit flow's: kit delivery releases or builds --json refuses with 2, naming it" {
  github-world
  for tab in releases builds
  do
    run --separate-stderr "${kit_root}/bin/kit" delivery "${tab}" --json
    [ "$status" -eq 2 ] || { echo "${tab}: ${status}"; return 1; }
    [ -z "$output" ]
    [[ "$stderr" == *"run: cd "*" && kit flow ${tab} --json"* ]]
  done
}

@test "kit delivery --json draws every stage, and a repository with no forge draws none" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" delivery --json
  [ "$status" -eq 0 ]
  expected='{"stages":["artifacts","environments","deployments"],"artifacts":{"rows":[],"errors":[]},'
  expected+='"environments":{"rows":[],"errors":[]},"deployments":{"rows":[],"errors":[]}}'
  [ "$output" = "${expected}" ]
  run --separate-stderr "${kit_root}/bin/kit" delivery --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"artifacts":null,"environments":null,"deployments":null}' ]
}

# ----------------------------------------------------------------------------------------------------------------------

@test "each stage prints the keys of the fixture kittui's tests decode, and the fields its catalog declares" {
  local stage failed=0
  github-world
  for stage in $("${kit_root}/bin/kit" delivery --stages)
  do
    "${kit_root}/bin/kit" delivery "${stage}" --json > "${BATS_TEST_TMPDIR}/${stage}.json" 2>/dev/null || true
    same-keys "projects/tui/internal/flow/testdata/delivery-${stage}.json" < "${BATS_TEST_TMPDIR}/${stage}.json" || failed=1
    fields-hold "delivery/${stage}" < "${BATS_TEST_TMPDIR}/${stage}.json" || failed=1
  done
  (( failed == 0 ))
}
