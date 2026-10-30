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
    # jq --run-tests then runs jq's own library tests, whose lines follow its summary and say nothing of the
    # module, so the report stops at the summary.
    jq -L "${kit_root}/modules/jq" --run-tests "${BATS_TEST_DIRNAME}/jq/${name}.test" > "${BATS_TEST_TMPDIR}/out" \
      || status=1
    sed '/tests passed/q' "${BATS_TEST_TMPDIR}/out"
  done
  return "${status}"
}

# ----------------------------------------------------------------------------------------------------------------------
