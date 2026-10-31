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

# lint:jq-compiles <file>… compiles every jq module and refers to each function it defines, so jq resolves every
# name each body calls. Importing a module alone does not: a function that calls an undefined one passes until
# something calls it. A module may import modules beside it, and the kit's own.
lint:jq-compiles() {
  local file name dir refs message n findings=0
  for file in "$@"
  do
    name=$(basename "${file}" .jq)
    dir=$(dirname "${file}")
    # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
    if ! refs=$(jq --null-input --raw-output -L "${dir}" -L "${kit_jq}" --arg m "${name}" '
        $m | modulemeta | .defs
        | map(split("/") | .[0] + (if .[1] == "0" then "" else "(\([range(.[1] | tonumber)] | map(".") | join(";")))" end))
        | join(", ")' 2>&1) \
      || ! message=$(jq --null-input -L "${dir}" -L "${kit_jq}" \
           "include \"${name}\"; if false then (${refs:-empty}) else empty end" 2>&1)
    then
      message=${message:-${refs}}
      [[ "${message}" =~ line\ ([0-9]+) ]] && n=${BASH_REMATCH[1]} || n=1
      printf '%s:%s: %s\n' "${file}" "${n}" "${message%%$'\n'*}"
      findings=$(( findings + 1 ))
    fi
  done
  (( findings == 0 ))
}

# lint:choices <key> [<answer>…] answers kit syntax --choices for <path>: an open set, every tracked file, since a
# path kit lint takes need not be among them.
lint:choices() {
  [[ "$1" == '<path>' ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'open\n'
    return 0
  fi
  git ls-files | awk '{ printf "%s\t\t\t\n", $0 }'
}

# ----------------------------------------------------------------------------------------------------------------------
