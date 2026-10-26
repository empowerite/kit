# shellcheck shell=bash
# shellcheck disable=SC2016  # jq filters name jq variables with $, in single quotes on purpose
# shellcheck disable=SC2154  # kit_root is exported by bin/kit
# release.sh - the release's modules, the one merge rule, the templates, and the manifests (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A release is bin/ and modules/. Each modules/<name>/ is a module: module.yml, its metadata; files/<base>/,
# mirrored at the base's root; templates/<base>/, with @key@ rendered from the repository's answers, then
# mirrored the same way; and regions/<base>/, each file placed as a region the kit owns inside a file the
# repository owns, between marker comments, so the rest of that file stays the repository's. A repository
# defines modules of its own the same way under .kit/modules/<name>/, and a name is looked up in the release
# first, then there. Two bases: repo, the repository root, recorded in .kit/manifest; and home, the user's home
# directory, recorded in ~/.kit/manifest, which every repository on the desk shares. A destination ending in
# .json is merged by key, the kit owning only the entries it writes, and a region is held to the release between
# its markers and nowhere else. The repository's choices live in .kit/settings.yml, which the kit reads and never
# writes. docs/modules.md is the format's description.
#
# Every function reads a release from a root, kit_root by default, which bin/kit exports. bin/kit also moves to
# the repository root, so .kit/manifest and every repo path here are relative to it. Every sort and comm here
# collates in the C locale, so an order written on one machine is the order read on another.

use yaml

# release:unreleased prints the version of a checkout: the release workflow stamps a version file, a checkout
# has none, and this is the one version kit update never fetches.
release:unreleased() { printf 'unreleased\n'; }

# release:version prints this release's version.
release:version() {
  cat "${kit_root}/version" 2>/dev/null || release:unreleased
}

# release:bases prints the destinations a module may deliver to, the subdirectories of files/ and templates/.
release:bases() { printf 'repo\nhome\n'; }

