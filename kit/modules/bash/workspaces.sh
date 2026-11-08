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

# workspaces:json prints the file as JSON on one line, {} where there is none. It returns 1, after yq's line, for a file
# that does not parse.
workspaces:json() {
  local file doc
  file=$(workspaces:file)
  [[ -f "${file}" ]] || { printf '{}\n'; return 0; }
  doc=$(yaml:json "${file}") || return 1
  printf '%s\n' "${doc:-{\}}"
}

# workspaces:normalize <entry> [dirs] prints the member an entry names: host and path as they are, less a .git, or the
# member of a git URL. With dirs, a directory names the member of the checkout there. It prints nothing otherwise.
workspaces:normalize() {
  local entry=$1 dirs=${2:-}
  if [[ -n "${dirs}" && -d "${entry}" ]]
  then
    workspaces:member-of "${entry}"
  elif [[ "${entry}" == *://* || "${entry}" =~ ^[^@/:]+@[^/:]+: ]]
  then
    workspaces:url-member "${entry}"
  elif [[ "${entry}" =~ ^[A-Za-z0-9.-]+(/[^/[:space:]]+){2,}$ ]]
  then
    printf '%s\n' "${entry%.git}"
  fi
}

# workspaces:names prints the groups' names, in the file's order.
workspaces:names() { workspaces:json | jq --raw-output '(.workspaces // {}) | keys_unsorted[]'; }

# workspaces:favorites prints the favorites, in the file's order.
workspaces:favorites() { workspaces:json | jq --raw-output '(.favorites // [])[] | tostring'; }

# workspaces:entries <name> <key> prints the group's entries under repos or exclude, as the file writes them.
workspaces:entries() {
  workspaces:json | jq --raw-output --arg name "$1" --arg key "$2" '(.workspaces[$name][$key] // [])[] | tostring'
}

# workspaces:fixed <name> prints the group's fixed members, each once, in the file's order, without its exclusions.
workspaces:fixed() {
  local entry member
  local -A seen=() excluded=()
  while IFS= read -r entry
  do
    excluded[$(workspaces:normalize "${entry}")]=1
  done < <(workspaces:entries "$1" exclude)
  while IFS= read -r entry
  do
    member=$(workspaces:normalize "${entry}")
    [[ -n "${member}" && -z "${seen[${member}]:-}" && -z "${excluded[${member}]:-}" ]] || continue
    seen[${member}]=1
    printf '%s\n' "${member}"
  done < <(workspaces:entries "$1" repos)
}

# workspaces:check returns 1, after a line that names the file and what is wrong, for a file that does not parse, a
# group whose name is not a name, or a repos or exclude entry that names no member; 0 for a file that holds none.
workspaces:check() {
  local file name key entry
  file=$(workspaces:file)
  workspaces:json >/dev/null || return 1
  while IFS= read -r name
  do
    if [[ ! "${name}" =~ ^[a-z0-9][a-z0-9._-]*$ ]]
    then
      error "${file}: '${name}' is not a workspace's name: lowercase letters, digits, and . _ -, no /"
      return 1
    fi
    for key in repos exclude
    do
      while IFS= read -r entry
      do
        if [[ -z "$(workspaces:normalize "${entry}")" ]]
        then
          error "${file}: '${entry}' in ${name}'s ${key} names no repository; write host/owner/repo or a git URL"
          return 1
        fi
      done < <(workspaces:entries "${name}" "${key}")
    done
  done < <(workspaces:names)
}

# workspaces:member-json <member> <by> <checkouts> prints a member as JSON: {repository, host, path, forge, remotes,
# checkout, checkouts, by}. <checkouts> is workspaces:checkouts' output. The forge and the remotes are the checkout's,
# read there; with no checkout, the forge is the host's own, github or gitlab, else null, and there are no remotes.
workspaces:member-json() {
  local member=$1 by=$2 checkouts=$3 host forge='' remotes='[]' path
  local -a paths=()
  host=${member%%/*}
  while IFS=$'\t' read -r m path
  do
    [[ "${m}" != "${member}" ]] || paths+=("${path}")
  done <<<"${checkouts}"
  if (( ${#paths[@]} > 0 ))
  then
    remotes=$(cd "${paths[0]}" && workspaces:remotes-json) || remotes='[]'
    forge=$(jq --raw-output '.[0].forge // ""' <<<"${remotes}")
  else
    case "${host}" in
      (github.com) forge=github ;;
      (gitlab.com) forge=gitlab ;;
    esac
  fi
  jq --null-input --compact-output --arg repository "${member}" --arg host "${host}" --arg path "${member#*/}" \
    --arg forge "${forge}" --argjson remotes "${remotes}" --arg by "${by}" --args '
    {repository: $repository, host: $host, path: $path, forge: (if $forge == "" then null else $forge end),
     remotes: $remotes, checkout: ($ARGS.positional[0] // null), checkouts: $ARGS.positional, by: $by}' \
    "${paths[@]}"
}

# workspaces:remotes-json prints the remotes of the repository here as JSON, the primary first: [{name, url, forge}],
# forge null for a remote with none.
workspaces:remotes-json() {
  local remote url forge
  {
    while IFS= read -r remote
    do
      url=$(git remote get-url --end-of-options "${remote}" 2>/dev/null) || url=''
      forge=$(forge:of "${remote}" 2>/dev/null) || forge=''
      jq --null-input --compact-output --arg name "${remote}" --arg url "${url}" --arg forge "${forge}" \
        '{name: $name, url: $url, forge: (if $forge == "" then null else $forge end)}'
    done < <(forge:remotes 2>/dev/null)
  } | jq --slurp --compact-output .
}

# workspaces:resolve <workspace> prints a workspace as JSON: {name, kind, members}. A group's name gives its fixed
# members, by fixed; host and path, or a git URL, gives that one repository, by given. It returns 2, after a line that
# says so, for a workspace that is neither.
workspaces:resolve() {
  local workspace=$1 kind member checkouts
  local -a members=() json=()
  if [[ "${workspace}" != */* && "${workspace}" != *:* ]]
  then
    if ! workspaces:names | grep --quiet --line-regexp --fixed-strings -- "${workspace}"
    then
      error "'${workspace}' is no workspace in $(workspaces:file); run: kit workspaces"
      return 2
    fi
    kind=group
    readarray -t members < <(workspaces:fixed "${workspace}")
  else
    member=$(workspaces:normalize "${workspace}")
    if [[ -z "${member}" ]]
    then
      error "'${workspace}' names no repository; write host/owner/repo, a git URL, or a group's name"
      return 2
    fi
    kind=repository workspace=${member} members=("${member}")
  fi
  checkouts=$(workspaces:checkouts)
  for member in "${members[@]}"
  do
    json+=("$(workspaces:member-json "${member}" "$([[ "${kind}" == group ]] && echo fixed || echo given)" \
      "${checkouts}")")
  done
  printf '%s\n' "${json[@]}" | jq --slurp --compact-output --arg name "${workspace}" --arg kind "${kind}" \
    '{name: $name, kind: $kind, members: .}'
}

# workspaces:list-json prints the workspaces as JSON: {file, roots, favorites, workspaces}, each group {name, members,
# favorite}, members its count of fixed members.
workspaces:list-json() {
  local name
  local -a groups=()
  while IFS= read -r name
  do
    groups+=("$(jq --null-input --compact-output --arg name "${name}" --argjson n "$(workspaces:fixed "${name}" | grep -c . || true)" \
      '{name: $name, members: $n}')")
  done < <(workspaces:names)
  printf '%s\n' "${groups[@]}" | jq --slurp --compact-output --arg file "$(workspaces:file)" \
    --argjson roots "$(workspaces:roots | jq --raw-input --slurp --compact-output 'split("\n") | map(select(. != ""))')" \
    --argjson favorites "$(workspaces:favorites | jq --raw-input --slurp --compact-output 'split("\n") | map(select(. != ""))')" \
    '{file: $file, roots: $roots, favorites: $favorites,
      workspaces: map(select(. != null) | . + {favorite: (.name as $n | $favorites | index($n) != null)})}'
}

# ----------------------------------------------------------------------------------------------------------------------
