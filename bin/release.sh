# shellcheck shell=bash
# shellcheck disable=SC2016  # jq filters name jq variables with $, in single quotes on purpose
# shellcheck disable=SC2154  # kit_root is exported by bin/kit
# release.sh - the release's layout, the one merge rule, and the manifest (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A release mirrors its tree/ at the repository root, and targets/<name>/ for each target taken; a symlink in
# either is delivered as the file it points at. A destination ending in .json is merged by key, with the kit's
# keys set, the kit's array entries added, and on update or removal only the previous release's entries taken
# out. .kit/manifest records the version, the targets, and the release's digest for every delivered path.
#
# Every function reads the release from kit_root, exported by bin/kit.

# A release carries a version file the release workflow stamps from the tag; a checkout carries none and is
# "unreleased", the one version kit update never asks mise for.
declare -gr release_unreleased=unreleased

# release:version prints this release's version.
release:version() {
  cat "${kit_root}/version" 2>/dev/null || printf '%s\n' "${release_unreleased}"
}

# release:layers <root> <target>… prints the directories mirrored at the repository root, tree first.
release:layers() {
  local root=$1 t
  shift
  printf '%s\n' "${root}/tree"
  for t in "$@"
  do
    printf '%s\n' "${root}/targets/${t}"
  done
}

# release:delivered <root> <target>… prints "<path>\t<source>" for every repository path the release at <root>
# delivers for those targets, sorted by path, the last layer winning. A symlink is a source only when it points
# at a file; a dangling one is refused here, where it is found.
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

# release:source-of <root> <path> <target>… prints the release file behind one repository path, or nothing.
release:source-of() {
  local root=$1 path=$2
  shift 2
  release:delivered "${root}" "$@" | awk -F '\t' -v p="${path}" '$1 == p { print $2 }'
}

# release:target-only <target> prints the paths the target delivers that tree/ does not.
release:target-only() {
  comm -13 <(release:delivered "${kit_root}" | cut -f 1) <(release:delivered "${kit_root}" "$1" | cut -f 1)
}

# release:known-targets prints the targets this release carries.
release:known-targets() {
  (cd "${kit_root}/targets" && find . -mindepth 1 -maxdepth 1 -type d | sed 's@^\./@@' | sort)
}

# release:require-targets <name>… refuses a target this release does not carry, naming the ones it does.
release:require-targets() {
  local t
  for t in "$@"
  do
    if [[ ! -d "${kit_root}/targets/${t}" ]]
    then
      error "'${t}' is not a target this release knows; the targets are: $(release:known-targets | paste -s -d ' ' -)"
      return 2
    fi
  done
}

release:merged() { [[ "$1" == *.json ]]; }

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

# release:place <source> <path> [<base-source>] puts the release's content at <path>: a merge for .json, a copy otherwise.
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

# release:write-manifest <target>… records the version, the targets, and the release's digest for every delivered path.
release:write-manifest() {
  local path src
  {
    printf '# .kit/manifest - written by kit, not by hand\n'
    printf 'version %s\n' "$(release:version)"
    printf 'targets%s\n' "${*:+ $*}"
    while IFS=$'\t' read -r path src
    do
      printf '%s  %s\n' "$(release:digest < "${src}")" "${path}"
    done < <(release:delivered "${kit_root}" "$@")
  } > .kit/manifest
}

release:manifest-version() { sed -n 's/^version //p' .kit/manifest; }
release:manifest-targets() { sed -n 's/^targets *//p' .kit/manifest; }

# release:require-manifest refuses to go on in a repository that never took the kit.
release:require-manifest() {
  if [[ ! -f .kit/manifest ]]
  then
    error "run: kit install (this repository carries no .kit/manifest)"
    return 1
  fi
}

# release:finish <target>… reports a completed install or update.
release:finish() {
  ok "kit $(release:version) is installed, targets: ${*:-none}; .kit/manifest is written."
  step "Commit the delivered files. The loop is in .kit/1st.md."
}

# ----------------------------------------------------------------------------------------------------------------------
