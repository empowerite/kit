#!/usr/bin/env bats
# lint.bats - kit lint over given paths and over the staged set, and its reported skip
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  printf '#!/usr/bin/env bash\necho $x\n' > bad.sh
  printf '#!/usr/bin/env bash\necho "${x:-}"\n' > good.sh
}

teardown() {
  rm -rf "${repo}"
}

@test "lint refuses a bash file shellcheck faults, and passes a clean one" {
  run "${kit_checkout}/bin/kit" lint bad.sh
  [ "$status" -eq 1 ]
  [[ "$output" == *"SC2086"* ]]
  run "${kit_checkout}/bin/kit" lint good.sh
  [ "$status" -eq 0 ]
}

@test "lint with no path reads the staged set" {
  git add bad.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 1 ]
  git reset --quiet && git add good.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
}

@test "lint reports a skip when the linter is not on PATH" {
  # a PATH of one directory, carrying only what the verb itself runs, and no shellcheck
  mkdir only
  for tool in bash git env cat sed dirname
  do
    ln -s "$(command -v "${tool}")" "only/${tool}"
  done
  run env PATH="${PWD}/only" "${kit_checkout}/bin/kit" lint bad.sh
  [ "$status" -eq 0 ]
  [[ "$output" == *"NOT linted"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
