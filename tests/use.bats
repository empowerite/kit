#!/usr/bin/env bats
# use.bats - the module path: every <module>.sh on it is sourced in order, and a verb runs with or without bin/kit
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo
  mkdir .kit bash
}

@test "with no bash_path setting, the path is the kit's own bash directory" {
  run kit-bash 'use:path'
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}/modules/bash" ]
}

@test "an entry is kit:, absolute, under ~/, or relative to the root; a missing one is skipped, as PATH skips it" {
  mkdir -p home/lib/bash abs
  printf 'bash_path:\n  - kit:modules/bash/   # the kit\x27s own\n  - nowhere/\n  - ~/lib/bash/\n  - %s/abs\n  - bash/\n' \
    "$(pwd -P)" > kit.yml
  run env HOME="$(pwd -P)/home" bash -c 'source "${kit_root}/modules/bash/1st.sh" && use:path'
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}/modules/bash"$'\n'"$(pwd -P)/home/lib/bash"$'\n'"$(pwd -P)/abs"$'\n'"$(pwd -P)/bash" ]
}

@test "use sources the module from every directory on the path, in order, so a later file overrides" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  printf 'yaml:keys() { echo overridden; }\nyaml:extra() { echo extra; }\n' > bash/yaml.sh
  printf 'answer: 42\n' > in.yml
  run kit-bash 'use yaml && yaml:value in.yml answer && yaml:keys in.yml && yaml:extra'
  [ "$status" -eq 0 ]
  [ "$output" = $'42\noverridden\nextra' ]
}

@test "use loads a module only the repository has, and only once" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - %s/bash\n' "${repo}" > kit.yml
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
  printf 'bash_path:\n  - bash/\n  - kit:modules/bash/\n' > kit.yml
  run kit-bash 'use yaml'
  [ "$status" -eq 2 ]
  [[ "$output" == *"bash_path must start with kit:modules/bash/"* ]]
}

@test "a bash_path entry with a root other than kit: is refused by name" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - other:bash/\n' > kit.yml
  run kit-bash 'use yaml'
  [ "$status" -eq 2 ]
  [[ "$output" == *"bash_path entry 'other:bash/' names a root the kit does not know"* ]]
}

@test "a kit.yml yq cannot read refuses use with 2, after yq's line, and the kit's modules do not stand in" {
  printf 'bash_path: [kit:modules/bash/\n' > kit.yml
  run kit-bash 'use yaml && echo loaded'
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit.yml: "* && "$output" != *loaded* ]]
}

@test "use:modules lists the modules on the whole path that match a pattern, once each, named from their directory" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  mkdir -p bash/forge
  : > bash/forge/local.sh
  : > bash/forge/github.sh
  run kit-bash 'use:modules "forge/*"'
  [ "$status" -eq 0 ]
  [ "$output" = $'forge/github\nforge/gitlab\nforge/local' ]
}

@test "a module in a subdirectory loads by its path, every copy on the path in order, and names its functions so" {
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  mkdir -p bash/toolchain
  printf 'toolchain/local:hello() { printf "hello %%s\\n" "$1"; }\n' > bash/toolchain/local.sh
  printf 'toolchain/brew:file() { printf "ours\\n"; }\n' > bash/toolchain/brew.sh
  run kit-bash 'use toolchain/local && toolchain/local:hello there'
  [ "$status" -eq 0 ]
  [ "$output" = "hello there" ]
  run kit-bash 'use toolchain/brew && toolchain/brew:file'
  [ "$output" = "ours" ]
  run kit-bash 'use toolchain && toolchain:providers'
  [ "$output" = "apt brew devcontainer dnf docker local mise nix pacman" ]
}

# child <script> runs the script in a child bash process that has loaded the loader, as a kit verb's child would.
child() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && eval "$1"' child "$1"
}

@test "a child process uses the module path its parent read, and reads kit.yml again once it changes" {
  export -f child
  printf 'bash_path:\n  - kit:modules/bash/\n' > kit.yml
  printf 'local:hello() { echo hello from bash/; }\n' > bash/local.sh
  run kit-bash 'use:path >/dev/null
    [[ "${kit_bash_path_dirs}" == "${kit_root}/modules/bash/"$'"'"'\n'"'"' ]] || exit 3
    child "use:path"
    printf "bash_path:\n  - kit:modules/bash/\n  - bash/\n" > kit.yml
    child "use local && local:hello"'
  [ "$status" -eq 0 ]
  [ "$output" = "${kit_root}/modules/bash"$'\n'"hello from bash/" ]
}

