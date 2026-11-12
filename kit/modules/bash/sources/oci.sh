# shellcheck shell=bash
# oci.sh - the oci source: an OCI registry, which serves an artifact as an image manifest's first layer (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# An oci: coordinate, oci:<registry>/<repository>:<tag>, is the first layer of the manifest the tag names, as oras
# pushes a file: through the registry's v2 API, the manifest at <url>/v2/<repository>/manifests/<tag> and the layer at
# <url>/v2/<repository>/blobs/<digest>. Its hash is the layer's sha256, which is its digest. The versions are the
# repository's tags, <url>/v2/<repository>/tags/list.
#
# direct is the registry the coordinate names, over https. A mirror, a pull-through cache, serves the repository
# under prefix:, as a cache project's name; without it, as the repository is named on its registry. A registry that
# answers 401 names, in WWW-Authenticate, where a pull token is had: kit asks there for the repository's pull scope,
# with the source's bearer token or netrc's login, and asks again with the token it is given. It takes prefix:.

sources/oci:fetch() {
  local repository tag manifest digest status=0
  sources/oci:split "$2" repository tag || return 4
  repository=$(sources/oci:prefixed "${repository}")
  manifest=$(sources/oci:get "$1" "${repository}" "manifests/${tag}" -) || return $?
  digest=$(jq --raw-output '.layers[0].digest // empty' <<<"${manifest}" 2>/dev/null) || digest=''
  [[ "${digest}" == sha256:* ]] || return 4
  sources/oci:get "$1" "${repository}" "blobs/${digest}" "$3" || status=$?
  return "${status}"
}

sources/oci:hash() { printf 'sha256:%s\n' "$(file:digest "$1")"; }

# sources/oci:home prints https:// and the registry the coordinate names, and nothing without one.
sources/oci:home() {
  local rest=${1:-}
  [[ "${rest}" == oci:*/* ]] || return 0
  rest=${rest#oci:}
  printf 'https://%s\n' "${rest%%/*}"
}

sources/oci:keys() { printf 'prefix\n'; }

sources/oci:serves() { [[ "$1" == oci:*/*:* ]]; }

# sources/oci:settings writes nothing: no language's manager reads an OCI registry.
sources/oci:settings() { :; }

sources/oci:versions() {
  local repository tag doc
  sources/oci:split "$2" repository tag || return 4
  repository=$(sources/oci:prefixed "${repository}")
  doc=$(sources/oci:get "$1" "${repository}" tags/list -) || return $?
  jq --raw-output '(.tags // [])[]' <<<"${doc}" 2>/dev/null || return 1
}

# The seam's calls end here; what follows is this provider's own.

# sources/oci:split <coordinate> <repository> <tag> sets the two variables to the repository, the registry left out,
# and the tag an oci: coordinate names, and returns 1 for one of another form.
sources/oci:split() {
  local -n sources_oci_repository=$2 sources_oci_tag=$3
  [[ "$1" =~ ^oci:[^/]+/(.+):([^:/]+)$ ]] || return 1
  # shellcheck disable=SC2034  # the caller's variables, through the names
  sources_oci_repository=${BASH_REMATCH[1]} sources_oci_tag=${BASH_REMATCH[2]}
}

# sources/oci:prefixed <repository> prints the repository as the source being asked serves it: under its entry's
# prefix:, sources_entry's, where it names one.
sources/oci:prefixed() {
  local prefix
  prefix=$(jq --raw-output '.prefix // ""' <<<"${sources_entry:-{\}}" 2>/dev/null) || prefix=''
  printf '%s%s\n' "${prefix:+${prefix%/}/}" "$1"
}

# sources/oci:get <url> <repository> <part> <file> fetches <url>/v2/<repository>/<part> into the file, or prints it for
# -, as sources:get answers: 4 for a miss, 1 when the registry cannot be reached. A 401 is answered once with a pull
# token from the realm its WWW-Authenticate names.
sources/oci:get() {
  local url=$1 repository=$2 part=$3 into=$4 status=0 code headers body challenge realm service token
  local -a accept=(--header 'Accept: application/vnd.oci.image.manifest.v1+json'
    --header 'Accept: application/vnd.docker.distribution.manifest.v2+json')
  file:temp headers oci-headers || return 1
  file:temp body oci-body || return 1
  code=$(curl --silent --location --netrc-optional "${accept[@]}" --dump-header "${headers}" --output "${body}" \
    --write-out '%{http_code}' "${url}/v2/${repository}/${part}") || status=$?
  (( status == 0 )) || { rm -f "${headers}" "${body}"; return 1; }
  if [[ "${code}" == 401 ]]
  then
    challenge=$(grep --ignore-case '^www-authenticate:' "${headers}" | head -n 1 | tr -d '\r') || challenge=''
    realm=$(sed -n -E 's/.*realm="([^"]*)".*/\1/p' <<<"${challenge}")
    service=$(sed -n -E 's/.*service="([^"]*)".*/\1/p' <<<"${challenge}")
    if [[ -z "${realm}" ]]
    then
      rm -f "${headers}" "${body}"
      return 4
    fi
    token=$(sources:get "${realm}?service=${service}&scope=repository:${repository}:pull" - \
      | jq --raw-output '.token // .access_token // empty' 2>/dev/null) || token=''
    if [[ -z "${token}" ]]
    then
      rm -f "${headers}" "${body}"
      return 4
    fi
    code=$(curl --silent --location "${accept[@]}" --header "Authorization: Bearer ${token}" --output "${body}" \
      --write-out '%{http_code}' "${url}/v2/${repository}/${part}") || status=$?
    (( status == 0 )) || { rm -f "${headers}" "${body}"; return 1; }
  fi
  rm -f "${headers}"
  if [[ "${code}" != 2* ]]
  then
    rm -f "${body}"
    return 4
  fi
  if [[ "${into}" == - ]]
  then
    cat "${body}"
    rm -f "${body}"
  else mv -f "${body}" "${into}"
  fi
}

# ----------------------------------------------------------------------------------------------------------------------
