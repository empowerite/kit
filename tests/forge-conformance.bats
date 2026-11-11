#!/usr/bin/env bats
# forge-conformance.bats - every forge provider answers the seam's calls in the same shape, through a fake of its tool
# ----------------------------------------------------------------------------------------------------------------------
# One world, seen through each forge's own API: the repository empowerite/example, the caller ada, issue 7 "Fix
# the thing" assigned to bob, and change 3 on branch 7-fix-the-thing, opened by ada, closing issue 7, with two
# checks that passed, the slower in fifty seconds, a failed run of one of them from the day before that a rerun
# superseded, and one unresolved review thread. Each provider is driven by tests/fakes/fake-tool
# standing in for its tool, so what is proven is the provider's own folding of its forge's answers into the seam's
# shape. What only a real forge can prove, that the API answers as the fixtures do, is not proven here.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh glab
  fresh-repo
}

# call <call> <argument>… runs one call of the seam, as a verb would.
call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && "forge:$0" "$@"' "$@"
}

# fresh-call <call> <argument>… runs one call of the seam as a stage given --fresh would: after forge:fresh.
fresh-call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge:fresh && "forge:$0" "$@"' "$@"
}

# reads <forge> prints how many times the provider's tool was asked for the open changes.
reads() {
  if [[ "$1" == github ]]
  then grep --count '^api graphql' "${FAKE_DIR}/gh.calls"
  else grep --count 'merge_requests?state=opened&per_page=100&order_by=' "${FAKE_DIR}/glab.calls"
  fi
}

# snapshot <forge> <call> prints the path of the call's snapshot, which forge:snapshot keeps under the common git
# directory.
snapshot() {
  printf '%s\n' "$(git rev-parse --path-format=absolute --git-common-dir)"/kit/forge/"$1"-*/"$2.json"
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
  fixture pr-all.json <<<'[{"state": "OPEN", "headRefOid": "1f2e3d4"}]'
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
{"data": {"repository": {"nameWithOwner": "empowerite/example",
  "issues": {"nodes": [{"number": 7, "assignees": {"nodes": [{"login": "bob"}]}}]},
  "pullRequests": {"nodes": [{
  "number": 3, "title": "Fix the thing", "isDraft": false, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"}, "labels": {"nodes": [{"name": "kind=fix", "color": "0E8A16"}]},
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing", "createdAt": "2026-09-20T09:00:00Z",
                                         "repository": {"nameWithOwner": "empowerite/example"},
                                         "labels": {"nodes": [{"name": "bug", "color": "D73A4A"}]}}]},
  "reviewRequests": {"totalCount": 1, "nodes": [{"requestedReviewer": {"login": "bob"}}]},
  "latestReviews": {"totalCount": 2, "nodes": [
    {"author": {"login": "ada"}, "state": "COMMENTED", "submittedAt": "2026-10-01T11:00:00Z"},
    {"author": {"login": "carol"}, "state": "APPROVED", "submittedAt": "2026-10-02T09:00:00Z"}]},
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
[{"number": 9, "title": "Other thing", "assignees": [], "createdAt": "2026-09-25T09:00:00Z",
  "url": "https://github.com/empowerite/example/issues/9"},
 {"number": 7, "title": "Fix the thing", "assignees": [{"login": "bob"}], "createdAt": "2026-09-20T09:00:00Z",
  "labels": [{"id": "LA_1", "name": "bug", "description": "", "color": "d73a4a"}],
  "url": "https://github.com/empowerite/example/issues/7"}]
JSON
  route gh 'issue edit 7 --add-assignee ada' '!0'
  route gh 'issue list --state open --limit 10000 --json *' issues.json
  route gh 'pr list --state open --json headRefName*' open-heads.json
  route gh 'pr list --head 7-fix-the-thing --state open*' pr-open.json
  route gh 'pr list --head 7-fix-the-thing --state all*' pr-all.json
  route gh 'pr create --draft --base main --head 7-fix-the-thing --title Fix the thing*' pr-url.txt
  route gh 'pr view 3 --json *' pr.json
  route gh 'api repos/empowerite/example/pulls/3 *' pull.json
  fixture gh-milestones.json <<'JSON'
[[{"number": 2, "title": "v0.1", "state": "open", "due_on": "2026-12-01T00:00:00Z", "open_issues": 3,
   "closed_issues": 2, "html_url": "https://github.com/empowerite/example/milestone/2"}]]
JSON
  fixture gh-iterations.json <<'JSON'
{"data": {"repository": {"projectsV2": {"totalCount": 1, "nodes": [{"url": "https://github.com/orgs/empowerite/projects/4",
  "fields": {"totalCount": 2, "nodes": [{}, {"configuration": {
    "iterations": [{"id": "PVTI_2", "title": "Sprint 2", "startDate": "2026-11-16", "duration": 14}],
    "completedIterations": [{"id": "PVTI_1", "title": "Sprint 1", "startDate": "2026-11-02", "duration": 14}]}}]}}]}}}}
JSON
  fixture gh-iterations-none.json <<<'{"data": {"repository": {"projectsV2": {"nodes": []}}}}'
  route gh 'api --paginate --slurp repos/{owner}/{repo}/milestones?state=all&per_page=100' gh-milestones.json
  delivery-github
  fixture gh-merged.json <<'JSON'
{"data": {"repository": {"pullRequests": {"totalCount": 1, "nodes": [
  {"number": 2, "title": "Do the other thing", "url": "https://github.com/empowerite/example/pull/2",
   "headRefName": "9-other", "mergedAt": "2026-10-03T09:00:00Z", "updatedAt": "2026-10-03T09:00:00Z",
   "author": {"login": "bob"},
   "mergedBy": {"login": "ada"}}]}}}}
JSON
  route gh 'api graphql*states: MERGED*' "${GITHUB_MERGED:-gh-merged.json}"
  route gh 'api graphql*deployments(first: 50*' "${GITHUB_DEPLOYMENTS:-gh-deployments.json}"
  route gh 'api graphql*projectsV2*' "${GITHUB_ITERATIONS:-gh-iterations.json}"
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
  fixture mr-states.json <<<'[{"state": "opened", "sha": "1f2e3d4"}]'
  fixture mr-url.txt <<<'https://gitlab.com/empowerite/example/-/merge_requests/3'
  fixture issue-url.txt <<<'https://gitlab.com/empowerite/example/-/issues/7'
  fixture mr.json <<'JSON'
{"iid": 3, "title": "Fix the thing", "draft": false, "source_branch": "7-fix-the-thing",
 "created_at": "2026-10-01T09:00:00.000Z", "author": {"username": "ada"}, "has_conflicts": false,
 "detailed_merge_status": "mergeable", "description": "Closes #7", "source_project_id": 1, "target_project_id": 1,
 "head_pipeline": {"id": 11, "project_id": 1, "status": "success"}, "labels": ["kind=fix"],
 "reviewers": [{"username": "bob"}, {"username": "carol"}]}
JSON
  fixture approvals.json <<'JSON'
{"approved": true, "approved_by": [{"user": {"username": "carol"}, "approved_at": "2026-10-02T09:00:00.000Z"}]}
JSON
  fixture gl-merged.json <<'JSON'
[{"iid": 2, "title": "Do the other thing", "author": {"username": "bob"}, "source_branch": "9-other",
  "merged_at": "2026-10-03T09:00:00.000Z", "merge_user": {"username": "ada"}, "merged_by": {"username": "ada"},
  "web_url": "https://gitlab.com/empowerite/example/-/merge_requests/2"}]
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
  "created_at": "2026-09-20T09:00:00.000Z", "assignees": [{"username": "bob"}], "labels": ["bug"]}]
JSON
  fixture labels.json <<<'[{"name": "bug", "color": "#D73A4A"}, {"name": "kind=fix", "color": "#0E8A16"}]'
  fixture none.json <<<'[]'
  route glab 'api user' user.json
  fixture gl-milestones.json <<'JSON'
{"data": {"project": {"milestones": {"pageInfo": {"hasNextPage": false, "endCursor": null}, "nodes": [
  {"iid": "2", "title": "v0.1", "state": "active", "dueDate": "2026-12-01",
   "webPath": "/empowerite/example/-/milestones/2", "stats": {"totalIssuesCount": 5, "closedIssuesCount": 2}}]}}}}
JSON
  fixture gl-iterations.json <<'JSON'
[{"id": 31, "iid": 1, "title": "Sprint 1", "start_date": "2026-11-02", "due_date": "2026-11-15",
  "web_url": "https://gitlab.com/groups/empowerite/-/iterations/31"}]
JSON
  fixture gl-forbidden.txt <<<'glab: 403 Forbidden'
  route glab 'api graphql*' "${GITLAB_MILESTONES:-gl-milestones.json}"
  route glab "api ${p}/iterations?include_ancestors=true&per_page=100*" "${GITLAB_ITERATIONS:-gl-iterations.json}"
  route glab "api ${p}/issues/7" issue.json
  route glab 'issue create*' issue-url.txt
  fixture issues.json <<'JSON'
[{"iid": 7, "title": "Fix the thing", "assignees": [{"username": "bob"}], "created_at": "2026-09-20T09:00:00.000Z",
  "labels": [{"id": 4, "name": "bug", "color": "#D73A4A", "text_color": "#FFFFFF"}],
  "references": {"full": "empowerite/example#7"}},
 {"iid": 9, "title": "Other thing", "assignees": [], "created_at": "2026-09-25T09:00:00.500Z",
  "references": {"full": "empowerite/example#9"}}]
JSON
  route glab 'issue update 7 --assignee +ada' '!0'
  route glab "api ${p}/issues?state=opened&per_page=100&order_by=created_at&sort=asc&with_labels_details=true*" \
    issues.json
  route glab "api ${p}/labels?per_page=100*" labels.json
  route glab "api ${p}/merge_requests?state=opened&per_page=100&order_by=*" mrs-list.json
  route glab "api ${p}/merge_requests?state=opened&per_page=100*" mrs-open.json
  route glab "api ${p}/merge_requests?state=opened&source_branch=7-fix-the-thing" mr-by-branch.json
  route glab "api ${p}/merge_requests?state=all&source_branch=7-fix-the-thing*" mr-states.json
  route glab "api ${p}/merge_requests/3/closes_issues?per_page=100*" "${GITLAB_CLOSES:-closes.json}"
  route glab "api ${p}/merge_requests/3/discussions*" discussions.json
  route glab "api ${p}/merge_requests/3/approvals" "${GITLAB_APPROVALS:-approvals.json}"
  route glab "api ${p}/merge_requests?state=merged&order_by=merged_at&per_page=51" "${GITLAB_MERGED:-gl-merged.json}"
  delivery-gitlab
  route glab "api ${p}/merge_requests/3" mr.json
  route glab 'api projects/1/pipelines/11/jobs*' jobs.json
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

@test "every provider unassigns, closes, comments on, and nests issues, and reads a change's state, in its own tool's words" {
  world github
  route gh 'issue edit 7 --remove-assignee bob' '!0'
  route gh 'issue edit 7 --add-sub-issue 9' '!0'
  route gh 'issue close 7 *' '!0'
  route gh 'issue comment 7 *' '!0'
  fixture pr-state.json <<<'{"state": "MERGED", "mergeCommit": {"oid": "abc123"}}'
  route gh 'pr view 5 --json state,mergeCommit*' pr-state.json
  call issue-unassign 7 bob
  call issue-close 7 not-planned
  call issue-close 7 duplicate 9
  call issue-comment 7 "Moved on"
  call issue-add-sub 7 9
  run call pr-state 5
  [ "$output" = $'MERGED\tabc123' ]
  run grep '^issue ' "${FAKE_DIR}/gh.calls"
  [ "$output" = $'issue edit 7 --remove-assignee bob\nissue close 7 --reason not planned\nissue close 7 --reason duplicate --duplicate-of 9\nissue comment 7 --body Moved on\nissue edit 7 --add-sub-issue 9' ]
  world gitlab
  local p='projects/empowerite%2Fexample'
  route glab 'issue update 7 --assignee -bob' '!0'
  route glab 'issue close 7' '!0'
  route glab 'issue note 7 *' '!0'
  route glab "api ${p}/issues/7/links --method POST*" '!0'
  fixture mr-merged.json <<<'{"iid": 3, "state": "merged", "merge_commit_sha": "def456"}'
  route glab "api ${p}/merge_requests/4" mr-merged.json
  : > "${FAKE_DIR}/glab.calls"
  call issue-unassign 7 bob
  call issue-close 7 not-planned
  call issue-close 7 duplicate 9
  call issue-comment 7 "Moved on"
  call issue-add-sub 7 9
  run call pr-state 4
  [ "$output" = $'MERGED\tdef456' ]
  run grep -v '^api user\|^api projects/empowerite%2Fexample/merge_requests/4$' "${FAKE_DIR}/glab.calls"
  [ "$output" = $'issue update 7 --assignee -bob\nissue close 7\nissue note 7 --message /duplicate #9\nissue note 7 --message Moved on\napi projects/empowerite%2Fexample/issues/7/links --method POST --raw-field target_project_id=empowerite/example --raw-field target_issue_iid=9' ]
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
    run call pr-head-for-branch 7-fix-the-thing
    [ "$output" = "1f2e3d4" ]
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
  expected='{"number":3,"branch":"7-fix-the-thing","fork":null,"title":"Fix the thing","author":"ada","pushed":null,'
  expected+='"state":"comments:open","checks":{"state":"SUCCESS","passed":2,"total":2,"failing":"","took":50},'
  expected+='"unresolved":1,"labels":[{"name":"kind=fix","color":"#0e8a16"}],'
  expected+='"closes":[{"number":7,"title":"Fix the thing","assignees":["bob"],'
  expected+='"labels":[{"name":"bug","color":"#d73a4a"}]}],'
  # the author's own comment is no review; bob is asked and has not answered; carol approved
  expected+='"reviewers":[{"login":"bob","state":"requested","at":null},'
  expected+='{"login":"carol","state":"approved","at":1790931600}],'
  # every run on the head commit, the failed one a rerun superseded among them
  expected+='"runs":[{"name":"a","verdict":"failed","at":1790762400,"took":300},'
  expected+='{"name":"a","verdict":"passed","at":1790848800,"took":30},'
  expected+='{"name":"b","verdict":"passed","at":1790848810,"took":50}]}'
  for forge in github gitlab
  do
    world "${forge}"
    run call open-prs
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys \
      'del(.age, .createdAt, .ref) | .closes |= map(del(.age, .source, .organization, .repository))' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
    [ "$(jq '.closes[0].age > 0' <<<"$output")" = true ]
    # each issue a change closes names the tracker that holds it, the forge's own, and where in it the issue is
    [ "$(jq --raw-output '[.closes[].source] | unique | join(" ")' <<<"$output")" = "${forge}" ]
    [ "$(jq --raw-output '[.closes[] | "\(.organization)/\(.repository)"] | unique | join(" ")' <<<"$output")" \
      = empowerite/example ]
  done
}

@test "every provider names a fork's owner and the last push, the same way, and reads a fork's project once" {
  local pushed
  pushed=$(jq --null-input --raw-output 'now - 7200 | todate')
  for forge in github gitlab
  do
    world "${forge}"
    if [[ "${forge}" == github ]]
    then
      jq --arg at "${pushed}" '.data.repository.pullRequests.nodes[0]
            |= (.isCrossRepository = true | .headRepositoryOwner = {login: "cy"}
                | .commits.nodes[0].commit += {committedDate: $at, author: {user: {login: "dee"}}})' \
        "${FAKE_DIR}/open-prs.json" > "${FAKE_DIR}/fork.json"
      mv "${FAKE_DIR}/fork.json" "${FAKE_DIR}/open-prs.json"
    else
      # The same merge request listed twice stands for two from one fork.
      fixture mrs-list.json <<<'[{"iid": 3}, {"iid": 3}]'
      fixture project.json <<<'{"id": 99, "namespace": {"full_path": "cy"}}'
      jq --arg at "${pushed}" '. + {source_project_id: 99, target_project_id: 1}
            | .head_pipeline += {created_at: $at, user: {username: "dee"}}' "${FAKE_DIR}/mr.json" \
        > "${FAKE_DIR}/fork.json"
      mv "${FAKE_DIR}/fork.json" "${FAKE_DIR}/mr.json"
      route glab 'api projects/99' project.json
    fi
    run call open-prs
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output '[.fork, .pushed.by, (.pushed.age >= 7200 and .pushed.age < 7300)]' <<<"${lines[0]}")" \
      = '["cy","dee",true]' ] || { echo "${forge}: ${output}"; return 1; }
  done
  grep --quiet 'isCrossRepository' "${FAKE_DIR}/gh.calls"
  grep --quiet 'headRepositoryOwner { login }' "${FAKE_DIR}/gh.calls"
  grep --quiet 'committedDate author { user { login } }' "${FAKE_DIR}/gh.calls"
  [ "$(grep --count '^api projects/99$' "${FAKE_DIR}/glab.calls")" -eq 1 ]
}

@test "a merge request from a fork the caller cannot read names no owner, and one without a pipeline no push" {
  world gitlab
  jq '. + {source_project_id: 99, target_project_id: 1} | del(.head_pipeline)' "${FAKE_DIR}/mr.json" \
    > "${FAKE_DIR}/fork.json"
  mv "${FAKE_DIR}/fork.json" "${FAKE_DIR}/mr.json"
  route glab 'api projects/99' '!1'
  run call open-prs
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.fork, .pushed, .checks.total]' <<<"$output")" = '[null,null,0]' ]
}

