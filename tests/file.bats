#!/usr/bin/env bats
# file.bats - file:replace and file:copy write a file whole or not at all, and one EXIT trap sweeps the scratch space
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  kit_bash_modules=file
  fresh-repo
}

@test "file:replace writes the producer's output over the path, whole, keeping its mode, and makes the directory it needs" {
  printf 'old\n' > a.txt && chmod 700 a.txt
  run kit-bash 'file:replace a.txt printf "new\n" && file:replace deep/down/b.txt printf "made\n"'
  [ "$status" -eq 0 ]
  [ "$(cat a.txt)" = "new" ]
  [ "$(ls -l a.txt | cut -c 1-10)" = "-rwx------" ]
  [ "$(cat deep/down/b.txt)" = "made" ]
  [ ! -e a.txt.kit-new ]
}

@test "file:replace leaves the path as it was, removes the temporary, and returns the producer's status when it fails" {
  printf 'kept\n' > a.txt
  run kit-bash 'file:replace a.txt bash -c "printf partial; exit 3"'
  [ "$status" -eq 3 ]
  [ "$(cat a.txt)" = "kept" ]
  [ ! -e a.txt.kit-new ]
  run kit-bash 'file:replace never.txt false'
  [ "$status" -eq 1 ]
  [ ! -e never.txt ]
  [ ! -e never.txt.kit-new ]
}

@test "file:replace --read-only takes the write bit off, and replaces a read-only file in place" {
  run kit-bash 'file:replace --read-only r.txt printf "one\n" && file:replace --read-only r.txt printf "two\n"'
  [ "$status" -eq 0 ]
  [ "$(cat r.txt)" = "two" ]
  [ ! -w r.txt ]
  # a read-only temporary an interrupted run left behind does not refuse the next write
  printf 'stale\n' > r.txt.kit-new && chmod a-w r.txt.kit-new
  run kit-bash 'file:replace --read-only r.txt printf "three\n"'
  [ "$status" -eq 0 ]
  [ "$(cat r.txt)" = "three" ]
}

@test "file:copy copies the source over the path, keeping the source's mode, or without the write bit" {
  printf '#!/usr/bin/env bash\n' > src.sh && chmod 755 src.sh
  run kit-bash 'file:copy src.sh bin/copy.sh && file:copy --read-only src.sh .kit/copy.sh'
  [ "$status" -eq 0 ]
  [ -x bin/copy.sh ]
  [ "$(cat bin/copy.sh)" = "#!/usr/bin/env bash" ]
  [ ! -w .kit/copy.sh ]
  run kit-bash 'file:copy nonesuch.sh bin/copy.sh'
  [ "$status" -ne 0 ]
  [ "$(cat bin/copy.sh)" = "#!/usr/bin/env bash" ]
  [ ! -e bin/copy.sh.kit-new ]
}

@test "file:scratch makes directories under TMPDIR that the one EXIT trap removes, and file:at-exit's command runs" {
  run kit-bash 'file:scratch a && file:scratch b && file:at-exit touch "${PWD}/ran" && trap -p EXIT && printf "%s\n" "${a}" "${b}"'
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "trap -- 'file:sweep' EXIT" ]
  [[ "${lines[1]}" == "${TMPDIR}/kit."* && "${lines[2]}" == "${TMPDIR}/kit."* && "${lines[1]}" != "${lines[2]}" ]]
  [ ! -e "${lines[1]}" ]
  [ ! -e "${lines[2]}" ]
  [ -f ran ]
  # the directories are there while the process runs, and a command's words are expanded when it is given
  run kit-bash 'file:scratch a && file:at-exit cp "${a}/in" "${PWD}/out" && printf "kept\n" > "${a}/in" && [[ -d "${a}" ]]'
  [ "$status" -eq 0 ]
  [ "$(cat out)" = kept ]
}

