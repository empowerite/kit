# shellcheck shell=bash
# toolchain-docker.sh - a container image as a toolchain: a Dockerfile holds the tools (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The provider bin/toolchain.sh loads for `provider: docker`, or for a toolchain.Dockerfile at the root; `file:`
# names another and `image:` the image's name, kit-toolchain by default. The image is built from the file and
# tagged by the file's digest, so a change to the file is a new image and an unchanged one is never rebuilt. A
# command runs in it with the checkout mounted at /work, as the caller's user, so what it writes is the caller's.
#
# What the file pins is its base: `FROM name:tag@sha256:<digest>`. outdated asks the registry for the tag's
# current digest and names every base whose pin is behind it, or that has no pin; update rewrites the pins. docker
# with buildx is the requirement.

toolchain-docker:file() {
  local file
  file=$(toolchain:field docker file)
  printf '%s\n' "${file:-toolchain.Dockerfile}"
}

toolchain-docker:require() {
  command -v docker >/dev/null 2>&1 && return 0
  error "install Docker: https://docs.docker.com/get-docker/ (the docker toolchain needs it)"
  return 1
}

# toolchain-docker:tag prints the image's tag: its name and the first twelve hex digits of the file's digest.
toolchain-docker:tag() {
  local file name
  file=$(toolchain-docker:file)
  name=$(toolchain:field docker image)
  if [[ ! -f "${file}" ]]
  then
    error "${file} is missing; the docker toolchain builds its image from it"
    return 1
  fi
  printf '%s:%s\n' "${name:-kit-toolchain}" "$(release:digest < "${file}" | cut -c 1-12)"
}

toolchain-docker:install() {
  local file tag
  toolchain-docker:require || return 1
  file=$(toolchain-docker:file)
  tag=$(toolchain-docker:tag) || return 1
  docker image inspect "${tag}" >/dev/null 2>&1 && return 0
  docker build --quiet --file "${file}" --tag "${tag}" "$(dirname "${file}")" >/dev/null
}

# toolchain-docker:bases prints "<name:tag>\t<pinned digest or nothing>" for every FROM line naming an image: not
# scratch, and not an earlier stage of the same file named by `AS <stage>`.
toolchain-docker:bases() {
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
    }' "$(toolchain-docker:file)"
}

# toolchain-docker:latest <name:tag> prints the digest the registry now gives the tag.
toolchain-docker:latest() {
  docker buildx imagetools inspect "$1" --format '{{json .Manifest}}' 2>/dev/null | jq --raw-output '.digest'
}

toolchain-docker:outdated() {
  local base pinned latest
  toolchain-docker:require || return 1
  while IFS=$'\t' read -r base pinned
  do
    [[ -n "${base}" ]] || continue
    latest=$(toolchain-docker:latest "${base}") || { warn "${base}: the registry did not answer"; continue; }
    [[ "${latest}" == "${pinned}" ]] || printf '%s\t%s\t%s\n' "${base}" "${pinned:-unpinned}" "${latest}"
  done < <(toolchain-docker:bases)
}

toolchain-docker:update() {
  local file base pinned latest quoted from
  file=$(toolchain-docker:file)
  while IFS=$'\t' read -r base pinned latest
  do
    [[ -n "${base}" ]] || continue
    # An image reference never holds a comma, so a comma delimits each sed command.
    quoted=$(ere:quote "${base}")
    if [[ "${pinned}" == unpinned ]]
    then
      from='^([[:space:]]*[Ff][Rr][Oo][Mm][[:space:]]+(--platform=[^[:space:]]+[[:space:]]+)?)'
      sed -i.kit-old -E "s,${from}${quoted}([[:space:]]|\$),\\1${base}@${latest}\\3," "${file}"
    else
      sed -i.kit-old "s,${quoted}@${pinned},${base}@${latest}," "${file}"
    fi
    rm -f "${file}.kit-old"
  done < <(toolchain-docker:outdated)
  toolchain-docker:install
}

# The tools live in the image, so nothing is put on PATH here; a command reaches them through run.
toolchain-docker:path() { return 0; }

toolchain-docker:run() {
  local tag
  toolchain-docker:require || return 1
  tag=$(toolchain-docker:tag) || return 1
  toolchain-docker:install || return 1
  docker run --rm --volume "${PWD}:/work" --workdir /work --user "$(id -u):$(id -g)" --env HOME=/tmp \
    "${tag}" "$@"
}

# ----------------------------------------------------------------------------------------------------------------------
