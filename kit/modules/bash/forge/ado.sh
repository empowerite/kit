# shellcheck shell=bash
# ado.sh - the Azure DevOps forge provider: Azure Repos and Azure Boards, through the az CLI (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh, forge_list_limit set by modules/bash/forge.sh
# shellcheck disable=SC2016  # jq filters name their variables with $, in single quotes on purpose
#
# modules/bash/forge.sh loads this provider when the repository's forge is ado:
#
#   - Azure DevOps Services, detected from the primary remote's host: dev.azure.com, ssh.dev.azure.com, or a host
#     under visualstudio.com
#   - Azure DevOps Server, self-hosted: `forge: ado` in kit.yml
#
# The primary remote's URL names the organization, the project, and the repository, so nothing else need say them:
#
#   https://dev.azure.com/<organization>/<project>/_git/<repository>
#   git@ssh.dev.azure.com:v3/<organization>/<project>/<repository>
#   https://<organization>.visualstudio.com/<project>/_git/<repository>
#   https://<server>/<collection>/<project>/_git/<repository>
#
# It implements the calls listed in the header of modules/bash/forge.sh, with the same output as the GitHub
# provider. The Azure DevOps equivalents:
#
#   pull request     a pull request, !<id> in Azure DevOps' own references
#   issue            a work item of the project's Azure Boards, #<id> and AB#<id> alike
#   checks           the pull request's build and status policies
#   review threads   its threads that a person opened, active or pending until resolved
#   closing issues   the work items linked to it, which its completion can close
#   sub-issues       a work item's parent relation
#   runs             Azure Pipelines' runs
#
# Its forge lacks rate limits it shows before it delays a request, marks it reads cheaply, releases, and branches
# it links to a work item: a branch for a work item is the one kit names and pushes, and the pull request kit opens on
# it links the work item.
#
# A work item opens as the type forge_settings.ado.type names (default: Task), closes completed to
# forge_settings.ado.closed_state (default: Closed), and otherwise to forge_settings.ado.removed_state (default:
# Removed). The caller (me) is the identity az is signed in as.
#
# Only this file requires az, with its azure-devops extension. Every call names the organization, the project, and
# the repository, and never lets az detect them from git, and shapes every result with jq, so a fake az can drive it
# in a test.
#
# see: modules/bash/forge.sh, modules/bash/tracker/ado.sh

# forge/ado:artifacts prints one document per artifact of the project's 20 newest pipeline runs, newest first, as the
# seam's header says: a read of 21 runs, then a read of each of the 20 newest's artifacts, until 100 are drawn. It
# prints {"limited": N} after the N drawn when it stopped at 100, or when a 21st run says there are more.
forge/ado:artifacts() {
  local org project runs id run artifacts results docs='' doc drawn=0
  local -a ids
  IFS=$'\t' read -r org project _ < <(forge/ado:place) || return 1
  results="${org}/$(forge/ado:encode "${project}")/_build/results"
  forge/ado:ask runs 'the artifacts' pipelines runs list --project "${project}" --top 21 --query-order QueueTimeDesc \
    || return 1
  list:read ids forge/ado:fold 'the artifacts' "${runs}" --raw-output '.[:20][].id' || return 1
  for id in "${ids[@]}"
  do
    [[ -n "${id}" ]] || continue
    forge/ado:ask artifacts "the artifacts of run ${id}" pipelines runs artifact list --run-id "${id}" \
      --project "${project}" || return 1
    run=$(jq --compact-output --argjson id "${id}" 'first(.[] | select(.id == $id))' <<<"${runs}")
    doc=$(forge/ado:fold "the artifacts of run ${id}" "${artifacts}" --compact-output --argjson now "$(date +%s)" \
      --argjson run "${run}" --arg results "${results}" \
      'import "forge" as forge; forge::ado_artifacts($now; $run; $results)') || return 1
    [[ -z "${doc}" ]] || docs+=${doc}$'\n'
    drawn=$(jq --slurp length <<<"${docs}")
    (( drawn < 100 )) || break
  done
  [[ -z "${docs}" ]] || sed -n '1,100p' <<<"${docs%$'\n'}"
  jq --compact-output --argjson drawn "${drawn}" \
    'if $drawn >= 100 then {limited: 100} elif length > 20 then {limited: $drawn} else empty end' <<<"${runs}"
}

forge/ado:attestations() { return 3; }

forge/ado:bot-app() { return 3; }
forge/ado:bot-installation() { return 3; }
forge/ado:bot-installations() { return 3; }
forge/ado:bot-proof() { return 3; }
forge/ado:bot-repositories() { return 3; }
forge/ado:bot-revoke() { return 3; }
forge/ado:bot-token() { return 3; }

forge/ado:bots() { return 3; }

