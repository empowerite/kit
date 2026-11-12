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
  [ "${lines[1]}" = "🛬 fetched 1 new commit on main" ]
  [ "${lines[2]}" = "" ]
  [ "${lines[3]}" = "📡 upstream · gitlab · ../upstream.git" ]
  [ "${lines[4]}" = "🛬 fetched 1 new commit on main · 1 new branch · 1 new tag" ]
  [ "${#lines[@]}" -eq 5 ]
  run kit flow incoming
  [ "$output" = "📡 origin · github · ${work}/origin.git"$'\n'"🛬 unchanged"$'\n\n'"📡 upstream · gitlab · ../upstream.git"$'\n'"🛬 unchanged" ]
}

@test "incoming --json is the primary and one object per remote, each with what came from it" {
  push-to upstream u1 main main:feature HEAD:refs/tags/v2
  run kit flow incoming --json
  [ "$status" -eq 0 ]
  [ "$(jq -c 'keys_unsorted' <<<"$output")" = '["fetched","primary","remotes","rows"]' ]
  [ "$(jq -c '[.fetched, .primary, [.remotes[].name]]' <<<"$output")" = '[true,"origin",["origin","upstream"]]' ]
  [ "$(jq -c '.remotes[0] | keys_unsorted' <<<"$output")" \
    = '["name","forge","host","default","commits","branches","tags"]' ]
  [ "$(jq -c '.remotes[0]' <<<"$output")" = '{"name":"origin","forge":"github","host":null,"default":"main",'"\
"'"commits":[],"branches":{"new":[],"updated":[],"gone":[]},"tags":[]}' ]
  [ "$(jq -c '.remotes[1] | [.forge, .default, (.commits | map(.subject)), .branches, .tags]' <<<"$output")" \
    = '["gitlab","main",["u1"],{"new":["feature"],"updated":[],"gone":[]},["v2"]]' ]
}

@test "incoming --json's rows are one a thing incoming: each remote, commit, branch, and tag, and each local branch behind" {
  push-to origin o0 main:7-mine
  git fetch --quiet origin
  git branch --quiet --track 7-mine origin/7-mine
  git worktree add --quiet ../wt 7-mine
  push-to origin '#7 o1' main main:42-new HEAD:refs/tags/v1
  git -C "${work}/other" push --quiet --force origin main:7-mine
  run kit flow incoming --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.rows[] | [.remote, .kind, .change, .ref]]' <<<"$output")" = '[["origin","remote","fetched","main"],'"\
"'["origin","commit","new","main"],["origin","branch","new","42-new"],["origin","branch","forced","7-mine"],'"\
"'["origin","tag","new","v1"],["origin","tracking","behind","7-mine"],["origin","tracking","behind","main"],'"\
"'["upstream","remote","unchanged","main"]]' ]
  [ "$(jq -c '.rows[0] | keys_unsorted' <<<"$output")" = '["key","remote","primary","forge","host","url","default",'"\
"'"kind","change","ref","sha","old","commits","author","age","subject","local","worktrees","ahead","behind","issues",'"\
"'"organization","repository","source"]' ]
  [ "$(jq -c '[.rows[] | .source] | unique' <<<"$output")" = '["git"]' ]
  [ "$(jq -c '[.rows[] | [.key, .primary]] | .[0], .[1][1], .[-1]' <<<"$output" | paste -sd ' ' -)" \
    = '["origin remote","primary"] "primary" ["upstream remote",null]' ]
  # a commit names the issues its subject does; a branch the one its name starts with, and the local branch and
  # worktree on it; a tracking row how far ahead and behind the local branch is
  [ "$(jq -c '.rows[1] | [.subject, .issues, (.sha | length >= 7), (.age >= 0), (.author != null)]' <<<"$output")" \
    = '["#7 o1",["#7"],true,true,true]' ]
  [ "$(jq -c '.rows[2] | [.commits, .issues, .local]' <<<"$output")" = '[0,["#42","#7"],[]]' ]
  [ "$(jq -c '.rows[3] | [.commits, (.old | length), .local, (.worktrees | map(split("/")[-1]))]' <<<"$output")" \
    = '[1,7,["7-mine"],["wt"]]' ]
  [ "$(jq -c '.rows[5:7] | map([.local, .ahead, .behind, .commits])' <<<"$output")" \
    = '[[["7-mine"],1,1,1],[["main"],0,1,1]]' ]
  # without the fetch, each remote says so, and the local branches behind stand as the last fetch left them
  run kit flow incoming --no-fetch --json
  [ "$(jq -c '[.rows[] | [.remote, .kind, .change, .ref]]' <<<"$output")" = '[["origin","remote","not fetched","main"],'"\
"'["origin","tracking","behind","7-mine"],["origin","tracking","behind","main"],["upstream","remote","not fetched","main"]]' ]
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

