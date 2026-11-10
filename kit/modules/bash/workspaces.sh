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
  local -a roots
  list:read roots yaml:list "$(workspaces:file)" roots || return 1
  for root in "${roots[@]}"
  do
    case "${root}" in
      (\~) root=${HOME} ;;
      (\~/*) root=${HOME}/${root#\~/} ;;
    esac
    printf '%s\n' "${root}"
  done
}

# workspaces:checkouts prints "<member>\t<path>" for every desk under the roots that has a member, the roots in
# order and the desks in path order under each, a desk reached from two roots once.
workspaces:checkouts() {
  local root desk member
  local -a roots desks
  local -A seen=()
  list:read roots workspaces:roots || return 1
  for root in "${roots[@]}"
  do
    [[ -d "${root}" ]] || continue
    list:read desks git:desks "${root}" "${workspaces_walk_depth}" || return 1
    for desk in "${desks[@]}"
    do
      [[ -z "${seen[${desk}]:-}" ]] || continue
      seen[${desk}]=1
      member=$(workspaces:member-of "${desk}")
      [[ -z "${member}" ]] || printf '%s\t%s\n' "${member}" "${desk}"
    done
  done
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
# member of a git URL. A host holds a dot, as github.com does, so a relative path, ./edge or ../edge, names none. With
# dirs, a directory, read from where kit was run, names the member of the checkout there. It prints nothing otherwise.
workspaces:normalize() {
  local entry=$1 dirs=${2:-} dir=$1
  [[ "${dir}" == /* ]] || dir=${kit_invoked_in:-${PWD}}/${entry}
  if [[ -n "${dirs}" && -d "${dir}" ]]
  then
    workspaces:member-of "${dir}"
  elif [[ "${entry}" == *://* || "${entry}" =~ ^[^@/:]+@[^/:]+: ]]
  then
    workspaces:url-member "${entry}"
  elif [[ "${entry}" =~ ^[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)+(/[^/[:space:]]+){2,}$ && "${entry}/" != *"/./"* &&
    "${entry}/" != *"/../"* ]]
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
  local -a entries
  local -A seen=() excluded=()
  list:read entries workspaces:entries "$1" exclude || return 1
  for entry in "${entries[@]}"
  do
    excluded[$(workspaces:normalize "${entry}")]=1
  done
  list:read entries workspaces:entries "$1" repos || return 1
  for entry in "${entries[@]}"
  do
    member=$(workspaces:normalize "${entry}")
    [[ -n "${member}" && -z "${seen[${member}]:-}" && -z "${excluded[${member}]:-}" ]] || continue
    seen[${member}]=1
    printf '%s\n' "${member}"
  done
}

# workspaces:check returns 1, after a line that names the file and what is wrong, for a file that does not parse, a
# group whose name is not a name, or a repos or exclude entry that names no member; 0 for a file that holds none.
workspaces:check() {
  local file name key entry doc wrong
  local -a names entries
  file=$(workspaces:file)
  doc=$(workspaces:json) || return 1
  wrong=$(jq --raw-output '
    def strings: type == "array" and all(.[]; type == "string");
    if (.roots // []) | strings | not then "roots is a list of directories"
    elif (.favorites // []) | strings | not then "favorites is a list of workspaces"
    elif (.workspaces // {}) | type != "object" then "workspaces is a map of names to groups"
    else first((.workspaces // {}) | to_entries[]
      | select((.value // {}) | type != "object" or ((.repos // []) | strings | not)
          or ((.exclude // []) | strings | not))
      | "\(.key) is a group, its repos and its exclude each a list of repositories") // empty
    end' <<<"${doc}")
  if [[ -n "${wrong}" ]]
  then
    error "${file}: ${wrong}"
    return 1
  fi
  list:read names workspaces:names || return 1
  for name in "${names[@]}"
  do
    if [[ ! "${name}" =~ ^[a-z0-9][a-z0-9._-]*$ ]]
    then
      error "${file}: '${name}' is not a workspace's name: lowercase letters, digits, and . _ -, no /"
      return 1
    fi
    for key in repos exclude
    do
      list:read entries workspaces:entries "${name}" "${key}" || return 1
      for entry in "${entries[@]}"
      do
        if [[ -z "$(workspaces:normalize "${entry}")" ]]
        then
          error "${file}: '${entry}' in ${name}'s ${key} names no repository; write host/owner/repo or a git URL"
          return 1
        fi
      done
    done
  done
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
  local -a remotes
  {
    list:read remotes forge:remotes 2>/dev/null || exit 1
    for remote in "${remotes[@]}"
    do
      url=$(git remote get-url --end-of-options "${remote}" 2>/dev/null) || url=''
      forge=$(forge:of "${remote}" 2>/dev/null) || forge=''
      jq --null-input --compact-output --arg name "${remote}" --arg url "${url}" --arg forge "${forge}" \
        '{name: $name, url: $url, forge: (if $forge == "" then null else $forge end)}'
    done
  } | jq --slurp --compact-output .
}

# workspaces:kind <workspace> prints "group <name>" for a group's name, or "repository <member>" for a repository, host
# and path or a git URL, without looking for a checkout. It returns 2, after a line that says so, for neither, and 1
# when the file's groups cannot be read.
workspaces:kind() {
  local workspace=$1 member
  local -a names
  if [[ "${workspace}" != */* && "${workspace}" != *:* ]]
  then
    list:read names workspaces:names || return 1
    if ! list:among "${workspace}" "${names[@]}"
    then
      error "'${workspace}' is no workspace in $(workspaces:file); run: kit workspaces"
      return 2
    fi
    printf 'group %s\n' "${workspace}"
    return 0
  fi
  member=$(workspaces:normalize "${workspace}")
  if [[ -z "${member}" ]]
  then
    error "'${workspace}' names no repository; write host/owner/repo, a git URL, or a group's name"
    return 2
  fi
  printf 'repository %s\n' "${member}"
}

