# shellcheck shell=bash
# render.sh - renders a package's templates and regions once per process, with the repository's answers (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# Each packages/<name>/ delivers from files/<base>/, copied as they are; templates/<base>/, whose @key@ placeholders
# are filled from the repository's settings; and regions/<base>/, rendered the same way and placed as regions. A
# render keeps the packages/<package>/<kind>/<base> layout under a temporary directory, so a rendered region is
# still recognized as one, and render:layers lists the directories a base's files come from, in package order.
#
# see: docs/packages.md

use release resolve settings sources place region file

# render_rendered maps "<root>|<package>|<kind>|<base>" to the directory render:package rendered it into, or to
# nothing when the package has no such kind; render_root_digest caches a release root's digest; and
# render_rendered_all names the root and packages render:all last rendered whole, in this process.
declare -gA render_rendered=() render_root_digest=()
declare -g render_rendered_all=''

# render:tmp sets render_tmp, the directory for this process's rendered templates, on first use; file:sweep removes it
# at exit. It prints nothing, and a caller reads render_tmp, so the directory is made in this shell, where a subshell's
# exit cannot remove it. render:all does the rendering; every verb runs it before reading a release.
render:tmp() {
  [[ -n "${render_tmp:-}" ]] || file:scratch render_tmp
}

# render:package <root> <package> <kind> <base> renders the package's <kind> (templates or regions) for <base>, once
# per process, and prints the directory that contains the result. It prints nothing when the package has none. The
# directory keeps the packages/<package>/<kind>/<base> layout, so a rendered region is still recognized as a region.
#
# A placeholder is written @key@ (autoconf's form), which cannot be confused with bash's ${var} or GitHub's
# ${{ }}. A key can be dotted: @answers.key@. A key with no value fails, and the error names the key.
render:package() {
  local root=$1 package=$2 kind=$3 base=$4 source out key value text file rel map memo="$1|$2|$3|$4"
  local -A answer=()
  # A render already done in this process, or in the parent of this subshell, answers from render_rendered, with
  # no fork: render:layers asks for every package's layers each time a path is resolved.
  if [[ -n "${render_rendered[${memo}]+x}" ]]
  then
    [[ -z "${render_rendered[${memo}]}" ]] || printf '%s\n' "${render_rendered[${memo}]}"
    return 0
  fi
  source="$(resolve:home "${root}" "${package}")/packages/${package}/${kind}/${base}"
  if [[ ! -d "${source}" ]]
  then
    render_rendered[${memo}]=''
    return 0
  fi
  [[ -n "${render_root_digest[${root}]+x}" ]] \
    || render_root_digest[${root}]=$(printf '%s' "${root}" | file:digest)
  render:tmp || return 1
  out="${render_tmp}/${render_root_digest[${root}]}/packages/${package}/${kind}/${base}"
  if [[ ! -d "${out}" ]]
  then
    # Made before the loop, so a kind with a directory and no file left in it is still a layer that exists.
    mkdir -p "${out}"
    while IFS=$'\t' read -r key value
    do
      [[ -n "${key}" ]] && answer[${key}]=${value}
    done < <(settings:context "${root}" "${package}")
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
          error "${rel} needs an answer for '${key}'; add '${key##*.}: <value>' under ${map}: in kit.yml" \
            "(package ${package})"
          return 1
        fi
        text=${text//"@${key}@"/"${answer[${key}]}"}
      done
      mkdir -p "${out}/$(dirname "${rel}")"
      printf '%s\n' "${text}" > "${out}/${rel}"
    done < <(find "${source}" -type f)
  fi
  render_rendered[${memo}]=${out}
  printf '%s\n' "${out}"
}

# render:all <root> <package>… prepares the release at <root> for the named packages. In order, it:
#
#   1. validates the settings
#   2. fails on a named package that is not in reach
#   3. validates every package in reach
#   4. renders the named packages, as render:packages does
render:all() {
  local root=$1 package memo="$*"
  shift
  # Every verb renders once, and a module that renders again for the same packages in the same process, as
  # generate:region does inside kit check, finds it done.
  [[ "${render_rendered_all}" == "${memo}" ]] && return 0
  settings:validate || return 1
  # Every source must be fetched, its digest checked, before the packages are rendered.
  sources:fetch-all || return 1
  resolve:require "${root}" "$@" || return 2
  for package in $(resolve:reach "${root}")
  do
    resolve:validate "${root}" "${package}" || return 1
  done
  render:packages "${root}" "$@" || return $?
  render_rendered_all=${memo}
}

# render:packages <root> <package>… renders the named packages' templates and regions. It fails on a base
# the kit does not deliver to, on a delivered file that is a link to nothing, and on regions that
# render:regions-apart rejects. A package the release at <root> does not have renders nothing, so kit update can
# read a base release that predates a package: that release delivered nothing for it.
render:packages() {
  local root=$1 package kind base
  shift
  for package in "$@"
  do
    for kind in files templates regions hosts
    do
      for base in $(cd "$(resolve:home "${root}" "${package}")/packages/${package}/${kind}" 2>/dev/null \
        && find . -mindepth 1 -maxdepth 1 -type d | sed 's@^\./@@')
      do
        # shellcheck disable=SC2046  # the bases are single words
        if ! list:among "${base}" $(release:bases)
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
        render:package "${root}" "${package}" "${kind}" "${base}" >/dev/null || return 1
      done
    done
  done
  for base in $(release:bases)
  do
    place:delivered "${root}" "${base}" "$@" >/dev/null || return 1
    render:regions-apart "${root}" "${base}" "$@" || return 1
  done
}

# render:regions-apart <root> <base> <package>… fails when two of the packages deliver a region to the same path (a
# file can have only one kit region), or when a region is in a kind of file whose comment syntax the kit does not
# know.
render:regions-apart() {
  local root=$1 base=$2 package home file rel owner
  shift 2
  local -A owner_of=()
  for package in "$@"
  do
    home=$(resolve:home "${root}" "${package}")
    [[ -d "${home}/packages/${package}/regions/${base}" ]] || continue
    while IFS= read -r file
    do
      rel=${file#"${home}/packages/${package}/regions/${base}/"}
      if ! region:known "${rel}"
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

# render:layers <root> <base> <package>… prints the directories copied to the base's root, in package order: each
# package's files/<base>, its rendered templates/<base>, and its rendered regions/<base>.
render:layers() {
  local root=$1 base=$2 package home kind rendered
  shift 2
  for package in "$@"
  do
    home=$(resolve:home "${root}" "${package}")
    [[ -d "${home}/packages/${package}/files/${base}" ]] && printf '%s\n' "${home}/packages/${package}/files/${base}"
    for kind in templates regions
    do
      rendered=$(render:package "${root}" "${package}" "${kind}" "${base}")
      [[ -n "${rendered}" ]] && printf '%s\n' "${rendered}"
    done
  done
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
