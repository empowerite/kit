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

# A test that shuts the cache directory leaves it shut when it fails, and bats removes its directory after: it is
# opened again first.
teardown() {
  [[ ! -d "${XDG_CACHE_HOME:-}" ]] || chmod -R u+w "${XDG_CACHE_HOME}"
  helper:outside-checkout
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

@test "a workspace given before the verb names the one --current says, and every other form refuses it with 2" {
  run "${kit_root}/bin/kit" --workspace=w workspaces --use w
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit --workspace=<workspace> names the workspace kit workspaces --current says, and no other form's"* ]]
  run "${kit_root}/bin/kit" --workspace=none workspaces
  [ "$status" -eq 2 ]
}

@test "--workspace= wins, none included, and --use refuses a repository" {
  fresh-repo >/dev/null
  git remote add origin git@github.com:empowerite/edge.git
  run --separate-stderr "${kit_root}/bin/kit" --workspace=none workspaces --current --json
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

@test "twelve --add at once all land; a dead holder's lock is taken over, and a live one's refuses with 1" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/m0]}}'
  for n in 1 2 3 4 5 6 7 8 9 10 11 12; do "${kit_root}/bin/kit" workspaces --add=w "github.com/a/m${n}" & done; wait
  lock="${XDG_CACHE_HOME}/kit/workspaces/workspaces.yml.lock"
  true & gone=$!; wait "${gone}"; ln -s "${gone}" "${lock}"   # ln fails when an --add left its lock behind
  kit-bash 'workspaces:add w github.com/a/m13'
  sleep 30 3>&- & holder=$!; ln -s "${holder}" "${lock}"
  run kit-bash 'workspaces_lock_wait=2; workspaces:add w github.com/a/m14'
  kill "${holder}"
  [[ "$status" -eq 1 && "$output" == *"workspaces.yml is being edited by process ${holder}"* ]]
  [ "$(kit-bash 'workspaces:fixed w' | grep -c .)" -eq 14 ]
}

@test "the lock's wait is the clock's, however long each turn of it takes" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/one]}}'
  mkdir -p "${XDG_CACHE_HOME}/kit/workspaces"
  sleep 30 3>&- & holder=$!; ln -s "${holder}" "${XDG_CACHE_HOME}/kit/workspaces/workspaces.yml.lock"
  # each turn's sleep takes a second, as a turn does under load, so twenty turns would take twenty seconds
  SECONDS=0
  run kit-bash 'sleep() { command sleep 1; }; workspaces_lock_wait=20; workspaces:add w github.com/a/two'
  took=${SECONDS}
  kill "${holder}"
  [[ "$status" -eq 1 && "$output" == *"workspaces.yml is being edited by process ${holder}"* ]]
  (( took < 10 ))
  [ "$(kit-bash 'workspaces:fixed w')" = github.com/a/one ]
}

@test "--use waits for the file's lock, as an edit does" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/one]}}'
  mkdir -p "${BATS_TEST_TMPDIR}/outside" "${XDG_CACHE_HOME}/kit/workspaces" && cd "${BATS_TEST_TMPDIR}/outside"
  last="${XDG_CACHE_HOME}/kit/workspaces/last"
  sleep 30 3>&- & holder=$!; ln -s "${holder}" "${XDG_CACHE_HOME}/kit/workspaces/workspaces.yml.lock"
  "${kit_root}/bin/kit" workspaces --use=w >/dev/null 3>&- & use=$!
  sleep 3
  written=no
  [[ ! -e "${last}" ]] || written=yes
  kill "${holder}"
  wait "${use}"
  [ "${written}" = no ]
  [ "$(<"${last}")" = w ]
}

