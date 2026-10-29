# shellcheck shell=bash
# release.sh - reads a release's packages, renders templates and regions, merges JSON, writes manifests (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2016  # jq filters name jq variables with $, in single quotes on purpose
# shellcheck disable=SC2154  # kit_root is exported by bash/1st.sh
#
# A release is bin/, bash/, and packages/. Each packages/<name>/ is a package, which contains:
#
#   package.yml          the package's metadata
#   files/<base>/       files copied to the base's root, at the same paths
#   templates/<base>/   files whose @key@ placeholders are filled from the repository's settings, then copied the
#                       same way
#   regions/<base>/     files rendered like templates, each placed as a region inside a file the repository owns
#
# A region is the part of a repository's file that the kit owns. It sits between a bare kit:generate line and an
# end line, both naming the path. The rest of the file belongs to the repository. kit generate replaces the
# region's content with what kit region prints.
#
# A repository can define its own packages the same way, under .kit/packages/<name>/. A package name is looked up in
# the release first, then there.
#
# The bases a package delivers to:
#
#   repo    the repository root, recorded in .kit/manifest
#   home    the user's home directory, recorded in ~/.kit/manifest, which every repository on the machine shares
#
# How a delivered file is placed:
#
#   - a destination ending in .json (except *.schema.json) is merged by key; the kit owns only the entries it
#     writes
#   - a region is regenerated, never merged
#   - every other file is copied
#
# The repository's choices are in .kit/settings.yml. The kit reads that file and never writes it.
#
# Every function reads a release from a root directory: kit_root by default, which bash/1st.sh exports. kit:enter
# changes to the repository root, so .kit/manifest and every repo path here are relative to it. Every sort and
# comm here uses the C locale, so an order written on one machine reads the same on another.
#
# see: docs/packages.md

use yaml

# release:unreleased prints `unreleased`, the version of a kit checkout. The release workflow
# (.github/workflows/release.yml) writes a version file into each release; a checkout has none. kit update never
# fetches this version.
release:unreleased() { printf 'unreleased\n'; }

# release:version prints this release's version.
release:version() {
  cat "${kit_root}/version" 2>/dev/null || release:unreleased
}

# release:bases prints the bases a package can deliver to. Each base is a subdirectory name under files/,
# templates/, and regions/.
release:bases() { printf 'repo\nhome\n'; }

# release:base-root <base> prints the directory a base's files are copied to: `.` (the repository root, where the
# verbs run) for repo, and the home directory for home. KIT_HOME, when set, replaces the home directory, so a test
# can use a stand-in.
release:base-root() {
  case "$1" in
    (repo) printf '.\n' ;;
    (home) printf '%s\n' "${KIT_HOME:-${HOME}}" ;;
    (*)
      error "'$1' is not a base; the bases are: $(release:bases | paste -s -d ' ' -)"
      return 2
  esac
}

# release:dest <base> <path> prints where a delivered path is written.
release:dest() {
  if [[ "$1" == repo ]]
  then printf '%s\n' "$2"
  else printf '%s/%s\n' "$(release:base-root "$1")" "$2"
  fi
}

# release:manifest <base> prints the base's manifest: .kit/manifest in the repository, ~/.kit/manifest for home.
release:manifest() { release:dest "$1" .kit/manifest; }

# release_local is the root of the repository's own packages, .kit/packages/<name>/, laid out like the release's.
declare -gr release_local=.kit

# release:settings prints the path of the repository's settings file.
release:settings() { printf '.kit/settings.yml\n'; }

# release:setting <key> prints the value of one top-level setting, or nothing.
release:setting() { yaml:value "$(release:settings)" "$1"; }