# workspaces:resolve <workspace> prints a workspace as JSON: {name, kind, members}. A group's name gives its fixed
# members, by fixed; host and path, or a git URL, gives that one repository, by given. It returns 2, after a line that
# says so, for a workspace that is neither, and 1 when the file cannot be read.
workspaces:resolve() {
  local kind workspace member checkouts line
  local -a members=() json=()
  line=$(workspaces:kind "$1") || return $?
  read -r kind workspace <<<"${line}"
  if [[ "${kind}" == group ]]
  then
    list:read members workspaces:fixed "${workspace}" || return 1
  else
    members=("${workspace}")
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

# workspaces:lines-json prints the lines it reads as a JSON array of strings, an empty line left out.
workspaces:lines-json() { jq --raw-input --slurp --compact-output 'split("\n") | map(select(. != ""))'; }

# workspaces:list-json prints the workspaces as JSON: {file, roots, favorites, workspaces}, each group {name, members,
# favorite}, members its count of fixed members.
workspaces:list-json() {
  local name
  local -a groups=() names fixed
  list:read names workspaces:names || return 1
  for name in "${names[@]}"
  do
    list:read fixed workspaces:fixed "${name}" || return 1
    groups+=("$(jq --null-input --compact-output --arg name "${name}" --argjson n "${#fixed[@]}" \
      '{name: $name, members: $n}')")
  done
  printf '%s\n' "${groups[@]}" | jq --slurp --compact-output --arg file "$(workspaces:file)" \
    --argjson roots "$(workspaces:roots | workspaces:lines-json)" \
    --argjson favorites "$(workspaces:favorites | workspaces:lines-json)" \
    '{file: $file, roots: $roots, favorites: $favorites,
      workspaces: map(select(. != null) | . + {favorite: (.name as $n | $favorites | index($n) != null)})}'
}

# workspaces:last prints the last group used, or nothing: none was, or the file has no group of that name. It fails
# when the file's groups cannot be read.
workspaces:last() {
  local file last
  local -a names
  file=$(workspaces:cache)/last
  [[ -f "${file}" ]] || return 0
  IFS= read -r last < "${file}" || true  # a last line with no newline is read, and returns 1
  [[ -n "${last}" ]] || return 0
  list:read names workspaces:names || return 1
  ! list:among "${last}" "${names[@]}" || printf '%s\n' "${last}"
}

# workspaces:remember <name> keeps <name> as the last group used.
workspaces:remember() {
  local dir
  dir=$(workspaces:cache)
  mkdir -p "${dir}" && printf '%s\n' "$1" > "${dir}/last"
}

# workspaces:current [<workspace>] prints the workspace kit starts on as JSON: {kind, name, member, checkout, why}. A
# workspace given is it, why given, and none names none; else, inside a repository, that repository, named by its
# member or, with none, by its path; else the last group used; else none. It returns 2 for a workspace given that is
# unknown, and 1 when the file's groups cannot be read.
workspaces:current() {
  local given=${1:-} kind='' name='' member='' checkout='' why json
  if [[ "${given}" == none ]]
  then
    kind=none why=given
  elif [[ -n "${given}" ]]
  then
    json=$(workspaces:kind "${given}") || return $?
    read -r kind name <<<"${json}"
    why=given
    if [[ "${kind}" == repository ]]
    then
      member=${name} checkout=$(workspaces:checkouts | awk -F '\t' -v m="${name}" '$1 == m { print $2; exit }')
    fi
  elif checkout=$(git -C "${kit_invoked_in:-${PWD}}" rev-parse --show-toplevel 2>/dev/null)
  then
    kind=repository why=repository member=$(workspaces:member-of "${checkout}")
    name=${member:-${checkout}}
  else
    checkout=''
    name=$(workspaces:last) || return 1
    if [[ -n "${name}" ]]
    then
      kind=group why=last
    else
      kind=none why=none
    fi
  fi
  jq --null-input --compact-output --arg kind "${kind}" --arg name "${name}" --arg member "${member}" \
    --arg checkout "${checkout}" --arg why "${why}" '
    def some: if . == "" then null else . end;
    {kind: $kind, name: ($name | some), member: ($member | some), checkout: ($checkout | some), why: $why}'
}

# workspaces:menu-json <current> prints the current workspace's JSON, as workspaces:current prints it, with what the
# workspace menu draws added: members, a group's fixed members as --show gives each; groups, for a repository with a
# member, each group whose fixed members hold it, {name, members}, members a count; and favorites, each that names a
# group in the file or a repository, {name, kind, members, checkout}, members a group's count and checkout a
# repository's. The roots are walked once, and only when a member or a favorite needs a checkout.
workspaces:menu-json() {
  local current=$1 kind member name favorite fkind checkouts=''
  local -a members=() groups=() favorites=() fixed=() names favored
  kind=$(jq --raw-output .kind <<<"${current}")
  member=$(jq --raw-output '.member // ""' <<<"${current}")
  list:read favored workspaces:favorites || return 1
  if [[ "${kind}" == group || "${favored[*]}" == */* ]]
  then
    checkouts=$(workspaces:checkouts)
  fi
  if [[ "${kind}" == group ]]
  then
    list:read names workspaces:fixed "$(jq --raw-output .name <<<"${current}")" || return 1
    for name in "${names[@]}"
    do
      members+=("$(workspaces:member-json "${name}" fixed "${checkouts}")")
    done
  fi
  if [[ "${kind}" == repository && -n "${member}" ]]
  then
    list:read names workspaces:names || return 1
    for name in "${names[@]}"
    do
      list:read fixed workspaces:fixed "${name}" || return 1
      [[ " ${fixed[*]} " != *" ${member} "* ]] ||
        groups+=("$(jq --null-input --compact-output --arg name "${name}" --argjson n "${#fixed[@]}" \
          '{name: $name, members: $n}')")
    done
  fi
  for favorite in "${favored[@]}"
  do
    read -r fkind name < <(workspaces:kind "${favorite}" 2>/dev/null) || continue
    if [[ "${fkind}" == group ]]
    then
      list:read fixed workspaces:fixed "${name}" || return 1
      favorites+=("$(jq --null-input --compact-output --arg name "${favorite}" --argjson n "${#fixed[@]}" \
        '{name: $name, kind: "group", members: $n, checkout: null}')")
    else
      favorites+=("$(jq --null-input --compact-output --arg name "${favorite}" \
        --arg checkout "$(awk -F '\t' -v m="${name}" '$1 == m { print $2; exit }' <<<"${checkouts}")" \
        '{name: $name, kind: "repository", members: null,
          checkout: (if $checkout == "" then null else $checkout end)}')")
    fi
  done
  jq --compact-output \
    --argjson members "$(printf '%s\n' "${members[@]}" | jq --slurp --compact-output 'map(select(. != null))')" \
    --argjson groups "$(printf '%s\n' "${groups[@]}" | jq --slurp --compact-output 'map(select(. != null))')" \
    --argjson favorites "$(printf '%s\n' "${favorites[@]}" | jq --slurp --compact-output 'map(select(. != null))')" \
    '. + {members: $members, groups: $groups, favorites: $favorites}' <<<"${current}"
}

# workspaces:edit <expression> runs a yq expression over the file in place, which keeps its comments, making the file,
# and its directory, first where there is none. Values reach the expression through the environment, by strenv, never
# in its text.
workspaces:edit() {
  local file
  file=$(workspaces:file)
  if [[ ! -f "${file}" ]]
  then
    mkdir -p "$(dirname "${file}")" || return 1
    printf '# workspaces.yml - your workspaces, which kit reads and edits; see: kit help workspaces\n# %s\n' \
      "$(printf -- '-%.0s' {1..118})" > "${file}" || return 1
  fi
  yq --inplace "$1" "${file}"
}

# workspaces:json-list <item>… prints the items as a JSON array, for an expression to read through strenv.
workspaces:json-list() { jq --null-input --compact-output '$ARGS.positional' --args "$@"; }

# workspaces:new <name> <member>… makes a group of the members, in order, each once.
workspaces:new() {
  local name=$1
  shift
  kit_ws_name=${name} kit_ws_list=$(workspaces:json-list "$@") \
    workspaces:edit '.workspaces[strenv(kit_ws_name)] = {"repos": (strenv(kit_ws_list) | from_yaml)}'
}

# workspaces:add <name> <member>… appends to the group's repos the members not among them.
workspaces:add() {
  local name=$1
  shift
  kit_ws_name=${name} kit_ws_list=$(workspaces:json-list "$@") \
    workspaces:edit '.workspaces[strenv(kit_ws_name)].repos += (strenv(kit_ws_list) | from_yaml)'
}

# workspaces:drop <name> <entry>… takes the entries, as the file writes them, out of the group's repos.
workspaces:drop() {
  local name=$1
  shift
  # shellcheck disable=SC2016  # $drop and $e are yq's
  kit_ws_name=${name} kit_ws_list=$(workspaces:json-list "$@") \
    workspaces:edit '(strenv(kit_ws_list) | from_yaml) as $drop
      | .workspaces[strenv(kit_ws_name)].repos |= map(select(. as $e | ($drop | map(select(. == $e)) | length) == 0))'
}

# workspaces:favor <workspace> adds a workspace to the favorites; workspaces:disfavor takes it out.
workspaces:favor() { kit_ws_name=$1 workspaces:edit '.favorites += [strenv(kit_ws_name)]'; }
workspaces:disfavor() {
  kit_ws_name=$1 workspaces:edit '.favorites |= ((. // []) | map(select(. != strenv(kit_ws_name))))'
}

# ----------------------------------------------------------------------------------------------------------------------
