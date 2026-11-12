#!/usr/bin/env bats
# sources.bats - artifact sources: packages pinned by digest and looked up last first, and the pull path's layers
# ----------------------------------------------------------------------------------------------------------------------
# Each package source here is a tarball on disk, fetched by a file:// URL, so the tests need no network. A git: URL is
# checked by the address it names. A kind beyond kit-packages is a provider the repository adds on its bash module
# path, bash/sources/<kind>.sh, which fetches nothing.

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  world-repo installed
}

setup() {
  world-copy-repo installed
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

# source-tarball <name> <package> <note> makes a tarball whose packages/<package> delivers .kit/notes/<note>.md and
# bash/<name>.sh, and prints its path and its digest, tab-separated.
source-tarball() {
  local dir="${BATS_TEST_TMPDIR}/src-$1" digest
  mkdir -p "${dir}/top/packages/$2/files/repo/.kit/notes" "${dir}/top/bash"
  printf 'description: "the %s package from %s"\ndepends:\n  - kit\n' "$2" "$1" > "${dir}/top/packages/$2/package.yml"
  printf '%s\n' "# kit:generated -- a note, from the $2 package" "# see: kit packages remove $2" '' "$3" \
    > "${dir}/top/packages/$2/files/repo/.kit/notes/$3.md"
  printf '%s:hello() { printf "hello from %s\\n"; }\n' "$1" "$1" > "${dir}/top/bash/$1.sh"
  tar --create --gzip --file "${dir}/source.tar.gz" --directory "${dir}" top
  digest=$(shasum -a 256 "${dir}/source.tar.gz" | cut -d ' ' -f 1)
  printf '%s\tsha256:%s\n' "${dir}/source.tar.gz" "${digest}"
}

@test "a source's package installs, comes from its source, and its bash directory joins the module path" {
  IFS=$'\t' read -r file digest < <(source-tarball team guides first)
  printf 'sources:\n  - name: team\n    url: file://%s\n    digest: %s\nbash_path:\n  - kit:modules/bash/\n  - team:bash\n' \
    "${file}" "${digest}" >> kit.yml
  run kit packages add guides
  [ "$status" -eq 0 ]
  [ -f .kit/notes/first.md ]
  run kit packages guides
  [[ "$output" == *"guides"*"team"*"the guides package from team"* ]]
  run bash -c "source '${kit_checkout}/modules/bash/1st.sh' && use team && team:hello"
  [ "$output" = "hello from team" ]
  run kit check
  [ "$status" -eq 0 ]
}

@test "a source whose bytes have another digest is refused before anything of it is read" {
  IFS=$'\t' read -r file _ < <(source-tarball team guides first)
  printf 'sources:\n  - name: team\n    url: file://%s\n    digest: sha256:%064d\n' "${file}" 0 >> kit.yml
  run kit packages add guides
  [ "$status" -eq 1 ]
  [[ "$output" == *"the source team has digest sha256:"*", not the sha256:0000"* ]]
  [[ "$output" != *"is not one of the choices"* ]]
  [ ! -e "${XDG_CACHE_HOME}/kit/sources" ] || [ -z "$(ls "${XDG_CACHE_HOME}/kit/sources")" ]
  printf 'sources:\n  - name: bare\n    url: file://%s\n' "${file}" > kit.yml.extra
  sed -i.bak '/^sources:/,$d' kit.yml && rm kit.yml.bak && cat kit.yml.extra >> kit.yml
  run kit packages
  [[ "$output" == *"the source bare has no digest"* ]]
}

@test "of two sources with a package of one name, the later one's is the package" {
  IFS=$'\t' read -r one one_digest < <(source-tarball one guides first)
  IFS=$'\t' read -r two two_digest < <(source-tarball two guides second)
  printf 'sources:\n  - name: one\n    url: file://%s\n    digest: %s\n  - name: two\n    url: file://%s\n    digest: %s\n' \
    "${one}" "${one_digest}" "${two}" "${two_digest}" >> kit.yml
  run kit packages add guides
  [ "$status" -eq 0 ]
  [ -f .kit/notes/second.md ]
  [ ! -e .kit/notes/first.md ]
  run kit packages guides
  [[ "$output" == *"guides"*"two"* ]]
}

@test "a git: source names its forge's tarball of the tag" {
  run bash -c "source '${kit_checkout}/modules/bash/1st.sh' && use sources/kit-packages \
    && sources/kit-packages:tarball git:acme/rules@v1.2.0 && sources/kit-packages:tarball git:gitlab.com/acme/rules@v1.2.0 \
    && sources/kit-packages:tarball https://example.com/r.tar.gz"
  [ "${lines[0]}" = "https://codeload.github.com/acme/rules/tar.gz/refs/tags/v1.2.0" ]
  [ "${lines[1]}" = "https://gitlab.com/acme/rules/-/archive/v1.2.0/rules-v1.2.0.tar.gz" ]
  [ "${lines[2]}" = "https://example.com/r.tar.gz" ]
}

# provider <kind> writes a provider of the sources seam under bash/, on the module path, that serves nothing and takes
# the key mirror.
provider() {
  mkdir -p bash/sources
  {
    printf '# shellcheck shell=bash\n'
    printf 'sources/%s:%s() { return 4; }\n' "$1" fetch "$1" versions
    printf 'sources/%s:%s() { :; }\n' "$1" hash "$1" home "$1" settings
    printf 'sources/%s:keys() { printf "mirror\\n"; }\n' "$1"
    printf 'sources/%s:serves() { [[ "$1" == release:* ]]; }\n' "$1"
  } > "bash/sources/$1.sh"
  grep --quiet '^bash_path:' kit.yml || printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' >> kit.yml
}

# path prints the pull path's names and layers, one "<name>@<layer>" per line, as sources:path reads them.
path() {
  bash -c "source '${kit_checkout}/modules/bash/1st.sh' && use sources && sources:path" | cut -f 1,4 | tr '\t' @
}

@test "a source's kind is kit-packages where it is left out, and the pull path takes the other kinds, direct with none" {
  provider fake
  IFS=$'\t' read -r file digest < <(source-tarball team guides first)
  printf 'sources:\n  - name: team\n    url: file://%s\n    digest: %s\n' "${file}" "${digest}" >> kit.yml
  [ "$(path)" = "direct@kit.yml" ]
  printf '  - name: corp\n    kind: fake\n    url: https://corp.example/files\n    mirror: yes\n' >> kit.yml
  [ "$(path | paste -s -d ' ' -)" = "corp@kit.yml" ]
  run bash -c "source '${kit_checkout}/modules/bash/1st.sh' && use sources && sources:list | cut -d \$'\\x1f' -f 1"
  [ "$output" = "team" ]
  run kit packages add guides
  [ "$status" -eq 0 ]
  [ -f .kit/notes/first.md ]
  run kit tools sources --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output . <<<"$output")" = '{"name":"corp","kind":"fake","url":"https://corp.example/files","from":"kit.yml"}' ]
}