@test "a change that closes many issues is drawn with every one: GitLab's hundred, and GitHub's first ten, then the rest" {
  for forge in github gitlab
  do
    world "${forge}"
    if [[ "${forge}" == github ]]
    then
      # the first page holds ten and says there are more; the rest come in pages of their own, after its cursor
      jq '.data.repository.pullRequests.nodes[0].closingIssuesReferences
            = {pageInfo: {hasNextPage: true, endCursor: "c10"}, nodes: [range(1; 11) | {number: ., title: "Issue \(.)"}]}' \
        "${FAKE_DIR}/open-prs.json" > "${FAKE_DIR}/many.json"
      mv "${FAKE_DIR}/many.json" "${FAKE_DIR}/open-prs.json"
      fixture rest-1.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {
        "pageInfo": {"hasNextPage": true, "endCursor": "c11"}, "nodes": [{"number": 11, "title": "Issue 11"}]}}}}}'
      fixture rest-2.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {
        "pageInfo": {"hasNextPage": false, "endCursor": "c12"}, "nodes": [{"number": 12, "title": "Issue 12"}]}}}}}'
      { printf '%s\t%s\n' 'api graphql*pr=3*after=c10*' rest-1.json 'api graphql*pr=3*after=c11*' rest-2.json
        cat "${FAKE_DIR}/gh.routes"; } > "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/gh.routes"
    else
      jq --null-input '[range(1; 13) | {iid: ., title: "Issue \(.)", state: "opened",
                                        references: {full: "empowerite/example#\(.)"}}]' > "${FAKE_DIR}/closes.json"
    fi
    run call open-prs
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output '[.closes[].number]' <<<"$output")" = "$(jq --null-input --compact-output '[range(1; 13)]')" ]
  done
  # GitHub prices the labels under each issue a change closes by how many issues it may give, so it is asked for ten,
  # and the rest, where there are more, a hundred at a time
  grep --quiet 'closingIssuesReferences(first: 10)' "${FAKE_DIR}/gh.calls"
  [ "$(grep --count 'closingIssuesReferences(first: 100, after: $after)' "${FAKE_DIR}/gh.calls")" -eq 2 ]
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

