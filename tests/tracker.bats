#!/usr/bin/env bats
# tracker.bats - a reference names its tracker, the forge's own or a declared one, and each answers the same calls
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh az
  fresh-repo
  git remote add origin https://github.com/empowerite/example.git
  mkdir .kit
  cat > kit.yml <<'YAML'
trackers:
  - prefix: "AB#"
    provider: ado
    organization: https://dev.azure.com/example
    project: Claims Team
YAML
}

# in_tracker <command>… runs a command with the tracker seam loaded, as a verb would.
in_tracker() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use tracker && "$@"' _ "$@"
}

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

@test "trackers that cannot be read fail a reference with 1, and a reference no tracker answers is a usage error, 2" {
  printf 'trackers: [\n' > kit.yml
  for call in "tracker:parse AB#4521" "tracker:normal AB#4521" "tracker:key AB#4521" "tracker:of AB#4521 title"
  do
    # shellcheck disable=SC2086  # each holds its words
    run in_tracker ${call}
    [ "$status" -eq 1 ]
  done
  [ ! -s "${FAKE_DIR}/az.calls" ]
  run "${kit_root}/bin/kit" assign 'AB#4521'
  [ "$status" -eq 1 ]
  run "${kit_root}/bin/kit" close --not-planned 'AB#4521'
  [ "$status" -eq 1 ]
  printf 'trackers: []\n' > kit.yml
  for call in "tracker:normal JIRA-12" "tracker:key JIRA-12" "tracker:of JIRA-12 title"
  do
    # shellcheck disable=SC2086  # each holds its words
    run in_tracker ${call}
    [ "$status" -eq 2 ]
  done
  run "${kit_root}/bin/kit" assign 'JIRA-12'
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
  route az 'boards work-item update --id 4521 --assigned-to  *' '!0'
  route az 'boards work-item update --id 4521 --state *' '!0'
  route az 'boards work-item update --id 4521 --discussion *' '!0'
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
  # unassigning someone not assigned changes nothing; closing sets the closing state, a duplicate after a comment
  : > "${FAKE_DIR}/az.calls"
  in_tracker tracker:of 'AB#4521' unassign carol@example.com
  in_tracker tracker:of 'AB#4521' unassign bob@example.com
  in_tracker tracker:of 'AB#4521' close completed
  in_tracker tracker:of 'AB#4521' close not-planned
  in_tracker tracker:of 'AB#4521' close duplicate 4522
  run grep -o 'update --id 4521 --[a-z-]* [^-]*' "${FAKE_DIR}/az.calls"
  [ "$output" = $'update --id 4521 --assigned-to  \nupdate --id 4521 --state Closed \nupdate --id 4521 --state Removed \nupdate --id 4521 --discussion Duplicate of AB#4522 \nupdate --id 4521 --state Removed ' ]
}

@test "each tracker gives the page a new issue is filed at, its title in the query, and a web tracker refuses the rest" {
  fixture repo.json <<<'{"url": "https://github.com/empowerite/example"}'
  route gh 'repo view*' repo.json
  cat >> kit.yml <<'YAML'
  - prefix: "LIN#"
    provider: web
    new: https://linear.app/example/team/ENG/new?title={title}
    url: https://linear.app/example/issue/ENG-{id}
  - prefix: "BZ#"
    provider: web
YAML
  run in_tracker tracker:at '#' new-url 'Fix it & more'
  [ "$output" = "https://github.com/empowerite/example/issues/new?title=Fix%20it%20%26%20more" ]
  run in_tracker tracker:at '#' new-url
  [ "$output" = "https://github.com/empowerite/example/issues/new" ]
  run in_tracker tracker:at 'AB#' new-url 'Fix it'
  [ "$output" = "https://dev.azure.com/example/Claims%20Team/_workitems/create/Task?%5BSystem.Title%5D=Fix%20it" ]
  run in_tracker tracker:at 'LIN#' new-url 'Fix it'
  [ "$output" = "https://linear.app/example/team/ENG/new?title=Fix%20it" ]
  run in_tracker tracker:of 'LIN#12' url
  [ "$output" = "https://linear.app/example/issue/ENG-12" ]
  # a web tracker does nothing but open its pages: the rest refuses, naming the page to do it on, or the setting
  run in_tracker tracker:of 'LIN#12' title
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit cannot read its title in the LIN# tracker, which it reaches through its pages alone; open https://linear.app/example/issue/ENG-12"* ]]
  run in_tracker tracker:at 'LIN#' create 'Fix it'
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: kit new issue LIN#, which opens its page"* ]]
  run in_tracker tracker:at 'BZ#' new-url
  [ "$status" -eq 1 ]
  [[ "$output" == *"the BZ# tracker has no new page; set new in its record under trackers in kit.yml"* ]]
  run in_tracker tracker:of 'BZ#3' assign ada
  [[ "$output" == *"kit cannot assign it in the BZ# tracker"*"open its tracker's page"* ]]
  # on GitLab, the page is under the project's /-/, the title as issue[title]
  git remote set-url origin git@gitlab.com:empowerite/group/example.git
  run in_tracker tracker:at '#' new-url 'Fix it'
  [ "$output" = "https://gitlab.com/empowerite/group/example/-/issues/new?issue%5Btitle%5D=Fix%20it" ]
}

