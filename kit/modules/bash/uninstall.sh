# shellcheck shell=bash
# uninstall.sh - takes everything the active packages delivered out of the repository and the home directory (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# kit uninstall is kit packages remove over every active package, the kit package included, followed by the manifests
# and the declaration. Each delivered path is taken out as place:remove takes it out for a removed package: a copy
# is removed, a merged .json file keeps everything but the kit's entries, a region file keeps everything but the
# region. A copy the repository edited is the one path left alone, and named, as kit update leaves one.
#
# see: bin/kit-uninstall, modules/bash/place.sh, modules/bash/settings.sh

use release file settings render place materialize manifest resolve

# uninstall:run [--dry-run] takes out what every active package delivered, under the repository and under home, then
# the manifests, then the declaration. With --dry-run it prints what it would do and changes nothing.
uninstall:run() {
  local dry=false base path src
  local -a packages
  [[ "${1:-}" != --dry-run ]] || dry=true
  readarray -t packages < <(settings:packages)
  render:all "${kit_root}" "${packages[@]}"
  if [[ "${dry}" == true ]]
  then section "kit uninstall --dry-run: what would be taken out"
  else section "kit uninstall, packages: ${packages[*]}"
  fi
  for base in $(release:bases)
  do
    while IFS=$'\t' read -r path src
    do
      [[ -n "${path}" ]] || continue
      uninstall:path "${dry}" "${base}" "${path}" "${src}"
    done < <(place:delivered "${kit_root}" "${base}" "${packages[@]}")
  done
  uninstall:manifests "${dry}"
  uninstall:declaration "${dry}"
  uninstall:left "${packages[@]}"
  [[ "${dry}" == true ]] || ok "kit is uninstalled; commit what changed. kit install takes it again."
}

# uninstall:left <package>… names what the kit leaves for the person, since the kit never writes it: each line a
# package expected in a file the repository owns, and a pre-commit hook that still runs lefthook.
uninstall:left() {
  local path pattern _
  while IFS=$'\x1f' read -r path pattern _
  do
    [[ -n "${path}" && -f "${path}" ]] || continue
    grep --quiet --extended-regexp -- "${pattern}" "${path}" || continue
    warn "left to do in ${path}: take out the line that names ${pattern//\\/}"
  done < <(resolve:expectations "${kit_root}" "$@")
  if list:among hooks/lefthook "$@" && grep --quiet lefthook "$(git rev-parse --git-path hooks/pre-commit)" 2>/dev/null
  then
    warn "left to do: the pre-commit hook still runs lefthook; run: lefthook uninstall"
  fi
}

# uninstall:path <dry> <base> <path> <source> takes one delivered path out, or says what it would do. A copy outside
# .kit/ that differs from the release's file was edited by the repository: it is named and left as it is.
uninstall:path() {
  local dry=$1 base=$2 path=$3 src=$4 dest kind
  dest=$(release:dest "${base}" "${path}")
  [[ -e "${dest}" || -L "${dest}" ]] || return 0
  kind=$(place:kind "${src}" "${dest}")
  if [[ "${kind}" == copy ]] && ! materialize:owned "${base}" "${path}" && ! cmp --silent "${src}" "${dest}"
  then
    warn "${dest}: edited since the kit placed it; it is left as it is"
    return 0
  fi
  if [[ "${dry}" == false ]]
  then
    place:remove "${src}" "${dest}"
    # A merged file holding nothing once the kit's entries are out held only the kit's, so it goes as a copy goes.
    if [[ "${kind}" == merge && -f "${dest}" ]] && jq --exit-status 'length == 0' "${dest}" >/dev/null 2>&1
    then
      rm -f "${dest}"
      step "${dest}: removed; only the kit's entries were in it"
    fi
    [[ -e "${dest}" ]] || uninstall:prune "${dest}"
    return 0
  fi
  case "${kind}" in
    (merge) step "${dest}: the kit's entries would be taken out" ;;
    (region) step "${dest}: the kit's region would be taken out" ;;
    (*) step "${dest}: would be removed"
  esac
}