@test "the pull path is KIT_SOURCES over the workspace over kit.yml, a name met again is the first's, and off ends it" {
  provider fake
  printf 'sources:\n  - name: corp\n    kind: fake\n    url: https://corp.example\n  - direct\n' >> kit.yml
  [ "$(path | paste -s -d ' ' -)" = "corp@kit.yml direct@kit.yml" ]
  printf 'sources:\n  - name: mirror\n    kind: fake\n    url: https://mirror.example\n  - corp\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run path
  [[ "$output" == *"corp is no word of sources; a word is direct or off"* ]]
  printf 'sources:\n  - name: mirror\n    kind: fake\n    url: https://mirror.example\n  - name: corp\n    kind: fake\n    url: https://other.example\n' \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  [ "$(path | paste -s -d ' ' -)" = "mirror@workspace corp@workspace direct@kit.yml" ]
  export KIT_SOURCES='runner=fake:https://runner.example, direct'
  [ "$(path | paste -s -d ' ' -)" = "runner@env direct@env mirror@workspace corp@workspace" ]
  export KIT_SOURCES='runner=fake:https://runner.example,off'
  [ "$(path | paste -s -d ' ' -)" = "runner@env off@env" ]
  run kit tools sources
  [ "$status" -eq 0 ]
  [[ "${lines[1]}" == *"runner"*"fake"*"https://runner.example"*"env"* ]]
  [[ "${lines[2]}" == *"off"*"the path ends here"*"env"* ]]
  [ "${#lines[@]}" -eq 3 ]
}

