#!/usr/bin/env bats
# forge.bats - the forge is the setting, else the one origin's host names, and an unknown one is refused by name
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  fresh-repo
}

# provider prints the provider the seam picks here.
provider() {
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use forge && forge:provider'
}

@test "an origin at github.com names the github provider, and one at gitlab.com the gitlab one, by https or ssh" {
  git remote add origin https://github.com/empowerite/kit.git
  run provider
  [ "$status" -eq 0 ]
  [ "$output" = "github" ]
  git remote set-url origin git@github.com:empowerite/kit.git
  run provider
  [ "$output" = "github" ]
  git remote set-url origin git@gitlab.com:empowerite/kit.git
  run provider
  [ "$output" = "gitlab" ]
}

@test "an origin elsewhere, or none, is refused naming the setting and the forges the kit knows" {
  git remote add origin https://codeberg.org/empowerite/kit.git
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"set forge in kit.yml; origin's host 'codeberg.org' names no forge the kit knows"* ]]
  [[ "$output" == *"(github gitlab)"* ]]
  git remote remove origin
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"origin's host 'none'"* ]]
}

@test "the forge setting wins, and an unknown one is refused by name" {
  git remote add origin https://codeberg.org/empowerite/kit.git
  mkdir .kit && printf 'forge: github\n' > kit.yml
  run provider
  [ "$status" -eq 0 ]
  [ "$output" = "github" ]
  printf 'forge: sourcehut\n' > kit.yml
  run provider
  [ "$status" -eq 1 ]
  [[ "$output" == *"'sourcehut' is not a forge the kit knows; the forges are: github gitlab"* ]]
}

@test "the core verbs need no forge and no gh" {
  mkdir -p "${BATS_TEST_TMPDIR}/shims" && printf '#!/usr/bin/env bash\nexit 127\n' > "${BATS_TEST_TMPDIR}/shims/gh"
  chmod +x "${BATS_TEST_TMPDIR}/shims/gh"
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" install agents/claude
  [ "$status" -eq 0 ]
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" check
  [ "$status" -eq 0 ]
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" context justfile
  [ "$status" -eq 0 ]
  run env PATH="${BATS_TEST_TMPDIR}/shims:${PATH}" "${kit_root}/bin/kit" status --no-fetch
  [ "$status" -eq 0 ]
  [[ "$output" == *"no origin: nothing fetched, no pull request shown"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
