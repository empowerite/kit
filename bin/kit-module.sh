#!/usr/bin/env bash
# kit-module - list, activate, or deactivate the modules this repository takes (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/release.sh, docs/modules.md

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

usage() { echo "usage: kit module list | kit module add <module>… | kit module remove <module>…" >&2; }

verb=${1:-list}
shift || true
case "${verb}" in
  (--help) usage; exit 0 ;;
  (list | add | remove) ;;
  (*) usage; exit 2
esac
release:require-manifest
read -r -a active <<<"$(release:manifest-modules)"

case "${verb}" in
  (list)
    for module in $(release:modules)
    do
      if release:among "${module}" "${active[@]}"
      then mark='✅'
      else mark='  '
      fi
      printf '%s %-10s %s\n' "${mark}" "${module}" "$(release:field "${kit_root}" "${module}" description)"
    done
    ;;
  (add)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq
    release:require-modules "${kit_root}" "$@"
    read -r -a wanted <<<"$(release:resolve "${kit_root}" "${active[@]}" "$@" | paste -s -d ' ' -)"
    release:render-all "${kit_root}" "${wanted[@]}"
    section "kit module add $*"
    while IFS= read -r path
    do
      [[ -n "${path}" ]] || continue
      step "${path}"
      release:place "$(release:source-of "${kit_root}" "${path}" "${wanted[@]}")" "${path}"
    done < <(release:paths-beyond "${kit_root}" "${active[*]}" "${wanted[*]}")
    release:write-manifest "${wanted[@]}"
    ok "modules: ${wanted[*]}"
    ;;
  (remove)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq
    release:require-modules "${kit_root}" "$@"
    remaining=("${active[@]}")
    for module in "$@"
    do
      if ! release:among "${module}" "${remaining[@]}"
      then
        step "${module}: not active"
        continue
      fi
      read -r -a dependents <<<"$(release:dependents "${kit_root}" "${module}" "${remaining[@]}" | paste -s -d ' ' -)"
      if (( ${#dependents[@]} > 0 ))
      then
        error "${module} cannot be removed while ${dependents[*]} depend(s) on it"
        exit 1
      fi
      read -r -a remaining <<<"$(release:without "${module}" "${remaining[@]}" | paste -s -d ' ' -)"
    done
    release:render-all "${kit_root}" "${active[@]}"
    section "kit module remove $*"
    while IFS= read -r path
    do
      [[ -n "${path}" ]] || continue
      release:unplace "$(release:source-of "${kit_root}" "${path}" "${active[@]}")" "${path}"
    done < <(release:paths-beyond "${kit_root}" "${remaining[*]}" "${active[*]}")
    release:write-manifest "${remaining[@]}"
    ok "modules: ${remaining[*]:-none}"
esac

# ----------------------------------------------------------------------------------------------------------------------
