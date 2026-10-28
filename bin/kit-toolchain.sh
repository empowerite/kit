#!/usr/bin/env bash
# kit-toolchain - install, report, and update the tools the repository's gate runs, whatever manages them (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# One verb over every declared toolchain, mise, brew, or docker, so a desk, a runner, and the delivered CI job say
# the same thing whichever manages the tools:
#
#   list                    each declared toolchain and the file it reads
#   install                 every toolchain's pinned tools installed
#   outdated [--check]      one line per tool behind its source; --check fails when there is one
#   update                  every toolchain moved to its latest, its pins and lock rewritten
#   path                    the directories that put the tools first on PATH, one per line, for a runner's PATH
#   run <command>…          the command with the tools: inside the container for docker, else with the path first
#
# see: bin/toolchain.sh, docs/toolchains.md

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use toolchain

# Every subcommand reads the declared toolchains once, so a refused declaration stops it.
providers=$(toolchain:declared) || exit 1

usage() { echo "usage: kit toolchain list | install | outdated [--check] | update | path | run <command>…" >&2; }

verb=${1:-}
[[ -n "${verb}" ]] && shift
case "${verb}" in
  (--help | '')
    usage
    exit 0
    ;;
  (list)
    for provider in ${providers}
    do
      printf '%-8s %s\n' "${provider}" "$(toolchain:call "${provider}" file)"
    done
    ;;
  (install)
    if [[ -z "${providers}" ]]
    then
      ok "no toolchain is declared or detected; the gate runs with the tools already on PATH"
      exit 0
    fi
    for provider in ${providers}
    do
      step "${provider}: installing from $(toolchain:call "${provider}" file)"
      toolchain:call "${provider}" install || { error "${provider}: the tools could not be installed; see above"; exit 1; }
    done
    ok "toolchains installed: ${providers//$'\n'/ }"
    summary "✅ Toolchains installed: ${providers//$'\n'/ }."
    ;;
  (outdated)
    check=false
    [[ "${1:-}" == --check ]] && check=true
    behind=0
    for provider in ${providers}
    do
      while IFS=$'\t' read -r name installed latest
      do
        [[ -n "${name}" ]] || continue
        printf '%-8s %-32s %-16s → %s\n' "${provider}" "${name}" "${installed}" "${latest}"
        behind=$(( behind + 1 ))
      done < <(toolchain:call "${provider}" outdated)
    done
    if (( behind == 0 ))
    then
      ok "every tool is at its latest"
    elif [[ "${check}" == true ]]
    then
      error "${behind} tool(s) are behind; run: kit toolchain update"
      exit 1
    else
      warn "${behind} tool(s) are behind; run: kit toolchain update"
    fi
    ;;
  (update)
    toolchain:each update
    ok "every toolchain is updated; commit the pins and the locks it rewrote"
    ;;
  (path)
    toolchain:path
    ;;
  (run)
    (( $# > 0 )) || { usage; exit 2; }
    toolchain:run "$@"
    ;;
  (*)
    usage
    exit 2
esac

# ----------------------------------------------------------------------------------------------------------------------
