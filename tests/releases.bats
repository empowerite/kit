#!/usr/bin/env bats
# releases.bats - kit release tags HEAD and pushes the tag after the release rules; kit flow releases lists the forge's
# ----------------------------------------------------------------------------------------------------------------------
# origin is a bare repository beside the test's, so the tags and pushes are real git; gh is tests/fakes/fake-tool, so
# the listing is the provider's folding of a fixture.

bats_require_minimum_version 1.5.0

load helper

setup() {
  fake-tools gh
  repo=$(mktemp -d)
  git init --quiet --bare --initial-branch=main "${repo}/origin.git"
  git clone --quiet "${repo}/origin.git" "${repo}/desk" 2>/dev/null
  cd "${repo}/desk"
  commit one && git push --quiet origin main && git remote set-head origin main
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
  [[ "${lines[0]}" == *"tagging $(git rev-parse --short HEAD) one as release/0.1.0"* ]]
  [[ "${lines[${#lines[@]} - 1]}" == *"pushed release/0.1.0"* ]]
  git switch --quiet --create topic && commit two && commit three
  run "${kit_root}/bin/kit" release 0.2.0-rc.1
  [ "$status" -eq 0 ]
  [ "$(on-origin release/0.2.0-rc.1)" = "$(git rev-parse HEAD)" ]
  [[ "$output" == *"the push sends 2 commit(s) no branch on origin has"* ]]
  [ -z "$(git ls-remote --heads origin topic)" ]
}