# uninstall:prune <path> removes the directories a removed path leaves empty, up to the base's root, which stays.
uninstall:prune() {
  local dir
  dir=$(dirname "$1")
  while [[ "${dir}" != . && "${dir}" != "${KIT_HOME:-${HOME}}" && -d "${dir}" ]] \
    && [[ -z "$(find "${dir}" -mindepth 1 -maxdepth 1 | head -n 1)" ]]
  do
    rmdir "${dir}"
    dir=$(dirname "${dir}")
  done
}

# uninstall:manifests <dry> removes the repository's manifest, .kit/.gitignore, what kit cached in .kit/cache/, and
# its logs in .kit/logs/.
# It also removes the project toolchain choice made here, kept in the git directory. It removes .kit/ itself when
# nothing is left in it. It writes home's manifest without this repository's paths, which removes it when no line is
# left.
uninstall:manifests() {
  local dry=$1 choice
  choice=$(uninstall:choice)
  if [[ "${dry}" == true ]]
  then
    [[ ! -e .kit/manifest ]] || step ".kit/manifest and .kit/.gitignore: would be removed"
    [[ ! -e .kit/cache ]] || step ".kit/cache/: would be removed"
    [[ ! -e .kit/logs ]] || step ".kit/logs/: would be removed"
    [[ -z "${choice}" ]] || step "${choice#"${PWD}/"}: would be removed"
    return 0
  fi
  rm -f .kit/manifest .kit/.gitignore
  rm -rf .kit/cache .kit/logs
  if [[ -n "${choice}" ]]
  then
    rm -f "${choice}"
    rmdir "$(dirname "${choice}")" 2>/dev/null || true
  fi
  if [[ -d .kit && -z "$(find .kit -mindepth 1 -maxdepth 1 | head -n 1)" ]]
  then
    rmdir .kit
    step ".kit/: removed, nothing was left in it"
  elif [[ -d .kit ]]
  then
    step ".kit/: kept, with the repository's own $(find .kit -mindepth 1 -maxdepth 1 | sed 's@^\.kit/@@' | paste -s -d ' ' -)"
  fi
  manifest:write home ''
}

# uninstall:choice prints the file that keeps the project toolchain choice made here, when it is there, and nothing
# otherwise. At the desk, that is the clone's choice. In a linked worktree, it is that worktree's own; the clone's
# stays for the desk to remove.
uninstall:choice() {
  local kept
  use toolchain || return 0
  kept=$(toolchain:kept 2>/dev/null) || return 0
  [[ -f "${kept}" ]] || return 0
  printf '%s\n' "${kept}"
}

# uninstall:declaration <dry> drops version, packages, and excludes from kit.yml, and removes the file when nothing
# but the schema modeline is left.
uninstall:declaration() {
  local dry=$1 settings kept
  settings=$(settings:file)
  [[ -f "${settings}" ]] || return 0
  # Runs of blank lines where the blocks stood become one, and none opens or closes the file.
  kept=$(settings:declaration-kept "${settings}" '' | awk 'NF { if (held && printed) print ""; held = 0; print; printed = 1; next } { held = 1 }')
  if ! grep --quiet --invert-match '^# yaml-language-server:' <<<"${kept}"
  then
    if [[ "${dry}" == true ]]
    then step "${settings}: would be removed, nothing else is in it"
    else rm -f "${settings}" && step "${settings}: removed, nothing else was in it"
    fi
    return 0
  fi
  if [[ "${dry}" == true ]]
  then
    step "${settings}: version and packages would be dropped"
  else
    file:replace "${settings}" printf '%s\n' "${kept}"
    step "${settings}: version and packages dropped"
  fi
}

# ----------------------------------------------------------------------------------------------------------------------