@test "every provider lists its open issues in one shape, oldest first, with the assignees by login, the labels, and the repository" {
  expected='[{"number":7,"ref":"#7","title":"Fix the thing","assignees":["bob"],'
  expected+='"labels":[{"name":"bug","color":"#d73a4a"}]},'
  expected+='{"number":9,"ref":"#9","title":"Other thing","assignees":[],"labels":[]}]'
  for forge in github gitlab
  do
    world "${forge}"
    run call open-issues
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys --slurp 'map(del(.age, .createdAt, .source, .organization, .repository))' \
      <<<"$output")" = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
    # each issue names the tracker that holds it, the forge's own, and where in it the issue is
    [ "$(jq --raw-output --slurp 'map(.source) | unique | join(" ")' <<<"$output")" = "${forge}" ]
    [ "$(jq --raw-output --slurp 'map("\(.organization)/\(.repository)") | unique | join(" ")' <<<"$output")" \
      = empowerite/example ]
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

@test "every provider reads the caller's rate limits in one shape, GitHub's GraphQL first, and kit flow --quota lists them" {
  fixture limits.json <<<'{"resources": {"core": {"limit": 5000, "remaining": 4990, "reset": 1800000000},
    "graphql": {"limit": 5000, "remaining": 4871, "reset": 1800000300}, "search": {"limit": 30, "remaining": 30}}}'
  printf 'HTTP/2.0 200 OK\r\nRatelimit-Limit: 2000\r\nRatelimit-Name: throttle_authenticated_api\r\n%b\r\n\r\n%s\n' \
    'Ratelimit-Remaining: 1998\r\nRatelimit-Reset: 1800000060' '{"username": "ada"}' > "${FAKE_DIR}/user-headers.txt"
  world github
  route gh 'api rate_limit' limits.json
  run call quota
  [ "$status" -eq 0 ]
  [ "$output" = '{"forge":"github","limits":[{"name":"graphql","limit":5000,"remaining":4871,"reset":1800000300},{"name":"rest","limit":5000,"remaining":4990,"reset":1800000000}]}' ]
  run "${kit_root}/bin/kit" flow --quota
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output 'map(.forge) | join(" ")' <<<"$output")" = github ]
  world gitlab
  route glab 'api --include user' user-headers.txt
  run call quota
  [ "$status" -eq 0 ]
  [ "$output" = '{"forge":"gitlab","limits":[{"name":"throttle_authenticated_api","limit":2000,"remaining":1998,"reset":1800000060}]}' ]
}

@test "every provider reads a mark of each kind of its data, and GitHub's marks cost nothing while nothing changed" {
  printf 'HTTP/2.0 200 OK\r\nEtag: W/"%s"\r\n\r\n[]\n' i1 > "${FAKE_DIR}/issues-200.txt"
  printf 'HTTP/2.0 200 OK\r\nETag: W/"%s"\r\n\r\n{}\n' r1 > "${FAKE_DIR}/runs-200.txt"
  printf 'HTTP/2.0 200 OK\r\netag: W/"%s"\r\n\r\n[]\n' v1 > "${FAKE_DIR}/releases-200.txt"
  printf 'HTTP/2.0 200 OK\r\nETag: W/"%s"\r\n\r\n[]\n' m1 > "${FAKE_DIR}/milestones-200.txt"
  printf 'HTTP/2.0 304 Not Modified\r\n\r\n' > "${FAKE_DIR}/304.txt"
  world github
  route gh 'api --include --header If-None-Match: W/"i1" repos/{owner}/{repo}/issues*' '!1 >304.txt'
  route gh 'api --include --header If-None-Match: W/"r1" repos/{owner}/{repo}/actions/runs*' '!1 >304.txt'
  route gh 'api --include --header If-None-Match: W/"m1" repos/{owner}/{repo}/milestones*' '!1 >304.txt'
  route gh 'api --include repos/{owner}/{repo}/issues?state=all&sort=updated&direction=desc&per_page=1' issues-200.txt
  route gh 'api --include repos/{owner}/{repo}/actions/runs?per_page=10' runs-200.txt
  route gh 'api --include*repos/{owner}/{repo}/releases?per_page=1' releases-200.txt
  route gh 'api --include*repos/{owner}/{repo}/milestones?state=all&per_page=100' milestones-200.txt
  run call marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"issues":"W/\"i1\"","runs":"W/\"r1\"","releases":"W/\"v1\"","milestones":"W/\"m1\""}' ]
  # the second read sends each ETag kept, and a 304 keeps the mark
  : > "${FAKE_DIR}/gh.calls"
  run call marks
  [ "$status" -eq 0 ]
  [ "$output" = '{"issues":"W/\"i1\"","runs":"W/\"r1\"","releases":"W/\"v1\"","milestones":"W/\"m1\""}' ]
  [ "$(grep --count 'If-None-Match' "${FAKE_DIR}/gh.calls")" -eq 4 ]
  # a change gives a new ETag, which is the new mark
  printf 'HTTP/2.0 200 OK\r\nEtag: W/"%s"\r\n\r\n[]\n' i2 > "${FAKE_DIR}/issues-200.txt"
  printf '%s\n' 'api --include --header If-None-Match: W/"i1" repos/{owner}/{repo}/issues*'$'\t''issues-200.txt' \
    > "${FAKE_DIR}/routes.tmp" && grep --invert-match --fixed-strings 'W/"i1"' "${FAKE_DIR}/gh.routes" \
    >> "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/gh.routes"
  [ "$(call marks | jq --raw-output .issues)" = 'W/"i2"' ]
  # GitLab's marks say what its newest items say of when they changed
  world gitlab
  local p='projects/empowerite%2Fexample'
  fixture gl-issues.json <<<'[{"iid": 7, "updated_at": "2026-11-01T09:00:00.000Z"}]'
  fixture gl-mrs.json <<<'[{"iid": 3, "updated_at": "2026-11-02T09:00:00.000Z"}]'
  fixture gl-pipelines.json <<<'[{"id": 11, "status": "success", "updated_at": "2026-11-03T09:00:00.000Z"}]'
  fixture gl-releases.json <<<'[{"tag_name": "v1", "released_at": "2026-11-04T09:00:00.000Z"}]'
  fixture gl-milestones.json <<<'[{"iid": 2, "updated_at": "2026-11-05T09:00:00.000Z"}]'
  route glab "api ${p}/issues?order_by=updated_at&sort=desc&per_page=1" gl-issues.json
  route glab "api ${p}/merge_requests?order_by=updated_at&sort=desc&per_page=1" gl-mrs.json
  route glab "api ${p}/pipelines?per_page=10" gl-pipelines.json
  route glab "api ${p}/releases?per_page=1" gl-releases.json
  route glab "api ${p}/milestones?sort=updated_desc&per_page=1" gl-milestones.json
  run call marks
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.issues, .runs, .releases, .milestones' <<<"$output")" = \
    '[[7,"2026-11-01T09:00:00.000Z"],[3,"2026-11-02T09:00:00.000Z"]]
[[11,"success","2026-11-03T09:00:00.000Z"]]
[["v1","2026-11-04T09:00:00.000Z"]]
[[2,"2026-11-05T09:00:00.000Z"],[7,"2026-11-01T09:00:00.000Z"]]' ]
}

@test "a provider that cannot read the rate limits says how to fix it, and kit flow --quota prints nothing" {
  world github
  route gh 'api rate_limit' '!1'
  run call quota
  [ "$status" -eq 1 ]
  [[ "$output" == *"gh auth status"* ]]
  run bash -c '"$0" flow --quota 2>/dev/null' "${kit_root}/bin/kit"
  [ "$status" -ne 0 ]
  [ -z "$output" ]
}

