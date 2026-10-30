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
  cp -R "${kit_checkout}/bin" "${kit_checkout}/modules" "${kit_checkout}/packages" "$1/"
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
  [ ! -w .kit/.gitignore ]
  [ ! -w .kit/manifest ]
  [ -f .kit/README.md ]
  [ ! -w .kit/README.md ]
  [ "$(git status --porcelain --untracked-files=all | grep ' \.kit/')" = "?? .kit/settings.yml" ]
  run bash -c 'echo edited > .kit/rules/bash.md'
  [ "$status" -ne 0 ]
  [ "$(head -n 1 .kit/rules/bash.md)" != "edited" ]
  run bash -c 'echo /mine >> .kit/.gitignore'
  [ "$status" -ne 0 ]
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

@test "a new worktree is hydrated by the kit that runs there: copies from a release outside it, links to one it carries" {
  "${kit_checkout}/bin/kit" install
  git add --all && git commit --quiet --message init
  git worktree add --quiet wt-copies -b copies
  [ ! -e wt-copies/.kit/1st.md ]
  run bash -c 'source "$0/modules/bash/1st.sh" && use issue && issue:hydrate "$1"' "${kit_checkout}" wt-copies
  [ "$status" -eq 0 ]
  [ -f wt-copies/.kit/1st.md ]
  [ ! -L wt-copies/.kit/1st.md ]
  [ ! -w wt-copies/.kit/1st.md ]
  # a repository that carries its release, as the kit's own does, gets links into the worktree's own copy of it
  cp -R "${kit_checkout}" kit
  git add --all && git commit --quiet --message release
  git worktree add --quiet wt-links -b links
  run bash -c 'source "$0/modules/bash/1st.sh" && use issue && issue:hydrate "$1"' "${kit_checkout}" wt-links
  [ "$status" -eq 0 ]
  [ -L wt-links/.kit/rules/bash.md ]
  [ "$(readlink wt-links/.kit/rules/bash.md)" = "../../kit/packages/kit/files/repo/.kit/rules/bash.md" ]
}

@test "where the release is inside the worktree, a reading verb turns a readonly copy back into a link" {
  cp -R "${kit_checkout}" kit
  kit/bin/kit install >/dev/null 2>&1
  [ -L .kit/rules/bash.md ]
  rm .kit/rules/bash.md && cp kit/packages/kit/files/repo/.kit/rules/bash.md .kit/rules/bash.md && chmod a-w .kit/rules/bash.md
  run kit/bin/kit context AGENTS.md
  [ "$status" -eq 0 ]
  [ -L .kit/rules/bash.md ]
}

@test "a declaration edited by hand is noticed by the next verb, which places what it added" {
  "${kit_checkout}/bin/kit" install
  [ ! -e .kit/lefthook.yml ]
  touch -t 202001010000 .kit/.gitignore
  printf '  - hooks/lefthook\n' >> .kit/settings.yml
  printf 'x\n' > x.md
  run "${kit_checkout}/bin/kit" context x.md
  [ "$status" -eq 0 ]
  [ -f .kit/lefthook.yml ]
  grep --quiet --line-regexp '/lefthook.yml' .kit/.gitignore
  [ .kit/.gitignore -nt .kit/settings.yml ]
}

@test "kit hydrate removes what the declaration dropped and records what it added, so a pulled declaration is enough" {
  "${kit_checkout}/bin/kit" install style/yaml
  [ -f .kit/styles/yaml.md ]
  sed -i.bak '/^  - style\/yaml$/d' .kit/settings.yml && rm .kit/settings.yml.bak
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 0 ]
  [ ! -e .kit/styles/yaml.md ]
  [ ! -e .kit/templates/yaml ]
  run ! grep --quiet 'styles/yaml.md' .kit/.gitignore
  run ! grep --quiet 'styles/yaml.md' .kit/manifest
  printf '  - style/json\n' >> .kit/settings.yml
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 0 ]
  [ -f .kit/styles/json.md ]
  grep --quiet '  .kit/styles/json.md$' .kit/manifest
  [ "$(git status --porcelain --untracked-files=all | grep ' \.kit/')" = "?? .kit/settings.yml" ]
}

@test "a readonly .kit-new left by an interrupted run does not block the next hydrate" {
  "${kit_checkout}/bin/kit" install
  cp .kit/rules/bash.md .kit/rules/bash.md.kit-new && chmod a-w .kit/rules/bash.md.kit-new
  chmod u+w .kit/rules/bash.md && printf 'edited\n' >> .kit/rules/bash.md
  run "${kit_checkout}/bin/kit" hydrate
  [ "$status" -eq 0 ]
  run ! grep --quiet '^edited$' .kit/rules/bash.md
  [ ! -e .kit/rules/bash.md.kit-new ]
}

@test "hydrate says which kit placed the files, and warns when it is not the declared one" {
  "${kit_checkout}/bin/kit" install
  later-release "${repo}.later"
  run "${repo}.later/bin/kit" hydrate
  [ "$status" -eq 0 ]
  [[ "$output" == *"declares kit unreleased, and this is kit 9.9.9"* ]]
  [[ "$output" == *"run: kit update"* ]]
  [[ "$output" == *"kit 9.9.9 hydrated"* ]]
}

@test "a release under a nested checkout counts as outside the worktree, so nothing links into what kit tidy removes" {
  mkdir -p worktrees/x && printf 'gitdir: elsewhere\n' > worktrees/x/.git
  cp -R "${kit_checkout}" worktrees/x/kit
  run worktrees/x/kit/bin/kit install
  [ "$status" -eq 0 ]
  [ -f .kit/rules/bash.md ]
  [ ! -L .kit/rules/bash.md ]
  [ ! -w .kit/rules/bash.md ]
}

@test "a worktree's own kit is trusted only where the repository is the kit: a release declared, a kit/bin/kit of its own is not run" {
  "${kit_checkout}/bin/kit" install
  git add --all && git commit --quiet --message init
  git worktree add --quiet wt -b wt
  sed -i.bak 's/^version: unreleased$/version: 9.9.9/' wt/.kit/settings.yml && rm wt/.kit/settings.yml.bak
  mkdir -p wt/kit/bin && printf '#!/usr/bin/env bash\nexit 7\n' > wt/kit/bin/kit && chmod +x wt/kit/bin/kit
  run bash -c 'source "$0/modules/bash/1st.sh" && use issue && issue:hydrate "$1"' "${kit_checkout}" wt
  [ "$status" -eq 0 ]
  [ -f wt/.kit/1st.md ]
  [ ! -L wt/.kit/1st.md ]
}

@test "kit hydrate --quick says nothing and starts nothing when everything is in place, and does the whole job otherwise" {
  "${kit_checkout}/bin/kit" install
  run "${kit_checkout}/bin/kit" hydrate --quick
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  rm .kit/rules/bash.md
  run "${kit_checkout}/bin/kit" hydrate --quick
  [ "$status" -eq 0 ]
  [[ "$output" == *".kit/rules/bash.md: placed"* ]]
  [[ "$output" == *"hydrated .kit/"* ]]
  [ -f .kit/rules/bash.md ]
}

# ----------------------------------------------------------------------------------------------------------------------
