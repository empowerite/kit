#!/usr/bin/env bats
# workspaces.bats - kit workspaces: members and checkouts, the file, the startup rule, and editing the file
# ----------------------------------------------------------------------------------------------------------------------
# Each test runs with HOME, XDG_CONFIG_HOME, and XDG_CACHE_HOME of its own, as helper:isolate sets them, so the
# workspace file and the last group used start missing. A clone is a git init with an origin added; nothing fetches.

bats_require_minimum_version 1.5.0

load helper

setup() {
  hide-system-toolchains
  kit_bash_modules=workspaces
}

# clone-at <path> <url> makes a repository at <path> whose origin is <url>; nothing is fetched.
clone-at() {
  git init --quiet "$1"
  git -C "$1" remote add origin "$2"
}

# workspaces-file writes what it reads to the workspace file.
workspaces-file() {
  mkdir -p "${XDG_CONFIG_HOME}/kit"
  cat > "${XDG_CONFIG_HOME}/kit/workspaces.yml"
}

@test "git:desks stops at a desk, skips hidden directories and links, and goes four levels down" {
  root="${BATS_TEST_TMPDIR}/code"
  mkdir -p "${root}/a/.git" "${root}/a/inner/.git" "${root}/.hidden/h/.git"
  mkdir -p "${root}/1/2/3/4/.git" "${root}/1/2/3/4/5/.git"
  mkdir -p "${root}/x/y/z/w/v/.git"
  ln -s "${root}/a" "${root}/link"
  run kit-bash "git:desks '${root}' 4"
  [ "$status" -eq 0 ]
  [ "$output" = "${root}/1/2/3/4
${root}/a" ]
}

@test "a remote's URL gives its member, every segment of its path kept" {
  run kit-bash 'for u in git@github.com:empowerite/edge.git https://github.com/empowerite/kit \
      ssh://git@gitlab.com/g/sub/repo.git /srv/repo.git file:///srv/r https://host/one
    do printf "[%s]\n" "$(workspaces:url-member "$u")"; done'
  [ "$output" = "[github.com/empowerite/edge]
[github.com/empowerite/kit]
[gitlab.com/g/sub/repo]
[]
[]
[]" ]
}

@test "a checkout's member is its primary remote's: kit.yml's primary, else origin" {
  clone-at "${BATS_TEST_TMPDIR}/fork" git@github.com:me/edge.git
  git -C "${BATS_TEST_TMPDIR}/fork" remote add upstream git@github.com:empowerite/edge.git
  run kit-bash "workspaces:member-of '${BATS_TEST_TMPDIR}/fork'"
  [ "$output" = "github.com/me/edge" ]
  printf 'primary: upstream\n' > "${BATS_TEST_TMPDIR}/fork/kit.yml"
  run kit-bash "workspaces:member-of '${BATS_TEST_TMPDIR}/fork'"
  [ "$output" = "github.com/empowerite/edge" ]
}

@test "the checkouts are every desk under the roots with a member, the earliest root first" {
  clone-at "${BATS_TEST_TMPDIR}/b/edge" git@github.com:empowerite/edge.git
  clone-at "${BATS_TEST_TMPDIR}/a/github.com/empowerite/kit" git@github.com:empowerite/kit.git
  clone-at "${BATS_TEST_TMPDIR}/a/kit-again" https://github.com/empowerite/kit.git
  git init --quiet "${BATS_TEST_TMPDIR}/a/no-remote"
  clone-at "${BATS_TEST_TMPDIR}/a/local" /srv/repo.git
  t=${BATS_TEST_TMPDIR}
  printf 'roots: [%s/b, %s/a, %s/a]\n' "$t" "$t" "$t" | workspaces-file
  run --separate-stderr kit-bash workspaces:checkouts
  [ "$status" -eq 0 ]
  [ -z "$stderr" ]
  [ "$output" = "$(printf '%s\t%s\n' \
    github.com/empowerite/edge "${BATS_TEST_TMPDIR}/b/edge" \
    github.com/empowerite/kit "${BATS_TEST_TMPDIR}/a/github.com/empowerite/kit" \
    github.com/empowerite/kit "${BATS_TEST_TMPDIR}/a/kit-again")" ]
}

@test "a root given with ~ is under HOME" {
  printf 'roots: ["~/Code"]\n' | workspaces-file
  run kit-bash workspaces:roots
  [ "$output" = "${HOME}/Code" ]
}

# empowerite-file makes a clone of edge under the code root and writes a file of two groups, one favorite.
empowerite-file() {
  clone-at "${BATS_TEST_TMPDIR}/code/edge" git@github.com:empowerite/edge.git
  workspaces-file <<YML
roots: [${BATS_TEST_TMPDIR}/code]
favorites: [empowerite]
workspaces:
  empowerite:
    repos: [github.com/empowerite/kit, github.com/empowerite/edge, github.com/empowerite/attic]
    exclude: [github.com/empowerite/attic]
  solo:
    repos: [github.com/me/solo]
YML
}

@test "--json lists the groups, their member counts, and the favorites" {
  empowerite-file
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.workspaces[] | [.name, .members, .favorite]]' <<<"$output")" \
    = '[["empowerite",2,true],["solo",1,false]]' ]
}

@test "--show resolves a group's members to their checkouts, and a missing one to none" {
  empowerite-file
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show empowerite --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output .kind <<<"$output")" = group ]
  [ "$(jq --compact-output '.members[0] | [.repository, .checkout, .forge, .by]' <<<"$output")" \
    = '["github.com/empowerite/kit",null,"github","fixed"]' ]
  [ "$(jq --compact-output '.members[1] | [.repository, .checkout, .forge, .by]' <<<"$output")" \
    = '["github.com/empowerite/edge","'"${BATS_TEST_TMPDIR}"'/code/edge","github","fixed"]' ]
  [ "$(jq --compact-output '.members[1].remotes' <<<"$output")" \
    = '[{"name":"origin","url":"git@github.com:empowerite/edge.git","forge":"github"}]' ]
}

