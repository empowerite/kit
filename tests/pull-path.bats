#!/usr/bin/env bats
# pull-path.bats - tools found on the pull path: a GitHub release's file, a Go module, an npm package, by any source
# ----------------------------------------------------------------------------------------------------------------------
# tests/fakes/fake-curl stands in for curl: it serves ${FAKE_RELEASES}/<host>/<name> before ${FAKE_RELEASES}/<name>,
# so a mirror and the open internet can serve one file differently, and fails to connect to a host named in
# ${FAKE_RELEASES}/unreachable. Every artifact here is made on disk, and its hash is checked as kit checks it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  fake-tools gh
  export FAKE_RELEASES="${BATS_TEST_TMPDIR}/releases"
  mkdir -p "${FAKE_RELEASES}"
  ln -sf "${BATS_TEST_DIRNAME}/fakes/fake-curl" "${FAKE_DIR}/bin/curl"
  unset CI KIT_SOURCES
  fresh-repo
  platform=$(kit-bash 'use toolchain/kit && toolchain/kit:platform')
  os=${platform%-*} arch=${platform#*-}
}

# release <version> [<host>] serves gh's file for this platform at the version, a tarball whose command is under
# gh_<version>/bin/, from every host, or from <host> alone, and prints the file's name.
release() {
  local stage="${BATS_TEST_TMPDIR}/stage-gh-$1-${2:-any}" file="gh_$1_${os}_${arch}.tar.gz" dir=${FAKE_RELEASES}
  mkdir -p "${stage}/gh_$1/bin"
  printf '#!/bin/sh\necho gh %s %s\n' "$1" "${2:-any}" > "${stage}/gh_$1/bin/gh"
  chmod +x "${stage}/gh_$1/bin/gh"
  [[ -z "${2:-}" ]] || dir="${FAKE_RELEASES}/$2"
  mkdir -p "${dir}"
  tar --create --gzip --file "${dir}/${file}" --directory "${stage}" "gh_$1"
  printf '%s\n' "${file}"
}

# pin-gh <version> writes kit.yml's tools: with gh as a GitHub release's file, for this platform alone.
pin-gh() {
  cat > kit.yml <<YAML
tools:
  gh:
    version: "$1"
    release: cli/cli
    asset: gh_{version}_{os}_{arch}.tar.gz
    bin: gh_{version}/bin
    platforms: [${platform}]
YAML
}

# tool <name> runs the installed tool of that name from the kit toolchain's path.
tool() {
  PATH="$("${kit_root}/bin/kit" tools path --toolchain=kit | paste -s -d ':' -):${PATH}" "$@"
}

@test "a release's file is found on github.com, locked as what it is with its sha256, and never where it came from" {
  file=$(release 2.1.0)
  pin-gh 2.1.0
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp "https://github.com/cli/cli/releases/download/v2.1.0/${file}" "${FAKE_RELEASES}/curl.calls"
  grep --quiet --line-regexp $'gh\t2.1.0\t'"${platform}"$'\tboth\trelease:cli/cli@v2.1.0/'"${file}"$'\tsha256:'"$(shasum -a 256 "${FAKE_RELEASES}/${file}" | cut -d ' ' -f 1)"$'\tgh_2.1.0/bin' tools.lock
  [ "$(tool gh)" = "gh 2.1.0 any" ]
}

@test "a mirror in the workspace is asked first, and a desk that ends its path at off never reaches the internet" {
  file=$(release 2.1.0 mirror.example)
  pin-gh 2.1.0
  printf 'sources:\n  - name: mirror\n    kind: github-releases\n    url: https://mirror.example/github\n  - off\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [ "$(cat "${FAKE_RELEASES}/curl.calls")" = "https://mirror.example/github/cli/cli/releases/download/v2.1.0/${file}" ]
  [ "$(tool gh)" = "gh 2.1.0 mirror.example" ]
  # a runner with its own path installs the same bytes the lock pins, from the internet
  rm -rf "${KIT_HOME}/.kit/tools" "${XDG_CACHE_HOME}/kit/downloads" "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  cp "${FAKE_RELEASES}/mirror.example/${file}" "${FAKE_RELEASES}/${file}"
  CI=true KIT_SOURCES=direct run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [ "$(tail -n 1 "${FAKE_RELEASES}/curl.calls")" = "https://github.com/cli/cli/releases/download/v2.1.0/${file}" ]
}

