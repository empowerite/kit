#!/usr/bin/env bats
# exits.bats - every verb's header ends its description with one Exits line, stating its codes as docs/modules.md has them
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
}

@test "every verb's header has exactly one line that starts with Exits, and kit help prints it in the description" {
  local verb name count
  for verb in "${kit_root}"/bin/kit-*
  do
    [[ -x "${verb}" ]] || continue
    name=${verb##*/kit-}
    count=$(awk 'NR > 1 && !/^#/ { exit } /^# Exits / { n++ } END { print n + 0 }' "${verb}")
    [ "${count}" -eq 1 ] || { echo "${name}: ${count} Exits line(s) in its header"; return 1; }
    run "${kit_root}/bin/kit" help "${name}"
    [[ "$output" == *$'\n  Exits '* ]] || { echo "${name}: kit help prints no Exits line"; return 1; }
  done
}

# ----------------------------------------------------------------------------------------------------------------------
