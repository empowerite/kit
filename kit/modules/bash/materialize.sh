# shellcheck shell=bash
# materialize.sh - what kit keeps under .kit/: a link or a read-only copy per path, and .kit/.gitignore (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_root is exported by modules/bash/1st.sh
#
# Every path the kit places under .kit/, other than the repository's own packages, is a link to its source when the
# source is inside the worktree, and otherwise a copy without the write bit. .kit/.gitignore names every one, so
# git never sees them: the repository commits the declaration in kit.yml and its own packages, and nothing kit
# places. materialize:hydrate brings it all to that state from the declaration alone, and every verb that reads a
# delivered file calls it first.
#
# see: docs/packages.md

use release place manifest settings render file

# materialize:owned <base> <path> succeeds for a path the kit materializes: one under .kit/ in the repository, other than
# the repository's own packages. Such a path is a link to its source when the source is inside the worktree, and
# otherwise a copy without the write bit. .kit/.gitignore names every one, so git never sees them: the repository
# commits the declaration in kit.yml and its own packages under .kit/packages/, and nothing kit places.
materialize:owned() { [[ "$1" == repo && "$2" == .kit/* && "$2" != .kit/packages/* ]]; }

# materialize:within <real-path> succeeds when an absolute real path is inside the worktree: under the directory the verb
# runs in, the repository root, and not under a nested checkout such as a worktree under worktrees/, which has a .git
# of its own and goes away with kit tidy. This repository's release, kit/, is within. A release that mise or brew
# installed for a consumer is not, and neither is a rendered template, written under a temporary directory. A link
# never leaves the worktree: the worktree is what gets bind-mounted, copied, or synced as a unit, and a link that
# left it would dangle there, and again when the tool manager moved the release.
materialize:within() {
  local dir
  [[ -n "${materialize_root:-}" ]] || materialize_root=$(pwd -P)
  [[ "$1" == "${materialize_root}"/* ]] || return 1
  dir=${1%/*}
  while [[ "${dir}" == "${materialize_root}"/* ]]
  do
    [[ ! -e "${dir}/.git" ]] || return 1
    dir=${dir%/*}
  done
}

# materialize:relative <from-dir> <to> prints <to> as a path relative to <from-dir>. Both are absolute real paths.
materialize:relative() {
  local from=$1 to=$2 up=''
  while [[ "${to}" != "${from}"/* && "${from}" != / ]]
  do
    from=$(dirname "${from}")
    up+='../'
  done
  printf '%s%s\n' "${up}" "${to#"${from}"/}"
}

# materialize:state <source> <dest> prints the state the kit keeps a materialized path in: `link <target>`, the relative
# link to a source inside the worktree, or `copy` for a source outside it.
materialize:state() {
  local real
  real=$(readlink -f "$1")
  if materialize:within "${real}"
  then printf 'link %s\n' "$(materialize:relative "${materialize_root}/${2%/*}" "${real}")"
  else printf 'copy\n'
  fi
}

# materialize:ok <source> <dest> [<state>] succeeds when the destination is in its state (materialize:state's,
# computed here when not given): the link to the source, or a regular file with the source's content. The write bit
# is not judged here; materialize:one takes it back off a copy without a word.
materialize:ok() {
  local state=${3:-}
  [[ -n "${state}" ]] || state=$(materialize:state "$1" "$2")
  if [[ "${state}" == link* ]]
  then [[ -L "$2" && "$(readlink "$2")" == "${state#link }" ]]
  else [[ ! -L "$2" && -f "$2" ]] && cmp --silent "$1" "$2"
  fi
}

# materialize:one <source> <dest> brings a materialized path to its state and prints what it did: kept,
# placed where nothing was there, or replaced where something else was.
materialize:one() {
  local src=$1 dest=$2 state
  state=$(materialize:state "${src}" "${dest}")
  if materialize:ok "${src}" "${dest}" "${state}"
  then
    if [[ "${state}" == copy && -w "${dest}" ]]
    then
      chmod a-w "${dest}"
    fi
    printf 'kept\n'
    return 0
  fi
  if [[ -e "${dest}" || -L "${dest}" ]]
  then printf 'replaced\n'
  else printf 'placed\n'
  fi
  mkdir -p "$(dirname "${dest}")"
  if [[ "${state}" == link* ]]
  then
    ln -sfn "${state#link }" "${dest}"
  else
    file:copy --read-only "${src}" "${dest}"
  fi
}

# materialize:prune <path> removes the directories a removed path leaves empty, up to .kit/ itself, which stays.
materialize:prune() {
  local dir
  dir=$(dirname "$1")
  while [[ "${dir}" == .kit/* && -d "${dir}" && -z "$(find "${dir}" -mindepth 1 -maxdepth 1 | head -n 1)" ]]
  do
    rmdir "${dir}"
    dir=$(dirname "${dir}")
  done
}

# materialize:ignore-text <path>… prints .kit/.gitignore: each path kit placed under .kit/, the manifest, and the file
# itself, so git never sees them.
materialize:ignore-text() {
  local path
  printf '# .kit/.gitignore - what kit placed here, which git never sees (written by kit, not by hand)\n'
  {
    printf '/.gitignore\n/manifest\n'
    for path in "$@"
    do
      printf '/%s\n' "${path#.kit/}"
    done
  } | LC_ALL=C sort
}

# materialize:all [--check] <package>… brings every path the packages deliver under .kit/ to its state and writes
# .kit/.gitignore. It says what it changed, one line per path. A path the manifest never recorded that holds a
# different file is the repository's: it is left alone and named, as kit install names it. It warns about a file
# under a delivered directory that no active package delivers, and about a delivered path git still tracks, both of
# which a person settles; an empty directory left behind by a removed package is removed without a word. With
# --check it changes nothing: each of those is an error, and it prints how many on stdout.
materialize:all() {
  local check=false path src dir entry state did ignore findings=0
  local -a owned=() tracked=()
  local -A source=() within=()
  if [[ "${1:-}" == --check ]]
  then
    check=true
    shift
  fi
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    materialize:owned repo "${path}" || continue
    [[ "$(place:kind "${src}" "${path}")" == copy ]] || continue
    owned+=("${path}")
    source[${path}]=${src}
    dir=$(dirname "${path}")
    while [[ "${dir}" != .kit ]]
    do
      within[${dir}]=1
      dir=$(dirname "${dir}")
    done
  done < <(place:delivered "${kit_root}" repo "$@")
  # A path the manifest records that no active package delivers any more, as after a pulled declaration dropped a
  # package: removed, and the directories it leaves empty with it.
  while IFS= read -r path
  do
    [[ -n "${path}" ]] || continue
    materialize:owned repo "${path}" || continue
    [[ -z "${source[${path}]+x}" ]] || continue
    [[ -e "${path}" || -L "${path}" ]] || continue
    if [[ "${check}" == true ]]
    then
      error "${path} is no longer delivered by any active package; run: kit hydrate"
      findings=$(( findings + 1 ))
    else
      step "${path}: removed, no active package delivers it"
      rm -f "${path}"
      materialize:prune "${path}"
    fi
  done < <(manifest:paths repo)
  for path in "${owned[@]}"
  do
    src=${source[${path}]}
    if [[ "${check}" == false ]]
    then
      if [[ ( -e "${path}" || -L "${path}" ) && -z "$(manifest:recorded repo "${path}")" ]] \
        && ! materialize:ok "${src}" "${path}"
      then
        warn "${path}: a different file is there already; move it aside, or remove it if it is not yours"
        continue
      fi
      did=$(materialize:one "${src}" "${path}")
      case "${did}" in
        (placed) step "${path}: placed" ;;
        (replaced) warn "${path} was changed in place, or came from another release; replaced"
      esac
    elif [[ ! -e "${path}" && ! -L "${path}" ]]
    then
      error "${path} is missing; run: kit hydrate"
      findings=$(( findings + 1 ))
    elif ! materialize:ok "${src}" "${path}"
    then
      state=$(materialize:state "${src}" "${path}")
      if [[ "${state}" == link* ]]
      then error "${path} is not a link to ${state#link }; run: kit hydrate"
      elif [[ -z "$(manifest:recorded repo "${path}")" ]]
      then error "${path} holds a different file; move it aside, or remove it if it is not yours, then run: kit hydrate"
      else error "${path} was changed in place; run: kit hydrate (a delivered file changes by a new release, never in place)"
      fi
      findings=$(( findings + 1 ))
    fi
  done
  for dir in "${!within[@]}"
  do
    [[ -d "${dir}" ]] || continue
    while IFS= read -r entry
    do
      [[ -n "${entry}" && "${entry##*/}" != .* ]] || continue
      [[ -z "${source[${entry}]+x}" && -z "${within[${entry}]+x}" ]] || continue
      if [[ -d "${entry}" && -z "$(find "${entry}" -mindepth 1 -maxdepth 1 | head -n 1)" ]]
      then
        [[ "${check}" == true ]] || rmdir "${entry}"
        continue
      fi
      if [[ "${check}" == true ]]
      then
        error "${entry} is not delivered by any active package; add it through a package under .kit/packages/, or remove it"
        findings=$(( findings + 1 ))
      else
        warn "${entry} is not delivered by any active package; add it through a package under .kit/packages/, or remove it"
      fi
    done < <(find "${dir}" -mindepth 1 -maxdepth 1 | LC_ALL=C sort)
  done
  if (( ${#owned[@]} > 0 )) && git rev-parse --is-inside-work-tree >/dev/null 2>&1
  then
    readarray -t tracked < <(git ls-files -- "${owned[@]}" .kit/manifest .kit/.gitignore)
    for path in "${tracked[@]}"
    do
      [[ -n "${path}" ]] || continue
      if [[ "${check}" == true ]]
      then
        error "${path} is tracked by git; run: git rm --cached '${path}', and commit, so the kit's file never reaches a commit"
        findings=$(( findings + 1 ))
      else
        warn "${path} is tracked by git; run: git rm --cached '${path}', and commit, so the kit's file never reaches a commit"
      fi
    done
  fi
  ignore=$(materialize:ignore-text "${owned[@]}")
  if [[ "${check}" == true ]]
  then
    if [[ ! -f .kit/.gitignore || "$(<.kit/.gitignore)" != "${ignore}" ]]
    then
      error ".kit/.gitignore is not what kit writes; run: kit hydrate"
      findings=$(( findings + 1 ))
    fi
    printf '%s\n' "${findings}"
  elif [[ ! -f .kit/.gitignore || "$(<.kit/.gitignore)" != "${ignore}" ]]
  then
    # The file has no write bit, as every file kit places here has none.
    file:replace --read-only .kit/.gitignore printf '%s\n' "${ignore}"
    step ".kit/.gitignore: written"
  else
    # Newer than kit.yml again: materialize:hydrated reads that as the declaration having been seen.
    touch .kit/.gitignore
  fi
}

# materialize:hydrated succeeds when .kit/.gitignore exists and every path it names is there, as a link or without the
# write bit, and .kit/manifest exists. Where this release is itself inside the worktree, as the kit's own kit/ is,
# every path must be a link: a copy there came from a kit outside the worktree, and the full run turns it into a
# link. It starts one process, to resolve the release's root, so a reading verb asks on every run. A fresh clone, a
# `git clean -fdx`, a
# link an editor's save replaced, and a copy with its write bit back all fail it, and the full materialize:all
# follows, as it does when kit.yml is newer than .kit/.gitignore: a declaration edited by hand or pulled
# since the last full run, whose packages may have come or gone. A copy changed and put back without the write bit
# passes it: kit check judges that. A template rendered under .kit/ is a copy even inside, so a release that delivers
# one there takes the full run every time; none does.
materialize:hydrated() {
  local line path inside=false
  [[ -f .kit/.gitignore && -f .kit/manifest ]] || return 1
  [[ ! kit.yml -nt .kit/.gitignore ]] || return 1
  # Both are kit's to write, so both are without the write bit, except as root, where the bit says nothing.
  [[ ! -w .kit/.gitignore && ! -w .kit/manifest ]] || (( EUID == 0 )) || return 1
  ! materialize:within "$(readlink -f "${kit_root}")" || inside=true
  while IFS= read -r line
  do
    [[ "${line}" == /* && "${line}" != /.gitignore && "${line}" != /manifest ]] || continue
    path=.kit${line}
    if [[ "${inside}" == true ]]
    then
      [[ -L "${path}" && -e "${path}" ]] || return 1
    else
      [[ -e "${path}" ]] || return 1
      # As root every file is writable, so the write bit says nothing, and a regular file passes.
      [[ -L "${path}" || ! -w "${path}" ]] || (( EUID == 0 )) || return 1
    fi
  done < .kit/.gitignore
}

# materialize:hydrate [--full] places what the declared packages deliver under .kit/, as materialize:all does, in a
# repository that declares a kit version, and does nothing elsewhere. Every verb that reads a delivered file calls it
# first, so a fresh clone or worktree, a `git clean -fdx`, or a link an editor's save replaced heals on the next kit
# command. Where materialize:hydrated finds everything in place it does nothing more, so the common run costs nothing;
# with --full, as kit hydrate calls it, it always does the whole job and says what it finds. It warns when this
# release is not the declared one, since what it places is this release's. It rewrites .kit/manifest, so what the
# declaration added is recorded, and a later release's change to it is kit's to replace. Where the declaration still
# sits on the manifest's lines, from a release before it, it moves into kit.yml first.
materialize:hydrate() {
  local declared
  local -a packages
  declared=$(settings:version)
  [[ -n "${declared}" ]] || return 0
  if [[ -z "$(settings:value version)" ]]
  then
    # A manifest from a release before the declaration still carries it. It moves into kit.yml now, before
    # anything rewrites the manifest, which would drop its only copy.
    readarray -t packages < <(settings:packages)
    settings:declare "${declared}" "$(settings:excludes | paste -s -d ' ' -)" "${packages[@]}"
    step "kit.yml: the declaration moved here from .kit/manifest"
  fi
  if [[ "${1:-}" != --full ]] && materialize:hydrated
  then
    return 0
  fi
  if [[ "${declared}" != "$(release:version)" ]]
  then
    warn "kit.yml declares kit ${declared}, and this is kit $(release:version); what is placed is this" \
      "kit's; run: kit update"
  fi
  readarray -t packages < <(settings:packages)
  render:all "${kit_root}" "${packages[@]}" || return 1
  materialize:all "${packages[@]}"
  manifest:write repo '' "${packages[@]}"
}

# ----------------------------------------------------------------------------------------------------------------------
