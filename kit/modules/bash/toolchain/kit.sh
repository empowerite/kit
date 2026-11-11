# shellcheck shell=bash
# kit.sh - the kit toolchain provider: tools: in kit.yml pins the tools, tools.lock each download (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# kit fills the project role where kit.yml has tools:, or tools.lock is here. It needs curl and tar, and unzip for a
# download that is a zip. Each tool under tools: names its version, and the url of its download as a template:
#
#   tools:
#     actionlint:
#       version: "1.7.12"
#       url: https://github.com/rhysd/actionlint/releases/download/v{version}/actionlint_{version}_{os}_{arch}.tar.gz
#       os: {macos: darwin}
#       arch: {x64: amd64}
#
# {version}, {platform}, {os}, {arch}, and {ext} are filled in. kit names a platform <os>-<arch>, with linux or macos
# and x64 or arm64; os: and arch: rename them as the tool's downloads do, and ext: gives each os its download's
# ending, as {linux: tar.gz, macos: zip}. bin: names the directory, in what the url unpacks to, that holds the tool's
# commands, one of them named as the tool: . by default, filled in the same way. profile: desk or ci keeps a tool to
# one of them; a run where CI is set is a runner's, ci, and any other a desk's. platforms: narrows the platforms it is
# locked for.
#
# A url that ends in .tar.gz, .tgz, .tar.xz, .txz, .tar.bz2, or .tar is unpacked by tar, one that ends in .zip by
# unzip, and any other url is the command itself, saved under the tool's name.
#
# tools.lock, beside kit.yml, holds one line per tool and platform, tab-separated: name, version, platform, profile,
# url, sha256, and bin. install locks again each tool whose pin moved, downloading its file for every platform, and a
# runner refuses to. path and active read the lock alone, so a shell hook that asks for the path stays fast.
#
# A tool is installed under ${KIT_HOME:-$HOME}/.kit/tools/<name>/<version>/<platform>/, which every checkout shares,
# and each download is kept under ${XDG_CACHE_HOME:-$HOME/.cache}/kit/downloads/, named by its sha256. A download
# whose bytes have another digest than the lock's is refused, and nothing is installed from it.
#
# outdated asks a GitHub release's latest tag, for a release's download or a source archive, through gh where it is
# installed and GitHub's API otherwise; Node.js's latest LTS; Go's latest release; and the newest file a GNU mirror's
# directory lists. For any other url, it says it cannot tell.
#
# see: docs/toolchains/using-kit.md, issue #181

use file yaml

# toolchain_kit_platforms are the platforms a tool is locked for, unless its platforms: names others.
declare -gr toolchain_kit_platforms='linux-x64 linux-arm64 macos-x64 macos-arm64'

toolchain/kit:active() {
  toolchain/kit:configured || return 1
  toolchain/kit:on-path
}

toolchain/kit:file() { printf 'kit.yml\n'; }

toolchain/kit:inputs() { printf 'tool\tcurl\ntool\ttar\nfile\tkit.yml\nfile\ttools.lock\nenv\tCI\n'; }

# toolchain/kit:install locks again what moved, then installs each tool this platform and profile take that is not
# installed at the lock's digest. One tool that fails stops none of the rest.
toolchain/kit:install() {
  local row status=0
  local -a rows
  toolchain/kit:require || return 1
  toolchain/kit:lock || return 1
  list:read rows toolchain/kit:here || return 1
  for row in "${rows[@]}"
  do
    [[ -n "${row}" ]] || continue
    toolchain/kit:place "${row}" || status=1
  done
  return "${status}"
}

toolchain/kit:outdated() {
  local name version url latest row status=0 asked
  local -a tools
  list:read tools toolchain/kit:tools || return 1
  for row in "${tools[@]}"
  do
    IFS=$'\t' read -r name version url <<<"${row}"
    [[ -n "${name}" ]] || continue
    asked=0
    latest=$(toolchain/kit:latest "${url}") || asked=$?
    if (( asked == 2 ))
    then
      error "${name}: kit cannot tell the latest version from ${url}"
      status=1
    elif (( asked != 0 )) || [[ -z "${latest}" ]]
    then
      error "${name}: the latest version could not be read for ${url}"
      status=1
    elif [[ "${latest}" != "${version}" ]]
    then
      printf '%s\t%s\t%s\n' "${name}" "${version}" "${latest}"
    fi
  done
  return "${status}"
}

