#!/usr/bin/env bats
# incoming.bats - kit flow incoming fetches every remote, the primary first, and says what came from each
# ----------------------------------------------------------------------------------------------------------------------
# Two bare repositories beside the desk are its remotes, origin and upstream, so every fetch is real git. kit.yml
# names their forges: origin is github through the forge setting, and upstream gitlab through the forges map. A
# remote is moved on from a clone of its own, as another desk would push to it.

bats_require_minimum_version 1.5.0

load helper

setup() {
  work=$(mktemp -d)
  cd "${work}"
  git init --quiet --bare --initial-branch=main origin.git
  git init --quiet --bare --initial-branch=main upstream.git
  git clone --quiet origin.git desk 2>/dev/null
  cd desk
  printf 'forge: github\nforges: {upstream: gitlab}\n' > kit.yml
  git add kit.yml && git commit --quiet --message one
  git push --quiet origin main
  git remote set-head origin main
  git remote add upstream ../upstream.git
  git push --quiet upstream main
  git fetch --quiet upstream
}

# push-to <remote> <message> <refspec>… makes a commit in a clone of the bare <remote> and pushes the refspecs.
push-to() {
  local remote=$1 message=$2
  shift 2
  rm -rf "${work}/other"
  git clone --quiet "${work}/${remote}.git" "${work}/other" 2>/dev/null
  (
    cd "${work}/other"
    printf '%s\n' "${message}" > "${message}"
    git add "${message}" && git commit --quiet --message "${message}"
    git push --quiet origin "$@"
  )
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

@test "incoming fetches every remote, the primary first, and draws a block for each: its name, forge, and host" {
  push-to upstream u1 main main:feature HEAD:refs/tags/v2
  push-to origin o1 main
  run --keep-empty-lines kit flow incoming
  [ "$status" -eq 0 ]
  [ "${lines[0]}" = "📡 origin · github · ${work}/origin.git" ]
  [ "${lines[1]}" = "📥 fetched 1 new commit on main" ]
  [ "${lines[2]}" = "" ]
  [ "${lines[3]}" = "📡 upstream · gitlab · ../upstream.git" ]
  [ "${lines[4]}" = "📥 fetched 1 new commit on main · 1 new branch · 1 new tag" ]
  [ "${#lines[@]}" -eq 5 ]
  run kit flow incoming
  [ "$output" = "📡 origin · github · ${work}/origin.git"$'\n'"📥 unchanged"$'\n\n'"📡 upstream · gitlab · ../upstream.git"$'\n'"📥 unchanged" ]
}

@test "incoming --json is the primary and one object per remote, each with what came from it" {
  push-to upstream u1 main main:feature HEAD:refs/tags/v2
  run kit flow incoming --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'keys_unsorted' <<<"$output")" = '["fetched","primary","remotes"]' ]
  [ "$(jq -c '[.fetched, .primary, [.remotes[].name]]' <<<"$output")" = '[true,"origin",["origin","upstream"]]' ]
  [ "$(jq -c '.remotes[0] | keys_unsorted' <<<"$output")" \
    = '["name","forge","host","default","commits","branches","tags"]' ]
  [ "$(jq -c '.remotes[0]' <<<"$output")" = '{"name":"origin","forge":"github","host":null,"default":"main",'"\
"'"commits":[],"branches":{"new":[],"updated":[],"gone":[]},"tags":[]}' ]
  [ "$(jq -c '.remotes[1] | [.forge, .default, (.commits | map(.subject)), .branches, .tags]' <<<"$output")" \
    = '["gitlab","main",["u1"],{"new":["feature"],"updated":[],"gone":[]},["v2"]]' ]
}

@test "incoming never rebases, merges, or pulls: the desk's branch stays where it was" {
  push-to origin o1 main
  local before
  before=$(git rev-parse HEAD)
  run kit flow incoming
  [ "$status" -eq 0 ]
  [ "$(git rev-parse HEAD)" = "${before}" ]
  [ "$(git rev-parse origin/main)" != "${before}" ]
}

