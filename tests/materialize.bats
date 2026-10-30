#!/usr/bin/env bats
# materialize.bats - what kit delivers under .kit/ is a link or a readonly copy, ignored, and healed by every verb
# ----------------------------------------------------------------------------------------------------------------------
#
# Each test gets an empty git repository under a temporary directory and runs the kit from this checkout against
# it. The checkout's release is outside the repository, so its files are copies; a test that wants links copies the
# release into the repository first.

bats_require_minimum_version 1.5.0

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  repo=$(mktemp -d)
  git -C "${repo}" init --quiet --initial-branch=main
  git -C "${repo}" config user.name t
  git -C "${repo}" config user.email t@example.com
  git -C "${repo}" config commit.gpgsign false
  export KIT_HOME="${repo}.home" XDG_CACHE_HOME="${repo}.cache"
  mkdir "${KIT_HOME}" "${XDG_CACHE_HOME}"
  cd "${repo}"
}

teardown() {
  rm -rf "${repo}" "${repo}.later" "${repo}.clone" "${repo}.home" "${repo}.cache"
}

# later-release <dir> copies this checkout's release to <dir> as 9.9.9, without the just rule.
later-release() {
  mkdir -p "$1"
  cp -R "${kit_checkout}/bin" "${kit_checkout}/bash" "${kit_checkout}/packages" "$1/"
  printf '9.9.9\n' > "$1/version"
  rm "$1/packages/kit/files/repo/.kit/rules/just.md"
}

@test "in a consumer a delivered file is a copy without the write bit, named in .kit/.gitignore, and git sees only the declaration" {
  run "${kit_checkout}/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ ! -L .kit/1st.md ]
  [ ! -w .kit/rules/bash.md ]
  grep --quiet --line-regexp '/rules/bash.md' .kit/.gitignore
  grep --quiet --line-regexp '/1st.md' .kit/.gitignore
  grep --quiet --line-regexp '/.gitignore' .kit/.gitignore
  grep --quiet --line-regexp '/manifest' .kit/.gitignore
  [ "$(git status --porcelain --untracked-files=all | grep ' \.kit/')" = "?? .kit/settings.yml" ]
  run bash -c 'echo edited > .kit/rules/bash.md'
  [ "$status" -ne 0 ]
  [ "$(head -n 1 .kit/rules/bash.md)" != "edited" ]
}

@test "a copy changed in place and a stray file are findings for check; the next verb restores the copy and leaves the stray to a person" {
  "${kit_checkout}/bin/kit" install
  chmod u+w .kit/rules/bash.md && printf 'edited\n' >> .kit/rules/bash.md
  printf 'rogue\n' > .kit/rules/rogue.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/bash.md was changed in place"* ]]
  [[ "$output" == *".kit/rules/rogue.md is not delivered by any active package"* ]]
  [[ "$output" == *"2 finding(s)"* ]]
  run "${kit_checkout}/bin/kit" context .kit/1st.md
  [ "$status" -eq 0 ]
  run ! grep --quiet '^edited$' .kit/rules/bash.md
  [ ! -w .kit/rules/bash.md ]
  [ -f .kit/rules/rogue.md ]
  rm .kit/rules/rogue.md
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "with the release inside the worktree a delivered file is a relative link into it, and a repository package's file a link into the package" {
  cp -R "${kit_checkout}" kit
  mkdir -p .kit/packages/house/files/repo/.kit/rules
  printf 'description: "house"\ndepends:\n  - kit\n' > .kit/packages/house/package.yml
  printf -- '---\npaths:\n  - "**/*.toml"\n---\n# TOML\n' > .kit/packages/house/files/repo/.kit/rules/toml.md
  run kit/bin/kit install house
  [ "$status" -eq 0 ]
  [ -L .kit/rules/bash.md ]
  [ "$(readlink .kit/rules/bash.md)" = "../../kit/packages/kit/files/repo/.kit/rules/bash.md" ]
  [ -L .kit/rules/toml.md ]
  [ "$(readlink .kit/rules/toml.md)" = "../packages/house/files/repo/.kit/rules/toml.md" ]
  # a write through the link lands in the source, where it belongs, and check stays green
  printf 'More.\n' >> .kit/rules/toml.md
  grep --quiet '^More.$' .kit/packages/house/files/repo/.kit/rules/toml.md
  run kit/bin/kit check
  [ "$status" -eq 0 ]
  # a link replaced by a regular file is a finding, and the next verb relinks it
  rm .kit/rules/bash.md && cp kit/packages/kit/files/repo/.kit/rules/bash.md .kit/rules/bash.md
  run kit/bin/kit check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/bash.md is not a link to ../../kit/packages/kit/files/repo/.kit/rules/bash.md"* ]]
  run kit/bin/kit hydrate
  [ "$status" -eq 0 ]
  [ -L .kit/rules/bash.md ]
}

@test "kit hydrate places what a fresh clone lacks from the declaration alone, and names a delivered path git still tracks" {
  "${kit_checkout}/bin/kit" install
  git add --all && git commit --quiet --message init
  git clone --quiet . "${repo}.clone" 2>/dev/null
  cd "${repo}.clone"
  [ ! -e .kit/1st.md ]
  [ ! -e .kit/.gitignore ]
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 0 ]
  [[ "$output" == *".kit/1st.md: placed"* ]]
  [ -f .kit/1st.md ]
  [ ! -w .kit/1st.md ]
  [ -f .kit/.gitignore ]
  [ -f .kit/manifest ]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 0 ]
  [[ "$output" != *"placed"* ]]
  cd "${repo}"
  git add --force .kit/rules/bash.md && git commit --quiet --message tracked
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 0 ]
  [[ "$output" == *"git rm --cached '.kit/rules/bash.md'"* ]]
  run "${kit_checkout}/bin/kit" check
  [ "$status" -eq 1 ]
  [[ "$output" == *".kit/rules/bash.md is tracked by git"* ]]
}

@test "kit hydrate refuses a repository that declares no kit version, and a reading verb does nothing there" {
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: kit install"* ]]
  printf 'x\n' > x.md
  run "${kit_checkout}/bin/kit" context x.md
  [ "$status" -eq 0 ]
  [ ! -e .kit ]
}

@test "update brings a path the new release delivers to its state, and removes one it retired, with .kit/.gitignore following" {
  "${kit_checkout}/bin/kit" install
  grep --quiet --line-regexp '/rules/just.md' .kit/.gitignore
  later-release "${repo}.later"
  run env KIT_BASE="${kit_checkout}" "${repo}.later/bin/kit" update
  [ "$status" -eq 0 ]
  [ ! -e .kit/rules/just.md ]
  run ! grep --quiet '/rules/just.md' .kit/.gitignore
  [ ! -w .kit/rules/bash.md ]
  run "${repo}.later/bin/kit" check
  [ "$status" -eq 0 ]
}

# ----------------------------------------------------------------------------------------------------------------------
