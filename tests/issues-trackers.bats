#!/usr/bin/env bats
# issues-trackers.bats - kit flow issues lists the open issues of every tracker, the forge's own and each kit.yml
# declares, by reference, and stands in for the forge's closing of another tracker's
# ----------------------------------------------------------------------------------------------------------------------
# The world, seen through a fake gh and a fake az: the caller ada on GitHub; the forge's issue 7, closed by change 3, a
# draft by ada on 7-fix-the-thing, whose body says Closes AB#4521; and an Azure DevOps tracker, AB#, whose project
# Claims has four work items open: 4521, which change 3 names in its body; 4522, which change 3 does not name, but whose
# branch ab4522-start is on origin; 4523, which a merged change names, so it is done; and 4524, with none of those, in
# the backlog.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh az
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  mkdir .kit
  cat > kit.yml <<'YAML'
forge: github
trackers:
  - prefix: "AB#"
    provider: ado
    organization: https://dev.azure.com/example
    project: Claims
YAML
  git add --all && git commit --quiet --message '#1 start: the first'
  git push --quiet origin main main:7-fix-the-thing main:ab4522-start
  git remote set-head origin main
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture issues.json <<'JSON'
[{"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  fixture open-prs.json <<'JSON'
{"data": {"repository": {"nameWithOwner": "empowerite/example",
  "issues": {"nodes": [{"number": 7, "assignees": {"nodes": [{"login": "bob"}]}}]},
  "pullRequests": {"nodes": [{
  "number": 3, "title": "Fix the thing", "body": "Closes AB#4521", "isDraft": true, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"},
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing", "createdAt": "2026-09-20T09:00:00Z",
                                         "repository": {"nameWithOwner": "empowerite/example"}}]},
  "reviewThreads": {"nodes": []},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": null}}]}}]}}}}
JSON
  fixture merged.json <<'JSON'
{"data": {"repository": {"pullRequests": {"totalCount": 1, "nodes": [{"number": 2, "title": "Done", "url": "u2",
  "body": "Fixes AB#4523", "headRefName": "2-done", "mergedAt": "2026-10-02T09:00:00Z",
  "updatedAt": "2026-10-02T09:00:00Z", "author": {"login": "ada"}, "mergedBy": {"login": "ada"}}]}}}}
JSON
  fixture items.json <<'JSON'
[{"id": 4524, "fields": {"System.Title": "Later", "System.CreatedDate": "2026-09-28T09:00:00Z"}},
 {"id": 4521, "fields": {"System.Title": "Fix the claim form", "System.CreatedDate": "2026-09-22T09:00:00Z",
  "System.AssignedTo": {"uniqueName": "ada@example.com"}, "System.Tags": "forms; ui",
  "System.IterationPath": "Claims\\Sprint 4"}},
 {"id": 4522, "fields": {"System.Title": "Started", "System.CreatedDate": "2026-09-24T09:00:00Z"}},
 {"id": 4523, "fields": {"System.Title": "Merged already", "System.CreatedDate": "2026-09-26T09:00:00Z"}}]
JSON
  route 'api user*' user.json
  route 'repo view*' repo.json
  route 'issue list --state open --limit 10000 --json *' issues.json
  route 'api graphql*MERGED*' merged.json
  route 'api graphql*' open-prs.json
  route az 'boards query --project Claims --org https://dev.azure.com/example --output json --wiql *' items.json
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

@test "the issues stage lists every tracker's open issues by reference, the oldest first, each saying its tracker" {
  run kit flow issues --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map([.ref, .tracker, .status, .branch, .pr.number])' <<<"$output")" \
    = '[["#7","#","underway","7-fix-the-thing",3],["AB#4521","AB#","underway","7-fix-the-thing",3],["AB#4522","AB#","underway","ab4522-start",null],["AB#4524","AB#","backlog",null,null]]' ]
  [ "$(jq --compact-output '.[1] | [.number, .title, .assignees, .labels, .iteration, .source, .organization,
                                  .repository, .url]' <<<"$output")" \
    = '[4521,"Fix the claim form",["ada@example.com"],[{"name":"forms","color":null},{"name":"ui","color":null}],"Sprint 4","ado","Claims",null,"https://dev.azure.com/example/Claims/_workitems/edit/4521"]' ]
  run kit flow issues
  [ "$status" -eq 0 ]
  in-order "${lines[0]}" '#7' 'bob' '🛫' '#3' 'draft' 'Fix the thing'
  in-order "${lines[1]}" 'AB#4521' 'ada@example.com' '🛫' '#3' 'draft' 'Fix the claim form'
  in-order "${lines[2]}" 'AB#4522' '🛫' 'ab4522-start' 'Started'
  in-order "${lines[3]}" 'AB#4524' '💤' 'Later'
  [ "${#lines[@]}" -eq 4 ]
}

@test "the forge's own issues alone ask no tracker and no merged change, and draw as ever" {
  printf 'forge: github\n' > kit.yml
  run kit flow issues --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map([.ref, .tracker])' <<<"$output")" = '[["#7","#"]]' ]
  ! grep --quiet 'boards query' "${FAKE_DIR}/az.calls"
  ! grep --quiet 'MERGED' "${FAKE_DIR}/gh.calls"
}

@test "a tracker that cannot be read fails the stage, naming it; a web tracker lists none" {
  : > "${FAKE_DIR}/az.routes"
  route az 'boards query *' '!1'
  run kit flow issues --json
  [ "$status" -eq 1 ]
  [[ "$output" == *"az login (the work items of Claims could not be read from https://dev.azure.com/example)"* ]]
  [[ "$output" == *"the AB# tracker's open issues could not be read"* ]]
  cat > kit.yml <<'YAML'
forge: github
trackers:
  - prefix: "LIN#"
    provider: web
    url: https://linear.app/example/issue/ENG-{id}
YAML
  forge-forget
  run kit flow issues --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map(.ref)' <<<"$output")" = '["#7"]' ]
}

@test "the tracker seam's open-issues answers for the forge's own, as forge:open-issues does" {
  run bash -c 'source "${kit_root}/modules/bash/1st.sh" && use tracker && tracker:at "#" open-issues' _
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.ref' <<<"$output")" = "#7" ]
}
