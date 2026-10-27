#!/usr/bin/env bash
# kit-generate - expand every kit:generate region to what its command makes, or check that it is (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The paths are the arguments, taken from the directory the command was typed in, else every tracked file and
# every untracked one git does not ignore. Named files are expanded once, or --iterations=<n> times; the whole
# repository is expanded until a pass changes nothing. --check writes nothing and fails naming every stale file;
# kit check runs it, and the lefthook fragment runs it over the staged paths.
#
# see: bin/generate.sh, docs/generate.md

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use context generate

args=()
for arg in "$@"
do
  case "${arg}" in
    (--help)
      echo "usage: kit generate [--check] [--iterations=<n>] [<path>…]" >&2
      exit 0
      ;;
    (--check | --iterations=*) args+=("${arg}") ;;
    (-*)
      error "unknown flag '${arg}' (usage: kit generate [--check] [--iterations=<n>] [<path>…])"
      exit 2
      ;;
    (*) args+=("$(context:relative "${arg}")")
  esac
done
generate:run "${args[@]}"

# ----------------------------------------------------------------------------------------------------------------------
