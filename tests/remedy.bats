#!/usr/bin/env bats
# remedy.bats - a remedy kit prints, the command after run:, runs from anywhere: cd and its folder go in front of it
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup() {
  fresh-repo
}

@test "error and warn put cd and the folder kit runs in in front of each remedy, and keep one that has its own" {
  run --separate-stderr kit-bash 'error "x is missing; run: kit hydrate (it holds the kit)"'
  [ "$stderr" = "❌ x is missing; run: cd ${repo} && kit hydrate (it holds the kit)" ]
  run --separate-stderr kit-bash 'warn "move it aside, then run: kit hydrate; or run: cd /srv && kit update"'
  [ "$stderr" = "⚠️ move it aside, then run: cd ${repo} && kit hydrate; or run: cd /srv && kit update" ]
  run --separate-stderr kit-bash 'error "nothing to run here"'
  [ "$stderr" = "❌ nothing to run here" ]
  # in CI, the annotation says what the line says
  run --separate-stderr env GITHUB_ACTIONS=true bash -c "cd '${repo}' && source '${kit_root}/modules/bash/1st.sh' \
    && error 'run: kit hydrate'"
  [[ "$stderr" == *"::error::run: cd ${repo} && kit hydrate" ]]
}

@test "the folder is written as a shell reads it back: ~ for the home folder, quoted where it needs it" {
  mkdir -p "${HOME}/src/app" "${HOME}/it's here" "${BATS_TEST_TMPDIR}/my app"
  run --separate-stderr bash -c "cd ~ && source '${kit_root}/modules/bash/1st.sh' && error 'run: kit help'"
  [ "$stderr" = "❌ run: cd ~ && kit help" ]
  cd "${HOME}/src/app"
  run --separate-stderr kit-bash 'error "run: kit hydrate"'
  [ "$stderr" = "❌ run: cd ~/src/app && kit hydrate" ]
  cd "${HOME}/it's here"
  run --separate-stderr kit-bash 'error "run: kit hydrate"'
  [ "$stderr" = "❌ run: cd ~/'it'\\''s here' && kit hydrate" ]
  # the cd, read back by a shell, lands where kit ran
  local remedy=${stderr#❌ run: }
  run bash -c "${remedy% && kit hydrate} && pwd"
  [ "$output" = "${HOME}/it's here" ]
  cd "${BATS_TEST_TMPDIR}/my app"
  run --separate-stderr kit-bash 'error "run: kit hydrate"'
  [ "$stderr" = "❌ run: cd '${BATS_TEST_TMPDIR}/my app' && kit hydrate" ]
  remedy=${stderr#❌ run: }
  run bash -c "${remedy% && kit hydrate} && pwd"
  [ "$output" = "${BATS_TEST_TMPDIR}/my app" ]
}
