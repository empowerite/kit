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
readarray -t active < <(release:manifest-modules)

case "${verb}" in
  (list)
    for module in $(release:modules)
    do
      if release:among "${module}" "${active[@]}"
      then mark='✅'
      else mark='  '
      fi
      printf '%s %-10s %s\n' "${mark}" "${module}" "$(release:description "${kit_root}" "${module}")"
    done
    ;;
  (add)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq
    release:require-modules "${kit_root}" "$@"
    readarray -t wanted < <(release:resolve "${kit_root}" "${active[@]}" "$@")
    release:render-all "${kit_root}" "${wanted[@]}"
    section "kit module add $*"
    refused=0
    for base in $(release:bases)
    do
      while IFS= read -r path
      do
        [[ -n "${path}" ]] || continue
        release:place-new "$(release:source-of "${kit_root}" "${base}" "${path}" "${wanted[@]}")" \
          "$(release:dest "${base}" "${path}")" "placed" || refused=$(( refused + 1 ))
      done < <(release:paths-beyond "${kit_root}" "${base}" "${active[*]}" "${wanted[*]}")
    done
    if (( refused > 0 ))
    then
      error "${refused} path(s) already hold a different file; move each aside, then run: kit module add $*"
      exit 1
    fi
    release:write-manifests "${wanted[@]}"
    ok "modules: ${wanted[*]}"
    ;;
  (remove)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq
    release:require-modules "${kit_root}" "$@"
    remaining=("${active[@]}")
    for module in "$@"
    do
      release:among "${module}" "${active[@]}" || step "${module}: not active"
      readarray -t remaining < <(release:without "${module}" "${remaining[@]}")
    done
    # Dependents are judged against what stays, so the order the modules are named in does not matter.
    for module in "$@"
    do
      readarray -t dependents < <(release:dependents "${kit_root}" "${module}" "${remaining[@]}")
      if (( ${#dependents[@]} > 0 ))
      then
        error "${module} cannot be removed while ${dependents[*]} depend(s) on it"
        exit 1
      fi
    done
    release:render-all "${kit_root}" "${active[@]}"
    section "kit module remove $*"
    for base in $(release:bases)
    do
      while IFS= read -r path
      do
        [[ -n "${path}" ]] || continue
        release:unplace "$(release:source-of "${kit_root}" "${base}" "${path}" "${active[@]}")" \
          "$(release:dest "${base}" "${path}")"
      done < <(release:paths-beyond "${kit_root}" "${base}" "${remaining[*]}" "${active[*]}")
    done
    release:write-manifests "${remaining[@]}"
    ok "modules: ${remaining[*]:-none}"
esac

# ----------------------------------------------------------------------------------------------------------------------
