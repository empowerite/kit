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
  [[ "$output" == *"kit.yml: each tool under tools: needs a version and one of url, release with its asset, module, npm, archive, dist with its file, or image: a, b, c"* ]]
}

@test "behind a mirror, outdated says what is tagged upstream that the mirror does not serve, and a pin to it is pending" {
  release 2.1.0 >/dev/null
  pin-gh 2.1.0
  "${kit_root}/bin/kit" tools install --toolchain=kit >/dev/null 2>&1
  mkdir -p "${FAKE_RELEASES}/mirror.example"
  printf '[{"tag_name": "v2.2.0", "draft": false}, {"tag_name": "v2.1.0", "draft": false}]\n' \
    > "${FAKE_RELEASES}/mirror.example/releases?per_page=100"
  route gh 'api repos/cli/cli/releases?per_page=100' 'tags'
  printf '[{"tag_name": "v2.5.0", "draft": false}, {"tag_name": "v2.2.0", "draft": false}]\n' | fixture tags
  export KIT_SOURCES='mirror=github-releases:https://mirror.example,off'
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet -E '^ +name +· pinned +· installed +· latest +· tagged$' <<<"$output"
  grep --quiet -E '^🔺 gh +· 2\.1\.0 +· 2\.1\.0 +· 2\.2\.0 +· 2\.5\.0$' <<<"$output"
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit --json
  [ "$(jq --compact-output '[.latest, .tagged]' <<<"$output")" = '["2.2.0","2.5.0"]' ]
  pin-gh 2.5.0
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"gh: release:cli/cli@v2.5.0/gh_2.5.0_${os}_${arch}.tar.gz is tagged upstream, and the pull path does not serve it yet: it is pending"* ]]
  # on the open internet alone, nothing is behind a mirror
  unset KIT_SOURCES
  pin-gh 2.1.0
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  grep --quiet -E '^🔺 gh +· 2\.1\.0 +· 2\.1\.0 +· 2\.5\.0$' <<<"$output"
}

# stage-tool <name> <dir> makes a tarball at <dir>/<name>.tar.gz whose <name>/bin/<name> prints the name, and prints
# the tarball's path.
stage-tool() {
  local stage="${BATS_TEST_TMPDIR}/stage-$1"
  mkdir -p "${stage}/$1/bin" "$2"
  printf '#!/bin/sh\necho %s\n' "$1" > "${stage}/$1/bin/$1"
  chmod +x "${stage}/$1/bin/$1"
  tar --create --gzip --file "$2/$1.tar.gz" --directory "${stage}" "$1"
  printf '%s\n' "$2/$1.tar.gz"
}

@test "a GitHub tag's source tarball is an archive: pin, found where a release's files are, its tag v{version}" {
  file=$(stage-tool bats "${FAKE_RELEASES}")
  mv "${file}" "${FAKE_RELEASES}/v1.14.0.tar.gz"
  cat > kit.yml <<YAML
tools:
  bats:
    version: "1.14.0"
    archive: bats-core/bats-core
    bin: bats/bin
    platforms: [${platform}]
YAML
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'https://github.com/bats-core/bats-core/archive/refs/tags/v1.14.0.tar.gz' \
    "${FAKE_RELEASES}/curl.calls"
  [ "$(awk -F '\t' '{ print $5 }' tools.lock | grep -v '^artifact$' | tail -n 1)" = 'archive:bats-core/bats-core@v1.14.0' ]
  [ "$(tool bats)" = bats ]
}