toolchain/kit:path() {
  local name version platform profile url digest bin row dir root
  local -a rows
  list:read rows toolchain/kit:here || return 1
  root=$(toolchain/kit:root)
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r name version platform profile url digest bin <<<"${row}"
    [[ -n "${name}" ]] || continue
    dir="${root}/${name}/${version}/${platform}"
    [[ "${bin}" == . ]] || dir+="/${bin}"
    printf '%s\n' "${dir}"
  done
}

# toolchain/kit:pins reads kit.yml, and never the network: each tool this platform and profile take, with its pin,
# and the pin again where it is installed.
toolchain/kit:pins() {
  local name version platform profile url bin row here mine root
  local -a rows
  here=$(toolchain/kit:platform) || return 1
  mine=$(toolchain/kit:profile)
  root=$(toolchain/kit:root)
  list:read rows toolchain/kit:wanted || return 1
  for row in "${rows[@]}"
  do
    IFS=$'\t' read -r name version platform profile url bin <<<"${row}"
    [[ -n "${name}" && "${platform}" == "${here}" ]] || continue
    [[ "${profile}" == both || "${profile}" == "${mine}" ]] || continue
    if [[ -f "${root}/${name}/${version}/${platform}/.kit-sha256" ]]
    then printf '%s\t%s\t%s\n' "${name}" "${version}" "${version}"
    else printf '%s\t%s\t\n' "${name}" "${version}"
    fi
  done
}

toolchain/kit:require() {
  local tool
  for tool in curl tar
  do
    command -v "${tool}" >/dev/null 2>&1 && continue
    error "install ${tool} (the kit toolchain downloads and unpacks the tools with it)"
    return 1
  done
}

toolchain/kit:roles() { printf 'project\n'; }

# toolchain/kit:update moves each tool behind to its latest version in kit.yml, then locks and installs it. A tool
# whose latest kit cannot read is left as it was, and the run fails once the others have moved.
toolchain/kit:update() {
  local behind name current latest status=0
  toolchain/kit:require || return 1
  behind=$(toolchain/kit:outdated) || status=1
  while IFS=$'\t' read -r name current latest
  do
    [[ -n "${name}" ]] || continue
    step "${name}: ${current} to ${latest}"
    kit_tool=${name} kit_version=${latest} yq --inplace '.tools[strenv(kit_tool)].version = strenv(kit_version)' \
      kit.yml || return 1
  done <<<"${behind}"
  toolchain/kit:install || return 1
  return "${status}"
}

toolchain/kit:usable() {
  [[ "$1" == project ]] || return 1
  command -v curl >/dev/null 2>&1 && command -v tar >/dev/null 2>&1 && toolchain/kit:configured
}

# The seam's calls end here; what follows is this provider's own.

# toolchain/kit:configured succeeds when tools.lock is here, or kit.yml has tools:.
toolchain/kit:configured() {
  [[ -f tools.lock ]] && return 0
  [[ -f kit.yml ]] && grep --quiet '^tools:' kit.yml
}

