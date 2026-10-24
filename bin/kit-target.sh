#!/usr/bin/env bash
# kit-target - list, take, or drop the targets this repository takes (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# see: bin/release.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

usage() { echo "usage: kit target list | kit target add <target>… | kit target remove <target>…" >&2; }

verb=${1:-list}
shift || true
case "${verb}" in
  (--help) usage; exit 0 ;;
  (list | add | remove) ;;
  (*) usage; exit 2
esac
release:require-manifest
read -r -a targets <<<"$(release:manifest-targets)"

case "${verb}" in
  (list)
    printf '%s\n' "${targets[@]}"
    ;;
  (add)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq
    release:require-targets "$@"
    for t in "$@"
    do
      if release:among "${t}" "${targets[@]}"
      then
        step "${t}: already taken"
        continue
      fi
      section "kit target add ${t}"
      while IFS= read -r path
      do
        step "${path}"
        release:place "${kit_root}/targets/${t}/${path}" "${path}"
      done < <(release:target-only "${t}")
      targets+=("${t}")
    done
    release:write-manifest "${targets[@]}"
    ok "targets: ${targets[*]}"
    ;;
  (remove)
    (( $# > 0 )) || { usage; exit 2; }
    guard:require tool:jq
    release:require-targets "$@"
    for t in "$@"
    do
      if ! release:among "${t}" "${targets[@]}"
      then
        step "${t}: not taken"
        continue
      fi
      section "kit target remove ${t}"
      while IFS= read -r path
      do
        src="${kit_root}/targets/${t}/${path}"
        if release:merged "${path}"
        then
          [[ -f "${path}" ]] || continue
          step "${path}: the kit's entries taken out"
          # shellcheck disable=SC2016  # a jq filter, with jq variables
          release:json-write "${path}" 'unmerge(.; $base[0])' --slurpfile base "${src}"
        else
          step "removing ${path}"
          rm -f "${path}"
        fi
      done < <(release:target-only "${t}")
      read -r -a targets <<<"$(release:without "${t}" "${targets[@]}" | paste -s -d ' ' -)"
    done
    release:write-manifest "${targets[@]}"
    ok "targets: ${targets[*]:-none}"
esac

# ----------------------------------------------------------------------------------------------------------------------
