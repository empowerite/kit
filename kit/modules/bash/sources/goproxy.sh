# shellcheck shell=bash
# goproxy.sh - the goproxy source: a Go module proxy, which serves each module's versions as zips (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# A module: coordinate, module:<path>@<version>, is served at <url>/<path>/@v/<version>.zip, the path and the version
# case-encoded, each capital letter as ! and the letter in lower case, as the GOPROXY protocol has it. Its hash is
# Go's own, h1:, the dirhash go.sum holds: the sha256 of a line per file of the zip, in name order, each the file's
# sha256 and its name, in base64. The versions are what <url>/<path>/@v/list lists. Go's toolchain is the module
# golang.org/toolchain, so a proxy supplies go.
#
# It takes private:, a list of module patterns that have no checksum database, as GOPRIVATE and GONOSUMDB take them.
# settings writes GOPROXY, GOPRIVATE, and GONOSUMDB; the seam joins each across the path.

sources/goproxy:fetch() {
  local module version
  module=${2#module:}
  version=${module##*@}
  module=${module%@*}
  [[ "$2" == module:*@* ]] || return 4
  sources:get "$1/$(sources/goproxy:escape "${module}")/@v/$(sources/goproxy:escape "${version}").zip" "$3"
}

# sources/goproxy:hash prints the zip's h1:, as dirhash.HashZip makes it: a directory's entry is no file, and is left
# out.
sources/goproxy:hash() {
  local name sum listing
  listing=$(unzip -Z1 "$1") || return 1
  listing=$(LC_ALL=C sort <<<"${listing}")
  {
    while IFS= read -r name
    do
      [[ -n "${name}" && "${name}" != */ ]] || continue
      sum=$(unzip -p "$1" "${name}" | file:digest /dev/stdin) || return 1
      printf '%s  %s\n' "${sum}" "${name}"
    done <<<"${listing}"
  } > "$1.h1" || return 1
  printf 'h1:%s\n' "$(file:digest "$1.h1" | perl -MMIME::Base64 -ne 'chomp; print encode_base64(pack("H*", $_), "")')"
  rm -f "$1.h1"
}

sources/goproxy:home() { printf 'https://proxy.golang.org\n'; }

sources/goproxy:keys() { printf 'private\n'; }

sources/goproxy:serves() { [[ "$1" == module:* ]]; }

# sources/goproxy:settings prints GOPROXY as the source's url, and GOPRIVATE and GONOSUMDB as its private patterns,
# joined by commas. direct is the proxy at its home and then Go's own direct, as Go's default is; off is off.
sources/goproxy:settings() {
  local private
  if jq --exit-status '.direct' <<<"${2:-{\}}" >/dev/null 2>&1
  then printf 'GOPROXY=%s,direct\n' "$1"
  else printf 'GOPROXY=%s\n' "$1"
  fi
  private=$(jq --raw-output '(.private // []) | if type == "array" then join(",") else tostring end' <<<"${2:-{\}}")
  [[ -z "${private}" ]] || printf 'GOPRIVATE=%s\nGONOSUMDB=%s\n' "${private}" "${private}"
}

sources/goproxy:versions() {
  local module status=0
  module=${2#module:}
  module=${module%@*}
  sources:get "$1/$(sources/goproxy:escape "${module}")/@v/list" - || status=$?
  return "${status}"
}

# The seam's calls end here; what follows is this provider's own.

# sources/goproxy:escape <path> prints the path case-encoded: each capital letter as ! and the letter in lower case.
sources/goproxy:escape() {
  printf '%s\n' "$1" | sed -e 's/\([A-Z]\)/!\1/g' | tr '[:upper:]' '[:lower:]'
}

# ----------------------------------------------------------------------------------------------------------------------
