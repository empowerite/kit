#!/usr/bin/env bats
# forge-conformance.bats - every forge provider answers the seam's calls in the same shape, through a fake of its tool
# ----------------------------------------------------------------------------------------------------------------------
#
# One world, seen through each forge's own API: the repository empowerite/example, the caller ada, issue 7 "Fix
# the thing" assigned to bob, and change 3 on branch 7-fix-the-thing, opened by ada, closing issue 7, with two
# checks that passed, the slower in fifty seconds, a failed run of one of them from the day before that a rerun
# superseded, and one unresolved review thread. Each provider is driven by tests/fakes/fake-tool
# standing in for its tool, so what is proven is the provider's own folding of its forge's answers into the seam's
# shape. What only a real forge can prove, that the API answers as the fixtures do, is not proven here.

load helper

setup() {
  fake-tools gh glab
  fresh-repo
}

# call <call> <argument>… runs one call of the seam, as a verb would.
call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && "forge:$0" "$@"' "$@"
}

# provider prints the provider the seam picks here.
provider() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge:provider'
}

# world <forge> makes this repository's origin the forge's, and its tool's routes the world above.
world() {
  git remote remove origin 2>/dev/null || true
  "world-$1"
}

world-github() {
  git remote add origin https://github.com/empowerite/example.git
  : > "${FAKE_DIR}/gh.routes"
  fixture user.json <<<'{"login": "ada"}'
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture issue.json <<<'{"title": "Fix the thing", "state": "OPEN", "assignees": [{"login": "bob"}]}'
  fixture open-heads.json <<<'[{"headRefName": "9-other"}, {"headRefName": "7-fix-the-thing"}]'
  fixture pr-open.json <<<'[{"number": 3}]'
  fixture pr-all.json <<<'[{"state": "OPEN"}]'
  fixture pr-url.txt <<<'https://github.com/empowerite/example/pull/3'
  fixture issue-url.txt <<<'https://github.com/empowerite/example/issues/7'
  fixture pr.json <<<'{"title": "Fix the thing", "body": "Closes #7"}'
  fixture pull.json <<<'{"user": {"login": "ada"}}'
  fixture closing.json <<'JSON'
{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": [
  {"number": 7, "state": "OPEN", "repository": {"nameWithOwner": "empowerite/example"}}]}}}}}
JSON
  fixture closing-none.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": []}}}}}'
  fixture open-prs.json <<'JSON'
{"data": {"repository": {"pullRequests": {"nodes": [{
  "number": 3, "title": "Fix the thing", "isDraft": false, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"},
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing", "createdAt": "2026-09-20T09:00:00Z",
                                         "assignees": {"nodes": [{"login": "bob"}]}}]},
  "reviewThreads": {"nodes": [{"isResolved": true}, {"isResolved": false}]},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": {"contexts": {"nodes": [
    {"__typename": "CheckRun", "name": "a", "status": "COMPLETED", "conclusion": "FAILURE",
     "startedAt": "2026-09-30T10:00:00Z", "completedAt": "2026-09-30T10:05:00Z"},
    {"__typename": "CheckRun", "name": "a", "status": "COMPLETED", "conclusion": "SUCCESS",
     "startedAt": "2026-10-01T10:00:00Z", "completedAt": "2026-10-01T10:00:30Z"},
    {"__typename": "CheckRun", "name": "b", "status": "COMPLETED", "conclusion": "SUCCESS",
     "startedAt": "2026-10-01T10:00:10Z", "completedAt": "2026-10-01T10:01:00Z"}]}}}}]}}]}}}}
JSON
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'issue view 7 *' issue.json
  route gh 'issue create*' issue-url.txt
  fixture issues.json <<'JSON'
[{"number": 9, "title": "Other thing", "assignees": [], "createdAt": "2026-09-25T09:00:00Z"},
 {"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z"}]
JSON
  route gh 'issue edit 7 --add-assignee ada' '!0'
  route gh 'issue list --state open --limit 1000 --json *' issues.json
  route gh 'pr list --state open --json headRefName*' open-heads.json
  route gh 'pr list --head 7-fix-the-thing --state open*' pr-open.json
  route gh 'pr list --head 7-fix-the-thing --state all*' pr-all.json
  route gh 'pr create --draft --base main --head 7-fix-the-thing --title Fix the thing*' pr-url.txt
  route gh 'pr view 3 --json *' pr.json
  route gh 'api repos/empowerite/example/pulls/3 *' pull.json
  route gh 'api graphql*number=3*' "${GITHUB_CLOSING:-closing.json}"
  route gh 'api graphql*' open-prs.json
}