@test "a repos entry given as a URL is read as its member; a path is refused, naming it" {
  workspaces-file <<YML
workspaces:
  w:
    repos: [git@github.com:empowerite/edge.git]
YML
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show w --json
  [ "$(jq --raw-output '.members[0].repository' <<<"$output")" = github.com/empowerite/edge ]
  workspaces-file <<YML
workspaces:
  w:
    repos: [/srv/nowhere]
YML
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show w --json
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"workspaces.yml"*"/srv/nowhere"* ]]
}

@test "an unknown workspace is a usage error; a member is a workspace of one" {
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show nonesuch
  [ "$status" -eq 2 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show github.com/me/x --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.kind, .members[0].by, .members[0].checkout]' <<<"$output")" \
    = '["repository","given",null]' ]
}

@test "with no file, the list is empty and says where the file goes" {
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '[.workspaces, .favorites, .file]' <<<"$output")" \
    = '[[],[],"'"${XDG_CONFIG_HOME}"'/kit/workspaces.yml"]' ]
}

@test "inside a repository, the workspace is that repository, even in a linked worktree" {
  fresh-repo >/dev/null
  git remote add origin git@github.com:empowerite/edge.git
  git commit --quiet --allow-empty -m one
  git worktree add --quiet "${BATS_TEST_TMPDIR}/wt" -b wt
  cd "${BATS_TEST_TMPDIR}/wt"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .name, .why]' <<<"$output")" \
    = '["repository","github.com/empowerite/edge","repository"]' ]
}

@test "a repository with no remote is named by its path, with no member" {
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  # git names the checkout by its physical path, as /private/var for macOS's /var
  [ "$(jq --compact-output '[.kind, .name, .member]' <<<"$output")" \
    = '["repository","'"$(cd "${repo}" && pwd -P)"'",null]' ]
}

@test "outside a repository, the workspace is the last group used, or none" {
  workspaces-file <<YML
workspaces:
  empowerite:
    repos: [github.com/empowerite/kit]
YML
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .why]' <<<"$output")" = '["none","none"]' ]
  run "${kit_root}/bin/kit" workspaces --use empowerite
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .name, .why]' <<<"$output")" = '["group","empowerite","last"]' ]
}

