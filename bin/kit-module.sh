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

# shellcheck disable=SC2154  # release_bases is declared by bin/release.sh, which use loaded above
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
    read -r -a wanted <<<"$(release:resolve "${kit_root}" "${active[@]}" "$@" | paste -s -d ' ' -)"
    release:render-all "${kit_root}" "${wanted[@]}"
    section "kit module add $*"
    for base in "${release_bases[@]}"
    do
      while IFS= read -r path
      do
        [[ -n "${path}" ]] || continue
        dest=$(release:dest "${base}" "${path}")
        step "${dest}"
        release:place "$(release:source-of "${kit_root}" "${base}" "${path}" "${wanted[@]}")" "${dest}"
      done < <(release:paths-beyond "${kit_root}" "${base}" "${active[*]}" "${wanted[*]}")
    done
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
    for base in "${release_bases[@]}"
    do
      while IFS= read -r path
      do
        [[ -n "${path}" ]] || continue
        release:unplace "$(release:source-of "${kit_root}" "${base}" "${path}" "${active[@]}")" "$(release:dest "${base}" "${path}")"
      done < <(release:paths-beyond "${kit_root}" "${base}" "${remaining[*]}" "${active[*]}")
    done
    release:write-manifests "${remaining[@]}"
    ok "modules: ${remaining[*]:-none}"
esac

# ----------------------------------------------------------------------------------------------------------------------
