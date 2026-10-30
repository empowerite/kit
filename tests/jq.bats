#!/usr/bin/env bats
# jq.bats - every jq module passes its own tests, tests/jq/<module>.test, run by jq --run-tests
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
}

@test "every jq module has a test file, and passes it" {
  local module name status=0
  for module in "${kit_root}"/modules/jq/*.jq
  do
    name=$(basename "${module}" .jq)
    if [[ ! -f "${BATS_TEST_DIRNAME}/jq/${name}.test" ]]
    then
      echo "modules/jq/${name}.jq has no tests/jq/${name}.test"
      status=1
      continue
    fi
    jq -L "${kit_root}/modules/jq" --run-tests "${BATS_TEST_DIRNAME}/jq/${name}.test" || status=1
  done
  return "${status}"
}

# ----------------------------------------------------------------------------------------------------------------------
