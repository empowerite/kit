# shellcheck shell=bash
# shellcheck disable=SC2016  # jq filters name jq variables with $, in single quotes on purpose
# shellcheck disable=SC2154  # kit_root is exported by bin/kit
# release.sh - the release's modules, the one merge rule, the templates, and the manifest (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A release is bin/ and modules/. Each modules/<name>/ is a module: module.yml, its metadata; files/<base>/,
# mirrored at the base's root; and templates/<base>/, with @key@ rendered from the repository's answers, then
# mirrored the same way. Two bases: repo, the repository root, recorded in .kit/manifest; and home, the user's
# home directory, recorded in ~/.kit/manifest, which is the desk's record rather than the repository's. A
# destination ending in .json is merged by key, with the kit's keys set, the kit's array entries added, and on
# update or removal only the previous release's entries taken out. .kit/manifest records the version, the active
# modules, and the release's digest for every delivered path. docs/modules.md is the format's description.
#
# Every function reads a release from a root, kit_root by default, which bin/kit exports.

use yaml

# release_bases are the destinations a module may deliver to, as the subdirectories of files/ and templates/.
declare -gra release_bases=(repo home)

# release:base-root <base> prints the directory the base's files are mirrored at: the repository root, which the
# verbs run from, or the home directory, KIT_HOME when set so a test can stand in for it.
release:base-root() {
  case "$1" in
    (repo) printf '.\n' ;;
    (home) printf '%s\n' "${KIT_HOME:-${HOME}}" ;;
    (*)
      error "'$1' is not a base; the bases are: ${release_bases[*]}"
      return 2
  esac
}

# release:dest <base> <path> prints where a delivered path lands.
release:dest() {
  if [[ "$1" == repo ]]
  then printf '%s\n' "$2"
  else printf '%s/%s\n' "$(release:base-root "$1")" "$2"
  fi
}

# release:manifest <base> prints the base's manifest: .kit/manifest in the repository, ~/.kit/manifest for home.
release:manifest() { release:dest "$1" .kit/manifest; }

# A release carries a version file the release workflow stamps from the tag; a checkout carries none and is
# "unreleased", the one version kit update never asks mise for.
declare -gr release_unreleased=unreleased

# release:version prints this release's version.
release:version() {
  cat "${kit_root}/version" 2>/dev/null || printf '%s\n' "${release_unreleased}"
}

# release:modules [<root>] prints the modules the release carries: every modules/<name>/ with a module.yml.
release:modules() {
  (cd "${1:-${kit_root}}/modules" && find . -mindepth 2 -maxdepth 2 -name module.yml | sed 's@^\./@@; s@/module.yml$@@' | sort)
}

# release:meta <root> <module> prints the path of the module's metadata.
release:meta() { printf '%s/modules/%s/module.yml\n' "$1" "$2"; }

# release:description <root> <module> prints the module's one-line description.
release:description() { yaml:value "$(release:meta "$1" "$2")" description; }

# release:depends <root> <module> prints the modules this one depends on, one per line.
release:depends() { yaml:list "$(release:meta "$1" "$2")" depends; }

# release:default-modules <root> prints the modules active unless removed: those whose default is true.
release:default-modules() {
  local module
  for module in $(release:modules "$1")
  do
    [[ "$(yaml:value "$(release:meta "$1" "${module}")" default)" == true ]] && printf '%s\n' "${module}"
  done
  return 0
}

# release:require-modules <root> <name>… refuses a module the release does not carry, naming the ones it does.
release:require-modules() {
  local root=$1 module
  shift
  for module in "$@"
  do
    if [[ ! -f "$(release:meta "${root}" "${module}")" ]]
    then
      error "'${module}' is not a module this release knows; the modules are: $(release:modules "${root}" | paste -s -d ' ' -)"
      return 2
    fi
  done
}

