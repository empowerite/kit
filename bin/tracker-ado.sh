# shellcheck shell=bash
# tracker-ado.sh - Azure DevOps work items, through az boards, as a tracker (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The provider bin/tracker.sh loads for a prefix whose record names `provider: ado`. The record carries the
# organization's URL and the project a new work item is created in, and optionally its type, Task by default:
#
#   trackers:
#     - prefix: "AB#"
#       provider: ado
#       organization: https://dev.azure.com/example
#       project: Claims
#       type: Task
#
# The az CLI with its azure-devops extension is this file's requirement and nobody else's. A work item is CLOSED
# in the states Closed, Done, and Removed, the closing states of Azure DevOps' process templates, and OPEN in
# every other; the caller is the identity az is signed in as, which need not be the forge's login.

# shellcheck disable=SC2154  # TRACKER_PREFIX is set by tracker:of and tracker:at for the call
tracker-ado:organization() {
  local org
  org=$(tracker:field "${TRACKER_PREFIX}" organization)
  if [[ -z "${org}" ]]
  then
    error "add organization: <https://dev.azure.com/name> to the ${TRACKER_PREFIX} tracker in .kit/settings.yml"
    return 1
  fi
  printf '%s\n' "${org%/}"
}

# tracker-ado:show <id> prints the work item as az shows it.
tracker-ado:show() {
  local org
  org=$(tracker-ado:organization) || return 1
  guard:require tool:az || return 1
  az boards work-item show --id "$1" --org "${org}" --output json 2>/dev/null \
    || { error "run: az login (work item ${TRACKER_PREFIX}$1 could not be read from ${org})"; return 1; }
}

# tracker-ado:read <id> <filter> applies a jq filter to the work item, failing where az could not show it.
tracker-ado:read() {
  local item
  item=$(tracker-ado:show "$1") || return 1
  jq --raw-output "$2" <<<"${item}"
}

tracker-ado:title() { tracker-ado:read "$1" '.fields["System.Title"]'; }

tracker-ado:state() {
  tracker-ado:read "$1" 'if (.fields["System.State"] | IN("Closed", "Done", "Removed")) then "CLOSED" else "OPEN" end'
}

tracker-ado:assignees() { tracker-ado:read "$1" '.fields["System.AssignedTo"].uniqueName // ""'; }

tracker-ado:assign() {
  local org
  org=$(tracker-ado:organization) || return 1
  az boards work-item update --id "$1" --assigned-to "$2" --org "${org}" --output none 2>/dev/null
}

tracker-ado:me() {
  guard:require tool:az || return 1
  az account show --query user.name --output tsv 2>/dev/null \
    || { error "run: az login (the Azure CLI is not signed in)"; return 1; }
}

tracker-ado:create() {
  local org project type item
  org=$(tracker-ado:organization) || return 1
  project=$(tracker:field "${TRACKER_PREFIX}" project)
  type=$(tracker:field "${TRACKER_PREFIX}" type)
  if [[ -z "${project}" ]]
  then
    error "add project: <name> to the ${TRACKER_PREFIX} tracker in .kit/settings.yml"
    return 1
  fi
  item=$(az boards work-item create --title "$1" --type "${type:-Task}" --project "${project}" --org "${org}" \
    --output json 2>/dev/null) || { error "run: az login (the work item could not be created)"; return 1; }
  jq --raw-output '.id' <<<"${item}"
}

tracker-ado:url() {
  local org project
  org=$(tracker-ado:organization) || return 1
  project=$(tracker:field "${TRACKER_PREFIX}" project)
  printf '%s/%s/_workitems/edit/%s\n' "${org}" "${project// /%20}" "$1"
}

# ----------------------------------------------------------------------------------------------------------------------
