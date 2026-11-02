#!/usr/bin/env bats
# builds.bats - kit flow builds lists the runs no pull request owns: on the default branch, on a tag, and scheduled
# ----------------------------------------------------------------------------------------------------------------------
#
# origin is a bare repository beside the test's, so the tags are real git; gh is tests/fakes/fake-tool, so the stage
# is the provider's folding of a fixture. The world: on main, a push that passed and a schedule still running; on the
# tag release/0.1.0, a push that failed at its release step, on its second attempt; on 9-other, a push left out and a
# manual run cancelled; and a run for a pull request, left out. Each run has its number, commit, and title; a title
# that only repeats the workflow is not drawn.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh
  repo=$(mktemp -d)
  git init --quiet --bare --initial-branch=main "${repo}/origin.git"
  git clone --quiet "${repo}/origin.git" "${repo}/desk" 2>/dev/null
  cd "${repo}/desk"
  mkdir .kit && printf 'forge: github\n' > kit.yml
  git add kit.yml && git commit --quiet --message one && git push --quiet origin main && git remote set-head origin main
  git tag release/0.1.0
  fixture user.json <<<'{"login": "ada"}'
  fixture runs.json <<'JSON'
[{"databaseId": 6, "workflowName": "kit", "headBranch": "7-fix-the-thing", "event": "pull_request",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T11:00:00Z",
  "startedAt": "2026-10-01T11:00:00Z", "updatedAt": "2026-10-01T11:03:00Z", "url": "https://github.com/r/6"},
 {"databaseId": 5, "number": 41, "attempt": 1, "workflowName": "kit", "headBranch": "main", "event": "push",
  "headSha": "8e817bff623de8d1c5976bee0ef99404a4059921", "displayTitle": "#45 status: the builds stage",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T10:00:00Z",
  "startedAt": "2026-10-01T10:00:00Z", "updatedAt": "2026-10-01T10:03:36Z", "url": "https://github.com/r/5"},
 {"databaseId": 4, "number": 7, "attempt": 2, "workflowName": "release", "headBranch": "release/0.1.0", "event": "push",
  "headSha": "1a2b3c4d5e6f708192a3b4c5d6e7f80912a3b4c5", "displayTitle": "release",
  "status": "completed", "conclusion": "failure", "createdAt": "2026-10-01T09:00:00Z",
  "startedAt": "2026-10-01T09:00:00Z", "updatedAt": "2026-10-01T09:04:02Z", "url": "https://github.com/r/4"},
 {"databaseId": 3, "number": 12, "attempt": 1, "workflowName": "nightly", "headBranch": "main", "event": "schedule",
  "headSha": "8e817bff623de8d1c5976bee0ef99404a4059921", "displayTitle": "nightly",
  "status": "in_progress", "conclusion": "", "createdAt": "2026-10-01T08:00:00Z",
  "startedAt": "2026-10-01T08:00:00Z", "updatedAt": "2026-10-01T08:01:00Z", "url": "https://github.com/r/3"},
 {"databaseId": 2, "workflowName": "kit", "headBranch": "9-other", "event": "push",
  "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T07:00:00Z",
  "startedAt": "2026-10-01T07:00:00Z", "updatedAt": "2026-10-01T07:01:00Z", "url": "https://github.com/r/2"},
 {"databaseId": 1, "number": 39, "attempt": 1, "workflowName": "kit", "headBranch": "9-other",
  "event": "workflow_dispatch", "headSha": "5d376dd0e1f2a3b4c5d6e7f8091a2b3c4d5e6f70", "displayTitle": "kit",
  "status": "completed", "conclusion": "cancelled", "createdAt": "2026-10-01T06:00:00Z",
  "startedAt": "2026-10-01T06:00:00Z", "updatedAt": "2026-10-01T06:00:42Z", "url": "https://github.com/r/1"}]
JSON
  fixture jobs.json <<'JSON'
{"jobs": [{"name": "ci", "status": "completed", "conclusion": "success",
           "startedAt": "2026-10-01T10:00:03Z", "completedAt": "2026-10-01T10:03:33Z",
           "steps": [{"name": "checkout", "conclusion": "success"}, {"name": "kit ci", "conclusion": "success"}]}]}
JSON
  fixture failed-jobs.json <<'JSON'
{"jobs": [{"name": "build", "status": "completed", "conclusion": "success",
           "startedAt": "2026-10-01T09:00:03Z", "completedAt": "2026-10-01T09:01:03Z", "steps": []},
          {"name": "publish", "status": "completed", "conclusion": "failure",
           "startedAt": "2026-10-01T09:01:05Z", "completedAt": "2026-10-01T09:03:55Z",
           "steps": [{"name": "checkout", "conclusion": "success"}, {"name": "kit release", "conclusion": "failure"},
                     {"name": "notify", "conclusion": "skipped"}]}]}
JSON
  route 'api user*' user.json
  route 'run list --limit 50 --json *' runs.json
  route 'run view 5 --json jobs' jobs.json
  route 'run view 4 --json jobs' failed-jobs.json
}

@test "kit flow builds draws a row per run: glyph, workflow, number, how it began, branch, commit, title, age, took" {
  run "${kit_root}/bin/kit" flow builds
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 4 ]
  [[ "${lines[0]}" == "✅  kit      #41  push      main           8e817bf  #45 status: the builds stage  "*"d  3m 36s" ]]
  [[ "${lines[1]}" == "❌  release   #7  tag       release/0.1.0  1a2b3c4                                "*"d  4m 02s" ]]
  [[ "${lines[2]}" == "🏃  nightly  #12  schedule  main           8e817bf                                "*"d       …" ]]
  [[ "${lines[3]}" == "🛑  kit      #39  manual    9-other        5d376dd                                "*"d     42s" ]]
  # the rows end in one column
  [ "${#lines[0]}" -eq "${#lines[1]}" ] && [ "${#lines[2]}" -eq "${#lines[0]}" ]
}