@test "a cache directory kit cannot write stops --use and an edit at once, with one line that names it" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/one]}}'
  cp "${XDG_CONFIG_HOME}/kit/workspaces.yml" "${BATS_TEST_TMPDIR}/before.yml"
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  # kit's other caches under XDG_CACHE_HOME/kit are made before it is shut, so only kit/workspaces/ cannot be
  "${kit_root}/bin/kit" workspaces >/dev/null
  cache=${XDG_CACHE_HOME}/kit/workspaces
  # the directory cannot be made, and then it is there but cannot be written
  for shut in "${XDG_CACHE_HOME}/kit" "${cache}"
  do
    [[ "${shut}" != "${cache}" ]] || { chmod u+w "${XDG_CACHE_HOME}/kit" && mkdir "${cache}"; }
    chmod a-w "${shut}"
    SECONDS=0
    for args in "--use=w" "--add=w github.com/a/two"
    do
      # shellcheck disable=SC2086  # each holds its words
      run --separate-stderr "${kit_root}/bin/kit" workspaces ${args}
      [ "$status" -eq 1 ]
      [ "${#stderr_lines[@]}" -eq 1 ]
      [[ "${stderr_lines[0]}" == "❌ "*"${cache}"* ]]
    done
    run --separate-stderr kit-bash 'workspaces:remember w'
    [ "$status" -eq 1 ]
    [ "${#stderr_lines[@]}" -eq 1 ]
    [[ "${stderr_lines[0]}" == "❌ "*"${cache}"* ]]
    (( SECONDS < 10 ))
    [ ! -e "${cache}/last" ]
    [ ! -e "${cache}/workspaces.yml.lock" ]
  done
  chmod -R u+w "${XDG_CACHE_HOME}"
  cmp "${BATS_TEST_TMPDIR}/before.yml" "${XDG_CONFIG_HOME}/kit/workspaces.yml"
}

@test "--delete forgets the group as the last used, so a new group of its name is not the last used" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/one]}, v: {repos: [github.com/a/two]}}'
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  last="${XDG_CACHE_HOME}/kit/workspaces/last"
  "${kit_root}/bin/kit" workspaces --use=w
  "${kit_root}/bin/kit" workspaces --delete=v
  [ "$(<"${last}")" = w ]
  run "${kit_root}/bin/kit" workspaces --delete=w
  [ "$status" -eq 0 ]
  [ ! -e "${last}" ]
  "${kit_root}/bin/kit" workspaces --new=w github.com/a/one
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .why]' <<<"$output")" = '["none","none"]' ]
}

@test "a working tree is inside a repository; a bare repository and a .git directory are outside it" {
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/one]}}'
  mkdir -p "${XDG_CACHE_HOME}/kit/workspaces" && printf 'w\n' > "${XDG_CACHE_HOME}/kit/workspaces/last"
  git init --quiet --bare "${BATS_TEST_TMPDIR}/bare.git"
  fresh-repo >/dev/null
  run --separate-stderr kit-bash 'workspaces:in-repository'
  [ "$status" -eq 0 ]
  [ "$output" = "$(pwd -P)" ]
  run --separate-stderr kit-bash 'workspaces:across'
  [[ "$status" -eq 0 && -z "$output" && -z "$stderr" ]]
  for dir in "${BATS_TEST_TMPDIR}/bare.git" "${repo}/.git"
  do
    cd "${dir}"
    run --separate-stderr kit-bash 'workspaces:in-repository'
    [[ "$status" -eq 1 && -z "$output" && -z "$stderr" ]]
    run --separate-stderr kit-bash 'workspaces:across'
    [[ "$status" -eq 0 && "$output" = w && -z "$stderr" ]]
  done
}

@test "--rename and --delete keep the comment, the favorites, and the last group; a member is matched once" {
  clone-at "${BATS_TEST_TMPDIR}/code/one" https://GitHub.com/A/One.git
  printf 'roots: ["%s", "%s/"]\nfavorites: [w, gone]\nworkspaces:\n  # the core\n  w: {repos: [github.com/a/one]}\n' \
    "${BATS_TEST_TMPDIR}/code" "${BATS_TEST_TMPDIR}/code" | workspaces-file
  for words in "--use w" "--new=z github.com/a/two" "--rename=w core" "--unfavorite=gone" \
    "--add=core GitHub.com/A/One github.com/a/three"
  do
    "${kit_root}/bin/kit" workspaces ${words}
  done
  for words in "--rename=core z" "--rename=core none" "--rename=core Bad/Name" "--rename=no x" "--delete=no"
  do
    run "${kit_root}/bin/kit" workspaces ${words}
    [ "$status" -eq 2 ]
  done
  grep --quiet '# the core' "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq -c '[.name, .why, .favorites[].name, (.members[].checkouts | length)]' <<<"$output")" = \
    '["core","last","core",1,0]' ]
  "${kit_root}/bin/kit" workspaces --delete=core
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq -c '[.kind, .favorites]' <<<"$output")" = '["none",[]]' ]
}

