#!/usr/bin/env bats
# forge-ado.bats - the Azure DevOps forge answers the seam's calls in the shape GitHub and GitLab answer them
# ----------------------------------------------------------------------------------------------------------------------
# The world of tests/forge-conformance.bats, seen through Azure DevOps: the project empowerite's repository example in
# the organization acme, the caller ada, work item 7 "Fix the thing" assigned to bob, and pull request 3 on branch
# 7-fix-the-thing, opened by ada, linked to work item 7, with two build policies that passed, the slower in fifty
# seconds, and one unresolved thread beside a resolved one and one Azure DevOps wrote itself. tests/fakes/fake-tool
# stands in for az, so what is proven is the provider's folding of Azure DevOps' answers into the seam's shape. What
# only a live organization can prove, that az answers as the fixtures do, is not proven here.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools az gh glab
  fresh-repo
  world
}

# call <call> <argument>… runs one call of the seam, as a verb would.
call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && "forge:$0" "$@"' "$@"
}

# in_tracker <command>… runs a command with the tracker seam loaded, as a verb would.
in_tracker() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use tracker && "$@"' _ "$@"
}

# az-route <glob> <response> routes an az call that names the organization, as every call the provider makes does.
az-route() { route az "$1 --org https://dev.azure.com/acme --detect false --only-show-errors" "$2"; }

world() {
  git remote add origin https://ada@dev.azure.com/acme/empowerite/_git/example
  : > "${FAKE_DIR}/az.routes"
  local list='repos pr list --project empowerite --repository example'
  printf 'ada\n' | fixture me.txt
  fixture item-7.json <<'JSON'
{"id": 7, "fields": {"System.Title": "Fix the thing", "System.State": "Active", "System.Tags": "bug",
 "System.AssignedTo": {"displayName": "Bob", "uniqueName": "bob"}, "System.CreatedDate": "2026-09-20T09:00:00Z"}}
JSON
  fixture items.json <<'JSON'
[{"id": 9, "fields": {"System.Title": "Other thing", "System.State": "New", "System.CreatedDate": "2026-09-25T09:00:00.5Z"}},
 {"id": 7, "fields": {"System.Title": "Fix the thing", "System.State": "Active", "System.Tags": "bug",
  "System.AssignedTo": {"uniqueName": "bob"}, "System.CreatedDate": "2026-09-20T09:00:00Z",
  "System.IterationPath": "empowerite\\Sprint 4"}}]
JSON
  fixture linked.json <<'JSON'
[{"id": 7, "fields": {"System.Title": "Fix the thing", "System.State": "Active", "System.Tags": "bug",
  "System.AssignedTo": {"uniqueName": "bob"}, "System.CreatedDate": "2026-09-20T09:00:00Z"}}]
JSON
  fixture created.json <<<'{"id": 7}'
  fixture pr.json <<'JSON'
{"pullRequestId": 3, "title": "Fix the thing", "description": "Closes #7", "status": "active", "isDraft": false,
 "sourceRefName": "refs/heads/7-fix-the-thing", "targetRefName": "refs/heads/main", "mergeStatus": "succeeded",
 "creationDate": "2026-10-01T09:00:00.123Z", "createdBy": {"uniqueName": "ada"},
 "lastMergeSourceCommit": {"commitId": "1f2e3d4"}, "labels": [{"name": "kind=fix", "active": true}],
 "reviewers": [{"uniqueName": "bob", "displayName": "Bob", "vote": 0}, {"uniqueName": "carol", "vote": 10}]}
JSON
  jq --compact-output '[.]' "${FAKE_DIR}/pr.json" | fixture prs.json
  fixture prs-all.json <<'JSON'
[{"pullRequestId": 1, "status": "abandoned", "lastMergeSourceCommit": {"commitId": "0a0a0a0"}},
 {"pullRequestId": 3, "status": "active", "lastMergeSourceCommit": {"commitId": "1f2e3d4"}}]
JSON
  fixture policies.json <<'JSON'
[{"status": "approved", "configuration": {"isBlocking": true, "type": {"displayName": "Build"}},
  "context": {"buildDefinitionName": "a"},
  "startedDate": "2026-10-01T10:00:00.120Z", "completedDate": "2026-10-01T10:00:30Z"},
 {"status": "approved", "configuration": {"isBlocking": true, "type": {"displayName": "Build"}},
  "context": {"buildDefinitionName": "b"},
  "startedDate": "2026-10-01T10:00:10Z", "completedDate": "2026-10-01T10:01:00.450Z"},
 {"status": "approved", "configuration": {"isBlocking": true, "type": {"displayName": "Minimum number of reviewers"}}},
 {"status": "notApplicable", "configuration": {"isBlocking": false, "type": {"displayName": "Build"}},
  "context": {"buildDefinitionName": "c"}}]
JSON
  fixture threads.json <<'JSON'
{"count": 3, "value": [{"status": "fixed", "comments": [{"commentType": "text"}]},
 {"status": "active", "comments": [{"commentType": "text"}]},
 {"status": "active", "comments": [{"commentType": "system"}]}]}
JSON
  fixture none.json <<<'[]'
  route az 'account show --query user.name --output tsv' me.txt
  route az 'extension show --name azure-devops*' '!0'
  az-route 'boards work-item show --id 7 --output json' item-7.json
  az-route 'boards work-item create --title Fix the thing --type Task --description  --project empowerite --output json' created.json
  az-route 'boards work-item update --id 7 *' '!0'
  az-route 'boards work-item relation add --id 9 --relation-type parent --target-id 7 --output none' '!0'
  az-route "boards query --project empowerite --wiql * --output json" items.json
  az-route "${list} --status active --top 100 --skip 0 --source-branch 7-fix-the-thing --output json" prs.json
  az-route "${list} --status all --top 100 --skip 0 --source-branch 7-fix-the-thing --output json" prs-all.json
  az-route "${list} --status * --top 100 --skip 0 --source-branch * --output json" none.json
  az-route "${list} --status active --top 100 --skip 0 --output json" prs.json
  fixture merged.json <<'JSON'
[{"pullRequestId": 2, "title": "Do the other thing", "status": "completed", "sourceRefName": "refs/heads/9-other",
  "createdBy": {"uniqueName": "bob"}, "closedBy": {"uniqueName": "ada"}, "closedDate": "2026-10-03T09:00:00.1234567Z"}]
JSON
  az-route "${list} --status completed --top 51 --output json" merged.json
  az-route 'repos pr show --id 3 --output json' pr.json
  az-route 'repos pr create --draft true --source-branch 7-fix-the-thing --target-branch main --title Fix the thing --project empowerite --repository example --work-items 7 --output json' pr.json
  az-route 'repos pr work-item list --id 3 --output json' linked.json
  az-route 'repos pr policy list --id 3 --top 10000 --output json' policies.json
  az-route 'devops invoke --area git --resource pullRequestThreads --route-parameters project=empowerite repositoryId=example pullRequestId=3 --api-version 7.1 --output json' threads.json
  az-route 'repos pr update --id 3 *' '!0'
  fixture iterations.json <<'JSON'
[{"id": "a1", "name": "Sprint 1", "path": "empowerite\\Sprint 1",
  "attributes": {"startDate": "2026-11-02T00:00:00Z", "finishDate": "2026-11-15T00:00:00Z", "timeFrame": "current"}},
 {"id": "a0", "name": "Backlog", "path": "empowerite", "attributes": null}]
JSON
  az-route 'boards iteration team list --project empowerite --output json' iterations.json
  az-route 'boards iteration team list --team Alpha --project empowerite --output json' iterations.json
}