forge/ado:branch-for-issue() {
  local open
  forge/ado:prs open 'the open pull requests' active || return 1
  forge/ado:fold 'the open pull requests' "${open}" --raw-output --arg n "$1" \
    '[.[] | (.sourceRefName | ltrimstr("refs/heads/")) | select(startswith("\($n)-"))] | .[0] // empty'
}

# The provider implements contract 7, and its forge lacks these features: a bot account among them, as kit makes one
# of a GitHub App alone.
forge/ado:contract() { printf '7\n'; }

forge/ado:lacks() {
  printf '%s\n' quota marks releases linked-branches milestones bots attestations code-scanning dependabot \
    secret-scanning bypasses swimlanes bot-accounts
}

forge/ado:bypasses() { return 3; }

forge/ado:code-scanning-alerts() { return 3; }

# forge/ado:collaborators prints one document per member of the project's teams, as the seam's header says: the
# teams, as forge/ado:teams-read reads them, each member once, an administrator of any of them admin. Azure DevOps
# lists its users only for the whole organization, with no project to filter them by, so the project's teams are
# who has access here; one who has it only through a security group the teams do not hold is not among them.
forge/ado:collaborators() {
  local teams members
  forge/ado:teams-read teams members || return 1
  forge/ado:fold 'the teams' "${teams}" --compact-output --argjson members "${members}" \
    'import "forge" as forge; forge::ado_collaborators($members; 20)'
}

forge/ado:dependabot-alerts() { return 3; }

# forge/ado:deployments prints one document per deployment, newest first, as the seam's header says: Azure DevOps keeps
# them as each environment's deployment records, so it reads the environments, then the newest 51 records of each of
# the 20 changed last, as Azure DevOps lists them, which its reference leaves unsaid and is taken to be newest first,
# and the project's 100 newest runs, which name each record's branch, commit, and who asked for it. {"limited": N}
# follows the N drawn when more than 20 environments are there, an environment gives a 51st record, or the records
# read are more than 50.
forge/ado:deployments() {
  local org project repo envs id records all='[]' runs cut=false base
  local -a ids
  IFS=$'\t' read -r org project repo < <(forge/ado:place) || return 1
  base="${org}/$(forge/ado:encode "${project}")"
  forge/ado:environments-list envs "${project}" || return 1
  list:read ids forge/ado:fold 'the deployments' "${envs}" --raw-output \
    'import "forge" as forge; .value | sort_by(.lastModifiedOn | if . == null then 0 else forge::epoch end) | reverse
     | .[:20][].id' || return 1
  [[ "$(jq '.value | length' <<<"${envs}")" -le 20 ]] || cut=true
  for id in "${ids[@]}"
  do
    [[ -n "${id}" ]] || continue
    forge/ado:ask records "the deployments of environment ${id}" devops invoke --area distributedtask \
      --resource environmentdeploymentrecords --route-parameters "project=${project}" "environmentId=${id}" \
      --query-parameters top=51 --api-version 7.1 || return 1
    all=$(forge/ado:fold "the deployments of environment ${id}" "${all}" --compact-output \
      --argjson records "${records}" '. + ($records.value | arrays // error)') || return 1
    [[ "$(jq '.value | length' <<<"${records}")" -le 50 ]] || cut=true
  done
  forge/ado:ask runs 'the deployments' pipelines runs list --project "${project}" --top 100 \
    --query-order QueueTimeDesc || return 1
  forge/ado:fold 'the deployments' "${all}" --compact-output --argjson now "$(date +%s)" --argjson envs "${envs}" \
    --argjson runs "${runs}" --argjson cut "${cut}" --arg results "${base}/_build/results" \
    --argjson repo "$(jq --null-input --compact-output --arg name "${repo}" --arg url "$(forge/ado:repo-url)" \
      '{name: $name, url: $url}')" \
    'import "forge" as forge; forge::ado_deployments($now; $envs; $runs; 50; $cut; $results; $repo)'
}

# forge/ado:environments prints one document per environment of the project, in Azure DevOps' order, as the seam's
# header says: a read of the environments, then one of the checks on them all, which hold their approvals and branch
# controls.
forge/ado:environments() {
  local org project envs checks='{"value": []}' dir
  IFS=$'\t' read -r org project _ < <(forge/ado:place) || return 1
  forge/ado:environments-list envs "${project}" || return 1
  if [[ "$(jq '.value | length' <<<"${envs}")" -gt 0 ]]
  then
    file:scratch dir || return 1
    # shellcheck disable=SC2154  # dir is set by file:scratch
    jq --compact-output '[.value[] | {type: "environment", id: (.id | tostring), name}]' <<<"${envs}" \
      > "${dir}/resources.json" || return 1
    forge/ado:ask checks 'the checks of the environments' devops invoke --area PipelinesChecks \
      --resource queryconfigurations --route-parameters "project=${project}" --query-parameters '$expand=settings' \
      --http-method POST --in-file "${dir}/resources.json" --api-version 7.1-preview.1 || return 1
  fi
  forge/ado:fold 'the environments' "${envs}" --compact-output --argjson checks "${checks}" \
    --arg url "${org}/$(forge/ado:encode "${project}")" \
    'import "forge" as forge; forge::ado_environments($checks; $url)'
}