# toolchain/kit:on-path succeeds when the lock names a directory for this platform, and each is on PATH.
toolchain/kit:on-path() {
  local dir
  local -a dirs
  list:read dirs toolchain/kit:path || return 1
  (( ${#dirs[@]} )) || return 1
  for dir in "${dirs[@]}"
  do
    [[ ":${PATH}:" == *":${dir}:"* ]] || return 1
  done
}

# toolchain/kit:platform prints this machine's platform as kit names it: linux-x64, linux-arm64, macos-x64, or
# macos-arm64.
toolchain/kit:platform() {
  local os arch
  case "$(uname -s)" in
    (Darwin) os=macos ;;
    (Linux) os=linux ;;
    (*)
      error "the kit toolchain has no platform for $(uname -s); see: docs/toolchains/using-kit.md"
      return 1
  esac
  case "$(uname -m)" in
    (x86_64 | amd64) arch=x64 ;;
    (arm64 | aarch64) arch=arm64 ;;
    (*)
      error "the kit toolchain has no platform for $(uname -m); see: docs/toolchains/using-kit.md"
      return 1
  esac
  printf '%s-%s\n' "${os}" "${arch}"
}

# toolchain/kit:profile prints ci where CI is set, as a runner sets it, and desk anywhere else.
toolchain/kit:profile() {
  if [[ -n "${CI:-}" ]]
  then printf 'ci\n'
  else printf 'desk\n'
  fi
}

toolchain/kit:root() { printf '%s/.kit/tools\n' "${KIT_HOME:-${HOME}}"; }

toolchain/kit:cache() { printf '%s/kit/downloads\n' "${XDG_CACHE_HOME:-${HOME}/.cache}"; }

# toolchain/kit:spec prints the tools under tools: in kit.yml as one JSON array, each a version and a url as strings,
# whatever YAML made of them, with bin, profile, os, arch, and platforms filled in where kit.yml leaves them out.
toolchain/kit:spec() {
  yaml:eval kit.yml 'kit-toolchain spec' '[(.tools // {}) | to_entries[] | {
      "name": .key,
      "version": ((.value.version // "") | tostring),
      "url": ((.value.url // "") | tostring),
      "bin": ((.value.bin // ".") | tostring),
      "profile": ((.value.profile // "both") | tostring),
      "os": (.value.os // {}),
      "arch": (.value.arch // {}),
      "ext": (.value.ext // {}),
      "platforms": (.value.platforms // [])}]' --output-format=json --indent=0
}

# toolchain/kit:wanted prints what kit.yml asks the lock to hold: "<name>\t<version>\t<platform>\t<profile>\t<url>\t
# <bin>", one line per tool and platform. A tool without a version or a url is refused by name.
toolchain/kit:wanted() {
  local spec missing
  spec=$(toolchain/kit:spec) || return 1
  missing=$(jq --raw-output '[.[] | select(.version == "" or .url == "") | .name] | join(", ")' <<<"${spec:-[]}")
  if [[ -n "${missing}" ]]
  then
    error "kit.yml: each tool under tools: needs a version and a url, and ${missing} lacks one"
    return 1
  fi
  # shellcheck disable=SC2154  # kit_jq is exported by modules/bash/1st.sh
  jq -L "${kit_jq}" --raw-output --arg platforms "${toolchain_kit_platforms}" \
    'import "toolchain" as toolchain; toolchain::kit_wanted($platforms)' <<<"${spec:-[]}"
}

# toolchain/kit:tools prints "<name>\t<version>\t<url>" for each tool under tools:, the url as kit.yml writes it.
toolchain/kit:tools() {
  local spec
  spec=$(toolchain/kit:spec) || return 1
  jq --raw-output '.[] | [.name, .version, .url] | join("\t")' <<<"${spec:-[]}"
}

# toolchain/kit:locked prints the lines of tools.lock, its comments and blank lines left out.
toolchain/kit:locked() {
  [[ -f tools.lock ]] || return 0
  sed -e '/^#/d' -e '/^$/d' tools.lock
}

# toolchain/kit:here prints the lines of tools.lock that this platform and profile install.
toolchain/kit:here() {
  local platform profile locked
  platform=$(toolchain/kit:platform) || return 1
  profile=$(toolchain/kit:profile)
  locked=$(toolchain/kit:locked) || return 1
  [[ -n "${locked}" ]] || return 0
  awk -F '\t' -v platform="${platform}" -v profile="${profile}" \
    '$3 == platform && ($4 == "both" || $4 == profile)' <<<"${locked}"
}

# toolchain/kit:lock rewrites tools.lock when kit.yml asks for other lines than it holds: each tool whose pin moved is
# downloaded for every platform, a url they share once, and the lines of the rest are kept. Only this platform's
# downloads stay in the cache; the others were fetched to be hashed. A runner refuses a lock behind kit.yml.
toolchain/kit:lock() {
  local wanted locked name version platform profile url bin digest key entry names='' rows=0 here cache
  local -A have=() moved=() fetched=() kept=()
  local -a lines=()
  wanted=$(toolchain/kit:wanted) || return 1
  locked=$(toolchain/kit:locked) || return 1
  while IFS=$'\t' read -r name version platform profile url digest bin
  do
    [[ -n "${name}" ]] || continue
    have[${name}$'\t'${platform}]=${version}$'\t'${profile}$'\t'${url}$'\t'${bin}$'\t'${digest}
    rows=$(( rows + 1 ))
  done <<<"${locked}"
  while IFS=$'\t' read -r name version platform profile url bin
  do
    [[ -n "${name}" ]] || continue
    rows=$(( rows - 1 ))
    entry=${have[${name}$'\t'${platform}]:-}
    if [[ "${entry%$'\t'*}" != "${version}"$'\t'"${profile}"$'\t'"${url}"$'\t'"${bin}" && -z "${moved[${name}]:-}" ]]
    then
      moved[${name}]=1
      names+="${names:+, }${name}"
    fi
  done <<<"${wanted}"
  [[ -n "${names}" ]] || (( rows != 0 )) || return 0
  if [[ "$(toolchain/kit:profile)" == ci ]]
  then
    error "tools.lock is behind tools: in kit.yml for ${names:-the tools it no longer pins}; run kit tools install" \
      "at a desk, and commit tools.lock"
    return 1
  fi
  here=$(toolchain/kit:platform) || return 1
  cache=$(toolchain/kit:cache)
  while IFS=$'\t' read -r name version platform profile url bin
  do
    [[ -n "${name}" ]] || continue
    key=${name}$'\t'${platform}
    if [[ -z "${moved[${name}]:-}" ]]
    then digest=${have[${key}]##*$'\t'}
    elif [[ -n "${fetched[${url}]:-}" ]]
    then digest=${fetched[${url}]}
    else
      digest=$(toolchain/kit:fetch "${name}" "${url}") || return 1
      fetched[${url}]=${digest}
    fi
    [[ "${platform}" != "${here}" ]] || kept[${digest}]=1
    lines+=("${name}"$'\t'"${version}"$'\t'"${platform}"$'\t'"${profile}"$'\t'"${url}"$'\t'"${digest}"$'\t'"${bin}")
  done <<<"${wanted}"
  for digest in "${fetched[@]}"
  do
    [[ -n "${kept[${digest}]:-}" ]] || rm -f "${cache}/${digest}"
  done
  file:replace tools.lock printf '%s\n' \
    '# tools.lock - each tool'"'"'s download for every platform, written by kit tools install from tools: in kit.yml' \
    $'# name\tversion\tplatform\tprofile\turl\tsha256\tbin' "${lines[@]}" || return 1
  if [[ -n "${names}" ]]
  then ok "tools.lock: locked ${names}; commit it with kit.yml"
  else ok "tools.lock: left out the tools kit.yml no longer pins; commit it with kit.yml"
  fi
}

# toolchain/kit:fetch <name> <url> [<digest>] keeps the download of the url in the cache, named by its sha256, and
# prints the sha256. With a digest, a download already kept is not fetched again, and one with other bytes is refused.
toolchain/kit:fetch() {
  local name=$1 url=$2 want=${3:-} cache have download
  cache=$(toolchain/kit:cache)
  if [[ -n "${want}" && -f "${cache}/${want}" ]]
  then
    printf '%s\n' "${want}"
    return 0
  fi
  file:temp download "${name}" || return 1
  step "${name}: downloading ${url}"
  if ! curl --fail --silent --show-error --location --output "${download}" "${url}"
  then
    error "${name}: ${url} could not be downloaded"
    return 1
  fi
  have=$(file:digest "${download}")
  if [[ -n "${want}" && "${have}" != "${want}" ]]
  then
    error "${name}: ${url} has digest sha256:${have}, not the sha256:${want} tools.lock pins; nothing was installed"
    return 1
  fi
  mkdir -p "${cache}" && mv -f "${download}" "${cache}/${have}" || return 1
  printf '%s\n' "${have}"
}

# toolchain/kit:place <line> installs the tool a line of tools.lock names, unless it is installed at its digest: the
# download is unpacked apart, its bin: checked to hold a command named as the tool, and only then moved into place.
toolchain/kit:place() {
  local name version platform profile url digest bin dir stage
  IFS=$'\t' read -r name version platform profile url digest bin <<<"$1"
  dir="$(toolchain/kit:root)/${name}/${version}/${platform}"
  if [[ -f "${dir}/.kit-sha256" && "$(cat "${dir}/.kit-sha256")" == "${digest}" ]]
  then
    return 0
  fi
  digest=$(toolchain/kit:fetch "${name}" "${url}" "${digest}") || return 1
  file:scratch stage || return 1
  mkdir -p "${stage}/tree"
  if ! toolchain/kit:unpack "$(toolchain/kit:cache)/${digest}" "${url}" "${stage}/tree" "${name}"
  then
    error "${name}: ${url} could not be unpacked"
    return 1
  fi
  if [[ ! -d "${stage}/tree/${bin}" ]]
  then
    error "${name}: ${bin} is not in what ${url} unpacks to; set bin: under tools: ${name} in kit.yml"
    return 1
  fi
  if [[ ! -x "${stage}/tree/${bin}/${name}" ]]
  then
    error "${name}: ${bin} in what ${url} unpacks to holds no ${name} command; set bin: under tools: ${name} in kit.yml"
    return 1
  fi
  printf '%s\n' "${digest}" > "${stage}/tree/.kit-sha256"
  mkdir -p "$(dirname "${dir}")" && rm -rf "${dir}" && mv "${stage}/tree" "${dir}" || return 1
  step "${name}: ${version} installed"
}

# toolchain/kit:unpack <file> <url> <into> <name> unpacks the download as its url's ending says, or saves it as the
# command <name> when the url ends in no archive's suffix.
toolchain/kit:unpack() {
  case "${2%%\?*}" in
    (*.tar.gz | *.tgz | *.tar.xz | *.txz | *.tar.bz2 | *.tar) tar --extract --file "$1" --directory "$3" ;;
    (*.zip) unzip -q "$1" -d "$3" ;;
    (*) cp "$1" "$3/$4" && chmod +x "$3/$4"
  esac
}

# toolchain/kit:listed <url> prints the newest version among the files the url's directory lists whose names have the
# shape of its last part, {version} standing for digits and dots. The directory is asked for without its trailing
# slash, which a mirror redirects.
toolchain/kit:listed() {
  local url=$1 file pattern listing
  file=${url##*/}
  pattern=$(printf '%s' "${file}" | sed -e 's/[][\.*^$+?(){}|/]/\\&/g' -e 's/\\{version\\}/([0-9][0-9.]*[0-9])/')
  listing=$(curl --fail --silent --location "${url%/*}") || return 1
  grep --only-matching --extended-regexp "${pattern}" <<<"${listing}" \
    | sed -E "s/^${pattern}\$/\\1/" | sort --version-sort | tail -n 1
}

# toolchain/kit:latest <url> prints the latest version of the tool the url template downloads, with the leading
# letters of its tag taken off, as v1.2.0 or go1.27.0. It returns 2 for a url whose source it cannot ask, and 1 when
# the source does not answer.
toolchain/kit:latest() {
  local url=$1 repo tag
  case "${url}" in
    (https://github.com/*/releases/download/* | https://github.com/*/archive/*)
      repo=${url#https://github.com/}
      repo=${repo%%/releases/*}
      repo=${repo%%/archive/*}
      if command -v gh >/dev/null 2>&1
      then tag=$(gh api "repos/${repo}/releases/latest" --jq .tag_name) || return 1
      else
        tag=$(curl --fail --silent --location "https://api.github.com/repos/${repo}/releases/latest" \
          | jq --raw-output .tag_name) || return 1
      fi
      ;;
    (https://nodejs.org/dist/*)
      tag=$(curl --fail --silent --location https://nodejs.org/dist/index.json \
        | jq --raw-output '[.[] | select(.lts != false)][0].version') || return 1
      ;;
    (https://go.dev/dl/* | https://dl.google.com/go/*)
      tag=$(curl --fail --silent --location 'https://go.dev/dl/?mode=json' | jq --raw-output '.[0].version') || return 1
      ;;
    (https://ftp.gnu.org/gnu/*/*)
      tag=$(toolchain/kit:listed "${url}") || return 1
      ;;
    (*) return 2
  esac
  [[ -n "${tag}" && "${tag}" != null ]] || return 1
  printf '%s\n' "${tag}" | sed 's/^[^0-9]*//'
}

# ----------------------------------------------------------------------------------------------------------------------