@test "the primary setting names the primary: it is fetched first, and what is behind counts against it" {
  printf 'forge: gitlab\nprimary: upstream\nforges: {origin: github}\n' > kit.yml
  push-to upstream u1 main
  push-to upstream u2 main
  run kit flow incoming --json
  [ "$status" -eq 0 ]
  [ "$(jq -c '[.primary, [.remotes[] | [.name, .forge, (.commits | length)]]]' <<<"$output")" \
    = '["upstream",[["upstream","gitlab",2],["origin","github",0]]]' ]
  run kit flow incoming
  in-order "${lines[0]}" 'upstream' 'gitlab' '../upstream.git'
  run kit workspace worktrees --json
  [ "$(jq -c '.rows[0] | [.kind, .base]' <<<"$output")" = '["desk",[0,2]]' ]
}

@test "a remote whose fetch fails shows ❌ and git's line, the others still show, and the stage exits 1" {
  git remote add gone "${work}/nowhere.git"
  push-to upstream u1 main
  run --keep-empty-lines kit flow incoming
  [ "$status" -eq 1 ]
  in-order "${lines[1]}" 'unchanged'
  in-order "${lines[3]}" 'gone' 'no forge' "${work}/nowhere.git"
  [[ "${lines[4]}" == "❌ fatal: "*"nowhere.git"* ]]
  in-order "${lines[6]}" 'upstream' 'gitlab' '../upstream.git'
  in-order "${lines[7]}" 'fetched 1 new commit on main'
  run kit flow incoming --json
  [ "$status" -eq 1 ]
  [ "$(jq -c '[.remotes[] | [.name, (.error != null)]]' <<<"$output")" \
    = '[["origin",false],["gone",true],["upstream",false]]' ]
  [ "$(jq -c '.remotes[1] | [.commits, .branches.new, .tags]' <<<"$output")" = '[[],[],[]]' ]
  # each remote's row says how its fetch went, the failed one with git's line
  [ "$(jq -c '[.rows[] | select(.kind == "remote") | [.remote, .change, (.subject != null)]]' <<<"$output")" \
    = '[["origin","unchanged",false],["gone","failed",true],["upstream","unchanged",false]]' ]
  # in the picture, incoming draws the failed remote beside the others, and the picture exits 1
  printf 'forge: github\nforges: {upstream: gitlab}\nflow: [incoming]\n' > kit.yml
  run kit flow
  [ "$status" -eq 1 ]
  in-order "$output" $'INCOMING\n' 'origin' 'github' 'gone' 'no forge' $'\n❌ fatal: '
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
  [ "$(jq -c 'del(.rows)' <<<"$output")" = '{"fetched":false,"primary":"origin","remotes":[{"name":"origin","forge":"github","host":null},'"\
"'{"name":"mirror","forge":"github","host":"git.example.test"},{"name":"plain","forge":null,"host":"elsewhere.test"},'"\
"'{"name":"upstream","forge":"gitlab","host":null}]}' ]
  GH_CONFIG_DIR="${BATS_TEST_TMPDIR}/gh" run kit flow incoming --no-fetch
  in-order "${lines[2]}" 'mirror' 'github' 'git.example.test'
  in-order "${lines[3]}" 'not fetched'
  in-order "${lines[4]}" 'plain' 'no forge' 'elsewhere.test'
}

@test "kit help flow holds the incoming legend: the remote's glyph and a failed fetch" {
  run kit help flow
  [ "$status" -eq 0 ]
  [[ "$output" == *$'\n  incoming:\n\n    🛬  fetch: what it brought from the remotes\n    📡  remote: its name, its forge or no forge, and its host\n'* ]]
  [[ "$output" == *$'\n    ❌  failed: a check, a build, or a fetch, or kit could not read the forge\n'* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
