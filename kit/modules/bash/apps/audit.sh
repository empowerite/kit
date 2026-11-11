# shellcheck shell=bash
# audit.sh - kit audit's stages: the forge's attestations, alerts, controls, and waivers, and the policy's rules
# ----------------------------------------------------------------------------------------------------------------------
# shellcheck disable=SC2154  # kit_jq is set by modules/bash/1st.sh
#
# Trust is the app whose tabs are all its own. Four stages ask the forge through the seam of modules/bash/forge.sh,
# and the last reads kit's own policy:
#
#   evidence  a row per attestation of the newest releases' assets and artifacts: its kind and who signed it
#   findings  a row per alert of the code scanning, Dependabot, and secret scanning alerts, open or dismissed, each
#             row its scanner's, the tool; a scanner off here, or one that refuses the caller, is its line under
#             errors, and the others still draw
#   controls  a row per ruleset or protection that guards the default branch: its rules, and how it is enforced
#   waivers   a row per bypass of a control, and per alert dismissed, by whom and why: the bypasses the forge says, and
#             the dismissed among the alerts findings reads, each scanner read once a process for both; a scanner
#             off here, or one that refuses the caller, is under errors as in findings, and the rest still draw
#   policy    a row per rule kit policy judges by, as kit policy --rules --json lists them
#
# evidence's mark holds the forge's marks of its runs and its releases, as a run or a release makes an attestation;
# findings', its runs' and its alerts', as a scan is a run and a person dismisses an alert without one. controls and
# waivers have none, and are read again on Refresh only. policy's is the hash of kit.yml and .kit/policy, and
# on a branch past the default, of the base's, which judge it.
#
# see: bin/kit-audit, modules/bash/apps.sh, modules/bash/forge.sh, docs/apps.md
use apps forge policy settings file

# audit_stages are kit audit's stages, in Trust's tab order.
# shellcheck disable=SC2034  # read by modules/bash/apps.sh
audit_stages=(evidence findings controls waivers policy)

# apps/audit:evidence prints the attestations document, through apps::attestation, limited when the forge holds more
# than it read.
apps/audit:evidence() { apps:forge attestations attestations attestation; }

# apps_audit_reads is a directory this process keeps each scanner's document in, once read, so the findings and the
# waivers of one picture share one read of each; the stages run in subshells of the shell that loads this module.
apps_audit_reads=''
file:scratch apps_audit_reads || apps_audit_reads=''

# apps_audit_scanners are the three alerts calls: each one's feature and call, the feature being the rows' tool.
apps_audit_scanners=('code-scanning code-scanning-alerts' 'dependabot dependabot-alerts'
  'secret-scanning secret-scanning-alerts')

# apps/audit:alerts <feature> <call> prints the document of the scanner's call, its alerts as the call prints them,
# through apps::kept, read once a process. One whose read fails is {feature, error}, its line, under errors, with no
# rows, and it returns 1 after it, as it does each time it is asked.
apps/audit:alerts() {
  local feature=$1 call=$2 kept said doc line status=0
  kept=${apps_audit_reads:+${apps_audit_reads}/${feature}}
  if [[ -n "${kept}" && -f "${kept}.json" && -f "${kept}.status" ]]
  then
    cat "${kept}.json"
    return "$(<"${kept}.status")"
  fi
  file:temp said audit || return 1
  if ! doc=$(apps:forge "${feature}" "${call}" kept 2>"${said}")
  then
    status=1
    line=$(grep -v '^[[:space:]]*$' "${said}" | tail -n 1 || true)  # grep's exit 1: the read said nothing
    doc=$(jq --null-input --compact-output --arg feature "${feature}" --arg error "${line#❌ }" \
      '{rows: [], errors: [{feature: $feature, error: $error}]}') || return 1
  fi
  # A document that cannot be kept costs only another read of it.
  if [[ -n "${kept}" ]] && printf '%s\n' "${doc}" > "${kept}.json"
  then
    printf '%s\n' "${status}" > "${kept}.status" || rm -f "${kept}.json"
  fi
  printf '%s\n' "${doc}"
  return "${status}"
}