@test "kit flow builds --json keeps a push on main or a tag, a schedule, and a manual run, never a pull request's" {
  run "${kit_root}/bin/kit" flow builds --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output 'map([.id, .event, .branch, .tag, .state, .took])' <<<"$output")" \
    = '[[5,"push","main",false,"passed",216],[4,"push","release/0.1.0",true,"failed",242],'"\
"'[3,"schedule","main",false,"running",null],[1,"workflow_dispatch","9-other",false,"cancelled",42]]' ]
  [ "$(jq --compact-output 'map([.number, .attempt, .sha, .title])' <<<"$output")" \
    = '[[41,1,"8e817bf","#45 status: the builds stage"],[7,2,"1a2b3c4","release"],[12,1,"8e817bf","nightly"],'"\
"'[39,1,"5d376dd","kit"]]' ]
  [ "$(jq --compact-output '.[0].jobs' <<<"$output")" = '[{"name":"ci","state":"passed","took":210,"step":null}]' ]
  # a failed job names the step that failed; a step skipped after it does not count
  [ "$(jq --compact-output '.[1].jobs' <<<"$output")" \
    = '[{"name":"build","state":"passed","took":60,"step":null},'"\
"'{"name":"publish","state":"failed","took":170,"step":"kit release"}]' ]
  [ "$(jq --compact-output '[.[2:][].jobs]' <<<"$output")" = '[[],[]]' ]
  [ "$(jq '.[0] | keys_unsorted' --compact-output <<<"$output")" \
    = '["id","number","attempt","workflow","branch","tag","event","sha","title","state","age","took","url","jobs"]' ]
}

@test "a push is a build on the default branch or on a tag the repository has, and on no other branch" {
  git tag --delete release/0.1.0 >/dev/null
  run "${kit_root}/bin/kit" flow builds --json
  [ "$(jq --compact-output 'map(.id)' <<<"$output")" = '[5,3,1]' ]
}

@test "the stage holds the 20 newest runs at most, and only the 5 newest ask for their jobs" {
  jq '[range(30) as $i | .[1] | .databaseId = 100 + $i | .createdAt = "2026-10-01T10:\(10 + $i):00Z"]' \
    "${FAKE_DIR}/runs.json" \
    > "${FAKE_DIR}/many.json"
  mv "${FAKE_DIR}/many.json" "${FAKE_DIR}/runs.json"
  run "${kit_root}/bin/kit" flow builds --json
  [ "$(jq 'length' <<<"$output")" -eq 20 ]
  [ "$(jq '.[0].id' <<<"$output")" -eq 129 ]
  [ "$(grep -c '^run view ' "${FAKE_DIR}/gh.calls")" -eq 5 ]
}

@test "with no build, the stage says so in one dim line, and --json prints []" {
  local esc=$'\x1b'
  fixture runs.json <<<'[{"databaseId": 6, "workflowName": "kit", "headBranch": "7-fix-the-thing",
    "event": "pull_request", "status": "completed", "conclusion": "success", "createdAt": "2026-10-01T11:00:00Z",
    "startedAt": "2026-10-01T11:00:00Z", "updatedAt": "2026-10-01T11:03:00Z", "url": "https://github.com/r/6"}]'
  run "${kit_root}/bin/kit" flow builds
  [ "$status" -eq 0 ]
  [ "$output" = "no build since the last 20 runs" ]
  run env -u NO_COLOR CLICOLOR_FORCE=1 "${kit_root}/bin/kit" flow builds
  [ "$output" = "${esc}[90mno build since the last 20 runs${esc}[0m" ]
  run "${kit_root}/bin/kit" flow builds --json
  [ "$output" = "[]" ]
}

@test "a narrow terminal clips the title, then the branch, then the workflow, and keeps the rest whole" {
  jq '.[1].headBranch = "main" | .[1].workflowName = "a-workflow-with-a-very-long-name-that-goes-on-and-on"
      | .[3].headBranch = "main"' "${FAKE_DIR}/runs.json" > "${FAKE_DIR}/long.json"
  mv "${FAKE_DIR}/long.json" "${FAKE_DIR}/runs.json"
  git tag release/a-tag-with-a-very-long-name-that-goes-on-and-on-and-on-and-on-past-the-edge
  jq '.[2].headBranch = "release/a-tag-with-a-very-long-name-that-goes-on-and-on-and-on-and-on-past-the-edge"' \
    "${FAKE_DIR}/runs.json" > "${FAKE_DIR}/long.json"
  mv "${FAKE_DIR}/long.json" "${FAKE_DIR}/runs.json"
  COLUMNS=80 run "${kit_root}/bin/kit" flow builds
  [ "$status" -eq 0 ]
  local line
  for line in "${lines[@]}"
  do
    # past the glyph, two columns wide, and its gap, every row fills the 76 columns left, and ends in its numbers
    line=${line#*  }
    [ "${#line}" -eq 76 ] || { echo "not 76 wide: ${line}"; return 1; }
    [[ "${line}" == *"s" || "${line}" == *"…" ]]
  done
  [[ "${lines[0]}" == "✅  a-workflow-wit…  #41  push      main        8e817bf  #45 statu…  "* ]]
  [[ "${lines[1]}" == "❌  release           #7  tag       release/a…  1a2b3c4              "* ]]
}

@test "kit help flow holds the builds legend: the stage, and each state's glyph" {
  run "${kit_root}/bin/kit" help flow
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\n  builds:\n\n    🏭  a build: a run no pull request owns\n    🏃  running or queued\n'* ]]
  [[ "$output" == *$'\n    🛑  cancelled or skipped\n'* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
