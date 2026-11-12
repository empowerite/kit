# shellcheck shell=bash
# kit-packages.sh - the kit-packages source: a tarball of kit packages, pinned by its sha256 digest (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A package source is never on the pull path: kit.yml alone names one, and sources:fetch unpacks it into the cache.
# Its coordinate is package:, as the source's own url names the one tarball it serves. A url is
# git:<owner>/<repo>@<tag>, the forge's tarball of that tag, on github.com or on a host named first, as
# git:gitlab.com/<owner>/<repo>@<tag>, or any URL curl fetches. It takes digest:, the sha256 of the tarball's bytes.

sources/kit-packages:fetch() {
  local status=0
  curl --fail --silent --location --output "$3" "$(sources/kit-packages:tarball "$1")" || status=$?
  case "${status}" in
    (0) return 0 ;;
    (22) return 4 ;;
    (*) return 1
  esac
}

sources/kit-packages:hash() { printf 'sha256:%s\n' "$(file:digest "$1")"; }

sources/kit-packages:home() { :; }

sources/kit-packages:keys() { printf 'digest\n'; }

sources/kit-packages:serves() { [[ "$1" == package:* ]]; }

# sources/kit-packages:settings writes nothing: no language's manager reads kit's packages.
sources/kit-packages:settings() { :; }

# sources/kit-packages:versions has none to list: a package source is the one tarball its digest pins.
sources/kit-packages:versions() { return 4; }

# The seam's calls end here; what follows is this provider's own.

# sources/kit-packages:tarball <url> prints the address of the tarball a url names: a git: url as its forge's tarball
# of the tag.
sources/kit-packages:tarball() {
  local spec host repo tag
  case "$1" in
    (git:*)
      spec=${1#git:}
      tag=${spec##*@}
      spec=${spec%@*}
      host=github.com
      [[ "${spec%%/*}" == *.* ]] && host=${spec%%/*} spec=${spec#*/}
      repo=${spec##*/}
      case "${host}" in
        (github.com) printf 'https://codeload.github.com/%s/tar.gz/refs/tags/%s\n' "${spec}" "${tag}" ;;
        (*) printf 'https://%s/%s/-/archive/%s/%s-%s.tar.gz\n' "${host}" "${spec}" "${tag}" "${repo}" "${tag}"
      esac
      ;;
    (*) printf '%s\n' "$1"
  esac
}

# ----------------------------------------------------------------------------------------------------------------------