@test "an edit says what it changed, or that nothing needed to, and leaves the file as it was then" {
  workspaces-file <<<'favorites: [w]
workspaces: {w: {repos: [github.com/a/one]}, v: {repos: [github.com/a/two]}}'
  cp "${XDG_CONFIG_HOME}/kit/workspaces.yml" "${BATS_TEST_TMPDIR}/before.yml"
  for each in "--add=w github.com/a/one|📚 w holds them already" "--remove=w github.com/a/two|📚 w holds none of them" \
    "--favorite=w|📚 w is a favorite already" "--unfavorite=v|📚 v is not a favorite"
  do
    run "${kit_root}/bin/kit" workspaces ${each%%|*}
    [ "$status" -eq 0 ]
    [ "$output" = "→ ${each#*|}" ]
  done
  cmp "${BATS_TEST_TMPDIR}/before.yml" "${XDG_CONFIG_HOME}/kit/workspaces.yml"
  run "${kit_root}/bin/kit" workspaces --unfavorite=w
  [ "$output" = "→ 📚 w is not a favorite" ]
  run "${kit_root}/bin/kit" workspaces --rename=w core
  [ "$output" = "→ 📚 w is renamed 📚 core" ]
  run "${kit_root}/bin/kit" workspaces --use=core
  [ "$status" -eq 0 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --show=core --json
  [ "$(jq -c '[.name, .members[].repository]' <<<"$output")" = '["core","github.com/a/one"]' ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq -c '[.name, .why, .favorites]' <<<"$output")" = '["core","last",[]]' ]
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

@test "a file that is no map, and a group's key kit does not read, are refused by name" {
  for each in "workspaces: {q: {match: [{owner: github.com/o}]}}|q's match is not built yet" \
    "workspaces: {p: {project: github.com/orgs/o/projects/3}}|p's project is not built yet" \
    "workspaces: {w: {repo: [github.com/o/a]}}|'repo' in w is no key of a group" \
    "just a string|the file is a map of roots, favorites, and workspaces"
  do
    workspaces-file <<<"${each%%|*}"
    run --separate-stderr "${kit_root}/bin/kit" workspaces --json
    [[ "$status" -eq 1 && -z "$output" && "$stderr" == *"workspaces.yml: ${each#*|}"* && "$stderr" != *jq:* ]]
  done
}

@test "a group named none is refused with 1, and the file is as it was" {
  workspaces-file <<<'workspaces: {none: {repos: [github.com/o/a]}}'
  cp "${XDG_CONFIG_HOME}/kit/workspaces.yml" "${BATS_TEST_TMPDIR}/before.yml"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --add none github.com/o/b
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"workspaces.yml: 'none' is not a workspace's name: --workspace=none names no workspace"* ]]
  cmp "${BATS_TEST_TMPDIR}/before.yml" "${XDG_CONFIG_HOME}/kit/workspaces.yml"
}

@test "a name that starts with . _ or - is refused by its first character, and no group is made" {
  for name in .dot _under
  do
    run --separate-stderr "${kit_root}/bin/kit" workspaces --new="${name}" github.com/a/one
    [ "$status" -eq 2 ]
    [[ "$stderr" == *"'${name}' is not a workspace's name: it starts with a lowercase letter or a digit"* ]]
    [[ "$stderr" == *"and holds lowercase letters, digits, and . _ - alone"* ]]
    [ "${#stderr_lines[@]}" -eq 1 ]
    [ ! -e "${XDG_CONFIG_HOME}/kit/workspaces.yml" ]
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
  for args in "workspaces --use w" "workspaces --favorite w" "workspaces --unfavorite w" \
    "--workspace=w workspaces --current"
  do
    # shellcheck disable=SC2086  # each holds its words
    run --separate-stderr "${kit_root}/bin/kit" ${args}
    [ "$status" -eq 0 ]
    [[ "$stderr" != *"broken"* ]]
  done
}

