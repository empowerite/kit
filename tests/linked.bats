#!/usr/bin/env bats
# linked.bats - kit linked names the pull request it needs when none can be found; the forge fakes test the rest
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  fresh-repo
  git remote add origin https://github.com/empowerite/example.git
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
}

@test "linked names the pull request it needs when none can be found" {
  # gh answers that the branch has no pull request: its filter of an empty list prints nothing
  mkdir -p "${BATS_TEST_TMPDIR}/bin"
  printf '#!/usr/bin/env bash\nexit 0\n' > "${BATS_TEST_TMPDIR}/bin/gh" && chmod +x "${BATS_TEST_TMPDIR}/bin/gh"
  PATH="${BATS_TEST_TMPDIR}/bin:${PATH}" run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" linked
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit linked <number>"* ]]
}

@test "linked fails, naming the read, when the forge cannot say which pull request this checkout is working on" {
  PATH=$(shims gh) run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" linked
  [ "$status" -eq 1 ]
  [[ "$output" == *"run: gh auth status (the open pull request on main could not be read)"* ]]
  [[ "$output" != *"kit linked <number>"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
