#!/usr/bin/env bash
# kit-generate - expand every kit:generate region to what its command makes, or check that it is (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit generate [--check] [--iterations=<n>] [<path>…]
#
# Paths are relative to the current directory. With no paths, kit generate expands every tracked file and every
# untracked file git does not ignore.
#
#   - Named files are expanded once, or <n> times with --iterations=<n>.
#   - With no paths, kit generate repeats until a pass changes nothing.
#   - --check writes nothing, and fails naming every stale file.
#
# kit check runs kit generate --check, and the lefthook pre-commit hook runs it over the staged paths.
#
# Exits 0 on success, 2 on an unknown flag, and non-zero when a file is stale (with --check) or cannot be expanded.
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
