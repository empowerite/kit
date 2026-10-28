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
  bash -c 'source "${kit_root}/bash/1st.sh" && eval "$1"' kit-bash "$1"
}

@test "with no bash_path setting, the path is the kit's own bash directory" {
  run kit-bash 'use:path && printf "%s\n" "${kit_bash_path[@]}"'
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}/bash" ]
}

@test "use sources the module from every directory on the path, in order, so a later file overrides" {
  printf 'bash_path:\n  - kit:bash/\n  - bash/\n' > .kit/settings.yml
  printf 'yaml:keys() { echo overridden; }\nyaml:extra() { echo extra; }\n' > bash/yaml.sh
  printf 'answer: 42\n' > in.yml
  run kit-bash 'use yaml && yaml:value in.yml answer && yaml:keys in.yml && yaml:extra'
  [ "$status" -eq 0 ]
  [ "$output" = $'42\noverridden\nextra' ]
}

@test "use loads a module only the repository has, and only once" {
  printf 'bash_path:\n  - kit:bash/\n  - %s/bash\n' "${repo}" > .kit/settings.yml
  printf 'count=$(( ${count:-0} + 1 ))\n' > bash/local.sh
  run kit-bash 'use local && use local yaml && echo "${count}"'
  [ "$status" -eq 0 ]
  [ "$output" = 1 ]
}

@test "use refuses a module no directory on the path has, and names the path" {
  run kit-bash 'use nonesuch'
  [ "$status" -eq 2 ]
  [[ "$output" == *"no such kit bash module: nonesuch (looked in ${kit_root}/bash)"* ]]
}

@test "a bash_path that does not start with kit:bash/ is refused" {
  printf 'bash_path:\n  - bash/\n  - kit:bash/\n' > .kit/settings.yml
  run kit-bash 'use yaml'
  [ "$status" -eq 2 ]
  [[ "$output" == *"bash_path must start with kit:bash/"* ]]
}

@test "a bash_path entry that is not a directory, or has a root other than kit:, is refused by name" {
  printf 'bash_path:\n  - kit:bash/\n  - nowhere/\n' > .kit/settings.yml
  run kit-bash 'use yaml'
  [ "$status" -eq 2 ]
  [[ "$output" == *"bash_path entry 'nowhere/' is not a directory"* ]]
  printf 'bash_path:\n  - kit:bash/\n  - other:bash/\n' > .kit/settings.yml
  run kit-bash 'use yaml'
  [ "$status" -eq 2 ]
  [[ "$output" == *"bash_path entry 'other:bash/' names a root the kit does not know"* ]]
}

@test "use:modules lists the modules on the whole path that match a pattern, once each" {
  printf 'bash_path:\n  - kit:bash/\n  - bash/\n' > .kit/settings.yml
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

@test "run kit runs this release's kit, not the one on PATH, one level deeper" {
  mkdir shims
  printf '#!/usr/bin/env bash\necho impostor\n' > shims/kit && chmod +x shims/kit
  run env PATH="${repo}/shims:${PATH}" bash -c 'source "${kit_root}/bash/1st.sh" && run kit version'
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