# added is what contracts 4 and 5 add that Azure DevOps lacks, as JSON strings: it has iterations, reviewers, the
# changes merged, artifacts, environments, deployments, who has access, teams, and the default branch's policies, and
# no milestone, bot, attestation, alert, waiver, or swimlane.
added='"milestones","bots","attestations","code-scanning","dependabot","secret-scanning","bypasses","swimlanes"'

@test "dev.azure.com, its ssh host, and a host under visualstudio.com are Azure DevOps, and each names the repository" {
  run call provider
  [ "$output" = ado ]
  run call repo
  [ "$output" = empowerite/example ]
  run call repo-url
  [ "$output" = https://dev.azure.com/acme/empowerite/_git/example ]
  run call pr-url 3
  [ "$output" = https://dev.azure.com/acme/empowerite/_git/example/pullrequest/3 ]
  run call noun
  [ "$output" = $'!\tpull request' ]
  for url in git@ssh.dev.azure.com:v3/acme/empowerite/example ssh://git@ssh.dev.azure.com/v3/acme/empowerite/example
  do
    git remote set-url origin "${url}"
    run call repo-url
    [ "$output" = https://dev.azure.com/acme/empowerite/_git/example ] || { echo "${url}: ${output}"; return 1; }
  done
  git remote set-url origin https://acme.visualstudio.com/DefaultCollection/Claims%20Team/_git/example
  run call provider
  [ "$output" = ado ]
  run call repo
  [ "$output" = 'Claims Team/example' ]
  run call repo-url
  [ "$output" = https://acme.visualstudio.com/Claims%20Team/_git/example ]
  # An Azure DevOps Server is the setting; its collection is part of the organization's URL.
  git remote set-url origin https://tfs.example.com/tfs/Main/empowerite/_git/example
  printf 'forge: ado\n' > kit.yml
  run call repo-url
  [ "$output" = https://tfs.example.com/tfs/Main/empowerite/_git/example ]
}

@test "the provider answers the plain calls, and reads and changes work items in az's words" {
  run call me
  [ "$output" = ada ]
  run call issue-title 7
  [ "$output" = "Fix the thing" ]
  run call issue-state 7
  [ "$output" = OPEN ]
  run call issue-assignees 7
  [ "$output" = bob ]
  run call issue-create "Fix the thing"
  [ "$output" = 7 ]
  : > "${FAKE_DIR}/az.calls"
  call issue-assign 7 ada
  call issue-unassign 7 bob
  call issue-close 7 completed
  call issue-close 7 duplicate 9
  call issue-comment 7 "Moved on"
  call issue-add-sub 7 9
  run grep -v 'work-item show' "${FAKE_DIR}/az.calls"
  [ "$(sed 's/ --org https:\/\/dev.azure.com\/acme --detect false --only-show-errors//' <<<"$output")" \
    = "$(printf '%s\n' 'boards work-item update --id 7 --assigned-to ada --output none' \
      'boards work-item update --id 7 --assigned-to  --output none' \
      'boards work-item update --id 7 --state Closed --output none' \
      'boards work-item update --id 7 --discussion Duplicate of #9 --output none' \
      'boards work-item update --id 7 --state Removed --output none' \
      'boards work-item update --id 7 --discussion Moved on --output none' \
      'boards work-item relation add --id 9 --relation-type parent --target-id 7 --output none')" ]
}

@test "a work item opens and closes as forge_settings.ado says, for a process whose states differ" {
  printf 'forge_settings:\n  ado:\n    type: Product Backlog Item\n    closed_state: Done\n' > kit.yml
  az-route 'boards work-item create --title Fix the thing --type Product Backlog Item --description  --project empowerite --output json' \
    created.json
  run call issue-create "Fix the thing"
  [ "$output" = 7 ]
  : > "${FAKE_DIR}/az.calls"
  call issue-close 7 completed
  call issue-close 7 not-planned
  [ "$(grep -o -- '--state [A-Za-z]*' "${FAKE_DIR}/az.calls")" = $'--state Done\n--state Removed' ]
}

@test "the provider finds, opens, and reads a pull request by its branch, as the others do" {
  run call branch-for-issue 7
  [ "$output" = 7-fix-the-thing ]
  run call branch-for-issue 8
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run call pr-for-branch 7-fix-the-thing
  [ "$output" = 3 ]
  run call pr-for-branch 8-nothing
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run call pr-state-for-branch 7-fix-the-thing
  [ "$output" = OPEN ]
  run call pr-head-for-branch 7-fix-the-thing
  [ "$output" = 1f2e3d4 ]
  run call pr-last-for-branch 7-fix-the-thing
  [ "$output" = 3 ]
  run call pr-create 7-fix-the-thing main "Fix the thing"
  [ "$output" = 3 ]
  # the pull request links the work item its branch is named for, so its completion closes it
  grep -q -- '--work-items 7' "${FAKE_DIR}/az.calls"
  run call pr-closing-issues 3
  [ "$output" = "7 OPEN empowerite/example" ]
  run call pr-title 3
  [ "$output" = "Fix the thing" ]
  run call pr-body 3
  [ "$output" = "Closes #7" ]
  run call pr-author 3
  [ "$output" = ada ]
  run call pr-state 3
  [ "$output" = $'OPEN\t' ]
  SYSTEM_PULLREQUEST_PULLREQUESTID=3 run call pr-here
  [ "$output" = 3 ]
}

@test "a completed pull request is merged, with its merge commit, and an abandoned one closed" {
  jq '.status = "completed" | .lastMergeCommit = {commitId: "def456"}' "${FAKE_DIR}/pr.json" | fixture done.json
  jq '.status = "abandoned"' "${FAKE_DIR}/pr.json" | fixture gone.json
  az-route 'repos pr show --id 4 --output json' done.json
  az-route 'repos pr show --id 5 --output json' gone.json
  run call pr-state 4
  [ "$output" = $'MERGED\tdef456' ]
  run call pr-state 5
  [ "$output" = $'CLOSED\t' ]
}

@test "the provider folds its open pull requests into the one document kit flow draws" {
  expected='{"number":3,"branch":"7-fix-the-thing","fork":null,"title":"Fix the thing","author":"ada","pushed":null,'
  expected+='"state":"comments:open","checks":{"state":"SUCCESS","passed":2,"total":2,"failing":"","took":50},'
  expected+='"unresolved":1,"labels":[{"name":"kind=fix","color":null}],'
  expected+='"closes":[{"number":7,"title":"Fix the thing","assignees":["bob"],'
  expected+='"labels":[{"name":"bug","color":null}]}],'
  # a vote of 10 approves; 0 is a reviewer asked, who has not voted; Azure DevOps keeps no time of a vote
  expected+='"reviewers":[{"login":"bob","state":"requested","at":null},'
  expected+='{"login":"carol","state":"approved","at":null}],'
  expected+='"runs":[{"name":"a","verdict":"passed","at":1790848800,"took":30},'
  expected+='{"name":"b","verdict":"passed","at":1790848810,"took":50}]}'
  run call open-prs
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --sort-keys \
    'del(.age, .createdAt, .ref, .source) | .closes |= map(del(.age, .source, .organization, .repository))' \
    <<<"$output")" = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
  [ "$(jq --raw-output '"\(.ref) \(.source)"' <<<"$output")" = '!3 ado' ]
  [ "$(jq --raw-output '[.closes[] | "\(.source) \(.organization)/\(.repository)"] | join(" ")' <<<"$output")" \
    = 'ado empowerite/example' ]
  # a draft is drawn as one, and a blocking review policy that has not approved blocks the merge
  jq '.[0].isDraft = true' "${FAKE_DIR}/prs.json" > "${FAKE_DIR}/prs.new" \
    && mv "${FAKE_DIR}/prs.new" "${FAKE_DIR}/prs.json"
  forge-forget
  [ "$(call open-prs | jq --raw-output .state)" = draft ]
  jq '.[0].isDraft = false' "${FAKE_DIR}/prs.json" > "${FAKE_DIR}/prs.new" \
    && mv "${FAKE_DIR}/prs.new" "${FAKE_DIR}/prs.json"
  jq '.value |= map(.status = "fixed")' "${FAKE_DIR}/threads.json" > "${FAKE_DIR}/threads.new" \
    && mv "${FAKE_DIR}/threads.new" "${FAKE_DIR}/threads.json"
  jq '.[2].status = "queued"' "${FAKE_DIR}/policies.json" > "${FAKE_DIR}/policies.new" \
    && mv "${FAKE_DIR}/policies.new" "${FAKE_DIR}/policies.json"
  forge-forget
  [ "$(call open-prs | jq --raw-output .state)" = blocked:approval ]
}

@test "the provider lists the pull requests completed last, as the others do, and says when it stopped at 50" {
  expected='{"number":2,"title":"Do the other thing","author":"bob","branch":"9-other","merged":1791018000,"by":"ada",'
  expected+='"url":"https://dev.azure.com/acme/empowerite/_git/example/pullrequest/2"}'
  run --separate-stderr call merged-prs
  [ "$status" -eq 0 ]
  [ "$output" = "${expected}" ]
  jq '[range(51) as $i | .[0] | .pullRequestId = (60 - $i)]' "${FAKE_DIR}/merged.json" > "${FAKE_DIR}/merged.new" \
    && mv "${FAKE_DIR}/merged.new" "${FAKE_DIR}/merged.json"
  run --separate-stderr call merged-prs
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].number, .[49].number, .[50]]' <<<"$output")" \
    = '[51,60,11,{"limited":50}]' ]
}

@test "the provider lists its open work items in one shape, oldest first, with the assignee, the tags, and the repository" {
  expected='[{"number":7,"ref":"#7","title":"Fix the thing","assignees":["bob"],'
  expected+='"labels":[{"name":"bug","color":null}],"milestone":null,"iteration":"Sprint 4","swimlane":null},'
  expected+='{"number":9,"ref":"#9","title":"Other thing","assignees":[],"labels":[],'
  expected+='"milestone":null,"iteration":null,"swimlane":null}]'
  run call open-issues
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --sort-keys --slurp 'map(del(.age, .createdAt, .source, .organization, .repository))' \
    <<<"$output")" = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
  [ "$(jq --raw-output --slurp 'map("\(.source) \(.organization)/\(.repository)") | unique | join(" ")' <<<"$output")" \
    = 'ado empowerite/example' ]
}

@test "the provider lists the pipeline runs not owned by any pull request, in the shape the others do" {
  git commit --quiet --allow-empty --message one && git tag release/0.1.0
  fixture runs.json <<'JSON'
[{"id": 104, "buildNumber": "50", "sourceVersion": "aaaaaaa0000000000", "definition": {"name": "kit"},
  "sourceBranch": "refs/heads/7-fix-the-thing", "reason": "pullRequest", "status": "completed", "result": "succeeded",
  "queueTime": "2026-10-01T11:00:00Z", "startTime": "2026-10-01T11:00:00Z", "finishTime": "2026-10-01T11:03:00Z"},
 {"id": 103, "buildNumber": "49", "sourceVersion": "8e817bf0000000000", "definition": {"name": "kit"},
  "sourceBranch": "refs/heads/main", "reason": "individualCI", "status": "completed", "result": "succeeded",
  "queueTime": "2026-10-01T10:00:00.123000+00:00", "startTime": "2026-10-01T12:00:00.500000+02:00",
  "finishTime": "2026-10-01T10:03:36.500000+00:00"},
 {"id": 102, "buildNumber": "48", "sourceVersion": "1a2b3c40000000000", "definition": {"name": "kit"},
  "sourceBranch": "refs/tags/release/0.1.0", "reason": "individualCI", "status": "completed", "result": "failed",
  "queueTime": "2026-10-01T09:00:00Z", "startTime": "2026-10-01T09:00:00Z", "finishTime": "2026-10-01T09:04:02Z"},
 {"id": 101, "buildNumber": "47", "sourceVersion": "8e817bf0000000000", "definition": {"name": "kit"},
  "sourceBranch": "refs/heads/main", "reason": "schedule", "status": "inProgress",
  "queueTime": "2026-10-01T08:00:00Z", "startTime": "2026-10-01T08:00:00Z"},
 {"id": 100, "buildNumber": "46", "sourceVersion": "bbbbbbb0000000000", "definition": {"name": "kit"},
  "sourceBranch": "refs/heads/9-other", "reason": "individualCI", "status": "completed", "result": "succeeded",
  "queueTime": "2026-10-01T07:00:00Z", "startTime": "2026-10-01T07:00:00Z", "finishTime": "2026-10-01T07:01:00Z"},
 {"id": 99, "buildNumber": "45", "sourceVersion": "5d376dd0000000000", "definition": {"name": "kit"},
  "sourceBranch": "refs/heads/9-other", "reason": "manual", "status": "completed", "result": "canceled",
  "queueTime": "2026-10-01T06:00:00Z", "startTime": "2026-10-01T06:00:00Z", "finishTime": "2026-10-01T06:00:42Z"}]
JSON
  az-route 'pipelines runs list --project empowerite --top 50 --query-order QueueTimeDesc --output json' runs.json
  expected='[{"id":103,"number":49,"attempt":1,"workflow":"kit","branch":"main","tag":false,"event":"push",'
  expected+='"sha":"8e817bf","state":"passed","took":216},'
  expected+='{"id":102,"number":48,"attempt":1,"workflow":"kit","branch":"release/0.1.0","tag":true,"event":"push",'
  expected+='"sha":"1a2b3c4","state":"failed","took":242},'
  expected+='{"id":101,"number":47,"attempt":1,"workflow":"kit","branch":"main","tag":false,"event":"schedule",'
  expected+='"sha":"8e817bf","state":"running","took":null},'
  expected+='{"id":99,"number":45,"attempt":1,"workflow":"kit","branch":"9-other","tag":false,'
  expected+='"event":"workflow_dispatch","sha":"5d376dd","state":"cancelled","took":42}]'
  run call runs
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --sort-keys 'map(del(.age, .url, .jobs, .title))' <<<"$output")" \
    = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
  [ "$(jq 'map(.age > 0 and .jobs == []) | all' <<<"$output")" = true ]
  # az drops a run's _links, so its page is built from the organization, the project, and the run's id
  [ "$(jq --raw-output '.[0].url' <<<"$output")" = 'https://dev.azure.com/acme/empowerite/_build/results?buildId=103' ]
}