@test "every provider lists its releases in one shape, newest first, with the kind, the latest, the page, the notes, and the assets" {
  # gh api --paginate prints each page's array after the one before
  fixture gh-releases.json <<'JSON'
[{"tag_name": "release/0.1.0-rc.6", "name": "kit 0.1.0-rc.6", "prerelease": true, "draft": false,
  "published_at": "2026-10-27T07:30:14Z", "created_at": "2026-10-27T07:30:14Z",
  "html_url": "https://example.test/release/0.1.0-rc.6", "body": "notes\r\n* one",
  "assets": [{"name": "kit.tar.gz", "browser_download_url": "https://example.test/kit.tar.gz"}]}]
[{"tag_name": "release/0.0.9", "name": "kit 0.0.9", "prerelease": false, "draft": false,
  "published_at": "2026-10-20T07:30:14Z", "created_at": "2026-10-20T07:30:14Z",
  "html_url": "https://example.test/release/0.0.9", "body": null, "assets": []},
 {"tag_name": "release/0.0.8", "name": "kit 0.0.8", "prerelease": false, "draft": false,
  "published_at": "2026-10-13T07:30:14Z", "created_at": "2026-10-13T07:30:14Z",
  "html_url": "https://example.test/release/0.0.8", "body": "", "assets": []}]
JSON
  # GitLab's own assets are its links; the sources it makes of every tag are no asset of the release's
  fixture gl-releases.json <<'JSON'
[{"tag_name": "release/0.1.0-rc.6", "name": "kit 0.1.0-rc.6", "released_at": "2026-10-27T07:30:14.000Z",
  "upcoming_release": false, "description": "notes\n* one",
  "_links": {"self": "https://example.test/release/0.1.0-rc.6"},
  "assets": {"count": 2, "sources": [{"format": "zip", "url": "https://example.test/source.zip"}],
             "links": [{"name": "kit.tar.gz", "url": "https://example.test/kit.tar.gz?raw",
                        "direct_asset_url": "https://example.test/kit.tar.gz"}]}},
 {"tag_name": "release/0.0.9", "name": "kit 0.0.9", "released_at": "2026-10-20T07:30:14.000Z",
  "upcoming_release": false, "description": null, "_links": {"self": "https://example.test/release/0.0.9"},
  "assets": {"count": 0, "sources": [], "links": []}},
 {"tag_name": "release/0.0.8", "name": "kit 0.0.8", "released_at": "2026-10-13T07:30:14.000Z",
  "upcoming_release": false, "_links": {"self": "https://example.test/release/0.0.8"}}]
JSON
  expected='[{"version":"0.1.0-rc.6","tag":"release/0.1.0-rc.6","name":"kit 0.1.0-rc.6","kind":"pre-release","latest":false,'
  expected+='"url":"https://example.test/release/0.1.0-rc.6","notes":"notes\n* one",'
  expected+='"assets":[{"name":"kit.tar.gz","url":"https://example.test/kit.tar.gz"}]},'
  expected+='{"version":"0.0.9","tag":"release/0.0.9","name":"kit 0.0.9","kind":"release","latest":true,'
  expected+='"url":"https://example.test/release/0.0.9","notes":"","assets":[]},'
  expected+='{"version":"0.0.8","tag":"release/0.0.8","name":"kit 0.0.8","kind":"release","latest":false,'
  expected+='"url":"https://example.test/release/0.0.8","notes":"","assets":[]}]'
  for forge in github gitlab
  do
    world "${forge}"
    route gh 'api --paginate repos/{owner}/{repo}/releases?per_page=100' gh-releases.json
    route glab 'api projects/empowerite%2Fexample/releases?per_page=100*' gl-releases.json
    run call releases
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys --slurp 'map(del(.age, .at))' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected}")" ] || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq --slurp 'map(.age > 0) | all' <<<"$output")" = true ]
  done
}

@test "every provider lists its milestones in one shape: its number, state, due, issue counts, and page" {
  expected='{"number":2,"title":"v0.1","state":"open","due":1796083200,"open":3,"closed":2,'
  for forge in github gitlab
  do
    world "${forge}"
    run call milestones
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys 'del(.url)' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected%,}}")" ] || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq --raw-output '.url | startswith("https://")' <<<"$output")" = true ]
  done
}

@test "every provider lists its iterations by start, in one shape, and a forge without them for this repository is off" {
  expected='{"id":"x","title":"Sprint 1","start":1793577600,"end":1794787199}'
  for forge in github gitlab
  do
    world "${forge}"
    run call iterations
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys 'del(.id, .url)' <<<"$(head -n 1 <<<"$output")")" \
      = "$(jq --compact-output --sort-keys 'del(.id)' <<<"${expected}")" ] || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq --raw-output '.url | startswith("https://")' <<<"$(head -n 1 <<<"$output")")" = true ]
  done
  # GitHub's are those of its linked projects, oldest first, and none with no project linked
  world github
  run call iterations
  [ "$(jq --raw-output '.title' <<<"$output" | paste -sd, -)" = 'Sprint 1,Sprint 2' ]
  [ "$(jq --raw-output 'select(.title == "Sprint 2") | .end' <<<"$output")" = 1795996799 ]
  GITHUB_ITERATIONS=gh-iterations-none.json world github
  run call iterations
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  # GitLab's tiers without iterations answer 403 or 404, which is off here
  for said in 403 404
  do
    printf 'glab: %s\n' "${said}" > "${FAKE_DIR}/gl-forbidden.txt"
    GITLAB_ITERATIONS='!1 gl-forbidden.txt' world gitlab
    run --separate-stderr call iterations
    [ "$status" -eq 3 ] || { echo "${said}: ${status}"; return 1; }
    [ -z "$output" ]
  done
  GITLAB_ITERATIONS='!1' world gitlab
  run call iterations
  [ "$status" -eq 1 ]
}

@test "every provider lists the changes merged last in one shape, and says when it stopped at 50" {
  local expected='{"number":2,"title":"Do the other thing","author":"bob","branch":"9-other",'
  expected+='"merged":1791018000,"by":"ada"}'
  for forge in github gitlab
  do
    world "${forge}"
    run --separate-stderr call merged-prs
    [ "$status" -eq 0 ] || { echo "${forge}: ${stderr}"; return 1; }
    [ "$(jq --compact-output 'del(.url)' <<<"$output")" = "${expected}" ]
  done
  [ "$(jq --raw-output .url <<<"$output")" = 'https://gitlab.com/empowerite/example/-/merge_requests/2' ]
  grep --quiet 'pullRequests(states: MERGED, first: 50, orderBy: {field: UPDATED_AT, direction: DESC}) {' \
    "${FAKE_DIR}/gh.calls"
  # 51 merged changes on GitLab, merged last first, or GitHub's count past those it read, give the first 50, then
  # limited
  jq '[range(51) as $i | (50 - $i) as $m | .[0] | .iid = (60 - $i)
       | .merged_at = "2026-10-03T09:\(if $m < 10 then "0" else "" end)\($m):00Z"]' \
    "${FAKE_DIR}/gl-merged.json" | fixture gl-merged-51.json
  GITLAB_MERGED=gl-merged-51.json world gitlab
  run --separate-stderr call merged-prs
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].number, .[0].merged, .[49].number, .[50]]' <<<"$output")" \
    = '[51,60,1791021000,11,{"limited":50}]' ]
  # past the 50 GitHub read, a change merged before the oldest update among them may be missing, so one merged long ago
  # and updated since is left out, and limited counts those kept
  jq '.data.repository.pullRequests |= (.totalCount = 70 | .nodes += [.nodes[0] | .number = 1
       | .mergedAt = "2026-01-05T09:00:00Z" | .updatedAt = "2026-10-05T09:00:00Z"])' "${FAKE_DIR}/gh-merged.json" \
    | fixture gh-merged-70.json
  GITHUB_MERGED=gh-merged-70.json world github
  run --separate-stderr call merged-prs
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output 'map(.number // .)' <<<"$output")" = '[2,{"limited":1}]' ]
  grep --quiet 'totalCount nodes { number title url headRefName mergedAt updatedAt author { login }' \
    "${FAKE_DIR}/gh.calls"
}

@test "a GitLab older than 17.2, which refuses order_by=merged_at, has its merged changes read by update instead" {
  local p='projects/empowerite%2Fexample'
  world gitlab
  fixture gl-order.txt <<<'glab: order_by does not have a valid value (HTTP 400)'
  # newest updated first: one merged long ago and updated since, then fifty merged and updated a minute apart
  jq '[(.[0] | .iid = 7 | .merged_at = "2026-01-05T09:00:00Z" | .updated_at = "2026-10-05T09:00:00Z"),
       (range(1; 51) as $i | (50 - $i) as $m | .[0] | .iid = (100 - $i)
        | .merged_at = "2026-10-03T09:\(if $m < 10 then "0" else "" end)\($m):00Z" | .updated_at = .merged_at)]' \
    "${FAKE_DIR}/gl-merged.json" | fixture gl-merged-by-update.json
  : > "${FAKE_DIR}/glab.routes"
  route glab 'api user' user.json
  route glab "api ${p}/merge_requests?state=merged&order_by=merged_at&per_page=51" '!1 gl-order.txt'
  route glab "api ${p}/merge_requests?state=merged&order_by=updated_at&sort=desc&per_page=51" gl-merged-by-update.json
  run --separate-stderr call merged-prs
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  # of the first 50 by update, those merged since the oldest update among them, merged last first, then limited
  [ "$(jq --slurp --compact-output '[length, .[0].number, .[48].number, .[49]]' <<<"$output")" \
    = '[50,99,51,{"limited":49}]' ]
  # any other refusal is no older GitLab: it fails, and asks nothing more
  fixture gl-denied.txt <<<'glab: 401 Unauthorized'
  : > "${FAKE_DIR}/glab.routes"
  route glab "api ${p}/merge_requests?state=merged&order_by=merged_at&per_page=51" '!1 gl-denied.txt'
  run --separate-stderr call merged-prs
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "$stderr" = '❌ run: glab auth status (the merged merge requests could not be read)' ]
  [ "$(grep -c 'order_by=updated_at' "${FAKE_DIR}/glab.calls")" -eq 1 ]
}