@test "a source that serves other bytes, or cannot be reached, is named and passed by, and the next one's are installed" {
  release 2.1.0 >/dev/null
  pin-gh 2.1.0
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  rm -rf "${KIT_HOME}/.kit/tools" "${XDG_CACHE_HOME}/kit/downloads"
  release 2.1.0 mirror.example >/dev/null
  printf 'down.example\n' > "${FAKE_RELEASES}/unreachable"
  KIT_SOURCES='down=github-releases:https://down.example,mirror=github-releases:https://mirror.example,direct' \
    run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [[ "$output" == *"mirror serves release:cli/cli@v2.1.0/gh_2.1.0_${os}_${arch}.tar.gz with the hash sha256:"*", not the sha256:"*" the lock pins; kit passed it by"* ]]
  [ "$(tool gh)" = "gh 2.1.0 any" ]
  rm -rf "${KIT_HOME}/.kit/tools" "${XDG_CACHE_HOME}/kit/downloads"
  KIT_SOURCES='down=github-releases:https://down.example,mirror=github-releases:https://mirror.example,off' \
    run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"no source on the pull path served release:cli/cli@v2.1.0/gh_2.1.0_${os}_${arch}.tar.gz: down could not be reached; mirror served other bytes; the path ends at off"* ]]
  [ ! -d "${KIT_HOME}/.kit/tools/gh" ]
}

@test "a source's token is sent from KIT_SOURCE_TOKEN_<NAME>, and netrc is read where there is one" {
  release 2.1.0 >/dev/null
  pin-gh 2.1.0
  KIT_SOURCES='corp-files=github-releases:https://files.example' KIT_SOURCE_TOKEN_CORP_FILES=s3cret \
    run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  [ "$(cat "${FAKE_RELEASES}/curl.headers")" = "Authorization: Bearer s3cret" ]
  [[ "$output" != *"s3cret"* ]]
}

@test "outdated and update move to the newest version the path serves, its tag's own shape kept, a pre-release passed by" {
  release 2.1.0 >/dev/null
  pin-gh 2.1.0
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  mkdir -p "${FAKE_RELEASES}/mirror.example"
  printf '[{"tag_name": "v2.3.0-rc1", "draft": false}, {"tag_name": "v2.4.0", "draft": true}, {"tag_name": "v2.2.0", "draft": false}, {"tag_name": "v2.1.0", "draft": false}]\n' \
    > "${FAKE_RELEASES}/mirror.example/releases?per_page=100"
  export KIT_SOURCES='mirror=github-releases:https://mirror.example,off'
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet -E '^🔺 gh +· 2\.1\.0 +· 2\.1\.0 +· 2\.2\.0$' <<<"$output"
  grep --quiet --line-regexp 'https://mirror.example/api/v3/repos/cli/cli/releases?per_page=100' "${FAKE_RELEASES}/curl.calls"
  release 2.2.0 mirror.example >/dev/null
  run "${kit_root}/bin/kit" tools update --toolchain=kit
  [ "$status" -eq 0 ]
  [ "$(yq '.tools.gh.version' kit.yml)" = 2.2.0 ]
  [ "$(tool gh)" = "gh 2.2.0 mirror.example" ]
  # on github.com the tags come through gh
  unset KIT_SOURCES
  route gh 'api repos/cli/cli/releases?per_page=100' 'tags'
  printf '[{"tag_name": "v2.5.0", "draft": false}]\n' | fixture tags
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  grep --quiet -E '^🔺 gh +· 2\.2\.0 +· 2\.2\.0 +· 2\.5\.0$' <<<"$output"
}