@test "the trap survives a module that asks for scratch space after another did, and the exit status survives the sweep" {
  run kit-bash 'use render && file:scratch a && render:tmp && trap -p EXIT && printf "%s\n" "${a}" "${render_tmp}" && exit 3'
  [ "$status" -eq 3 ]
  [ "${lines[0]}" = "trap -- 'file:sweep' EXIT" ]
  [[ "${lines[1]}" == "${TMPDIR}/kit."* && "${lines[2]}" == "${TMPDIR}/kit."* ]]
  [ ! -e "${lines[1]}" ]
  [ ! -e "${lines[2]}" ]
  # a command that fails at exit changes nothing: the sweep goes on, and the status stays
  run kit-bash 'file:at-exit false && file:scratch a && file:at-exit touch "${PWD}/after" && printf "%s\n" "${a}"'
  [ "$status" -eq 0 ]
  [ ! -e "${lines[0]}" ]
  [ -f after ]
}

@test "no writer in the modules or the verbs moves a .kit-new over a path on its own" {
  # Every write beside-and-move goes through file.sh, so a producer's failure never leaves an empty file behind.
  run grep -l '\.kit-new' "${kit_root}"/bin/kit-* "${kit_root}"/modules/bash/*.sh "${kit_root}"/modules/bash/*/*.sh
  [ "$output" = "${kit_root}/modules/bash/file.sh" ]
}

@test "a scratch made in a subshell is swept when the subshell ends, and the parent's stays until the parent exits" {
  run kit-bash 'file:scratch p && ( file:scratch s && printf "%s\n" "${s}" > sub && [[ -d "${s}" && -d "${p}" ]] ) &&
    [[ ! -e "$(cat sub)" && -d "${p}" ]] && printf "%s\n" "${p}"'
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "${TMPDIR}/kit."* && "$(cat sub)" == "${TMPDIR}/kit."* && "${lines[0]}" != "$(cat sub)" ]]
  [ ! -e "${lines[0]}" ]
  [ ! -e "$(cat sub)" ]
}

@test "an EXIT trap a verb set before the first scratch still runs at the sweep, and file:sweep is the one trap" {
  run kit-bash 'trap "touch ${PWD}/own" EXIT; file:scratch a && trap -p EXIT && printf "%s\n" "${a}"'
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "trap -- 'file:sweep' EXIT" ]
  [ ! -e "${lines[1]}" ]
  [ -f own ]
}

@test "a command at exit that exits stops neither the sweep nor the others, and leaves the exit status as it was" {
  run kit-bash 'file:at-exit exit 9; file:scratch a && file:at-exit touch "${PWD}/after" && printf "%s\n" "${a}"'
  [ "$status" -eq 0 ]
  [ ! -e "${lines[0]}" ]
  [ -f after ]
  run kit-bash 'file:at-exit exit 9; file:scratch a; exit 3'
  [ "$status" -eq 3 ]
}

@test "writes to one path at once each land whole, and none fails another" {
  local script
  script=$(cat <<'EOF'
pids=()
for i in 1 2 3 4 5 6 7 8
do
  file:replace same.txt bash -c 'sleep 0.2; printf "%s\n" "$1"' writer "${i}" &
  pids+=("$!")
  file:copy src.txt copied.txt &
  pids+=("$!")
done
failed=0
for pid in "${pids[@]}"
do
  wait "${pid}" || failed=$(( failed + 1 ))
done
exit "${failed}"
EOF
)
  printf 'source\n' > src.txt
  run kit-bash "${script}"
  [ "$status" -eq 0 ]
  [[ "$(cat same.txt)" =~ ^[1-8]$ ]]
  [ "$(cat copied.txt)" = source ]
  [ -z "$(find . -name '*.kit-new*')" ]
}

@test "a write stopped midway leaves the path as it was and no partial file" {
  printf 'old\n' > x.txt
  run kit-bash 'file:replace x.txt bash -c "printf part; kill -TERM \$PPID; sleep 1"'
  [ "$(cat x.txt)" = old ]
  [ -z "$(find . -name 'x.txt.kit-new*')" ]
}
@test "every scratch file the kit makes is under TMPDIR, through file:scratch" {
  # A bare mktemp on macOS ignores TMPDIR, so what it makes escapes the sweep, and the tests' isolation.
  run grep -l 'mktemp' "${kit_root}"/bin/kit* "${kit_root}"/modules/bash/*.sh "${kit_root}"/modules/bash/*/*.sh
  [ "$output" = "${kit_root}/modules/bash/file.sh" ]
}
