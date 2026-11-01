#!/usr/bin/env bats
# file.bats - file:replace and file:copy write a file whole or not at all: a failed producer leaves the file as it was
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

@test "no writer in the modules or the verbs moves a .kit-new over a path on its own" {
  # Every write beside-and-move goes through file.sh, so a producer's failure never leaves an empty file behind.
  run grep -l '\.kit-new' "${kit_root}"/bin/kit-* "${kit_root}"/modules/bash/*.sh "${kit_root}"/modules/bash/*/*.sh
  [ "$output" = "${kit_root}/modules/bash/file.sh" ]
}
