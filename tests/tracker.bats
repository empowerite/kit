#!/usr/bin/env bats
# tracker.bats - a reference names its tracker, the forge's own or a declared one, and each answers the same calls
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}" "${FAKE_DIR}/bin"
  for tool in gh az
  do
    ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/${tool}"
    : > "${FAKE_DIR}/${tool}.routes"
  done
  export PATH="${FAKE_DIR}/bin:${PATH}"
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  git remote add origin https://github.com/empowerite/example.git
  mkdir .kit
  cat > .kit/settings.yml <<'YAML'
trackers:
  - prefix: "AB#"
    provider: ado
    organization: https://dev.azure.com/example
    project: Claims Team
YAML
  unset GITHUB_EVENT_PATH
}

teardown() {
  rm -rf "${repo}"
}

# in_tracker <command>… runs a command with the tracker seam loaded, as a verb would.
in_tracker() {
  bash -c 'source "${kit_root}/bash/1st.sh" && use tracker && "$@"' _ "$@"
}

fixture() { cat > "${FAKE_DIR}/$1"; }
route() { printf '%s\t%s\n' "$2" "$3" >> "${FAKE_DIR}/$1.routes"; }

@test "a reference names its tracker by its prefix, a bare number is the forge's own, and an unknown one is refused" {
  run in_tracker tracker:parse 7
  [ "$output" = "$(printf '#\t7')" ]
  run in_tracker tracker:parse '#7'
  [ "$output" = "$(printf '#\t7')" ]
  run in_tracker tracker:parse 'AB#4521'
  [ "$output" = "$(printf 'AB#\t4521')" ]
  run in_tracker tracker:parse 'JIRA-12'
  [ "$status" -eq 2 ]
  [[ "$output" == *"'JIRA-12' is not an issue reference here; the trackers answer AB#<number> #<number>"* ]]
  run in_tracker tracker:parse 'AB#x'
  [ "$status" -eq 2 ]
}

@test "a reference's key opens its branch name, and a branch name is read back to its reference" {
  run in_tracker tracker:key 7
  [ "$output" = "7" ]
  run in_tracker tracker:key 'AB#4521'
  [ "$output" = "ab4521" ]
  run in_tracker tracker:branch-ref 7-fix-the-thing
  [ "$output" = "#7" ]
  run in_tracker tracker:branch-ref ab4521-fix-the-thing
  [ "$output" = "AB#4521" ]
  run in_tracker tracker:branch-ref topic
  [ -z "$output" ]
  in_tracker tracker:is-forge '#7'
  run ! in_tracker tracker:is-forge 'AB#4521'
}

@test "the body's closing keywords name references to the trackers the forge does not link by itself" {
  body=$'Some text.\nCloses AB#4521\nfixes ab#9 and Resolves: AB#10\nCloses #7'
  run in_tracker tracker:body-refs "${body}"
  [ "$(paste -s -d ' ' - <<<"$output")" = "AB#4521 AB#9 AB#10" ]
}

@test "an Azure DevOps work item answers the tracker's calls through az, its closing states read as CLOSED" {
  fixture active.json <<'JSON'
{"id": 4521, "fields": {"System.Title": "Fix the claim form", "System.State": "Active",
 "System.AssignedTo": {"uniqueName": "bob@example.com"}}}
JSON
  fixture done.json <<<'{"id": 4522, "fields": {"System.Title": "Old", "System.State": "Done"}}'
  fixture created.json <<<'{"id": 4600}'
  fixture me.txt <<<'ada@example.com'
  route az 'boards work-item show --id 4521 --org https://dev.azure.com/example*' active.json
  route az 'boards work-item show --id 4522 *' done.json
  route az 'boards work-item update --id 4521 --assigned-to ada@example.com *' '!0'
  route az 'boards work-item create --title New work --type Task --project Claims Team *' created.json
  route az 'account show*' me.txt
  run in_tracker tracker:of 'AB#4521' title
  [ "$output" = "Fix the claim form" ]
  run in_tracker tracker:of 'AB#4521' state
  [ "$output" = "OPEN" ]
  run in_tracker tracker:of 'AB#4522' state
  [ "$output" = "CLOSED" ]
  run in_tracker tracker:of 'AB#4521' assignees
  [ "$output" = "bob@example.com" ]
  run in_tracker tracker:of 'AB#4521' assign ada@example.com
  [ "$status" -eq 0 ]
  run in_tracker tracker:at 'AB#' me
  [ "$output" = "ada@example.com" ]
  run in_tracker tracker:at 'AB#' create "New work"
  [ "$output" = "4600" ]
  run in_tracker tracker:of 'AB#4521' url
  [ "$output" = "https://dev.azure.com/example/Claims%20Team/_workitems/edit/4521" ]
}

@test "a tracker record without its organization is refused with the setting to add" {
  printf 'trackers:\n  - prefix: "AB#"\n    provider: ado\n' > .kit/settings.yml
  run in_tracker tracker:of 'AB#4521' title
  [ "$status" -ne 0 ]
  [[ "$output" == *"add organization: <https://dev.azure.com/name> to the AB# tracker in .kit/settings.yml"* ]]
}

@test "kit linked takes a work item a closing keyword names in the body, as it takes the forge's own issues" {
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture no-closing.json <<<'{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": []}}}}}'
  fixture body.json <<<'{"body": "Does the work.\nCloses AB#4521"}'
  fixture active.json <<<'{"id": 4521, "fields": {"System.Title": "Fix it", "System.State": "Active"}}'
  route gh 'repo view*' repo.json
  route gh 'api graphql*number=3*' no-closing.json
  route gh 'pr view 3 --json body*' body.json
  route az 'boards work-item show --id 4521 *' active.json
  run "${kit_root}/bin/kit" linked 3
  [ "$status" -eq 0 ]
  [[ "$output" == *"pull request #3 is linked to AB#4521"* ]]
  fixture body.json <<<'{"body": "Does the work."}'
  run "${kit_root}/bin/kit" linked 3
  [ "$status" -eq 1 ]
  [[ "$output" == *"pull request #3 is not linked to any issue"* ]]
  grep --quiet 'Closes <reference>' "${GITHUB_STEP_SUMMARY}"
}

@test "the settings take a tracker with a prefix and a provider, and refuse one without" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  run "${kit_root}/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'trackers:\n  - prefix: "A B#"\n    provider: ado\n' >> .kit/settings.yml
  run "${kit_root}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"a tracker needs a prefix, letters and an optional #, and a provider (prefix 'A B#')"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
