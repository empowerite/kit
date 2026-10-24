#!/usr/bin/env bash
# kit-target - list, take, or drop the targets this repository takes (kit verb, run through bin/kit)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bash/kit.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bash/1st.sh
source "${kit_root:?run through bin/kit}/bash/1st.sh"
use kit

usage() { echo "usage: kit target list | kit target add <target>… | kit target remove <target>…" >&2; }

verb=${1:-list}
shift || true
case "${verb}" in
  (--help) usage; exit 0 ;;
  (list | add | remove) ;;
  (*) usage; exit 2
esac
kit:require-manifest
read -r -a targets <<<"$(kit:manifest-targets)"

case "${verb}" in
  (list)
    printf '%s\n' "${targets[@]}"
    ;;
  (add)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq tool:git
    kit:require-targets "$@"
    for t in "$@"
    do
      if [[ " ${targets[*]} " == *" ${t} "* ]]
      then
        step "${t}: already taken"
        continue
      fi
      section "kit target add ${t}"
      while IFS= read -r path
      do
        step "${path}"
        kit:place "${kit_root}/targets/${t}/${path}" "${path}"
      done < <(kit:target-only "${t}")
      targets+=("${t}")
    done
    kit:write-manifest "${targets[@]}"
    ok "targets: ${targets[*]}"
    ;;
  (remove)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq tool:git
    kit:require-targets "$@"
    for t in "$@"
    do
      if [[ " ${targets[*]} " != *" ${t} "* ]]
      then
        step "${t}: not taken"
        continue
      fi
      section "kit target remove ${t}"
      while IFS= read -r path
      do
        src="${kit_root}/targets/${t}/${path}"
        if kit:merged "${path}"
        then
          [[ -f "${path}" ]] || continue
          step "${path}: the kit's entries taken out"
          # shellcheck disable=SC2016  # a jq filter, with jq variables
          kit:json-write "${path}" 'unmerge(.; $base[0])' --slurpfile base "${src}"
        else
          step "removing ${path}"
          rm -f "${path}"
        fi
      done < <(kit:target-only "${t}")
      kept=()
      for path in "${targets[@]}"
      do
        [[ "${path}" == "${t}" ]] || kept+=("${path}")
      done
      targets=("${kept[@]}")
    done
    kit:write-manifest "${targets[@]}"
    ok "targets: ${targets[*]:-none}"
esac

# ----------------------------------------------------------------------------------------------------------------------
