#!/usr/bin/env bats
# hooks.bats - the desk guard: pre-rebase refuses the default branch, pre-commit refuses a lint finding
# ----------------------------------------------------------------------------------------------------------------------

setup() {
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  git config user.name t && git config user.email t@example.com && git config commit.gpgsign false
  git config core.hooksPath "${kit_checkout}/.kit/hooks"
  printf 'one\n' > a && git add a && git commit --quiet --message one
}

teardown() {
  rm -rf "${repo}"
}

@test "pre-rebase refuses a rebase of main that would rewrite a commit, and allows one in a branch" {
  git switch --quiet --create feature
  printf 'two\n' > b && git add b && git commit --quiet --message two
  git switch --quiet main
  printf 'three\n' > c && git add c && git commit --quiet --message three
  run git rebase feature
  [ "$status" -eq 1 ]
  [[ "$output" == *"refusing to rebase 'main'"* ]]
  git switch --quiet feature
  run git rebase main
  [ "$status" -eq 0 ]
}

@test "pre-rebase lets main fast-forward" {
  git switch --quiet --create feature
  printf 'two\n' > b && git add b && git commit --quiet --message two
  git switch --quiet main
  run git rebase feature
  [ "$status" -eq 0 ]
}

@test "pre-commit refuses a bash file shellcheck faults, and passes a clean one" {
  printf '#!/usr/bin/env bash\necho $x\n' > bad.sh
  git add bad.sh
  run git commit --quiet --message bad
  [ "$status" -eq 1 ]
  [[ "$output" == *"SC2086"* ]]
  [[ "$output" == *"refusing the commit"* ]]
  printf '#!/usr/bin/env bash\necho "${x:-}"\n' > bad.sh
  git add bad.sh
  run git commit --quiet --message good
  [ "$status" -eq 0 ]
}

@test "pre-commit reports a skip when the linter is not on PATH" {
  printf '#!/usr/bin/env bash\necho $x\n' > bad.sh
  git add bad.sh
  # a PATH without any directory that carries a shellcheck
  without=""
  while IFS= read -r dir
  do
    [[ -n "${dir}" && ! -x "${dir}/shellcheck" ]] && without="${without:+${without}:}${dir}"
  done < <(printf '%s' "${PATH}" | tr ':' '\n')
  run env PATH="${without}" git commit --quiet --message bad
  [ "$status" -eq 0 ]
  [[ "$output" == *"NOT linted"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