# release:packages [<root>] prints the packages in reach, in the release at <root> and in the repository's
# .kit/packages/. A package is a directory that contains a package.yml: packages/<name>/, or
# packages/<namespace>/<name>/ under a namespace, a directory with no package.yml of its own. Its name is its path:
# kit, or style/bash. A package's own subdirectories are never packages.
release:packages() {
  local root=${1:-${kit_root}} dir
  for dir in "${root}" "${release_local}"
  do
    [[ -d "${dir}/packages" ]] || continue
    (cd "${dir}/packages" && find . -mindepth 2 -maxdepth 3 -name package.yml | sed 's@^\./@@; s@/package.yml$@@')
  done | LC_ALL=C sort --unique | awk '
    { name[NR] = $0; if ($0 !~ /\//) top[$0] = 1 }
    END {
      for (i = 1; i <= NR; i++) { split(name[i], part, "/"); if (!(name[i] ~ /\// && part[1] in top)) print name[i] }
    }'
}

# release:expand <root> <name>… prints each package a name stands for, in reach of the release at <root>: the name
# itself, or every package a glob matches, such as style/*. It fails on a glob that matches nothing, and leaves a
# plain name to release:require-packages.
release:expand() {
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
    done < <(release:packages "${root}")
    if (( matched == 0 ))
    then
      error "'${name}' matches no package in reach; the packages are: $(release:packages "${root}" | paste -s -d ' ' -)"
      return 2
    fi
  done
}

# release:home <root> <package> prints the directory that contains the package: <root> when the release has the
# package, otherwise the repository's .kit. The package's files are under <home>/packages/<package>/.
release:home() {
  if [[ -f "$1/packages/$2/package.yml" || ! -f "${release_local}/packages/$2/package.yml" ]]
  then printf '%s\n' "$1"
  else printf '%s\n' "${release_local}"
  fi
}

# release:origin <root> <package> prints where the package comes from: "release" or "repository".
release:origin() {
  if [[ "$(release:home "$1" "$2")" == "$1" ]]
  then printf 'release\n'
  else printf 'repository\n'
  fi
}

# release:meta <root> <package> prints the path of the package's metadata.
release:meta() { printf '%s/packages/%s/package.yml\n' "$(release:home "$1" "$2")" "$2"; }

# release:description <root> <package> prints the package's one-line description.
release:description() { yaml:value "$(release:meta "$1" "$2")" description; }

# release:depends <root> <package> prints the packages this one depends on, one per line.
release:depends() { yaml:list "$(release:meta "$1" "$2")" depends; }

# release:includes <root> <package> prints the members of this bundle, one per line: packages activated with it, each
# of which can be removed on its own.
release:includes() { yaml:list "$(release:meta "$1" "$2")" includes; }

# release:default-packages <root> prints the packages whose default is true. They are active unless removed.
release:default-packages() {
  local package
  for package in $(release:packages "$1")
  do
    [[ "$(yaml:value "$(release:meta "$1" "${package}")" default)" == true ]] && printf '%s\n' "${package}"
  done
  return 0
}

# release:require-packages <root> <name>… fails on a package that is not in reach, and lists the packages that are.
release:require-packages() {
  local root=$1 package
  shift
  for package in "$@"
  do
    if [[ ! -f "$(release:meta "${root}" "${package}")" ]]
    then
      error "'${package}' is not a package in reach; the packages are:" \
        "$(release:packages "${root}" | paste -s -d ' ' -)"
      return 2
    fi
  done
}

# release:resolve <root> <excluded> <name>… prints the named packages, the packages they depend on, and the members
# they include, each once. <excluded> is a space-separated list of packages that no bundle brings in; a package that
# another depends on comes in regardless. Each package comes after the packages it depends on and includes. Ties
# are broken alphabetically, so the order is the same on every machine. It fails on a cycle.
release:resolve() {
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
    for dep in $(release:depends "${root}" "${package}")
    do
      queue+=("${dep}")
    done
    for dep in $(release:includes "${root}" "${package}")
    do
      # shellcheck disable=SC2086  # the list splits on purpose
      release:among "${dep}" ${excluded} || queue+=("${dep}")
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
      for dep in $(release:depends "${root}" "${package}") $(release:includes "${root}" "${package}")
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

# release:dependents <root> <package> <active>… prints the active packages that depend on <package>.
release:dependents() {
  local root=$1 package=$2 other dep
  shift 2
  for other in "$@"
  do
    for dep in $(release:depends "${root}" "${other}")
    do
      [[ "${dep}" == "${package}" ]] && printf '%s\n' "${other}"
    done
  done
  return 0
}

# release:includers <root> <package> <active>… prints the active packages that include <package>.
release:includers() {
  local root=$1 package=$2 other member
  shift 2
  for other in "$@"
  do
    for member in $(release:includes "${root}" "${other}")
    do
      [[ "${member}" == "${package}" ]] && printf '%s\n' "${other}"
    done
  done
  return 0
}

# release:exclusions <excluded> <active>… prints the packages of <excluded>, a space-separated list, that are not
# active: sorted, each once, one per line.
release:exclusions() {
  local excluded=$1 package
  shift
  for package in ${excluded}
  do
    release:among "${package}" "$@" || printf '%s\n' "${package}"
  done | LC_ALL=C sort --unique
}

# release:validate <root> <package> fails on metadata that the schema rejects, as far as bash can check without a
# validator:
#
#   - a key the kit does not know
#   - a missing description
#   - a default that is not true or false
#   - a dependency on, or a member that is, a package not in reach
#
# The gate validates against packages/package.schema.json in full.
release:validate() {
  local root=$1 package=$2 meta key value dep
  meta=$(release:meta "${root}" "${package}")
  for key in $(yaml:keys "${meta}")
  do
    case "${key}" in
      (description | default | depends | includes | answers | expects) ;;
      (*)
        error "$(release:meta "${root}" "${package}") has a key the kit does not know: ${key}" \
          "(see packages/package.schema.json)"
        return 1
    esac
  done
  if [[ -z "$(yaml:value "${meta}" description)" ]]
  then
    error "$(release:meta "${root}" "${package}") has no description"
    return 1
  fi
  value=$(yaml:value "${meta}" default)
  if [[ -n "${value}" && "${value}" != true && "${value}" != false ]]
  then
    error "$(release:meta "${root}" "${package}"): default must be true or false, not '${value}'"
    return 1
  fi
  for dep in $(release:depends "${root}" "${package}")
  do
    if [[ ! -f "$(release:meta "${root}" "${dep}")" ]]
    then
      error "$(release:meta "${root}" "${package}") depends on '${dep}', which is not in reach"
      return 1
    fi
  done
  for dep in $(release:includes "${root}" "${package}")
  do
    if [[ ! -f "$(release:meta "${root}" "${dep}")" ]]
    then
      error "$(release:meta "${root}" "${package}") includes '${dep}', which is not in reach"
      return 1
    fi
  done
}

# release:expectations <root> <package>… prints "<path>\x1f<pattern>\x1f<remedy>" for each line the packages expect
# in a file the repository owns. Such a line is the include that pulls in the kit's file, such as lefthook's
# extends or just's import.
release:expectations() {
  local root=$1 package
  shift
  for package in "$@"
  do
    yaml:records "$(release:meta "${root}" "${package}")" expects path pattern remedy
  done
}

# release:unmet <root> <package>… prints the remedy for each expectation the repository does not meet, one per
# line. An expectation is unmet when its file is missing, or when no line in the file matches its pattern (an
# extended regular expression). The kit never writes to such a file; it only reports what is missing.
release:unmet() {
  local root=$1 path pattern remedy
  shift
  while IFS=$'\x1f' read -r path pattern remedy
  do
    [[ -n "${path}" ]] || continue
    if [[ ! -f "${path}" ]] || ! grep --quiet --extended-regexp -- "${pattern}" "${path}"
    then
      printf '%s: %s\n' "${path}" "${remedy:-a line matching ${pattern} is expected}"
    fi
  done < <(release:expectations "${root}" "$@")
  return 0
}

# release:validate-settings fails on a settings file that the schema rejects, as far as bash can check:
#
#   - a key the kit does not know
#   - a require_signing that is not true or false
#   - an answer key with characters outside [a-z_]
#   - a tracker without a prefix (letters and an optional #) or without a provider
#
# The gate validates this repository's settings against .kit/settings.schema.json in full. The kit delivers the
# schema so an editor can validate against it anywhere.
release:validate-settings() {
  local settings key value
  settings=$(release:settings)
  [[ -f "${settings}" ]] || return 0
  for key in $(yaml:keys "${settings}")
  do
    case "${key}" in
      (forge | gate | require_signing | answers | trackers | toolchains | policy | policy_engine | bash_path) ;;
      (*)
        error "${settings} has a key the kit does not know: ${key} (see .kit/settings.schema.json)"
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

# release:context <root> <package> prints "<key>\t<value>" for every key a template or region of the package can
# use. The keys come from:
#
#   1. the package's defaults under answers: in package.yml, as <key> and as answers.<key>
#   2. .kit/settings.yml: each top-level scalar as <key>, each map entry as <map>.<name>, and each entry of the
#      answers map also as <name>
#
# A later source overrides an earlier one, so the repository's answer wins over the package's default.
release:context() {
  local settings key name value
  settings=$(release:settings)
  {
    while IFS=$'\t' read -r name value
    do
      [[ -n "${name}" ]] || continue
      printf '%s\t%s\n%s\t%s\n' "${name}" "${value}" "answers.${name}" "${value}"
    done < <(yaml:map "$(release:meta "$1" "$2")" answers)
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

# release:tmp prints a directory for this process's rendered templates. It is created on first use and removed
# at exit. release:render-all does the rendering; every verb runs it before reading a release.
release:tmp() {
  if [[ -z "${release_tmp:-}" ]]
  then
    release_tmp=$(mktemp -d)
    # shellcheck disable=SC2064  # expanded now on purpose: the path is fixed from here on
    trap "rm -rf '${release_tmp}'" EXIT
  fi
  printf '%s\n' "${release_tmp}"
}

# release:render <root> <package> <kind> <base> renders the package's <kind> (templates or regions) for <base>, once
# per process, and prints the directory that contains the result. It prints nothing when the package has none. The
# directory keeps the packages/<package>/<kind>/<base> layout, so a rendered region is still recognized as a region.
#
# A placeholder is written @key@ (autoconf's form), which cannot be confused with bash's ${var} or GitHub's
# ${{ }}. A key can be dotted: @answers.key@. A key with no value fails, and the error names the key.
release:render() {
  local root=$1 package=$2 kind=$3 base=$4 source out key value text file rel map
  local -A answer=()
  source="$(release:home "${root}" "${package}")/packages/${package}/${kind}/${base}"
  [[ -d "${source}" ]] || return 0
  out="$(release:tmp)/$(printf '%s' "${root}" | release:digest)/packages/${package}/${kind}/${base}"
  if [[ ! -d "${out}" ]]
  then
    while IFS=$'\t' read -r key value
    do
      [[ -n "${key}" ]] && answer[${key}]=${value}
    done < <(release:context "${root}" "${package}")
    while IFS= read -r file
    do
      rel=${file#"${source}"/}
      text=$(<"${file}")
      for key in $(grep --only-matching '@[a-z_][a-z_.]*@' "${file}" | LC_ALL=C sort --unique | tr -d '@')
      do
        if [[ -z "${answer[${key}]+x}" ]]
        then
          map=answers
          [[ "${key}" == *.* ]] && map=${key%.*}
          error "${rel} needs an answer for '${key}'; add '${key##*.}: <value>' under ${map}: in .kit/settings.yml" \
            "(package ${package})"
          return 1
        fi
        text=${text//"@${key}@"/"${answer[${key}]}"}
      done
      mkdir -p "${out}/$(dirname "${rel}")"
      printf '%s\n' "${text}" > "${out}/${rel}"
    done < <(find "${source}" -type f)
  fi
  printf '%s\n' "${out}"
}

# release:render-all <root> <package>… prepares the release at <root> for the named packages. In order, it:
#
#   1. validates the settings
#   2. fails on a named package that is not in reach
#   3. validates every package in reach
#   4. renders the named packages, as release:render-packages does
release:render-all() {
  local root=$1 package
  shift
  release:validate-settings || return 1
  release:require-packages "${root}" "$@" || return 2
  for package in $(release:packages "${root}")
  do
    release:validate "${root}" "${package}" || return 1
  done
  release:render-packages "${root}" "$@"
}

# release:render-packages <root> <package>… renders the named packages' templates and regions. It fails on a base
# the kit does not deliver to, on a delivered file that is a link to nothing, and on regions that
# release:regions-apart rejects. A package the release at <root> does not have renders nothing, so kit update can
# read a base release that predates a package: that release delivered nothing for it.
release:render-packages() {
  local root=$1 package kind base
  shift
  for package in "$@"
  do
    for kind in files templates regions
    do
      for base in $(cd "$(release:home "${root}" "${package}")/packages/${package}/${kind}" 2>/dev/null \
        && find . -mindepth 1 -maxdepth 1 -type d | sed 's@^\./@@')
      do
        # shellcheck disable=SC2046  # the bases are single words
        if ! release:among "${base}" $(release:bases)
        then
          error "packages/${package}/${kind}/${base} names a base the kit does not deliver to; the bases are:" \
            "$(release:bases | paste -s -d ' ' -)"
          return 1
        fi
      done
    done
    for kind in templates regions
    do
      for base in $(release:bases)
      do
        release:render "${root}" "${package}" "${kind}" "${base}" >/dev/null || return 1
      done
    done
  done
  for base in $(release:bases)
  do
    release:delivered "${root}" "${base}" "$@" >/dev/null || return 1
    release:regions-apart "${root}" "${base}" "$@" || return 1
  done
}

# release:regions-apart <root> <base> <package>… fails when two of the packages deliver a region to the same path (a
# file can have only one kit region), or when a region is in a kind of file whose comment syntax the kit does not
# know.
release:regions-apart() {
  local root=$1 base=$2 package home file rel owner
  shift 2
  local -A owner_of=()
  for package in "$@"
  do
    home=$(release:home "${root}" "${package}")
    [[ -d "${home}/packages/${package}/regions/${base}" ]] || continue
    while IFS= read -r file
    do
      rel=${file#"${home}/packages/${package}/regions/${base}/"}
      if ! release:region-known "${rel}"
      then
        error "packages/${package}/regions/${base}/${rel}: the kit knows no comment syntax for that kind of file," \
          "so it cannot mark a region there"
        return 1
      fi
      owner=${owner_of[${rel}]:-}
      if [[ -n "${owner}" ]]
      then
        error "packages ${owner} and ${package} both deliver a region to ${rel}; one file holds one kit region"
        return 1
      fi
      owner_of[${rel}]=${package}
    done < <(find "${home}/packages/${package}/regions/${base}" -type f)
  done
}

# release:layers <root> <base> <package>… prints the directories copied to the base's root, in package order: each
# package's files/<base>, its rendered templates/<base>, and its rendered regions/<base>.
release:layers() {
  local root=$1 base=$2 package home kind rendered
  shift 2
  for package in "$@"
  do
    home=$(release:home "${root}" "${package}")
    [[ -d "${home}/packages/${package}/files/${base}" ]] && printf '%s\n' "${home}/packages/${package}/files/${base}"
    for kind in templates regions
    do
      rendered=$(release:render "${root}" "${package}" "${kind}" "${base}")
      [[ -n "${rendered}" ]] && printf '%s\n' "${rendered}"
    done
  done
  return 0
}

# release:kind <source> <path> prints how the content is placed:
#
#   region   the source is under a package's regions/<base>/
#   merge    the destination is a merged .json file (see release:merged)
#   copy     anything else
#
# The region test looks only at the source's last packages/ segment, so a checkout under a directory named packages
# works the same as one anywhere else.
release:kind() {
  if [[ "${1##*/packages/}" =~ ^[^/]+(/[^/]+)?/regions/(repo|home)/ ]]
  then printf 'region\n'
  elif release:merged "$2"
  then printf 'merge\n'
  else printf 'copy\n'
  fi
}

# release:region-name <dest> prints the region's name: the path relative to the base's root.
release:region-name() {
  local home
  home=$(release:base-root home)
  if [[ "$1" == "${home}"/* ]]
  then printf '%s\n' "${1#"${home}"/}"
  else printf '%s\n' "$1"
  fi
}

# release:region-command <dest> prints the command that generates the region. Under the repository it prints
# nothing, because the bare directive means `kit region <name>`. Under the home directory it prints
# `kit region --home <name>`, which the bare form cannot express.
release:region-command() {
  local home
  home=$(release:base-root home)
  if [[ "$1" == "${home}"/* ]]
  then printf 'kit region --home %s\n' "${1#"${home}"/}"
  fi
}

# release:region-markers <dest> prints the region's two marker lines in the file's comment syntax, chosen by
# extension: an HTML comment, //, --, or #. The first line is the directive, `kit:generate <name>`, followed by
# the command in backticks when there is one. The second is the end marker, `kit:end <name>`. A kind of file with
# no comment syntax the kit knows is rejected earlier, by release:render-all, when it reads the package.
release:region-markers() {
  local open close name command
  case "${1##*.}" in
    (md | markdown | html | htm | xml | svg) open='<!-- ' close=' -->' ;;
    (js | mjs | cjs | ts | tsx | jsx | go | rs | c | h | cc | cpp | hpp | java | kt | swift | cs | scala)
      open='// ' close='' ;;
    (sql | lua | hs) open='-- ' close='' ;;
    (*) open='# ' close='' ;;
  esac
  name=$(release:region-name "$1")
  command=$(release:region-command "$1")
  printf '%skit:generate %s%s%s\n%skit:end %s%s\n' "${open}" "${name}" "${command:+ \`${command}\`}" "${close}" \
    "${open}" "${name}" "${close}"
}

# release:region-known <path> succeeds when the kit knows the file's comment syntax, so a region can be marked in
# it.
release:region-known() {
  case "${1##*/}" in
    (*.json | *.jsonc | *.json5) return 1 ;;
    (*.*) case "${1##*.}" in
      (md | markdown | html | htm | xml | svg | js | mjs | cjs | ts | tsx | jsx | go | rs | c | h | cc | cpp | hpp \
        | java | kt | swift | cs | scala | sql | lua | hs | sh | bash | zsh | yml | yaml | toml | ini | cfg | conf \
        | py | rb | pl | txt | gitignore | gitattributes | editorconfig | env | just | mk | Makefile) return 0 ;;
      (*) return 1 ;;
    esac ;;
    (*) return 0 ;;
  esac
}

# release:target <path> prints the file that a write to <path> changes: the path itself, or the target of a
# symlink there. A repository that keeps AGENTS.md or a settings file as a link into another directory keeps the
# link.
release:target() {
  if [[ -L "$1" ]]
  then readlink -f "$1"
  else printf '%s\n' "$1"
  fi
}

# release:region-has <dest> succeeds when the file contains the region's directive. A directive with no end
# marker fails (status 2) and names the file, because a write would drop everything after the directive.
release:region-has() {
  local path=$1 begin end
  [[ -f "${path}" ]] || return 1
  { read -r begin; read -r end; } < <(release:region-markers "$1")
  grep --quiet --line-regexp --fixed-strings "${begin}" "${path}" || return 1
  if ! grep --quiet --line-regexp --fixed-strings "${end}" "${path}"
  then
    error "${path}: the kit's region opens with '${begin}' and never closes; add '${end}' where it ends"
    return 2
  fi
}

# release:region-read <dest> prints the region's content, between its directive and its end marker. It fails
# when the file or the region is missing.
release:region-read() {
  local path=$1 begin end
  release:region-has "$1" || return $?
  { read -r begin; read -r end; } < <(release:region-markers "$1")
  awk -v b="${begin}" -v e="${end}" '$0 == e { on = 0 } on { print } $0 == b { on = 1 }' "${path}"
}

# release:region-write <dest> <content-file> sets the region's content to <content-file>, the output of the
# region's command. It writes the content:
#
#   - between the directive and the end marker, when the file has them
#   - after a blank line at the end of the file, when the file has no region
#   - as the whole file, when there is no file
release:region-write() {
  local path content=$2 begin end status=0
  path=$(release:target "$1")
  { read -r begin; read -r end; } < <(release:region-markers "$1")
  mkdir -p "$(dirname "${path}")"
  release:region-has "${path}" || status=$?
  (( status == 2 )) && return 1
  if (( status == 0 ))
  then
    awk -v b="${begin}" -v e="${end}" -v c="${content}" '
      $0 == b { print; while ((getline line < c) > 0) print line; skip = 1; next }
      $0 == e { skip = 0 }
      !skip { print }' "${path}" > "${path}.kit-new"
  else
    {
      if [[ -s "${path}" ]]
      then
        cat "${path}"
        [[ "$(tail -c 1 "${path}")" == "" ]] || printf '\n'
        printf '\n'
      fi
      printf '%s\n' "${begin}"
      cat "${content}"
      printf '%s\n' "${end}"
    } > "${path}.kit-new"
  fi
  mv "${path}.kit-new" "${path}"
}

# release:region-remove <dest> removes the region, its directive, its end marker, and the blank line before it.
# A file left empty is deleted.
release:region-remove() {
  local path begin end
  path=$(release:target "$1")
  [[ -f "${path}" ]] || return 0
  { read -r begin; read -r end; } < <(release:region-markers "$1")
  awk -v b="${begin}" -v e="${end}" '
    $0 == b { skip = 1; held = 0; next }
    $0 == e { skip = 0; next }
    skip { next }
    held && printed { print ""; held = 0 }
    $0 == "" { held = 1; next }
    { print; printed = 1 }' "${path}" > "${path}.kit-new"
  if [[ -s "${path}.kit-new" ]]
  then mv "${path}.kit-new" "${path}"
  else rm -f "${path}.kit-new" "${path}"
  fi
}

# release:delivered <root> <base> <package>… prints "<path>\t<source>" for every path under the base's root that
# the release at <root> delivers for those packages, sorted by path. When layers deliver the same path, the last
# layer wins. A symlink counts as a source only when it points at a file. A dangling symlink fails here, where it
# is found.
release:delivered() {
  local layer path src
  while IFS= read -r layer
  do
    (cd "${layer}" && find . \( -type f -o -type l \) | sed 's@^\./@@' \
      | awk -v l="${layer}" '{ print $0 "\t" l "/" $0 }')
  done < <(release:layers "$@") | awk -F '\t' '{ src[$1] = $2 } END { for (p in src) print p "\t" src[p] }' \
    | LC_ALL=C sort \
    | while IFS=$'\t' read -r path src
  do
    if [[ -L "${src}" && ! -e "${src}" ]]
    then
      error "${src} is a link to nothing; the release is broken"
      return 1
    fi
    printf '%s\t%s\n' "${path}" "${src}"
  done
}

# release:source-of <root> <base> <path> <package>… prints the release file behind one delivered path, or nothing.
release:source-of() {
  local root=$1 base=$2 path=$3
  shift 3
  release:delivered "${root}" "${base}" "$@" | awk -F '\t' -v p="${path}" '$1 == p { print $2 }'
}

# release:paths-beyond <root> <base> <before> <after> prints the paths under the base that the package set <after>
# delivers and <before> does not. Each set is passed as one space-separated word.
release:paths-beyond() {
  local root=$1 base=$2
  local -a before after
  read -r -a before <<<"$3"
  read -r -a after <<<"$4"
  LC_ALL=C comm -13 <(release:delivered "${root}" "${base}" "${before[@]}" | cut -f 1) \
    <(release:delivered "${root}" "${base}" "${after[@]}" | cut -f 1)
}

# release:merged <path> succeeds for a destination that is merged by key instead of copied: a .json settings
# file. Every .json file the kit delivers is a settings file, except a schema (*.schema.json).
release:merged() { [[ "$1" == *.json && "$1" != *.schema.json ]]; }

# release:fetch <version> prints the root of that release, fetching it if needed. It tries, in order:
#
#   1. the kit's cache
#   2. a copy installed by mise, when mise is present
#   3. the release's public tarball, downloaded with curl and unpacked into the cache
#
# The version comes from a manifest, so it must have a version's shape before it is used in a path.
release:fetch() {
  local version=$1 root cache fresh
  if [[ ! "${version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]
  then
    error "'${version}' is not a kit version"
    return 1
  fi
  cache="${XDG_CACHE_HOME:-${HOME}/.cache}/kit/kit-${version}"
  if [[ -x "${cache}/bin/kit" ]]
  then
    printf '%s\n' "${cache}"
    return 0
  fi
  if command -v mise >/dev/null 2>&1
  then
    root=$(mise where "github:empowerite/kit@${version}" 2>/dev/null || true)
    if [[ -x "${root}/bin/kit" ]]
    then
      printf '%s\n' "${root}"
      return 0
    fi
  fi
  step "Fetching kit ${version} from its release"
  fresh=$(mktemp -d)
  if ! curl --fail --silent --location \
      "https://github.com/empowerite/kit/releases/download/release/${version}/kit-${version}.tar.gz" 2>/dev/null \
    | tar --extract --gzip --directory "${fresh}" --strip-components 1 2>/dev/null
  then
    rm -rf "${fresh}"
    error "kit ${version} could not be fetched from its release; set KIT_BASE to a copy of it"
    return 1
  fi
  mkdir -p "$(dirname "${cache}")"
  mv "${fresh}" "${cache}"
  printf '%s\n' "${cache}"
}

# release:among <name> <name>… succeeds when the first name is one of the rest.
release:among() {
  local needle=$1 item
  shift
  for item in "$@"
  do
    [[ "${item}" == "${needle}" ]] && return 0
  done
  return 1
}

# release:without <name> <name>… prints the rest without the first name, one per line.
release:without() {
  local needle=$1 item
  shift
  for item in "$@"
  do
    [[ "${item}" == "${needle}" ]] || printf '%s\n' "${item}"
  done
}

# The JSON merge functions, in jq:
#
#   merge($cur; $base; $new)   applies the release's $new over the current $cur, key by key. For an array, it keeps
#                              the current entries that were not in the previous release's $base, then adds $new's
#   unmerge($cur; $base)       removes $base's array entries from $cur, key by key; scalar values stay
#   holds($cur; $new)          true when $cur contains every key, array entry, and value in $new
#
# The parameters are values ($name), not filters, so each is evaluated once, at the right level.
release_jq='
  def merge($cur; $base; $new):
    if ($new | type) == "object" then
      reduce ($new | keys[]) as $k ($cur // {}; .[$k] = merge(($cur // {})[$k]; ($base // {})[$k]; $new[$k]))
    elif ($new | type) == "array" then
      ((($cur // []) - ($base // [])) as $kept | $kept + ($new - $kept))
    else $new end;
  def unmerge($cur; $base):
    if ($base | type) == "object" then
      reduce ($base | keys[]) as $k ($cur; if has($k) then .[$k] = unmerge($cur[$k]; $base[$k]) else . end)
    elif ($base | type) == "array" then ($cur // []) - $base
    else $cur end;
  def holds($cur; $new):
    if ($new | type) == "object" then
      all($new | keys[]; . as $k | ($cur | type) == "object" and ($cur | has($k)) and holds($cur[$k]; $new[$k]))
    elif ($new | type) == "array" then all($new[]; . as $e | any(($cur // [])[]; . == $e))
    else $cur == $new end;
'

# release:json-write <path> <filter> [<jq option>…] rewrites the JSON at <path> through <filter>. It creates the
# file, as {}, when it is missing.
release:json-write() {
  local path filter=$2
  path=$(release:target "$1")
  shift 2
  mkdir -p "$(dirname "${path}")"
  [[ -f "${path}" ]] || printf '{}\n' > "${path}"
  jq --indent 2 "${release_jq} ${filter}" "$@" "${path}" > "${path}.kit-new"
  mv "${path}.kit-new" "${path}"
}

# release:json-holds <path> <source> succeeds when the JSON at <path> contains every entry that the release's
# <source> writes.
release:json-holds() {
  jq --exit-status "${release_jq} holds(.; \$new[0])" --slurpfile new "$2" "$1" >/dev/null
}

# release:digest prints the sha256 of stdin, using sha256sum or shasum, whichever the machine has.
release:digest() {
  if command -v sha256sum >/dev/null 2>&1
  then
    sha256sum | cut -d ' ' -f 1
  else
    shasum -a 256 | cut -d ' ' -f 1
  fi
}

# release:place <source> <path> [<base-source>] writes the release's content to <path>. It merges a .json file,
# writes the region for a region source, and otherwise copies the file, keeping the source's mode.
release:place() {
  local src=$1 path=$2 base=${3:-/dev/null}
  case "$(release:kind "${src}" "${path}")" in
    (merge)
      release:json-write "${path}" 'merge(.; $base[0]; $new[0])' --slurpfile new "${src}" --slurpfile base "${base}"
      ;;
    (region)
      release:region-write "${path}" "${src}"
      ;;
    (*)
      path=$(release:target "${path}")
      mkdir -p "$(dirname "${path}")"
      cp "${src}" "${path}.kit-new"
      mv "${path}.kit-new" "${path}"
  esac
}

# release:installed <source> <path> prints the repository's copy of the release's content at <path>: the file,
# or the region's content. It prints nothing and fails when there is none.
release:installed() {
  local src=$1 path=$2
  case "$(release:kind "${src}" "${path}")" in
    (region) release:region-read "${path}" ;;
    (*) [[ -f "${path}" ]] && cat "${path}" ;;
  esac
}

# release:holds <source> <path> succeeds when the repository has the release's content at <path>: the same file,
# every merged entry, or the same region.
release:holds() {
  local src=$1 path=$2
  case "$(release:kind "${src}" "${path}")" in
    (merge) [[ -f "${path}" ]] && release:json-holds "${path}" "${src}" ;;
    (*) cmp --silent "${src}" <(release:installed "${src}" "${path}") ;;
  esac
}

# release:place-new <source> <path> [<label>] places content the repository has not recorded. It writes the
# content when:
#
#   - nothing is at <path>
#   - the destination is a .json file to merge into
#   - the file exists but has no region yet, for a region source
#
# When the same content is already there, it reports that and writes nothing. A region already there is
# regenerated. A different file fails, and is named and left as it is.
release:place-new() {
  local src=$1 path=$2 label=${3:-} kind
  kind=$(release:kind "${src}" "${path}")
  if [[ -e "${path}" && ! -f "${path}" ]]
  then
    error "${path}: a directory, or something that is not a file, is there; move it aside"
    return 1
  fi
  if [[ "${kind}" == merge || ! -f "${path}" ]] \
    || { [[ "${kind}" == region ]] && ! release:installed "${src}" "${path}" >/dev/null; }
  then
    step "${path}${label:+: ${label}}"
    release:place "${src}" "${path}"
  elif release:holds "${src}" "${path}"
  then
    step "${path}${label:+: ${label}}, already as the release has it"
  elif [[ "${kind}" == region ]]
  then
    step "${path}${label:+: ${label}}, the region regenerated"
    release:place "${src}" "${path}"
  else
    error "${path}: a different file is there already; move it aside, or remove it if it is not yours"
    return 1
  fi
}

# release:place-all <root> <base> <package>… places every path the packages deliver under the base. A path where a
# different file exists fails. It returns the number of failures.
release:place-all() {
  local root=$1 base=$2 path src refused=0
  shift 2
  while IFS=$'\t' read -r path src
  do
    [[ -n "${path}" ]] || continue
    release:place-new "${src}" "$(release:dest "${base}" "${path}")" || refused=$(( refused + 1 ))
  done < <(release:delivered "${root}" "${base}" "$@")
  return "${refused}"
}

# release:unplace <source> <path> removes the release's content from <path>: the kit's entries from a .json
# file, the kit's region from a region file, and otherwise the file itself.
release:unplace() {
  local src=$1 path=$2
  case "$(release:kind "${src}" "${path}")" in
    (merge)
      [[ -f "${path}" ]] || return 0
      step "${path}: the kit's entries taken out"
      release:json-write "${path}" 'unmerge(.; $base[0])' --slurpfile base "${src}"
      ;;
    (region)
      step "${path}: the kit's region taken out"
      release:region-remove "${path}"
      ;;
    (*)
      step "removing ${path}"
      rm -f "${path}"
  esac
}

# release:retire <base-root> <base> <previous> <current> <package>… handles the paths that the base's manifest
# records and the current release no longer delivers. <base-root> is the base release (the previous version), or
# empty when it is not available. For each such path that still exists:
#
#   - a merged .json file: the previous release's entries are removed when the base release is available;
#     otherwise it warns and leaves the file as it is
#   - any other file: it is removed when its content still matches the digest the manifest records; otherwise it
#     warns that the file was edited and leaves it as it is
#
# Under home, whose manifest every repository shares, it touches only paths the previous release delivered for
# these packages, and nothing at all without the base release.
release:retire() {
  local base_root=$1 base=$2 previous=$3 current=$4 path dest previous_src candidates
  shift 4
  candidates=$(LC_ALL=C comm -23 <(release:manifest-paths "${base}" | LC_ALL=C sort) \
    <(release:delivered "${kit_root}" "${base}" "$@" | cut -f 1 | LC_ALL=C sort))
  if [[ "${base}" != repo ]]
  then
    [[ -n "${base_root}" ]] || return 0
    candidates=$(LC_ALL=C comm -12 <(printf '%s\n' "${candidates}" | LC_ALL=C sort) \
      <(release:delivered "${base_root}" "${base}" "$@" | cut -f 1 | LC_ALL=C sort))
  fi
  while IFS= read -r path
  do
    [[ -n "${path}" ]] || continue
    dest=$(release:dest "${base}" "${path}")
    [[ -e "${dest}" ]] || continue
    previous_src=""
    [[ -n "${base_root}" ]] && previous_src=$(release:source-of "${base_root}" "${base}" "${path}" "$@")
    if release:merged "${dest}"
    then
      if [[ -n "${previous_src}" ]]
      then
        step "${dest}: retired in ${current}, the previous release's entries taken out"
        release:json-write "${dest}" 'unmerge(.; $base[0])' --slurpfile base "${previous_src}"
      else
        warn "${dest}: retired in ${current}; without the base release its entries stay, and it is left as it was"
      fi
    elif [[ "$(release:recorded "${base}" "${path}")" \
      == "$(release:installed "${previous_src:-/dev/null}" "${dest}" | release:digest)" ]]
    then
      step "${dest}: retired in ${current}, removed"
      release:unplace "${previous_src:-/dev/null}" "${dest}"
    else
      warn "${dest}: retired in ${current}, but edited since ${previous}; it is left as it was"
    fi
  done <<<"${candidates}"
}

# release:write-manifest <base> <package>… writes the base's manifest: the version, the active packages, and the
# digest of the release's file for every path delivered under the base. The repository's manifest is always
# written. Home's manifest is shared by every repository on the machine. It is written when these packages deliver
# to home. When they do not, it drops the lines whose paths no longer exist, and removes the file once no line
# is left.
release:write-manifest() {
  local base=$1 excluded=$2 manifest path src delivered kept
  shift 2
  manifest=$(release:manifest "${base}")
  delivered=$(release:delivered "${kit_root}" "${base}" "$@")
  if [[ "${base}" != repo && -z "${delivered}" ]]
  then
    [[ -f "${manifest}" ]] || return 0
    kept=$(while IFS= read -r path
      do
        [[ -n "${path}" && -e "$(release:dest "${base}" "${path}")" ]] && grep -F "  ${path}" "${manifest}"
      done < <(release:manifest-paths "${base}"); true)
    if [[ -z "${kept}" ]]
    then
      rm -f "${manifest}"
    else
      { awk '!(length($1) == 64 && $0 ~ /^[0-9a-f]+  /)' "${manifest}"; printf '%s\n' "${kept}"; } > "${manifest}.kit-new"
      mv "${manifest}.kit-new" "${manifest}"
    fi
    return 0
  fi
  mkdir -p "$(dirname "${manifest}")"
  {
    printf '# .kit/manifest - written by kit, not by hand\n'
    printf 'version %s\n' "$(release:version)"
    if (( $# > 0 ))
    then printf 'packages %s\n' "$*"
    else printf 'packages\n'
    fi
    [[ -z "${excluded}" ]] || printf 'excludes %s\n' "${excluded}"
    while IFS=$'\t' read -r path src
    do
      # Use if, not &&: as the last command in the loop body, a false && list would end the script under errexit.
      if [[ -n "${path}" ]]
      then
        printf '%s  %s\n' "$(release:digest < "${src}")" "${path}"
      fi
    done <<<"${delivered}"
  } > "${manifest}"
}

# release:write-manifests <excluded> <package>… writes every base's manifest: the active packages, and the excluded
# ones as a space-separated list, which is empty when there are none.
release:write-manifests() {
  local excluded=$1 base
  shift
  for base in $(release:bases)
  do
    release:write-manifest "${base}" "${excluded}" "$@"
  done
}

# release:manifest-version prints the version the repository's manifest records.
release:manifest-version() { sed -n 's/^version //p' .kit/manifest; }

# release:manifest-packages prints the active packages the repository's manifest records, one per line.
release:manifest-packages() { sed -n 's/^packages *//p' .kit/manifest | tr ' ' '\n' | awk 'NF'; }

# release:manifest-excludes prints the packages the repository's manifest records as excluded, one per line: members
# of an active bundle that were removed, which kit update leaves out.
release:manifest-excludes() { sed -n 's/^excludes *//p' .kit/manifest | tr ' ' '\n' | awk 'NF'; }

# release:manifest-paths <base> prints every path the base's manifest records. A record line is a 64-character
# hex digest, two spaces, and the path, which can contain spaces.
release:manifest-paths() {
  local manifest
  manifest=$(release:manifest "$1")
  [[ -f "${manifest}" ]] || return 0
  awk 'length($1) == 64 && $0 ~ /^[0-9a-f]+  / { print substr($0, 67) }' "${manifest}"
}

# release:recorded <base> <path> prints the digest the base's manifest records for <path>, or nothing.
release:recorded() {
  local manifest
  manifest=$(release:manifest "$1")
  [[ -f "${manifest}" ]] || return 0
  awk -v p="$2" 'length($1) == 64 && substr($0, 67) == p { print $1 }' "${manifest}"
}

# release:require-manifest fails in a repository where the kit was never installed.
release:require-manifest() {
  if [[ ! -f .kit/manifest ]]
  then
    error "run: kit install (this repository carries no .kit/manifest)"
    return 1
  fi
}

# release:finish <package>… reports a finished install or update.
release:finish() {
  ok "kit $(release:version) is installed, packages: ${*:-none}; .kit/manifest is written."
  release:say-unmet "$@"
  step "Commit the delivered files. The workflow is in .kit/1st.md."
}

# release:say-unmet <package>… warns about each unmet expectation of the packages, as a step left to do.
release:say-unmet() {
  local line
  while IFS= read -r line
  do
    [[ -n "${line}" ]] && warn "left to do in ${line}"
  done < <(release:unmet "${kit_root}" "$@")
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
