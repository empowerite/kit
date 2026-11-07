# shellcheck shell=bash
# web.sh - the tracker provider for a ticket system kit reaches through its pages alone (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# modules/bash/tracker.sh loads this provider for a prefix whose record sets `provider: web`: a ticket system, such as
# Linear or Bugzilla, that kit has no command line for. The record gives the page a person files a new issue at, new,
# where {title} stands for the title, and an issue's page, url, where {id} stands for its number:
#
#   trackers:
#     - prefix: "LIN#"
#       provider: web
#       new: https://linear.app/example/team/ENG/new?title={title}
#       url: https://linear.app/example/issue/ENG-{id}
#
# kit opens those pages and does nothing else there: it cannot read, assign, close, or comment on an issue, nor say who
# the caller is, so those calls refuse, naming the page to do it on.

tracker/web:assign() { tracker/web:refuse "$1" 'assign it'; }

tracker/web:assignees() { tracker/web:refuse "$1" 'read its assignees'; }

tracker/web:close() { tracker/web:refuse "$1" 'close it'; }

tracker/web:comment() { tracker/web:refuse "$1" 'comment on it'; }

tracker/web:create() {
  error "kit cannot create an issue in the ${TRACKER_PREFIX} tracker; run: kit new issue ${TRACKER_PREFIX}, which opens its page"
  return 1
}

tracker/web:me() {
  error "kit cannot tell who you are in the ${TRACKER_PREFIX} tracker, which it reaches through its pages alone"
  return 1
}

tracker/web:new-url() {
  local page
  page=$(tracker/web:field new) || return 1
  if [[ "${page}" == *'{title}'* ]]
  then
    printf '%s\n' "${page//\{title\}/$(jq --null-input --raw-output --arg title "${1:-}" '$title | @uri')}"
  else
    printf '%s\n' "${page}"
  fi
}

tracker/web:state() { tracker/web:refuse "$1" 'read its state'; }

tracker/web:title() { tracker/web:refuse "$1" 'read its title'; }

tracker/web:unassign() { tracker/web:refuse "$1" 'unassign it'; }

tracker/web:url() {
  local page
  page=$(tracker/web:field url) || return 1
  printf '%s\n' "${page//\{id\}/$1}"
}

# The seam's calls end here; what follows is this provider's own.

# tracker/web:field <field> prints the field of the prefix's record, or refuses, naming the setting to add.
tracker/web:field() {
  local value
  value=$(tracker:field "${TRACKER_PREFIX}" "$1")
  if [[ -z "${value}" ]]
  then
    error "the ${TRACKER_PREFIX} tracker has no $1 page; set $1 in its record under trackers in kit.yml"
    return 1
  fi
  printf '%s\n' "${value}"
}

# tracker/web:refuse <id> <what> refuses to do what was asked to an issue, naming its page, where a person does it.
tracker/web:refuse() {
  local page
  page=$(tracker/web:url "$1" 2>/dev/null) || page="its tracker's page"
  error "kit cannot $2 in the ${TRACKER_PREFIX} tracker, which it reaches through its pages alone; open ${page}"
  return 1
}

# ----------------------------------------------------------------------------------------------------------------------