@test "a Go module is found on a proxy, its path case-encoded, locked by its h1:, and its commands made executable" {
  local stage="${BATS_TEST_TMPDIR}/module" goos=${os/macos/darwin} goarch=${arch/x64/amd64} version
  version="v0.0.1-go1.27.3.${goos}-${goarch}"
  mkdir -p "${stage}/golang.org/toolchain@${version}/bin" "${stage}/golang.org/toolchain@${version}/pkg/tool/x"
  printf '#!/bin/sh\necho go 1.27.3\n' > "${stage}/golang.org/toolchain@${version}/bin/go"
  printf '#!/bin/sh\necho compile\n' > "${stage}/golang.org/toolchain@${version}/pkg/tool/x/compile"
  chmod -x "${stage}/golang.org/toolchain@${version}/bin/go"
  mkdir -p "${FAKE_RELEASES}/proxy.example"
  (cd "${stage}" && zip --quiet --recurse-paths "${FAKE_RELEASES}/proxy.example/${version}.zip" golang.org)
  cat > kit.yml <<YAML
tools:
  go:
    version: "1.27.3"
    module: golang.org/toolchain
    module_version: v0.0.1-go{version}.{os}-{arch}
    os: {macos: darwin}
    arch: {x64: amd64}
    bin: golang.org/toolchain@v0.0.1-go{version}.{os}-{arch}/bin
    platforms: [${platform}]
  widget:
    version: "1.0.0"
    module: example.com/Acme/Widget
    bin: example.com/Acme/Widget@v1.0.0/bin
    platforms: [${platform}]
YAML
  # a zip names its files by the module's own path, its capitals kept; the proxy's address escapes them
  mkdir -p "${stage}/m/example.com/Acme/Widget@v1.0.0/bin"
  printf '#!/bin/sh\necho widget\n' > "${stage}/m/example.com/Acme/Widget@v1.0.0/bin/widget"
  (cd "${stage}/m" && zip --quiet --recurse-paths "${FAKE_RELEASES}/proxy.example/v1.0.0.zip" example.com)
  export KIT_SOURCES='corp-go=goproxy:https://proxy.example/go,off'
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp "https://proxy.example/go/golang.org/toolchain/@v/${version}.zip" "${FAKE_RELEASES}/curl.calls"
  grep --quiet --line-regexp 'https://proxy.example/go/example.com/!acme/!widget/@v/v1.0.0.zip' "${FAKE_RELEASES}/curl.calls"
  hash=$(awk -F '\t' '$1 == "go" { print $6 }' tools.lock)
  [[ "${hash}" =~ ^h1:[A-Za-z0-9+/]{43}=$ ]]
  [ "$(awk -F '\t' '$1 == "go" { print $5 }' tools.lock)" = "module:golang.org/toolchain@${version}" ]
  [ "$(tool go)" = "go 1.27.3" ]
  [ -x "${KIT_HOME}/.kit/tools/go/1.27.3/${platform}/golang.org/toolchain@${version}/pkg/tool/x/compile" ]
  [ "$(tool widget)" = widget ]
  # the hash is Go's dirhash of the zip's files, whatever the zip's own bytes
  rm -rf "${KIT_HOME}/.kit/tools" "${XDG_CACHE_HOME}/kit/downloads" "${FAKE_RELEASES}/proxy.example/${version}.zip"
  (cd "${stage}" && zip --quiet -9 --recurse-paths "${FAKE_RELEASES}/proxy.example/${version}.zip" golang.org)
  CI=true run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  printf 'v0.0.1-go1.27.3.%s-%s\nv0.0.1-go1.28.0.%s-%s\nv0.0.1-go1.28rc1.%s-%s\n' "${goos}" "${goarch}" \
    "${goos}" "${goarch}" "${goos}" "${goarch}" > "${FAKE_RELEASES}/proxy.example/list"
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  grep --quiet -E '^🔺 go +· 1\.27\.3 +· 1\.27\.3 +· 1\.28\.0$' <<<"$output"
}

@test "h1: is Go's dirhash: the sha256 of each file's sha256 and name, in name order, in base64" {
  mkdir -p z/example.com/tool@v1.2.0/bin
  printf '#!/bin/sh\necho tool 1.2.0\n' > z/example.com/tool@v1.2.0/bin/tool
  printf 'module example.com/tool\n' > z/example.com/tool@v1.2.0/go.mod
  (cd z && zip --quiet --recurse-paths ../mod.zip example.com)
  # what golang.org/x/mod/sumdb/dirhash.HashZip gives for these two files
  [ "$(kit-bash 'use sources sources/goproxy && sources/goproxy:hash mod.zip')" = 'h1:yCNzAx6IMct2qi7AHoCnqHPrRfjJVHoMjIaKLlm2h0k=' ]
}

