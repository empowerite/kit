#!/usr/bin/env bats
# kind.bats - kit kind prints the kinds a file is of, from one matcher compiled from every kind and kept until they change
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

setup() {
  export kit_checkout NO_COLOR=1
  kit_checkout=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo="${BATS_TEST_TMPDIR}/repo"
  git init --quiet --initial-branch=main "${repo}"
  cd "${repo}"
  "${kit_checkout}/bin/kit" install >/dev/null 2>&1
}

@test "one path prints its kinds one per line, by name, by first line, and through the kinds made of others" {
  run "${kit_checkout}/bin/kit" kind README.md
  [ "$status" -eq 0 ]
  [ "$output" = $'any\nmarkdown\ntext' ]
  mkdir bin && printf '#!/usr/bin/env bash\necho\n' > bin/tool
  run "${kit_checkout}/bin/kit" kind bin/tool
  [ "$output" = $'any\nbash\ntext' ]
  run "${kit_checkout}/bin/kit" kind bin/other
  [ "$output" = 'any' ]
  run "${kit_checkout}/bin/kit" kind bin
  [ "$status" -eq 0 ]
  [ "$output" = 'any' ]
}

@test "several paths print a line each, --json an object each, and a relative path starts where the command was typed" {
  mkdir -p .github/workflows docs
  run "${kit_checkout}/bin/kit" kind justfile .github/workflows/ci.yml image.png
  [ "${lines[0]}" = $'justfile\tany just text' ]
  [ "${lines[1]}" = $'.github/workflows/ci.yml\tany text workflows yaml' ]
  [ "${lines[2]}" = $'image.png\tany' ]
  run "${kit_checkout}/bin/kit" kind --json notes.txt a.json
  [ "$(jq -sc . <<<"$output")" = '[{"path":"notes.txt","kinds":["any","text"]},{"path":"a.json","kinds":["any","json","text"]}]' ]
  cd docs
  run "${kit_checkout}/bin/kit" kind --json page.md
  [ "$(jq -c .path <<<"$output")" = '"docs/page.md"' ]
}

@test "the kinds are compiled once into the git directory, and compiled afresh when a kind changes" {
  run "${kit_checkout}/bin/kit" kind a.toml
  [ "$output" = 'any' ]
  [ "$(find .git/kit -name 'kinds-*.awk' | wc -l)" -eq 1 ]
  mkdir -p .kit/packages/toml/files/repo/.kit/kinds
  printf 'description: "x"\ndepends:\n  - kit\n' > .kit/packages/toml/package.yml
  printf 'name: toml\npaths:\n  - "**/*.toml"\n' > .kit/packages/toml/files/repo/.kit/kinds/toml.yml
  "${kit_checkout}/bin/kit" package add toml >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" kind a.toml
  [ "$output" = $'any\ntoml' ]
  [ "$(find .git/kit -name 'kinds-*.awk' | wc -l)" -eq 1 ]
  printf 'name: toml\npaths:\n  - "**/*.tml"\n' > .kit/packages/toml/files/repo/.kit/kinds/toml.yml
  run "${kit_checkout}/bin/kit" kind a.toml
  [ "$output" = 'any' ]
}

@test "kit kind needs a path" {
  run "${kit_checkout}/bin/kit" kind
  [ "$status" -eq 2 ]
  [[ "$output" == *"run: kit help kind"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
