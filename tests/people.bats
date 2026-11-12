#!/usr/bin/env bats
# people.bats - kit people: who has access to the repository, its teams, and the bots that acted in it
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them. A repository's
# origin is a GitHub URL that nothing fetches: the fake gh answers for it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
}

# github-world makes a repository whose origin is GitHub's, and the fake gh's answers: ada is an admin, bob may write,
# and so may dependabot[bot], a bot; the team core may write, ada and bob its members; and dependabot opened a change
# that was merged, which bob reviewed.
github-world() {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  git remote add origin https://github.com/o/r.git
  fake-tools gh
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "o/r", "url": "https://github.com/o/r"}'
  fixture collaborators.json <<'JSON'
[[{"login": "ada", "type": "User", "name": "Ada", "role_name": "admin"},
  {"login": "bob", "type": "User", "role_name": "write"},
  {"login": "dependabot[bot]", "type": "Bot", "name": null, "role_name": "write"}]]
JSON
  fixture teams.json <<'JSON'
[{"slug": "core", "name": "Core", "permission": "push", "html_url": "https://github.com/orgs/o/teams/core"}]
JSON
  fixture members.json <<'JSON'
{"data": {"organization": {"t0": {"members": {"totalCount": 2, "nodes": [{"login": "ada"}, {"login": "bob"}]}}}}}
JSON
  fixture bots.json <<'JSON'
{"data": {"repository": {
  "defaultBranchRef": {"target": {"history": {"nodes": [
    {"committedDate": "2026-10-01T09:00:00Z",
     "author": {"name": "Ada", "user": {"login": "ada", "__typename": "User"}}}]}}},
  "open": {"nodes": []},
  "merged": {"nodes": [
    {"createdAt": "2026-10-02T09:00:00Z",
     "author": {"__typename": "Bot", "login": "dependabot", "url": "https://github.com/apps/dependabot"},
     "latestReviews": {"nodes": [{"submittedAt": "2026-10-02T10:00:00Z", "author": {"__typename": "User",
       "login": "bob", "url": "https://github.com/bob"}}]}}]}}}}
JSON
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'api --paginate --slurp repos/{owner}/{repo}/collaborators?per_page=100' collaborators.json
  route gh 'api repos/{owner}/{repo}/teams?per_page=21' teams.json
  route gh 'api graphql*team(slug:*' members.json
  route gh 'api graphql*history(first: 100)*' bots.json
}

@test "kit people --stages prints its three stages, and --marks a null for each, as each reloads when asked" {
  run "${kit_root}/bin/kit" people --stages
  [ "$status" -eq 0 ]
  [ "$output" = $'people\nteams\nagents' ]
  github-world
  run --separate-stderr "${kit_root}/bin/kit" people --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"people":null,"teams":null,"agents":null}' ]
  [ ! -e "${FAKE_DIR}/gh.calls" ] || ! grep --quiet 'api --include' "${FAKE_DIR}/gh.calls"
}

@test "people is a row per person or bot with access, keyed by the login: the name, the role, and the kind" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" people people --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  expected='[{"key":"ada","login":"ada","name":"Ada","role":"admin","kind":"user","links":{}},'
  expected+='{"key":"bob","login":"bob","name":null,"role":"write","kind":"user","links":{}},'
  expected+='{"key":"dependabot[bot]","login":"dependabot[bot]","name":null,"role":"write","kind":"bot","links":{}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c '.errors' <<<"$output")" = '[]' ]
}

@test "teams is a row per team, keyed by it: its role and its members, its page the team's link" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" people teams --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  expected='[{"key":"core","team":"core","name":"Core","role":"write","members":["ada","bob"],'
  expected+='"links":{"team":"https://github.com/orgs/o/teams/core"}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
  [ "$(jq -c 'has("limited")' <<<"$output")" = false ]
  # 25 teams: kit asks for 21, and the 21st says GitHub holds more than the 20 drawn
  jq '[range(21) as $i | .[0] | .slug = "t\(100 + $i)"]' "${FAKE_DIR}/teams.json" > "${FAKE_DIR}/teams.new" \
    && mv "${FAKE_DIR}/teams.new" "${FAKE_DIR}/teams.json"
  jq '.data.organization = ([range(20) as $i | {key: "t\($i)", value: .data.organization.t0}] | from_entries)' \
    "${FAKE_DIR}/members.json" > "${FAKE_DIR}/members.new" && mv "${FAKE_DIR}/members.new" "${FAKE_DIR}/members.json"
  run --separate-stderr "${kit_root}/bin/kit" people teams --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq -c '[(.rows | length), .rows[0].key, .rows[19].key, .limited]' <<<"$output")" = '[20,"t100","t119",20]' ]
}

@test "agents is a row per bot that acted, keyed by its login: what it did and when, its page the login's link" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" people agents --json
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  expected='[{"key":"dependabot[bot]","login":"dependabot[bot]","acted":["change"],"last":1790931600,'
  expected+='"links":{"login":"https://github.com/apps/dependabot"}}]'
  [ "$(jq -c '.rows | map(del(.organization, .repository, .source))' <<<"$output")" = "${expected}" ]
}

@test "kit people with --text prints a stage as text, a list of members apart by commas" {
  github-world
  run --separate-stderr "${kit_root}/bin/kit" people teams --text
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "team  name  role   members" ]
  [ "${lines[1]}" = "core  Core  write  ada, bob" ]
  [ "${#lines[@]}" -eq 2 ]
}

@test "kit people --json draws every stage, and a repository with no forge draws none" {
  mkdir -p "${BATS_TEST_TMPDIR}/r"
  git init --quiet --initial-branch=main "${BATS_TEST_TMPDIR}/r"
  cd "${BATS_TEST_TMPDIR}/r"
  run --separate-stderr "${kit_root}/bin/kit" people --json
  [ "$status" -eq 0 ]
  expected='{"stages":["people","teams","agents"],"people":{"rows":[],"errors":[]},'
  expected+='"teams":{"rows":[],"errors":[]},"agents":{"rows":[],"errors":[]}}'
  [ "$output" = "${expected}" ]
  run --separate-stderr "${kit_root}/bin/kit" people --marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"people":null,"teams":null,"agents":null}' ]
}

# ----------------------------------------------------------------------------------------------------------------------

@test "each stage prints the keys of the fixture kittui's tests decode, and the fields its catalog declares" {
  local stage failed=0
  github-world
  for stage in $("${kit_root}/bin/kit" people --stages)
  do
    "${kit_root}/bin/kit" people "${stage}" --json > "${BATS_TEST_TMPDIR}/${stage}.json" 2>/dev/null || true
    same-keys "projects/tui/internal/flow/testdata/people-${stage}.json" < "${BATS_TEST_TMPDIR}/${stage}.json" || failed=1
    fields-hold "people/${stage}" < "${BATS_TEST_TMPDIR}/${stage}.json" || failed=1
  done
  (( failed == 0 ))
}
