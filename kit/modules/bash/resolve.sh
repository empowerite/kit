# shellcheck shell=bash
# resolve.sh - the packages in reach, and the active set from names, dependencies, and bundles (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# A package is a directory with a package.yml: packages/<name>/ in the release, in a source kit.yml names, or under
# the repository's own .kit/packages/. A name is looked up in the sources last first, then the release, then the
# repository. package.yml says whether the package is on by default, which packages it depends on, which members a
# bundle includes, what a template asks for, and what the package expects of a file the repository owns.
#
# see: docs/packages.md

use yaml sources

# resolve_local is the root of the repository's own packages, .kit/packages/<name>/, laid out like the release's.
declare -gr resolve_local=.kit

# resolve:reach [<root>] prints the packages in reach, in the release at <root>, in the sources kit.yml
# names, and in the repository's .kit/packages/. A package is a directory that contains a package.yml: packages/<name>/, or
# packages/<namespace>/<name>/ under a namespace, a directory with no package.yml of its own. Its name is its path:
# kit, or style/bash. A package's own subdirectories are never packages.
resolve:reach() {
  local root=${1:-${kit_root}} dir
  resolve:roots || return 1
  for dir in "${root}" "${resolve_source_dirs[@]}" "${resolve_local}"
  do
    [[ -d "${dir}/packages" ]] || continue
    (cd "${dir}/packages" && find . -mindepth 2 -maxdepth 3 -name package.yml | sed 's@^\./@@; s@/package.yml$@@')
  done | LC_ALL=C sort --unique | awk '
    { name[NR] = $0; if ($0 !~ /\//) top[$0] = 1 }
    END {
      for (i = 1; i <= NR; i++) { split(name[i], part, "/"); if (!(name[i] ~ /\// && part[1] in top)) print name[i] }
    }'
}

# resolve:expand <root> <name>… prints each package a name stands for, in reach of the release at <root>: the name
# itself, or every package a glob matches, such as style/*. It fails on a glob that matches nothing, and leaves a
# plain name to resolve:require.
resolve:expand() {
  local root=$1 name package matched
  local -a reach
  shift
  for name in "$@"
  do
    if [[ "${name}" != *[*?[]* ]]
    then
      printf '%s\n' "${name}"
      continue
    fi
    matched=0
    list:read reach resolve:reach "${root}" || return 1
    for package in "${reach[@]}"
    do
      # shellcheck disable=SC2053  # the glob is the point
      if [[ "${package}" == ${name} ]]
      then
        printf '%s\n' "${package}"
        matched=1
      fi
    done
    if (( matched == 0 ))
    then
      error "'${name}' matches no package in reach; the packages are: ${reach[*]}"
      return 2
    fi
  done
}

# resolve_source_names and resolve_source_dirs hold the sources kit.yml names that are in the cache, in
# order, each source's name beside the directory that holds its packages/; resolve:roots fills them once.
declare -ga resolve_source_names=() resolve_source_dirs=()
declare -g resolve_sources_read=false

# resolve:roots reads the sources' roots once per process, fetching first any source the cache lacks, so a
# source's digest is checked before anything of it is read. A source that cannot be fetched is named, and left out.
resolve:roots() {
  [[ "${resolve_sources_read}" == false ]] || return 0
  local name dir row
  local -a rows
  # A source that cannot be fetched has said so, and is left out: sources:roots lists only what is in the cache.
  sources:fetch-all || true
  list:read rows sources:roots || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r name dir <<<"${row}"
    [[ -n "${name}" ]] || continue
    resolve_source_names+=("${name}")
    resolve_source_dirs+=("${dir}")
  done
  resolve_sources_read=true
}

# resolve_homes maps "<root>|<package>" to the directory that contains the package, and resolve_fields maps
# "<root>|<package>|<field>" to what the package's metadata holds for a field, space-separated, and
# "<root>|<package>|" to 1 once that metadata is read:
#
#   keys         its top-level keys
#   description  1 when it has a description, and nothing when it has none
#   default      true or false, other for any other value, and nothing when it has none
#   depends      the packages it depends on
#   includes     the members it includes
#
# resolve:locate and resolve:read fill them once per process: filled in a verb's own shell, as render:all fills them,
# they answer every subshell under it with no fork. resolve:locate sets resolve_home and resolve_meta for its caller
# to read.
declare -gA resolve_homes=() resolve_fields=()
declare -g resolve_home='' resolve_meta=''

# resolve:locate <root> <package> sets resolve_home to the directory that contains the package, and resolve_meta to
# the path of its metadata: the last source that has it, else <root> when the release has it, else the repository's
# .kit. The package's files are under <home>/packages/<package>/. It prints nothing, so a caller asks with no fork.
resolve:locate() {
  local memo="$1|$2" i home
  if [[ -z "${resolve_homes[${memo}]+x}" ]]
  then
    resolve:roots || return 1
    home=$1
    for (( i = ${#resolve_source_dirs[@]} - 1; i >= 0; i-- ))
    do
      if [[ -f "${resolve_source_dirs[i]}/packages/$2/package.yml" ]]
      then
        home=${resolve_source_dirs[i]}
        break
      fi
    done
    if (( i < 0 )) && [[ ! -f "$1/packages/$2/package.yml" && -f "${resolve_local}/packages/$2/package.yml" ]]
    then
      home=${resolve_local}
    fi
    resolve_homes[${memo}]=${home}
  fi
  resolve_home=${resolve_homes[${memo}]}
  resolve_meta=${resolve_home}/packages/$2/package.yml
}

# resolve:home <root> <package> prints the directory that contains the package, as resolve:locate finds it.
resolve:home() {
  resolve:locate "$1" "$2" || return 1
  printf '%s\n' "${resolve_home}"
}

# resolve:origin <root> <package> prints where the package comes from: "release", "repository", or the name of the
# source it comes from.
resolve:origin() {
  local i
  resolve:locate "$1" "$2" || return 1
  for i in "${!resolve_source_dirs[@]}"
  do
    if [[ "${resolve_home}" == "${resolve_source_dirs[i]}" ]]
    then
      printf '%s\n' "${resolve_source_names[i]}"
      return 0
    fi
  done
  if [[ "${resolve_home}" == "$1" ]]
  then printf 'release\n'
  else printf 'repository\n'
  fi
}

# resolve:meta <root> <package> prints the path of the package's metadata.
resolve:meta() {
  resolve:locate "$1" "$2" || return 1
  printf '%s\n' "${resolve_meta}"
}

# resolve:description <root> <package> prints the package's one-line description.
resolve:description() {
  resolve:locate "$1" "$2" || return 1
  yaml:value "${resolve_meta}" description
}

# resolve:read <root> <package> reads the package's metadata into resolve_fields, once per process, in one read: each
# read of a package.yml is a yq process, until the yaml module keeps its result. A read that fails is not kept, so it
# is tried again, and says why again, the next time it is asked.
resolve:read() {
  local memo="$1|$2|" row field
  local -a rows
  [[ -z "${resolve_fields[${memo}]+x}" ]] || return 0
  resolve:locate "$1" "$2" || return 1
  # Each part is an array, taken apart at the end: yq gives a literal a value even where a select matched nothing.
  yaml:lines rows eval "${resolve_meta}" fields 'explode(.) | (if kind == "map" then . else {} end)
    | (keys | map("keys\t" + tostring))
      + ([.description] | map(select(kind == "scalar" and tag != "!!null")) | map(select(tostring | test("[^\n]")))
        | map("description\t1"))
      + ([.default] | map(select(kind == "scalar" and tag != "!!null"))
        | map(if tostring == "true" or tostring == "false" then "default\t" + tostring else "default\tother" end))
      + (.depends | (if kind == "seq" then . else [] end) | map(select(kind == "scalar" and tag != "!!null"))
        | map("depends\t" + tostring))
      + (.includes | (if kind == "seq" then . else [] end) | map(select(kind == "scalar" and tag != "!!null"))
        | map("includes\t" + tostring))
    | .[]' || return 1
  for field in keys description default depends includes
  do
    resolve_fields[${memo}${field}]=''
  done
  for row in "${rows[@]}"
  do
    field=${row%%$'\t'*}
    resolve_fields[${memo}${field}]+=${resolve_fields[${memo}${field}]:+ }${row#*$'\t'}
  done
  resolve_fields[${memo}]=1
}

# resolve:field <array> <root> <package> <field> fills the named array with the words resolve_fields holds for the
# field, reading the package's metadata first when this process has not.
resolve:field() {
  local -n resolve_field_into=$1
  local IFS=$' \t\n'
  resolve_field_into=()
  resolve:read "$2" "$3" || return 1
  # shellcheck disable=SC2034  # the caller reads the array
  read -r -a resolve_field_into <<<"${resolve_fields[$2|$3|$4]}"
}

# resolve:depends <root> <package> prints the packages this one depends on, one per line.
resolve:depends() {
  local -a depends
  resolve:field depends "$1" "$2" depends || return 1
  (( ${#depends[@]} == 0 )) || printf '%s\n' "${depends[@]}"
}

# resolve:includes <root> <package> prints the members of this bundle, one per line: packages activated with it, each
# of which can be removed on its own.
resolve:includes() {
  local -a includes
  resolve:field includes "$1" "$2" includes || return 1
  (( ${#includes[@]} == 0 )) || printf '%s\n' "${includes[@]}"
}

# resolve:defaults <root> prints the packages whose default is true. They are active unless removed.
resolve:defaults() {
  local package
  local -a reach
  list:read reach resolve:reach "$1" || return 1
  for package in "${reach[@]}"
  do
    # A read that fails has said so, and the package is not on by default.
    resolve:read "$1" "${package}" || continue
    [[ "${resolve_fields[$1|${package}|default]}" == true ]] && printf '%s\n' "${package}"
  done
  return 0
}

# resolve:require <root> <name>… fails on a package that is not in reach, and lists the packages that are.
resolve:require() {
  local root=$1 package
  shift
  for package in "$@"
  do
    resolve:locate "${root}" "${package}" || return 1
    if [[ ! -f "${resolve_meta}" ]]
    then
      error "'${package}' is not a package in reach; the packages are:" \
        "$(resolve:reach "${root}" | paste -s -d ' ' -)"
      return 2
    fi
  done
}

# resolve:active <root> <excluded> <name>… prints the named packages, the packages they depend on, and the members
# they include, each once. <excluded> is a space-separated list of packages that no bundle brings in; a package that
# another depends on comes in regardless. Each package comes after the packages it depends on and includes. Ties
# are broken alphabetically, so the order is the same on every machine. It fails on a cycle.
resolve:active() {
  local root=$1 excluded=$2 package dep ready placed_count=0
  shift 2
  (( $# > 0 )) || return 0
  local -A wanted=() placed=()
  local -a queue=("$@") names=() order=() depends includes
  # A read that fails has said so, and the package stands with no depends or includes, as the others are placed.
  while (( ${#queue[@]} > 0 ))
  do
    package=${queue[0]}
    queue=("${queue[@]:1}")
    [[ -n "${wanted[${package}]:-}" ]] && continue
    wanted[${package}]=1
    resolve:field depends "${root}" "${package}" depends || true  # a read that fails has said so
    resolve:field includes "${root}" "${package}" includes || true  # a read that fails has said so
    queue+=("${depends[@]}")
    for dep in "${includes[@]}"
    do
      # shellcheck disable=SC2086  # the list splits on purpose
      list:among "${dep}" ${excluded} || queue+=("${dep}")
    done
  done
  readarray -t names < <(printf '%s\n' "${!wanted[@]}" | LC_ALL=C sort)
  while (( placed_count < ${#names[@]} ))
  do
    ready=''
    for package in "${names[@]}"
    do
      [[ -n "${placed[${package}]:-}" ]] && continue
      # An excluded member is not waited for.
      resolve:field depends "${root}" "${package}" depends || true  # a read that fails has said so
      resolve:field includes "${root}" "${package}" includes || true  # a read that fails has said so
      for dep in "${depends[@]}" "${includes[@]}"
      do
        [[ -z "${wanted[${dep}]:-}" || -n "${placed[${dep}]:-}" ]] || continue 2
      done
      ready=${package}
      break
    done
    if [[ -z "${ready}" ]]
    then
      error "the packages' depends and includes lines form a cycle among: ${names[*]}"
      return 1
    fi
    placed[${ready}]=1
    placed_count=$(( placed_count + 1 ))
    order+=("${ready}")
  done
  # Nothing is printed until the order is whole, so a reader that loses the status still reads no partial set.
  printf '%s\n' "${order[@]}"
}

# resolve:dependents <root> <package> <active>… prints the active packages that depend on <package>.
resolve:dependents() {
  local root=$1 package=$2 other dep
  local -a depends
  shift 2
  for other in "$@"
  do
    resolve:field depends "${root}" "${other}" depends || true  # a read that fails has said so
    for dep in "${depends[@]}"
    do
      [[ "${dep}" == "${package}" ]] && printf '%s\n' "${other}"
    done
  done
  return 0
}

# resolve:includers <root> <package> <active>… prints the active packages that include <package>.
resolve:includers() {
  local root=$1 package=$2 other member
  local -a includes
  shift 2
  for other in "$@"
  do
    resolve:field includes "${root}" "${other}" includes || true  # a read that fails has said so
    for member in "${includes[@]}"
    do
      [[ "${member}" == "${package}" ]] && printf '%s\n' "${other}"
    done
  done
  return 0
}

# resolve:exclusions <excluded> <active>… prints the packages of <excluded>, a space-separated list, that are not
# active: sorted, each once, one per line.
resolve:exclusions() {
  local excluded=$1 package
  shift
  for package in ${excluded}
  do
    list:among "${package}" "$@" || printf '%s\n' "${package}"
  done | LC_ALL=C sort --unique
}

# resolve:validate <root> <package> fails on metadata that the schema rejects, as far as bash can check without a
# validator:
#
#   - a key the kit does not know
#   - a missing description
#   - a default that is not true or false
#   - a dependency on, or a member that is, a package not in reach
#
# The gate validates against packages/package.schema.json in full.
resolve:validate() {
  local root=$1 package=$2 meta key value dep kind
  local -a words
  local -A says=([depends]='depends on' [includes]=includes)
  resolve:locate "${root}" "${package}" || return 1
  meta=${resolve_meta}
  resolve:read "${root}" "${package}" || return 1
  resolve:field words "${root}" "${package}" keys
  for key in "${words[@]}"
  do
    case "${key}" in
      (description | default | depends | includes | answers | expects | notices) ;;
      (*)
        error "${meta} has a key the kit does not know: ${key}" \
          "(see packages/package.schema.json)"
        return 1
    esac
  done
  if [[ -z "${resolve_fields[${root}|${package}|description]}" ]]
  then
    error "${meta} has no description"
    return 1
  fi
  # A default that is neither true nor false is read again, as it is written, to be named.
  if [[ "${resolve_fields[${root}|${package}|default]}" == other ]]
  then
    value=$(yaml:value "${meta}" default)
    if [[ -n "${value}" && "${value}" != true && "${value}" != false ]]
    then
      error "${meta}: default must be true or false, not '${value}'"
      return 1
    fi
  fi
  for kind in depends includes
  do
    resolve:field words "${root}" "${package}" "${kind}"
    for dep in "${words[@]}"
    do
      resolve:locate "${root}" "${dep}" || return 1
      if [[ ! -f "${resolve_meta}" ]]
      then
        error "${meta} ${says[${kind}]} '${dep}', which is not in reach"
        return 1
      fi
    done
  done
}

# resolve:expectations <root> <package>… prints "<path>\x1f<pattern>\x1f<remedy>" for each line the packages expect
# in a file the repository owns. Such a line is the include that pulls in the kit's file, such as lefthook's
# extends or just's import.
resolve:expectations() {
  local root=$1 package
  shift
  for package in "$@"
  do
    resolve:locate "${root}" "${package}" || return 1
    yaml:records "${resolve_meta}" expects path pattern remedy || return 1
  done
}

# resolve:unmet <root> <package>… prints the remedy for each expectation the repository does not meet, one per
# line. An expectation is unmet when its file is missing, or when no line in the file matches its pattern (an
# extended regular expression). The kit never writes to such a file; it only reports what is missing. It returns 1
# when the expectations cannot be read.
resolve:unmet() {
  local root=$1 path pattern remedy row
  local -a rows
  shift
  list:read rows resolve:expectations "${root}" "$@" || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\x1f' read -r path pattern remedy <<<"${row}"
    [[ -n "${path}" ]] || continue
    if [[ ! -f "${path}" ]] || ! grep --quiet --extended-regexp -- "${pattern}" "${path}"
    then
      printf '%s: %s\n' "${path}" "${remedy:-a line matching ${pattern} is expected}"
    fi
  done
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