forge/ado:issue-add-sub() {
  forge/ado:az boards work-item relation add --id "$2" --relation-type parent --target-id "$1" --output none \
    >/dev/null 2>&1
}

forge/ado:issue-assign() {
  forge/ado:az boards work-item update --id "$1" --assigned-to "$2" --output none >/dev/null 2>&1
}

forge/ado:issue-assignees() {
  local item
  forge/ado:ask item "the assignees of #$1" boards work-item show --id "$1" || return 1
  forge/ado:fold "the assignees of #$1" "${item}" --raw-output 'import "forge" as forge; forge::ado_assignees | join(" ")'
}

# forge/ado:issue-close sets the work item's closing state. Azure Boards keeps no reason, so a work item not planned
# or a duplicate is set to the removed state, and a duplicate also gets a comment naming the original.
# forge/ado:issue-body prints the work item's description, which Azure Boards keeps as HTML, as the text a Markdown
# reader would take from it: a heading as ## and its text, a list item as - and its text, a paragraph or a line break as
# a new line, other tags dropped, and the entities read. An empty description prints nothing.
forge/ado:issue-body() {
  local item
  forge/ado:ask item "the body of #$1" boards work-item show --id "$1" || return 1
  forge/ado:fold "the body of #$1" "${item}" --raw-output 'import "forge" as forge;
    .fields["System.Description"] // "" | strings // error | forge::ado_text | select(. != "")'
}

forge/ado:issue-close() {
  local state
  if [[ "$2" == completed ]]
  then
    state=$(forge:setting ado closed_state)
    state=${state:-Closed}
  else
    state=$(forge:setting ado removed_state)
    state=${state:-Removed}
  fi
  if [[ "$2" == duplicate ]]
  then
    forge/ado:issue-comment "$1" "Duplicate of #$3" || return 1
  fi
  forge/ado:az boards work-item update --id "$1" --state "${state}" --output none >/dev/null 2>&1
}

forge/ado:issue-comment() {
  forge/ado:az boards work-item update --id "$1" --discussion "$2" --output none >/dev/null 2>&1
}

forge/ado:issue-create() {
  local type item project
  type=$(forge:setting ado type)
  IFS=$'\t' read -r _ project _ < <(forge/ado:place) || return 1
  if ! item=$(forge/ado:az boards work-item create --title "$1" --type "${type:-Task}" --description "${2:-}" \
    --project "${project}" --output json 2>/dev/null) || ! forge:json "${item}"
  then
    error "run: az login (the work item could not be created)"
    return 1
  fi
  forge/ado:fold 'the work item it created' "${item}" --raw-output '.id | numbers // error'
}

forge/ado:issue-state() {
  local item
  forge/ado:ask item "the state of #$1" boards work-item show --id "$1" || return 1
  forge/ado:fold "the state of #$1" "${item}" --raw-output \
    'import "forge" as forge; if (.fields["System.State"] | strings // error) and forge::ado_closed then "CLOSED"
     else "OPEN" end'
}

forge/ado:issue-title() {
  local item
  forge/ado:ask item "the title of #$1" boards work-item show --id "$1" || return 1
  forge/ado:fold "the title of #$1" "${item}" --raw-output '.fields["System.Title"] | strings // error'
}

# forge/ado:issue-unassign clears the work item's one assignee, when it is the login named.
forge/ado:issue-unassign() {
  local assignees
  assignees=$(forge/ado:issue-assignees "$1") || return 1
  [[ "${assignees}" == "$2" ]] || return 0
  forge/ado:az boards work-item update --id "$1" --assigned-to "" --output none >/dev/null 2>&1
}

# forge/ado:iterations prints one document per iteration of a team's, by start, as the seam's header says: the team
# forge_settings.ado.team names, else the project's default team, which az takes when it is not named.
forge/ado:iterations() {
  local org project team list
  local -a named=()
  IFS=$'\t' read -r org project _ < <(forge/ado:place) || return 1
  team=$(forge:setting ado team)
  [[ -z "${team}" ]] || named=(--team "${team}")
  forge/ado:ask list 'the iterations' boards iteration team list "${named[@]}" --project "${project}" || return 1
  forge/ado:fold 'the iterations' "${list}" --compact-output \
    --arg url "${org}/$(forge/ado:encode "${project}")/_sprints" 'import "forge" as forge; forge::ado_iterations($url)'
}