@test "a workspace file that does not parse stops --use, --favorite, and --current with 1; an unknown group is 2" {
  workspaces-file <<<'workspaces: [x'
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  for args in "workspaces --use w" "workspaces --favorite w" "workspaces --unfavorite w" \
    "--workspace=w workspaces --current"
  do
    # shellcheck disable=SC2086  # each holds its words
    run --separate-stderr "${kit_root}/bin/kit" ${args}
    [ "$status" -eq 1 ]
  done
  [ "$(cat "${XDG_CONFIG_HOME}/kit/workspaces.yml")" = 'workspaces: [x' ]
  workspaces-file <<<'workspaces: {w: {repos: [github.com/a/one]}}'
  run --separate-stderr "${kit_root}/bin/kit" workspaces --use nosuch
  [ "$status" -eq 2 ]
}

@test "the last group used, and the menu's favorites, fail when the file's groups or favorites cannot be read" {
  workspaces-file <<<'workspaces: foo'
  run --separate-stderr kit-bash 'workspaces:remember w && workspaces:last'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  workspaces-file <<<'favorites: foo'
  run --separate-stderr kit-bash 'workspaces:menu-json "{\"kind\": \"none\", \"member\": null}"'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  # the groups, which the menu lists and a favorite's kind needs, cannot be read: the menu fails, rather than leaving
  # them out
  printf 'favorites: [w, github.com/o/a]\nworkspaces: foo\n' | workspaces-file
  run --separate-stderr kit-bash 'workspaces:menu-json "{\"kind\": \"none\", \"member\": null}"'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"the groups in ${XDG_CONFIG_HOME}/kit/workspaces.yml could not be read, so the menu cannot list"* ]]
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
  run --separate-stderr "${kit_root}/bin/kit" --workspace=w workspaces --current --text
  [ "$status" -eq 0 ]
  [ "$output" = "📚 w (given)" ]
  [[ "$stderr" != *"broken"* ]]
}

@test "--current --json walks the roots once, however many repository favorites there are" {
  clone-at "${BATS_TEST_TMPDIR}/code/broken" git@github.com:a/broken.git
  printf 'primary: [\n' > "${BATS_TEST_TMPDIR}/code/broken/kit.yml"
  # A yq that logs its words: each walk of the roots reads the broken clone's kit.yml again, as a read that fails is
  # never kept.
  calls=${BATS_TEST_TMPDIR}/yq-calls
  mkdir -p "${BATS_TEST_TMPDIR}/spy"
  printf '#!/usr/bin/env bash\nprintf "%%s\\n" "$*" >> %q\nexec %q "$@"\n' "${calls}" "$(command -v yq)" \
    > "${BATS_TEST_TMPDIR}/spy/yq"
  chmod +x "${BATS_TEST_TMPDIR}/spy/yq"
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  printf 'roots: [%s/code]\nfavorites: [github.com/a/one]\n' "${BATS_TEST_TMPDIR}" | workspaces-file
  PATH="${BATS_TEST_TMPDIR}/spy:${PATH}" run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$status" -eq 0 ]
  [ -z "$stderr" ]
  once=$(grep --count 'broken/kit.yml' "${calls}")
  : > "${calls}"
  printf 'roots: [%s/code]\nfavorites: [github.com/a/one, github.com/a/two, github.com/a/three]\n' \
    "${BATS_TEST_TMPDIR}" | workspaces-file
  PATH="${BATS_TEST_TMPDIR}/spy:${PATH}" run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$status" -eq 0 ]
  (( once > 0 ))
  [ "$(grep --count 'broken/kit.yml' "${calls}")" -eq "${once}" ]
}

