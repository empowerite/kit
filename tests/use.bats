#!/usr/bin/env bats
# use.bats - the module path: every <module>.sh on it is sourced in order, and a verb runs with or without bin/kit
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  mkdir .kit bash
}

teardown() {
  rm -rf "${repo}"
}

# kit-bash <code> runs <code> in a fresh bash that has sourced the loader.
kit-bash() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && eval "$1"' kit-bash "$1"
}

@test "with no bash_path setting, the path is the kit's own bash directory" {
  run kit-bash 'use:path'
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}/modules/bash" ]
}

@test "an entry is kit:, absolute, under ~/, or relative to the root; a missing one is skipped, as PATH skips it" {
  mkdir -p home/lib/bash abs
  printf 'bash_path:\n  - kit:modules/bash/   # the kit\x27s own\n  - nowhere/\n  - ~/lib/bash/\n  - %s/abs\n  - bash/\n' \
    "$(pwd -P)" > .kit/settings.yml
  run env HOME="$(pwd -P)/home" bash -c 'source "${kit_root}/modules/bash/1st.sh" && use:path'
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}/modules/bash"$'\n'"$(pwd -P)/home/lib/bash"$'\n'"$(pwd -P)/abs"$'\n'"$(pwd -P)/bash" ]
}

@test "use sources the module from every directory on the path, in order, so a later file overrides" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > .kit/settings.yml
  printf 'yaml:keys() { echo overridden; }\nyaml:extra() { echo extra; }\n' > bash/yaml.sh
  printf 'answer: 42\n' > in.yml
  run kit-bash 'use yaml && yaml:value in.yml answer && yaml:keys in.yml && yaml:extra'
  [ "$status" -eq 0 ]
  [ "$output" = $'42\noverridden\nextra' ]
}

@test "use loads a module only the repository has, and only once" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - %s/bash\n' "${repo}" > .kit/settings.yml
  printf 'count=$(( ${count:-0} + 1 ))\n' > bash/local.sh
  run kit-bash 'use local && use local yaml && echo "${count}"'
  [ "$status" -eq 0 ]
  [ "$output" = 1 ]
}

@test "use refuses a module no directory on the path has, and names the path" {
  run kit-bash 'use nonesuch'
  [ "$status" -eq 2 ]
  [[ "$output" == *"no such kit bash module: nonesuch (looked in ${kit_root}/modules/bash)"* ]]
}

@test "a bash_path that does not start with kit:modules/bash/ is refused" {
  printf 'bash_path:\n  - bash/\n  - kit:modules/bash/\n' > .kit/settings.yml
  run kit-bash 'use yaml'
  [ "$status" -eq 2 ]
  [[ "$output" == *"bash_path must start with kit:modules/bash/"* ]]
}

@test "a bash_path entry with a root other than kit: is refused by name" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - other:bash/\n' > .kit/settings.yml
  run kit-bash 'use yaml'
  [ "$status" -eq 2 ]
  [[ "$output" == *"bash_path entry 'other:bash/' names a root the kit does not know"* ]]
}

@test "use:modules lists the modules on the whole path that match a pattern, once each" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > .kit/settings.yml
  : > bash/forge-local.sh
  : > bash/forge-github.sh
  run kit-bash 'use:modules "forge-*"'
  [ "$status" -eq 0 ]
  [ "$output" = $'forge-github\nforge-gitlab\nforge-local' ]
}

@test "a verb runs when called directly, through a symlink, from a subdirectory" {
  mkdir -p sub links
  ln -s "${kit_root}/bin/kit-version" links/kit-version
  cd sub
  run env -u kit_root "${repo}/links/kit-version"
  [ "$status" -eq 0 ]
  [ "$output" = unreleased ]
}

@test "sourcing the loader again is a no-op: it keeps the loaded modules and does not fail under errexit" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > .kit/settings.yml
  printf 'count=$(( ${count:-0} + 1 ))\n' > bash/local.sh
  run bash -c 'set -o errexit; source "${kit_root}/modules/bash/1st.sh"; use local
    source "${kit_root}/modules/bash/1st.sh"; use local; echo "${count}"'
  [ "$status" -eq 0 ]
  [ "$output" = 1 ]
}

@test "kit root prints the release, so a script outside the kit can source its loader, and BASH_ENV can name it" {
  run "${kit_root}/bin/kit" root
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}" ]
  run env -u kit_root bash -c 'source "$("${0}" root)/modules/bash/1st.sh" && use yaml && use:path' "${kit_root}/bin/kit"
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}/modules/bash" ]
  run env -u kit_root BASH_ENV="${kit_root}/modules/bash/1st.sh" "${kit_root}/bin/kit" version
  [ "$status" -eq 0 ]
  [ "$output" = unreleased ]
}

@test "run kit runs this release's kit, not the one on PATH, one level deeper" {
  mkdir shims
  printf '#!/usr/bin/env bash\necho impostor\n' > shims/kit && chmod +x shims/kit
  run env PATH="${repo}/shims:${PATH}" bash -c 'source "${kit_root}/modules/bash/1st.sh" && run kit version'
  [ "$status" -eq 0 ]
  [ "$output" = unreleased ]
  run kit-bash 'kit_depth=2 run bash -c "echo \${kit_depth}"'
  [ "$output" = 3 ]
}

@test "run refuses a kit-<verb> this release does not have" {
  run kit-bash 'run kit-nonesuch'
  [ "$status" -eq 2 ]
  [[ "$output" == *"no such kit command: kit-nonesuch"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