@test "--workspace wins, none included, and --use refuses a repository" {
  fresh-repo >/dev/null
  git remote add origin git@github.com:empowerite/edge.git
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --workspace none --json
  [ "$(jq --compact-output '[.kind, .why]' <<<"$output")" = '["none","given"]' ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --use github.com/empowerite/edge
  [ "$status" -eq 2 ]
}

@test "a last group no longer in the file is forgotten" {
  mkdir -p "${XDG_CACHE_HOME}/kit/workspaces" && printf 'gone\n' > "${XDG_CACHE_HOME}/kit/workspaces/last"
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --raw-output .kind <<<"$output")" = none ]
}

@test "--new makes the file and its directory, and the group with its members" {
  fresh-repo >/dev/null
  git remote add origin git@github.com:empowerite/edge.git
  run "${kit_root}/bin/kit" workspaces --new empowerite . \
    https://github.com/empowerite/kit.git github.com/empowerite/kit
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show empowerite --json
  [ "$(jq --compact-output '[.members[].repository]' <<<"$output")" \
    = '["github.com/empowerite/edge","github.com/empowerite/kit"]' ]
  run "${kit_root}/bin/kit" workspaces --new empowerite
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" workspaces --new Bad/Name
  [ "$status" -eq 2 ]
}

@test "--add, --remove, --favorite, and --unfavorite keep the file's comments" {
  workspaces-file <<'YML'
# mine
roots: [~/Code] # where I clone
workspaces:
  w:
    # the core
    repos: [github.com/a/one]
YML
  fresh-repo >/dev/null
  "${kit_root}/bin/kit" workspaces --add w github.com/a/two github.com/a/one
  "${kit_root}/bin/kit" workspaces --remove w github.com/a/one
  "${kit_root}/bin/kit" workspaces --favorite w
  "${kit_root}/bin/kit" workspaces --favorite w
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$(jq --compact-output '[.favorites, .workspaces[0].members]' <<<"$output")" = '[["w"],1]' ]
  grep --quiet '^# mine$' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  grep --quiet '# where I clone' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  grep --quiet '# the core' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  "${kit_root}/bin/kit" workspaces --unfavorite w
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$(jq --compact-output .favorites <<<"$output")" = '[]' ]
}

@test "editing a group that is not there is a usage error" {
  fresh-repo >/dev/null
  run "${kit_root}/bin/kit" workspaces --add nope github.com/a/one
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" workspaces --favorite nope
  [ "$status" -eq 2 ]
}

@test "a relative path is no member: the file refuses one, and the verb reads a directory from where it runs" {
  workspaces-file <<YML
workspaces:
  w:
    repos: [./code/edge]
YML
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show w --json
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"./code/edge"* ]]
  rm "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  git remote add origin git@github.com:empowerite/edge.git
  clone-at "${BATS_TEST_TMPDIR}/jig" git@github.com:empowerite/jig.git
  mkdir -p src && cd src
  run "${kit_root}/bin/kit" workspaces --new w .. ../../jig
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show w --json
  [ "$(jq --compact-output '[.members[].repository]' <<<"$output")" \
    = '["github.com/empowerite/edge","github.com/empowerite/jig"]' ]
  run "${kit_root}/bin/kit" workspaces --new x ./nonexistent/a/b
  [ "$status" -eq 2 ]
}

@test "a file of the wrong shape is refused, naming what is wrong" {
  fresh-repo >/dev/null
  for doc in 'workspaces: {w: {repos: github.com/a/b}}' 'workspaces: [a, b]' 'favorites: foo' 'roots: ~/Code' \
    'workspaces: {w: [github.com/a/b]}'
  do
    printf '%s\n' "${doc}" | workspaces-file
    run --separate-stderr "${kit_root}/bin/kit" workspaces --json
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"workspaces.yml"* && "$stderr" != *"jq: error"* ]]
  done
}

@test "--option=value runs as --option value" {
  workspaces-file <<YML
workspaces:
  w:
    repos: [github.com/a/one]
YML
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show=w --json
  [ "$(jq --raw-output .kind <<<"$output")" = group ]
  run "${kit_root}/bin/kit" workspaces --new=n github.com/a/two
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show n --json
  [ "$(jq --raw-output '.members[0].repository' <<<"$output")" = github.com/a/two ]
}