@test "Azure DevOps lacks rate limits, marks, releases, and linked branches, and says so in its declaration" {
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$status" -eq 0 ]
  [ "$(jq -c '{forge, provider, none: [.features | to_entries[] | select(.value == "none") | .key]}' <<<"$output")" \
    = "{\"forge\":\"ado\",\"provider\":5,\"none\":[\"linked-branches\",\"quota\",\"marks\",\"releases\",${added}]}" ]
  for c in quota marks releases 'link-branch 7 main'
  do
    read -r -a words <<<"${c}" && run --separate-stderr call "${words[@]}"
    [ "$status" -eq 3 ] || { echo "${c}: ${status}"; return 1; }
  done
  run "${kit_root}/bin/kit" flow --quota
  [ "$output" = '[]' ]
}

@test "Azure DevOps lists its iterations of the project's default team, or the team forge_settings.ado.team names" {
  expected='{"id":"a1","title":"Sprint 1","start":1793577600,"end":1794787199,'
  expected+='"url":"https://dev.azure.com/acme/empowerite/_sprints"}'
  run call iterations
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --slurp 'map(select(.title == "Sprint 1"))[0]' <<<"$output")" = "${expected}" ]
  [ "$(jq --compact-output 'select(.title == "Backlog") | [.start, .end]' <<<"$output")" = '[null,null]' ]
  printf 'forge_settings:\n  ado:\n    team: Alpha\n' > kit.yml
  : > "${FAKE_DIR}/az.calls"
  run call iterations
  [ "$status" -eq 0 ]
  grep --quiet -- '--team Alpha' "${FAKE_DIR}/az.calls"
  run --separate-stderr call milestones
  [ "$status" -eq 3 ]
}

