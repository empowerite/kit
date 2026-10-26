#!/usr/bin/env bash
# kit-linked - refuse a pull request not linked to an open issue of this repository (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The delivered CI job's last check, and a check a desk can run on any pull request. The pull request is the
# argument, else the one the forge says this checkout is working on. A linked issue of another repository, or a
# closed one, is noted and does not count. On a runner the remedy also lands in the step summary.
#
# see: bin/forge.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use forge

if [[ "${1:-}" == --help ]]
then
  echo "usage: kit linked [<pull-request>]" >&2
  exit 0
fi

number=${1:-}
[[ -n "${number}" ]] || number=$(forge:pr-here)
if [[ ! "${number}" =~ ^[0-9]+$ ]]
then
  error "name the pull request: kit linked <number> (none was given, and this checkout is working on none)"
  exit 2
fi
repo=$(forge:repo) || exit 1
url=$(forge:repo-url) || exit 1
remedy=$(printf '%s\n' \
  "Link an open issue of this repository to [the pull request](${url}/pull/${number}):" "" \
  "1. open it, and under **Development** pick one or more open issues" \
  "2. re-run the check: linking fires no event, so it does not re-run on its own" "" \
  "\`kit issue <N>\` makes this link when it starts the work.")
refs=$(forge:pr-closing-issues "${number}") || exit 1
if [[ -z "${refs}" ]]
then
  error "pull request #${number} is not linked to any issue"
  summary "### ❌ Pull request #${number} is not linked to any issue" "" "${remedy}"
  exit 1
fi
open=0
while read -r n state issue_repo
do
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
if (( open == 0 ))
then
  error "every issue pull request #${number} is linked to is closed or belongs to another repository"
  summary "### ❌ Every linked issue is closed or belongs to another repository" "" "${remedy}"
  exit 1
fi
closes=$(cut -d ' ' -f 1 <<<"${refs}" | sed 's/^/#/' | paste -s -d ' ' -)
ok "pull request #${number} is linked to ${closes}"
summary "✅ Linked to ${closes}."

# ----------------------------------------------------------------------------------------------------------------------
