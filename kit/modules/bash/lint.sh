# shellcheck shell=bash
# lint.sh - the kit's own checks, which a rule names under checks: in its frontmatter (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# A linter is a tool a rule names under lint:. A check is a function here, lint:<name>, which kit lint runs over the
# files a rule governs when the rule's frontmatter lists <name> under checks:. A check prints each finding as
# <file>:<line>: <message>, and returns 1 when it found any.
#
# see: bin/kit-lint

# lint:names prints the name of every check, one per line.
lint:names() {
  declare -F | sed -n 's/^declare -f lint://p' | grep -v '^names$' | LC_ALL=C sort
}

# lint:pinned <file>… holds every `uses:` in a workflow to a local action (./…), a docker image (docker://…), or
# an action pinned to a full commit SHA with the version as a trailing comment: owner/repo@<40-hex>  # v1.2.3. A
# tag or a branch can move; the comment is the version a reader, and Dependabot, see.
lint:pinned() {
  local file line n value comment findings=0
  for file in "$@"
  do
    n=0
    while IFS= read -r line || [[ -n "${line}" ]]
    do
      n=$(( n + 1 ))
      [[ "${line}" =~ ^[[:space:]]*(-[[:space:]]+)?uses:[[:space:]]*([^[:space:]#]+)[[:space:]]*(#.*)?$ ]] || continue
      value=${BASH_REMATCH[2]} comment=${BASH_REMATCH[3]:-}
      value=${value#[\"\']}
      value=${value%[\"\']}
      case "${value}" in
        (./* | docker://*) continue ;;
      esac
      if [[ ! "${value}" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(/[^@[:space:]]+)?@[0-9a-f]{40}$ ]]
      then
        printf '%s:%d: uses: %s is not pinned to a commit SHA; write owner/repo@<40-hex>  # v<version>\n' \
          "${file}" "${n}" "${value}"
        findings=$(( findings + 1 ))
      elif [[ ! "${comment}" =~ ^#[[:space:]]*v[0-9] ]]
      then
        printf '%s:%d: uses: %s has no version comment; add  # v<version> after the SHA\n' "${file}" "${n}" "${value}"
        findings=$(( findings + 1 ))
      fi
    done < "${file}"
  done
  (( findings == 0 ))
}

# ----------------------------------------------------------------------------------------------------------------------
