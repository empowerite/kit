#!/usr/bin/env bats
# tools.bats - kit tools lists, installs, updates, and runs with what the chosen project toolchain pins
# ----------------------------------------------------------------------------------------------------------------------
# Each provider is driven by tests/fakes/fake-tool standing in for its tool, so what is proven is how kit tools reads
# the tool's answers and what it asks of it. What only the real tool can prove, that it answers as the fixtures do, is
# proven by running kit tools on a desk that has it.

load helper

setup() {
  hide-system-toolchains
  fake-tools mise nix brew docker devcontainer
  # a probe runs mise and docker once, to know each is installed; brew is never run, since it calls git
  route mise '--version' '!0'
  route docker '--version' '!0'
  # brew reads a Brewfile on macOS alone, so every run is on macOS unless a test says otherwise
  export OSTYPE=darwin24
  fresh-repo
}

# reached prints every call the fakes saw that could reach the network: outdated, imagetools, or upgrade.
reached() {
  cat "${FAKE_DIR}"/*.calls 2>/dev/null | grep -E 'outdated|imagetools|upgrade' || true
}

@test "kit tools lists what the chosen toolchain pins, with no network" {
  touch mise.toml
  fixture ls.json <<'JSON'
{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}],
 "just": [{"version": "1.40.0", "requested_version": "1.40", "installed": false}]}
JSON
  route mise 'ls --json --current' ls.json
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E 'name +· pinned +· installed$' <<<"$output"
  grep --quiet -E '^✅ jq +· 1\.8 +· 1\.8\.0$' <<<"$output"
  # a tool not installed has no glyph of an installed one, and nothing in its installed cell
  grep --quiet -E '^❌ just +· 1\.40$' <<<"$output"
  run "${kit_root}/bin/kit" tools --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output --slurp . <<<"$output")" \
    = '[{"name":"jq","pinned":"1.8","installed":"1.8.0","from":"mise"},{"name":"just","pinned":"1.40","installed":null,"from":"mise"}]' ]
  [ -z "$(reached)" ]
}

@test "kit tools --outdated --check fails when one is behind, naming kit tools update" {
  touch mise.toml
  fixture ls.json <<<'{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}]}'
  fixture outdated.json <<<'{"jq": {"requested": "1.8", "current": "1.8.0", "latest": "1.8.1"}}'
  route mise 'ls --json --current' ls.json
  route mise 'outdated --json' outdated.json
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 0 ]
  grep --quiet -E 'name +· pinned +· installed +· latest$' <<<"$output"
  grep --quiet -E '^🔺 jq +· 1\.8 +· 1\.8\.0 +· 1\.8\.1$' <<<"$output"
  [[ "$output" == *"run: kit tools update"* ]]
  run "${kit_root}/bin/kit" tools --outdated --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"1 tool(s) are behind; run: kit tools update"* ]]
  run "${kit_root}/bin/kit" tools --outdated --json
  [ "$(jq --compact-output . <<<"$output")" = '{"name":"jq","pinned":"1.8","installed":"1.8.0","latest":"1.8.1","from":"mise"}' ]
  run "${kit_root}/bin/kit" tools --check
  [ "$status" -eq 2 ]
}

@test "ambient, kit tools lists the rules' tools and where each is found, and install refuses" {
  mkdir -p .kit/rules .kit/styles
  printf -- '---\nlint: shellcheck --shell=bash\n---\n# Bash\n' > .kit/rules/bash.md
  printf -- '---\nlint: kit-test-absent-linter --strict\n---\n# Absent\n' > .kit/rules/absent.md
  printf -- '---\nlint: shellcheck\n---\n# Twice\n' > .kit/styles/twice.md
  printf -- '---\nkind:\n  - text\n---\n# No linter\n' > .kit/rules/text.md
  fake-tools shellcheck
  # a row is clipped to the terminal's width, and the test's path is long
  export COLUMNS=500
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E 'name +· found$' <<<"$output"
  grep --quiet -E "^✅ shellcheck +· ${FAKE_DIR}/bin/shellcheck$" <<<"$output"
  grep --quiet -E '^❌ kit-test-absent-linter +· missing$' <<<"$output"
  [ "$(grep --count shellcheck <<<"$output")" -eq 1 ]
  run "${kit_root}/bin/kit" tools --json
  [ "$(jq --compact-output --slurp 'sort_by(.name) | map([.name, .path])' <<<"$output")" \
    = "[[\"kit-test-absent-linter\",null],[\"shellcheck\",\"${FAKE_DIR}/bin/shellcheck\"]]" ]
  [ "$(jq --compact-output --slurp 'map(keys_unsorted) | unique' <<<"$output")" = '[["name","path","from"]]' ]
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 1 ]
  # each missing tool comes with the system toolchain's command that installs it, or else its name alone
  grep --quiet --line-regexp '    brew install kit-test-absent-linter' <<<"$output"
  [[ "$output" != *"shellcheck"* ]]
  "${kit_root}/bin/kit" toolchains choose ambient --system
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 1 ]
  grep --quiet --line-regexp '    kit-test-absent-linter' <<<"$output"
  [[ "$output" != *"brew install"* ]]
  # ambient, the path is empty and a command runs as is
  run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_root}/bin/kit" tools run printf ran
  [ "$status" -eq 0 ]
  [ "$output" = ran ]
  # --toolchain is kit's wherever it stands, so the usage names it before run; the command never sees it
  run "${kit_root}/bin/kit" tools --toolchain ambient run echo ran
  [ "$status" -eq 0 ]
  [ "$output" = ran ]
  run "${kit_root}/bin/kit" tools run echo ran --toolchain ambient
  [ "$status" -eq 0 ]
  [ "$output" = ran ]
}

@test "ambient, a found path too long for the terminal gives way at its start, so its file name shows" {
  mkdir -p .kit/rules
  printf -- '---\nlint: shellcheck --shell=bash\n---\n# Bash\n' > .kit/rules/bash.md
  fake-tools shellcheck
  export COLUMNS=60
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  line=$(grep '^✅ shellcheck' <<<"$output")
  [[ "${line}" == *" · …"*"/bin/shellcheck" ]]
  # a glyph is drawn two wide, so the row takes one character less than the width
  [ "${#line}" -eq 59 ]
}

@test "ambient, with every tool the rules need on PATH, install has nothing to do" {
  mkdir -p .kit/rules
  printf -- '---\nlint: shellcheck --shell=bash\n---\n# Bash\n' > .kit/rules/bash.md
  fake-tools shellcheck
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"every tool the rules need is on PATH"* ]]
}

@test "kit tools acts on the toolchain named for the run, and refuses one not usable here" {
  touch mise.toml toolchain.Dockerfile
  fixture paths.txt <<<'/m/jq/bin'
  route mise 'bin-paths' paths.txt
  # with two usable and no terminal, kit refuses and names the command that picks each
  run "${kit_root}/bin/kit" tools path </dev/null
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit toolchains choose mise"* ]]
  [[ "$output" == *"kit toolchains choose docker"* ]]
  KIT_TOOLCHAIN=mise run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [ "$output" = '/m/jq/bin' ]
  run "${kit_root}/bin/kit" tools path --toolchain brew
  [ "$status" -eq 1 ]
  [[ "$output" == *"brew"*"not usable"* ]]
}

@test "kit tools path prints the chosen toolchain's directories, then brew's only where it fills in" {
  touch mise.toml Brewfile
  fixture paths.txt <<<'/m/jq/bin'
  fixture prefix.txt <<<'/opt/brew'
  route mise 'bin-paths' paths.txt
  route brew '--prefix' prefix.txt
  run kit-bash 'use toolchain && toolchain:path'
  [ "$status" -eq 0 ]
  [ "$output" = $'/m/jq/bin\n/opt/brew/bin\n/opt/brew/sbin' ]
  # on Linux the Brewfile is not read, so only mise's directories are put first
  OSTYPE=linux-gnu run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [ "$output" = '/m/jq/bin' ]
  # docker runs commands in its container, so brew fills in for nothing there
  touch toolchain.Dockerfile
  run "${kit_root}/bin/kit" toolchains choose docker
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 0 ]
  [[ "$output" != *"/opt/brew"* ]]
}

@test "brew filling in lists, installs, and moves only the Brewfile's tools the chosen toolchain does not pin" {
  touch mise.toml Brewfile
  fixture ls.json <<<'{"jq": [{"version": "1.8.2", "requested_version": "1.8", "installed": true}]}'
  route mise 'ls --json --current' ls.json
  route mise 'install --quiet' '!0'
  route mise 'trust --quiet mise.toml' '!0'
  route mise 'outdated --json' empty.json
  route mise 'upgrade --quiet' '!0'
  fixture empty.json <<<'{}'
  fixture names.txt <<<$'jq\nbash\nuser/tap/direnv'
  fixture versions.txt <<<$'jq 1.7.1\nbash 5.3.20'
  fixture outdated.json <<'JSON'
{"formulae": [{"name": "jq", "installed_versions": ["1.7.1"], "current_version": "1.8.1"},
              {"name": "bash", "installed_versions": ["5.3.20"], "current_version": "5.3.21"}],
 "casks": []}
JSON
  route brew 'bundle list --file Brewfile --formula --cask' names.txt
  route brew 'list --versions' versions.txt
  route brew 'outdated --json=v2' outdated.json
  route brew 'install --quiet user/tap/direnv' '!0'
  route brew 'upgrade bash' '!0'
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet 'brew · Brewfile, for what mise does not pin' <<<"$output"
  grep --quiet -E '^✅ bash +· +· 5\.3\.20$' <<<"$output"
  grep --quiet -E '^❌ user/tap/direnv$' <<<"$output"
  [ "$(grep --count -E '^✅ jq' <<<"$output")" -eq 1 ]
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"brew installed the tools Brewfile pins that mise does not"* ]]
  grep --quiet --line-regexp 'install --quiet user/tap/direnv' "${FAKE_DIR}/brew.calls"
  ! grep --quiet 'bundle install' "${FAKE_DIR}/brew.calls"
  run "${kit_root}/bin/kit" tools --outdated
  grep --quiet -E '^🔺 bash ' <<<"$output"
  [[ "$output" != *"🔺 jq"* ]]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade bash' "${FAKE_DIR}/brew.calls"
  ! grep --quiet 'upgrade jq' "${FAKE_DIR}/brew.calls"
}

@test "mise installs, reports, updates, and puts its bin paths first" {
  touch mise.toml
  fixture ls.json <<<'{"jq": [{"version": "1.8.0", "requested_version": "1.8", "installed": true}]}'
  fixture outdated.json <<<'{"jq": {"requested": "1.8", "current": "1.8.0", "latest": "1.8.1"}}'
  fixture paths.txt <<<$'/m/jq/bin\n/m/just/bin'
  route mise 'trust*' '!0'
  route mise 'install --quiet' '!0'
  route mise 'ls --json --current' ls.json
  route mise 'outdated --json' outdated.json
  route mise 'upgrade --quiet' '!0'
  route mise 'bin-paths' paths.txt
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"mise: installing from mise.toml"* ]]
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 0 ]
  grep --quiet -E '^🔺 jq +· 1\.8 +· 1\.8\.0 +· 1\.8\.1$' <<<"$output"
  run "${kit_root}/bin/kit" tools --outdated --check
  [ "$status" -eq 1 ]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade --quiet' "${FAKE_DIR}/mise.calls"
  run "${kit_root}/bin/kit" tools path
  [ "$output" = $'/m/jq/bin\n/m/just/bin' ]
  run "${kit_root}/bin/kit" tools run bash -c 'printf "%s" "${PATH%%:*}"'
  [ "$output" = "/m/jq/bin" ]
}

@test "nix installs the dev shell, puts its PATH first, and lists the flake's inputs" {
  printf '{ outputs = { self }: { devShells.x.default = {}; }; }\n' > flake.nix
  fixture flakes.txt <<<'true'
  fixture lock.json <<'JSON'
{"version": 7, "root": "root", "nodes": {
  "root": {"inputs": {"nixpkgs": "nixpkgs", "utils": "flake-utils", "follows": ["nixpkgs"]}},
  "nixpkgs": {"locked": {"type": "github", "owner": "NixOS", "repo": "nixpkgs", "rev": "aaaaaaa1111111111"},
              "original": {"type": "github", "owner": "NixOS", "repo": "nixpkgs", "ref": "nixos-unstable"}},
  "flake-utils": {"locked": {"type": "github", "owner": "numtide", "repo": "flake-utils", "rev": "bbbbbbb2222222222"},
                  "original": {"type": "github", "owner": "numtide", "repo": "flake-utils"}}}}
JSON
  cp "${FAKE_DIR}/lock.json" flake.lock
  fixture meta-nixpkgs.json <<<'{"locked": {"rev": "ccccccc3333333333"}}'
  fixture meta-utils.json <<<'{"locked": {"rev": "bbbbbbb2222222222"}}'
  fixture env.json <<<'{"variables": {"PATH": {"type": "exported", "value": "/n/jq/bin:/n/just/bin"}}}'
  route nix 'eval --expr builtins ? getFlake' flakes.txt
  route nix 'develop --command true' '!0'
  route nix 'flake metadata --json github:NixOS/nixpkgs/nixos-unstable' meta-nixpkgs.json
  route nix 'flake metadata --json github:numtide/flake-utils' meta-utils.json
  route nix 'flake update' '!0'
  route nix 'print-dev-env --json' env.json
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  [[ "$output" == *"nix: installing from flake.nix"* ]]
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E '^✅ nixpkgs +· aaaaaaa +· aaaaaaa$' <<<"$output"
  grep --quiet -E '^✅ utils +· bbbbbbb +· bbbbbbb$' <<<"$output"
  [ -z "$(grep -E 'metadata|print-dev-env' "${FAKE_DIR}/nix.calls" | head -n 1)" ]
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 0 ]
  grep --quiet -E '^🔺 nixpkgs +· aaaaaaa +· aaaaaaa +· ccccccc$' <<<"$output"
  [[ "$output" != *utils* ]]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'flake update' "${FAKE_DIR}/nix.calls"
  run "${kit_root}/bin/kit" tools path
  [ "$output" = $'/n/jq/bin\n/n/just/bin' ]
  run "${kit_root}/bin/kit" tools run bash -c 'printf "%s" "${PATH%%:*}"'
  [ "$output" = "/n/jq/bin" ]
}

@test "nix path fails, naming why, when the dev shell cannot be built or sets no PATH" {
  printf '{ outputs = { self }: { devShells.x.default = {}; }; }\n' > flake.nix
  fixture flakes.txt <<<'true'
  fixture env.json <<<'{"variables": {"HOME": {"type": "exported", "value": "/h"}}}'
  route nix 'eval --expr builtins ? getFlake' flakes.txt
  route nix 'print-dev-env --json' '!1'
  run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 1 ]
  [[ "$output" == *"nix could not build the dev shell; see: nix develop"* ]]
  : > "${FAKE_DIR}/nix.routes"
  route nix 'eval --expr builtins ? getFlake' flakes.txt
  route nix 'print-dev-env --json' env.json
  run "${kit_root}/bin/kit" tools path
  [ "$status" -eq 1 ]
  [[ "$output" == *"the dev shell sets no PATH; see: nix print-dev-env"* ]]
}

@test "nix outdated skips a non-flake input, keeps dir, and reports an unreadable input without stopping, failing" {
  printf '{ outputs = { self }: { devShells.x.default = {}; }; }\n' > flake.nix
  fixture flakes.txt <<<'true'
  fixture lock.json <<'JSON'
{"version": 7, "root": "root", "nodes": {
  "root": {"inputs": {"src": "src", "mono": "mono", "gone": "gone", "nixpkgs": "nixpkgs"}},
  "src": {"flake": false, "locked": {"rev": "1111111aaaa"}, "original": {"type": "github", "owner": "o", "repo": "src"}},
  "mono": {"locked": {"rev": "2222222aaaa"}, "original": {"type": "github", "owner": "o", "repo": "mono", "dir": "sub"}},
  "gone": {"locked": {"rev": "3333333aaaa"}, "original": {"type": "github", "owner": "o", "repo": "gone"}},
  "nixpkgs": {"locked": {"rev": "4444444aaaa"}, "original": {"type": "github", "owner": "NixOS", "repo": "nixpkgs"}}}}
JSON
  cp "${FAKE_DIR}/lock.json" flake.lock
  fixture meta-mono.json <<<'{"locked": {"rev": "5555555bbbb"}}'
  fixture meta-nixpkgs.json <<<'{"locked": {"rev": "6666666cccc"}}'
  route nix 'eval --expr builtins ? getFlake' flakes.txt
  route nix 'flake metadata --json github:o/mono?dir=sub' meta-mono.json
  route nix 'flake metadata --json github:o/gone' '!1'
  route nix 'flake metadata --json github:NixOS/nixpkgs' meta-nixpkgs.json
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 1 ]
  [[ "$output" == *"nix could not read github:o/gone, the source of gone"* ]]
  grep --quiet -E '^🔺 mono +· 2222222 +· 2222222 +· 5555555$' <<<"$output"
  grep --quiet -E '^🔺 nixpkgs +· 4444444 +· 4444444 +· 6666666$' <<<"$output"
  [[ "$output" == *"nix: could not tell whether every tool is behind; see above"* ]]
  [ -z "$(grep 'github:o/src' "${FAKE_DIR}/nix.calls")" ]
  # with every readable input at its lock, the unreadable one still fails --check, never passes it
  fixture meta-mono.json <<<'{"locked": {"rev": "2222222aaaa"}}'
  fixture meta-nixpkgs.json <<<'{"locked": {"rev": "4444444aaaa"}}'
  run "${kit_root}/bin/kit" tools --outdated --check
  [ "$status" -eq 1 ]
  [[ "$output" == *"nix could not read github:o/gone, the source of gone"* ]]
  [[ "$output" != *"at its latest"* ]]
}

@test "brew installs from the Brewfile without upgrading, and reports and moves only the Brewfile's tools" {
  touch Brewfile
  fixture names.txt <<<$'jq\nshellcheck\nnode@22'
  fixture versions.txt <<<$'jq 1.7.1\nwget 1.24\nnode@22 22.11.0'
  fixture outdated.json <<'JSON'
{"formulae": [{"name": "jq", "installed_versions": ["1.7.1"], "current_version": "1.8.1"},
              {"name": "wget", "installed_versions": ["1.24"], "current_version": "1.25"}],
 "casks": []}
JSON
  fixture prefix.txt <<<'/opt/brew'
  route brew 'bundle install --file Brewfile --no-upgrade --quiet' '!0'
  route brew 'bundle list --file Brewfile --formula --cask' names.txt
  route brew 'list --versions' versions.txt
  route brew 'outdated --json=v2' outdated.json
  route brew 'upgrade jq' '!0'
  route brew '--prefix' prefix.txt
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  # a Brewfile pins by name; a name with @ pins that version
  grep --quiet -E '^✅ jq +· +· 1\.7\.1$' <<<"$output"
  grep --quiet -E '^❌ shellcheck$' <<<"$output"
  grep --quiet -E '^✅ node@22 +· 22 +· 22\.11\.0$' <<<"$output"
  [[ "$output" != *"wget"* ]]
  run "${kit_root}/bin/kit" tools --outdated
  grep --quiet -E '^🔺 jq +· +· 1\.7\.1 +· 1\.8\.1$' <<<"$output"
  [[ "$output" != *"wget"* ]]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade jq' "${FAKE_DIR}/brew.calls"
  run "${kit_root}/bin/kit" tools path
  [ "$output" = $'/opt/brew/bin\n/opt/brew/sbin' ]
}

@test "docker builds its image once per file digest, runs a command in it, and pins its bases" {
  printf 'FROM alpine:3.20@sha256:old AS tools\nRUN true\nFROM tools\nFROM debian:12\n' > toolchain.Dockerfile
  fixture alpine.json <<<'{"digest": "sha256:new"}'
  fixture debian.json <<<'{"digest": "sha256:deb"}'
  route docker 'image inspect *' '!1'
  route docker 'build --quiet --file toolchain.Dockerfile --tag kit-toolchain:* .' '!0'
  route docker 'run --rm *' '!0'
  route docker 'buildx imagetools inspect alpine:3.20 *' alpine.json
  route docker 'buildx imagetools inspect debian:12 *' debian.json
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E '^❌ alpine:3\.20 +· sha256:old$' <<<"$output"
  grep --quiet -E '^❌ debian:12$' <<<"$output"
  [ -z "$(reached)" ]
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  tag=$(sed -n 's/^build --quiet --file toolchain.Dockerfile --tag \(kit-toolchain:[0-9a-f]*\) \.$/\1/p' \
    "${FAKE_DIR}/docker.calls")
  [ "${#tag}" -eq 26 ]
  run "${kit_root}/bin/kit" tools run just ci
  [ "$status" -eq 0 ]
  call="run --rm --volume $(pwd -P):/work --workdir /work --user $(id -u):$(id -g) --env HOME=/tmp ${tag} just ci"
  grep --quiet -- "${call}" "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" tools --outdated
  grep --quiet -E '^🔺 alpine:3\.20 +· sha256:old +· +· sha256:new$' <<<"$output"
  grep --quiet -E '^🔺 debian:12 +· +· +· sha256:deb$' <<<"$output"
  [[ "$output" != *"🔺 tools"* ]]
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  pinned=$'FROM alpine:3.20@sha256:new AS tools\nRUN true\nFROM tools\nFROM debian:12@sha256:deb'
  [ "$(cat toolchain.Dockerfile)" = "${pinned}" ]
}

@test "docker update fails and changes no pin when the registry does not answer for a base" {
  printf 'FROM alpine:3.20@sha256:old\nFROM debian:12\n' > toolchain.Dockerfile
  fixture alpine.json <<<'{"digest": "sha256:new"}'
  route docker 'buildx imagetools inspect alpine:3.20 *' alpine.json
  route docker 'buildx imagetools inspect debian:12 *' '!1'
  route docker 'image inspect *' '!1'
  route docker 'build *' '!0'
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 1 ]
  [[ "$output" == *"debian:12: the registry did not answer"* ]]
  [[ "$output" == *"no pin was changed"* ]]
  [ "$(cat toolchain.Dockerfile)" = $'FROM alpine:3.20@sha256:old\nFROM debian:12' ]
  if grep --quiet '^build ' "${FAKE_DIR}/docker.calls"; then false; fi
}

@test "docker reports its bases installed only when its daemon is local, so the listing never reaches the network" {
  printf 'FROM alpine:3.20@sha256:old\n' > toolchain.Dockerfile
  route docker 'image inspect *' '!0'
  for host in '' unix:///var/run/docker.sock
  do
    DOCKER_HOST=${host} run "${kit_root}/bin/kit" tools
    [ "$status" -eq 0 ]
    grep --quiet -E '^✅ alpine:3\.20 +· sha256:old +· sha256:old$' <<<"$output"
  done
  : > "${FAKE_DIR}/docker.calls"
  DOCKER_HOST=tcp://build.example.com:2376 run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E '^❌ alpine:3\.20 +· sha256:old$' <<<"$output"
  if grep --quiet 'image inspect' "${FAKE_DIR}/docker.calls"; then false; fi
}

# docker-context <name> <host> saves a context the way docker context create does, under DOCKER_CONFIG.
docker-context() {
  local meta
  meta="${DOCKER_CONFIG}/contexts/meta/$(printf '%s' "$1" | shasum -a 256 | cut -d ' ' -f 1)"
  mkdir -p "${meta}"
  jq --null-input --arg name "$1" --arg host "$2" '{Name: $name, Endpoints: {docker: {Host: $host}}}' \
    > "${meta}/meta.json"
}

@test "docker's daemon is local only when the context in use, named or saved by docker context use, is" {
  printf 'FROM alpine:3.20@sha256:old\n' > toolchain.Dockerfile
  route docker 'image inspect *' '!0'
  export DOCKER_CONFIG="${BATS_TEST_TMPDIR}/docker"
  docker-context remote tcp://build.example.com:2376
  docker-context desktop "unix://${BATS_TEST_TMPDIR}/docker.sock"
  DOCKER_CONTEXT=remote run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E '^❌ alpine:3\.20 +· sha256:old$' <<<"$output"
  printf '{"currentContext": "remote"}\n' > "${DOCKER_CONFIG}/config.json"
  run "${kit_root}/bin/kit" tools
  grep --quiet -E '^❌ alpine:3\.20 +· sha256:old$' <<<"$output"
  if grep --quiet 'image inspect' "${FAKE_DIR}/docker.calls"; then false; fi
  for context in default desktop
  do
    printf '{"currentContext": "%s"}\n' "${context}" > "${DOCKER_CONFIG}/config.json"
    run "${kit_root}/bin/kit" tools
    grep --quiet -E '^✅ alpine:3\.20 +· sha256:old +· sha256:old$' <<<"$output"
  done
  DOCKER_CONTEXT=default run "${kit_root}/bin/kit" tools
  grep --quiet -E '^✅ alpine:3\.20 +· sha256:old +· sha256:old$' <<<"$output"
  printf '{"currentContext": "default"}\n' > "${DOCKER_CONFIG}/config.json"
  DOCKER_CONTEXT=remote run "${kit_root}/bin/kit" tools
  grep --quiet -E '^❌ alpine:3\.20 +· sha256:old$' <<<"$output"
}

@test "kit gate runs through the chosen toolchain, inside docker when docker is chosen" {
  mkdir .kit && printf 'gate: echo "${PATH%%%%:*}"\n' > kit.yml
  printf 'FROM alpine:3.20\n' > toolchain.Dockerfile
  touch mise.toml
  fixture paths.txt <<<'/m/tools/bin'
  route mise 'bin-paths' paths.txt
  route docker 'image inspect *' '!0'
  route docker 'run --rm *' '!0'
  # with mise and docker both usable and none chosen, the gate cannot run, and that is not the gate failing; the
  # step summary says why, in toolchain:chosen's words
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
  run "${kit_root}/bin/kit" gate </dev/null
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit toolchains choose docker"* ]]
  [[ "$output" != *"the gate failed"* ]]
  grep --quiet 'The gate cannot run' "${GITHUB_STEP_SUMMARY}"
  grep --quiet 'several toolchains are usable for the project here' "${GITHUB_STEP_SUMMARY}"
  KIT_TOOLCHAIN=brew run "${kit_root}/bin/kit" gate </dev/null
  [ "$status" -eq 1 ]
  grep --quiet 'the toolchain named for this run, brew, is not usable' "${GITHUB_STEP_SUMMARY}"
  run "${kit_root}/bin/kit" toolchains choose docker
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  grep --quiet --fixed-strings -- ' bash -c echo "${PATH%%:*}"' "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" toolchains choose mise
  [ "$status" -eq 0 ]
  : > "${FAKE_DIR}/docker.calls"
  run "${kit_root}/bin/kit" gate
  [ "$status" -eq 0 ]
  [[ "$output" == *"/m/tools/bin"* ]]
  if grep --quiet '^run ' "${FAKE_DIR}/docker.calls"; then false; fi
}

@test "kit toolchains no longer takes install, outdated, update, path, or run" {
  for verb in install outdated update path run
  do
    run "${kit_root}/bin/kit" toolchains "${verb}"
    [ "$status" -eq 2 ]
  done
}

# ----------------------------------------------------------------------------------------------------------------------

@test "outside, kit tools run runs inside the devcontainer; inside, it runs as is" {
  mkdir .devcontainer && printf '{}\n' > .devcontainer/devcontainer.json
  route devcontainer 'up --workspace-folder .' '!0'
  route devcontainer 'exec --workspace-folder . *' '!0'
  run "${kit_root}/bin/kit" tools install
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'up --workspace-folder \.' "${FAKE_DIR}/devcontainer.calls"
  run "${kit_root}/bin/kit" tools run just ci
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'exec --workspace-folder \. just ci' "${FAKE_DIR}/devcontainer.calls"
  run "${kit_root}/bin/kit" tools path
  [ -z "$output" ]
  # inside, the provider has a run and is active, so the command runs here, as is
  : > "${FAKE_DIR}/devcontainer.calls"
  REMOTE_CONTAINERS=true run "${kit_root}/bin/kit" tools run bash -c 'printf "%s" "${PATH}"'
  [ "$status" -eq 0 ]
  [ "$output" = "${PATH}" ]
  [ ! -s "${FAKE_DIR}/devcontainer.calls" ]
  REMOTE_CONTAINERS=true run "${kit_root}/bin/kit" tools run bash -c 'echo ran inside'
  [ "$output" = "ran inside" ]
}

@test "devcontainer lists its features with their locked versions, and outdated asks its CLI" {
  mkdir .devcontainer && printf '{ // features\n "features": {"ghcr.io/a/jq:1": {},},\n}\n' > .devcontainer/devcontainer.json
  cat > .devcontainer/devcontainer-lock.json <<'JSON'
{"features": {
  "ghcr.io/a/jq:1": {"version": "1.2.3", "resolved": "ghcr.io/a/jq@sha256:aa", "integrity": "sha256:aa"},
  "ghcr.io/a/go:1": {"version": "1.0.0", "resolved": "ghcr.io/a/go@sha256:bb", "integrity": "sha256:bb"}}}
JSON
  fixture outdated.json <<'JSON'
{"features": {
  "ghcr.io/a/jq:1": {"current": "1.2.3", "wanted": "1.2.4", "latest": "2.0.0"},
  "ghcr.io/a/go:1": {"current": "1.0.0", "wanted": "1.0.0", "latest": "1.0.0"}}}
JSON
  route devcontainer 'outdated --output-format json *' outdated.json
  route devcontainer 'upgrade *' '!0'
  run "${kit_root}/bin/kit" tools
  [ "$status" -eq 0 ]
  grep --quiet -E '^✅ ghcr.io/a/jq:1 +· 1\.2\.3 +· 1\.2\.3$' <<<"$output"
  grep --quiet -E '^✅ ghcr.io/a/go:1 +· 1\.0\.0 +· 1\.0\.0$' <<<"$output"
  [ ! -s "${FAKE_DIR}/devcontainer.calls" ]
  run "${kit_root}/bin/kit" tools --outdated
  [ "$status" -eq 0 ]
  grep --quiet -E '^🔺 ghcr.io/a/jq:1 +· 1\.2\.3 +· +1\.2\.3 +· 2\.0\.0$' <<<"$output"
  [[ "$output" != *ghcr.io/a/go* ]]
  grep --quiet --line-regexp 'outdated --output-format json --workspace-folder .' "${FAKE_DIR}/devcontainer.calls"
  run "${kit_root}/bin/kit" tools update
  [ "$status" -eq 0 ]
  grep --quiet --line-regexp 'upgrade --workspace-folder .' "${FAKE_DIR}/devcontainer.calls"
}
