#!/usr/bin/env bats
# ci.bats - kit ci runs kit linked, kit policy, the gate, and kit check, in that order, and reports every one
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

# ci-world makes the repository every test starts from: origin on GitHub, the kit installed, and gh failing.
ci-world() {
  mkdir shims && printf '#!/usr/bin/env bash\nexit 1\n' > shims/gh && chmod +x shims/gh
  git init --quiet --initial-branch=main repo && cd repo \
    && git remote add origin https://github.com/empowerite/example.git \
    && PATH="${PWD}/../shims:${PATH}" "${kit_checkout}/bin/kit" install
}

setup_file() {
  world-make installed ci-world
}

setup() {
  PATH=$(shims gh)
  world-copy-repo installed
  export GITHUB_STEP_SUMMARY="${BATS_TEST_TMPDIR}/summary.md"
}

@test "every check runs and the refusals are counted" {
  printf 'ci:\n  echo gate ran\n' > justfile
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" ci
  [ "$status" -eq 1 ]
  [[ "$output" == *"gate ran"* ]]
  [[ "$output" == *"every delivered file matches"* ]]
  # gh fails, so kit linked cannot tell which pull request this checkout is working on, and refuses, naming the read
  [[ "$output" == *"run: cd "*" && gh auth status (the open pull request on main could not be read)"* ]]
  [[ "$output" == *"the patterns policy allows"* ]]
  [[ "$output" == *"1 of 4 checks refused"* ]]
  # a stray file under a delivered directory: check reports it, and no verb heals it
  printf 'rogue\n' > .kit/rules/rogue.md
  rm justfile
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" ci
  [ "$status" -eq 1 ]
  [[ "$output" == *"3 of 4 checks refused"* ]]
}

@test "the issue link and the policy are judged before the gate and kit check run any command of the branch" {
  printf 'ci:\n  echo gate ran\n' > justfile
  run env -u GITHUB_EVENT_PATH "${kit_checkout}/bin/kit" ci
  [ "$status" -eq 1 ]
  in-order "$output" "── kit linked ──" "── kit policy ──" "── kit gate ──" "gate ran" \
    "── kit check ──"
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

@test "the CI job stops after the minutes the ci_timeout_minutes answer gives, 30 by default" {
  fresh-repo other >/dev/null
  "${kit_checkout}/bin/kit" install github/ci >/dev/null
  workflow=.github/workflows/kit-ci.yml
  grep --quiet "^    timeout-minutes: \${{ fromJSON('30') }}$" "${workflow}"
  fresh-repo third >/dev/null
  mkdir .kit && printf 'answers:\n  ci_timeout_minutes: "12"\n' > kit.yml
  "${kit_checkout}/bin/kit" install github/ci >/dev/null
  grep --quiet "^    timeout-minutes: \${{ fromJSON('12') }}$" "${workflow}"
}

@test "the CI workflow installs with kit tools, and caches the kit toolchain's tools by tools.lock" {
  fresh-repo other >/dev/null
  "${kit_checkout}/bin/kit" install github/ci >/dev/null
  workflow=.github/workflows/kit-ci.yml
  grep --quiet '^          kit tools install && kit tools path >> "${GITHUB_PATH}"$' "${workflow}"
  # a step runs by the toolchain the answer names, never by which files are here
  run ! grep --quiet 'if: .*hashFiles' "${workflow}"
  grep --quiet "^        if: \${{ env.KIT_TOOLCHAIN == 'kit' }}$" "${workflow}"
  grep --quiet "^          key: kit-tools-\${{ runner.os }}-\${{ runner.arch }}-\${{ hashFiles('tools.lock') }}$" "${workflow}"
}

@test "ambient, the toolchain step fails naming a tool the rules need and the runner lacks" {
  fresh-repo other >/dev/null
  "${kit_checkout}/bin/kit" install github/ci >/dev/null
  printf -- '---\nlint: kit-test-absent-linter --strict\n---\n# Absent\n' > .kit/rules/absent.md
  # the step's own command lines, as the workflow runs them
  step=$(sed -n '/- name: toolchain$/,/- name: kit-ci$/p' .github/workflows/kit-ci.yml)
  step=$(sed -n '/run: |$/,$p' <<<"${step}" | sed '1d;$d')
  export GITHUB_PATH="${BATS_TEST_TMPDIR}/github-path"
  run env PATH="${kit_checkout}/bin:${PATH}" KIT_TOOLCHAIN=ambient bash -c "${step}"
  [ "$status" -eq 1 ]
  [[ "$output" == *"kit-test-absent-linter"* ]]
}
