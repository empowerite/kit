#!/usr/bin/env bats
# lint.bats - kit lint runs the linters the delivered fragment declares, over the staged paths or every path
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  "${kit_checkout}/bin/kit" install >/dev/null
  printf 'extends:\n  - .kit/lefthook.yml\n' > lefthook.yml
  printf '#!/usr/bin/env bash\necho $x\n' > bad.sh
  printf '#!/usr/bin/env bash\necho "${x:-}"\n' > good.sh
}

teardown() {
  rm -rf "${repo}"
}

@test "lint refuses a staged bash file shellcheck faults, and passes a clean one" {
  git add bad.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 1 ]
  [[ "$output" == *"SC2086"* ]]
  git reset --quiet && git add good.sh
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
}

@test "lint --all-files reads every tracked path, where lint alone reads only the staged ones" {
  git -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false add bad.sh
  git -c user.name=t -c user.email=t@example.com -c commit.gpgsign=false commit --quiet --no-verify --message bad
  run "${kit_checkout}/bin/kit" lint
  [ "$status" -eq 0 ]
  run "${kit_checkout}/bin/kit" lint --all-files
  [ "$status" -eq 1 ]
  [[ "$output" == *"bad.sh"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