@test "kit new issue opens the tracker's new-issue page, and lists every tracker with its page, the forge's own first" {
  fake-tools gh az open xdg-open
  fixture repo.json <<<'{"url": "https://github.com/empowerite/example"}'
  fixture none.txt < /dev/null
  route gh 'repo view*' repo.json
  route open '*' none.txt
  route xdg-open '*' none.txt
  : > "${FAKE_DIR}/open.calls"
  : > "${FAKE_DIR}/xdg-open.calls"
  printf '  - prefix: "BZ#"\n    provider: web\n' >> kit.yml
  run "${kit_root}/bin/kit" new issue 'AB#'
  [ "$status" -eq 0 ]
  [ "$output" = "https://dev.azure.com/example/Claims%20Team/_workitems/create/Task" ]
  [ "$(cat "${FAKE_DIR}/open.calls" "${FAKE_DIR}/xdg-open.calls")" = "${output}" ]
  # --print opens nothing; the forge's own is the default
  : > "${FAKE_DIR}/open.calls"
  : > "${FAKE_DIR}/xdg-open.calls"
  run "${kit_root}/bin/kit" new issue --print
  [ "$output" = "https://github.com/empowerite/example/issues/new" ]
  [ -z "$(cat "${FAKE_DIR}/open.calls" "${FAKE_DIR}/xdg-open.calls")" ]
  run "${kit_root}/bin/kit" new issue 'XY#' --print
  [ "$status" -eq 2 ]
  [[ "$output" == *"no tracker answers 'XY#'"* ]]
  # the list: the forge's own first, each with its label and page, or the refusal where it has none
  run "${kit_root}/bin/kit" new issue --list --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.[] | [.prefix, .provider, .label, .url != null]]' <<<"${output}")" = \
    '[["#","forge","github empowerite/example",true],["AB#","ado","ado Claims Team",true],["BZ#","web","web",false]]' ]
  [[ "$(jq --raw-output '.[2].error' <<<"${output}")" == "the BZ# tracker has no new page; set new in its record under trackers in kit.yml" ]]
  run "${kit_root}/bin/kit" new issue --list
  [ "${lines[0]}" = "$(printf '#\tgithub empowerite/example\thttps://github.com/empowerite/example/issues/new')" ]
}