# The forge lacks branches linked to a work item, as forge/ado:lacks declares, so the seam answers unsupported and never calls this.
forge/ado:link-branch() { return 3; }

# The forge lacks marks it reads cheaply, as forge/ado:lacks declares, so the seam answers unsupported and never calls this.
forge/ado:marks() { return 3; }

forge/ado:me() {
  local login
  if login=$(az account show --query user.name --output tsv 2>/dev/null) && [[ -n "${login}" ]]
  then
    printf '%s\n' "${login}"
    return 0
  fi
  error "run: az login (the caller's login could not be read)"
  return 1
}

# forge/ado:merged-prs prints one document per completed pull request, as the seam's header says: one read of 51, in
# the order Azure DevOps lists them, which its reference's examples show newest first and no parameter sets, so a
# 51st says it holds more than the 50 printed, which end with {"limited": 50}. A pull request completes when it
# merges, and its page is under the repository's.
forge/ado:merged-prs() {
  local list project repo url
  IFS=$'\t' read -r _ project repo < <(forge/ado:place) || return 1
  url=$(forge/ado:repo-url) || return 1
  forge/ado:ask list 'the completed pull requests' repos pr list --project "${project}" --repository "${repo}" \
    --status completed --top 51 || return 1
  forge/ado:fold 'the completed pull requests' "${list}" --exit-status 'arrays' >/dev/null || return 1
  forge/ado:fold 'the completed pull requests' "${list}" --compact-output --arg url "${url}" \
    'import "forge" as forge; forge::ado_merged(50; $url)'
}

# The forge lacks milestones, as forge/ado:lacks declares, so the seam answers unsupported and never calls this.
forge/ado:milestones() { return 3; }

forge/ado:noun() { printf '!\tpull request\n'; }

# forge/ado:open-issues prints one document per open work item of the project, oldest first, as the seam's header
# says: the work items an Azure Boards query finds in no closing state.
forge/ado:open-issues() {
  local items project repo
  IFS=$'\t' read -r _ project repo < <(forge/ado:place) || return 1
  forge/ado:ask items 'the work items' boards query --project "${project}" --wiql \
    "SELECT [System.Id], [System.Title], [System.AssignedTo], [System.Tags], [System.CreatedDate],
            [System.IterationPath] FROM WorkItems
     WHERE [System.TeamProject] = @project AND [System.State] NOT IN ('Closed', 'Done', 'Removed')
     ORDER BY [System.CreatedDate] ASC" || return 1
  forge/ado:fold 'the work items' "${items}" --exit-status 'type == "array"' >/dev/null || return 1
  forge:whole 'the work items' "$(jq length <<<"${items}")" || return 1
  forge/ado:fold 'the work items' "${items}" --compact-output --argjson now "$(date +%s)" \
    --arg path "${project}/${repo}" 'import "forge" as forge; forge::ado_issues($now; $path)'
}

