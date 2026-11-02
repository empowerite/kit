# shellcheck shell=bash
# settings.sh - kit.yml, the repository's settings: read, validated, and the declaration written into it (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The repository's choices are in kit.yml at the root. The kit reads every key it knows and writes three: version,
# packages, and excludes, the declaration (settings:declare), rewritten where it stands so the repository's own keys
# and comments keep their places. A repository from a release before the declaration still carries it on the first
# lines of .kit/manifest, and settings:version reads it there until materialize:hydrate moves it into kit.yml.
#
# see: docs/packages.md

use yaml file resolve

# settings:file prints the path of the repository's settings file.
settings:file() { printf 'kit.yml\n'; }

# settings:value <key> prints the value of one top-level setting, or nothing.
settings:value() { yaml:value "$(settings:file)" "$1"; }

# settings:validate fails on a settings file that the schema rejects, as far as bash can check:
#
#   - a key the kit does not know
#   - a require_signing that is not true or false
#   - an answer key with characters outside [a-z_]
#   - a tracker without a prefix (letters and an optional #) or without a provider
#
# The gate validates this repository's settings against .kit/kit.schema.json in full. The kit delivers the
# schema so an editor can validate against it anywhere.
settings:validate() {
  local settings key value
  settings=$(settings:file)
  [[ -f "${settings}" ]] || return 0
  for key in $(yaml:keys "${settings}")
  do
    case "${key}" in
      (version | packages | excludes) ;;
      (forge | gate | require_signing | answers | trackers | toolchains | policy | policy_engine | bash_path) ;;
      (linked_exempt | sources | status) ;;
      (*)
        error "${settings} has a key the kit does not know: ${key} (see .kit/kit.schema.json)"
        return 1
    esac
  done
  value=$(yaml:value "${settings}" require_signing)
  if [[ -n "${value}" && "${value}" != true && "${value}" != false ]]
  then
    error "${settings}: require_signing must be true or false, not '${value}'"
    return 1
  fi
  while IFS=$'\t' read -r key value
  do
    [[ -z "${key}" || "${key}" =~ ^[a-z_]+$ ]] && continue
    error "${settings}: the answer key '${key}' is not letters and underscores, as a template's @key@ must be"
    return 1
  done < <(yaml:map "${settings}" answers)
  value=$(yaml:value "${settings}" linked_exempt)
  if [[ -n "${value}" ]]
  then
    error "${settings}: linked_exempt must be a list of logins, one per line under it, not '${value}'"
    return 1
  fi
  while IFS=$'\x1f' read -r key value
  do
    [[ -n "${key}${value}" ]] || continue
    if [[ -z "${key}" || ! "${key}" =~ ^[A-Za-z]*#?$ || -z "${value}" ]]
    then
      error "${settings}: a tracker needs a prefix, letters and an optional #, and a provider (prefix '${key}')"
      return 1
    fi
  done < <(yaml:records "${settings}" trackers prefix provider)
}

# settings:context <root> <package> prints "<key>\t<value>" for every key a template or region of the package can
# use. The keys come from:
#
#   1. the package's defaults under answers: in package.yml, as <key> and as answers.<key>
#   2. kit.yml: each top-level scalar as <key>, each map entry as <map>.<name>, and each entry of the
#      answers map also as <name>
#
# A later source overrides an earlier one, so the repository's answer wins over the package's default.
settings:context() {
  local settings key name value
  settings=$(settings:file)
  {
    while IFS=$'\t' read -r name value
    do
      [[ -n "${name}" ]] || continue
      printf '%s\t%s\n%s\t%s\n' "${name}" "${value}" "answers.${name}" "${value}"
    done < <(yaml:map "$(resolve:meta "$1" "$2")" answers)
    for key in $(yaml:keys "${settings}")
    do
      value=$(yaml:value "${settings}" "${key}")
      [[ -n "${value}" ]] && printf '%s\t%s\n' "${key}" "${value}"
      while IFS=$'\t' read -r name value
      do
        [[ -n "${name}" ]] || continue
        printf '%s\t%s\n' "${key}.${name}" "${value}"
        [[ "${key}" == answers ]] && printf '%s\t%s\n' "${name}" "${value}"
      done < <(yaml:map "${settings}" "${key}")
    done
  } | awk -F '\t' '{ a[$1] = $2 } END { for (k in a) print k "\t" a[k] }'
}

# settings:declare <version> <excluded> <package>… writes the repository's declaration into kit.yml: the
# top-level keys version, packages, and excludes, the last left out when nothing is excluded. Every other line of
# the file stays as it is, and in its place: the three keys' blocks are dropped and the declaration is written
# where the first of them stood, or appended after one blank line when the file had none. A missing file is created
# with the schema modeline. yaml.sh only reads, so this is a line-oriented rewrite.
settings:declare() {
  local version=$1 excluded=$2 settings kept declaration item
  local -a excludes=()
  shift 2
  settings=$(settings:file)
  if [[ -f "${settings}" ]]
  then
    # A marker, \001, takes the first block's place. Command substitution drops the trailing blank lines.
    kept=$(settings:declaration-kept "${settings}" $'\001')
  else
    # shellcheck disable=SC2016  # $schema is the modeline's own text, not the shell's
    kept='# yaml-language-server: $schema=.kit/kit.schema.json'
  fi
  [[ -z "${excluded}" ]] || read -r -a excludes <<<"${excluded}"
  declaration=$(
    printf 'version: %s\n' "${version}"
    printf 'packages:\n'
    for item in "$@"
    do
      printf '  - %s\n' "${item}"
    done
    if (( ${#excludes[@]} > 0 ))
    then
      printf 'excludes:\n'
      for item in "${excludes[@]}"
      do
        printf '  - %s\n' "${item}"
      done
    fi
  )
  file:replace "${settings}" settings:declaration-text "${kept}" "${declaration}"
}

# settings:declaration-kept <settings> <marker> prints the settings file without the blocks of version, packages, and
# excludes, with <marker> on a line of its own where the first block stood, or nothing there when it is empty. A block
# is its key line, the indented lines after it, and the comments and blank lines among them. A comment or blank line
# after the block's last item belongs to what follows, and stays.
settings:declaration-kept() {
  awk -v marker="$2" '
    function flush() { for (i = 1; i <= held; i++) print hold[i]; held = 0 }
    /^(version|packages|excludes):/ { if (!seen) { if (marker != "") print marker; seen = 1 }; skip = 1; held = 0; next }
    skip && /^  / { held = 0; next }
    skip && (/^#/ || /^[[:space:]]*$/) { hold[++held] = $0; next }
    { if (skip) flush(); skip = 0; print }
    END { if (skip) flush() }' "$1"
}

# settings:declaration-text <kept> <declaration> prints kit.yml: the kept lines with the declaration where the marker
# stands, or, with no marker, the kept lines, a blank line, and the declaration.
settings:declaration-text() {
  local kept=$1 declaration=$2 line
  if [[ "${kept}" == *$'\001'* ]]
  then
    while IFS= read -r line
    do
      if [[ "${line}" == $'\001' ]]
      then printf '%s\n' "${declaration}"
      else printf '%s\n' "${line}"
      fi
    done <<<"${kept}"
  else
    [[ -z "${kept}" ]] || printf '%s\n\n' "${kept}"
    printf '%s\n' "${declaration}"
  fi
}

# settings:version prints the kit version the repository declares in kit.yml. A manifest that
# still carries a version line, from a release before the declaration, is read as the declaration until kit update
# moves it, and so are its packages and excludes lines below.
settings:version() {
  local version
  version=$(settings:value version)
  [[ -n "${version}" ]] || version=$(sed -n 's/^version //p' .kit/manifest 2>/dev/null)
  printf '%s\n' "${version}"
}

# settings:declared-line <key> prints the items of an old manifest's <key> line, one per line, where kit.yml
# declares no version.
settings:declared-line() {
  [[ -z "$(settings:value version)" ]] || return 0
  sed -n "s/^$1 *//p" .kit/manifest 2>/dev/null | tr ' ' '\n' | awk 'NF'
}

# settings:packages prints the active packages the repository declares in kit.yml, one per line.
settings:packages() {
  if [[ -n "$(settings:value version)" ]]
  then yaml:list "$(settings:file)" packages
  else settings:declared-line packages
  fi
}

# settings:excludes prints the packages the repository declares as excluded in kit.yml, one per
# line: members of an active bundle that were removed, which kit update leaves out.
settings:excludes() {
  if [[ -n "$(settings:value version)" ]]
  then yaml:list "$(settings:file)" excludes
  else settings:declared-line excludes
  fi
}

# settings:require fails in a repository where the kit was never installed: its kit.yml
# declares no kit version.
settings:require() {
  if [[ -z "$(settings:version)" ]]
  then
    error "run: kit install (kit.yml declares no kit version in this repository)"
    return 1
  fi
}

# settings_stages names the stages of kit status, in the order work moves: the values the status setting takes.
declare -gra settings_stages=(backlog incoming desk worktrees changes builds releases)

# settings:stages prints the stages the status setting names, one per line, in its order: what kit status draws.
# Without the setting, or with it null, it prints every stage. It refuses, with the valid names, and returns 2: a
# setting that is not a list, a list that names no stage, a name that is no stage, and a name given twice.
settings:stages() {
  local settings type value stage
  local -a stages=() seen=()
  settings=$(settings:file)
  type=$(yaml:type "${settings}" status) || return 2
  case "${type}" in
    ('' | '!!null') stages=("${settings_stages[@]}") ;;
    ('!!seq')
      readarray -t stages < <(yaml:list "${settings}" status)
      if (( ${#stages[@]} == 0 ))
      then
        error "status in ${settings} names no stage; the stages are: ${settings_stages[*]}"
        return 2
      fi
      ;;
    (*)
      value=$(yaml:value "${settings}" status)
      error "status in ${settings} must be a list of stages, not '${value:-${type#!!}}'"
      return 2
  esac
  for stage in "${stages[@]}"
  do
    if ! list:among "${stage}" "${settings_stages[@]}"
    then
      error "'${stage}' is not a stage; the stages are: ${settings_stages[*]} (status in ${settings})"
      return 2
    fi
    if list:among "${stage}" "${seen[@]}"
    then
      error "'${stage}' is named twice in status in ${settings}; name each stage once"
      return 2
    fi
    seen+=("${stage}")
  done
  printf '%s\n' "${stages[@]}"
}

# ----------------------------------------------------------------------------------------------------------------------
