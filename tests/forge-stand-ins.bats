#!/usr/bin/env bats
# forge-stand-ins.bats - a stand-in the repository chooses answers a forge feature in the forge's place
# ----------------------------------------------------------------------------------------------------------------------
# The forge is a provider the repository adds on its module path, whose calls each log their arguments, so what is
# proven is the seam's own answer from a stand-in: which of the provider's calls it makes, and what it prints. The last
# tests drive GitHub's, GitLab's, and Azure DevOps' providers through fakes of their tools, for the calls version 3 of
# the contract adds.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh glab az
  fresh-repo
  git remote add origin https://code.example/team/example.git
  mkdir -p .kit bash/forge
}

# call <call> <argument>… runs one call of the seam, as a verb would.
call() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && "forge:$0" "$@"' "$@"
}

# provider <contract> <lacks>… writes the forge plain under bash/forge/, implementing <contract>, its forge lacking
# <lacks>, and kit.yml naming it, with the stand-ins in ${STAND_INS} (YAML, indented under stand_ins) when set. Every
# call logs its name and arguments to calls.log.
provider() {
  local contract=$1 call
  shift
  {
    printf 'forge: plain\nbash_path:\n  - kit:modules/bash/\n  - bash/\n'
    [[ -z "${STAND_INS:-}" ]] || printf 'forge_settings:\n  plain:\n    stand_ins:\n%s\n' "${STAND_INS}"
  } > kit.yml
  {
    printf 'forge/plain:contract() { printf "%%s\\n" %s; }\n' "${contract}"
    (( $# == 0 )) || printf 'forge/plain:lacks() { printf "%%s\\n" %s; }\n' "$*"
    printf 'log() { printf "%%s\\n" "$*" >> "%s/calls.log"; }\n' "${repo}"
    printf 'forge/plain:require() { :; }\n'
    printf 'forge/plain:noun() { printf "#\\tpull request\\n"; }\n'
    printf 'forge/plain:repo() { echo team/example; }\n'
    printf 'forge/plain:pr-here() { echo 5; }\n'
    printf 'forge/plain:pr-url() { echo "https://code.example/team/example/pull/$1"; }\n'
    printf 'forge/plain:pr-create() { log pr-create "$@"; echo 5; }\n'
    printf 'forge/plain:pr-title() { log pr-title "$@"; echo "[draft] Fix it"; }\n'
    printf 'forge/plain:pr-body() { log pr-body "$@"; printf "Closes #7, and fixes #9.\\nSee #8.\\n"; }\n'
    printf 'forge/plain:issue-state() { log issue-state "$@"; [[ $1 == 9 ]] && echo CLOSED || echo OPEN; }\n'
    for call in pr-ready pr-set-title pr-label-add pr-label-remove pr-closing-issues marks
    do
      printf 'forge/plain:%s() { log %s "$@"; }\n' "${call}" "${call}"
    done
    printf 'forge/plain:open-prs() { log open-prs; printf "%%s\\n" %s %s; }\n' \
      "'{\"number\":5,\"title\":\"[draft] Fix it\",\"state\":\"ready\",\"labels\":[{\"name\":\"draft\",\"color\":null}]}'" \
      "'{\"number\":6,\"title\":\"Other\",\"state\":\"ready\",\"labels\":[]}'"
  } > bash/forge/plain.sh
}

@test "with no stand-in, a change opens as the forge's own draft, and kit ready ends it on the forge" {
  provider 3
  run call pr-create 5-fix main "Fix it"
  [ "$output" = 5 ]
  run "${kit_root}/bin/kit" ready
  [ "$status" -eq 0 ]
  [[ "$output" == *"pull request #5 is ready for review"* ]]
  [ "$(cat calls.log)" = $'pr-create 5-fix main Fix it\npr-ready 5' ]
}

@test "a label stands in for a draft: the change opens ready and labelled, draws as a draft, and kit ready unlabels it" {
  STAND_INS='      drafts: {label: draft}' provider 3
  run call pr-create 5-fix main "Fix it"
  [ "$output" = 5 ]
  [ "$(cat calls.log)" = $'pr-create 5-fix main Fix it false\npr-label-add 5 draft' ]
  run call open-prs
  [ "$(jq --slurp --compact-output 'map([.number, .state])' <<<"$output")" = '[[5,"draft"],[6,"ready"]]' ]
  : > calls.log
  run "${kit_root}/bin/kit" ready 5
  [ "$status" -eq 0 ]
  [ "$(cat calls.log)" = 'pr-label-remove 5 draft' ]
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$(jq -c '[.features.drafts, .stand_ins]' <<<"$output")" = '["stand-in",{"drafts":{"label":"draft"}}]' ]
}

@test "a title prefix stands in for a draft: kit judges the title without it, and kit ready takes it off" {
  STAND_INS='      drafts: {title_prefix: "[draft] "}' provider 3
  run call pr-create 5-fix main "Fix it"
  [ "$(cat calls.log)" = 'pr-create 5-fix main [draft] Fix it false' ]
  run call pr-title 5
  [ "$output" = "Fix it" ]
  run call open-prs
  [ "$(jq --slurp --compact-output 'map([.number, .state, .title])' <<<"$output")" \
    = '[[5,"draft","Fix it"],[6,"ready","Other"]]' ]
  : > calls.log
  run call pr-ready 5
  [ "$status" -eq 0 ]
  [ "$(cat calls.log)" = $'pr-title 5\npr-set-title 5 Fix it' ]
}

@test "a forge without drafts and no stand-in opens a change ready, and kit ready says it is already" {
  provider 3 drafts
  run "${kit_root}/bin/kit" ready 5
  [ "$status" -eq 0 ]
  [[ "$output" == *"pull request #5 is open for review already: the plain forge has no drafts, and kit.yml chooses no stand-in"* ]]
  [ ! -e calls.log ]
}

@test "the body stands in for the issues a change closes: each one a closing keyword names, with its state" {
  STAND_INS='      closing-issues: body' provider 3 closing-issues
  run call pr-closing-issues 5
  [ "$status" -eq 0 ]
  [ "$output" = $'7 OPEN team/example\n9 CLOSED team/example' ]
  [ "$(grep -c pr-closing-issues calls.log)" -eq 0 ]
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$(jq -r '.features["closing-issues"]' <<<"$output")" = stand-in ]
}

@test "a timer stands in for marks: every stage of the forge is stale once its span is past" {
  STAND_INS='      marks: {every: 60}' provider 3 marks
  printf 'flow: [releases, builds]\n' >> kit.yml
  run --separate-stderr "${kit_root}/bin/kit" flow --marks
  [ "$status" -eq 0 ]
  span=$(( $(date +%s) / 60 ))
  [[ "$output" == "{\"releases\":\"${span}\",\"builds\":\"${span}\"}" || \
     "$output" == "{\"releases\":\"$(( span + 1 ))\",\"builds\":\"$(( span + 1 ))\"}" ]]
  [ ! -e calls.log ]
}

@test "a stand-in the forge cannot carry, or that kit does not have, is refused naming the setting" {
  STAND_INS='      drafts: {label: draft}' provider 3 labels
  run call pr-create 5-fix main "Fix it"
  [ "$status" -eq 1 ]
  [[ "$output" == *"forge_settings.plain.stand_ins in kit.yml: drafts: a label, which the forge lacks"* ]]
  STAND_INS='      drafts: {title_prefix: "[draft] "}' provider 2
  run call pr-create 5-fix main "Fix it"
  [[ "$output" == *"drafts: a title prefix, which needs a provider of version 3"* ]]
  STAND_INS='      checks: {codebuild: [ci]}' provider 3
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$status" -eq 1 ]
  [[ "$output" == *"checks: no stand-in: the stand-ins are drafts, closing-issues, and marks"* ]]
  STAND_INS='      marks: {every: 1}' provider 3
  run call marks
  [[ "$output" == *"marks: {every: <seconds>}, 10 at the least, not {\"every\":1}"* ]]
  [ ! -e calls.log ]
}

@test "a call version 3 added is unsupported for a provider of version 2, and runs nothing" {
  provider 2
  for c in 'pr-ready 5' 'pr-set-title 5 x' 'pr-label-add 5 x' 'pr-label-remove 5 x'
  do
    read -r -a words <<<"${c}" && run --separate-stderr call "${words[@]}"
    [ "$status" -eq 3 ] || { echo "${c}: ${status}"; return 1; }
  done
  [ ! -e calls.log ]
  run "${kit_root}/bin/kit" flow --capabilities
  [ "$(jq -c '[.provider, .features.labels]' <<<"$output")" = '[2,"none"]' ]
}

@test "GitHub, GitLab, and Azure DevOps open a change ready, end a draft, retitle, and label it in their tools' words" {
  git remote set-url origin https://github.com/empowerite/example.git
  route gh 'auth status*' '!0'
  printf 'https://github.com/empowerite/example/pull/3\n' | fixture pr-url.txt
  route gh 'pr create --base main --head 7-fix --title Fix it --body ' pr-url.txt
  route gh 'pr *' '!0'
  call pr-create 7-fix main "Fix it" false
  call pr-ready 3
  call pr-set-title 3 "New"
  call pr-label-add 3 draft
  call pr-label-remove 3 draft
  [ "$(grep '^pr ' "${FAKE_DIR}/gh.calls")" = "$(printf '%s\n' 'pr create --base main --head 7-fix --title Fix it --body ' \
    'pr ready 3' 'pr edit 3 --title New' 'pr edit 3 --add-label draft' 'pr edit 3 --remove-label draft')" ]
  git remote set-url origin git@gitlab.com:empowerite/example.git
  printf 'https://gitlab.com/empowerite/example/-/merge_requests/3\n' | fixture mr-url.txt
  route glab 'mr create --source-branch 7-fix --target-branch main --title Fix it --description  --yes' mr-url.txt
  route glab '*' '!0'
  call pr-create 7-fix main "Fix it" false
  call pr-ready 3
  call pr-set-title 3 "New"
  call pr-label-add 3 draft
  call pr-label-remove 3 draft
  [ "$(cat "${FAKE_DIR}/glab.calls")" = "$(printf '%s\n' \
    'mr create --source-branch 7-fix --target-branch main --title Fix it --description  --yes' 'mr update 3 --ready' \
    'api projects/empowerite%2Fexample/merge_requests/3 --method PUT --raw-field title=New' \
    'mr update 3 --label draft' 'mr update 3 --unlabel draft')" ]
  git remote set-url origin https://dev.azure.com/acme/empowerite/_git/example
  route az '*' '!0'
  call pr-ready 3
  call pr-set-title 3 "New"
  call pr-label-add 3 draft
  call pr-label-remove 3 draft
  run sed 's/ --org https:\/\/dev.azure.com\/acme --detect false --only-show-errors//; s/--in-file [^ ]*/--in-file <file>/' \
    "${FAKE_DIR}/az.calls"
  [ "$output" = "$(printf '%s\n' 'repos pr update --id 3 --draft false --output none' \
    'repos pr update --id 3 --title New --output none' \
    'devops invoke --area git --resource pullRequestLabels --http-method POST --api-version 7.1 --route-parameters project=empowerite repositoryId=example pullRequestId=3 --in-file <file> --output none' \
    'devops invoke --area git --resource pullRequestLabels --http-method DELETE --api-version 7.1 --route-parameters project=empowerite repositoryId=example pullRequestId=3 labelIdOrName=draft --output none')" ]
}

# ----------------------------------------------------------------------------------------------------------------------