world-gitlab() {
  git remote add origin git@gitlab.com:empowerite/example.git
  : > "${FAKE_DIR}/glab.routes"
  local p='projects/empowerite%2Fexample'
  fixture user.json <<<'{"username": "ada"}'
  fixture issue.json <<<'{"iid": 7, "title": "Fix the thing", "state": "opened", "assignees": [{"username": "bob"}]}'
  fixture mrs-open.json <<<'[{"iid": 2, "source_branch": "9-other"}, {"iid": 3, "source_branch": "7-fix-the-thing"}]'
  fixture mrs-list.json <<<'[{"iid": 3}]'
  fixture mr-by-branch.json <<<'[{"iid": 3}]'
  fixture mr-states.json <<<'[{"state": "opened"}]'
  fixture mr-url.txt <<<'https://gitlab.com/empowerite/example/-/merge_requests/3'
  fixture issue-url.txt <<<'https://gitlab.com/empowerite/example/-/issues/7'
  fixture mr.json <<'JSON'
{"iid": 3, "title": "Fix the thing", "draft": false, "source_branch": "7-fix-the-thing",
 "created_at": "2026-10-01T09:00:00.000Z", "author": {"username": "ada"}, "has_conflicts": false,
 "detailed_merge_status": "mergeable", "description": "Closes #7", "head_pipeline": {"id": 11, "status": "success"}}
JSON
  fixture jobs.json <<'JSON'
[{"name": "a", "status": "failed",
  "started_at": "2026-09-30T10:00:00.000Z", "finished_at": "2026-09-30T10:05:00.000Z"},
 {"name": "a", "status": "success",
  "started_at": "2026-10-01T10:00:00.120Z", "finished_at": "2026-10-01T10:00:30.000Z"},
 {"name": "b", "status": "success",
  "started_at": "2026-10-01T10:00:10.000Z", "finished_at": "2026-10-01T10:01:00.450Z"}]
JSON
  fixture discussions.json <<'JSON'
[{"notes": [{"resolvable": true, "resolved": true}]}, {"notes": [{"resolvable": true, "resolved": false}]},
 {"notes": [{"resolvable": false, "resolved": false}]}]
JSON
  fixture closes.json <<'JSON'
[{"iid": 7, "title": "Fix the thing", "state": "opened", "references": {"full": "empowerite/example#7"},
  "created_at": "2026-09-20T09:00:00.000Z", "assignees": [{"username": "bob"}]}]
JSON
  fixture none.json <<<'[]'
  route glab 'api user' user.json
  route glab "api ${p}/issues/7" issue.json
  route glab 'issue create*' issue-url.txt
  fixture issues.json <<'JSON'
[{"iid": 7, "title": "Fix the thing", "assignees": [{"username": "bob"}], "created_at": "2026-09-20T09:00:00.000Z"},
 {"iid": 9, "title": "Other thing", "assignees": [], "created_at": "2026-09-25T09:00:00.500Z"}]
JSON
  route glab 'issue update 7 --assignee +ada' '!0'
  route glab "api ${p}/issues?state=opened&per_page=100&order_by=created_at&sort=asc" issues.json
  route glab "api ${p}/merge_requests?state=opened&per_page=100&order_by=*" mrs-list.json
  route glab "api ${p}/merge_requests?state=opened&per_page=100" mrs-open.json
  route glab "api ${p}/merge_requests?state=opened&source_branch=7-fix-the-thing" mr-by-branch.json
  route glab "api ${p}/merge_requests?state=all&source_branch=7-fix-the-thing*" mr-states.json
  route glab "api ${p}/merge_requests/3/closes_issues?per_page=100" "${GITLAB_CLOSES:-closes.json}"
  route glab "api ${p}/merge_requests/3/discussions*" discussions.json
  route glab "api ${p}/merge_requests/3" mr.json
  route glab "api ${p}/pipelines/11/jobs*" jobs.json
  route glab "api ${p}/repository/branches --method POST*" '!0'
  route glab 'mr create --draft --source-branch 7-fix-the-thing --target-branch main --title Fix the thing*' mr-url.txt
}

