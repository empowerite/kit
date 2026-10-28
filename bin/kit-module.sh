#!/usr/bin/env bash
# kit-module - list, activate, or deactivate the modules this repository takes (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# kit module list | add <module>… | remove <module>…
#
#   list                 prints every module in reach, with a ✅ beside the active ones, its origin (release or
#                        repository), and its description. This is the default.
#   add <module>…        activates the modules and the modules they depend on, places the files they deliver,
#                        and rewrites the manifests. When a different file is already at a delivered path, it is
#                        named and left alone, and no manifest is written.
#   remove <module>…     takes out what the modules delivered (files, the kit's JSON entries, and the kit's
#                        regions) and rewrites the manifests. A module that an active module depends on is refused.
#
# Exits 0 on success, 2 on a usage error or a module the release does not have, and 1 for any other failure.
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
      printf '%s %-10s %-10s %s\n' "${mark}" "${module}" "$(release:origin "${kit_root}" "${module}")" \
        "$(release:description "${kit_root}" "${module}")"
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
    release:say-unmet "${wanted[@]}"
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
    # Dependents are checked against the modules that remain, so the order of the named modules does not matter.
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
