#!/usr/bin/env bats
# ci.bats - kit ci runs the gate, kit check, kit linked, and kit policy, and reports every one
# ----------------------------------------------------------------------------------------------------------------------

load helper

setup() {
  PATH=$(shims gh)
  fresh-repo
  git remote add origin https://github.com/empowerite/example.git
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
  "${kit_checkout}/bin/kit" install >/dev/null
}

@test "every check runs and the refusals are counted" {
  printf 'ci:\n  echo gate ran\n' > justfile
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" ci
  [ "$status" -eq 1 ]
  [[ "$output" == *"gate ran"* ]]
  [[ "$output" == *"every delivered file matches"* ]]
  [[ "$output" == *"kit linked <number>"* ]]
  [[ "$output" == *"the patterns policy allows"* ]]
  [[ "$output" == *"1 of 4 checks refused"* ]]
  # a stray file under a delivered directory: check reports it, and no verb heals it
  printf 'rogue\n' > .kit/rules/rogue.md
  rm justfile
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" ci
  [ "$status" -eq 1 ]
  [[ "$output" == *"3 of 4 checks refused"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------

@test "the CI workflow names the toolchain the ci_toolchain answer gives, ambient by default" {
  fresh-repo other >/dev/null
  "${kit_checkout}/bin/kit" install github/ci >/dev/null
  workflow=.github/workflows/kit-ci.yml
  grep --quiet '^      KIT_TOOLCHAIN: "ambient"$' "${workflow}"
  # actionlint refuses a guard that compares two constants, so each guard reads the env the answer sets
  for toolchain in mise nix devcontainer
  do
    grep --quiet "if: \${{ env.KIT_TOOLCHAIN == '${toolchain}' }}" "${workflow}"
  done
  fresh-repo third >/dev/null
  mkdir .kit && printf 'answers:\n  ci_toolchain: nix\n' > kit.yml
  "${kit_checkout}/bin/kit" install github/ci >/dev/null
  grep --quiet '^      KIT_TOOLCHAIN: "nix"$' "${workflow}"
  # each bootstrap step is pinned: nix by commit SHA, the devcontainer CLI by exact version
  grep --quiet 'uses: DeterminateSystems/nix-installer-action@[0-9a-f]\{40\}  # v[0-9.]*$' "${workflow}"
  grep --quiet 'npm install --global @devcontainers/cli@[0-9][0-9.]*$' "${workflow}"
}

@test "the CI workflow installs with kit tools" {
  fresh-repo other >/dev/null
  "${kit_checkout}/bin/kit" install github/ci >/dev/null
  grep --quiet 'run: kit tools install && kit tools path >> "${GITHUB_PATH}"$' .github/workflows/kit-ci.yml
  ! grep --quiet "hashFiles" .github/workflows/kit-ci.yml
}
