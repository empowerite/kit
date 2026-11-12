# shellcheck shell=bash
# dist.sh - the dist source: a project's own download directory, or a mirror of it, by path (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A dist: coordinate, dist:<dist>/<path>, is a file a project serves from a directory of its own, at <home>/<path>:
#
#   node    https://nodejs.org/dist     index.json lists the versions, the LTS ones counted
#   go      https://go.dev/dl           ?mode=json&include=all lists them
#   gnu     https://ftp.gnu.org/gnu     each project's directory lists its tarballs, <project>-<version>.tar.*
#
# direct is the dist's home. A mirror serves a dist at <url>/<path>: it takes dist:, the dist it mirrors, and serves
# no other; without it, it is asked for any. The bytes are the project's own, so its hash is the file's sha256, as the
# lock held it for a url.

sources/dist:fetch() {
  local dist path
  sources/dist:split "$2" dist path || return 4
  sources/dist:mirrors "${dist}" || return 4
  sources:get "$1/${path}" "$3"
}

sources/dist:hash() { printf 'sha256:%s\n' "$(file:digest "$1")"; }

# sources/dist:home prints the home of the coordinate's dist, and nothing without a coordinate: a dist has no one home.
sources/dist:home() {
  local dist path
  sources/dist:split "${1:-}" dist path || return 0
  case "${dist}" in
    (node) printf 'https://nodejs.org/dist\n' ;;
    (go) printf 'https://go.dev/dl\n' ;;
    (gnu) printf 'https://ftp.gnu.org/gnu\n' ;;
    (*) ;;
  esac
}

sources/dist:keys() { printf 'dist\n'; }

sources/dist:serves() { [[ "$1" == dist:?*/?* ]]; }

# sources/dist:settings writes nothing: no language's manager reads a project's download directory.
sources/dist:settings() { :; }

sources/dist:versions() {
  local dist path doc project status=0
  sources/dist:split "$2" dist path || return 4
  sources/dist:mirrors "${dist}" || return 4
  case "${dist}" in
    (node)
      doc=$(sources:get "$1/index.json" -) || return $?
      jq --raw-output '.[] | select(.lts != false) | .version | ltrimstr("v")' <<<"${doc}" 2>/dev/null || return 1
      ;;
    (go)
      doc=$(sources:get "$1/?mode=json&include=all" -) || return $?
      jq --raw-output '.[] | .version | ltrimstr("go")' <<<"${doc}" 2>/dev/null || return 1
      ;;
    (gnu)
      project=${path%%/*}
      doc=$(sources:get "$1/${project}" -) || status=$?
      (( status == 0 )) || return "${status}"
      grep --only-matching --extended-regexp "${project}-[0-9][0-9.]*[0-9]\\.tar\\.(gz|bz2|xz)" <<<"${doc}" \
        | sed -E "s/^${project}-(.*)\\.tar\\.[a-z0-9]+\$/\\1/" | sort --unique --version-sort || true  # none listed is none
      ;;
    (*) return 4
  esac
}

# The seam's calls end here; what follows is this provider's own.

# sources/dist:split <coordinate> <dist> <path> sets the two variables to the dist and the path a dist: coordinate
# names, and returns 1 for a coordinate of another form.
sources/dist:split() {
  local -n sources_dist_name=$2 sources_dist_path=$3
  [[ "$1" =~ ^dist:([^/]+)/(.+)$ ]] || return 1
  # shellcheck disable=SC2034  # the caller's variables, through the names
  sources_dist_name=${BASH_REMATCH[1]} sources_dist_path=${BASH_REMATCH[2]}
}

# sources/dist:mirrors <dist> succeeds when the source being asked serves the dist: direct, or a mirror whose entry,
# sources_entry, names it under dist:.
sources/dist:mirrors() {
  local named
  named=$(jq --raw-output '.dist // ""' <<<"${sources_entry:-{\}}" 2>/dev/null) || named=''
  [[ -z "${named}" || "${named}" == "$1" ]]
}

# ----------------------------------------------------------------------------------------------------------------------
