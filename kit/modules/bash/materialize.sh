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

use release place manifest settings render file trust

# materialize_real maps a directory to its real path, for materialize:real: filled in the shell that asks, so every
# later path under a directory resolved once costs no process. materialize:all empties it first, so a long-running
# kit serve resolves each pass afresh.
declare -gA materialize_real=()

# materialize:owned <base> <path> succeeds for a path the kit materializes: one under .kit/ in the repository, with no
# . or .. segment, other than the repository's own packages. Such a path is a link to its source when the source is
# inside the worktree, and otherwise a copy without the write bit. .kit/.gitignore names every one, so git never sees
# them: the repository commits the declaration in kit.yml and its own packages under .kit/packages/, and nothing kit
# places.
materialize:owned() {
  [[ "$1" == repo && "$2" == .kit/* && "$2" != .kit/packages/* && "/$2/" != */../* && "/$2/" != */./* ]]
}

# materialize:linked <path> succeeds when .kit, or a directory between it and <path>, is a link: kit neither writes
# nor removes a path through one, as a link a clone tracks could lead out of the repository, and a rm -f or a write
# there would land wherever it points. <path> itself may be a link, which is what kit places; that is removed, not
# what it points at. It starts no process.
materialize:linked() {
  local dir=${1%/*}
  while [[ "${dir}" == .kit || "${dir}" == .kit/* ]]
  do
    [[ ! -L "${dir}" ]] || return 0
    [[ "${dir}" == */* ]] || break
    dir=${dir%/*}
  done
  return 1
}

# materialize:through-link <path> says that kit leaves <path> alone, as materialize:linked finds it under a link.
materialize:through-link() {
  error "$1 is under a link in .kit/, which kit neither writes nor removes through; remove the link, then run:" \
    "kit hydrate"
}

# materialize:within <real-path> succeeds when an absolute real path is inside the worktree: under the directory the verb
# runs in, the repository root, and not under a nested checkout such as a worktree under worktrees/, which has a .git
# of its own and goes away with kit tidy. This repository's release, kit/, is within. A release that mise or brew
# installed for a consumer is not, and neither is a rendered template, written under a temporary directory. A link
# never leaves the worktree: the worktree is what gets bind-mounted, copied, or synced as a unit, and a link that
# left it would dangle there, and again when the tool manager moved the release. The root is resolved once per shell,
# from PWD as materialize:real resolves a path, into materialize_root, as pwd -P prints it.
materialize:within() {
  local dir
  [[ -n "${materialize_root:-}" ]] || materialize:real materialize_root "${PWD}"
  [[ "$1" == "${materialize_root}"/* ]] || return 1
  dir=${1%/*}
  while [[ "${dir}" == "${materialize_root}"/* ]]
  do
    [[ ! -e "${dir}/.git" ]] || return 1
    dir=${dir%/*}
  done
}

# materialize:real <name> <path> puts in the variable <name> the real path of an existing <path>, as readlink -f
# prints it. An absolute path's directories are walked up, each checked for a link, to one resolved before, or to /
# when none is a link, so a path whose directories hold no link, or sit under one already resolved, costs no process.
# readlink resolves the rest: the first link met, which is remembered in materialize_real, a path that is a link
# itself, a relative path, and one with a . or .. segment. Its locals are named for it, so <name> can be any of the
# caller's.
materialize:real() {
  local materialize_real_path=$2 materialize_real_dir materialize_real_rest materialize_real_found
  if [[ "$2" != /* || -L "$2" || "/$2/" == */../* || "/$2/" == */./* ]]
  then
    materialize_real_found=$(readlink -f "$2")
    printf -v "$1" '%s' "${materialize_real_found}"
    return 0
  fi
  materialize_real_dir=${materialize_real_path%/*} materialize_real_rest=/${materialize_real_path##*/}
  while [[ -n "${materialize_real_dir}" && -z "${materialize_real[${materialize_real_dir}]+x}" \
    && ! -L "${materialize_real_dir}" ]]
  do
    # An empty segment, as // makes, is no directory.
    [[ -z "${materialize_real_dir##*/}" ]] \
      || materialize_real_rest=/${materialize_real_dir##*/}${materialize_real_rest}
    materialize_real_dir=${materialize_real_dir%/*}
  done
  if [[ -n "${materialize_real_dir}" ]]
  then
    [[ -n "${materialize_real[${materialize_real_dir}]+x}" ]] \
      || materialize_real[${materialize_real_dir}]=$(readlink -f "${materialize_real_dir}")
    materialize_real_rest=${materialize_real[${materialize_real_dir}]}${materialize_real_rest}
  fi
  printf -v "$1" '%s' "${materialize_real_rest}"
}

# materialize:relative <name> <from-dir> <to> puts in the variable <name> <to> as a path relative to <from-dir>. Both
# are absolute real paths.
materialize:relative() {
  local from=$2 to=$3 up=''
  while [[ "${to}" != "${from}"/* && "${from}" != / ]]
  do
    from=${from%/*}
    from=${from:-/}
    up+='../'
  done
  printf -v "$1" '%s%s' "${up}" "${to#"${from}"/}"
}

# materialize:state <name> <source> <dest> puts in the variable <name> the state the kit keeps a materialized path
# in: `link <target>`, the relative link to a source inside the worktree, or `copy` for a source outside it.
materialize:state() {
  local real target
  materialize:real real "$2"
  if materialize:within "${real}"
  then
    materialize:relative target "${materialize_root}/${3%/*}" "${real}"
    printf -v "$1" 'link %s' "${target}"
  else
    printf -v "$1" 'copy'
  fi
}

# materialize:ok <source> <dest> [<state>] succeeds when the destination is in its state (materialize:state's,
# computed here when not given): the link to the source, or a regular file with the source's content. The write bit
# is not judged here; materialize:one takes it back off a copy without a word.
materialize:ok() {
  local state=${3:-}
  [[ -n "${state}" ]] || materialize:state state "$1" "$2"
  if [[ "${state}" == link* ]]
  then [[ -L "$2" && "$(readlink "$2")" == "${state#link }" ]]
  else [[ ! -L "$2" && -f "$2" ]] && cmp --silent "$1" "$2"
  fi
}

# materialize:one <source> <dest> brings a materialized path to its state and prints what it did: kept,
# placed where nothing was there, or replaced where something else was.
materialize:one() {
  local did='' status=0
  materialize:bring did "$1" "$2" || status=$?
  [[ -z "${did}" ]] || printf '%s\n' "${did}"
  return "${status}"
}

# materialize:bring <name> <source> <dest> [<state> [<ok>]] does materialize:one's work and puts what it did in the
# variable <name> rather than printing it, so a caller in this shell keeps what it resolved and starts no process to
# read the word. <state> is materialize:state's, computed here when not given; <ok> is true or false when the caller
# judged the destination already, as materialize:ok does, and is judged here when not given.
materialize:bring() {
  local src=$2 dest=$3 state=${4:-} ok=${5:-}
  [[ -n "${state}" ]] || materialize:state state "${src}" "${dest}"
  if [[ -z "${ok}" ]]
  then
    ok=false
    ! materialize:ok "${src}" "${dest}" "${state}" || ok=true
  fi
  if [[ "${ok}" == true ]]
  then
    if [[ "${state}" == copy && -w "${dest}" ]]
    then
      chmod a-w "${dest}" || return 1
    fi
    printf -v "$1" kept
    return 0
  fi
  if [[ -e "${dest}" || -L "${dest}" ]]
  then printf -v "$1" replaced
  else printf -v "$1" placed
  fi
  if [[ "${state}" == link* ]]
  then
    file:parent "${dest}" || return 1
    ln -sfn "${state#link }" "${dest}"
  else
    file:copy --read-only "${src}" "${dest}"
  fi
}

# materialize:prune <path> removes the directories a removed path leaves empty, up to .kit/ itself, which stays.
materialize:prune() {
  local dir=${1%/*}
  while [[ "${dir}" == .kit/* && -d "${dir}" && -z "$(find "${dir}" -mindepth 1 -maxdepth 1 | head -n 1)" ]]
  do
    rmdir "${dir}"
    dir=${dir%/*}
  done
}

# materialize:judge <ok-map> <state-map> <source-map> <dest>… judges each destination as materialize:ok does, from
# its state and source in the two maps, and puts true or false for it in <ok-map>. Every copy and its source are
# digested together, by one process, and every link is read by one readlink, rather than a cmp or a readlink each. A
# destination a batch could not judge, as when a file cannot be read, is left out, for materialize:ok to judge.
# shellcheck disable=SC2034  # materialize_judge_ok is the caller's map, filled through the reference
materialize:judge() {
  local -n materialize_judge_ok=$1 materialize_judge_state=$2 materialize_judge_source=$3
  local -a materialize_judge_copies=() materialize_judge_sources=() materialize_judge_links=()
  local -a materialize_judge_targets=() materialize_judge_found=()
  local dest i n
  shift 3
  for dest in "$@"
  do
    if [[ "${materialize_judge_state["${dest}"]}" == link* ]]
    then
      if [[ -L "${dest}" ]]
      then
        materialize_judge_links+=("${dest}")
        materialize_judge_targets+=("${materialize_judge_state["${dest}"]#link }")
      else
        materialize_judge_ok["${dest}"]=false
      fi
    elif [[ ! -L "${dest}" && -f "${dest}" ]]
    then
      materialize_judge_copies+=("${dest}")
      materialize_judge_sources+=("${materialize_judge_source["${dest}"]}")
    else
      materialize_judge_ok["${dest}"]=false
    fi
  done
  n=${#materialize_judge_copies[@]}
  if (( n > 0 )) && file:digests materialize_judge_found \
    "${materialize_judge_sources[@]}" "${materialize_judge_copies[@]}" 2>/dev/null
  then
    for (( i = 0; i < n; i++ ))
    do
      dest=${materialize_judge_copies[i]}
      materialize_judge_ok["${dest}"]=false
      [[ "${materialize_judge_found[i]}" != "${materialize_judge_found[i + n]}" ]] \
        || materialize_judge_ok["${dest}"]=true
    done
  fi
  n=${#materialize_judge_links[@]}
  if (( n > 0 )) && list:read materialize_judge_found readlink -- "${materialize_judge_links[@]}" \
    && (( ${#materialize_judge_found[@]} == n ))
  then
    for (( i = 0; i < n; i++ ))
    do
      dest=${materialize_judge_links[i]}
      materialize_judge_ok["${dest}"]=false
      [[ "${materialize_judge_found[i]}" != "${materialize_judge_targets[i]}" ]] \
        || materialize_judge_ok["${dest}"]=true
    done
  fi
}

# materialize:list <dir>… prints what each directory holds, one level down, in byte order.
materialize:list() {
  { find "$@" -mindepth 1 -maxdepth 1 || true; } | LC_ALL=C sort  # a directory find cannot read lists what it can
}

# materialize:ignore-text <path>… prints .kit/.gitignore: each path kit placed under .kit/, the manifest, the file
# itself, cache/, where kit keeps what it found in this checkout, and logs/, where it logs its calls. git never sees
# any of them.
materialize:ignore-text() {
  local path
  printf '# .kit/.gitignore - what kit placed here, which git never sees (written by kit, not by hand)\n'
  {
    printf '/.gitignore\n/manifest\n/cache/\n/logs/\n'
    for path in "$@"
    do
      printf '/%s\n' "${path#.kit/}"
    done
  } | LC_ALL=C sort
}

# materialize:all [--check] <package>… brings every path the packages deliver under .kit/ to its state and writes
# .kit/.gitignore, keeping the clone's trust across it, as trust:keeping does. It says what it changed, one line per
# path. A path the manifest never recorded that holds a
# different file is the repository's: it is left alone and named, as kit install names it. It warns about a file
# under a delivered directory that no active package delivers, and about a delivered path git still tracks, both of
# which a person settles; an empty directory left behind by a removed package is removed without a word. A path under
# a link in .kit/, or a .kit that is a link, it neither writes nor removes, and names, and then it fails. With
# --check it changes nothing: each of those is an error, and it prints how many on stdout.
materialize:all() {
  local check=false path src dir entry state did ignore findings=0 refused=0 row kind
  local -a owned=() tracked=() rows dirs=()
  local -A source=() within=() recorded=() states=() ok=() listing=()
  if [[ "${1:-}" == --check ]]
  then
    check=true
    shift
  elif [[ -z "${trust_keeping:-}" ]]
  then
    trust:keeping materialize:all "$@"
    return
  fi
  if [[ -L .kit ]]
  then
    error ".kit is a link, which kit neither writes nor removes through; make it a directory, then run: kit hydrate"
    [[ "${check}" == false ]] || printf '1\n'
    return 1
  fi
  materialize_real=()
  list:read rows place:delivered "${kit_root}" repo "$@" || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r path src <<<"${row}"
    [[ -n "${path}" ]] || continue
    materialize:owned repo "${path}" || continue
    place:kind "${src}" "${path}" kind
    [[ "${kind}" == copy ]] || continue
    owned+=("${path}")
    source[${path}]=${src}
    dir=${path%/*}
    while [[ "${dir}" != .kit ]]
    do
      within[${dir}]=1
      dir=${dir%/*}
    done
  done
  manifest:records recorded repo || return 1
  # A path the manifest records that no active package delivers any more, as after a pulled declaration dropped a
  # package: removed, and the directories it leaves empty with it.
  list:read rows manifest:paths repo || return 1
  for path in "${rows[@]}"
  do
    [[ -n "${path}" ]] || continue
    materialize:owned repo "${path}" || continue
    [[ -z "${source[${path}]+x}" ]] || continue
    [[ -e "${path}" || -L "${path}" ]] || continue
    if materialize:linked "${path}"
    then
      materialize:through-link "${path}"
      findings=$(( findings + 1 )) refused=$(( refused + 1 ))
    elif [[ "${check}" == true ]]
    then
      error "${path} is no longer delivered by any active package; run: kit hydrate"
      findings=$(( findings + 1 ))
    else
      step "${path}: removed, no active package delivers it"
      rm -f "${path}"
      materialize:prune "${path}"
    fi
  done
  # Each path's state and whether it is in it, judged for all of them at once, before any is changed.
  for path in "${owned[@]}"
  do
    materialize:state state "${source[${path}]}" "${path}"
    states[${path}]=${state}
  done
  (( ${#owned[@]} == 0 )) || materialize:judge ok states source "${owned[@]}"
  for path in "${owned[@]}"
  do
    src=${source[${path}]}
    state=${states[${path}]}
    if materialize:linked "${path}"
    then
      materialize:through-link "${path}"
      findings=$(( findings + 1 )) refused=$(( refused + 1 ))
      continue
    fi
    if [[ -z "${ok[${path}]:-}" ]]
    then
      ok[${path}]=false
      ! materialize:ok "${src}" "${path}" "${state}" || ok[${path}]=true
    fi
    if [[ "${check}" == false ]]
    then
      if [[ ( -e "${path}" || -L "${path}" ) && -z "${recorded[${path}]:-}" && "${ok[${path}]}" == false ]]
      then
        warn "${path}: a different file is there already; move it aside, or remove it if it is not yours"
        continue
      fi
      materialize:bring did "${src}" "${path}" "${state}" "${ok[${path}]}"
      case "${did}" in
        (placed) step "${path}: placed" ;;
        (replaced) warn "${path} was changed in place, or came from another release; replaced"
      esac
    elif [[ ! -e "${path}" && ! -L "${path}" ]]
    then
      error "${path} is missing; run: kit hydrate"
      findings=$(( findings + 1 ))
    elif [[ "${ok[${path}]}" == false ]]
    then
      if [[ "${state}" == link* ]]
      then error "${path} is not a link to ${state#link }; run: kit hydrate"
      elif [[ -z "${recorded[${path}]:-}" ]]
      then error "${path} holds a different file; move it aside, or remove it if it is not yours, then run: kit hydrate"
      else error "${path} was changed in place; run: kit hydrate (a delivered file changes by a new release, never in place)"
      fi
      findings=$(( findings + 1 ))
    fi
  done
  # Every delivered directory is listed by one find, its entries filed under the directory they are in.
  for dir in "${!within[@]}"
  do
    [[ ! -d "${dir}" ]] || materialize:linked "${dir}/-" || dirs+=("${dir}")
  done
  if (( ${#dirs[@]} > 0 ))
  then
    list:read rows materialize:list "${dirs[@]}" || return 1
    for entry in "${rows[@]}"
    do
      listing[${entry%/*}]+=${entry}$'\n'
    done
  fi
  for dir in "${!within[@]}"
  do
    [[ -d "${dir}" ]] || continue
    ! materialize:linked "${dir}/-" || continue
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
    done <<<"${listing[${dir}]:-}"
  done
  if (( ${#owned[@]} > 0 )) && git rev-parse --is-inside-work-tree >/dev/null 2>&1
  then
    list:read tracked git ls-files -- "${owned[@]}" .kit/manifest .kit/.gitignore || return 1
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
  [[ "${check}" == true ]] || (( refused == 0 ))
}

# materialize:hydrated succeeds when .kit/.gitignore exists and every path it names is there, as a link or without the
# write bit, and .kit/manifest exists. cache/ and logs/ are named but never placed, so they are skipped. Where this
# release is itself inside the worktree, as the kit's own kit/ is, every path must be a link: a copy there came from a
# kit outside the worktree, and the full run turns it into a link. It starts at most three processes, to look under
# .kit/packages/, and to resolve a link on the worktree's path and on the release's, so a reading verb asks on every
# run. A fresh clone, a `git clean -fdx`, a link an editor's save replaced, and a copy with its write bit back all
# fail it, and the full materialize:all follows, as it does when kit.yml is newer than .kit/.gitignore: a declaration
# edited by hand or pulled since the last full run, whose packages may have come or gone. So does a file or directory
# under .kit/packages/ newer than .kit/.gitignore, or a look there that fails: a repository package's file added,
# changed, or removed since, by hand or by a pull, which .kit/.gitignore cannot name yet. A caller takes a failure for
# work to do, never for an error. A copy changed and put back without the write bit passes it: kit check judges that. A
# template rendered under .kit/ is a copy even inside, so a release that delivers one there takes the full run every
# time; none does.
materialize:hydrated() {
  local line path newer real inside=false
  [[ -f .kit/.gitignore && -f .kit/manifest ]] || return 1
  [[ ! kit.yml -nt .kit/.gitignore ]] || return 1
  if [[ -d .kit/packages ]]
  then
    newer=$(find .kit/packages -newer .kit/.gitignore -print -quit) || return 1
    [[ -z "${newer}" ]] || return 1
  fi
  # Both are kit's to write, so both are without the write bit, except as root, where the bit says nothing.
  [[ ! -w .kit/.gitignore && ! -w .kit/manifest ]] || (( EUID == 0 )) || return 1
  materialize:real real "${kit_root}"
  ! materialize:within "${real}" || inside=true
  while IFS= read -r line
  do
    [[ "${line}" == /* ]] || continue
    case "${line}" in
      (/.gitignore | /manifest | /cache/ | /logs/) continue
    esac
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
# with --full, as kit hydrate calls it, it always does the whole job and says what it finds. It returns 1 when kit.yml
# cannot be read, before it places anything, and when a package cannot be rendered or placed. It warns when this
# release is not the declared one, since what it places is this release's. It rewrites .kit/manifest, so what the
# declaration added is recorded, and a later release's change to it is kit's to replace. Where the declaration still
# sits on the manifest's lines, from a release before it, it moves into kit.yml first.
#
# Under kit serve, kit_serve_hydrated is set for every tool's verb, and it does not place anything: the server placed
# .kit/ already, and places it again when kit.yml changes, so a tool reads and never writes.
materialize:hydrate() {
  [[ -z "${kit_serve_hydrated:-}" ]] || return 0
  local declared in_file
  local -a packages excluded
  declared=$(settings:version) || return 1
  [[ -n "${declared}" ]] || return 0
  in_file=$(settings:value version) || return 1
  if [[ -z "${in_file}" ]]
  then
    # A manifest from a release before the declaration still carries it. It moves into kit.yml now, before
    # anything rewrites the manifest, which would drop its only copy.
    list:read packages settings:packages || return 1
    list:read excluded settings:excludes || return 1
    trust:keeping settings:declare "${declared}" "${excluded[*]}" "${packages[@]}" || return 1
    step "kit.yml: the declaration moved here from .kit/manifest"
  fi
  if [[ "${1:-}" != --full ]] && materialize:hydrated
  then
    return 0
  fi
  if [[ "${declared}" != "$(release:version)" ]]
  then
    warn "kit.yml declares kit ${declared}, and this is kit $(release:version); what is placed is this" \
      "kit's; run: $(release:update-command)"
  fi
  list:read packages settings:packages || return 1
  render:all "${kit_root}" "${packages[@]}" || return 1
  trust:keeping materialize:hydrate-write "${packages[@]}"
}

# materialize:hydrate-write <package>… places what the packages deliver under .kit/, and rewrites .kit/manifest.
materialize:hydrate-write() {
  materialize:all "$@" || return 1
  manifest:write repo '' "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
