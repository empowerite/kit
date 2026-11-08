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