# delivery-world adds the world's delivery, seen through Azure DevOps: run 103 of release/0.1.0 by ada published
# kit-0.1.0.tar.gz and deployed to production, which an approval by ada and a branch control of main protect.
delivery-world() {
  local runs='pipelines runs list --project empowerite --top' env='devops invoke --area distributedtask'
  fixture delivery-runs.json <<'JSON'
[{"id": 103, "buildNumber": "49", "sourceBranch": "refs/heads/release/0.1.0",
  "sourceVersion": "8e817bf000000000000000000000000000000000", "requestedFor": {"uniqueName": "ada"},
  "repository": {"id": "r1", "name": "example", "type": "TfsGit"}, "status": "completed", "result": "succeeded",
  "queueTime": "2026-10-01T09:50:00Z", "startTime": "2026-10-01T09:50:10Z", "finishTime": "2026-10-01T10:00:00Z"}]
JSON
  fixture run-artifacts.json <<'JSON'
[{"id": 13, "name": "kit-0.1.0.tar.gz", "source": "3dc8fd7e-4368-5a92-293e-d53cefc8c4b3",
  "resource": {"type": "PipelineArtifact", "data": "4FD1F5EFB0F2D1DD", "properties": {"artifactsize": "2048"},
               "url": "https://dev.azure.com/acme/empowerite/_apis/build/builds/103/artifacts?artifactName=kit",
               "downloadUrl": "https://artprodcus3.artifacts.visualstudio.com/x/content?format=zip"}}]
JSON
  fixture environments.json <<'JSON'
{"count": 2, "value": [
  {"id": 1, "name": "production", "createdOn": "2026-09-01T09:00:00Z", "lastModifiedOn": "2026-10-01T10:00:00Z"},
  {"id": 2, "name": "staging", "createdOn": "2026-09-01T09:00:00Z", "lastModifiedOn": "2026-09-02T10:00:00Z"}]}
JSON
  fixture checks.json <<'JSON'
{"count": 2, "value": [
  {"id": 14, "type": {"id": "8c6f20a7-a545-4486-9777-f762fafe0d4d", "name": "Approval"},
   "resource": {"type": "environment", "id": "1", "name": "production"},
   "settings": {"approvers": [{"displayName": "Ada", "uniqueName": "ada", "id": "a1"}], "minRequiredApprovers": 1}},
  {"id": 12, "type": {"id": "fe1de3ee-a436-41b4-bb20-f6eb4cb879a7", "name": "Task Check"},
   "resource": {"type": "environment", "id": "1", "name": "production"},
   "settings": {"definitionRef": {"id": "86b05a0c-73e6-4f7d-b3cf-e38f3b39a75b", "name": "evaluatebranchProtection"},
                "inputs": {"allowedBranches": "refs/heads/main"}}}]}
JSON
  fixture records-1.json <<'JSON'
{"count": 1, "value": [
  {"id": 9, "environmentId": 1, "owner": {"id": 103, "name": "49"}, "definition": {"id": 4, "name": "kit"},
   "planType": "Build", "stageName": "deploy", "jobName": "deploy", "queueTime": "2026-10-01T09:59:00Z",
   "startTime": "2026-10-01T10:00:00Z", "finishTime": "2026-10-01T10:05:00Z", "result": "succeeded"}]}
JSON
  fixture records-none.json <<<'{"count": 0, "value": []}'
  az-route "${runs} 21 --query-order QueueTimeDesc --output json" delivery-runs.json
  az-route "${runs} 100 --query-order QueueTimeDesc --output json" delivery-runs.json
  az-route 'pipelines runs artifact list --run-id * --project empowerite --output json' run-artifacts.json
  az-route "${env} --resource environments --route-parameters project=empowerite --query-parameters \$top=101 \
--api-version 7.1 --output json" environments.json
  az-route 'devops invoke --area PipelinesChecks --resource queryconfigurations --route-parameters project=empowerite '\
'--query-parameters $expand=settings --http-method POST --in-file * --api-version 7.1-preview.1 --output json' \
    checks.json
  az-route "${env} --resource environmentdeploymentrecords --route-parameters project=empowerite environmentId=1 \
--query-parameters top=51 --api-version 7.1 --output json" records-1.json
  az-route "${env} --resource environmentdeploymentrecords --route-parameters project=empowerite environmentId=* \
--query-parameters top=51 --api-version 7.1 --output json" records-none.json
}