@test "a tracker record without its organization is refused with the setting to add" {
  printf 'trackers:\n  - prefix: "AB#"\n    provider: ado\n' > kit.yml
  run in_tracker tracker:of 'AB#4521' title
  [ "$status" -eq 1 ]
  [[ "$output" == *"add organization: <https://dev.azure.com/name> to the AB# tracker in kit.yml"* ]]
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

@test "kit settle closes a merged change's work items with the merge commit, and puts a closed change's issues back, once" {
  fixture repo.json <<<'{"nameWithOwner": "empowerite/example", "url": "https://github.com/empowerite/example"}'
  fixture merged.json <<<'{"state": "MERGED", "mergeCommit": {"oid": "abcdef0123456789"}}'
  fixture closed.json <<<'{"state": "CLOSED", "mergeCommit": null}'
  fixture open.json <<<'{"state": "OPEN", "mergeCommit": null}'
  fixture body.json <<<'{"body": "Does the work.\nCloses AB#4521"}'
  fixture active.json <<'JSON'
{"id": 4521, "fields": {"System.Title": "Fix it", "System.State": "Active",
 "System.AssignedTo": {"uniqueName": "ada@example.com"}}}
JSON
  fixture done.json <<<'{"id": 4521, "fields": {"System.Title": "Fix it", "System.State": "Closed"}}'
  fixture author.json <<<'{"user": {"login": "ada"}}'
  fixture closing.json <<'JSON'
{"data": {"repository": {"pullRequest": {"closingIssuesReferences": {"nodes": [
  {"number": 7, "state": "OPEN", "repository": {"nameWithOwner": "empowerite/example"}},
  {"number": 8, "state": "OPEN", "repository": {"nameWithOwner": "someone/else"}}]}}}}}
JSON
  fixture issue.json <<<'{"title": "Seven", "state": "OPEN", "assignees": [{"login": "ada"}]}'
  fixture me.txt <<<'ada@example.com'
  fixture user.json <<<'{"login": "ada"}'
  route gh 'api user*' user.json
  route gh 'repo view*' repo.json
  route gh 'pr view 3 --json state,mergeCommit*' merged.json
  route gh 'pr view 4 --json state,mergeCommit*' closed.json
  route gh 'pr view 6 --json state,mergeCommit*' open.json
  route gh 'pr view * --json body*' body.json
  route gh 'api repos/empowerite/example/pulls/4 *' author.json
  route gh 'api graphql*number=4*' closing.json
  route gh 'issue view 7 *' issue.json
  route gh 'issue comment 7 *' '!0'
  route gh 'issue edit 7 --remove-assignee ada' '!0'
  route az 'boards work-item show --id 4521 *' active.json
  route az 'boards work-item update --id 4521 *' '!0'
  route az 'account show*' me.txt
  printf 'forge: github\n' >> kit.yml
  run "${kit_root}/bin/kit" settle 3
  [ "$status" -eq 0 ]
  [[ "$output" == *"AB#4521 closed: pull request #3 merged"* ]]
  grep -q 'update --id 4521 --discussion Closed by pull request #3, merged as abcdef012345: https://github.com/empowerite/example/pull/3' "${FAKE_DIR}/az.calls"
  grep -q 'update --id 4521 --state Closed' "${FAKE_DIR}/az.calls"
  run "${kit_root}/bin/kit" settle 4
  [ "$status" -eq 0 ]
  [[ "$output" == *"#7 is backlog again: pull request #4 closed unmerged, ada unassigned"* ]]
  [[ "$output" == *"AB#4521 is backlog again"* ]]
  [[ "$output" != *"#8"* ]]
  grep -q '^issue edit 7 --remove-assignee ada$' "${FAKE_DIR}/gh.calls"
  run "${kit_root}/bin/kit" settle 6
  [ "$status" -eq 0 ]
  [[ "$output" == *"pull request #6 is open; nothing to settle"* ]]
  # again, with the work item closed: nothing more is done to it
  route az 'boards work-item show --id 4521 *' done.json
  sed -i.bak '/show --id 4521 \*\tactive.json/d' "${FAKE_DIR}/az.routes"
  : > "${FAKE_DIR}/az.calls"
  run "${kit_root}/bin/kit" settle 3
  [ "$status" -eq 0 ]
  run grep -c update "${FAKE_DIR}/az.calls"
  [ "$output" = 0 ]
}

@test "the settings take a tracker with a prefix and a provider, and refuse one without" {
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  run "${kit_root}/bin/kit" check
  [ "$status" -eq 0 ]
  printf 'trackers:\n  - prefix: "A B#"\n    provider: ado\n' >> kit.yml
  run "${kit_root}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *"a tracker needs a prefix, letters and an optional #, and a provider (prefix 'A B#')"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
