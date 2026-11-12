# shellcheck shell=bash
# ado.sh - the Azure DevOps tracker provider: work items, through az boards (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# modules/bash/tracker.sh loads this provider for a prefix whose record sets `provider: ado`. The record gives the
# organization's URL, the project that new work items are created in, and optionally the work item type (default:
# Task):
#
#   trackers:
#     - prefix: "AB#"
#       provider: ado
#       organization: https://dev.azure.com/example
#       project: Claims
#       type: Task
#
# The provider needs the az CLI with its azure-devops extension. No other part of the kit needs az.
#
# A work item is CLOSED in the states Closed, Done, and Removed (the closing states of Azure DevOps' process
# templates) and OPEN in every other state. Closing a work item sets the record's closed_state (default: Closed),
# or, when it is not planned or a duplicate, its removed_state (default: Removed); a duplicate also gets a comment
# naming the original, as Azure DevOps has no duplicate reason. The caller (me) is the identity az is signed in as, which can differ
# from the login on the forge (GitHub or GitLab).

tracker/ado:assign() {
  local org
  org=$(tracker/ado:organization) || return 1
  az boards work-item update --id "$1" --assigned-to "$2" --org "${org}" --output none 2>/dev/null
}

tracker/ado:assignees() { tracker/ado:read "$1" '.fields["System.AssignedTo"].uniqueName // ""'; }

tracker/ado:close() {
  local org state
  org=$(tracker/ado:organization) || return 1
  if [[ "$2" == completed ]]
  then
    state=$(tracker:field "${TRACKER_PREFIX}" closed_state)
    state=${state:-Closed}
  else
    state=$(tracker:field "${TRACKER_PREFIX}" removed_state)
    state=${state:-Removed}
  fi
  [[ "$2" != duplicate ]] || tracker/ado:comment "$1" "Duplicate of ${TRACKER_PREFIX}$3" || return 1
  az boards work-item update --id "$1" --state "${state}" --org "${org}" --output none 2>/dev/null
}

tracker/ado:comment() {
  local org
  org=$(tracker/ado:organization) || return 1
  az boards work-item update --id "$1" --discussion "$2" --org "${org}" --output none 2>/dev/null
}

tracker/ado:create() {
  local org project type item
  org=$(tracker/ado:organization) || return 1
  project=$(tracker:field "${TRACKER_PREFIX}" project)
  type=$(tracker:field "${TRACKER_PREFIX}" type)
  if [[ -z "${project}" ]]
  then
    error "add project: <name> to the ${TRACKER_PREFIX} tracker in kit.yml"
    return 1
  fi
  item=$(az boards work-item create --title "$1" --type "${type:-Task}" --project "${project}" --org "${org}" \
    --output json 2>/dev/null) || { error "run: az login (the work item could not be created)"; return 1; }
  jq --raw-output '.id' <<<"${item}"
}

tracker/ado:me() {
  guard:require tool:az || return 1
  az account show --query user.name --output tsv 2>/dev/null \
    || { error "run: az login (the Azure CLI is not signed in)"; return 1; }
}

tracker/ado:new-url() {
  local org project type
  org=$(tracker/ado:organization) || return 1
  project=$(tracker:field "${TRACKER_PREFIX}" project)
  type=$(tracker:field "${TRACKER_PREFIX}" type)
  type=${type:-Task}
  tracker:query "${org}/${project// /%20}/_workitems/create/${type// /%20}" '[System.Title]' "${1:-}"
}

# tracker/ado:open-issues prints one document per open work item of the record's project, oldest first, as the seam's
# header says: those an Azure Boards query finds in no closing state, each its reference under the record's prefix, its
# page, and the project as its organization, as an Azure DevOps forge's work items have it.
tracker/ado:open-issues() {
  local org project items
  org=$(tracker/ado:organization) || return 1
  project=$(tracker:field "${TRACKER_PREFIX}" project)
  if [[ -z "${project}" ]]
  then
    error "add project: <name> to the ${TRACKER_PREFIX} tracker in kit.yml"
    return 1
  fi
  guard:require tool:az || return 1
  if ! items=$(az boards query --project "${project}" --org "${org}" --output json --wiql \
    "SELECT [System.Id], [System.Title], [System.AssignedTo], [System.Tags], [System.CreatedDate],
            [System.IterationPath] FROM WorkItems
     WHERE [System.TeamProject] = @project AND [System.State] NOT IN ('Closed', 'Done', 'Removed')
     ORDER BY [System.CreatedDate] ASC" 2>/dev/null) || ! jq --exit-status 'type == "array"' <<<"${items}" >/dev/null 2>&1
  then
    error "run: az login (the work items of ${project} could not be read from ${org})"
    return 1
  fi
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq --compact-output -L "${kit_jq}" --argjson now "$(date +%s)" --arg prefix "${TRACKER_PREFIX}" \
    --arg path "${project}/" --arg page "${org}/${project// /%20}/_workitems/edit/" '
    import "forge" as forge;
    forge::ado_issues($now; $path) | .ref = "\($prefix)\(.number)" | .repository = null | .url = "\($page)\(.number)"' \
    <<<"${items}"
}

tracker/ado:state() {
  tracker/ado:read "$1" 'if (.fields["System.State"] | IN("Closed", "Done", "Removed")) then "CLOSED" else "OPEN" end'
}

tracker/ado:title() { tracker/ado:read "$1" '.fields["System.Title"]'; }

# tracker/ado:unassign clears the work item's one assignee, when it is the login named.
tracker/ado:unassign() {
  local org
  org=$(tracker/ado:organization) || return 1
  [[ "$(tracker/ado:assignees "$1")" == "$2" ]] || return 0
  az boards work-item update --id "$1" --assigned-to "" --org "${org}" --output none 2>/dev/null
}

tracker/ado:url() {
  local org project
  org=$(tracker/ado:organization) || return 1
  project=$(tracker:field "${TRACKER_PREFIX}" project)
  printf '%s/%s/_workitems/edit/%s\n' "${org}" "${project// /%20}" "$1"
}

# The seam's calls end here; what follows is this provider's own.

# shellcheck disable=SC2154  # TRACKER_PREFIX is set by tracker:of and tracker:at for the call
tracker/ado:organization() {
  local org
  org=$(tracker:field "${TRACKER_PREFIX}" organization)
  if [[ -z "${org}" ]]
  then
    error "add organization: <https://dev.azure.com/name> to the ${TRACKER_PREFIX} tracker in kit.yml"
    return 1
  fi
  printf '%s\n' "${org%/}"
}

# tracker/ado:show <id> prints the work item as az shows it.
tracker/ado:show() {
  local org
  org=$(tracker/ado:organization) || return 1
  guard:require tool:az || return 1
  az boards work-item show --id "$1" --org "${org}" --output json 2>/dev/null \
    || { error "run: az login (work item ${TRACKER_PREFIX}$1 could not be read from ${org})"; return 1; }
}

# tracker/ado:read <id> <filter> applies a jq filter to the work item. It fails when az cannot show the item.
tracker/ado:read() {
  local item
  item=$(tracker/ado:show "$1") || return 1
  jq --raw-output "$2" <<<"${item}"
}

# ----------------------------------------------------------------------------------------------------------------------
