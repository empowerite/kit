#!/usr/bin/env bats
# modules.bats - every function says where it is defined, and a release is split by concern into modules of its own
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  fresh-repo
}

@test "every function in a module is named <module>:<name>, so that each says where it is defined" {
  local file module name stray=''
  while IFS= read -r file
  do
    module=${file#"${kit_root}/modules/bash/"}
    module=${module%.sh}
    [[ "${module}" != 1st ]] || continue
    while IFS= read -r name
    do
      # Two names are a convention outside the kit: super, which an override calls by that name, and _kit, which
      # the shell's completion registers.
      [[ "${name}" == super || "${name}" == _kit ]] && continue
      [[ "${name}" == "${module}:"* ]] || stray+="${module}: ${name}"$'\n'
    done < <(sed -n 's/^\([A-Za-z0-9_:\/-]*\)() {$/\1/p' "${file}")
  done < <(find "${kit_root}/modules/bash" -name '*.sh' | LC_ALL=C sort)
  [ -z "${stray}" ] || { printf '%s' "${stray}"; return 1; }
}

@test "the release is read, rendered, placed, and recorded by one module each, and use release loads none of them but resolve" {
  local module function
  while IFS=' ' read -r module function
  do
    [ -f "${kit_root}/modules/bash/${module}.sh" ] || { echo "no module ${module}"; return 1; }
    run kit-bash "use ${module} && declare -F ${module}:${function}"
    [ "$status" -eq 0 ] || { echo "no ${module}:${function}"; return 1; }
  done <<'LIST'
release fetch
release bases
settings declare
settings validate
resolve active
resolve reach
render all
region write
place content
place retire
materialize all
materialize hydrate
manifest write-all
manifest recorded
LIST
  # release:finish reports with the packages' metadata, so use release brings resolve; nothing that writes comes with it
  run kit-bash 'use release && declare -F | sed "s/^declare -f //" | grep -E "^(settings|render|region|place|materialize|manifest):"'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  run kit-bash 'use file && printf "abc" | file:digest && printf "abc" > f && file:digest f'
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" ]
  [ "${lines[1]}" = "${lines[0]}" ]
  run kit-bash 'list:among b a b c && ! list:among z a b c && list:without b a b c | paste -s -d " " -'
  [ "$status" -eq 0 ]
  [ "$output" = "a c" ]
}