@test "a project's download directory is a dist: pin, at its home or on a mirror of that dist alone, and lists its versions" {
  mkdir -p "${FAKE_RELEASES}/nodejs.org" "${FAKE_RELEASES}/mirror.example"
  file=$(stage-tool node "${FAKE_RELEASES}/mirror.example")
  mv "${file}" "${FAKE_RELEASES}/mirror.example/node-v24.22.2-${os/macos/darwin}-${arch}.tar.gz"
  cat > kit.yml <<YAML
tools:
  node:
    version: "24.22.2"
    dist: node
    file: v{version}/node-v{version}-{os}-{arch}.tar.gz
    os: {macos: darwin}
    bin: node/bin
    platforms: [${platform}]
YAML
  # a mirror of another dist is passed by; the node mirror serves it
  printf 'sources:\n  - name: gnu-mirror\n    kind: dist\n    url: https://gnu.example\n    dist: gnu\n  - name: node-mirror\n    kind: dist\n    url: https://mirror.example/node\n    dist: node\n  - off\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  run ! grep --quiet 'gnu.example' "${FAKE_RELEASES}/curl.calls"
  grep --quiet --line-regexp "https://mirror.example/node/v24.22.2/node-v24.22.2-${os/macos/darwin}-${arch}.tar.gz" \
    "${FAKE_RELEASES}/curl.calls"
  [ "$(tool node)" = node ]
  printf '[{"version": "v26.1.0", "lts": false}, {"version": "v24.23.0", "lts": "Krypton"}, {"version": "v24.22.2", "lts": "Krypton"}]\n' \
    > "${FAKE_RELEASES}/mirror.example/index.json"
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  grep --quiet -E '^🔺 node +· 24\.22\.2 +· 24\.22\.2 +· 24\.23\.0$' <<<"$output"
  # on the open internet, direct is the dist's home
  rm "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  [ "$(kit-bash 'use sources && sources:direct dist:node/v1/x.tar.gz && sources:direct dist:gnu/parallel/p.tar.bz2 && sources:direct dist:go/go1.tar.gz')" \
    = $'dist\thttps://nodejs.org/dist\ndist\thttps://ftp.gnu.org/gnu\ndist\thttps://go.dev/dl' ]
  # a GNU project's directory lists its tarballs, the newest found by version
  mkdir -p "${FAKE_RELEASES}/ftp.gnu.org"
  printf '<a href="parallel-20260922.tar.bz2">x</a> <a href="parallel-20261022.tar.bz2">y</a> <a href="parallel-latest.tar.bz2">z</a>\n' \
    > "${FAKE_RELEASES}/ftp.gnu.org/parallel"
  [ "$(sources_entry='{}' kit-bash 'use sources sources/dist && sources/dist:versions https://ftp.gnu.org/gnu dist:gnu/parallel/parallel-1.tar.bz2' | paste -s -d ' ' -)" \
    = '20260922 20261022' ]
}