# delivery-github adds to world-github's routes what Delivery reads: artifact 13, kit-0.1.0.tar.gz, from run 103; the
# environment production, which asks ada to review and lets main deploy, beside staging, which asks nothing; and
# deployment 9 of release/0.1.0 to production, by ada, which succeeded.
delivery-github() {
  fixture gh-artifacts.json <<'JSON'
{"total_count": 1, "artifacts": [{"id": 13, "name": "kit-0.1.0.tar.gz", "size_in_bytes": 2048,
  "digest": "sha256:ab12", "expired": false, "created_at": "2026-10-01T10:00:00Z",
  "expires_at": "2026-12-30T10:00:00Z", "workflow_run": {"id": 103, "head_branch": "main", "head_sha": "8e817bf"},
  "archive_download_url": "https://api.github.com/repos/empowerite/example/actions/artifacts/13/zip"}]}
JSON
  fixture gh-environments.json <<'JSON'
[{"total_count": 2, "environments": [
  {"id": 1, "name": "production",
   "html_url": "https://github.com/empowerite/example/deployments/activity_log?environments_filter=production",
   "protection_rules": [
     {"id": 5, "type": "required_reviewers", "prevent_self_review": false,
      "reviewers": [{"type": "User", "reviewer": {"login": "ada", "id": 1}}]},
     {"id": 6, "type": "branch_policy"}],
   "deployment_branch_policy": {"protected_branches": false, "custom_branch_policies": true}},
  {"id": 2, "name": "staging",
   "html_url": "https://github.com/empowerite/example/deployments/activity_log?environments_filter=staging",
   "protection_rules": [], "deployment_branch_policy": null}]}]
JSON
  fixture gh-branch-policies.json <<'JSON'
{"total_count": 1, "branch_policies": [{"id": 1, "name": "main", "type": "branch"}]}
JSON
  fixture gh-deployments.json <<'JSON'
{"data": {"repository": {"url": "https://github.com/empowerite/example", "deployments": {"totalCount": 1, "nodes": [
  {"databaseId": 9, "environment": "production", "ref": {"name": "release/0.1.0"},
   "commitOid": "8e817bf000000000000000000000000000000000", "state": "ACTIVE", "createdAt": "2026-10-01T10:00:00Z",
   "creator": {"login": "ada"},
   "commit": {"url": "https://github.com/empowerite/example/commit/8e817bf000000000000000000000000000000000"},
   "latestStatus": {"state": "SUCCESS", "logUrl": "https://github.com/empowerite/example/actions/runs/103/job/1",
                    "environmentUrl": null}}]}}}}
JSON
  route gh 'api repos/{owner}/{repo}/actions/artifacts?per_page=100' "${GITHUB_ARTIFACTS:-gh-artifacts.json}"
  route gh 'api --paginate --slurp repos/{owner}/{repo}/environments?per_page=100' \
    "${GITHUB_ENVIRONMENTS:-gh-environments.json}"
  route gh 'api repos/{owner}/{repo}/environments/*/deployment-branch-policies?per_page=100' \
    "${GITHUB_BRANCH_POLICIES:-gh-branch-policies.json}"
}

# delivery-gitlab is delivery-github's world, seen through GitLab: job 13 of pipeline 103 kept kit-0.1.0.tar.gz, and
# job 12 kept nothing; production is a protected environment ada approves.
delivery-gitlab() {
  local p='projects/empowerite%2Fexample'
  fixture gl-jobs.json <<'JSON'
[{"id": 13, "name": "package", "status": "success", "created_at": "2026-10-01T09:58:00.000Z",
  "finished_at": "2026-10-01T10:00:00.000Z", "artifacts_file": {"filename": "kit-0.1.0.tar.gz", "size": 2048},
  "artifacts": [{"file_type": "archive", "size": 2048, "filename": "kit-0.1.0.tar.gz", "file_format": "zip"}],
  "artifacts_expire_at": "2026-12-30T10:00:00.000Z", "pipeline": {"id": 103},
  "web_url": "https://gitlab.com/empowerite/example/-/jobs/13"},
 {"id": 12, "name": "lint", "status": "success", "created_at": "2026-10-01T09:50:00.000Z",
  "finished_at": "2026-10-01T09:51:00.000Z", "artifacts": [], "artifacts_expire_at": null, "pipeline": {"id": 103},
  "web_url": "https://gitlab.com/empowerite/example/-/jobs/12"}]
JSON
  fixture gl-environments.json <<'JSON'
[{"id": 1, "name": "production", "slug": "production", "state": "available", "tier": "production",
  "external_url": null, "created_at": "2026-09-01T09:00:00.000Z", "updated_at": "2026-10-01T10:00:00.000Z"},
 {"id": 2, "name": "staging", "slug": "staging", "state": "available", "tier": "staging",
  "external_url": null, "created_at": "2026-09-01T09:00:00.000Z", "updated_at": "2026-10-01T10:00:00.000Z"}]
JSON
  fixture gl-protected.json <<'JSON'
[{"name": "production",
  "deploy_access_levels": [{"id": 12, "access_level": 40, "access_level_description": "Maintainers",
                            "user_id": null, "group_id": null, "group_inheritance_type": 0}],
  "required_approval_count": 0,
  "approval_rules": [{"id": 1, "user_id": 1, "group_id": null, "access_level": null,
                      "access_level_description": "ada", "required_approvals": 1, "group_inheritance_type": 0}]}]
JSON
  fixture gl-deployments.json <<'JSON'
[{"id": 9, "iid": 1, "ref": "release/0.1.0", "sha": "8e817bf000000000000000000000000000000000",
  "created_at": "2026-10-01T10:00:00.000Z", "updated_at": "2026-10-01T10:05:00.000Z", "status": "success",
  "user": {"username": "ada"}, "environment": {"id": 1, "name": "production", "external_url": null},
  "deployable": {"id": 13, "web_url": "https://gitlab.com/empowerite/example/-/jobs/13"}}]
JSON
  route glab "api ${p}/jobs?scope[]=success&per_page=100" "${GITLAB_JOBS:-gl-jobs.json}"
  route glab "api ${p}/environments?per_page=100 --paginate" gl-environments.json
  route glab "api ${p}/protected_environments?per_page=100 --paginate" "${GITLAB_PROTECTED:-gl-protected.json}"
  route glab "api ${p}/deployments?order_by=created_at&sort=desc&per_page=51" \
    "${GITLAB_DEPLOYMENTS:-gl-deployments.json}"
}

@test "every provider lists its artifacts in one shape, newest first, and says when it stopped" {
  local expected='{"id":13,"name":"kit-0.1.0.tar.gz","digest":"sha256:ab12","size":2048,"run":103,'
  expected+='"expires":1798624800}'
  for forge in github gitlab
  do
    world "${forge}"
    run --separate-stderr call artifacts
    [ "$status" -eq 0 ] || { echo "${forge}: ${stderr}"; return 1; }
    [ "$(jq --slurp 'length' <<<"$output")" -eq 1 ] || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq --raw-output '.age > 0' <<<"$output")" = true ]
    # GitLab keeps no digest of an artifact that its API shows
    if [[ "${forge}" == github ]]
    then
      [ "$(jq --compact-output 'del(.age, .url)' <<<"$output")" = "${expected}" ]
      [ "$(jq --raw-output .url <<<"$output")" \
        = 'https://github.com/empowerite/example/actions/runs/103/artifacts/13' ]
    else
      [ "$(jq --compact-output 'del(.age, .url)' <<<"$output")" = "${expected/'"sha256:ab12"'/null}" ]
      [ "$(jq --raw-output .url <<<"$output")" = 'https://gitlab.com/empowerite/example/-/jobs/13' ]
    fi
  done
  # GitHub's count past the page read, or a page of 100 of GitLab's jobs, says the list stopped at those drawn
  jq '.total_count = 150' "${FAKE_DIR}/gh-artifacts.json" | fixture gh-artifacts-150.json
  GITHUB_ARTIFACTS=gh-artifacts-150.json world github
  run --separate-stderr call artifacts
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output 'map(.id // .)' <<<"$output")" = '[13,{"limited":1}]' ]
  jq '[range(100) as $i | .[$i % 2] | .id = (200 - $i)]' "${FAKE_DIR}/gl-jobs.json" | fixture gl-jobs-100.json
  GITLAB_JOBS=gl-jobs-100.json world gitlab
  run --separate-stderr call artifacts
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].id, .[-1]]' <<<"$output")" = '[51,200,{"limited":50}]' ]
}

@test "every provider lists its environments in one shape: the protection, who approves, and the branches that deploy" {
  for forge in github gitlab
  do
    world "${forge}"
    run --separate-stderr call environments
    [ "$status" -eq 0 ] || { echo "${forge}: ${stderr}"; return 1; }
    [ "$(jq --raw-output '.name' <<<"$output" | paste -sd, -)" = 'production,staging' ]
    [ "$(jq --compact-output 'select(.name == "production") | .approvers' <<<"$output")" = '["ada"]' ]
    [ "$(jq --compact-output 'select(.name == "staging") | [.protection, .approvers, .branches]' <<<"$output")" \
      = '[[],[],null]' ] || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq --raw-output 'select(.name == "production") | .url | startswith("https://")' <<<"$output")" = true ]
  done
  # GitLab keeps no branches an environment takes deployments from; who may deploy there is a protection of its own
  [ "$(jq --compact-output 'select(.name == "production") | [.protection, .branches, .url]' <<<"$output")" \
    = '[["deployers","reviewers"],null,"https://gitlab.com/empowerite/example/-/environments/1"]' ]
  world github
  : > "${FAKE_DIR}/gh.calls"
  run --separate-stderr call environments
  [ "$(jq --compact-output 'select(.name == "production") | [.protection, .branches]' <<<"$output")" \
    = '[["branches","reviewers"],["main"]]' ]
  # GitHub's branch policies are read only for an environment that has its own, by its name
  [ "$(grep -c 'deployment-branch-policies' "${FAKE_DIR}/gh.calls")" -eq 1 ]
  grep --quiet 'environments/production/deployment-branch-policies' "${FAKE_DIR}/gh.calls"
  # a GitLab tier without protected environments, or a caller who may not read them, leaves the protection unknown
  for said in 403 404
  do
    printf 'glab: %s\n' "${said}" > "${FAKE_DIR}/gl-forbidden.txt"
    GITLAB_PROTECTED='!1 gl-forbidden.txt' world gitlab
    run --separate-stderr call environments
    [ "$status" -eq 0 ] || { echo "${said}: ${stderr}"; return 1; }
    [ "$(jq --compact-output '[.name, .protection, .approvers]' <<<"$output" | paste -sd, -)" \
      = '["production",null,null],["staging",null,null]' ]
  done
  GITLAB_PROTECTED='!1' world gitlab
  run --separate-stderr call environments
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"run: glab auth status (the protected environments could not be read)"* ]]
}

