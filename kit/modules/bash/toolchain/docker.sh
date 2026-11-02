# shellcheck shell=bash
# docker.sh - the container image toolchain provider: a Dockerfile installs the tools (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# docker fills the project role. It is configured here by toolchain.Dockerfile at the repository root. It is never
# active: it runs a command in its container instead. The image is named kit-toolchain.
#
# The image is built from the file and tagged with the file's digest. A changed file builds a new image; an
# unchanged file is never rebuilt. A command runs in the container with the checkout mounted at /work, as the
# caller's user, so the files it writes belong to the caller.
#
# The file pins its base images: `FROM name:tag@sha256:<digest>`. outdated asks the registry for each tag's
# current digest and lists every base that has no pin or is pinned to an older digest. update rewrites the pins.
# The provider needs docker with buildx.

use file

toolchain/docker:active() { return 1; }

toolchain/docker:file() { printf 'toolchain.Dockerfile\n'; }

toolchain/docker:inputs() { printf 'tool\tdocker\nfile\ttoolchain.Dockerfile\n'; }

toolchain/docker:install() {
  local file tag
  toolchain/docker:require || return 1
  file=$(toolchain/docker:file)
  tag=$(toolchain/docker:tag) || return 1
  docker image inspect "${tag}" >/dev/null 2>&1 && return 0
  docker build --quiet --file "${file}" --tag "${tag}" "$(dirname "${file}")" >/dev/null
}

toolchain/docker:outdated() {
  local base pinned latest status=0
  toolchain/docker:require || return 1
  while IFS=$'\t' read -r base pinned
  do
    [[ -n "${base}" ]] || continue
    if ! latest=$(toolchain/docker:latest "${base}")
    then
      error "${base}: the registry did not answer"
      status=1
      continue
    fi
    [[ "${latest}" == "${pinned}" ]] || printf '%s\t%s\t%s\n' "${base}" "${pinned:-unpinned}" "${latest}"
  done < <(toolchain/docker:bases)
  return "${status}"
}

# The tools are in the image, so path prints nothing. A command reaches them through run.
toolchain/docker:path() { return 0; }

# toolchain/docker:pins prints each base image with its pinned digest, read from the file. Once the image built from
# the file is present, each base is installed at its pin, or as built when it has none. Only a local daemon is asked
# whether the image is present; with a remote one, nothing is reported installed.
toolchain/docker:pins() {
  local tag present=false base pinned
  toolchain/docker:require || return 1
  tag=$(toolchain/docker:tag) || return 1
  if toolchain/docker:local && docker image inspect "${tag}" >/dev/null 2>&1
  then
    present=true
  fi
  while IFS=$'\t' read -r base pinned
  do
    [[ -n "${base}" ]] || continue
    if [[ "${present}" == true ]]
    then printf '%s\t%s\t%s\n' "${base}" "${pinned}" "${pinned:-built}"
    else printf '%s\t%s\t\n' "${base}" "${pinned}"
    fi
  done < <(toolchain/docker:bases)
}

toolchain/docker:require() {
  command -v docker >/dev/null 2>&1 && return 0
  error "install Docker: https://docs.docker.com/get-docker/ (the docker toolchain needs it)"
  return 1
}

toolchain/docker:roles() { printf 'project\n'; }

toolchain/docker:update() {
  local file base pinned latest quoted from
  file=$(toolchain/docker:file)
  while IFS=$'\t' read -r base pinned latest
  do
    [[ -n "${base}" ]] || continue
    # An image reference never contains a comma, so each sed command uses a comma as its delimiter.
    quoted=$(ere:quote "${base}")
    if [[ "${pinned}" == unpinned ]]
    then
      from='^([[:space:]]*[Ff][Rr][Oo][Mm][[:space:]]+(--platform=[^[:space:]]+[[:space:]]+)?)'
      sed -i.kit-old -E "s,${from}${quoted}([[:space:]]|\$),\\1${base}@${latest}\\3," "${file}"
    else
      sed -i.kit-old "s,${quoted}@${pinned},${base}@${latest}," "${file}"
    fi
    rm -f "${file}.kit-old"
  done < <(toolchain/docker:outdated)
  toolchain/docker:install
}

# toolchain/docker:usable runs the docker client for its version, which asks nothing of the daemon.
toolchain/docker:usable() {
  [[ "$1" == project ]] || return 1
  command -v docker >/dev/null 2>&1 && docker --version >/dev/null 2>&1 && [[ -f toolchain.Dockerfile ]]
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/docker:tag prints the image's tag: the image name, then the first twelve hex digits of the file's
# digest.
toolchain/docker:tag() {
  local file
  file=$(toolchain/docker:file)
  if [[ ! -f "${file}" ]]
  then
    error "${file} is missing; the docker toolchain builds its image from it"
    return 1
  fi
  printf 'kit-toolchain:%s\n' "$(file:digest < "${file}" | cut -c 1-12)"
}

# toolchain/docker:bases prints "<name:tag>\t<pinned digest or nothing>" for every FROM line that names an image.
# It skips scratch and earlier stages of the same file (named by `AS <stage>`).
toolchain/docker:bases() {
  awk '
    toupper($1) == "FROM" {
      i = 2
      if ($i ~ /^--platform=/) i++
      ref = $i
      if (toupper($(i + 1)) == "AS") stage[tolower($(i + 2))] = 1
      if (ref == "scratch" || (tolower(ref) in stage)) next
      name = ref; digest = ""
      if (index(ref, "@") > 0) { name = substr(ref, 1, index(ref, "@") - 1); digest = substr(ref, index(ref, "@") + 1) }
      print name "\t" digest
    }' "$(toolchain/docker:file)"
}

# toolchain/docker:local succeeds when the daemon docker talks to is on this machine. DOCKER_HOST must be unset,
# empty, or a unix socket. The context in use is DOCKER_CONTEXT, or else the one docker context use saved in
# config.json. It must be the default, or one whose saved endpoint is a unix socket, as Docker Desktop's is.
toolchain/docker:local() {
  local config=${DOCKER_CONFIG:-${HOME}/.docker} context=${DOCKER_CONTEXT:-} host
  [[ -z "${DOCKER_HOST:-}" || "${DOCKER_HOST}" == unix://* ]] || return 1
  if [[ -z "${context}" && -f "${config}/config.json" ]]
  then
    context=$(jq --raw-output '.currentContext // empty' "${config}/config.json") || return 1
  fi
  [[ -n "${context}" && "${context}" != default ]] || return 0
  # docker keeps a context's endpoint under the digest of its name; a context it cannot read is not local.
  host=$(jq --raw-output '.Endpoints.docker.Host // empty' \
    "${config}/contexts/meta/$(printf '%s' "${context}" | file:digest)/meta.json" 2>/dev/null) || return 1
  [[ "${host}" == unix://* ]]
}

# toolchain/docker:latest <name:tag> prints the tag's current digest in the registry.
toolchain/docker:latest() {
  docker buildx imagetools inspect "$1" --format '{{json .Manifest}}' 2>/dev/null | jq --raw-output '.digest'
}

toolchain/docker:run() {
  local tag
  toolchain/docker:require || return 1
  tag=$(toolchain/docker:tag) || return 1
  toolchain/docker:install || return 1
  docker run --rm --volume "${PWD}:/work" --workdir /work --user "$(id -u):$(id -g)" --env HOME=/tmp \
    "${tag}" "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
