# shellcheck shell=bash
# shellcheck disable=SC2016  # jq filters name jq variables with $, in single quotes on purpose
# shellcheck disable=SC2154  # kit_root is exported by bin/kit
# kit.sh - the release's layout, the one merge rule, and the manifest (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A release mirrors its tree/ at the repository root, and targets/<name>/ for each target taken; a symlink in
# either is delivered as the file it points at. A destination ending in .json is merged by key, with the kit's
# keys set, the kit's array entries added, and on update or removal only the previous release's entries taken
# out. .kit/manifest records the version, the targets, and the release's digest for every delivered path.
#
# Every function reads the release from kit_root, exported by bin/kit.

# kit:layers <root> <target>… prints the directories mirrored at the repository root, tree first.
kit:layers() {
  local root=$1 t
  shift
  printf '%s\n' "${root}/tree"
  for t in "$@"
  do
    printf '%s\n' "${root}/targets/${t}"
  done
}

# kit:delivered <root> <target>… prints every repository path the release at <root> delivers for those targets.
kit:delivered() {
  local layer
  while IFS= read -r layer
  do
    (cd "${layer}" && find . \( -type f -o -type l \) | sed 's@^\./@@')
  done < <(kit:layers "$@") | sort --unique
}

# kit:source-of <root> <path> <target>… prints the release file behind a repository path, the last layer winning.
kit:source-of() {
  local root=$1 path=$2 layer found=""
  shift 2
  while IFS= read -r layer
  do
    [[ -e "${layer}/${path}" ]] && found="${layer}/${path}"
  done < <(kit:layers "${root}" "$@")
  [[ -n "${found}" ]] && printf '%s\n' "${found}"
}

# kit:target-only <target> prints the paths the target delivers that tree/ does not.
kit:target-only() {
  comm -13 <(kit:delivered "${kit_root}") <(kit:delivered "${kit_root}" "$1")
}

# kit:known-targets prints the targets this release carries.
kit:known-targets() {
  (cd "${kit_root}/targets" && find . -mindepth 1 -maxdepth 1 -type d | sed 's@^\./@@' | sort)
}

# kit:require-targets <name>… refuses a target this release does not carry, naming the ones it does.
kit:require-targets() {
  local t
  for t in "$@"
  do
    if [[ ! -d "${kit_root}/targets/${t}" ]]
    then
      error "'${t}' is not a target this release knows; the targets are: $(kit:known-targets | paste -s -d ' ' -)"
      return 2
    fi
  done
}

kit:merged() { [[ "$1" == *.json ]]; }

# The parameters are values ($), not filters: a filter parameter is re-read against the input at each use, which
# inside the recursion is the wrong level of the object.
kit_jq='
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

# kit:json-write <path> <filter> [<jq option>…] rewrites the JSON at <path> through <filter>, making it if missing.
kit:json-write() {
  local path=$1 filter=$2
  shift 2
  mkdir -p "$(dirname "${path}")"
  [[ -f "${path}" ]] || printf '{}\n' > "${path}"
  jq --indent 2 "${kit_jq} ${filter}" "$@" "${path}" > "${path}.kit-new"
  mv "${path}.kit-new" "${path}"
}

# kit:json-holds <path> <source> is true when the JSON at <path> carries every entry the release's <source> writes.
kit:json-holds() {
  jq --exit-status "${kit_jq} holds(.; \$new[0])" --slurpfile new "$2" "$1" >/dev/null
}

# kit:digest prints the sha256 of stdin, by whichever tool the machine carries.
kit:digest() {
  if command -v sha256sum >/dev/null 2>&1
  then
    sha256sum | cut -d ' ' -f 1
  else
    shasum -a 256 | cut -d ' ' -f 1
  fi
}

# kit:place <source> <path> [<base-source>] puts the release's content at <path>: a merge for .json, a copy otherwise.
kit:place() {
  local src=$1 path=$2 base=${3:-/dev/null}
  if kit:merged "${path}"
  then
    kit:json-write "${path}" 'merge(.; $base[0]; $new[0])' --slurpfile new "${src}" --slurpfile base "${base}"
  else
    mkdir -p "$(dirname "${path}")"
    cat "${src}" > "${path}.kit-new"
    mv "${path}.kit-new" "${path}"
  fi
}

# kit:write-manifest <target>… records the version, the targets, and the release's digest for every delivered path.
kit:write-manifest() {
  local path
  {
    printf '# .kit/manifest - written by kit, not by hand\n'
    printf 'version %s\n' "$(cat "${kit_root}/version")"
    printf 'targets%s\n' "${*:+ $*}"
    while IFS= read -r path
    do
      printf '%s  %s\n' "$(kit:digest < "$(kit:source-of "${kit_root}" "${path}" "$@")")" "${path}"
    done < <(kit:delivered "${kit_root}" "$@")
  } > .kit/manifest
}

kit:manifest-version() { sed -n 's/^version //p' .kit/manifest; }
kit:manifest-targets() { sed -n 's/^targets *//p' .kit/manifest; }

# kit:require-manifest refuses to go on in a repository that never took the kit.
kit:require-manifest() {
  if [[ ! -f .kit/manifest ]]
  then
    error "run: kit install (this repository carries no .kit/manifest)"
    return 1
  fi
}

# kit:finish <target>… reports a completed install or update.
kit:finish() {
  ok "kit $(cat "${kit_root}/version") is installed, targets: ${*:-none}; .kit/manifest is written."
  step "Commit the delivered files. The loop is in .kit/1st.md."
}

# ----------------------------------------------------------------------------------------------------------------------