@test "GitHub reads the branch policies of 20 environments at most, and says the list stopped there" {
  world github
  jq '.[0].environments = [range(25) as $i | .[0].environments[0] | .name = "env-\(100 + $i)"]
      | .[0].total_count = 25' "${FAKE_DIR}/gh-environments.json" | fixture gh-environments-25.json
  GITHUB_ENVIRONMENTS=gh-environments-25.json world github
  run --separate-stderr call environments
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq --slurp --compact-output '[length, .[0].name, .[19].name, .[20]]' <<<"$output")" \
    = '[21,"env-100","env-119",{"limited":20}]' ]
  [ "$(grep -c 'deployment-branch-policies' "${FAKE_DIR}/gh.calls")" -eq 20 ]
  # an environment with more branch policies than a page holds fails the read, as it would be partial
  jq '.total_count = 101' "${FAKE_DIR}/gh-branch-policies.json" | fixture gh-branch-policies-101.json
  GITHUB_BRANCH_POLICIES=gh-branch-policies-101.json world github
  run --separate-stderr call environments
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"kit reads at most 100 branch policies of an environment (the environments could not be read)"* ]]
}

@test "every provider lists its deployments in one shape, newest first, and says when it stopped at 50" {
  local expected='{"id":9,"environment":"production","ref":"release/0.1.0",'
  expected+='"sha":"8e817bf000000000000000000000000000000000","state":"success","actor":"ada"}'
  for forge in github gitlab
  do
    world "${forge}"
    run --separate-stderr call deployments
    [ "$status" -eq 0 ] || { echo "${forge}: ${stderr}"; return 1; }
    [ "$(jq --compact-output 'del(.age, .url, .commit)' <<<"$output")" = "${expected}" ] \
      || { echo "${forge}: ${output}"; return 1; }
    [ "$(jq --raw-output '.age > 0' <<<"$output")" = true ]
  done
  [ "$(jq --raw-output .url <<<"$output")" = 'https://gitlab.com/empowerite/example/-/jobs/13' ]
  [ "$(jq --raw-output .commit <<<"$output")" \
    = 'https://gitlab.com/empowerite/example/-/commit/8e817bf000000000000000000000000000000000' ]
  grep --quiet 'deployments(first: 50, orderBy: {field: CREATED_AT, direction: DESC}) { totalCount' \
    "${FAKE_DIR}/gh.calls"
  # 60 on GitHub, of which its read holds 50, or GitLab's 51st, give the 50 newest, then limited
  jq '.data.repository.deployments |= (.totalCount = 60
       | .nodes = [range(60) as $i | .nodes[0] | .databaseId = (100 - $i)])' "${FAKE_DIR}/gh-deployments.json" \
    | fixture gh-deployments-60.json
  GITHUB_DEPLOYMENTS=gh-deployments-60.json world github
  run --separate-stderr call deployments
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].id, .[49].id, .[50]]' <<<"$output")" = '[51,100,51,{"limited":50}]' ]
  jq '[range(51) as $i | .[0] | .id = (100 - $i)]' "${FAKE_DIR}/gl-deployments.json" | fixture gl-deployments-51.json
  GITLAB_DEPLOYMENTS=gl-deployments-51.json world gitlab
  run --separate-stderr call deployments
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].id, .[49].id, .[50]]' <<<"$output")" = '[51,100,51,{"limited":50}]' ]
  # a GitHub answer with no repository could not be read
  fixture gh-null.json <<<'{"data": {"repository": null}}'
  GITHUB_DEPLOYMENTS=gh-null.json world github
  run --separate-stderr call deployments
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"the deployments could not be read"* ]]
}

@test "GitLab's milestones follow the cursor past a page, and a failing page fails the call" {
  world gitlab
  : > "${FAKE_DIR}/glab.routes"
  fixture gl-page1.json <<'JSON'
{"data": {"project": {"milestones": {"pageInfo": {"hasNextPage": true, "endCursor": "c1"}, "nodes": [
  {"iid": "1", "title": "a", "state": "closed", "dueDate": null, "webPath": "/o/e/-/milestones/1",
   "stats": {"totalIssuesCount": 1, "closedIssuesCount": 1}}]}}}}
JSON
  route glab 'api graphql*after=c1*' gl-milestones.json
  route glab 'api graphql*' gl-page1.json
  run call milestones
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.number' <<<"$output" | paste -sd, -)" = '1,2' ]
  [ "$(jq --raw-output 'select(.number == 2) | [.open, .closed, .url] | join(" ")' <<<"$output")" \
    = '3 2 https://gitlab.com/empowerite/example/-/milestones/2' ]
  : > "${FAKE_DIR}/glab.routes"
  route glab 'api graphql*' '!1'
  run --separate-stderr call milestones
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "GitHub's iterations need the read:project scope, and name the command that adds it; a partial read fails" {
  world github
  fixture gh-scope.txt <<<'gh: Your token has not been granted the required scopes to execute this query. The '"'"'id'"'"' field requires one of the following scopes: ['"'"'read:project'"'"'], but your token has only been granted the: ['"'"'repo'"'"'] scopes. (INSUFFICIENT_SCOPES)'
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*' '!1 gh-scope.txt'
  run --separate-stderr call iterations
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"run: gh auth refresh --scopes read:project (the iterations could not be read)"* ]]
  [ "$(grep --count '❌' <<<"$stderr")" -eq 1 ]
  fixture gh-many.json <<'JSON'
{"data": {"repository": {"projectsV2": {"totalCount": 21, "nodes": [{"url": "u", "fields": {"totalCount": 1, "nodes": [{}]}}]}}}}
JSON
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*' gh-many.json
  run --separate-stderr call iterations
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"at most 20 projects and 50 fields"* ]]
  fixture gh-wide.json <<'JSON'
{"data": {"repository": {"projectsV2": {"totalCount": 1, "nodes": [{"url": "u", "fields": {"totalCount": 51, "nodes": [{}]}}]}}}}
JSON
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*' gh-wide.json
  run --separate-stderr call iterations
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"at most 20 projects"* ]]
  # an answer that is not the shape asked for is not a cap
  fixture gh-null.json <<<'{"data": {"repository": null}}'
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*' gh-null.json
  run --separate-stderr call iterations
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"the iterations could not be read"* && "$stderr" != *"at most 20"* ]]
}

# runs-world <forge> is world's, with the runs: their list, and the jobs of run 103.
runs-world() {
  world "$1"
  route gh 'run list --limit 50 --json *' gh-runs.json
  route gh 'run view 103 --json jobs' gh-jobs.json
  route glab 'api projects/empowerite%2Fexample/pipelines?per_page=50' gl-pipelines.json
}

