# shellcheck shell=bash
# package.sh - the active set of packages: add, remove, or select it whole, as kit packages does (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# kit packages add and kit packages remove move the active set one way each; kit packages select names the set
# itself, and reaches it by removing what the set omits and then adding what it lacks. Each function reads the
# declaration in kit.yml afresh, so one may follow another in a process. Each returns 2 on a usage error and 1 on a
# refusal, and lets the verb exit.

use release settings resolve render place materialize manifest

# package:named <name>… prints the packages the names stand for, a glob expanded to its matches, after checking that
# each is in reach. It returns 2 on a name that is not.
package:named() {
  local expanded
  local -a named
  expanded=$(resolve:expand "${kit_root}" "$@") || return 2
  readarray -t named < <(awk 'NF' <<<"${expanded}")
  resolve:require "${kit_root}" "${named[@]}" || return 2
  printf '%s\n' "${named[@]}"
}

# package:list [--active | --inactive] [--json] [<package>…] prints every package in reach, or those that match a
# name or a glob, one row each: a ✅ beside an active one and a ❌ beside an excluded one, its name, its origin (release
# or repository), and its description. --active keeps the active ones, --inactive the rest. --json prints one object
# per package instead: name, origin, state (active, excluded, or inactive), and description.
package:list() {
  local json=false state='' arg package mark origin width state_word
  local -a patterns=() active excluded shown=()
  for arg in "$@"
  do
    case "${arg}" in
      (--json) json=true ;;
      (--active) state=active ;;
      (--inactive) state=inactive ;;
      (*) patterns+=("${arg}")
    esac
  done
  readarray -t active < <(settings:packages)
  readarray -t excluded < <(settings:excludes)
  for package in $(resolve:reach)
  do
    if (( ${#patterns[@]} ))
    then
      package:matches "${package}" "${patterns[@]}" || continue
    fi
    if list:among "${package}" "${active[@]}"
    then
      [[ "${state}" != inactive ]] || continue
    else
      [[ "${state}" != active ]] || continue
    fi
    shown+=("${package}")
  done
  (( ${#shown[@]} )) || return 0
  width=$(printf '%s\n' "${shown[@]}" | awk 'length > w { w = length } END { print w }')
  for package in "${shown[@]}"
  do
    origin=$(resolve:origin "${kit_root}" "${package}")
    if list:among "${package}" "${active[@]}"
    then mark='✅' state_word=active
    elif list:among "${package}" "${excluded[@]}"
    then mark='❌' state_word=excluded
    else mark='  ' state_word=inactive
    fi
    if [[ "${json}" == true ]]
    then
      jq --null-input --compact-output --arg name "${package}" --arg origin "${origin}" --arg state "${state_word}" \
        --arg description "$(resolve:description "${kit_root}" "${package}")" \
        '{name: $name, origin: $origin, state: $state, description: $description}'
    else
      printf '%s %-*s %-10s %s\n' "${mark}" "${width}" "${package}" "${origin}" \
        "$(resolve:description "${kit_root}" "${package}")"
    fi
  done
}

# package:matches <package> <pattern>… is true when the package's name is one of the patterns, or matches one as a
# glob, as kit packages add expands 'style/*'.
package:matches() {
  local package=$1 pattern
  shift
  for pattern in "$@"
  do
    # shellcheck disable=SC2053  # the pattern is a glob on purpose
    [[ "${package}" == ${pattern} ]] && return 0
  done
  return 1
}

# package:add <package>… activates the packages, the packages they depend on, and the members they include, places
# the files they deliver, and rewrites the manifests. A package named here is no longer excluded. When a different
# file is already at a delivered path, it is named and left alone, no manifest is written, and it returns 1.
package:add() {
  local package base path refused=0 names
  local -a named active excluded wanted
  guard:require tool:jq || return 1
  names=$(package:named "$@") || return 2
  readarray -t named <<<"${names}"
  readarray -t active < <(settings:packages)
  readarray -t excluded < <(settings:excludes)
  # A package named here is wanted, whatever was recorded about it.
  for package in "${named[@]}"
  do
    readarray -t excluded < <(list:without "${package}" "${excluded[@]}")
  done
  readarray -t wanted < <(resolve:active "${kit_root}" "${excluded[*]}" "${active[@]}" "${named[@]}")
  render:all "${kit_root}" "${wanted[@]}"
  section "kit packages add ${named[*]}"
  for base in $(release:bases)
  do
    while IFS= read -r path
    do
      [[ -n "${path}" ]] || continue
      place:new "$(place:source-of "${kit_root}" "${base}" "${path}" "${wanted[@]}")" \
        "$(release:dest "${base}" "${path}")" "placed" || refused=$(( refused + 1 ))
    done < <(place:beyond "${kit_root}" "${base}" "${active[*]}" "${wanted[*]}")
  done
  if (( refused > 0 ))
  then
    error "${refused} path(s) could not be placed, each named above; settle each, then run:" \
      "kit packages add ${named[*]}"
    return 1
  fi
  readarray -t excluded < <(resolve:exclusions "${excluded[*]}" "${wanted[@]}")
  manifest:write-all "${excluded[*]}" "${wanted[@]}"
  materialize:all "${wanted[@]}"
  ok "packages: ${wanted[*]}"
  release:say-unmet "${wanted[@]}"
  release:say-notices "${wanted[@]}"
}

# package:remove <package>… takes out what the packages delivered (files, the kit's JSON entries, and the kit's
# regions) and rewrites the manifests. A package that an active package depends on is refused, and it returns 1. A
# member of a bundle that stays active is recorded as excluded, so that kit update does not bring it back.
package:remove() {
  local package base path newly='' names
  local -a named active excluded remaining dependents includers
  guard:require tool:jq || return 1
  names=$(package:named "$@") || return 2
  readarray -t named <<<"${names}"
  readarray -t active < <(settings:packages)
  readarray -t excluded < <(settings:excludes)
  remaining=("${active[@]}")
  for package in "${named[@]}"
  do
    list:among "${package}" "${active[@]}" || step "${package}: not active"
    readarray -t remaining < <(list:without "${package}" "${remaining[@]}")
  done
  # Dependents are checked against the packages that remain, so the order of the named packages does not matter.
  for package in "${named[@]}"
  do
    readarray -t dependents < <(resolve:dependents "${kit_root}" "${package}" "${remaining[@]}")
    if (( ${#dependents[@]} > 0 ))
    then
      error "${package} cannot be removed while ${dependents[*]} depend(s) on it"
      return 1
    fi
  done
  render:all "${kit_root}" "${active[@]}"
  section "kit packages remove ${named[*]}"
  for package in "${named[@]}"
  do
    readarray -t includers < <(resolve:includers "${kit_root}" "${package}" "${remaining[@]}")
    if (( ${#includers[@]} > 0 ))
    then
      newly+=" ${package}"
      step "${package}: excluded, so an update does not bring it back as a member of ${includers[*]}"
    fi
  done
  readarray -t excluded < <(resolve:exclusions "${excluded[*]}${newly}" "${remaining[@]}")
  for base in $(release:bases)
  do
    while IFS= read -r path
    do
      [[ -n "${path}" ]] || continue
      place:remove "$(place:source-of "${kit_root}" "${base}" "${path}" "${active[@]}")" \
        "$(release:dest "${base}" "${path}")"
    done < <(place:beyond "${kit_root}" "${base}" "${remaining[*]}" "${active[*]}")
  done
  manifest:write-all "${excluded[*]}" "${remaining[@]}"
  materialize:all "${remaining[@]}"
  ok "packages: ${remaining[*]:-none}"
}

# package:select <package>… makes exactly the named packages, with the packages they depend on and the members they
# include, the active set: it removes what the set omits, then adds what it lacks, and says which active package it
# kept for another's sake. A member removed earlier stays excluded unless it is named. With nothing to change it says
# the set and changes nothing.
package:select() {
  local package names
  local -a named active excluded wanted drop=() take=() dependents includers
  names=$(package:named "$@") || return 2
  readarray -t named <<<"${names}"
  readarray -t active < <(settings:packages)
  readarray -t excluded < <(settings:excludes)
  # A member removed earlier stays out of its bundle unless it is named here, as kit packages add keeps it out.
  for package in "${named[@]}"
  do
    readarray -t excluded < <(list:without "${package}" "${excluded[@]}")
  done
  readarray -t wanted < <(resolve:active "${kit_root}" "${excluded[*]}" "${named[@]}")
  for package in "${active[@]}"
  do
    list:among "${package}" "${wanted[@]}" || drop+=("${package}")
  done
  for package in "${wanted[@]}"
  do
    list:among "${package}" "${active[@]}" || take+=("${package}")
  done
  for package in "${active[@]}"
  do
    list:among "${package}" "${named[@]}" && continue
    list:among "${package}" "${wanted[@]}" || continue
    readarray -t dependents < <(resolve:dependents "${kit_root}" "${package}" "${wanted[@]}")
    readarray -t includers < <(resolve:includers "${kit_root}" "${package}" "${wanted[@]}")
    if (( ${#dependents[@]} > 0 ))
    then
      step "${package}: kept, ${dependents[*]} depend(s) on it"
    elif (( ${#includers[@]} > 0 ))
    then
      step "${package}: kept, a member of ${includers[*]}"
    fi
  done
  if (( ${#drop[@]} == 0 && ${#take[@]} == 0 ))
  then
    ok "packages: ${wanted[*]}"
    return 0
  fi
  if (( ${#drop[@]} > 0 ))
  then
    package:remove "${drop[@]}" || return $?
  fi
  if (( ${#take[@]} > 0 ))
  then
    package:add "${take[@]}" || return $?
  fi
}

# package:choices <key> [<answer>…] answers kit syntax --choices for <package>: every package in reach with its
# description and its state (active, excluded, or none); closed choices, and a set when the answers name select. When
# the answers name packages, as <package>=house, a package they omit but kit packages select would keep or bring, a
# dependency or a member of a named bundle, carries the line select prints about it.
package:choices() {
  local key=$1 answer package expanded state note word
  local -a active excluded named=() known=() wanted=() dependents includers
  shift
  [[ "${key}" == '<package>' ]] || return 1
  # kit packages select names the whole set, so its list is a set: the form starts from the active packages and marks
  # each row by what select will do; under add and remove the same packages are closed choices, picked one by one.
  if [[ "${1:-}" == --kind ]]
  then
    shift
    if list:among select "$@"
    then
      printf 'set\n'
    else
      printf 'closed\n'
    fi
    return 0
  fi
  for answer in "$@"
  do
    [[ "${answer}" == '<package>='* ]] && named+=("${answer#<package>=}")
  done
  # A name the answers give may be a glob, or not in reach yet; only the packages they stand for count.
  for answer in "${named[@]}"
  do
    expanded=$(resolve:expand "${kit_root}" "${answer}" 2>/dev/null) || continue
    for package in ${expanded}
    do
      [[ -f "$(resolve:meta "${kit_root}" "${package}")" ]] && known+=("${package}")
    done
  done
  readarray -t active < <(settings:packages)
  readarray -t excluded < <(settings:excludes)
  if (( ${#known[@]} > 0 ))
  then
    local -a kept=("${excluded[@]}")
    for package in "${known[@]}"
    do
      readarray -t kept < <(list:without "${package}" "${kept[@]}")
    done
    readarray -t wanted < <(resolve:active "${kit_root}" "${kept[*]}" "${known[@]}")
  fi
  for package in $(resolve:reach)
  do
    state='' note=''
    if list:among "${package}" "${active[@]}"
    then
      state=active
    elif list:among "${package}" "${excluded[@]}"
    then
      state=excluded
    fi
    if (( ${#known[@]} > 0 )) && ! list:among "${package}" "${known[@]}" && list:among "${package}" "${wanted[@]}"
    then
      word='comes along'
      [[ "${state}" != active ]] || word=kept
      readarray -t dependents < <(resolve:dependents "${kit_root}" "${package}" "${wanted[@]}")
      readarray -t includers < <(resolve:includers "${kit_root}" "${package}" "${wanted[@]}")
      if (( ${#dependents[@]} > 0 ))
      then
        note="${package}: ${word}, ${dependents[*]} depend(s) on it"
      elif (( ${#includers[@]} > 0 ))
      then
        note="${package}: ${word}, a member of ${includers[*]}"
      fi
    fi
    printf '%s\t%s\t%s\t%s\n' "${package}" "$(resolve:description "${kit_root}" "${package}")" "${state}" "${note}"
  done
}

# ----------------------------------------------------------------------------------------------------------------------