# apps/audit:scanned <def> prints one document of the three scanners' alerts, each row through apps::<def>, given the
# scanner's feature, its tool. A scanner off here is {feature, off: true} under errors; one whose read fails is
# {feature, error}, its line, and the others still draw; it returns 1 after the document then.
apps/audit:scanned() {
  local def=$1 each feature call doc unread=0
  local -a docs=()
  for each in "${apps_audit_scanners[@]}"
  do
    read -r feature call <<<"${each}"
    doc=$(apps/audit:alerts "${feature}" "${call}") || unread=1
    doc=$(jq --compact-output -L "${kit_jq}" --arg tool "${feature}" \
      "import \"apps\" as apps; .rows |= map(apps::${def}(\$tool))" <<<"${doc}") || return 1
    docs+=("${doc}")
  done
  apps:merge "${docs[@]}" || return 1
  return "${unread}"
}

# apps/audit:findings prints one document of the three scanners' alerts, each row through apps::alert, which gives it
# its tool, as apps/audit:scanned does.
apps/audit:findings() { apps/audit:scanned alert; }

# apps/audit:controls prints the controls document, through apps::control.
apps/audit:controls() { apps:forge rulesets rulesets control; }

# apps/audit:waivers prints one document of the forge's bypasses, through apps::waiver, and of the dismissed alerts,
# through apps::dismissal, as apps/audit:scanned reads them. A bypasses read that fails is {feature, error}, its line,
# under errors, as a scanner's is, and the rest still draw; it returns 1 after the document then.
apps/audit:waivers() {
  local bypasses dismissed said line unread=0
  file:temp said audit || return 1
  if ! bypasses=$(apps:forge bypasses bypasses waiver 2>"${said}")
  then
    unread=1
    line=$(grep -v '^[[:space:]]*$' "${said}" | tail -n 1 || true)  # grep's exit 1: the read said nothing
    bypasses=$(jq --null-input --compact-output --arg error "${line#❌ }" \
      '{rows: [], errors: [{feature: "bypasses", error: $error}]}') || return 1
  fi
  dismissed=$(apps/audit:scanned dismissal) || unread=1
  [[ -n "${dismissed}" ]] || return 1
  apps:merge "${bypasses}" "${dismissed}" || return 1
  return "${unread}"
}

# apps/audit:policy prints the policy's document, a row per line kit policy --rules --json prints. It returns 1 when
# kit policy fails, its line said.
apps/audit:policy() {
  local out
  out=$(run kit policy --rules --json) || return 1
  jq --slurp --compact-output -L "${kit_jq}" 'import "apps" as apps; {rows: map(apps::rule), errors: []}' <<<"${out}"
}

# apps/audit:mark-evidence prints the forge's marks of its runs and its releases.
apps/audit:mark-evidence() { apps:forge-mark runs releases; }

# apps/audit:mark-findings prints the forge's marks of its runs and its alerts.
apps/audit:mark-findings() { apps:forge-mark runs alerts; }

# apps/audit:mark-policy prints the hash of kit.yml and .kit/policy, and on a branch past the default, whose policy is
# its base's, the base's kit.yml and .kit/policy as git names them, which a fetch moves; nothing when none is there.
# A base that cannot be named leaves the base out: kit policy --rules then says why.
apps/audit:mark-policy() {
  local hash past base='' object
  local -a marks=()
  hash=$(apps:hash "$(settings:file)" .kit/policy) || return 1
  [[ -z "${hash}" ]] || marks+=("${hash}")
  if past=$(policy:branch-past 2>/dev/null) && [[ -n "${past}" ]] && base=$(policy:base 2>/dev/null)
  then
    for object in kit.yml .kit/policy
    do
      object=$(git rev-parse --quiet --verify "${base}:${object}" 2>/dev/null) || continue
      marks+=("${object}")
    done
  fi
  (( ${#marks[@]} == 0 )) || printf '%s\n' "${marks[*]}"
}

# ----------------------------------------------------------------------------------------------------------------------