@test "a child process starts no yq when its parent has seen yq and read bash_path, and a parent starts yq --version once" {
  export -f child
  # a yq that logs each call, then steps out of PATH and hands it to the yq behind it
  mkdir shims
  printf '#!/usr/bin/env bash\necho "$*" >> "%s/yq.log"\nPATH=${PATH#*:} exec yq "$@"\n' "${repo}" > shims/yq
  chmod +x shims/yq
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
  run env PATH="${repo}/shims:${PATH}" kit_yq='' kit_bash_path_dirs='' bash -c '
    source "${kit_root}/modules/bash/1st.sh" && use yaml && echo parent
    : > yq.log
    child "use yaml && use:path >/dev/null && echo child"'
  [ "$status" -eq 0 ]
  [ "$output" = $'parent\nchild' ]
  [ ! -s yq.log ]
  : > yq.log
  run env PATH="${repo}/shims:${PATH}" kit_yq='' kit_bash_path_dirs='' bash -c '
    source "${kit_root}/modules/bash/1st.sh" && use yaml && use:path >/dev/null'
  [ "$status" -eq 0 ]
  [ "$(grep -c -- --version yq.log)" -eq 1 ]
}

@test "use refuses a name that could reach outside the module path" {
  local name
  for name in /etc/passwd ../yaml forge/../yaml forge//github ./yaml forge/
  do
    run kit-bash "use '${name}'"
    [ "$status" -eq 2 ]
    [[ "$output" == *"no such kit bash module: ${name} (a module's name has no empty, ., or .. part"* ]] \
      || { echo "not refused: ${name}"; return 1; }
  done
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
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' > kit.yml
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

@test "list:read fills an array with a command's lines and returns its status, the array empty when it fails" {
  run kit-bash 'bad() { printf "a\n"; return 3; }; nul() { printf "x y\0z\0"; return "$1"; }
    list:read rows printf "a\n\nb c\n"; echo "$? ${#rows[@]} [${rows[2]}]"
    list:read rows bad; echo "$? ${#rows[@]}"
    list:read0 items nul 0; echo "$? ${#items[@]} [${items[0]}]"
    list:read0 items nul 4; echo "$? ${#items[@]}"'
  [ "$output" = $'0 3 [b c]\n3 0\n0 2 [x y]\n4 0' ]
}

@test "list:read runs a command as \$(…) runs it: a loop at a pipeline's end is in a subshell of its own" {
  run kit-bash 'first() { printf "a\nb\n" | while read -r line; do printf "%s\n" "${line}"; return 0; done
      printf "after\n"; }
    list:read rows first; echo "$? ${rows[*]}"
    echo "0 $(first | paste -s -d " " -)"'
  [ "${lines[0]}" = "0 a after" ]
  [ "${lines[1]}" = "${lines[0]}" ]
}

@test "with job control on, list:read fills the array, and returns the command's status with the array empty" {
  run kit-bash 'set -m; bad() { printf "a\n"; return 3; }
    list:read rows printf "a\nb\n"; echo "$? ${#rows[@]} [${rows[1]}]"
    list:read rows bad; echo "$? ${#rows[@]}"'
  [ "$output" = $'0 2 [b]\n3 0' ]
}

@test "a kit.yml yq cannot read fails packages and excludes, said once; absent or with no version, it declares none" {
  local both='use settings; list:read p settings:packages; echo "$? ${#p[@]}"'
  both+='; list:read e settings:excludes; echo "$? ${#e[@]}"'
  printf 'packages: [kit\n' > kit.yml
  run --separate-stderr kit-bash 'use settings; list:read p settings:packages; echo "$? ${#p[@]}"'
  [ "$output" = "1 0" ]
  [ "$(grep -c '❌' <<<"$stderr")" -eq 1 ]
  run --separate-stderr kit-bash 'use settings; list:read e settings:excludes; echo "$? ${#e[@]}"'
  [ "$output" = "1 0" ]
  [ "$(grep -c '❌' <<<"$stderr")" -eq 1 ]
  run --separate-stderr kit-bash 'use settings; settings:require; echo "$?"'
  [ "$output" = 1 ]
  [ "$(grep -c '❌' <<<"$stderr")" -eq 1 ]
  [[ "$stderr" != *"run: kit install"* ]]
  printf 'forge: github\n' > kit.yml
  run kit-bash "${both}"
  [ "$output" = $'0 0\n0 0' ]
  rm kit.yml
  run kit-bash "${both}"
  [ "$output" = $'0 0\n0 0' ]
}

# ----------------------------------------------------------------------------------------------------------------------