@test "--current --json lists every group in the file, with its count, wherever kit starts" {
  clone-at "${BATS_TEST_TMPDIR}/code/a" git@github.com:o/a.git
  workspaces-file <<YML
roots: [${BATS_TEST_TMPDIR}/code]
workspaces:
  w:
    repos: [github.com/o/a, github.com/o/b]
  x:
    repos: [github.com/o/b]
YML
  cd "${BATS_TEST_TMPDIR}/code/a"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output .workspaces <<<"$output")" = '[{"name":"w","members":2},{"name":"x","members":1}]' ]
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output '[.kind, (.workspaces | map(.name))]' <<<"$output")" = '["none",["w","x"]]' ]
}

@test "--workspace= with a folder's absolute path names the repository whose working tree holds it" {
  clone-at "${BATS_TEST_TMPDIR}/far/a" git@github.com:o/a.git
  mkdir -p "${BATS_TEST_TMPDIR}/far/a/sub" "${BATS_TEST_TMPDIR}/plain"
  git init --quiet "${BATS_TEST_TMPDIR}/far/local"
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run --separate-stderr "${kit_root}/bin/kit" --workspace="${BATS_TEST_TMPDIR}/far/a/sub" workspaces --current --json
  [ "$status" -eq 0 ]
  top=$(cd "${BATS_TEST_TMPDIR}/far/a" && pwd -P)
  [ "$(jq --compact-output '[.kind, .name, .member, .checkout, .why]' <<<"$output")" \
    = '["repository","github.com/o/a","github.com/o/a","'"${top}"'","given"]' ]
  run --separate-stderr "${kit_root}/bin/kit" --workspace="${BATS_TEST_TMPDIR}/far/local" workspaces --current --json
  [ "$(jq --compact-output '[.kind, .name, .member]' <<<"$output")" \
    = '["repository","'"$(cd "${BATS_TEST_TMPDIR}/far/local" && pwd -P)"'",null]' ]
  run --separate-stderr "${kit_root}/bin/kit" --workspace="${BATS_TEST_TMPDIR}/plain" workspaces --current --json
  [ "$status" -eq 2 ]
  [ -z "$output" ]
  [[ "$stderr" == *"'${BATS_TEST_TMPDIR}/plain' is no git working tree"* ]]
  run --separate-stderr "${kit_root}/bin/kit" --workspace="${BATS_TEST_TMPDIR}/nowhere" workspaces --current --json
  [ "$status" -eq 2 ]
  [[ "$stderr" == *"'${BATS_TEST_TMPDIR}/nowhere' is no git working tree"* ]]
}

@test "--use with a folder remembers its repository among the recent, the newest first, each once, ten at most" {
  for n in 0 1 2 3 4 5 6 7 8 9 10 11
  do
    git init --quiet "${BATS_TEST_TMPDIR}/r/${n}"
  done
  clone-at "${BATS_TEST_TMPDIR}/r/a" git@github.com:o/a.git
  mkdir -p "${BATS_TEST_TMPDIR}/r/a/sub" "${BATS_TEST_TMPDIR}/plain"
  cd "${BATS_TEST_TMPDIR}/r/a/sub"
  run "${kit_root}/bin/kit" workspaces --use=.
  [ "$status" -eq 0 ]
  top=$(cd "${BATS_TEST_TMPDIR}/r/a" && pwd -P)
  [[ "$output" == *"${top} is among the recent repositories"* ]]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq --compact-output .recent <<<"$output")" \
    = '[{"name":"github.com/o/a","member":"github.com/o/a","checkout":"'"${top}"'"}]' ]
  for n in 0 1 2 3 4 5 6 7 8 9 10 11
  do
    "${kit_root}/bin/kit" workspaces --use="${BATS_TEST_TMPDIR}/r/${n}" >/dev/null
  done
  "${kit_root}/bin/kit" workspaces --use="${BATS_TEST_TMPDIR}/r/5" >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  r=$(cd "${BATS_TEST_TMPDIR}/r" && pwd -P)
  [ "$(jq --compact-output '[.recent[] | .checkout]' <<<"$output")" = "$(printf '%s\n' 5 11 10 9 8 7 6 4 3 2 \
    | sed "s|^|${r}/|" | jq --raw-input --slurp --compact-output 'split("\n") | map(select(. != ""))')" ]
  # a folder that is no longer a working tree is left out
  rm -rf "${BATS_TEST_TMPDIR}/r/11"
  run --separate-stderr "${kit_root}/bin/kit" workspaces --current --json
  [ "$(jq '.recent | length' <<<"$output")" -eq 9 ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --use="${BATS_TEST_TMPDIR}/plain"
  [ "$status" -eq 2 ]
  [[ "$stderr" == *"is no git working tree"* ]]
  # the last group used is as it was
  [ ! -f "${XDG_CACHE_HOME}/kit/workspaces/last" ]
}

