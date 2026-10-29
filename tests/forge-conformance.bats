#!/usr/bin/env bats
# forge-conformance.bats - every forge provider answers the seam's calls in the same shape, through a fake of its tool
# ----------------------------------------------------------------------------------------------------------------------
#
# One world, seen through each forge's own API: the repository empowerite/example, the caller ada, issue 7 "Fix
# the thing" assigned to bob, and change 3 on branch 7-fix-the-thing, opened by ada, closing issue 7, with two
# checks that passed in a minute and one unresolved review thread. Each provider is driven by tests/fakes/fake-tool
# standing in for its tool, so what is proven is the provider's own folding of its forge's answers into the seam's
# shape. What only a real forge can prove, that the API answers as the fixtures do, is not proven here.

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}" "${FAKE_DIR}/bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/gh"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/glab"
  export PATH="${FAKE_DIR}/bin:${PATH}"
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  unset GITHUB_EVENT_PATH CI_MERGE_REQUEST_IID
}

teardown() {
  rm -rf "${repo}"
}

# call <call> <argument>… runs one call of the seam, as a verb would.
call() {
  bash -c 'source "${kit_root}/bash/1st.sh" && use forge && "forge:$0" "$@"' "$@"
}

# provider prints the provider the seam picks here.
provider() {
  bash -c 'source "${kit_root}/bash/1st.sh" && use forge && forge:provider'
}

# fixture <name> writes stdin to a response file the routes name.
fixture() { cat > "${FAKE_DIR}/$1"; }

# route <tool> <glob> <response> adds a route, the more particular before the more general.
route() { printf '%s\t%s\n' "$2" "$3" >> "${FAKE_DIR}/$1.routes"; }

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
  fixture closing.json <<'JSON'
{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": [
  {"number": 7, "state": "OPEN", "repository": {"nameWithOwner": "empowerite/example"}}]}}}}}
JSON
  fixture open-prs.json <<'JSON'
{"data": {"repository": {"pullRequests": {"nodes": [{
  "number": 3, "title": "Fix the thing", "isDraft": false, "headRefName": "7-fix-the-thing",
  "createdAt": "2026-10-01T09:00:00Z", "mergeStateStatus": "CLEAN", "reviewDecision": null,
  "author": {"login": "ada"},
  "closingIssuesReferences": {"nodes": [{"number": 7, "title": "Fix the thing"}]},
  "reviewThreads": {"nodes": [{"isResolved": true}, {"isResolved": false}]},
  "commits": {"nodes": [{"commit": {"statusCheckRollup": {"contexts": {"nodes": [
    {"__typename": "CheckRun", "name": "a", "status": "COMPLETED", "conclusion": "SUCCESS",
     "startedAt": "2026-10-01T10:00:00Z", "completedAt": "2026-10-01T10:00:30Z"},
    {"__typename": "CheckRun", "name": "b", "status": "COMPLETED", "conclusion": "SUCCESS",
     "startedAt": "2026-10-01T10:00:10Z", "completedAt": "2026-10-01T10:01:00Z"}]}}}}]}}]}}}}
JSON
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'issue view 7 *' issue.json
  route gh 'issue create*' issue-url.txt
  route gh 'issue edit 7 --add-assignee ada' '!0'
  route gh 'pr list --state open --json headRefName*' open-heads.json
  route gh 'pr list --head 7-fix-the-thing --state open*' pr-open.json
  route gh 'pr list --head 7-fix-the-thing --state all*' pr-all.json
  route gh 'pr create --draft --base main --head 7-fix-the-thing --title Fix the thing*' pr-url.txt
  route gh 'pr view 3 --json *' pr.json
  route gh 'api graphql*number=3*' closing.json
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
[{"name": "a", "status": "success",
  "started_at": "2026-10-01T10:00:00.120Z", "finished_at": "2026-10-01T10:00:30.000Z"},
 {"name": "b", "status": "success",
  "started_at": "2026-10-01T10:00:10.000Z", "finished_at": "2026-10-01T10:01:00.450Z"}]
JSON
  fixture discussions.json <<'JSON'
[{"notes": [{"resolvable": true, "resolved": true}]}, {"notes": [{"resolvable": true, "resolved": false}]},
 {"notes": [{"resolvable": false, "resolved": false}]}]
JSON
  fixture closes.json <<'JSON'
[{"iid": 7, "title": "Fix the thing", "state": "opened", "references": {"full": "empowerite/example#7"}}]
JSON
  fixture none.json <<<'[]'
  route glab 'api user' user.json
  route glab "api ${p}/issues/7" issue.json
  route glab 'issue create*' issue-url.txt
  route glab 'issue update 7 --assignee +ada' '!0'
  route glab "api ${p}/merge_requests?state=opened&per_page=100&order_by=*" mrs-list.json
  route glab "api ${p}/merge_requests?state=opened&per_page=100" mrs-open.json
  route glab "api ${p}/merge_requests?state=opened&source_branch=7-fix-the-thing" mr-by-branch.json
  route glab "api ${p}/merge_requests?state=all&source_branch=7-fix-the-thing*" mr-states.json
  route glab "api ${p}/merge_requests/3/closes_issues" "${GITLAB_CLOSES:-closes.json}"
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
  done
}

@test "every provider folds its open changes into the one document kit status draws" {
  expected='{"number":3,"branch":"7-fix-the-thing","title":"Fix the thing","author":"ada",'
  expected+='"state":"comments:open","checks":{"state":"SUCCESS","passed":2,"total":2,"failing":"","took":60},'
  expected+='"unresolved":1,"closes":[{"number":7,"title":"Fix the thing"}]}'
  for forge in github gitlab
  do
    world "${forge}"
    run call open-prs
    [ "$status" -eq 0 ]
    [ "$(jq --compact-output --sort-keys 'del(.age, .createdAt, .ref)' <<<"$output")" \
      = "$(jq --compact-output --sort-keys . <<<"${expected}")" ]
  done
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
  mkdir .kit && printf 'forge: gitlab\n' > .kit/settings.yml
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

# ----------------------------------------------------------------------------------------------------------------------
