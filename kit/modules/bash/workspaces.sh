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
use git yaml file

# workspaces_walk_depth is how many directories below each root kit looks for checkouts.
workspaces_walk_depth=4

# workspaces_lock_wait is how long workspaces:lock waits for a lock another process holds, in tenths of a second by the
# clock.
workspaces_lock_wait=300

# workspaces_checked is 1 once workspaces:check has passed, so a subshell does not check the file again; empty until it
# does.
workspaces_checked=''

# workspaces:file prints the path of the workspace file, which may not exist.
workspaces:file() { printf '%s\n' "${XDG_CONFIG_HOME:-${HOME}/.config}/kit/workspaces.yml"; }

# workspaces:cache prints the directory kit keeps its own workspace state in, which may not exist.
workspaces:cache() { printf '%s\n' "${XDG_CACHE_HOME:-${HOME}/.cache}/kit/workspaces"; }

# workspaces:unwritable <what> says that workspaces:cache's directory cannot be made or written, so <what> cannot be,
# and returns 1.
workspaces:unwritable() {
  error "make $(workspaces:cache) writable, or set XDG_CACHE_HOME ($1 could not be written there)"
  return 1
}

# workspaces:in-repository [<dir>] prints the top of the working tree <dir>, else the directory here, is in, and
# succeeds. Outside one it prints nothing and returns 1: a bare repository and a .git directory have no working tree, so
# they are outside a repository. A verb is here at its repository's top, as kit:enter moves there, or where it was run.
workspaces:in-repository() {
  git -C "${1:-${PWD}}" rev-parse --show-toplevel 2>/dev/null || return 1
}

# workspaces:spelled <member> prints a member as kit spells it, so one repository has one name: the host in lower case,
# and on github.com, gitlab.com, and the hosts under them, whose paths match in any case, the path in lower case too.
workspaces:spelled() {
  local host=${1%%/*} path=${1#*/}
  host=${host,,}
  case "${host}" in
    (github.com | gitlab.com | *.github.com | *.gitlab.com) path=${path,,} ;;
  esac
  printf '%s/%s\n' "${host}" "${path}"
}

# workspaces:url-member <url> prints the member a git URL names, host and path as workspaces:spelled spells them, or
# nothing for a remote on this machine or a path of fewer than two segments.
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
  workspaces:spelled "${host}/${path}"
}

# workspaces:member-of <dir> prints the member of the checkout at <dir>, from its primary remote, or nothing. A kit.yml
# that does not parse names no primary, so origin is read and yq's line is dropped: one clone does not stop the walk
# of the roots, and workspaces:unread says why where its member is drawn.
workspaces:member-of() {
  local dir=$1 primary='' url
  [[ ! -f "${dir}/kit.yml" ]] || primary=$(yaml:value "${dir}/kit.yml" primary 2>/dev/null) || primary=''
  url=$(git -C "${dir}" remote get-url --end-of-options "${primary:-origin}" 2>/dev/null) || return 0
  workspaces:url-member "${url}"
}