# ado-first <glob> <response> puts a route for az before every other, so it answers before delivery-world's own.
ado-first() {
  { printf '%s\t%s\n' "$1" "$2"; cat "${FAKE_DIR}/az.routes"; } > "${FAKE_DIR}/az.routes.new"
  mv "${FAKE_DIR}/az.routes.new" "${FAKE_DIR}/az.routes"
}

@test "the provider lists the artifacts of the 20 newest runs, as the others do, with no digest Azure DevOps keeps" {
  delivery-world
  expected='{"id":"103/13","name":"kit-0.1.0.tar.gz","digest":null,"size":2048,"run":103,"expires":null,'
  expected+='"url":"https://dev.azure.com/acme/empowerite/_build/results?buildId=103'
  expected+='&view=artifacts&type=publishedArtifacts"}'
  run --separate-stderr call artifacts
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq --compact-output 'del(.age)' <<<"$output")" = "${expected}" ]
  [ "$(jq --raw-output '.age > 0' <<<"$output")" = true ]
  # a 21st run says the list stopped at the artifacts of the 20 newest, each run's read once
  jq '[range(21) as $i | .[0] | .id = (200 - $i)]' "${FAKE_DIR}/delivery-runs.json" | fixture runs-21.json
  : > "${FAKE_DIR}/az.calls"
  ado-first 'pipelines runs list --project empowerite --top 21 *' runs-21.json
  run --separate-stderr call artifacts
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].run, .[19].run, .[20]]' <<<"$output")" \
    = '[21,200,181,{"limited":20}]' ]
  [ "$(grep -c 'runs artifact list' "${FAKE_DIR}/az.calls")" -eq 20 ]
  # past 100 artifacts, those of the newest runs are drawn and the list says it stopped, the runs past unread
  jq '[range(6) as $i | .[0] | .id = $i]' "${FAKE_DIR}/run-artifacts.json" | fixture run-artifacts-6.json
  ado-first 'pipelines runs artifact list *' run-artifacts-6.json
  : > "${FAKE_DIR}/az.calls"
  run --separate-stderr call artifacts
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].id, .[99].id, .[100]]' <<<"$output")" \
    = '[101,"200/0","184/3",{"limited":100}]' ]
  [ "$(grep -c 'runs artifact list' "${FAKE_DIR}/az.calls")" -eq 17 ]
}