@test "an npm package is found in the registry its document names, locked by its integrity, its bin made executable" {
  local stage="${BATS_TEST_TMPDIR}/npm"
  mkdir -p "${stage}/package/bin" "${FAKE_RELEASES}/registry.example"
  printf '#!/bin/sh\necho lint 1.0.0\n' > "${stage}/package/bin/lint"
  chmod -x "${stage}/package/bin/lint"
  mkdir -p "${FAKE_RELEASES}/files.example"
  tar --create --gzip --file "${FAKE_RELEASES}/files.example/lint-1.0.0.tgz" --directory "${stage}" package
  integrity="sha512-$(openssl dgst -sha512 -binary "${FAKE_RELEASES}/files.example/lint-1.0.0.tgz" | openssl base64 -A)"
  printf '{"versions": {"1.0.0": {"dist": {"tarball": "https://files.example/@acme/lint/-/lint-1.0.0.tgz"}}, "1.1.0": {}}}\n' \
    > "${FAKE_RELEASES}/registry.example/@acme%2flint"
  cat > kit.yml <<YAML
tools:
  lint:
    version: "1.0.0"
    npm: "@acme/lint"
    bin: package/bin
    platforms: [${platform}]
YAML
  # a registry that serves another scope alone does not serve the package
  printf 'sources:\n  - name: other\n    kind: npm\n    url: https://other.example\n    scope: "@other"\n  - name: corp-npm\n    kind: npm\n    url: https://registry.example\n    scope: "@acme"\n  - off\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  run ! grep --quiet 'other.example' "${FAKE_RELEASES}/curl.calls"
  [ "$(awk -F '\t' '$1 == "lint" { print $5 "\t" $6 }' tools.lock)" = "npm:@acme/lint@1.0.0"$'\t'"${integrity}" ]
  [ "$(tool lint)" = "lint 1.0.0" ]
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  grep --quiet -E '^🔺 lint +· 1\.0\.0 +· 1\.0\.0 +· 1\.1\.0$' <<<"$output"
}

@test "sources --env writes the same path for go and npm, and run runs with it; the default path writes nothing" {
  printf 'tools: {}\n' > kit.yml
  run "${kit_root}/bin/kit" tools sources --env
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  printf 'sources:\n  - name: corp-go\n    kind: goproxy\n    url: https://proxy.example/go\n    private: [github.com/acme/*, example.com/x]\n  - name: corp-npm\n    kind: npm\n    url: https://npm.example\n    scope: "@acme"\n  - direct\n' \
    >> kit.yml
  run "${kit_root}/bin/kit" tools sources --env
  [ "$status" -eq 0 ]
  [ "$output" = 'GOPROXY=https://proxy.example/go,https://proxy.golang.org,direct
GOPRIVATE=github.com/acme/*,example.com/x
GONOSUMDB=github.com/acme/*,example.com/x
npm_config_@acme:registry=https://npm.example
npm_config_registry=https://registry.npmjs.org' ]
  KIT_SOURCES='mirror=goproxy:https://mirror.example,off' run "${kit_root}/bin/kit" tools sources --env
  [ "$output" = 'GOPROXY=https://mirror.example,off' ]
  KIT_SOURCES='mirror=goproxy:https://mirror.example,off' run "${kit_root}/bin/kit" tools run --toolchain=kit \
    sh -c 'echo "$GOPROXY"'
  [ "$output" = 'https://mirror.example,off' ]
}

@test "a tool with no coordinate, or with two, or a release without its asset, is refused by name" {
  printf 'tools:\n  a:\n    version: "1"\n  b:\n    version: "1"\n    url: https://x/b\n    npm: b\n  c:\n    version: "1"\n    release: o/c\n' \
    > kit.yml
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml: each tool under tools: needs a version and one of url, release with its asset, module, or npm: a, b, c"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
