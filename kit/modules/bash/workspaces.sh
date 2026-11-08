# shellcheck shell=bash
# workspaces.sh - kit's workspaces: the file, members, checkouts, and the workspace kit starts on (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A workspace is what kit looks at: one repository, or a named group of them. A group is personal, kept in
# workspaces.yml under XDG_CONFIG_HOME, else ~/.config, in kit/; only kit reads and writes it. What kit keeps for
# itself, as the last group used, is under XDG_CACHE_HOME, else ~/.cache, in kit/workspaces/.
#
# A member is a repository named by its host and its whole path, github.com/empowerite/edge or gitlab.com/group/sub/
# repo, never by a folder. A checkout is a desk under one of the file's roots, at most four levels down, and it belongs
# to the member its primary remote points at: the primary setting in its kit.yml, else origin. Its other remotes
# never make it a checkout of another member, so a fork's clone counts once.
#
# see: docs/workspaces.md
use git yaml

# workspaces_walk_depth is how many directories below each root kit looks for checkouts.
workspaces_walk_depth=4

# workspaces:file prints the path of the workspace file, which may not exist.
workspaces:file() { printf '%s\n' "${XDG_CONFIG_HOME:-${HOME}/.config}/kit/workspaces.yml"; }

# workspaces:cache prints the directory kit keeps its own workspace state in, which may not exist.
workspaces:cache() { printf '%s\n' "${XDG_CACHE_HOME:-${HOME}/.cache}/kit/workspaces"; }

# workspaces:url-member <url> prints the member a git URL names, host and path, or nothing for a remote on this
# machine or a path of fewer than two segments.
workspaces:url-member() {
  local url=$1 host path=''
  host=$(forge:url-host "${url}")
  [[ -n "${host}" ]] || return 0
  if [[ "${url}" =~ ^[a-z][a-z0-9+.-]*://[^/]*/(.*)$ ]]
  then
    path=${BASH_REMATCH[1]}
  elif [[ "${url}" =~ ^([^@/:]+@)?[^/:]+:(.*)$ ]]
  then
    path=${BASH_REMATCH[2]}
  fi
  while [[ "${path}" == */ ]]
  do
    path=${path%/}
  done
  path=${path#/}
  path=${path%.git}
  [[ "${path}" == */* ]] || return 0
  printf '%s/%s\n' "${host}" "${path}"
}

# workspaces:member-of <dir> prints the member of the checkout at <dir>, from its primary remote, or nothing.
workspaces:member-of() {
  local dir=$1 primary='' url
  [[ ! -f "${dir}/kit.yml" ]] || primary=$(yaml:value "${dir}/kit.yml" primary)
  url=$(git -C "${dir}" remote get-url --end-of-options "${primary:-origin}" 2>/dev/null) || return 0
  workspaces:url-member "${url}"
}

# workspaces:roots prints each root the file names, a leading ~ as HOME, one per line, in the file's order.
workspaces:roots() {
  local root
  while IFS= read -r root
  do
    case "${root}" in
      (\~) root=${HOME} ;;
      (\~/*) root=${HOME}/${root#\~/} ;;
    esac
    printf '%s\n' "${root}"
  done < <(yaml:list "$(workspaces:file)" roots)
}

# workspaces:checkouts prints "<member>\t<path>" for every desk under the roots that has a member, the roots in
# order and the desks in path order under each, a desk reached from two roots once.
workspaces:checkouts() {
  local root desk member
  local -A seen=()
  while IFS= read -r root
  do
    [[ -d "${root}" ]] || continue
    while IFS= read -r desk
    do
      [[ -z "${seen[${desk}]:-}" ]] || continue
      seen[${desk}]=1
      member=$(workspaces:member-of "${desk}")
      [[ -z "${member}" ]] || printf '%s\t%s\n' "${member}" "${desk}"
    done < <(git:desks "${root}" "${workspaces_walk_depth}")
  done < <(workspaces:roots)
}

# ----------------------------------------------------------------------------------------------------------------------