@test "--use and --favorite name a workspace without walking the roots" {
  clone-at "${BATS_TEST_TMPDIR}/code/broken" git@github.com:a/broken.git
  printf 'primary: [\n' > "${BATS_TEST_TMPDIR}/code/broken/kit.yml"
  workspaces-file <<YML
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w:
    repos: [github.com/a/one]
YML
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  for args in "--use w" "--favorite w" "--unfavorite w" "--current --workspace w"
  do
    # shellcheck disable=SC2086  # each holds its words
    run --separate-stderr "${kit_root}/bin/kit" workspaces ${args}
    [ "$status" -eq 0 ]
    [[ "$stderr" != *"broken"* ]]
  done
}

@test "--current --json gives a group's members with their checkouts, and the favorites" {
  clone-at "${BATS_TEST_TMPDIR}/code/a" git@github.com:o/a.git
  workspaces-file <<YML
roots: [${BATS_TEST_TMPDIR}/code]
favorites: [w, github.com/o/a]
workspaces:
  w:
    repos: [github.com/o/a, github.com/o/b]
YML
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  "${kit_root}/bin/kit" workspaces --use w
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$status" -eq 0 ]
  [ "$(jq --raw-output .kind <<<"$output")" = group ]
  [ "$(jq --compact-output '[.members[] | [.repository, .checkout]]' <<<"$output")" \
    = '[["github.com/o/a","'"${BATS_TEST_TMPDIR}"'/code/a"],["github.com/o/b",null]]' ]
  [ "$(jq --compact-output .groups <<<"$output")" = '[]' ]
  [ "$(jq --compact-output .favorites <<<"$output")" = '[{"name":"w","kind":"group","members":2,"checkout":null},'\
'{"name":"github.com/o/a","kind":"repository","members":null,"checkout":"'"${BATS_TEST_TMPDIR}"'/code/a"}]' ]
}

@test "--current --json inside a member names the groups it belongs to" {
  clone-at "${BATS_TEST_TMPDIR}/code/a" git@github.com:o/a.git
  workspaces-file <<YML
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w:
    repos: [github.com/o/a, github.com/o/b]
  x:
    repos: [github.com/o/b]
  v:
    repos: [github.com/o/a]
YML
  cd "${BATS_TEST_TMPDIR}/code/a"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output .groups <<<"$output")" = '[{"name":"w","members":2},{"name":"v","members":1}]' ]
  [ "$(jq --compact-output '[.members, .favorites]' <<<"$output")" = '[[],[]]' ]
}

@test "a favorite naming a group no longer in the file is left out of --current" {
  printf 'favorites: [gone]\n' | workspaces-file
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output .favorites <<<"$output")" = '[]' ]
}

@test "--current as text gives none of the control's fields, and walks no roots" {
  clone-at "${BATS_TEST_TMPDIR}/code/broken" git@github.com:a/broken.git
  printf 'primary: [\n' > "${BATS_TEST_TMPDIR}/code/broken/kit.yml"
  workspaces-file <<YML
roots: [${BATS_TEST_TMPDIR}/code]
favorites: [github.com/a/one]
workspaces:
  w:
    repos: [github.com/a/one]
YML
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --workspace w --text
  [ "$status" -eq 0 ]
  [ "$output" = "📚 w (given)" ]
  [[ "$stderr" != *"broken"* ]]
}

@test "--current --json walks the roots once, however many repository favorites there are" {
  clone-at "${BATS_TEST_TMPDIR}/code/broken" git@github.com:a/broken.git
  printf 'primary: [\n' > "${BATS_TEST_TMPDIR}/code/broken/kit.yml"
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  printf 'roots: [%s/code]\nfavorites: [github.com/a/one]\n' "${BATS_TEST_TMPDIR}" | workspaces-file
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  once=$(grep -c . <<<"${stderr}")
  printf 'roots: [%s/code]\nfavorites: [github.com/a/one, github.com/a/two, github.com/a/three]\n' \
    "${BATS_TEST_TMPDIR}" | workspaces-file
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$status" -eq 0 ]
  (( once > 0 ))
  [ "$(grep -c . <<<"${stderr}")" -eq "${once}" ]
}