@test "the provider lists its environments, their approvals and branch controls read in one request for them all" {
  delivery-world
  expected='{"name":"production","protection":["branches","reviewers"],"approvers":["ada"],"branches":["main"],'
  expected+='"url":"https://dev.azure.com/acme/empowerite/_environments/1"}'
  : > "${FAKE_DIR}/az.calls"
  run --separate-stderr call environments
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(head -n 1 <<<"$output")" = "${expected}" ]
  [ "$(jq --compact-output 'select(.name == "staging") | [.protection, .approvers, .branches]' <<<"$output")" \
    = '[[],[],null]' ]
  [ "$(grep -c 'queryconfigurations' "${FAKE_DIR}/az.calls")" -eq 1 ]
  # more environments than kit reads fails the read, as it would be partial
  jq '.value = [range(101) as $i | .value[0] | .id = $i]' "${FAKE_DIR}/environments.json" | fixture envs-101.json
  ado-first 'devops invoke --area distributedtask --resource environments *' envs-101.json
  run --separate-stderr call environments
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"kit reads at most 100 environments (the environments could not be read)"* ]]
}

@test "the provider lists the deployments of 20 environments, the newest 50, each with its run's branch and commit" {
  delivery-world
  expected='{"id":9,"environment":"production","ref":"release/0.1.0","sha":"8e817bf000000000000000000000000000000000",'
  expected+='"state":"success","actor":"ada","url":"https://dev.azure.com/acme/empowerite/_build/results?buildId=103",'
  expected+='"commit":"https://dev.azure.com/acme/empowerite/_git/example/commit/'
  expected+='8e817bf000000000000000000000000000000000"}'
  run --separate-stderr call deployments
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq --compact-output 'del(.age)' <<<"$output")" = "${expected}" ]
  # two environments' 60 deployments give the newest 50, then limited
  for e in 1 2
  do
    jq --argjson e "${e}" '.value = [range(30) as $i | .value[0] | .id = ($e * 100 + $i) | .environmentId = $e
      | .startTime = "2026-10-01T10:\(10 + $i):0\($e)Z"]' "${FAKE_DIR}/records-1.json" | fixture "records-30-${e}.json"
    ado-first "devops invoke --area distributedtask --resource environmentdeploymentrecords \
--route-parameters project=empowerite environmentId=${e} *" "records-30-${e}.json"
  done
  run --separate-stderr call deployments
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[0].id, .[1].id, .[50]]' <<<"$output")" = '[51,229,129,{"limited":50}]' ]
  # one environment that holds more than 50 gives its 51st, and the list says it stopped
  jq '.value = [range(51) as $i | .value[0] | .id = (300 + $i)]' "${FAKE_DIR}/records-1.json" | fixture records-51.json
  ado-first "devops invoke --area distributedtask --resource environmentdeploymentrecords \
--route-parameters project=empowerite environmentId=1 *" records-51.json
  ado-first "devops invoke --area distributedtask --resource environmentdeploymentrecords \
--route-parameters project=empowerite environmentId=2 *" records-none.json
  run --separate-stderr call deployments
  [ "$status" -eq 0 ]
  [ "$(jq --slurp --compact-output '[length, .[50]]' <<<"$output")" = '[51,{"limited":50}]' ]
  grep --quiet 'top=51' "${FAKE_DIR}/az.calls"
  # past 20 environments, the 20 changed last are read, and the list says it stopped at their deployments
  jq '.value = [range(21) as $i | .value[0] | .id = (1 + $i) | .name = "env-\($i)"
                | .lastModifiedOn = "2026-10-\(10 + $i)T10:00:00Z"]' "${FAKE_DIR}/environments.json" \
    | fixture envs-21.json
  ado-first 'devops invoke --area distributedtask --resource environments *' envs-21.json
  for e in $(seq 2 21)
  do
    jq --argjson e "${e}" '.value[0] |= (.id = 100 + $e | .environmentId = $e
      | .startTime = "2026-10-\(10 + $e)T10:00:00Z")' "${FAKE_DIR}/records-1.json" | fixture "records-${e}.json"
    ado-first "devops invoke --area distributedtask --resource environmentdeploymentrecords \
--route-parameters project=empowerite environmentId=${e} *" "records-${e}.json"
  done
  : > "${FAKE_DIR}/az.calls"
  run --separate-stderr call deployments
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(grep -c 'environmentdeploymentrecords' "${FAKE_DIR}/az.calls")" -eq 20 ]
  ! grep --quiet 'environmentId=1 ' "${FAKE_DIR}/az.calls"
  [ "$(jq --slurp --compact-output '[length, .[0].id, .[0].environment, .[19].id, .[20]]' <<<"$output")" \
    = '[21,121,"env-20",102,{"limited":20}]' ]
}