@test "a source of a kind kit lacks, named twice, with a key its kind does not take, or a package source off kit.yml is refused" {
  provider fake
  printf 'sources:\n  - name: corp\n    kind: svn\n    url: https://corp.example\n' >> kit.yml
  run kit tools sources
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit.yml: the source corp has the kind svn, which no provider answers; the kinds are fake github-releases goproxy kit-packages npm"* ]]
  run kit toolchains
  [ "$status" -eq 1 ]
  [[ "$output" == *"the source corp has the kind svn"* ]]
  sed -i.bak 's/kind: svn/kind: fake/' kit.yml && rm kit.yml.bak
  printf '  - name: corp\n    kind: fake\n    url: https://again.example\n' >> kit.yml
  run kit tools sources
  [[ "$output" == *"kit.yml: the source corp is named twice"* ]]
  printf 'sources:\n  - name: corp\n    kind: fake\n    url: https://corp.example\n    digest: x\n' > "${BATS_TEST_TMPDIR}/s"
  sed -i.bak '/^sources:/,$d' kit.yml && rm kit.yml.bak && cat "${BATS_TEST_TMPDIR}/s" >> kit.yml
  run kit tools sources
  [[ "$output" == *"kit.yml: the source corp has the key digest, which a fake source does not take"* ]]
  sed -i.bak '/digest: x/d' kit.yml && rm kit.yml.bak
  printf '    expect: soon\n' >> kit.yml
  run kit tools sources
  [[ "$output" == *"the source corp has expect: soon, which is no duration, as 30m, 12h, or 1d"* ]]
  sed -i.bak 's/expect: soon/expect: 1d/' kit.yml && rm kit.yml.bak
  run kit tools sources
  [ "$status" -eq 0 ]
  printf 'sources:\n  - name: team\n    url: git:acme/team@v1\n    digest: sha256:%064d\n' 0 \
    > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run kit tools sources
  [ "$status" -eq 1 ]
  [[ "$output" == *"workspaces.yml: the source team is a package source, which kit.yml alone names"* ]]
  rm "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  KIT_SOURCES='runner' run kit tools sources
  [[ "$output" == *"KIT_SOURCES: runner is not direct, off, or <name>=<kind>:<url>"* ]]
  KIT_SOURCES='runner=fake:' run kit tools sources
  [[ "$output" == *"KIT_SOURCES: runner=fake: is not direct, off, or <name>=<kind>:<url>"* ]]
  KIT_SOURCES='Runner=fake:https://x' run kit tools sources
  [[ "$output" == *"KIT_SOURCES: the source \"Runner\" needs a name of lowercase letters"* ]]
}

@test "kit trust names each source by what it brings: packages, or tools by its kind and url" {
  printf 'sources:\n  - name: team\n    url: git:acme/team@v1\n  - name: corp\n    kind: goproxy\n    url: https://corp.example/go\n  - direct\n' \
    >> kit.yml
  run bash -c "source '${kit_checkout}/modules/bash/1st.sh' && use trust && trust:asks ."
  [[ "$output" == *"bring packages from the source team (sources)"* ]]
  [[ "$output" == *"fetch its tools from the goproxy source corp, at https://corp.example/go (sources)"* ]]
}

@test "kit new provider sources makes a kind from the seam's calls, and kit lint holds it to the template" {
  mkdir -p bash/sources
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' >> kit.yml
  run kit new provider sources mirror
  [ "$status" -eq 0 ]
  for call in fetch hash home keys serves settings versions
  do
    grep --quiet "^sources/mirror:${call}() {" bash/sources/mirror.sh
  done
  run kit lint bash/sources/mirror.sh
  [ "$status" -eq 0 ]
}

# ----------------------------------------------------------------------------------------------------------------------
