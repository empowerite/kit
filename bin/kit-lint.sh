#!/usr/bin/env bash
# kit-lint - run each rule's linter over the paths it governs (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The paths are the arguments, else the staged ones, else with --all-files every tracked one. Each is matched to
# the rules that govern it, as kit context matches, and every rule whose frontmatter carries a `lint:` command
# runs it once over its paths. A linter the rule names that is not on PATH is a refusal, not a skip. The exit
# status is the number of rules whose linter refused. No hook manager is assumed; the lefthook module wires its
# pre-commit hook to this verb.
#
# see: bin/context.sh, docs/context.md

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use context yaml

all=false
paths=()
for arg in "$@"
do
  case "${arg}" in
    (--help)
      echo "usage: kit lint [--all-files] [<path>…]" >&2
      exit 0
      ;;
    (--all-files) all=true ;;
    (-*)
      error "unknown flag '${arg}' (usage: kit lint [--all-files] [<path>…])"
      exit 2
      ;;
    (*) paths+=("${arg}")
  esac
done
if (( ${#paths[@]} == 0 ))
then
  if [[ "${all}" == true ]]
  then
    readarray -t paths < <(git ls-files)
  else
    readarray -t paths < <(git diff --cached --name-only --diff-filter=ACMR)
  fi
fi
if (( ${#paths[@]} == 0 ))
then
  ok "nothing to lint"
  exit 0
fi

# The paths each rule governs, newline-separated, keyed by the rule file.
declare -A governed=()
for path in "${paths[@]}"
do
  path=$(context:relative "${path}")
  [[ -f "${path}" ]] || continue
  while IFS= read -r rule
  do
    [[ -n "${rule}" ]] || continue
    governed[${rule}]+="${path}"$'\n'
  done < <(context:rules "${path}")
done

refused=0
for rule in $(printf '%s\n' "${!governed[@]}" | LC_ALL=C sort)
do
  lint=$(yaml:value "${rule}" lint)
  [[ -n "${lint}" ]] || continue
  read -r -a command <<<"${lint}"
  readarray -t targets < <(printf '%s' "${governed[${rule}]}")
  step "${rule}: ${lint} over ${#targets[@]} path(s)"
  if ! guard:require "tool:${command[0]}"
  then
    refused=$(( refused + 1 ))
    continue
  fi
  "${command[@]}" "${targets[@]}" || refused=$(( refused + 1 ))
done
if (( refused > 0 ))
then
  error "${refused} rule(s) refused; see above"
  exit "${refused}"
fi
ok "every governed path passes its rule's linter"

# ----------------------------------------------------------------------------------------------------------------------