# people-world adds to the world's routes the project's teams: its default team, empowerite Team, holds ada, its
# administrator, and core holds ada, bob, and the group Release managers.
people-world() {
  local team='devops team list-member --team'
  fixture teams.json <<'JSON'
[{"id": "t1", "name": "empowerite Team", "description": "The default project team.", "projectName": "empowerite"},
 {"id": "t2", "name": "core", "description": "", "projectName": "empowerite"}]
JSON
  fixture team-1.json <<'JSON'
[{"isTeamAdmin": true, "identity": {"id": "a1", "displayName": "Ada", "uniqueName": "ada"}}]
JSON
  fixture team-2.json <<'JSON'
[{"isTeamAdmin": false, "identity": {"id": "a1", "displayName": "Ada", "uniqueName": "ada"}},
 {"isTeamAdmin": false, "identity": {"id": "b2", "displayName": "Bob", "uniqueName": "bob"}},
 {"isTeamAdmin": false, "identity": {"id": "g3", "displayName": "Release managers",
  "uniqueName": "[acme]\\Release managers", "isContainer": true}}]
JSON
  az-route 'devops team list --project empowerite --top 21 --output json' teams.json
  az-route "${team} t1 --project empowerite --top 101 --output json" team-1.json
  az-route "${team} t2 --project empowerite --top 101 --output json" team-2.json
  az-route "${team} * --project empowerite --top 101 --output json" team-1.json
}

@test "the provider lists the project's teams, each with its members, as the others do, 20 teams at most" {
  people-world
  run --separate-stderr call teams
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq --compact-output '[.team, .name, .role, .members]' <<<"$output" | paste -sd, -)" \
    = '["empowerite Team","empowerite Team",null,["ada"]],'\
'["core","core",null,["ada","bob","[acme]\\Release managers"]]' ]
  [ "$(jq --raw-output .url <<<"$output" | sort -u)" = https://dev.azure.com/acme/empowerite/_settings/teams ]
  # more than 20 teams give the first 20, and a team of more than 100 members its first 100, each saying so
  jq '[range(21) as $i | .[0] | .id = "t\(100 + $i)" | .name = "team \(100 + $i)"]' "${FAKE_DIR}/teams.json" \
    | fixture teams-21.json
  ado-first 'devops team list --project empowerite --top 21 *' teams-21.json
  run --separate-stderr call teams
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq --slurp --compact-output '[length, .[0].team, .[19].team, .[20]]' <<<"$output")" \
    = '[21,"team 100","team 119",{"limited":20}]' ]
  jq '[range(101) as $i | .[1] | .identity.uniqueName = "u\($i)"]' "${FAKE_DIR}/team-2.json" | fixture team-101.json
  ado-first 'devops team list-member --team t2 *' team-101.json
  ado-first 'devops team list --project empowerite --top 21 *' teams.json
  run --separate-stderr call teams
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq --slurp --compact-output '[length, (.[1].members | length), .[2]]' <<<"$output")" = '[3,100,{"limited":2}]' ]
}