@test "the forge is picked by origin's host, github.com or gitlab.com, and either answers the same plain calls" {
  for forge in github gitlab
  do
    world "${forge}"
    run provider
    [ "$output" = "${forge}" ]
    run call repo
    [ "$output" = "empowerite/example" ]
    run call me
    [ "$output" = "ada" ]
    run call issue-title 7
    [ "$output" = "Fix the thing" ]
    run call issue-state 7
    [ "$output" = "OPEN" ]
    run call issue-assignees 7
    [ "$output" = "bob" ]
    run call issue-create "Fix the thing"
    [ "$output" = "7" ]
    run call issue-assign 7 ada
    [ "$status" -eq 0 ]
  done
}

@test "every provider finds, opens, and reads a change by its branch the same way" {
  for forge in github gitlab
  do
    world "${forge}"
    run call branch-for-issue 7
    [ "$output" = "7-fix-the-thing" ]
    run call branch-for-issue 8
    [ -z "$output" ]
    run call pr-for-branch 7-fix-the-thing
    [ "$output" = "3" ]
    run call pr-state-for-branch 7-fix-the-thing
    [ "$output" = "OPEN" ]
    run call pr-create 7-fix-the-thing main "Fix the thing"
    [ "$output" = "3" ]
    run call pr-closing-issues 3
    [ "$output" = "7 OPEN empowerite/example" ]
    run call pr-title 3
    [ "$output" = "Fix the thing" ]
    run call pr-body 3
    [ "$output" = "Closes #7" ]
    run call pr-author 3
    [ "$output" = "ada" ]
  done
}

@test "every provider folds its open changes into the one document kit flow draws" {
  expected='{"number":3,"branch":"7-fix-the-thing","title":"Fix the thing","author":"ada",'
  expected+='"state":"comments:open","checks":{"state":"SUCCESS","passed":2,"total":2,"failing":"","took":50},'
  expected+='"unresolved":1,"closes":[{"number":7,"title":"Fix the thing","assignees":["bob"]}]}'
  for forge in github gitlab
  do
    world "${forge}"
    run call open-prs
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys 'del(.age, .createdAt, .ref) | .closes |= map(del(.age))' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
    [ "$(jq '.closes[0].age > 0' <<<"$output")" = true ]
  done
}

@test "a change that closes more issues than a page holds is drawn with every one: each provider asks for a hundred" {
  for forge in github gitlab
  do
    world "${forge}"
    if [[ "${forge}" == github ]]
    then
      jq '.data.repository.pullRequests.nodes[0].closingIssuesReferences.nodes
            = [range(1; 13) | {number: ., title: "Issue \(.)"}]' "${FAKE_DIR}/open-prs.json" > "${FAKE_DIR}/many.json"
      mv "${FAKE_DIR}/many.json" "${FAKE_DIR}/open-prs.json"
    else
      jq --null-input '[range(1; 13) | {iid: ., title: "Issue \(.)", state: "opened",
                                        references: {full: "empowerite/example#\(.)"}}]' > "${FAKE_DIR}/closes.json"
    fi
    run call open-prs
    [ "$status" -eq 0 ]
    [ "$(jq '.closes | length' <<<"$output")" -eq 12 ]
  done
  grep --quiet 'closingIssuesReferences(first: 100)' "${FAKE_DIR}/gh.calls"
  grep --quiet 'merge_requests/3/closes_issues?per_page=100' "${FAKE_DIR}/glab.calls"
}

@test "each provider names the change in its own words, and the rest of the picture is identical" {
  world github
  run call noun
  [ "$output" = "$(printf '#\tpull request')" ]
  run call pr-url 3
  [ "$output" = "https://github.com/empowerite/example/pull/3" ]
  [ "$(call open-prs | jq --raw-output .ref)" = "#3" ]
  world gitlab
  run call noun
  [ "$output" = "$(printf '!\tmerge request')" ]
  run call pr-url 3
  [ "$output" = "https://gitlab.com/empowerite/example/-/merge_requests/3" ]
  [ "$(call open-prs | jq --raw-output .ref)" = "!3" ]
}

