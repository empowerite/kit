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
  resolve:roots
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
  shift
  for name in "$@"
  do
    if [[ "${name}" != *[*?[]* ]]
    then
      printf '%s\n' "${name}"
      continue
    fi
    matched=0
    while IFS= read -r package
    do
      # shellcheck disable=SC2053  # the glob is the point
      if [[ "${package}" == ${name} ]]
      then
        printf '%s\n' "${package}"
        matched=1
      fi
    done < <(resolve:reach "${root}")
    if (( matched == 0 ))
    then
      error "'${name}' matches no package in reach; the packages are: $(resolve:reach "${root}" | paste -s -d ' ' -)"
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
  local name dir
  sources:fetch-all || true
  while IFS=$'\t' read -r name dir
  do
    [[ -n "${name}" ]] || continue
    resolve_source_names+=("${name}")
    resolve_source_dirs+=("${dir}")
  done < <(sources:roots)
  resolve_sources_read=true
}

# resolve:home <root> <package> prints the directory that contains the package: the last source that has it, else
# <root> when the release has it, else the repository's .kit. The package's files are under
# <home>/packages/<package>/.
resolve:home() {
  local i
  resolve:roots
  for (( i = ${#resolve_source_dirs[@]} - 1; i >= 0; i-- ))
  do
    if [[ -f "${resolve_source_dirs[i]}/packages/$2/package.yml" ]]
    then
      printf '%s\n' "${resolve_source_dirs[i]}"
      return 0
    fi
  done
  if [[ -f "$1/packages/$2/package.yml" || ! -f "${resolve_local}/packages/$2/package.yml" ]]
  then printf '%s\n' "$1"
  else printf '%s\n' "${resolve_local}"
  fi
}

# resolve:origin <root> <package> prints where the package comes from: "release", "repository", or the name of the
# source it comes from.
resolve:origin() {
  local home i
  resolve:roots
  home=$(resolve:home "$1" "$2")
  for i in "${!resolve_source_dirs[@]}"
  do
    if [[ "${home}" == "${resolve_source_dirs[i]}" ]]
    then
      printf '%s\n' "${resolve_source_names[i]}"
      return 0
    fi
  done
  if [[ "${home}" == "$1" ]]
  then printf 'release\n'
  else printf 'repository\n'
  fi
}

# resolve:meta <root> <package> prints the path of the package's metadata.
resolve:meta() { printf '%s/packages/%s/package.yml\n' "$(resolve:home "$1" "$2")" "$2"; }

# resolve:description <root> <package> prints the package's one-line description.
resolve:description() { yaml:value "$(resolve:meta "$1" "$2")" description; }

# resolve:depends <root> <package> prints the packages this one depends on, one per line.
resolve:depends() { yaml:list "$(resolve:meta "$1" "$2")" depends; }

# resolve:includes <root> <package> prints the members of this bundle, one per line: packages activated with it, each
# of which can be removed on its own.
resolve:includes() { yaml:list "$(resolve:meta "$1" "$2")" includes; }

# resolve:defaults <root> prints the packages whose default is true. They are active unless removed.
resolve:defaults() {
  local package
  for package in $(resolve:reach "$1")
  do
    [[ "$(yaml:value "$(resolve:meta "$1" "${package}")" default)" == true ]] && printf '%s\n' "${package}"
  done
  return 0
}

# resolve:require <root> <name>… fails on a package that is not in reach, and lists the packages that are.
resolve:require() {
  local root=$1 package
  shift
  for package in "$@"
  do
    if [[ ! -f "$(resolve:meta "${root}" "${package}")" ]]
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
  local -a queue=("$@") names=()
  while (( ${#queue[@]} > 0 ))
  do
    package=${queue[0]}
    queue=("${queue[@]:1}")
    [[ -n "${wanted[${package}]:-}" ]] && continue
    wanted[${package}]=1
    for dep in $(resolve:depends "${root}" "${package}")
    do
      queue+=("${dep}")
    done
    for dep in $(resolve:includes "${root}" "${package}")
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
      for dep in $(resolve:depends "${root}" "${package}") $(resolve:includes "${root}" "${package}")
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
    printf '%s\n' "${ready}"
  done
}

# resolve:dependents <root> <package> <active>… prints the active packages that depend on <package>.
resolve:dependents() {
  local root=$1 package=$2 other dep
  shift 2
  for other in "$@"
  do
    for dep in $(resolve:depends "${root}" "${other}")
    do
      [[ "${dep}" == "${package}" ]] && printf '%s\n' "${other}"
    done
  done
  return 0
}

# resolve:includers <root> <package> <active>… prints the active packages that include <package>.
resolve:includers() {
  local root=$1 package=$2 other member
  shift 2
  for other in "$@"
  do
    for member in $(resolve:includes "${root}" "${other}")
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
  local root=$1 package=$2 meta key value dep
  meta=$(resolve:meta "${root}" "${package}")
  for key in $(yaml:keys "${meta}")
  do
    case "${key}" in
      (description | default | depends | includes | answers | expects | notices) ;;
      (*)
        error "$(resolve:meta "${root}" "${package}") has a key the kit does not know: ${key}" \
          "(see packages/package.schema.json)"
        return 1
    esac
  done
  if [[ -z "$(yaml:value "${meta}" description)" ]]
  then
    error "$(resolve:meta "${root}" "${package}") has no description"
    return 1
  fi
  value=$(yaml:value "${meta}" default)
  if [[ -n "${value}" && "${value}" != true && "${value}" != false ]]
  then
    error "$(resolve:meta "${root}" "${package}"): default must be true or false, not '${value}'"
    return 1
  fi
  for dep in $(resolve:depends "${root}" "${package}")
  do
    if [[ ! -f "$(resolve:meta "${root}" "${dep}")" ]]
    then
      error "$(resolve:meta "${root}" "${package}") depends on '${dep}', which is not in reach"
      return 1
    fi
  done
  for dep in $(resolve:includes "${root}" "${package}")
  do
    if [[ ! -f "$(resolve:meta "${root}" "${dep}")" ]]
    then
      error "$(resolve:meta "${root}" "${package}") includes '${dep}', which is not in reach"
      return 1
    fi
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
    yaml:records "$(resolve:meta "${root}" "${package}")" expects path pattern remedy
  done
}

# resolve:unmet <root> <package>… prints the remedy for each expectation the repository does not meet, one per
# line. An expectation is unmet when its file is missing, or when no line in the file matches its pattern (an
# extended regular expression). The kit never writes to such a file; it only reports what is missing.
resolve:unmet() {
  local root=$1 path pattern remedy
  shift
  while IFS=$'\x1f' read -r path pattern remedy
  do
    [[ -n "${path}" ]] || continue
    if [[ ! -f "${path}" ]] || ! grep --quiet --extended-regexp -- "${pattern}" "${path}"
    then
      printf '%s: %s\n' "${path}" "${remedy:-a line matching ${pattern} is expected}"
    fi
  done < <(resolve:expectations "${root}" "$@")
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
