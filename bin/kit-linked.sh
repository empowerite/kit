#!/usr/bin/env bash
# kit-linked - refuse a change not linked to an open issue of this repository (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The delivered CI job's last check, and a check a desk can run on any change: a pull request on GitHub, a merge
# request on GitLab, named in the forge's own words. The change is the argument, else the one the forge says this
# checkout is working on. A linked issue of another repository, or a
# closed one, is noted and does not count. On a runner the remedy also lands in the step summary.
#
# see: bin/forge.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use forge tracker

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit linked [<number>]" >&2
  exit 0
fi

number=${1:-}
[[ -n "${number}" ]] || number=$(forge:pr-here)
if [[ ! "${number}" =~ ^[0-9]+$ ]]
then
  error "name the change: kit linked <number> (none was given, and this checkout is working on none)"
  exit 2
fi
IFS=$'\t' read -r sigil noun < <(forge:noun) || exit 1
change="${noun} ${sigil}${number}"
repo=$(forge:repo) || exit 1
url=$(forge:pr-url "${number}") || exit 1
remedy=$(printf '%s\n' \
  "Link an open issue of this repository to [the ${noun}](${url}):" "" \
  "1. add a line \`Closes <reference>\` to its description for each open issue it resolves, #7 or AB#4521" \
  "2. re-run the check: linking fires no event, so it does not re-run on its own" "" \
  "\`kit issue <N>\` makes this link when it starts the work.")
refs=$(forge:pr-closing-issues "${number}") || exit 1
# A tracker the forge does not link by itself is linked by a closing keyword in the change's body.
others=$(tracker:body-refs "$(forge:pr-body "${number}")")
if [[ -z "${refs}${others}" ]]
then
  error "${change} is not linked to any issue"
  summary "### ❌ The ${change} is not linked to any issue" "" "${remedy}"
  exit 1
fi
open=0
while read -r n state issue_repo
do
  [[ -n "${n}" ]] || continue
  if [[ "${issue_repo}" != "${repo}" ]]
  then
    warn "skipping ${issue_repo}#${n}: it belongs to another repository"
  elif [[ "${state}" != OPEN ]]
  then
    warn "#${n} is ${state}; merging will not change it"
  else
    open=$(( open + 1 ))
  fi
done <<<"${refs}"
while read -r ref
do
  [[ -n "${ref}" ]] || continue
  state=$(tracker:of "${ref}" state) || continue
  if [[ "${state}" != OPEN ]]
  then
    warn "${ref} is ${state}; merging will not change it"
  else
    open=$(( open + 1 ))
  fi
done <<<"${others}"
if (( open == 0 ))
then
  error "every issue ${change} is linked to is closed or belongs to another repository"
  summary "### ❌ Every linked issue is closed or belongs to another repository" "" "${remedy}"
  exit 1
fi
closes=$({
  [[ -z "${refs}" ]] || cut -d ' ' -f 1 <<<"${refs}" | sed 's/^/#/'
  [[ -z "${others}" ]] || printf '%s\n' "${others}"
} | paste -s -d ' ' -)
ok "${change} is linked to ${closes}"
summary "✅ Linked to ${closes}."

# ----------------------------------------------------------------------------------------------------------------------