@test "GitLab makes the branch for an issue as its issue page would, and a self-hosted GitLab is the setting" {
  world gitlab
  run call link-branch 7 main
  [ "$status" -eq 0 ]
  grep --quiet -- '--raw-field branch=7-fix-the-thing --raw-field ref=main' "${FAKE_DIR}/glab.calls"
  git remote set-url origin https://git.example.com/empowerite/example.git
  run provider
  [ "$status" -eq 1 ]
  mkdir .kit && printf 'forge: gitlab\n' > kit.yml
  run provider
  [ "$output" = "gitlab" ]
  run call repo-url
  [ "$output" = "https://git.example.com/empowerite/example" ]
}

@test "kit linked speaks the forge's words: a merge request on GitLab, its URL in the remedy" {
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
  GITLAB_CLOSES=none.json world gitlab
  run "${kit_root}/bin/kit" linked 3
  [ "$status" -eq 1 ]
  [[ "$output" == *"merge request !3 is not linked to any issue"* ]]
  grep --quiet 'https://gitlab.com/empowerite/example/-/merge_requests/3' "${GITHUB_STEP_SUMMARY}"
  world gitlab
  run "${kit_root}/bin/kit" linked 3
  [ "$status" -eq 0 ]
  [[ "$output" == *"merge request !3 is linked to #7"* ]]
}

@test "kit linked passes a change by a login the repository exempts, naming the exemption, and checks any other" {
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
  for forge in github gitlab
  do
    GITHUB_CLOSING=closing-none.json GITLAB_CLOSES=none.json world "${forge}"
    mkdir -p .kit && printf 'linked_exempt:\n  - ada\n' > kit.yml
    : > "${GITHUB_STEP_SUMMARY}"
    run "${kit_root}/bin/kit" linked 3
    [ "$status" -eq 0 ]
    [[ "$output" == *"3 is ada's, exempt from the linked-issue check by kit.yml"* ]]
    grep --quiet "✅ .*3 is ada's, exempt" "${GITHUB_STEP_SUMMARY}"
    printf 'linked_exempt:\n  - bob\n' > kit.yml
    run "${kit_root}/bin/kit" linked 3
    [ "$status" -eq 1 ]
    [[ "$output" == *"3 is not linked to any issue"* ]]
    rm -r .kit
  done
}

@test "every provider lists its open issues in one shape, oldest first, with the assignees by login" {
  expected='[{"number":7,"ref":"#7","title":"Fix the thing","assignees":["bob"]},'
  expected+='{"number":9,"ref":"#9","title":"Other thing","assignees":[]}]'
  for forge in github gitlab
  do
    world "${forge}"
    run call open-issues
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys --slurp 'map(del(.age, .createdAt))' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
  done
}

@test "every provider reaches its tool with color forced off, so a desk's CLICOLOR_FORCE cannot color its data" {
  for forge in github gitlab
  do
    world "${forge}"
    CLICOLOR_FORCE=1 GH_FORCE_TTY=1 run call open-issues
    [ "$status" -eq 0 ]
    [ "$(jq --slurp --compact-output 'map(.number)' <<<"$output")" = "[7,9]" ]
  done
}

@test "every provider lists its releases in one shape, newest first, with the kind and the latest" {
  fixture gh-releases.json <<'JSON'
[{"tagName": "release/0.0.9", "name": "kit 0.0.9", "isPrerelease": false, "isDraft": false, "isLatest": true,
  "publishedAt": "2026-10-20T07:30:14Z", "createdAt": "2026-10-20T07:30:14Z"},
 {"tagName": "release/0.1.0-rc.6", "name": "kit 0.1.0-rc.6", "isPrerelease": true, "isDraft": false, "isLatest": false,
  "publishedAt": "2026-10-27T07:30:14Z", "createdAt": "2026-10-27T07:30:14Z"}]
JSON
  fixture gl-releases.json <<'JSON'
[{"tag_name": "release/0.1.0-rc.6", "name": "kit 0.1.0-rc.6", "released_at": "2026-10-27T07:30:14.000Z",
  "upcoming_release": false},
 {"tag_name": "release/0.0.9", "name": "kit 0.0.9", "released_at": "2026-10-20T07:30:14.000Z",
  "upcoming_release": false}]
JSON
  expected='[{"version":"0.1.0-rc.6","tag":"release/0.1.0-rc.6","name":"kit 0.1.0-rc.6","kind":"pre-release","latest":false},'
  expected+='{"version":"0.0.9","tag":"release/0.0.9","name":"kit 0.0.9","kind":"release","latest":true}]'
  for forge in github gitlab
  do
    world "${forge}"
    route gh 'release list --limit 1000 --json *' gh-releases.json
    route glab 'api projects/empowerite%2Fexample/releases?per_page=100' gl-releases.json
    run call releases
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys --slurp 'map(del(.age, .at))' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected}")" ] || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq --slurp 'map(.age > 0) | all' <<<"$output")" = true ]
  done
}