@test "an OCI image's first layer is an image: pin, its pull token had from the realm a 401 names" {
  local layer="${BATS_TEST_TMPDIR}/layer" digest
  printf '#!/bin/sh\necho tool 1.0.0\n' > "${layer}"
  digest=sha256:$(shasum -a 256 "${layer}" | cut -d ' ' -f 1)
  mkdir -p "${FAKE_RELEASES}/ghcr.io" "${FAKE_RELEASES}/auth.example"
  # the manifest answers 401 until a token is sent; the fake serves the same name once a token is had
  printf '401\n' > "${FAKE_RELEASES}/ghcr.io/1.0.0.status"
  printf 'HTTP/2 401\r\nwww-authenticate: Bearer realm="https://auth.example/token",service="ghcr.io",scope="repository:acme/tool:pull"\r\n' \
    > "${FAKE_RELEASES}/ghcr.io/1.0.0.headers"
  # the fake serves a URL by its last part, here the scope's tool:pull
  printf '{"token": "t0k"}\n' > "${FAKE_RELEASES}/auth.example/tool:pull"
  printf '{"layers": [{"digest": "%s", "mediaType": "application/octet-stream"}]}\n' "${digest}" \
    > "${FAKE_RELEASES}/ghcr.io/1.0.0"
  cp "${layer}" "${FAKE_RELEASES}/ghcr.io/${digest}"
  cat > kit.yml <<YAML
tools:
  widget:
    version: "1.0.0"
    image: ghcr.io/acme/tool
    platforms: [${platform}]
YAML
  # the fake answers by name alone, so the second ask, with the token, is answered 200 by taking the status away
  # after the first; a shim curl does that
  mkdir -p "${BATS_TEST_TMPDIR}/shim"
  printf '#!/usr/bin/env bash\nfor a in "$@"; do [[ "$a" != "Authorization: Bearer t0k" ]] || rm -f "%s"; done\nexec "%s" "$@"\n' \
    "${FAKE_RELEASES}/ghcr.io/1.0.0.status" "${BATS_TEST_DIRNAME}/fakes/fake-curl" > "${BATS_TEST_TMPDIR}/shim/curl"
  chmod +x "${BATS_TEST_TMPDIR}/shim/curl"
  PATH="${BATS_TEST_TMPDIR}/shim:${PATH}" run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'https://ghcr.io/v2/acme/tool/manifests/1.0.0' "${FAKE_RELEASES}/curl.calls"
  grep --quiet --line-regexp 'https://auth.example/token?service=ghcr.io&scope=repository:acme/tool:pull' \
    "${FAKE_RELEASES}/curl.calls"
  grep --quiet --line-regexp "https://ghcr.io/v2/acme/tool/blobs/${digest}" "${FAKE_RELEASES}/curl.calls"
  grep --quiet "oci:ghcr.io/acme/tool:1.0.0"$'\t'"${digest}" tools.lock
  [ "$(tool widget)" = "tool 1.0.0" ]
  # a pull-through cache serves it under its prefix, and lists the tags
  mkdir -p "${FAKE_RELEASES}/cache.example"
  printf '{"tags": ["1.0.0", "1.2.0", "2.0.0-rc.1"]}\n' > "${FAKE_RELEASES}/cache.example/list"
  KIT_SOURCES='cache=oci:https://cache.example,off' run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  grep --quiet -E '^🔺 widget +· 1\.0\.0 +· 1\.0\.0 +· 1\.2\.0$' <<<"$output"
  printf 'sources:\n  - name: cache\n    kind: oci\n    url: https://cache.example\n    prefix: ghcr\n  - off\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run "${kit_root}/bin/kit" tools --outdated --toolchain=kit
  grep --quiet --line-regexp 'https://cache.example/v2/ghcr/acme/tool/tags/list' "${FAKE_RELEASES}/curl.calls"
}

@test "an Artifactory repository is asked as its type, a 403 is blocked, and its settings are its type's" {
  release 2.1.0 >/dev/null
  pin-gh 2.1.0
  mkdir -p "${FAKE_RELEASES}/arti.example"
  cp "${FAKE_RELEASES}/gh_2.1.0_${os}_${arch}.tar.gz" "${FAKE_RELEASES}/arti.example/"
  printf 'sources:\n  - name: corp\n    kind: artifactory\n    url: https://arti.example\n    repository: github-remote\n    type: github-releases\n  - name: corp-go\n    kind: artifactory\n    url: https://arti.example\n    repository: go-virtual\n    type: goproxy\n  - off\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp "https://arti.example/artifactory/github-remote/cli/cli/releases/download/v2.1.0/gh_2.1.0_${os}_${arch}.tar.gz" \
    "${FAKE_RELEASES}/curl.calls"
  run "${kit_root}/bin/kit" tools sources --env
  [ "$output" = 'GOPROXY=https://arti.example/artifactory/api/go/go-virtual,off' ]
  # Xray's block answers 403: the download is blocked, and the next source is not asked for it as for a miss
  rm -rf "${KIT_HOME}/.kit/tools" "${XDG_CACHE_HOME}/kit/downloads"
  printf '403\n' > "${FAKE_RELEASES}/arti.example/gh_2.1.0_${os}_${arch}.tar.gz.status"
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [ "$status" -eq 1 ]
  [[ "$output" == *"corp blocks it"* ]]
  printf 'sources:\n  - name: corp\n    kind: artifactory\n    url: https://arti.example\n    repository: r\n    type: svn\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run "${kit_root}/bin/kit" tools install --toolchain=kit
  [[ "$output" == *"an artifactory source's type: is github-releases, goproxy, npm, or dist, not 'svn'"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