@test "the provider lists who has access as its teams' members, an administrator of one admin, a group a group" {
  people-world
  run --separate-stderr call collaborators
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  [ "$(jq --compact-output '[.login, .name, .role, .kind]' <<<"$output" | paste -sd, -)" \
    = '["ada","Ada","admin","user"],["bob","Bob","member","user"],'\
'["[acme]\\Release managers","Release managers","member","group"]' ]
  # Azure DevOps names none of its bots, and kit asks it for none
  run --separate-stderr call bots
  [ "$status" -eq 3 ]
  [ -z "$output" ]
  [ -z "$stderr" ]
}

@test "the provider lists the default branch's policies as its controls, each enforced, optional, or off" {
  fixture repo.json <<<'{"id": "r-1", "name": "example", "defaultBranch": "refs/heads/main"}'
  fixture branch-policies.json <<'JSON'
[{"id": 21, "isEnabled": true, "isBlocking": true, "isDeleted": false,
  "type": {"id": "fa4e907d-c16b-4a4c-9dfa-4906e5d171dd", "displayName": "Minimum number of reviewers"},
  "settings": {"minimumApproverCount": 2, "creatorVoteCounts": false,
               "scope": [{"refName": "refs/heads/main", "matchKind": "Exact", "repositoryId": "r-1"}]}},
 {"id": 22, "isEnabled": true, "isBlocking": false, "type": {"displayName": "Build"},
  "settings": {"buildDefinitionId": 5, "displayName": "ci", "queueOnSourceUpdateOnly": true, "validDuration": 0,
               "scope": [{"refName": "refs/heads/main", "matchKind": "Exact", "repositoryId": "r-1"}]}},
 {"id": 23, "isEnabled": false, "isBlocking": true, "type": {"displayName": "Comment requirements"},
  "settings": {"scope": [{"refName": "refs/heads/", "matchKind": "Prefix", "repositoryId": null}]}}]
JSON
  az-route 'repos show --repository example --project empowerite --output json' repo.json
  az-route 'repos policy list --branch main --repository-id r-1 --project empowerite --output json' \
    branch-policies.json
  run --separate-stderr call rulesets
  [ "$status" -eq 0 ] || { echo "${stderr}"; return 1; }
  local url='"url":"https://dev.azure.com/acme/empowerite/_settings/repositories?repo=r-1&_a=policiesMid'
  url+='&refs=refs/heads/main"}'
  [ "$output" = '{"id":21,"control":"Minimum number of reviewers","target":["refs/heads/main"],"enforcement":"active",'\
'"rules":["minimumApproverCount=2"],"source":"protection",'"${url}"$'\n'\
'{"id":22,"control":"Build","target":["refs/heads/main"],"enforcement":"evaluate",'\
'"rules":["buildDefinitionId=5","displayName=ci","queueOnSourceUpdateOnly=true"],"source":"protection",'"${url}"$'\n'\
'{"id":23,"control":"Comment requirements","target":["refs/heads/*"],"enforcement":"disabled","rules":[],'\
'"source":"protection",'"${url}" ]
  # Azure DevOps keeps none of the attestations, alerts, or waivers kit reads
  for c in attestations code-scanning-alerts dependabot-alerts secret-scanning-alerts bypasses
  do
    run --separate-stderr call "${c}"
    [ "$status" -eq 3 ] || { echo "${c}: ${status}"; return 1; }
  done
}

@test "on an Azure DevOps forge, #7 and AB#7 are both the forge's own work item, and a record still claims AB#" {
  run in_tracker tracker:parse 'AB#7'
  [ "$output" = $'AB#\t7' ]
  run in_tracker tracker:provider 'AB#'
  [ "$output" = forge ]
  run in_tracker tracker:of 'AB#7' title
  [ "$output" = "Fix the thing" ]
  run in_tracker tracker:of 7 title
  [ "$output" = "Fix the thing" ]
  run in_tracker tracker:key 'AB#7'
  [ "$output" = 7 ]
  run in_tracker tracker:of 7 url
  [ "$output" = https://dev.azure.com/acme/empowerite/_workitems/edit/7 ]
  run in_tracker tracker:at '#' new-url "Fix it"
  [ "$output" = 'https://dev.azure.com/acme/empowerite/_workitems/create/Task?%5BSystem.Title%5D=Fix%20it' ]
  printf 'trackers:\n  - prefix: "AB#"\n    provider: web\n    url: https://boards.example/{id}\n' > kit.yml
  run in_tracker tracker:provider 'AB#'
  [ "$output" = web ]
}

@test "Azure DevOps that cannot answer fails every read with one line, and never answers nothing" {
  printf 'account show*\tme.txt\nextension show*\t!0\n' > "${FAKE_DIR}/az.routes"
  for c in 'branch-for-issue 7' 'pr-for-branch 7-fix-the-thing' 'pr-last-for-branch 7-fix-the-thing' \
    'pr-state-for-branch 7-fix-the-thing' 'pr-head-for-branch 7-fix-the-thing' 'pr-title 3' 'pr-body 3' \
    'pr-author 3' 'pr-closing-issues 3' 'pr-state 3' 'issue-state 7' 'issue-assignees 7' 'issue-title 7' open-prs \
    open-issues runs
  do
    read -r -a words <<<"${c}" && run --separate-stderr call "${words[@]}"
    [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"run: az login ("*"could not be read)"* ]] \
      || { echo "${c}: exit ${status}, printed '${output}', said '${stderr}'"; return 1; }
  done
  : > "${FAKE_DIR}/az.routes"
  run --separate-stderr call me
  [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"run: az login (the caller's login could not be read)"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