# runs-world <forge> is world's, with the runs: their list, and the jobs of run 103.
runs-world() {
  world "$1"
  route gh 'run list --limit 50 --json *' gh-runs.json
  route gh 'run view 103 --json jobs' gh-jobs.json
  route glab 'api projects/empowerite%2Fexample/pipelines?per_page=50' gl-pipelines.json
}

@test "every provider lists the runs no pull request owns in one shape, newest first, the newest with its jobs" {
  git commit --quiet --allow-empty --message one && git tag release/0.1.0
  fixture gh-runs.json <<'JSON'
[{"databaseId": 104, "number": 50, "headSha": "aaaaaaa0000000000", "displayTitle": "title 104",
  "workflowName": "kit", "headBranch": "7-fix-the-thing", "event": "pull_request",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T11:00:00Z",
  "startedAt": "2026-10-01T11:00:00Z", "updatedAt": "2026-10-01T11:03:00Z", "url": "https://github.com/r/104"},
 {"databaseId": 99, "number": 45, "headSha": "5d376dd0000000000", "displayTitle": "title 99",
  "workflowName": "kit", "headBranch": "9-other", "event": "workflow_dispatch",
  "status": "completed", "conclusion": "cancelled", "createdAt": "2026-10-01T06:00:00Z",
  "startedAt": "2026-10-01T06:00:00Z", "updatedAt": "2026-10-01T06:00:42Z", "url": "https://github.com/r/99"},
 {"databaseId": 103, "number": 49, "headSha": "8e817bf0000000000", "displayTitle": "title 103",
  "workflowName": "kit", "headBranch": "main", "event": "push",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T10:00:00Z",
  "startedAt": "2026-10-01T10:00:00Z", "updatedAt": "2026-10-01T10:03:36Z", "url": "https://github.com/r/103"},
 {"databaseId": 102, "number": 48, "headSha": "1a2b3c40000000000", "displayTitle": "title 102",
  "workflowName": "kit", "headBranch": "release/0.1.0", "event": "push",
  "status": "completed", "conclusion": "failure", "createdAt": "2026-10-01T09:00:00Z",
  "startedAt": "2026-10-01T09:00:00Z", "updatedAt": "2026-10-01T09:04:02Z", "url": "https://github.com/r/102"},
 {"databaseId": 101, "number": 47, "headSha": "8e817bf0000000000", "displayTitle": "title 101",
  "workflowName": "kit", "headBranch": "main", "event": "schedule",
  "status": "in_progress", "conclusion": "", "createdAt": "2026-10-01T08:00:00Z",
  "startedAt": "2026-10-01T08:00:00Z", "updatedAt": "2026-10-01T08:01:00Z", "url": "https://github.com/r/101"},
 {"databaseId": 100, "number": 46, "headSha": "bbbbbbb0000000000", "displayTitle": "title 100",
  "workflowName": "kit", "headBranch": "9-other", "event": "push",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T07:00:00Z",
  "startedAt": "2026-10-01T07:00:00Z", "updatedAt": "2026-10-01T07:01:00Z", "url": "https://github.com/r/100"}]
JSON
  fixture gh-jobs.json <<'JSON'
{"jobs": [{"name": "ci", "status": "completed", "conclusion": "success",
           "startedAt": "2026-10-01T10:00:03Z", "completedAt": "2026-10-01T10:03:33Z", "steps": []},
          {"name": "lint", "status": "in_progress", "conclusion": "",
           "startedAt": "2026-10-01T10:00:03Z", "completedAt": "0001-01-01T00:00:00Z"}]}
JSON
  fixture gl-pipelines.json <<'JSON'
[{"id": 104, "iid": 50, "sha": "aaaaaaa0000000000",
  "name": "kit", "ref": "7-fix-the-thing", "source": "merge_request_event", "status": "success",
  "created_at": "2026-10-01T11:00:00.000Z", "updated_at": "2026-10-01T11:03:00.000Z",
  "web_url": "https://gl.com/p/104"},
 {"id": 103, "iid": 49, "sha": "8e817bf0000000000",
  "name": "kit", "ref": "main", "source": "push", "status": "success",
  "created_at": "2026-10-01T10:00:00.000Z", "updated_at": "2026-10-01T10:03:36.000Z",
  "web_url": "https://gl.com/p/103"},
 {"id": 102, "iid": 48, "sha": "1a2b3c40000000000",
  "name": "kit", "ref": "release/0.1.0", "source": "push", "status": "failed", "tag": true,
  "created_at": "2026-10-01T09:00:00.000Z", "updated_at": "2026-10-01T09:04:02.000Z",
  "web_url": "https://gl.com/p/102"},
 {"id": 101, "iid": 47, "sha": "8e817bf0000000000",
  "name": "kit", "ref": "main", "source": "schedule", "status": "running",
  "created_at": "2026-10-01T08:00:00.000Z", "updated_at": "2026-10-01T08:01:00.000Z",
  "web_url": "https://gl.com/p/101"},
 {"id": 100, "iid": 46, "sha": "bbbbbbb0000000000",
  "name": "kit", "ref": "9-other", "source": "push", "status": "success",
  "created_at": "2026-10-01T07:00:00.000Z", "updated_at": "2026-10-01T07:01:00.000Z",
  "web_url": "https://gl.com/p/100"},
 {"id": 99, "iid": 45, "sha": "5d376dd0000000000",
  "name": "kit", "ref": "9-other", "source": "web", "status": "canceled",
  "created_at": "2026-10-01T06:00:00.000Z", "updated_at": "2026-10-01T06:00:42.000Z",
  "web_url": "https://gl.com/p/99"}]
JSON
  expected='[{"id":103,"number":49,"attempt":1,"workflow":"kit","branch":"main","tag":false,"event":"push",'
  expected+='"sha":"8e817bf","state":"passed","took":216},'
  expected+='{"id":102,"number":48,"attempt":1,"workflow":"kit","branch":"release/0.1.0","tag":true,"event":"push",'
  expected+='"sha":"1a2b3c4","state":"failed","took":242},'
  expected+='{"id":101,"number":47,"attempt":1,"workflow":"kit","branch":"main","tag":false,"event":"schedule",'
  expected+='"sha":"8e817bf","state":"running","took":null},'
  expected+='{"id":99,"number":45,"attempt":1,"workflow":"kit","branch":"9-other","tag":false,'
  expected+='"event":"workflow_dispatch","sha":"5d376dd","state":"cancelled","took":42}]'
  for forge in github gitlab
  do
    runs-world "${forge}"
    run call runs
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys 'map(del(.age, .url, .jobs, .title))' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected}")" ] || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq 'map(.age > 0 and (.url | startswith("https://"))) | all' <<<"$output")" = true ]
  done
  # GitLab's list gives no title, nor jobs; GitHub's runs carry their title, and the newest their jobs, a call each
  [ "$(call runs | jq --compact-output 'map([.title, .jobs])')" = '[["",[]],["",[]],["",[]],["",[]]]' ]
  runs-world github
  [ "$(call runs | jq --compact-output 'map(.title)')" = '["title 103","title 102","title 101","title 99"]' ]
  : > "${FAKE_DIR}/gh.calls"
  [ "$(call runs | jq --compact-output '.[0].jobs')" \
    = '[{"name":"ci","state":"passed","took":210,"step":null},'"\
"'{"name":"lint","state":"running","took":null,"step":null}]' ]
  [ "$(grep -c '^run view ' "${FAKE_DIR}/gh.calls")" -eq 4 ]
}

# ----------------------------------------------------------------------------------------------------------------------