@test "a pattern among the repositories matches the checkouts under the roots, and --remove's, the group's members" {
  clone-at "${BATS_TEST_TMPDIR}/code/a" git@github.com:o/homebrew-a.git
  clone-at "${BATS_TEST_TMPDIR}/code/b" git@github.com:O/Homebrew-B.git
  clone-at "${BATS_TEST_TMPDIR}/code/c" git@github.com:o/other.git
  printf 'roots: [%s/code]\n' "${BATS_TEST_TMPDIR}" | workspaces-file
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  run "${kit_root}/bin/kit" workspaces --new=brew 'github.com/o/homebrew-*' github.com/o/kit
  [ "$status" -eq 0 ]
  [ "$(yq --output-format=json --indent=0 '.workspaces.brew.repos' "${XDG_CONFIG_HOME}/kit/workspaces.yml")" \
    = '["github.com/o/homebrew-a","github.com/o/homebrew-b","github.com/o/kit"]' ]
  run "${kit_root}/bin/kit" workspaces --remove=brew 'github.com/o/home*'
  [ "$status" -eq 0 ]
  [ "$(yq --output-format=json --indent=0 '.workspaces.brew.repos' "${XDG_CONFIG_HOME}/kit/workspaces.yml")" \
    = '["github.com/o/kit"]' ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --add=brew 'github.com/x/*'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *"'github.com/x/*' matches no repository among the checkouts under the roots"* ]]
}

@test "--from runs a command whose lines are repositories; a pattern keeps those it matches" {
  printf 'workspaces: {}\n' | workspaces-file
  mkdir -p "${BATS_TEST_TMPDIR}/outside" && cd "${BATS_TEST_TMPDIR}/outside"
  # shellcheck disable=SC2016  # the command is the verb's to run
  list='printf "%s\n" https://github.com/o/a git@github.com:o/b-tool.git "" github.com/o/c-tool'
  run "${kit_root}/bin/kit" workspaces --new=all --from="${list}"
  [ "$status" -eq 0 ]
  [ "$(yq --output-format=json --indent=0 '.workspaces.all.repos' "${XDG_CONFIG_HOME}/kit/workspaces.yml")" \
    = '["github.com/o/a","github.com/o/b-tool","github.com/o/c-tool"]' ]
  run "${kit_root}/bin/kit" workspaces --new=tools --from="${list}" 'github.com/o/*-tool'
  [ "$status" -eq 0 ]
  [ "$(yq --output-format=json --indent=0 '.workspaces.tools.repos' "${XDG_CONFIG_HOME}/kit/workspaces.yml")" \
    = '["github.com/o/b-tool","github.com/o/c-tool"]' ]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --add=tools --from='echo owner/repo'
  [ "$status" -eq 2 ]
  [[ "$stderr" == *"'owner/repo', which the command printed, names no repository"* ]]
  run --separate-stderr "${kit_root}/bin/kit" workspaces --add=tools --from='echo partial; exit 3'
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"the command failed"* ]]
  [ "$(yq '.workspaces.tools.repos | length' "${XDG_CONFIG_HOME}/kit/workspaces.yml")" -eq 2 ]
}

@test "--json gives each group's members" {
  empowerite-file
  fresh-repo >/dev/null
  run --separate-stderr "${kit_root}/bin/kit" workspaces --json
  [ "$(jq --compact-output '[.workspaces[] | .repos]' <<<"$output")" \
    = '[["github.com/empowerite/kit","github.com/empowerite/edge"],["github.com/me/solo"]]' ]
}
