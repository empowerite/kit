# shellcheck shell=bash
# render.sh - renders a package's templates and regions once per process, with the repository's answers (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # resolve_home is set by resolve:locate, in modules/bash/resolve.sh
#
# Each packages/<name>/ delivers from files/<base>/, copied as they are; templates/<base>/, whose @key@ placeholders
# are filled from the repository's settings; and regions/<base>/, rendered the same way and placed as regions. A
# render keeps the packages/<package>/<kind>/<base> layout under a temporary directory, so a rendered region is
# still recognized as one, and render:layers lists the directories a base's files come from, in package order.
#
# see: docs/packages.md

use release resolve settings sources place region file

# render_rendered maps "<root>|<package>|<kind>|<base>" to the directory render:package rendered it into, or to
# nothing when the package has no such kind; render_out is the directory render:package last answered with;
# render_root_digest caches a release root's digest; render_rendered_all names the root and packages render:all
# last rendered whole, in this process; and render_holes holds the placeholders render:holes last found.
declare -gA render_rendered=() render_root_digest=()
declare -g render_rendered_all='' render_out=''
declare -ga render_holes=()

# render:tmp sets render_tmp, the directory for this process's rendered templates, on first use; file:sweep removes it
# at exit. It prints nothing, and a caller reads render_tmp, so the directory is made in this shell, where a subshell's
# exit cannot remove it. render:all does the rendering; every verb runs it before reading a release.
render:tmp() {
  [[ -n "${render_tmp:-}" ]] || file:scratch render_tmp
}

# render:package <root> <package> <kind> <base> renders the package's <kind> (templates or regions) for <base>, once
# per process, and sets render_out to the directory that contains the result, or to nothing when the package has none.
# It prints nothing, so a caller asks with no fork. The directory keeps the packages/<package>/<kind>/<base> layout, so
# a rendered region is still recognized as a region.
#
# A placeholder is written @key@ (autoconf's form), which cannot be confused with bash's ${var} or GitHub's
# ${{ }}. A key can be dotted: @answers.key@. A key with no value fails, and the error names the key.
render:package() {
  local root=$1 package=$2 kind=$3 base=$4 source out memo="$1|$2|$3|$4"
  local -a rows
  render_out=''
  # A render already done in this process, or in the parent of this subshell, answers from render_rendered:
  # render:layers asks for every package's layers each time a path is resolved.
  if [[ -n "${render_rendered[${memo}]+x}" ]]
  then
    render_out=${render_rendered[${memo}]}
    return 0
  fi
  resolve:locate "${root}" "${package}" || return 1
  source="${resolve_home}/packages/${package}/${kind}/${base}"
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
    list:read rows settings:context "${root}" "${package}" || return 1
    render:fill "${source}" "${out}" "${package}" "${rows[@]}" || return 1
  fi
  render_rendered[${memo}]=${out}
  render_out=${out}
}

# render:fill <source> <out> <package> <row>… renders every file under <source> into <out>, at its path below
# <source>, each @key@ placeholder filled from the rows, "<key>\t<value>" each, as settings:context prints them for
# <package>. A placeholder with no row fails, naming the key and the package. <out> is made before any file is read,
# so a kind with a directory and no file left in it is still a layer that exists. kit policy renders a package's
# templates as a branch's base delivers them through it, with the base's answers (policy:home-held).
render:fill() {
  local source=$1 out=$2 package=$3 row key value text file rel dir map
  local -A answer=()
  shift 3
  for row in "$@"
  do
    IFS=$'\t' read -r key value <<<"${row}"
    [[ -n "${key}" ]] && answer[${key}]=${value}
  done
  mkdir -p "${out}"
  while IFS= read -r file
  do
    rel=${file#"${source}"/}
    text=$(<"${file}")
    render:holes "${text}"
    for key in "${render_holes[@]}"
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
    dir=${out}
    [[ "${rel}" != */* ]] || dir+=/${rel%/*}
    [[ -d "${dir}" ]] || mkdir -p "${dir}"
    printf '%s\n' "${text}" > "${out}/${rel}"
  done < <(find "${source}" -type f)
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
  # Every source must be fetched, its digest checked, before the packages are rendered. Their roots are read here, in
  # this shell, so every subshell below inherits them: read in one, they are read again in the next.
  sources:fetch-all || return 1
  resolve:roots || return 1
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
  local root=$1 package kind base dir
  local -a bases
  shift
  list:read bases release:bases || return 1
  for package in "$@"
  do
    resolve:locate "${root}" "${package}" || return 1
    # Each directory under a kind is a base, hidden or not; a link is not one.
    for kind in files templates regions hosts
    do
      for dir in "${resolve_home}/packages/${package}/${kind}"/{*,.*}
      do
        base=${dir##*/}
        [[ "${base}" != . && "${base}" != .. && -d "${dir}" && ! -L "${dir}" ]] || continue
        if ! list:among "${base}" "${bases[@]}"
        then
          error "packages/${package}/${kind}/${base} names a base the kit does not deliver to; the bases are:" \
            "${bases[*]}"
          return 1
        fi
      done
    done
    for kind in templates regions
    do
      for base in "${bases[@]}"
      do
        render:package "${root}" "${package}" "${kind}" "${base}" || return 1
      done
    done
  done
  for base in "${bases[@]}"
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
    resolve:locate "${root}" "${package}" || return 1
    home=${resolve_home}
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

# render:holes <text> fills render_holes with the placeholders the text holds, each <key> of an @<key>@ once, in C
# order, the order render:fill fills them in, as grep --only-matching and sort --unique would find them, starting no
# process; none for a text with none.
render:holes() {
  local rest=$1 key i
  local -A seen=()
  local LC_ALL=C
  render_holes=()
  while [[ "${rest}" =~ @([a-z_][a-z_.]*)@ ]]
  do
    rest=${rest#*"${BASH_REMATCH[0]}"}
    key=${BASH_REMATCH[1]}
    [[ -z "${seen[${key}]:-}" ]] || continue
    seen[${key}]=1
    # an insertion, in the byte order of @<key>@, as sort orders them: a key that another starts with comes after it
    # where the other goes on in a character before @, as a.. before a; a template holds a handful
    i=${#render_holes[@]}
    while (( i > 0 )) && [[ "${key}@" < "${render_holes[i - 1]}@" ]]
    do
      render_holes[i]=${render_holes[i - 1]}
      i=$(( i - 1 ))
    done
    render_holes[i]=${key}
  done
}

# render:layers <root> <base> <package>… prints the directories copied to the base's root, in package order: each
# package's files/<base>, its rendered templates/<base>, and its rendered regions/<base>.
render:layers() {
  local root=$1 base=$2 package kind
  shift 2
  for package in "$@"
  do
    resolve:locate "${root}" "${package}" || return 1
    [[ -d "${resolve_home}/packages/${package}/files/${base}" ]] \
      && printf '%s\n' "${resolve_home}/packages/${package}/files/${base}"
    for kind in templates regions
    do
      render:package "${root}" "${package}" "${kind}" "${base}" || return 1
      [[ -n "${render_out}" ]] && printf '%s\n' "${render_out}"
    done
  done
  return 0
}

# ----------------------------------------------------------------------------------------------------------------------