@test "every provider lists the runs not owned by any pull request in one shape, newest first, the newest with its jobs" {
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

@test "every provider reads its open changes once a minute: a second call takes the snapshot, an old one is read again" {
  local file
  for forge in github gitlab
  do
    world "${forge}"
    : > "${FAKE_DIR}/gh.calls"
    : > "${FAKE_DIR}/glab.calls"
    first=$(call open-prs)
    [ "$(call open-prs)" = "${first}" ]
    [ "$(reads "${forge}")" -eq 1 ] || { echo "${forge}: $(reads "${forge}") reads"; return 1; }
    # a snapshot is kept under the second its read began; one older than a minute is read again, its ages anew
    file=$(snapshot "${forge}" open-prs)
    [ -f "${file}" ]
    { printf '%s\n' "$(( $(date +%s) - 61 ))"; tail -n +2 "${file}"; } > "${file}.old"
    mv "${file}.old" "${file}"
    [ "$(call open-prs | jq --raw-output .ref)" = "$(jq --raw-output .ref <<<"${first}")" ]
    [ "$(reads "${forge}")" -eq 2 ]
    call open-prs > /dev/null
    [ "$(reads "${forge}")" -eq 2 ]
  done
}

@test "a stage given --fresh reads afresh, and a call that changes the forge drops the snapshot" {
  for forge in github gitlab
  do
    world "${forge}"
    : > "${FAKE_DIR}/gh.calls"
    : > "${FAKE_DIR}/glab.calls"
    call open-prs > /dev/null
    # a snapshot read in the second the stage began counts as fresh, so the stage begins a second later
    sleep 1
    fresh-call open-prs > /dev/null
    [ "$(reads "${forge}")" -eq 2 ]
    # the fresh read is the snapshot now
    call open-prs > /dev/null
    [ "$(reads "${forge}")" -eq 2 ]
    call pr-create 7-fix-the-thing main "Fix the thing" > /dev/null
    [ ! -e "$(snapshot "${forge}" open-prs)" ]
    call open-prs > /dev/null
    [ "$(reads "${forge}")" -eq 3 ]
  done
}

@test "processes that ask at once make one read: the first takes the lock, the others wait for its snapshot" {
  local i
  local -a pids=()
  world github
  # gh answers slowly, so every process asks while the first still reads
  mkdir -p "${FAKE_DIR}/slow"
  mv "${FAKE_DIR}/bin/gh" "${FAKE_DIR}/slow/gh"
  cat > "${FAKE_DIR}/bin/gh" <<'SH'
#!/usr/bin/env bash
sleep 1
exec "${FAKE_DIR}/slow/gh" "$@"
SH
  chmod +x "${FAKE_DIR}/bin/gh"
  : > "${FAKE_DIR}/gh.calls"
  for i in 1 2 3 4
  do
    call open-prs > "${BATS_TEST_TMPDIR}/answer.${i}" 3>&- &
    pids+=("$!")
  done
  # bats has a process of its own in the background, so the wait names the test's
  wait "${pids[@]}"
  [ "$(reads github)" -eq 1 ]
  for i in 2 3 4
  do
    cmp "${BATS_TEST_TMPDIR}/answer.1" "${BATS_TEST_TMPDIR}/answer.${i}"
  done
  [ "$(jq --raw-output .ref "${BATS_TEST_TMPDIR}/answer.4")" = "#3" ]
  # so do the stages of one picture given --fresh, which take no snapshot read before the picture began
  : > "${FAKE_DIR}/gh.calls"
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge:fresh
    for i in 1 2 3
    do
      forge:open-prs > /dev/null &
    done
    wait' 3>&-
  [ "$(reads github)" -eq 1 ]
  [ ! -e "$(dirname "$(snapshot github open-prs)")/open-prs.lock" ]
}

@test "a lock whose process is gone is taken over, and the open issues are kept as the open changes are" {
  local dir
  world github
  call open-issues > /dev/null
  dir=$(dirname "$(snapshot github open-issues)")
  rm -f "${dir}"/*.json
  # a process that does not exist holds the lock
  ln -s 999999 "${dir}/open-prs.lock"
  run call open-prs
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output .ref <<<"$output")" = "#3" ]
  [ ! -e "${dir}/open-prs.lock" ]
  : > "${FAKE_DIR}/gh.calls"
  call open-issues > /dev/null
  call open-issues > /dev/null
  [ "$(grep --count '^issue list' "${FAKE_DIR}/gh.calls")" -eq 1 ]
}

@test "GitHub's open changes cost 18 points: closed issues' assignees come from the open issues, labels from each" {
  world github
  run call open-prs
  [ "$status" -eq 0 ]
  # the reviewers are asked in the same request, which costs 2 points more than without them: still one read
  [ "$(reads github)" -eq 1 ]
  grep --quiet 'reviewRequests(first: 20) { totalCount nodes { requestedReviewer {' "${FAKE_DIR}/gh.calls"
  grep --quiet 'latestReviews(first: 20) { totalCount nodes { author { login } state submittedAt } }' \
    "${FAKE_DIR}/gh.calls"
  [ "$(jq --compact-output '.closes | map(.assignees)' <<<"$output")" = '[["bob"]]' ]
  [ "$(jq --compact-output '[.labels, .closes[0].labels] | map(map(.name))' <<<"$output")" = '[["kind=fix"],["bug"]]' ]
  grep --quiet 'issues(states: OPEN, first: 100, filterBy: {assignee: "\*"}' "${FAKE_DIR}/gh.calls"
  grep --quiet 'closingIssuesReferences(first: 10) { pageInfo { hasNextPage endCursor } nodes {$' "${FAKE_DIR}/gh.calls"
  grep --quiet 'number title createdAt repository { nameWithOwner } labels(first: 20) { nodes { name color } }' \
    "${FAKE_DIR}/gh.calls"
  [ "$(grep --count 'assignees(first: 10)' "${FAKE_DIR}/gh.calls")" -eq 1 ]
}

@test "a pull request with more reviewers than GitHub's open changes read fails them, saying so" {
  world github
  jq '.data.repository.pullRequests.nodes[0].latestReviews.totalCount = 21' "${FAKE_DIR}/open-prs.json" \
    | fixture open-prs-many.json
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*' open-prs-many.json
  local what='the pull requests'
  run --separate-stderr call open-prs
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "$stderr" = "❌ kit reads at most 20 review requests and 20 reviews of a change (${what} could not be read)" ]
}

@test "a read gh's login lacks a scope for names the command that adds the scope GitHub names" {
  world github
  local said="gh: Your token has not been granted the required scopes to execute this query. The 'pullRequests'"
  said+=" field requires one of the following scopes: ['repo'], but your token has only been granted the: ['gist']"
  said+=" scopes. (INSUFFICIENT_SCOPES)"
  printf '%s\n' "${said}" | fixture gh-repo.txt
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*' '!1 gh-repo.txt'
  run --separate-stderr call merged-prs
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "$stderr" = '❌ run: gh auth refresh --scopes repo (the merged pull requests could not be read)' ]
}

@test "a token without read:org still reads the open changes: asked again without teams, it draws none of them" {
  world github
  local said="gh: Your token has not been granted the required scopes to execute this query. The 'slug' field"
  said+=" requires one of the following scopes: ['read:org'], but your token has only been granted the: ['repo']"
  said+=" scopes. (INSUFFICIENT_SCOPES)"
  printf '%s\n' "${said}" | fixture gh-org.txt
  # asked without the team fragment, GitHub gives a team asked as an empty object
  jq '.data.repository.pullRequests.nodes[0].reviewRequests.nodes += [{"requestedReviewer": {}}]' \
    "${FAKE_DIR}/open-prs.json" | fixture open-prs-teamless.json
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*on Team*' '!1 gh-org.txt'
  route gh 'api graphql*' open-prs-teamless.json
  : > "${FAKE_DIR}/gh.calls"
  run --separate-stderr call open-prs
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ -z "$stderr" ]
  [ "$(jq --compact-output '.reviewers | map([.login, .state])' <<<"$output")" \
    = '[["bob","requested"],["carol","approved"]]' ]
  [ "$(reads github)" -eq 2 ]
  [ "$(grep -c 'on Team { slug }' "${FAKE_DIR}/gh.calls")" -eq 1 ]
}

@test "a GitHub answer with no repository says it could not be read, never that a cap was reached" {
  world github
  fixture no-repo.json <<<'{"data": {"repository": null}}'
  : > "${FAKE_DIR}/gh.routes"
  route gh 'repo view*' repo.json
  route gh 'api graphql*' no-repo.json
  run --separate-stderr call open-prs
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "$stderr" = '❌ run: gh auth status (the pull requests could not be read)' ]
}

@test "a read GitHub's rate limit stops says which limit, and when it resets, never gh auth status" {
  local reset
  world github
  reset=$(( $(date +%s) + 700 ))
  fixture limited.txt <<<'gh: API rate limit already exceeded for user ID 1.'
  fixture limits.json <<<"{\"resources\": {\"core\": {\"remaining\": 4990, \"reset\": ${reset}},
    \"graphql\": {\"remaining\": 0, \"reset\": ${reset}}}}"
  printf 'api rate_limit*\tlimits.json\n' > "${FAKE_DIR}/routes.first"
  printf 'api graphql*\t!1 limited.txt\nissue list*\t!1 limited.txt\n' >> "${FAKE_DIR}/routes.first"
  cat "${FAKE_DIR}/routes.first" "${FAKE_DIR}/gh.routes" > "${FAKE_DIR}/routes.all"
  mv "${FAKE_DIR}/routes.all" "${FAKE_DIR}/gh.routes"
  run call open-prs
  [ "$status" -eq 1 ]
  [[ "$output" == *"wait until $(jq --null-input --raw-output "${reset} | strflocaltime(\"%H:%M\")"), in 12m, when GitHub's GraphQL rate limit resets (the pull requests could not be read)"* ]]
  [[ "$output" != *"gh auth status"* ]]
  run call open-issues
  [[ "$output" == *"GraphQL rate limit resets (the issues could not be read)"* ]]
  # nothing is kept from a read that failed, so the next process asks again
  [ ! -e "$(snapshot github open-prs)" ]
  # a secondary rate limit names no reset; GitHub asks for a wait of a few minutes
  fixture limited.txt <<<'gh: You have exceeded a secondary rate limit. Please wait a few minutes before you try again.'
  run call open-prs
  [ "$status" -eq 1 ]
  [[ "$output" == *"wait a few minutes: GitHub's secondary rate limit refused the read (the pull requests could not be read)"* ]]
  # any other failure still names gh auth status
  fixture limited.txt <<<'gh: Bad credentials (HTTP 401)'
  run call open-prs
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: gh auth status (the pull requests could not be read)"* ]]
}

@test "a forge that cannot answer fails every read with one line, and never answers nothing" {
  for forge in github gitlab
  do
    world "${forge}"
    printf 'repo view*\trepo.json\n' | tee "${FAKE_DIR}/gh.routes" > "${FAKE_DIR}/glab.routes"
    for call in 'branch-for-issue 7' 'pr-for-branch 7-fix-the-thing' 'pr-last-for-branch 7-fix-the-thing' \
      'pr-state-for-branch 7-fix-the-thing' 'pr-head-for-branch 7-fix-the-thing' pr-here 'pr-title 3' 'pr-body 3' \
      'pr-author 3' 'pr-closing-issues 3' 'pr-state 3' 'issue-state 7' 'issue-assignees 7' 'issue-title 7' open-prs \
      open-issues releases marks quota runs me
    do
      read -r -a words <<<"${call}" && run --separate-stderr call "${words[@]}"
      [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"could not be read)"* ]] \
        || { echo "${forge} ${call}: exit ${status}, printed '${output}', said '${stderr}'"; return 1; }
    done
  done
}

@test "one merge request's read that fails fails the open changes, and no part is kept as the snapshot" {
  world gitlab
  local p='projects/empowerite%2Fexample'
  fixture mrs-list.json <<<'[{"iid": 3}, {"iid": 4}]'
  { printf '%s\t%s\n' "api ${p}/merge_requests/4/discussions*" '!1' "api ${p}/merge_requests/4/closes_issues*" \
      closes.json "api ${p}/merge_requests/4" mr.json
    cat "${FAKE_DIR}/glab.routes"; } > "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/glab.routes"
  run call open-prs
  [ "$status" -eq 1 ]
  [ ! -f "$(snapshot gitlab open-prs)" ]
}

@test "a merge request's own pipeline whose jobs cannot be read fails the open changes, and says so" {
  world gitlab
  { printf '%s\t%s\n' 'api projects/1/pipelines/11/jobs*' '!1'
    cat "${FAKE_DIR}/glab.routes"; } > "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/glab.routes"
  run --separate-stderr call open-prs
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "$(grep --count '❌' <<<"$stderr")" -eq 1 ]
  [[ "$stderr" == *"run: glab auth status (the jobs of pipeline 11 could not be read)"* ]]
}

@test "a merge request list that is not a list fails the read, never reads as none" {
  world gitlab
  fixture mr-by-branch.json <<<'null'
  fixture mr-states.json <<<'null'
  for call in pr-for-branch pr-last-for-branch pr-state-for-branch pr-head-for-branch
  do
    run --separate-stderr call "${call}" 7-fix-the-thing
    [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"could not be read)"* ]] \
      || { echo "${call}: exit ${status}, printed '${output}', said '${stderr}'"; return 1; }
  done
}

@test "a read stopped by an answer kit cannot use says the remedy, then what could not be read" {
  world github
  printf 'HTTP/2.0 200 OK\r\n\r\n[]\n' > "${FAKE_DIR}/no-etag.txt"
  route gh 'api --include*' no-etag.txt
  run --separate-stderr call marks
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"GitHub gave no ETag for repos/{owner}/{repo}/issues"*"(the marks could not be read)" ]]
  # a GitLab checkout with no origin names the remote to add
  world gitlab
  git remote remove origin
  mkdir -p .kit && printf 'forge: gitlab\n' > kit.yml
  run --separate-stderr call issue-title 7
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"run: git remote add origin <url>"*"(the project could not be read)" ]]
}

@test "a change's state that GitHub's rate limit stops says which limit, never gh auth status" {
  world github
  fixture limited.txt <<<'gh: You have exceeded a secondary rate limit. Please wait a few minutes before you try again.'
  { printf '%s\t%s\n' 'pr view 5 --json state,mergeCommit*' '!1 limited.txt'
    cat "${FAKE_DIR}/gh.routes"; } > "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/gh.routes"
  run --separate-stderr call pr-state 5
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"GitHub's secondary rate limit refused the read (pull request #5 could not be read)"* ]]
  [[ "$stderr" != *"gh auth status"* ]]
}

@test "a list in pages is read whole" {
  world gitlab
  # glab prints each page's array after the one before
  fixture issues.json <<'JSON'
[{"iid": 7, "title": "Fix the thing", "assignees": [], "created_at": "2026-09-20T09:00:00.000Z"}]
[{"iid": 9, "title": "Other thing", "assignees": [], "created_at": "2026-09-25T09:00:00.000Z"}]
JSON
  run call open-issues
  [ "$(jq --slurp length <<<"$output")" -eq 2 ]
  grep --quiet 'issues?state=opened.* --paginate$' "${FAKE_DIR}/glab.calls"
}

@test "a fork's pipeline that cannot be read is one check, the pipeline's own status" {
  world gitlab
  jq '.head_pipeline = {id: 11, project_id: 99, status: "failed"}' "${FAKE_DIR}/mr.json" > "${FAKE_DIR}/fork.json"
  mv "${FAKE_DIR}/fork.json" "${FAKE_DIR}/mr.json"
  route glab 'api projects/99/pipelines/11/jobs*' '!1'
  run call open-prs
  [ "$(jq --compact-output '[.checks.total, .checks.failing]' <<<"$output")" = '[1,"pipeline"]' ]
}

@test "an answer that is not JSON fails the read" {
  world github
  fixture issues.json <<<'<html>'
  run --separate-stderr call open-issues
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"run: gh auth status (the issues could not be read)"* ]]
}

@test "GitHub's open pull requests are read page after page, each after the cursor the one before ended on" {
  world github
  jq '.data.repository.pullRequests.pageInfo = {hasNextPage: true, endCursor: "p1"}' "${FAKE_DIR}/open-prs.json" \
    > "${FAKE_DIR}/page-1.json"
  jq '.data.repository.pullRequests.nodes[0].number = 5
      | .data.repository.pullRequests.pageInfo = {hasNextPage: false, endCursor: "p2"}' "${FAKE_DIR}/open-prs.json" \
    > "${FAKE_DIR}/page-2.json"
  { printf '%s\t%s\n' 'api graphql*after=p1*' page-2.json 'api graphql*' page-1.json
    cat "${FAKE_DIR}/gh.routes"; } > "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/gh.routes"
  run call open-prs
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output 'map(.number)' <<<"$output")" = '[3,5]' ]
}

@test "the issues a pull request closes are read page after page, past the first 100; a missing pull request fails" {
  world github
  fixture closing-1.json <<'JSON'
{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {
  "pageInfo": {"hasNextPage": true, "endCursor": "i1"},
  "nodes": [{"number": 7, "state": "OPEN", "repository": {"nameWithOwner": "empowerite/example"}}]}}}}}
JSON
  fixture closing-2.json <<'JSON'
{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {
  "pageInfo": {"hasNextPage": false, "endCursor": "i2"},
  "nodes": [{"number": 8, "state": "CLOSED", "repository": {"nameWithOwner": "empowerite/example"}}]}}}}}
JSON
  fixture closing-gone.json <<<'{"data": {"repository": {"pullRequest": null}}}'
  { printf '%s\t%s\n' 'api graphql*number=3 --raw-field after=i1 *' closing-2.json 'api graphql*number=3 *' \
      closing-1.json 'api graphql*number=4 *' closing-gone.json
    cat "${FAKE_DIR}/gh.routes"; } > "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/gh.routes"
  run call pr-closing-issues 3
  [ "$status" -eq 0 ]
  [ "$output" = $'7 OPEN empowerite/example\n8 CLOSED empowerite/example' ]
  run --separate-stderr call pr-closing-issues 4
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"(pull request #4 could not be read)"* ]]
}

# many-world makes pull request #3's review threads and checks run past their first page, its head commit abc, and
# routes the second page of each ahead of the world's routes: the checks' at <checks-page>.
many-world() {
  world github
  jq '.data.repository.pullRequests.nodes[0]
      |= (.reviewThreads.pageInfo = {hasNextPage: true, endCursor: "t1"}
          | .commits.nodes[0].commit.oid = "abc"
          | .commits.nodes[0].commit.statusCheckRollup.contexts.pageInfo = {hasNextPage: true, endCursor: "c1"})' \
    "${FAKE_DIR}/open-prs.json" > "${FAKE_DIR}/many.json"
  mv "${FAKE_DIR}/many.json" "${FAKE_DIR}/open-prs.json"
  fixture threads-2.json <<<'{"data": {"repository": {"pullRequest": {"reviewThreads": {
    "pageInfo": {"hasNextPage": false, "endCursor": "t2"}, "nodes": [{"isResolved": false}]}}}}}'
  fixture checks-2.json <<'JSON'
{"data": {"repository": {"pullRequest": {"commits": {"nodes": [{"commit": {"oid": "abc", "statusCheckRollup": {
  "contexts": {"pageInfo": {"hasNextPage": false, "endCursor": "c2"}, "nodes": [
    {"__typename": "CheckRun", "name": "c", "status": "COMPLETED", "conclusion": "FAILURE",
     "startedAt": "2026-10-01T10:02:00Z", "completedAt": "2026-10-01T10:03:00Z"}]}}}}]}}}}}
JSON
  jq '.data.repository.pullRequest.commits.nodes[0].commit.oid = "def"' "${FAKE_DIR}/checks-2.json" \
    > "${FAKE_DIR}/checks-moved.json"
  { printf '%s\t%s\n' 'api graphql*pr=3 *after=t1 *' threads-2.json 'api graphql*pr=3 *after=c1 *' "$1"
    cat "${FAKE_DIR}/gh.routes"; } > "${FAKE_DIR}/routes.tmp" && mv "${FAKE_DIR}/routes.tmp" "${FAKE_DIR}/gh.routes"
}

@test "a pull request with more review threads or checks than a page holds has the rest of each read" {
  many-world checks-2.json
  run call open-prs
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output '.unresolved' <<<"$output")" -eq 2 ]
  [[ "$(jq --compact-output '.checks' <<<"$output")" == *'"failing":"c"'* ]]
  grep --quiet 'pr=3 --raw-field after=t1 ' "${FAKE_DIR}/gh.calls"
  grep --quiet 'pr=3 --raw-field after=c1 ' "${FAKE_DIR}/gh.calls"
}

@test "the rest of a pull request's checks that cannot be read, or that are another commit's, fails the read" {
  many-world '!1'
  run --separate-stderr call open-prs
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"(the checks of #3 could not be read)"* ]]
  many-world checks-moved.json
  run --separate-stderr call open-prs
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"head commit moved"*"(the checks of #3 could not be read)"* ]]
}

@test "a list that reaches the most kit reads fails, on either forge, saying so" {
  for forge in github gitlab
  do
    world "${forge}"
    # gh is asked for no more than the most kit reads, and gives its two issues
    route gh 'issue list --state open --limit 2 --json *' issues.json
    run --separate-stderr bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge_list_limit=2 \
      && forge:open-issues'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ "$stderr" == *"kit reads at most 2 items of a list (the issues could not be read)"* ]] \
      || { echo "${forge}: ${stderr}"; return 1; }
  done
}

# ----------------------------------------------------------------------------------------------------------------------