@test "the primary setting names the primary: it is fetched first, and ahead and behind count against it" {
  printf 'forge: gitlab\nprimary: upstream\nforges: {origin: github}\n' > kit.yml
  push-to upstream u1 main
  push-to upstream u2 main
  run kit flow incoming --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.primary, [.remotes[] | [.name, .forge, (.commits | length)]]]' <<<"$output")" \
    = '["upstream",[["upstream","gitlab",2],["origin","github",0]]]' ]
  run kit flow incoming
  [ "${lines[0]}" = "📡 upstream · gitlab · ../upstream.git" ]
  run kit flow desk --json
  [ "$(jq -c '[.ahead, .behind]' <<<"$output")" = '[0,2]' ]
  run kit flow desk
  [[ "${lines[0]}" == *"↑0 ↓2"* ]]
}

@test "a remote whose fetch fails shows ❌ and git's line, the others still show, and the stage exits 1" {
  git remote add gone "${work}/nowhere.git"
  push-to upstream u1 main
  run --keep-empty-lines kit flow incoming
  [ "$status" -eq 1 ]
  [ "${lines[1]}" = "📥 unchanged" ]
  [ "${lines[3]}" = "📡 gone · no forge · ${work}/nowhere.git" ]
  [[ "${lines[4]}" == "❌ fatal: "*"nowhere.git"* ]]
  [ "${lines[6]}" = "📡 upstream · gitlab · ../upstream.git" ]
  [ "${lines[7]}" = "📥 fetched 1 new commit on main" ]
  run kit flow incoming --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.remotes[] | [.name, (.error != null)]]' <<<"$output")" \
    = '[["origin",false],["gone",true],["upstream",false]]' ]
  [ "$(jq -c '.remotes[1] | [.commits, .branches.new, .tags]' <<<"$output")" = '[[],[],[]]' ]
  # in the picture, incoming draws the failed remote beside the others, and the picture exits 1
  printf 'forge: github\nforges: {upstream: gitlab}\nflow: [incoming, desk]\n' > kit.yml
  run kit flow
  [ "$status" -eq 1 ]
  [[ "$output" == *$'📥 INCOMING\n📡 origin · github · '*$'\n📡 gone · no forge · '*$'\n❌ fatal: '* ]]
  [[ "$output" == *$'\n💻 DESK 1\n'* ]]
  run kit flow --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.incoming.remotes[].name]' <<<"$output")" = '["origin","gone","upstream"]' ]
}

@test "--no-fetch fetches nothing and names each remote, its forge, and its host" {
  git remote add mirror https://git.example.test/team/kit.git
  git remote add plain https://elsewhere.test/team/kit.git
  mkdir -p "${BATS_TEST_TMPDIR}/gh"
  printf 'git.example.test:\n    user: ada\n' > "${BATS_TEST_TMPDIR}/gh/hosts.yml"
  push-to origin o1 main
  local before
  before=$(git rev-parse origin/main)
  GH_CONFIG_DIR="${BATS_TEST_TMPDIR}/gh" run kit flow incoming --no-fetch --json
  [ "$status" -eq 0 ]
  [ "$(git rev-parse origin/main)" = "${before}" ]
  [ "$output" = '{"fetched":false,"primary":"origin","remotes":[{"name":"origin","forge":"github","host":null},'"\
"'{"name":"mirror","forge":"github","host":"git.example.test"},{"name":"plain","forge":null,"host":"elsewhere.test"},'"\
"'{"name":"upstream","forge":"gitlab","host":null}]}' ]
  GH_CONFIG_DIR="${BATS_TEST_TMPDIR}/gh" run kit flow incoming --no-fetch
  [ "${lines[2]}" = "📡 mirror · github · git.example.test" ]
  [ "${lines[3]}" = "📥 not fetched" ]
  [ "${lines[4]}" = "📡 plain · no forge · elsewhere.test" ]
}

@test "kit help flow holds the incoming legend: the remote's glyph and a failed fetch" {
  run kit help flow
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\n  incoming:\n\n    📡  a remote: its name, its forge or no forge, and its host\n'* ]]
  [[ "$output" == *$'\n    ❌  the fetch failed: git\'s line\n'* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