# workspaces:unread <checkout> prints "<checkout>/kit.yml does not parse, so its primary is read as origin: <yq's
# line>" for a checkout whose kit.yml yq cannot parse, or nothing.
workspaces:unread() {
  local file=$1/kit.yml said
  [[ -f "${file}" ]] || return 0
  ! said=$(yaml:value "${file}" primary 2>&1 >/dev/null) || return 0
  [[ -n "${said}" ]] || return 0
  # yaml:run's line is "❌ <file>: <yq's line>", painted where stderr is a terminal.
  said=${said%%$'\n'*}
  said=${said#*"${file}: "}
  printf '%s does not parse, so its primary is read as origin: %s\n' "${file}" "${said%$'\e[0m'}"
}

# workspaces:roots prints each root the file names once, a leading ~ as HOME and without its trailing slashes, one per
# line, in the file's order.
workspaces:roots() {
  local root
  local -a roots
  local -A seen=()
  list:read roots yaml:list "$(workspaces:file)" roots || return 1
  for root in "${roots[@]}"
  do
    case "${root}" in
      (\~) root=${HOME} ;;
      (\~/*) root=${HOME}/${root#\~/} ;;
    esac
    while [[ "${root}" == ?*/ ]]
    do
      root=${root%/}
    done
    # A space leads each key, as bash takes no empty key, and a root may be written empty.
    [[ -z "${seen[" ${root}"]:-}" ]] || continue
    seen[" ${root}"]=1
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

# workspaces:normalize <entry> [dirs] prints the member an entry names, as workspaces:spelled spells it: host and path,
# less a .git, or the member of a git URL. A host holds a dot, as github.com does, so a relative path, ./edge or
# ../edge, names none. With dirs, a directory, read from where kit was run, names the member of the checkout there. It
# prints nothing otherwise.
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
    workspaces:spelled "${entry%.git}"
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

# workspaces:fixed <name> prints the group's fixed members, each once, in the file's order, without its exclusions. An
# exclude entry that names no member takes none out; workspaces:check refuses it.
workspaces:fixed() {
  local entry member
  local -a entries
  local -A seen=() excluded=()
  list:read entries workspaces:entries "$1" exclude || return 1
  for entry in "${entries[@]}"
  do
    member=$(workspaces:normalize "${entry}")
    [[ -z "${member}" ]] || excluded[${member}]=1
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

# workspaces:misplaced <verb> <word>… refuses --workspace given among a verb's own words, where kit flow, kit
# workspace, and kit workspaces once took it: it is a kit option, before the verb, which bin/kit, the loader, reads.
# It returns 2, after a line that names the form, and 0 when no word is --workspace.
workspaces:misplaced() {
  local verb=$1 word
  shift
  for word in "$@"
  do
    [[ "${word}" != -- ]] || return 0
    if [[ "${word}" == --workspace || "${word}" == --workspace=* ]]
    then
      error "--workspace is a kit option, before the verb: kit --workspace=<workspace> ${verb}; see: kit help ${verb}"
      return 2
    fi
  done
  return 0
}

# workspaces:name-refusal <name> prints why <name> cannot name a group: it is none, which --workspace= takes for no
# workspace, or it does not start with a lowercase letter or a digit, or holds other than lowercase letters, digits,
# and . _ -. It prints nothing for a name a group can take.
workspaces:name-refusal() {
  if [[ "$1" == none ]]
  then
    printf '%s\n' "'none' is not a workspace's name: --workspace=none names no workspace"
  elif [[ ! "$1" =~ ^[a-z0-9][a-z0-9._-]*$ ]]
  then
    printf '%s %s\n' "'$1' is not a workspace's name: it starts with a lowercase letter or a digit, and holds" \
      "lowercase letters, digits, and . _ - alone"
  fi
}

# workspaces:check returns 1, after a line that names the file and what is wrong, for a file that does not parse or is
# not a map, a group that holds a key other than repos and exclude, a group whose name workspaces:name-refusal
# refuses, or a repos or exclude entry that names no member; 0 for a file that holds none. Passing, it sets
# workspaces_checked, and once it is set it checks nothing and returns 0.
workspaces:check() {
  local file name key entry doc wrong
  local -a names entries
  [[ -z "${workspaces_checked}" ]] || return 0
  file=$(workspaces:file)
  doc=$(workspaces:json) || return 1
  # yq prints a scalar file bare, as text jq does not parse, so jq's failure here is a file that is not a map.
  if ! jq --exit-status 'type == "object" or . == null' <<<"${doc}" >/dev/null 2>&1
  then
    error "${file}: the file is a map of roots, favorites, and workspaces"
    return 1
  fi
  # $q is a single quote, which the program's own quotes cannot hold.
  wrong=$(jq --raw-output --arg q "'" '
    def strings: type == "array" and all(.[]; type == "string");
    if (.roots // []) | strings | not then "roots is a list of directories"
    elif (.favorites // []) | strings | not then "favorites is a list of workspaces"
    elif (.workspaces // {}) | type != "object" then "workspaces is a map of names to groups"
    else first((.workspaces // {}) | to_entries[] | .key as $group | (.value // {})
      | if type != "object" or ((.repos // []) | strings | not) or ((.exclude // []) | strings | not)
        then "\($group) is a group, its repos and its exclude each a list of repositories"
        else first(keys_unsorted[] | select(. != "repos" and . != "exclude")
          | if . == "match" or . == "project"
            then "\($group)\($q)s \(.) is not built yet; this kit reads a group\($q)s repos and exclude, so list its"
              + " members under repos"
            else "\($q)\(.)\($q) in \($group) is no key of a group; a group has repos and exclude"
            end) // empty
        end) // empty
    end' <<<"${doc}") || return 1
  if [[ -n "${wrong}" ]]
  then
    error "${file}: ${wrong}"
    return 1
  fi
  list:read names workspaces:names || return 1
  for name in "${names[@]}"
  do
    wrong=$(workspaces:name-refusal "${name}")
    if [[ -n "${wrong}" ]]
    then
      error "${file}: ${wrong}"
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
  workspaces_checked=1
}

# workspaces:member-json <member> <by> <checkouts> prints a member as JSON: {repository, host, path, forge, remotes,
# checkout, checkouts, by}. <checkouts> is workspaces:checkouts' output. The forge and the remotes are the checkout's,
# read there; with no checkout, the forge is the host's own, github or gitlab, else null, and there are no remotes. A
# checkout whose kit.yml does not parse is warned of, with workspaces:unread's line.
workspaces:member-json() {
  local member=$1 by=$2 checkouts=$3 host forge='' remotes='[]' path unread
  local -a paths=()
  host=${member%%/*}
  while IFS=$'\t' read -r m path
  do
    [[ "${m}" != "${member}" ]] || paths+=("${path}")
  done <<<"${checkouts}"
  if (( ${#paths[@]} > 0 ))
  then
    unread=$(workspaces:unread "${paths[0]}")
    [[ -z "${unread}" ]] || warn "${unread}"
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
# when the file's groups cannot be read. It reads the file as it is: each caller runs workspaces:check first.
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
# members, by fixed; host and path, or a git URL, gives that one repository, by given. It checks the file first, as
# workspaces:check does. It returns 2, after a line that says so, for a workspace that is neither, and 1 when the file
# cannot be read or is refused.
workspaces:resolve() {
  local kind workspace member checkouts line
  local -a members=() json=()
  workspaces:check || return 1
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

# workspaces:remember <name> keeps <name> as the last group used. It returns 1, saying so, where workspaces:cache's
# directory cannot be made or written.
workspaces:remember() {
  local dir
  dir=$(workspaces:cache)
  { mkdir -p "${dir}" && printf '%s\n' "$1" > "${dir}/last"; } 2>/dev/null \
    || workspaces:unwritable "the last group used"
}

# workspaces:forget forgets the last group used. It returns 1, saying so, where the file cannot be removed.
workspaces:forget() {
  rm -f "$(workspaces:cache)/last" 2>/dev/null || workspaces:unwritable "the last group used"
}

# workspaces:current [<workspace>] prints the workspace kit starts on as JSON: {kind, name, member, checkout, why}. A
# workspace given is it, why given, and none names none; else, inside a repository, that repository, named by its
# member or, with none, by its path; else the last group used; else none. With a workspace given other than none, and
# outside a repository, it checks the file, as workspaces:check does, before it reads it; inside a repository with no
# workspace given, it reads no file. It returns 2 for a workspace given that is unknown, and 1 when the file's groups
# cannot be read or the file is refused.
workspaces:current() {
  local given=${1:-} kind='' name='' member='' checkout='' why json
  if [[ "${given}" == none ]]
  then
    kind=none why=given
  elif [[ -n "${given}" ]]
  then
    workspaces:check || return 1
    json=$(workspaces:kind "${given}") || return $?
    read -r kind name <<<"${json}"
    why=given
    if [[ "${kind}" == repository ]]
    then
      member=${name} checkout=$(workspaces:checkouts | awk -F '\t' -v m="${name}" '$1 == m { print $2; exit }')
    fi
  elif checkout=$(workspaces:in-repository)
  then
    kind=repository why=repository member=$(workspaces:member-of "${checkout}")
    name=${member:-${checkout}}
  else
    checkout=''
    workspaces:check || return 1
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

# workspaces:across [<workspace>] prints the workspace a verb draws across: the one given, unless it is none; with none
# given, the last group used, when workspaces:current starts there, outside a repository, as workspaces:in-repository
# decides; else nothing. Inside a repository, or with no group used, it reads nothing, not even the file. It returns
# workspaces:current's status.
workspaces:across() {
  local given=${1:-} json
  if [[ -n "${given}" ]]
  then
    [[ "${given}" == none ]] || printf '%s\n' "${given}"
    return 0
  fi
  ! workspaces:in-repository >/dev/null || return 0
  [[ -f "$(workspaces:cache)/last" ]] || return 0
  json=$(workspaces:current) || return $?
  jq --raw-output 'select(.why == "last") | .name' <<<"${json}"
}

# workspaces:menu-json <current> prints the current workspace's JSON, as workspaces:current prints it, with what the
# workspace menu draws added: members, a group's fixed members as --show gives each; groups, for a repository with a
# member, each group whose fixed members hold it, {name, members}, members a count; and favorites, each that names a
# group in the file or a repository, {name, kind, members, checkout}, members a group's count and checkout a
# repository's. The roots are walked once, and only when a member or a favorite needs a checkout. It returns 1, saying
# so, when the file's groups or favorites cannot be read.
workspaces:menu-json() {
  local current=$1 kind member name favorite fkind line status checkouts=''
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
    status=0
    line=$(workspaces:kind "${favorite}" 2>/dev/null) || status=$?
    # A favorite that names no group in the file and no repository, as one whose group was removed, is left out.
    (( status != 2 )) || continue
    if (( status != 0 ))
    then
      error "run: kit workspaces (the groups in $(workspaces:file) could not be read, so the favorite '${favorite}'" \
        "cannot be drawn)"
      return 1
    fi
    read -r fkind name <<<"${line}"
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

# workspaces:lock takes the lock on the workspace file for this command, or returns 0 when this command holds it. The
# lock is workspaces.yml.lock in workspaces:cache's directory: a symbolic link that names the command's process, made by
# ln -s, which fails when the link is there, so one process alone makes it, and removed at exit when it still names the
# process. A lock whose target is no number, or names a process that is gone, is removed and taken. Any other is
# looked at again each 0.1 second until workspaces_lock_wait tenths of a second have passed by the clock, however long
# each look takes, and then it returns 1, naming the process that holds it. Where the directory cannot be made or
# written, it returns 1 at once, saying so.
workspaces:lock() {
  local dir lock aside holder='' moved start=${SECONDS}
  dir=$(workspaces:cache)
  lock=${dir}/workspaces.yml.lock aside=${dir}/workspaces.yml.lock.$$
  [[ "$(readlink "${lock}" 2>/dev/null)" != "$$" ]] || return 0
  mkdir -p "${dir}" 2>/dev/null || workspaces:unwritable "the lock on $(workspaces:file)" || return 1
  until ln -s "$$" "${lock}" 2>/dev/null
  do
    # In a directory that cannot be written, no lock can be made, nor a stale one moved aside.
    if [[ ! -w "${dir}" ]]
    then
      workspaces:unwritable "the lock on $(workspaces:file)"
      return 1
    fi
    if holder=$(readlink "${lock}" 2>/dev/null) \
      && { [[ ! "${holder}" =~ ^[0-9]+$ ]] || ! kill -0 "${holder}" 2>/dev/null; }
    then
      # A stale lock is renamed aside, under a name of this process's own, before it is removed. A rename is atomic, so
      # of two processes that found the same dead holder, one moves the stale link; the other's move fails, as the link
      # is gone, or moves the link the first has made since. That one finds a live process in what it moved, puts the
      # link back by ln -s, which never replaces a link a third has made, and waits. Removed by readlink and then rm,
      # the second would remove the first's fresh lock, and both would hold it. Only a third that makes its link in
      # the instant the first's is aside holds the lock beside the first; no test can set that instant up.
      rm -f "${aside}"
      if mv "${lock}" "${aside}" 2>/dev/null
      then
        moved=$(readlink "${aside}" 2>/dev/null) || moved=''
        if [[ "${moved}" == "${holder}" ]]
        then
          rm -f "${aside}"
          continue
        fi
        [[ -z "${moved}" ]] || ln -s "${moved}" "${lock}" 2>/dev/null || true  # a lock a third made since stands
        rm -f "${aside}"
        holder=${moved}
      fi
    fi
    if (( (SECONDS - start) * 10 >= workspaces_lock_wait ))
    then
      if [[ -n "${holder}" ]]
      then
        error "$(workspaces:file) is being edited by process ${holder}; run the command again"
      else
        error "run the command again (the lock on $(workspaces:file) could not be made in ${dir})"
      fi
      return 1
    fi
    sleep 0.1
  done
  file:at-exit forge:unlock "${lock}" "$$"
}

# workspaces:edit <expression> takes the file's lock, as workspaces:lock does, and runs a yq expression over the file
# in place, which keeps its comments, making the file, and its directory, first where there is none. Values reach the
# expression through the environment, by strenv, never in its text.
workspaces:edit() {
  local file
  workspaces:lock || return 1
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

# workspaces:rename <name> <new> gives a group a new name, in its place in the file and with its comments, and gives
# its favorite the new name too. It writes favorites only where the file has them.
workspaces:rename() {
  kit_ws_name=$1 kit_ws_new=$2 workspaces:edit '(.workspaces[strenv(kit_ws_name)] | key) = strenv(kit_ws_new)
    | with(select(has("favorites")); (.favorites[] | select(. == strenv(kit_ws_name))) = strenv(kit_ws_new))'
}

# workspaces:delete <name> takes a group, and its favorite, out of the file. It writes favorites only where the file
# has them.
workspaces:delete() {
  kit_ws_name=$1 workspaces:edit 'del(.workspaces[strenv(kit_ws_name)])
    | with(select(has("favorites")); .favorites |= map(select(. != strenv(kit_ws_name))))'
}

# workspaces:favor <workspace> adds a workspace to the favorites; workspaces:disfavor takes it out.
workspaces:favor() { kit_ws_name=$1 workspaces:edit '.favorites += [strenv(kit_ws_name)]'; }
workspaces:disfavor() {
  kit_ws_name=$1 workspaces:edit '.favorites |= ((. // []) | map(select(. != strenv(kit_ws_name))))'
}

# ----------------------------------------------------------------------------------------------------------------------