# release:resolve <root> <name>… prints the named modules and everything they depend on, each once: a module
# after the modules it depends on, and alphabetical where that leaves a choice, so the order is the same on
# every machine. A cycle is refused.
release:resolve() {
  local root=$1 module dep ready placed_count=0
  shift
  local -A wanted=() placed=()
  local -a queue=("$@") names=()
  while (( ${#queue[@]} > 0 ))
  do
    module=${queue[0]}
    queue=("${queue[@]:1}")
    [[ -n "${wanted[${module}]:-}" ]] && continue
    wanted[${module}]=1
    for dep in $(release:depends "${root}" "${module}")
    do
      queue+=("${dep}")
    done
  done
  readarray -t names < <(printf '%s\n' "${!wanted[@]}" | sort)
  while (( placed_count < ${#names[@]} ))
  do
    ready=''
    for module in "${names[@]}"
    do
      [[ -n "${placed[${module}]:-}" ]] && continue
      for dep in $(release:depends "${root}" "${module}")
      do
        [[ -n "${placed[${dep}]:-}" ]] || continue 2
      done
      ready=${module}
      break
    done
    if [[ -z "${ready}" ]]
    then
      error "the modules' depends lines form a cycle among: ${names[*]}"
      return 1
    fi
    placed[${ready}]=1
    placed_count=$(( placed_count + 1 ))
    printf '%s\n' "${ready}"
  done
}

# release:dependents <root> <module> <active>… prints the active modules that depend on <module>.
release:dependents() {
  local root=$1 module=$2 other dep
  shift 2
  for other in "$@"
  do
    for dep in $(release:depends "${root}" "${other}")
    do
      [[ "${dep}" == "${module}" ]] && printf '%s\n' "${other}"
    done
  done
}

# release:answers <root> <module> prints "<key>\t<value>" for every answer the module's templates may use: the
# module's defaults under answers:, each overridden by a "<key> <value>" line in the repository's .kit/answers.
release:answers() {
  {
    yaml:map "$(release:meta "$1" "$2")" answers | tr '\t' ' '
    [[ -f .kit/answers ]] && grep --invert-match '^#' .kit/answers
    true
  } | awk 'NF >= 2 { v = $0; sub(/^[^ ]+ +/, "", v); a[$1] = v } END { for (k in a) print k "\t" a[k] }'
}

# release:tmp prints a directory for this process's rendered templates, made once and removed at exit.
release:tmp() {
  if [[ -z "${release_tmp:-}" ]]
  then
    release_tmp=$(mktemp -d)
    # shellcheck disable=SC2064  # expanded now on purpose: the path is fixed from here on
    trap "rm -rf '${release_tmp}'" EXIT
  fi
  printf '%s\n' "${release_tmp}"
}

# release:render <root> <module> <base> renders the module's templates for <base> once per process and prints the
# directory holding them, or prints nothing where the module has none. A key is written @key@, autoconf's form,
# which neither bash's ${var} nor GitHub's ${{ }} can be mistaken for; one with no answer refuses, naming the key.
release:render() {
  local root=$1 module=$2 base=$3 templates="$1/modules/$2/templates/$3" out key value text file rel
  local -A answer=()
  [[ -d "${templates}" ]] || return 0
  out="$(release:tmp)/$(printf '%s' "${root}" | release:digest)/${module}/${base}"
  if [[ ! -d "${out}" ]]
  then
    while IFS=$'\t' read -r key value
    do
      [[ -n "${key}" ]] && answer[${key}]=${value}
    done < <(release:answers "${root}" "${module}")
    while IFS= read -r file
    do
      rel=${file#"${templates}"/}
      text=$(<"${file}")
      for key in $(grep --only-matching '@[a-z_]*@' "${file}" | sort --unique | tr -d '@')
      do
        if [[ -z "${answer[${key}]+x}" ]]
        then
          error "${rel} needs an answer for '${key}'; add a line '${key} <value>' to .kit/answers (module ${module})"
          return 1
        fi
        text=${text//"@${key}@"/"${answer[${key}]}"}
      done
      mkdir -p "${out}/$(dirname "${rel}")"
      printf '%s\n' "${text}" > "${out}/${rel}"
    done < <(find "${templates}" -type f)
  fi
  printf '%s\n' "${out}"
}

# release:render-all <root> <module>… renders every named module's templates for every base, and refuses a base
# the kit does not know, or an unanswered key, naming it.
release:render-all() {
  local root=$1 module kind base
  shift
  for module in "$@"
  do
    for kind in files templates
    do
      for base in $(cd "${root}/modules/${module}/${kind}" 2>/dev/null && find . -mindepth 1 -maxdepth 1 -type d | sed 's@^\./@@')
      do
        if ! release:among "${base}" "${release_bases[@]}"
        then
          error "modules/${module}/${kind}/${base} names a base the kit does not deliver to; the bases are: ${release_bases[*]}"
          return 1
        fi
      done
    done
    for base in "${release_bases[@]}"
    do
      release:render "${root}" "${module}" "${base}" >/dev/null || return 1
    done
  done
}

# release:layers <root> <base> <module>… prints the directories mirrored at the base's root, in module order: each
# module's files/<base>, then its rendered templates/<base>.
release:layers() {
  local root=$1 base=$2 module rendered
  shift 2
  for module in "$@"
  do
    [[ -d "${root}/modules/${module}/files/${base}" ]] && printf '%s\n' "${root}/modules/${module}/files/${base}"
    rendered=$(release:render "${root}" "${module}" "${base}")
    [[ -n "${rendered}" ]] && printf '%s\n' "${rendered}"
  done
  return 0
}

# release:delivered <root> <base> <module>… prints "<path>\t<source>" for every path under the base's root that
# the release at <root> delivers for those modules, sorted by path, the last layer winning. A symlink is a source
# only when it points at a file; a dangling one is refused here, where it is found.
release:delivered() {
  local layer path src
  while IFS= read -r layer
  do
    (cd "${layer}" && find . \( -type f -o -type l \) | sed 's@^\./@@' | awk -v l="${layer}" '{ print $0 "\t" l "/" $0 }')
  done < <(release:layers "$@") | awk -F '\t' '{ src[$1] = $2 } END { for (p in src) print p "\t" src[p] }' | sort \
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

# release:source-of <root> <base> <path> <module>… prints the release file behind one delivered path, or nothing.
release:source-of() {
  local root=$1 base=$2 path=$3
  shift 3
  release:delivered "${root}" "${base}" "$@" | awk -F '\t' -v p="${path}" '$1 == p { print $2 }'
}

# release:paths-beyond <root> <base> <before> <after> prints the paths the module set <after> delivers under the
# base that <before> does not; each set is one space-separated word.
release:paths-beyond() {
  local root=$1 base=$2
  local -a before after
  read -r -a before <<<"$3"
  read -r -a after <<<"$4"
  comm -13 <(release:delivered "${root}" "${base}" "${before[@]}" | cut -f 1) \
    <(release:delivered "${root}" "${base}" "${after[@]}" | cut -f 1)
}

release:merged() { [[ "$1" == *.json ]]; }

# release:fetch <version> prints the root of that release, fetching it if need be: the kit's cache first, then a
# mise install if mise is here, then the GitHub release itself, unpacked into the cache. Nothing here assumes how
# this release was installed.
release:fetch() {
  local version=$1 root cache fresh
  # The version comes from a manifest, which is repository content, so it is held to a version's shape before it
  # becomes part of a path; the unpack goes into a directory made here and is moved into place only when whole.
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
  step "Fetching kit ${version} from its GitHub release"
  fresh=$(mktemp -d)
  if ! gh release download "v${version}" --repo empowerite/kit --pattern "kit-${version}.tar.gz" --output - 2>/dev/null \
    | tar --extract --gzip --directory "${fresh}" --strip-components 1
  then
    rm -rf "${fresh}"
    error "kit ${version} could not be fetched (run: gh auth status, or set KIT_BASE to a copy of it)"
    return 1
  fi
  mkdir -p "$(dirname "${cache}")"
  mv "${fresh}" "${cache}"
  printf '%s\n' "${cache}"
}

# release:among <name> <name>… is true when the first name is among the rest.
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

# The parameters are values ($), not filters: a filter parameter is re-read against the input at each use, which
# inside the recursion is the wrong level of the object.
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

# release:json-write <path> <filter> [<jq option>…] rewrites the JSON at <path> through <filter>, making it if missing.
release:json-write() {
  local path=$1 filter=$2
  shift 2
  mkdir -p "$(dirname "${path}")"
  [[ -f "${path}" ]] || printf '{}\n' > "${path}"
  jq --indent 2 "${release_jq} ${filter}" "$@" "${path}" > "${path}.kit-new"
  mv "${path}.kit-new" "${path}"
}

# release:json-holds <path> <source> is true when the JSON at <path> carries every entry the release's <source> writes.
release:json-holds() {
  jq --exit-status "${release_jq} holds(.; \$new[0])" --slurpfile new "$2" "$1" >/dev/null
}

# release:digest prints the sha256 of stdin, by whichever tool the machine carries.
release:digest() {
  if command -v sha256sum >/dev/null 2>&1
  then
    sha256sum | cut -d ' ' -f 1
  else
    shasum -a 256 | cut -d ' ' -f 1
  fi
}

# release:place <source> <path> [<base-source>] puts the release's content at <path>: a merge for .json, else a copy.
release:place() {
  local src=$1 path=$2 base=${3:-/dev/null}
  if release:merged "${path}"
  then
    release:json-write "${path}" 'merge(.; $base[0]; $new[0])' --slurpfile new "${src}" --slurpfile base "${base}"
  else
    mkdir -p "$(dirname "${path}")"
    cat "${src}" > "${path}.kit-new"
    mv "${path}.kit-new" "${path}"
  fi
}

# release:unplace <source> <path> takes the release's content back out of <path>: the kit's entries for .json,
# the file itself otherwise.
release:unplace() {
  local src=$1 path=$2
  if release:merged "${path}"
  then
    [[ -f "${path}" ]] || return 0
    step "${path}: the kit's entries taken out"
    release:json-write "${path}" 'unmerge(.; $base[0])' --slurpfile base "${src}"
  else
    step "removing ${path}"
    rm -f "${path}"
  fi
}

# release:write-manifest <base> <module>… records, in the base's manifest, the version, the active modules, and
# the release's digest for every path delivered under the base. The repository's is always written; home's only
# when something is delivered there, and it is removed when nothing is any more.
release:write-manifest() {
  local base=$1 manifest path src delivered
  shift
  manifest=$(release:manifest "${base}")
  delivered=$(release:delivered "${kit_root}" "${base}" "$@")
  if [[ "${base}" != repo && -z "${delivered}" ]]
  then
    rm -f "${manifest}"
    return 0
  fi
  mkdir -p "$(dirname "${manifest}")"
  {
    printf '# .kit/manifest - written by kit, not by hand\n'
    printf 'version %s\n' "$(release:version)"
    printf 'modules%s\n' "${*:+ $*}"
    while IFS=$'\t' read -r path src
    do
      [[ -n "${path}" ]] && printf '%s  %s\n' "$(release:digest < "${src}")" "${path}"
    done <<<"${delivered}"
  } > "${manifest}"
}

# release:write-manifests <module>… writes every base's manifest.
release:write-manifests() {
  local base
  for base in "${release_bases[@]}"
  do
    release:write-manifest "${base}" "$@"
  done
}

release:manifest-version() { sed -n 's/^version //p' .kit/manifest; }
release:manifest-modules() { sed -n 's/^modules *//p' .kit/manifest; }

# release:recorded <base> <path> prints the digest the base's manifest records for <path>, or nothing.
release:recorded() {
  local manifest
  manifest=$(release:manifest "$1")
  [[ -f "${manifest}" ]] || return 0
  awk -v p="$2" '$2 == p { print $1 }' "${manifest}"
}

# release:require-manifest refuses to go on in a repository that never took the kit.
release:require-manifest() {
  if [[ ! -f .kit/manifest ]]
  then
    error "run: kit install (this repository carries no .kit/manifest)"
    return 1
  fi
}

# release:finish <module>… reports a completed install or update.
release:finish() {
  ok "kit $(release:version) is installed, modules: ${*:-none}; .kit/manifest is written."
  step "Commit the delivered files. The loop is in .kit/1st.md."
}

# ----------------------------------------------------------------------------------------------------------------------