# forge/ado:open-prs prints one JSON object per open pull request: the document kit flow renders, which
# modules/jq/forge.jq folds the answers into. The list names each pull request's reviewers and their votes. Azure
# DevOps needs three reads per pull request beside the list: its policy evaluations, its threads, and the work items
# linked to it. Any read that fails fails the call, which then prints none of the pull requests.
forge/ado:open-prs() {
  local list project repo id policies threads closes doc docs=''
  local -a ids
  IFS=$'\t' read -r _ project repo < <(forge/ado:place) || return 1
  forge/ado:prs list 'the pull requests' active || return 1
  list:read ids forge/ado:fold 'the pull requests' "${list}" --raw-output 'sort_by(.pullRequestId) | .[].pullRequestId' \
    || return 1
  for id in "${ids[@]}"
  do
    [[ -n "${id}" ]] || continue
    forge/ado:ask policies "the policies of pull request !${id}" repos pr policy list --id "${id}" \
      --top "${forge_list_limit}" || return 1
    forge/ado:ask threads "the threads of pull request !${id}" devops invoke --area git --resource pullRequestThreads \
      --route-parameters "project=${project}" "repositoryId=${repo}" "pullRequestId=${id}" --api-version 7.1 \
      || return 1
    forge/ado:ask closes "the work items pull request !${id} links" repos pr work-item list --id "${id}" || return 1
    doc=$(forge/ado:fold "pull request !${id}" "${list}" --compact-output --argjson id "${id}" \
      --argjson now "$(date +%s)" --argjson policies "${policies}" --argjson threads "${threads}" \
      --argjson closes "${closes:-[]}" --arg path "${project}/${repo}" \
      'import "forge" as forge; .[] | select(.pullRequestId == $id)
       | forge::ado_pr($now; $policies; $threads; $closes; $path)') || return 1
    docs+=${doc}$'\n'
  done
  printf '%s' "${docs}"
}

forge/ado:pr-author() {
  local pr
  forge/ado:ask pr "the author of pull request !$1" repos pr show --id "$1" || return 1
  forge/ado:fold "the author of pull request !$1" "${pr}" --raw-output '.createdBy.uniqueName | strings // error'
}

forge/ado:pr-body() {
  local pr body
  forge/ado:ask pr "the body of pull request !$1" repos pr show --id "$1" || return 1
  body=$(forge/ado:fold "the body of pull request !$1" "${pr}" --raw-output '.description // "" | strings // error') \
    || return 1
  printf '%s\n' "${body}"
}

# forge/ado:pr-closing-issues prints the work items linked to the pull request, which its completion can close. They
# are the project's, so each is named the repository's, as kit linked asks.
forge/ado:pr-closing-issues() {
  local items repo
  forge/ado:ask items "the work items pull request !$1 links" repos pr work-item list --id "$1" || return 1
  repo=$(forge/ado:repo) || return 1
  forge/ado:fold "the work items pull request !$1 links" "${items:-[]}" --raw-output --arg repo "${repo}" \
    'import "forge" as forge; .[] | "\(.id) \(if forge::ado_closed then "CLOSED" else "OPEN" end) \($repo)"'
}

# forge/ado:pr-create opens the draft, or the change ready for review when the draft is false, and links the work
# item the branch is named for, <id>-<slug>, so the pull request closes it as the forge's own tracker closes an issue.
# az opens no pull request from a fork's branch, so with a push remote other than the primary it is refused, naming
# where to open it by hand.
forge/ado:pr-create() {
  local pr project repo fork url
  local -a items=()
  forge:pushes-elsewhere fork || return 1
  if [[ -n "${fork}" ]]
  then
    url=$(forge/ado:repo-url) || return 1
    error "open the pull request from ${fork}'s $1 by hand, at ${url}/pullrequestcreate (az opens none from a fork)"
    return 1
  fi
  IFS=$'\t' read -r _ project repo < <(forge/ado:place) || return 1
  [[ "$1" =~ ^([0-9]+)- ]] && items=(--work-items "${BASH_REMATCH[1]}")
  if ! pr=$(forge/ado:az repos pr create --draft "${4:-true}" --source-branch "$1" --target-branch "$2" --title "$3" \
    --project "${project}" --repository "${repo}" "${items[@]}" --output json 2>/dev/null) || ! forge:json "${pr}"
  then
    error "run: az login (the pull request could not be opened)"
    return 1
  fi
  forge/ado:fold 'the pull request it opened' "${pr}" --raw-output '.pullRequestId | numbers // error'
}

forge/ado:pr-for-branch() {
  forge/ado:on-branch "the open pull request on $1" "$1" active '.[0].pullRequestId // empty'
}

forge/ado:pr-head-for-branch() {
  forge/ado:on-branch "the pull requests on $1" "$1" all \
    'import "forge" as forge; forge::ado_latest | .lastMergeSourceCommit.commitId // empty'
}

# In an Azure Pipelines run for a pull request, SYSTEM_PULLREQUEST_PULLREQUESTID gives it. Otherwise it is the open
# pull request for the current branch; a detached HEAD has none.
forge/ado:pr-here() {
  local number=${SYSTEM_PULLREQUEST_PULLREQUESTID:-} branch
  if [[ -z "${number}" ]]
  then
    branch=$(git symbolic-ref --quiet --short HEAD 2>/dev/null || true)  # a detached HEAD has no pull request
    [[ -z "${branch}" ]] || number=$(forge/ado:pr-for-branch "${branch}") || return 1
  fi
  [[ -z "${number}" ]] || printf '%s\n' "${number}"
}

forge/ado:pr-label-add() {
  local dir
  file:scratch dir || return 1
  # shellcheck disable=SC2154  # dir is set by file:scratch
  jq --null-input --compact-output --arg name "$2" '{name: $name}' > "${dir}/label.json" || return 1
  forge/ado:labels "$1" POST --in-file "${dir}/label.json"
}

forge/ado:pr-label-remove() {
  forge/ado:labels "$1" DELETE "labelIdOrName=$2"
}

forge/ado:pr-last-for-branch() {
  forge/ado:on-branch "the pull requests on $1" "$1" all \
    'import "forge" as forge; forge::ado_latest | .pullRequestId // empty'
}

forge/ado:pr-ready() {
  forge/ado:az repos pr update --id "$1" --draft false --output none >/dev/null 2>&1
}

forge/ado:pr-set-body() {
  forge/ado:az repos pr update --id "$1" --description "$2" --output none >/dev/null 2>&1
}

forge/ado:pr-set-title() {
  forge/ado:az repos pr update --id "$1" --title "$2" --output none >/dev/null 2>&1
}

forge/ado:pr-state() {
  local pr
  forge/ado:ask pr "pull request !$1" repos pr show --id "$1" || return 1
  forge/ado:fold "pull request !$1" "${pr}" --raw-output \
    'import "forge" as forge; (.status | strings // error) as $_
     | "\(forge::ado_pr_state)\t\(if .status == "completed" then .lastMergeCommit.commitId // "" else "" end)"'
}

forge/ado:pr-state-for-branch() {
  forge/ado:on-branch "the pull requests on $1" "$1" all \
    'import "forge" as forge; forge::ado_latest | forge::ado_pr_state'
}

forge/ado:pr-title() {
  local pr
  forge/ado:ask pr "the title of pull request !$1" repos pr show --id "$1" || return 1
  forge/ado:fold "the title of pull request !$1" "${pr}" --raw-output '.title | strings // error'
}

forge/ado:pr-url() {
  local url
  url=$(forge/ado:repo-url) || return 1
  printf '%s/pullrequest/%s\n' "${url}" "$1"
}

# The forge lacks rate limits it shows, as forge/ado:lacks declares, so the seam answers unsupported and never calls this.
forge/ado:quota() { return 3; }

# The forge lacks releases, as forge/ado:lacks declares, so the seam answers unsupported and never calls this.
forge/ado:releases() { return 3; }

# forge/ado:repo prints the repository as owner/name, the project being the owner: a repository belongs to its
# project, and the organization is the forge's host, as gitlab.com is GitLab's.
forge/ado:repo() {
  local project repo
  IFS=$'\t' read -r _ project repo < <(forge/ado:place) || return 1
  printf '%s/%s\n' "${project}" "${repo}"
}

forge/ado:repo-url() {
  local org project repo
  IFS=$'\t' read -r org project repo < <(forge/ado:place) || return 1
  printf '%s/%s/_git/%s\n' "${org}" "$(forge/ado:encode "${project}")" "$(forge/ado:encode "${repo}")"
}

forge/ado:require() {
  guard:require tool:az || return 1
  if ! az extension show --name azure-devops --output none >/dev/null 2>&1
  then
    error "run: az extension add --name azure-devops (the Azure DevOps commands are not installed)"
    return 1
  fi
  forge/ado:me >/dev/null
}

# forge/ado:rulesets prints one document per branch policy on the default branch, as the seam's header says: a read of
# the repository, for its id, which az repos policy list takes to filter by a branch, then a read of the policies of
# the default branch, those the project sets for every repository among them.
forge/ado:rulesets() {
  local org project repo answer id branch policies url
  use git
  IFS=$'\t' read -r org project repo < <(forge/ado:place) || return 1
  branch=$(git:default-branch) || return 1
  forge/ado:ask answer 'the repository' repos show --repository "${repo}" --project "${project}" || return 1
  id=$(forge/ado:fold 'the repository' "${answer}" --raw-output '.id | strings') || return 1
  forge/ado:ask policies 'the branch policies' repos policy list --branch "${branch}" --repository-id "${id}" \
    --project "${project}" || return 1
  url="${org}/$(forge/ado:encode "${project}")/_settings/repositories?repo=${id}&_a=policiesMid"
  forge/ado:fold 'the branch policies' "${policies}" --compact-output --arg url "${url}&refs=refs/heads/${branch}" \
    'import "forge" as forge; if type == "array" then forge::ado_controls($url) else error end'
}

# forge/ado:runs prints the builds stage's runs as one JSON array, as the seam's header says: the 50 newest runs of
# the project's pipelines, those forge.jq's builds keeps. A run's jobs would cost a call each, so every run carries
# none.
forge/ado:runs() {
  local list default tags org project
  IFS=$'\t' read -r org project _ < <(forge/ado:place) || return 1
  forge/ado:ask list 'the pipeline runs' pipelines runs list --project "${project}" --top 50 --query-order QueueTimeDesc \
    || return 1
  default=$(git:default-branch) || return 1
  tags=$(forge:tags) || return 1
  forge/ado:fold 'the pipeline runs' "${list}" --compact-output --argjson now "$(date +%s)" --arg default "${default}" \
    --argjson tags "${tags}" --arg results "${org}/$(forge/ado:encode "${project}")/_build/results" \
    'import "forge" as forge; forge::ado_builds($now; $default; $tags; $results)'
}

# forge/ado:issue-url <id> prints the work item's browser URL, for the forge's own tracker.
forge/ado:issue-url() {
  local org project
  IFS=$'\t' read -r org project _ < <(forge/ado:place) || return 1
  printf '%s/%s/_workitems/edit/%s\n' "${org}" "$(forge/ado:encode "${project}")" "$1"
}

# forge/ado:issue-new-url [<title>] prints the page a person files a new work item at, the title filled in.
forge/ado:issue-new-url() {
  local org project type
  IFS=$'\t' read -r org project _ < <(forge/ado:place) || return 1
  type=$(forge:setting ado type)
  type=${type:-Task}
  tracker:query "${org}/$(forge/ado:encode "${project}")/_workitems/create/$(forge/ado:encode "${type}")" \
    '[System.Title]' "${1:-}"
}

forge/ado:secret-scanning-alerts() { return 3; }

# forge/ado:teams prints one document per team of the project, in Azure DevOps' order, as the seam's header says: the
# teams and the first 100 members of each of the first 20, as forge/ado:teams-read reads them. A 21st team, or a team
# past its first 100 members, ends the list with {"limited": N}.
forge/ado:teams() {
  local org project teams members
  IFS=$'\t' read -r org project _ < <(forge/ado:place) || return 1
  forge/ado:teams-read teams members || return 1
  forge/ado:fold 'the teams' "${teams}" --compact-output --argjson members "${members}" \
    --arg url "${org}/$(forge/ado:encode "${project}")" 'import "forge" as forge; forge::ado_teams($members; $url; 20)'
}

# The seam's calls end here; what follows is this provider's own.

# forge/ado:place prints the organization's URL, the project, and the repository, tab-separated, read from the
# primary remote's URL as the file header lists its forms; the project and the repository decoded from the URL.
forge/ado:place() {
  local url primary org project repo
  primary=$(forge:primary) || return $?
  if ! url=$(git remote get-url --end-of-options "${primary}" 2>/dev/null)
  then
    error "run: git remote add ${primary} <url>, as this checkout has no ${primary} (the repository could not be read)"
    return 1
  fi
  url=${url%.git}
  if [[ "${url}" =~ ^(ssh://)?[^@/]+@(ssh\.dev\.azure\.com|vs-ssh\.visualstudio\.com)[:/]v3/([^/]+)/([^/]+)/([^/]+)$ ]]
  then
    org=https://dev.azure.com/${BASH_REMATCH[3]} project=${BASH_REMATCH[4]} repo=${BASH_REMATCH[5]}
  elif [[ "${url}" =~ ^(https?://)([^@/]+@)?([^/]+)(/.*)?/([^/]+)/_git/([^/]+)$ ]]
  then
    org=${BASH_REMATCH[1]}${BASH_REMATCH[3]}${BASH_REMATCH[4]} project=${BASH_REMATCH[5]} repo=${BASH_REMATCH[6]}
    [[ "${BASH_REMATCH[3]}" != *.visualstudio.com ]] || org=${org%/DefaultCollection}
  else
    error "${primary}'s URL names no Azure DevOps repository: ${url}; it reads https://dev.azure.com/<organization>/<project>/_git/<repository>"
    return 1
  fi
  printf '%s\t%s\t%s\n' "${org}" "$(forge/ado:decode "${project}")" "$(forge/ado:decode "${repo}")"
}

# forge/ado:decode <text> prints a URL's path segment decoded: %20 as a space.
forge/ado:decode() { jq --null-input --raw-output --arg s "$1" '$s | gsub("%(?<h>[0-9A-Fa-f]{2})"; .h | ascii_downcase
  | explode | map(if . >= 97 then . - 87 else . - 48 end) | .[0] * 16 + .[1] | [.] | implode)'; }

# forge/ado:encode <text> prints the text encoded for a URL's path segment.
forge/ado:encode() { jq --null-input --raw-output --arg s "$1" '$s | @uri'; }

# forge/ado:az <az argument>… runs az on the organization, as JSON, with nothing detected from git and no warning.
forge/ado:az() {
  local org
  IFS=$'\t' read -r org _ < <(forge/ado:place) || return 1
  az "$@" --org "${org}" --detect false --only-show-errors
}

# forge/ado:ask <name> <what> <az argument>… runs az as forge/ado:az does and puts the answer in the variable
# <name>. When az fails, or answers with anything but JSON, it names az login for <what>, and returns 1. An empty
# answer, as az gives for a list with nothing in it, is the empty list.
forge/ado:ask() {
  local ask_into=$1 ask_what=$2 ask_out
  shift 2
  # The repository is read first, so a checkout with no remote says so, not az.
  forge/ado:place >/dev/null || return 1
  if ! ask_out=$(forge/ado:az "$@" --output json 2>/dev/null)
  then
    error "run: az login (${ask_what} could not be read)"
    return 1
  fi
  [[ -n "${ask_out}" ]] || ask_out='[]'
  if ! forge:json "${ask_out}"
  then
    error "run: az login (${ask_what} could not be read)"
    return 1
  fi
  printf -v "${ask_into}" '%s' "${ask_out}"
}

# forge/ado:fold <what> <answer> <jq argument>… is forge:fold, naming az login.
forge/ado:fold() { forge:fold 'run: az login' "$@"; }

# forge/ado:prs <name> <what> <status> [<source branch>] reads every pull request of the repository in the status,
# active or all, page after page, and puts the list, one JSON array, in the variable <name>.
forge/ado:prs() {
  local prs_into=$1 prs_what=$2 prs_status=$3 prs_project prs_repo prs_page prs_all='[]' prs_count
  local -i prs_skip=0
  local -a prs_branch=()
  [[ -z "${4:-}" ]] || prs_branch=(--source-branch "$4")
  IFS=$'\t' read -r _ prs_project prs_repo < <(forge/ado:place) || return 1
  while :
  do
    forge/ado:ask prs_page "${prs_what}" repos pr list --project "${prs_project}" --repository "${prs_repo}" \
      --status "${prs_status}" --top 100 --skip "${prs_skip}" "${prs_branch[@]}" || return 1
    prs_all=$(forge/ado:fold "${prs_what}" "${prs_all}" --compact-output --argjson page "${prs_page}" \
      'if ($page | type) == "array" then . + $page else error end') || return 1
    prs_count=$(jq length <<<"${prs_page}")
    forge:whole "${prs_what}" "$(jq length <<<"${prs_all}")" || return 1
    (( prs_count == 100 )) || break
    prs_skip+=100
  done
  printf -v "${prs_into}" '%s' "${prs_all}"
}

# forge/ado:environments-list <name> <project> reads the project's environments, {count, value}, 101 at most, and puts
# them in the variable <name>. az devops invoke shows no continuation token, so a 101st fails the read, as the list
# would be partial.
forge/ado:environments-list() {
  local list_into=$1 list_project=$2 list_answer
  forge/ado:ask list_answer 'the environments' devops invoke --area distributedtask --resource environments \
    --route-parameters "project=${list_project}" --query-parameters '$top=101' --api-version 7.1 || return 1
  forge/ado:fold 'the environments' "${list_answer}" --exit-status '.value | arrays' >/dev/null || return 1
  if [[ "$(jq '.value | length' <<<"${list_answer}")" -gt 100 ]]
  then
    error "kit reads at most 100 environments (the environments could not be read)"
    return 1
  fi
  printf -v "${list_into}" '%s' "${list_answer}"
}

# forge/ado:teams-read <teams> <members> reads the project's teams, 21 of them at most, so a 21st says there are more,
# into the variable <teams>, one JSON array, and the members of each of the first 20, 101 of them at most, so a 101st
# says there are more, into the variable <members>, one JSON object by the team's id: a read for the teams, and one
# for each team's members.
forge/ado:teams-read() {
  local read_teams=$1 read_members=$2 read_project read_list read_ids read_id read_team read_all='{}'
  local -a read_each=()
  IFS=$'\t' read -r _ read_project _ < <(forge/ado:place) || return 1
  forge/ado:ask read_list 'the teams' devops team list --project "${read_project}" --top 21 || return 1
  read_ids=$(forge/ado:fold 'the teams' "${read_list}" --raw-output \
    'if type == "array" then .[:20][].id else error end') || return 1
  [[ -z "${read_ids}" ]] || mapfile -t read_each <<<"${read_ids}"
  for read_id in "${read_each[@]}"
  do
    forge/ado:ask read_team "the members of team ${read_id}" devops team list-member --team "${read_id}" \
      --project "${read_project}" --top 101 || return 1
    read_all=$(forge/ado:fold "the members of team ${read_id}" "${read_all}" --compact-output --arg id "${read_id}" \
      --argjson team "${read_team}" '. + {($id): ($team | arrays // error)}') || return 1
  done
  printf -v "${read_teams}" '%s' "${read_list}"
  printf -v "${read_members}" '%s' "${read_all}"
}

# forge/ado:labels <n> <method> [<route parameter>] [<az devops invoke argument>…] changes the pull request's labels
# through the REST API, which az names no command for: POST adds the label the input file names, DELETE takes off the
# label the route parameter labelIdOrName names.
forge/ado:labels() {
  local id=$1 method=$2 project repo
  local -a route=()
  shift 2
  [[ "${1:-}" != labelIdOrName=* ]] || { route=("$1"); shift; }
  IFS=$'\t' read -r _ project repo < <(forge/ado:place) || return 1
  forge/ado:az devops invoke --area git --resource pullRequestLabels --http-method "${method}" --api-version 7.1 \
    --route-parameters "project=${project}" "repositoryId=${repo}" "pullRequestId=${id}" "${route[@]}" "$@" \
    --output none >/dev/null 2>&1
}

# forge/ado:on-branch <what> <branch> <status> <jq filter> reads the pull requests on the branch in the status and
# prints what the filter makes of them, or nothing when it makes nothing of them.
forge/ado:on-branch() {
  local mrs
  forge/ado:prs mrs "$1" "$3" "$2" || return 1
  forge/ado:fold "$1" "${mrs}" --raw-output "$4"
}

# ----------------------------------------------------------------------------------------------------------------------
