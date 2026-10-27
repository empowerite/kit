#!/usr/bin/env bash
# kit-region - print the content the active modules deliver as the region at a path (kit verb)
# ----------------------------------------------------------------------------------------------------------------------
#
# The command behind every region the kit delivers: the directive install writes into AGENTS.md reads
# `:generate:kit region AGENTS.md:until:"kit:end"`, and kit generate runs this to fill it. The path is relative
# to the repository, or with --home to the home directory. One active module delivers a region at a path, so
# the answer is that module's regions/<base>/<path>, and a path no active module delivers is refused.
#
# see: bin/generate.sh, bin/release.sh

set -o errexit -o nounset -o pipefail
# shellcheck source=bin/1st.sh
source "${kit_root:?run through bin/kit}/bin/1st.sh"
use release

base=repo
path=''
for arg in "$@"
do
  case "${arg}" in
    (--help)
      echo "usage: kit region [--home] <path>" >&2
      exit 0
      ;;
    (--home) base=home ;;
    (-*)
      error "unknown flag '${arg}' (usage: kit region [--home] <path>)"
      exit 2
      ;;
    (*) path=${arg}
  esac
done
if [[ -z "${path}" ]]
then
  echo "usage: kit region [--home] <path>" >&2
  exit 2
fi
release:require-manifest
readarray -t modules < <(release:manifest-modules)
release:render-all "${kit_root}" "${modules[@]}"
src=$(release:source-of "${kit_root}" "${base}" "${path}" "${modules[@]}")
if [[ -z "${src}" ]] || [[ "$(release:kind "${src}" "${path}")" != region ]]
then
  error "no active module delivers a region at ${path} (modules: ${modules[*]}); see .kit/manifest"
  exit 1
fi
cat "${src}"

# ----------------------------------------------------------------------------------------------------------------------
