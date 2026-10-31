# shellcheck shell=bash
# package.sh - the active set of packages: add to it, remove from it, or select it whole, as kit package does (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# kit package add and kit package remove move the active set one way each; kit package select names the set itself,
# and reaches it by removing what the set omits and then adding what it lacks. Each function reads the declaration in
# .kit/settings.yml afresh, so one may follow another in a process. Each returns 2 on a usage error and 1 on a
# refusal, and lets the verb exit.

use release

# package:named <name>… prints the packages the names stand for, a glob expanded to its matches, after checking that
# each is in reach. It returns 2 on a name that is not.
package:named() {
  local expanded
  local -a named
  expanded=$(release:expand "${kit_root}" "$@") || return 2
  readarray -t named < <(awk 'NF' <<<"${expanded}")
  release:require-packages "${kit_root}" "${named[@]}" || return 2
  printf '%s\n' "${named[@]}"
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
  readarray -t active < <(release:declared-packages)
  readarray -t excluded < <(release:declared-excludes)
  # A package named here is wanted, whatever was recorded about it.
  for package in "${named[@]}"
  do
    readarray -t excluded < <(release:without "${package}" "${excluded[@]}")
  done
  readarray -t wanted < <(release:resolve "${kit_root}" "${excluded[*]}" "${active[@]}" "${named[@]}")
  release:render-all "${kit_root}" "${wanted[@]}"
  section "kit package add ${named[*]}"
  for base in $(release:bases)
  do
    while IFS= read -r path
    do
      [[ -n "${path}" ]] || continue
      release:place-new "$(release:source-of "${kit_root}" "${base}" "${path}" "${wanted[@]}")" \
        "$(release:dest "${base}" "${path}")" "placed" || refused=$(( refused + 1 ))
    done < <(release:paths-beyond "${kit_root}" "${base}" "${active[*]}" "${wanted[*]}")
  done
  if (( refused > 0 ))
  then
    error "${refused} path(s) already hold a different file; move each aside, then run: kit package add ${named[*]}"
    return 1
  fi
  readarray -t excluded < <(release:exclusions "${excluded[*]}" "${wanted[@]}")
  release:write-manifests "${excluded[*]}" "${wanted[@]}"
  release:materialize "${wanted[@]}"
  ok "packages: ${wanted[*]}"
  release:say-unmet "${wanted[@]}"
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
  readarray -t active < <(release:declared-packages)
  readarray -t excluded < <(release:declared-excludes)
  remaining=("${active[@]}")
  for package in "${named[@]}"
  do
    release:among "${package}" "${active[@]}" || step "${package}: not active"
    readarray -t remaining < <(release:without "${package}" "${remaining[@]}")
  done
  # Dependents are checked against the packages that remain, so the order of the named packages does not matter.
  for package in "${named[@]}"
  do
    readarray -t dependents < <(release:dependents "${kit_root}" "${package}" "${remaining[@]}")
    if (( ${#dependents[@]} > 0 ))
    then
      error "${package} cannot be removed while ${dependents[*]} depend(s) on it"
      return 1
    fi
  done
  release:render-all "${kit_root}" "${active[@]}"
  section "kit package remove ${named[*]}"
  for package in "${named[@]}"
  do
    readarray -t includers < <(release:includers "${kit_root}" "${package}" "${remaining[@]}")
    if (( ${#includers[@]} > 0 ))
    then
      newly+=" ${package}"
      step "${package}: excluded, so an update does not bring it back as a member of ${includers[*]}"
    fi
  done
  readarray -t excluded < <(release:exclusions "${excluded[*]}${newly}" "${remaining[@]}")
  for base in $(release:bases)
  do
    while IFS= read -r path
    do
      [[ -n "${path}" ]] || continue
      release:unplace "$(release:source-of "${kit_root}" "${base}" "${path}" "${active[@]}")" \
        "$(release:dest "${base}" "${path}")"
    done < <(release:paths-beyond "${kit_root}" "${base}" "${remaining[*]}" "${active[*]}")
  done
  release:write-manifests "${excluded[*]}" "${remaining[@]}"
  release:materialize "${remaining[@]}"
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
  readarray -t active < <(release:declared-packages)
  readarray -t excluded < <(release:declared-excludes)
  # A member removed earlier stays out of its bundle unless it is named here, as kit package add keeps it out.
  for package in "${named[@]}"
  do
    readarray -t excluded < <(release:without "${package}" "${excluded[@]}")
  done
  readarray -t wanted < <(release:resolve "${kit_root}" "${excluded[*]}" "${named[@]}")
  for package in "${active[@]}"
  do
    release:among "${package}" "${wanted[@]}" || drop+=("${package}")
  done
  for package in "${wanted[@]}"
  do
    release:among "${package}" "${active[@]}" || take+=("${package}")
  done
  for package in "${active[@]}"
  do
    release:among "${package}" "${named[@]}" && continue
    release:among "${package}" "${wanted[@]}" || continue
    readarray -t dependents < <(release:dependents "${kit_root}" "${package}" "${wanted[@]}")
    readarray -t includers < <(release:includers "${kit_root}" "${package}" "${wanted[@]}")
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

# package:choices <key> [<answer>…] answers kit syntax --choices for <package>: a closed set, every package in reach
# with its description and its state (active, excluded, or none). When the answers name packages, as <package>=house,
# a package they omit but kit package select would keep or bring, a dependency or a member of a named bundle, carries
# the line select prints about it.
package:choices() {
  local key=$1 answer package expanded state note word
  local -a active excluded named=() known=() wanted=() dependents includers
  shift
  [[ "${key}" == '<package>' ]] || return 1
  if [[ "${1:-}" == --kind ]]
  then
    printf 'closed\n'
    return 0
  fi
  for answer in "$@"
  do
    [[ "${answer}" == '<package>='* ]] && named+=("${answer#<package>=}")
  done
  # A name the answers give may be a glob, or not in reach yet; only the packages they stand for count.
  for answer in "${named[@]}"
  do
    expanded=$(release:expand "${kit_root}" "${answer}" 2>/dev/null) || continue
    for package in ${expanded}
    do
      [[ -f "$(release:meta "${kit_root}" "${package}")" ]] && known+=("${package}")
    done
  done
  readarray -t active < <(release:declared-packages)
  readarray -t excluded < <(release:declared-excludes)
  if (( ${#known[@]} > 0 ))
  then
    local -a kept=("${excluded[@]}")
    for package in "${known[@]}"
    do
      readarray -t kept < <(release:without "${package}" "${kept[@]}")
    done
    readarray -t wanted < <(release:resolve "${kit_root}" "${kept[*]}" "${known[@]}")
  fi
  for package in $(release:packages)
  do
    state='' note=''
    if release:among "${package}" "${active[@]}"
    then
      state=active
    elif release:among "${package}" "${excluded[@]}"
    then
      state=excluded
    fi
    if (( ${#known[@]} > 0 )) && ! release:among "${package}" "${known[@]}" && release:among "${package}" "${wanted[@]}"
    then
      word='comes along'
      [[ "${state}" != active ]] || word=kept
      readarray -t dependents < <(release:dependents "${kit_root}" "${package}" "${wanted[@]}")
      readarray -t includers < <(release:includers "${kit_root}" "${package}" "${wanted[@]}")
      if (( ${#dependents[@]} > 0 ))
      then
        note="${package}: ${word}, ${dependents[*]} depend(s) on it"
      elif (( ${#includers[@]} > 0 ))
      then
        note="${package}: ${word}, a member of ${includers[*]}"
      fi
    fi
    printf '%s\t%s\t%s\t%s\n' "${package}" "$(release:description "${kit_root}" "${package}")" "${state}" "${note}"
  done
}

# ----------------------------------------------------------------------------------------------------------------------