@test "kit release pushes the tag to the primary remote, and judges the rules there" {
  git init --quiet --bare --initial-branch=main "${repo}/upstream.git"
  git remote add upstream "${repo}/upstream.git"
  git push --quiet upstream main
  printf 'primary: upstream\n' > kit.yml
  git push --quiet upstream HEAD:refs/tags/release/0.3.0
  run "${kit_root}/bin/kit" release 0.2.0
  [ "$status" -eq 1 ]
  [[ "$output" == *"0.2.0 is not higher than 0.3.0, the highest release"* ]]
  run "${kit_root}/bin/kit" release 0.4.0
  [ "$status" -eq 0 ]
  [ -n "$(git ls-remote upstream refs/tags/release/0.4.0)" ]
  [ -z "$(on-origin release/0.4.0)" ]
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

@test "a release whose tags on origin cannot be read is refused, and nothing is tagged" {
  "${kit_root}/bin/kit" release 0.3.0
  # origin is read from a repository that is not there, and pushed to the one that is
  git remote set-url origin "${repo}/gone.git"
  git remote set-url --push origin "${repo}/origin.git"
  run "${kit_root}/bin/kit" release 0.2.0-rc.1
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: git ls-remote --tags origin (the release tags on origin could not be read)"* ]]
  [ -z "$(git tag --list release/0.2.0-rc.1)" ]
  [ -z "$(git ls-remote "${repo}/origin.git" 'refs/tags/release/0.2.0-rc.1')" ]
}

@test "kit flow releases lists the forge's releases newest first: version, kind, age, and the latest" {
  printf 'forge: github\n' > kit.yml
  : > "${FAKE_DIR}/gh.routes"
  # gh api --paginate prints each page's array after the one before
  cat > "${FAKE_DIR}/releases.json" <<'JSON'
[{"tag_name": "release/0.1.0-rc.6", "name": "kit 0.1.0-rc.6", "prerelease": true, "draft": false,
  "published_at": "2026-10-27T07:30:14Z", "created_at": "2026-10-27T07:30:14Z",
  "html_url": "https://github.com/empowerite/kit/releases/tag/release/0.1.0-rc.6",
  "body": "## What's Changed\r\n\r\n* #42 settings\r\n",
  "assets": [{"name": "kit-0.1.0-rc.6.tar.gz",
              "browser_download_url": "https://github.com/empowerite/kit/releases/download/release/0.1.0-rc.6/kit-0.1.0-rc.6.tar.gz"},
             {"name": "kit-0.1.0-rc.6.tar.gz.sha256",
              "browser_download_url": "https://github.com/empowerite/kit/releases/download/release/0.1.0-rc.6/kit-0.1.0-rc.6.tar.gz.sha256"}]},
 {"tag_name": "release/0.0.9", "name": "kit 0.0.9", "prerelease": false, "draft": false,
  "published_at": "2026-10-20T07:30:14Z", "created_at": "2026-10-20T07:30:14Z",
  "html_url": "https://github.com/empowerite/kit/releases/tag/release/0.0.9", "body": null, "assets": []}]
[{"tag_name": "release/0.1.0-rc.7", "name": "kit 0.1.0-rc.7", "prerelease": true, "draft": true,
  "published_at": null, "created_at": "2026-10-30T07:30:14Z",
  "html_url": "https://github.com/empowerite/kit/releases/tag/untagged-1", "body": "", "assets": []}]
JSON
  printf '{"login": "ada"}\n' > "${FAKE_DIR}/user.json"
  printf '%s\t%s\n' 'api user*' user.json >> "${FAKE_DIR}/gh.routes"
  printf '%s\t%s\n' 'api --paginate repos/{owner}/{repo}/releases?per_page=100' releases.json >> "${FAKE_DIR}/gh.routes"
  run "${kit_root}/bin/kit" flow releases
  [ "$status" -eq 0 ]
  [[ "${lines[0]}" == "0.1.0-rc.7  📝 draft "* ]]
  [[ "${lines[1]}" == "0.1.0-rc.6  🧪 pre-release "* ]]
  [[ "${lines[2]}" == "0.0.9       📦 release "*"  ⭐ latest" ]]
  # the ages end in one column
  local line widths=()
  for line in "${lines[@]}"
  do
    line=${line%  ⭐ latest}
    widths+=("${#line}")
  done
  [ "${widths[0]}" -eq "${widths[1]}" ] && [ "${widths[1]}" -eq "${widths[2]}" ]
  run "${kit_root}/bin/kit" flow releases --json
  [ "$(jq -r 'map("\(.key) \(.kind) \(.latest)") | join(", ")' <<<"$output")" \
    = "release/0.1.0-rc.7 draft null, release/0.1.0-rc.6 pre-release null, release/0.0.9 release latest" ]
}

@test "a release's row in kit flow releases --json leads to its page and its assets, and carries its notes" {
  printf 'forge: github\n' > kit.yml
  : > "${FAKE_DIR}/gh.routes"
  cat > "${FAKE_DIR}/releases.json" <<'JSON'
[{"tag_name": "release/0.1.0", "name": "kit 0.1.0", "prerelease": false, "draft": false,
  "published_at": "2026-10-27T07:30:14Z", "created_at": "2026-10-27T07:30:14Z",
  "html_url": "https://github.com/empowerite/kit/releases/tag/release/0.1.0",
  "body": "## What's Changed\r\n\r\n* #42 settings\r\n",
  "assets": [{"name": "kit-0.1.0.tar.gz", "browser_download_url": "https://example.test/kit-0.1.0.tar.gz"},
             {"name": "kit-0.1.0.tar.gz.sha256", "browser_download_url": "https://example.test/kit-0.1.0.tar.gz.sha256"}]},
 {"tag_name": "release/0.0.9", "name": "kit 0.0.9", "prerelease": false, "draft": false,
  "published_at": "2026-10-20T07:30:14Z", "created_at": "2026-10-20T07:30:14Z",
  "html_url": "https://github.com/empowerite/kit/releases/tag/release/0.0.9", "body": null, "assets": []}]
JSON
  printf '{"login": "ada"}\n' > "${FAKE_DIR}/user.json"
  printf '%s\t%s\n' 'api user*' user.json >> "${FAKE_DIR}/gh.routes"
  printf '%s\t%s\n' 'api --paginate repos/{owner}/{repo}/releases?per_page=100' releases.json >> "${FAKE_DIR}/gh.routes"
  run "${kit_root}/bin/kit" flow releases --json
  [ "$status" -eq 0 ]
  [ "$(jq --compact-output '.[0] | del(.age, .organization, .repository, .source)' <<<"$output")" \
    = "$(jq --compact-output . <<'JSON'
{"key": "release/0.1.0", "version": "0.1.0", "tag": "release/0.1.0", "name": "kit 0.1.0", "kind": "release",
 "latest": "latest", "at": "2026-10-27T07:30:14Z",
 "url": "https://github.com/empowerite/kit/releases/tag/release/0.1.0",
 "notes": "## What's Changed\n\n* #42 settings\n",
 "assets": ["kit-0.1.0.tar.gz", "kit-0.1.0.tar.gz.sha256"],
 "downloads": ["https://example.test/kit-0.1.0.tar.gz", "https://example.test/kit-0.1.0.tar.gz.sha256"],
 "links": {"kind": "https://github.com/empowerite/kit/releases/tag/release/0.1.0",
           "url": "https://github.com/empowerite/kit/releases/tag/release/0.1.0",
           "assets": ["https://example.test/kit-0.1.0.tar.gz", "https://example.test/kit-0.1.0.tar.gz.sha256"],
           "downloads": ["https://example.test/kit-0.1.0.tar.gz", "https://example.test/kit-0.1.0.tar.gz.sha256"]}}
JSON
)" ]
  # a release without notes or assets has empty ones
  [ "$(jq --compact-output '.[1] | [.notes, .assets, .downloads, .links.assets]' <<<"$output")" = '["",[],[],[]]' ]
}

# ----------------------------------------------------------------------------------------------------------------------
