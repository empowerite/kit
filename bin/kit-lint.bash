#!/usr/bin/env bash
# kit-lint - lint the staged paths, or the paths given; a missing linter is a reported skip (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# The body of a pre-commit hook, whatever runs the hook: shellcheck over bash, actionlint over workflows, and
# markdownlint-cli2 over markdown when the repository carries a config for it. A linter that is not on PATH is a
# reported skip, never a silent pass and never a refusal: lint is not the enforcement of record. The working
# tree's copy of each path is linted, so a partial add can be told about a line the commit does not carry.

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit lint [<path>…]    (no path: the staged paths)" >&2
  exit 0
fi

paths=()
if (( $# > 0 ))
then
  paths=("$@")
else
  while IFS= read -r -d '' path
  do
    paths+=("${path}")
  done < <(git diff --cached --name-only --diff-filter=ACMR -z)
fi

bash_paths=()
workflow_paths=()
markdown_paths=()
for path in "${paths[@]}"
do
  [[ -f "${path}" ]] || continue
  case "${path}" in
    (*.sh | *.bash) bash_paths+=("${path}") ;;
    (.github/workflows/*.yml | .github/workflows/*.yaml) workflow_paths+=("${path}") ;;
    (*.md) markdown_paths+=("${path}")
  esac
done

findings=0
run-linter() {  # <linter> <what> <path>…  runs the linter over the paths, or reports the skip
  local linter=$1 what=$2
  shift 2
  if ! command -v "${linter}" >/dev/null 2>&1
  then
    warn "${linter} is not on PATH, so the ${what} were NOT linted"
    return 0
  fi
  "${linter}" "$@" || findings=$(( findings + 1 ))
}
(( ${#bash_paths[@]} == 0 ))     || run-linter shellcheck "bash files" --shell=bash --external-sources "${bash_paths[@]}"
(( ${#workflow_paths[@]} == 0 )) || run-linter actionlint workflows "${workflow_paths[@]}"
if (( ${#markdown_paths[@]} > 0 )) && compgen -G ".markdownlint-cli2.*" >/dev/null
then
  run-linter markdownlint-cli2 "markdown files" "${markdown_paths[@]}"
fi

if (( findings > 0 ))
then
  error "a linter found something in ${findings} group(s) of paths; each finding above names the file and line."
  exit 1
fi

# ----------------------------------------------------------------------------------------------------------------------