# release:base-root <base> prints the directory the base's files are mirrored at: the repository root, which the
# verbs run from, or the home directory, KIT_HOME when set so a test can stand in for it.
release:base-root() {
  case "$1" in
    (repo) printf '.\n' ;;
    (home) printf '%s\n' "${KIT_HOME:-${HOME}}" ;;
    (*)
      error "'$1' is not a base; the bases are: $(release:bases | paste -s -d ' ' -)"
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

# release_local is where a repository keeps modules of its own: .kit/modules/<name>/, in the release's shape.
declare -gr release_local=.kit

# release:settings prints the repository's settings file, which the kit reads and never writes.
release:settings() { printf '.kit/settings.yml\n'; }

# release:setting <key> prints the value of one top-level setting, or nothing.
release:setting() { yaml:value "$(release:settings)" "$1"; }

# release:modules [<root>] prints the modules in reach: every modules/<name>/ with a module.yml under the release
# at <root>, and under .kit/modules/ in the repository.
release:modules() {
  local root=${1:-${kit_root}} dir
  for dir in "${root}" "${release_local}"
  do
    [[ -d "${dir}/modules" ]] || continue
    (cd "${dir}/modules" && find . -mindepth 2 -maxdepth 2 -name module.yml | sed 's@^\./@@; s@/module.yml$@@')
  done | LC_ALL=C sort --unique
}

# release:home <root> <module> prints where the module lives: the release at <root> when it carries the module,
# else the repository's own .kit, so a path under it is <home>/modules/<module>/.
release:home() {
  if [[ -f "$1/modules/$2/module.yml" || ! -f "${release_local}/modules/$2/module.yml" ]]
  then printf '%s\n' "$1"
  else printf '%s\n' "${release_local}"
  fi
}

# release:origin <root> <module> prints "release" or "repository", where the module comes from.
release:origin() {
  if [[ "$(release:home "$1" "$2")" == "$1" ]]
  then printf 'release\n'
  else printf 'repository\n'
  fi
}

# release:meta <root> <module> prints the path of the module's metadata.
release:meta() { printf '%s/modules/%s/module.yml\n' "$(release:home "$1" "$2")" "$2"; }

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
      error "'${module}' is not a module in reach; the modules are:" \
        "$(release:modules "${root}" | paste -s -d ' ' -)"
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
  (( $# > 0 )) || return 0
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
  readarray -t names < <(printf '%s\n' "${!wanted[@]}" | LC_ALL=C sort)
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

# release:validate <root> <module> refuses metadata the schema would refuse, as far as bash can tell without a
# validator: a key the kit does not know, a default that is not true or false, a dependency on a module the
# release lacks. The gate validates against modules/module.schema.json in full.
release:validate() {
  local root=$1 module=$2 meta key value dep
  meta=$(release:meta "${root}" "${module}")
  for key in $(yaml:keys "${meta}")
  do
    case "${key}" in
      (description | default | depends | answers) ;;
      (*)
        error "$(release:meta "${root}" "${module}") has a key the kit does not know: ${key} (see modules/module.schema.json)"
        return 1
    esac
  done
  if [[ -z "$(yaml:value "${meta}" description)" ]]
  then
    error "$(release:meta "${root}" "${module}") has no description"
    return 1
  fi
  value=$(yaml:value "${meta}" default)
  if [[ -n "${value}" && "${value}" != true && "${value}" != false ]]
  then
    error "$(release:meta "${root}" "${module}"): default must be true or false, not '${value}'"
    return 1
  fi
  for dep in $(release:depends "${root}" "${module}")
  do
    if [[ ! -f "$(release:meta "${root}" "${dep}")" ]]
    then
      error "$(release:meta "${root}" "${module}") depends on '${dep}', which is not in reach"
      return 1
    fi
  done
}

# release:answers <root> <module> prints "<key>\t<value>" for every answer the module's templates may use: the
# module's defaults under its answers:, each overridden by the repository's, under answers: in .kit/settings.yml.
release:answers() {
  {
    yaml:map "$(release:meta "$1" "$2")" answers
    yaml:map "$(release:settings)" answers
  } | awk -F '\t' '{ a[$1] = $2 } END { for (k in a) print k "\t" a[k] }'
}

# release:tmp prints a directory for this process's rendered templates, made once and removed at exit. The
# rendering happens in release:render-all, which every verb runs before it reads a release.
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
  local root=$1 module=$2 base=$3 templates out key value text file rel
  local -A answer=()
  templates="$(release:home "${root}" "${module}")/modules/${module}/templates/${base}"
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
      for key in $(grep --only-matching '@[a-z_]*@' "${file}" | LC_ALL=C sort --unique | tr -d '@')
      do
        if [[ -z "${answer[${key}]+x}" ]]
        then
          error "${rel} needs an answer for '${key}'; add '${key}: <value>' under answers: in .kit/settings.yml (module ${module})"
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

# release:render-all <root> <module>… makes the release at <root> ready to read for the named modules: it refuses
# a module the release lacks, validates every module the release carries, refuses a base the kit does not deliver
# to, renders the named modules' templates, and refuses a link to nothing among the delivered files.
release:render-all() {
  local root=$1 module kind base
  shift
  release:require-modules "${root}" "$@" || return 2
  for module in $(release:modules "${root}")
  do
    release:validate "${root}" "${module}" || return 1
  done
  for module in "$@"
  do
    for kind in files templates regions
    do
      for base in $(cd "$(release:home "${root}" "${module}")/modules/${module}/${kind}" 2>/dev/null \
        && find . -mindepth 1 -maxdepth 1 -type d | sed 's@^\./@@')
      do
        # shellcheck disable=SC2046  # the bases are single words
        if ! release:among "${base}" $(release:bases)
        then
          error "modules/${module}/${kind}/${base} names a base the kit does not deliver to; the bases are:" \
            "$(release:bases | paste -s -d ' ' -)"
          return 1
        fi
      done
    done
    for base in $(release:bases)
    do
      release:render "${root}" "${module}" "${base}" >/dev/null || return 1
    done
  done
  for base in $(release:bases)
  do
    release:delivered "${root}" "${base}" "$@" >/dev/null || return 1
  done
}

# release:layers <root> <base> <module>… prints the directories mirrored at the base's root, in module order: each
# module's files/<base>, its rendered templates/<base>, and its regions/<base>.
release:layers() {
  local root=$1 base=$2 module home rendered
  shift 2
  for module in "$@"
  do
    home=$(release:home "${root}" "${module}")
    [[ -d "${home}/modules/${module}/files/${base}" ]] && printf '%s\n' "${home}/modules/${module}/files/${base}"
    rendered=$(release:render "${root}" "${module}" "${base}")
    [[ -n "${rendered}" ]] && printf '%s\n' "${rendered}"
    [[ -d "${home}/modules/${module}/regions/${base}" ]] && printf '%s\n' "${home}/modules/${module}/regions/${base}"
  done
  return 0
}

# release:kind <source> <path> prints how the content is placed: merge for a .json destination, region for a
# source under a module's regions/, copy otherwise.
release:kind() {
  if release:merged "$2"
  then printf 'merge\n'
  elif [[ "$1" == */modules/*/regions/* ]]
  then printf 'region\n'
  else printf 'copy\n'
  fi
}

# release:region-module <source> prints the module a region source belongs to, which names its markers.
release:region-module() {
  local rest=${1#*/modules/}
  printf '%s\n' "${rest%%/*}"
}

# release:region-markers <path> <module> prints the region's two marker lines, in the file's own comment
# syntax: an HTML comment for markdown, a hash comment otherwise.
release:region-markers() {
  if [[ "$1" == *.md ]]
  then printf '<!-- kit:begin %s -->\n<!-- kit:end %s -->\n' "$2" "$2"
  else printf '# kit:begin %s\n# kit:end %s\n' "$2" "$2"
  fi
}

# release:region-read <path> <module> prints the region's content, between its markers, and fails where the
# file or the region is absent.
release:region-read() {
  local path=$1 begin end
  [[ -f "${path}" ]] || return 1
  { read -r begin; read -r end; } < <(release:region-markers "$1" "$2")
  grep --quiet --line-regexp --fixed-strings "${begin}" "${path}" || return 1
  awk -v b="${begin}" -v e="${end}" '$0 == e { on = 0 } on { print } $0 == b { on = 1 }' "${path}"
}

# release:region-write <path> <module> <content-file> replaces the region's content, appending the region after a
# blank line where the file has none, and making the file where there is none.
release:region-write() {
  local path=$1 module=$2 content=$3 begin end
  { read -r begin; read -r end; } < <(release:region-markers "${path}" "${module}")
  mkdir -p "$(dirname "${path}")"
  if [[ -f "${path}" ]] && grep --quiet --line-regexp --fixed-strings "${begin}" "${path}"
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

# release:region-remove <path> <module> takes the region and its markers out, and the blank line before it;
# a file left empty is removed.
release:region-remove() {
  local path=$1 begin end
  [[ -f "${path}" ]] || return 0
  { read -r begin; read -r end; } < <(release:region-markers "$1" "$2")
  awk -v b="${begin}" -v e="${end}" '
    $0 == b { skip = 1; held = 0; next }
    $0 == e { skip = 0; next }
    skip { next }
    held { print ""; held = 0 }
    $0 == "" { held = 1; next }
    { print }' "${path}" > "${path}.kit-new"
  if [[ -s "${path}.kit-new" ]]
  then mv "${path}.kit-new" "${path}"
  else rm -f "${path}.kit-new" "${path}"
  fi
}

# release:delivered <root> <base> <module>… prints "<path>\t<source>" for every path under the base's root that
# the release at <root> delivers for those modules, sorted by path, the last layer winning. A symlink is a source
# only when it points at a file; a dangling one is refused here, where it is found.
release:delivered() {
  local layer path src
  while IFS= read -r layer
  do
    (cd "${layer}" && find . \( -type f -o -type l \) | sed 's@^\./@@' \
      | awk -v l="${layer}" '{ print $0 "\t" l "/" $0 }')
  done < <(release:layers "$@") | awk -F '\t' '{ src[$1] = $2 } END { for (p in src) print p "\t" src[p] }' | LC_ALL=C sort \
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
  LC_ALL=C comm -13 <(release:delivered "${root}" "${base}" "${before[@]}" | cut -f 1) \
    <(release:delivered "${root}" "${base}" "${after[@]}" | cut -f 1)
}

# release:merged <path> is true for a destination that is merged by key rather than copied: a .json file.
release:merged() { [[ "$1" == *.json ]]; }

# release:fetch <version> prints the root of that release, fetching it if need be: the kit's cache first, then a
# mise install if mise is here, then the release's public tarball with curl, unpacked into the cache. The version
# comes from a manifest, so it is held to a version's shape before it is used in a path.
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
      "https://github.com/empowerite/kit/releases/download/v${version}/kit-${version}.tar.gz" 2>/dev/null \
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

# The merge, as jq. The parameters are values ($), not filters, so each is read once at the right level.
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

# release:json-write <path> <filter> [<jq option>…] rewrites the JSON at <path> through <filter>, making it if
# missing.
release:json-write() {
  local path=$1 filter=$2
  shift 2
  mkdir -p "$(dirname "${path}")"
  [[ -f "${path}" ]] || printf '{}\n' > "${path}"
  jq --indent 2 "${release_jq} ${filter}" "$@" "${path}" > "${path}.kit-new"
  mv "${path}.kit-new" "${path}"
}

# release:json-holds <path> <source> is true when the JSON at <path> carries every entry the release's <source>
# writes.
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

# release:place <source> <path> [<base-source>] puts the release's content at <path>: a merge for .json, the
# region for a region source, else a copy that keeps the source's mode.
release:place() {
  local src=$1 path=$2 base=${3:-/dev/null}
  case "$(release:kind "${src}" "${path}")" in
    (merge)
      release:json-write "${path}" 'merge(.; $base[0]; $new[0])' --slurpfile new "${src}" --slurpfile base "${base}"
      ;;
    (region)
      release:region-write "${path}" "$(release:region-module "${src}")" "${src}"
      ;;
    (*)
      mkdir -p "$(dirname "${path}")"
      cp "${src}" "${path}.kit-new"
      mv "${path}.kit-new" "${path}"
  esac
}

# release:installed <source> <path> prints what the repository holds of the release's content at <path>: the
# file, or the region's content; nothing, failing, where there is none.
release:installed() {
  local src=$1 path=$2
  case "$(release:kind "${src}" "${path}")" in
    (region) release:region-read "${path}" "$(release:region-module "${src}")" ;;
    (*) [[ -f "${path}" ]] && cat "${path}" ;;
  esac
}

# release:holds <source> <path> is true when the repository holds the release's content at <path>: the same
# file, every merged entry, or the same region.
release:holds() {
  local src=$1 path=$2
  case "$(release:kind "${src}" "${path}")" in
    (merge) [[ -f "${path}" ]] && release:json-holds "${path}" "${src}" ;;
    (*) cmp --silent "${src}" <(release:installed "${src}" "${path}") ;;
  esac
}

# release:place-new <source> <path> [<label>] places content the repository did not record: where there is
# nothing, or the same thing, a .json to merge into, or a file to carry a region. A different file, or a region
# with different content, is refused, named, and left as it is.
release:place-new() {
  local src=$1 path=$2 label=${3:-} kind
  kind=$(release:kind "${src}" "${path}")
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
    error "${path}: the kit's region is there with different content; make it the release's, or remove it"
    return 1
  else
    error "${path}: a different file is there already; move it aside, or remove it if it is not yours"
    return 1
  fi
}

# release:place-all <root> <base> <module>… places every path the modules deliver under the base, refusing where
# a different file is. It returns the number of refusals.
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

# release:unplace <source> <path> takes the release's content back out of <path>: the kit's entries for .json,
# the kit's region for a region, the file itself otherwise.
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
      release:region-remove "${path}" "$(release:region-module "${src}")"
      ;;
    (*)
      step "removing ${path}"
      rm -f "${path}"
  esac
}

# release:retire <base-root> <base> <previous> <current> <module>… handles the paths the base's manifest records
# that the current release no longer delivers: removed when still the previous release's own content, their
# entries taken out of a merged .json when the base release is at hand, left with a warning otherwise. Under
# home, whose manifest every repository shares, only paths the previous release delivered for these modules are
# touched, and none without the base release.
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

# release:write-manifest <base> <module>… records, in the base's manifest, the version, the active modules, and
# the release's digest of every path delivered under the base. The repository's manifest is always written.
# Home's is shared by every repository on the desk: it is written when these modules deliver there, and when they
# do not, the lines whose paths are gone are dropped and the file is removed once nothing is left.
release:write-manifest() {
  local base=$1 manifest path src delivered kept
  shift
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
      { sed -n '1,3p' "${manifest}"; printf '%s\n' "${kept}"; } > "${manifest}.kit-new"
      mv "${manifest}.kit-new" "${manifest}"
    fi
    return 0
  fi
  mkdir -p "$(dirname "${manifest}")"
  {
    printf '# .kit/manifest - written by kit, not by hand\n'
    printf 'version %s\n' "$(release:version)"
    if (( $# > 0 ))
    then printf 'modules %s\n' "$*"
    else printf 'modules\n'
    fi
    while IFS=$'\t' read -r path src
    do
      # An if, not an && list: as the loop's last command, a false list would end the script under errexit.
      if [[ -n "${path}" ]]
      then
        printf '%s  %s\n' "$(release:digest < "${src}")" "${path}"
      fi
    done <<<"${delivered}"
  } > "${manifest}"
}

# release:write-manifests <module>… writes every base's manifest.
release:write-manifests() {
  local base
  for base in $(release:bases)
  do
    release:write-manifest "${base}" "$@"
  done
}

# release:manifest-version prints the version the repository's manifest records.
release:manifest-version() { sed -n 's/^version //p' .kit/manifest; }

# release:manifest-modules prints the active modules the repository's manifest records, one per line.
release:manifest-modules() { sed -n 's/^modules *//p' .kit/manifest | tr ' ' '\n' | awk 'NF'; }

# release:manifest-paths <base> prints every path the base's manifest records. A record line is a 64-digit
# digest, two spaces, and the path, which may itself hold spaces.
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
