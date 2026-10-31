#!/usr/bin/env bats
# releases.bats - kit release tags HEAD and pushes the tag after the release rules; kit releases lists the forge's
# ----------------------------------------------------------------------------------------------------------------------
#
# origin is a bare repository beside the test's, so the tags and pushes are real git; gh is tests/fakes/fake-tool, so
# the listing is the provider's folding of a fixture.

bats_require_minimum_version 1.5.0

setup() {
  export NO_COLOR=1 kit_root
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export FAKE_DIR="${BATS_TEST_TMPDIR}/fake"
  mkdir -p "${FAKE_DIR}/bin"
  ln -s "${BATS_TEST_DIRNAME}/fakes/fake-tool" "${FAKE_DIR}/bin/gh"
  export PATH="${FAKE_DIR}/bin:${PATH}"
  repo=$(mktemp -d)
  git init --quiet --bare --initial-branch=main "${repo}/origin.git"
  git clone --quiet "${repo}/origin.git" "${repo}/desk" 2>/dev/null
  cd "${repo}/desk"
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  git config tag.gpgsign false
  commit one && git push --quiet origin main && git remote set-head origin main
}

teardown() {
  rm -rf "${repo}"
}

# commit <name> makes a commit that adds a file of that name.
commit() { printf '%s\n' "$1" > "$1" && git add "$1" && git commit --quiet --message "$1"; }

# on-origin <tag> prints the commit an annotated tag names on origin, or nothing.
on-origin() { git ls-remote origin "refs/tags/$1*" | awk -v want="refs/tags/$1^{}" '$2 == want { print $1 }'; }

@test "kit release tags HEAD, annotated, and pushes that tag and no branch" {
  run "${kit_root}/bin/kit" release 0.1.0
  [ "$status" -eq 0 ]
  [ "$(on-origin release/0.1.0)" = "$(git rev-parse HEAD)" ]
  [ "$(git cat-file -t release/0.1.0)" = tag ]
  [[ "$output" == *"pushed release/0.1.0, at $(git rev-parse --short HEAD) one"* ]]
  git switch --quiet --create topic && commit two && commit three
  run "${kit_root}/bin/kit" release 0.2.0-rc.1
  [ "$status" -eq 0 ]
  [ "$(on-origin release/0.2.0-rc.1)" = "$(git rev-parse HEAD)" ]
  [[ "$output" == *"the push sends 2 commit(s) no branch on origin has"* ]]
  [ -z "$(git ls-remote --heads origin topic)" ]
}

@test "kit release refuses a word that is not a version, and a tag that exists" {
  run "${kit_root}/bin/kit" release 1.2
  [ "$status" -eq 2 ]
  [[ "$output" == *"'1.2' is not a version: <major>.<minor>.<patch>, with -<suffix> for a pre-release; run: kit help release"* ]]
  "${kit_root}/bin/kit" release 0.1.0
  run "${kit_root}/bin/kit" release 0.1.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"release/0.1.0 exists already; a version is tagged once"* ]]
}

@test "a release is tagged on the default branch at origin; a pre-release may be anywhere" {
  git switch --quiet --create topic && commit two
  run "${kit_root}/bin/kit" release 1.0.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"1.0.0 is a release, and HEAD is not on main at origin; tag a release there, or give a pre-release a suffix"* ]]
  [ -z "$(on-origin release/1.0.0)" ]
  [ -z "$(git tag --list release/1.0.0)" ]
  run "${kit_root}/bin/kit" release 1.0.0-rc.1
  [ "$status" -eq 0 ]
}

@test "a version not higher than every release tag on origin is refused unless --lower confirms it" {
  "${kit_root}/bin/kit" release 0.2.0-rc.2
  run "${kit_root}/bin/kit" release 0.2.0-rc.1
  [ "$status" -eq 1 ]
  [[ "$output" == *"0.2.0-rc.1 is not higher than 0.2.0-rc.2, the highest release; run: kit release 0.2.0-rc.1 --lower"* ]]
  [ -z "$(git tag --list release/0.2.0-rc.1)" ]
  run "${kit_root}/bin/kit" release 0.2.0-rc.10
  [ "$status" -eq 0 ]
  run "${kit_root}/bin/kit" release 0.2.0-beta
  [ "$status" -eq 1 ]
  run "${kit_root}/bin/kit" release 0.2.0-rc.1 --lower
  [ "$status" -eq 0 ]
  [ "$(on-origin release/0.2.0-rc.1)" = "$(git rev-parse HEAD)" ]
  run "${kit_root}/bin/kit" release 0.2.0
  [ "$status" -eq 0 ]
}

@test "kit releases lists the forge's releases newest first: version, kind, age, and the latest" {
  printf 'forge: github\n' > .kit-settings && mkdir -p .kit && mv .kit-settings .kit/settings.yml
  : > "${FAKE_DIR}/gh.routes"
  cat > "${FAKE_DIR}/releases.json" <<'JSON'
[{"tagName": "release/0.1.0-rc.6", "name": "kit 0.1.0-rc.6", "isPrerelease": true, "isDraft": false, "isLatest": false,
  "publishedAt": "2026-10-27T07:30:14Z", "createdAt": "2026-10-27T07:30:14Z"},
 {"tagName": "release/0.0.9", "name": "kit 0.0.9", "isPrerelease": false, "isDraft": false, "isLatest": true,
  "publishedAt": "2026-10-20T07:30:14Z", "createdAt": "2026-10-20T07:30:14Z"},
 {"tagName": "release/0.1.0-rc.7", "name": "kit 0.1.0-rc.7", "isPrerelease": true, "isDraft": true, "isLatest": false,
  "publishedAt": null, "createdAt": "2026-10-30T07:30:14Z"}]
JSON
  printf '{"login": "ada"}\n' > "${FAKE_DIR}/user.json"
  printf '%s\t%s\n' 'api user*' user.json >> "${FAKE_DIR}/gh.routes"
  printf '%s\t%s\n' 'release list --limit 1000 --json *' releases.json >> "${FAKE_DIR}/gh.routes"
  run "${kit_root}/bin/kit" releases
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "0.1.0-rc.7  📝 draft "* ]]
  [[ "${lines[1]}" == "0.1.0-rc.6  🧪 pre-release "* ]]
  [[ "${lines[2]}" == "0.0.9       📦 release "*"  ⭐ latest" ]]
  run "${kit_root}/bin/kit" releases --json
  [ "$(jq -r 'map("\(.version) \(.kind) \(.latest)") | join(", ")' <<<"$output")" \
    = "0.1.0-rc.7 draft false, 0.1.0-rc.6 pre-release false, 0.0.9 release true" ]
}

# ----------------------------------------------------------------------------------------------------------------------
