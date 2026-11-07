#!/usr/bin/env bats
# kinds.bats - kit kinds lists the kinds, or prints the kinds a path is of, from one matcher kept until a kind changes
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  world-repo installed
}

setup() {
  world-copy-repo installed
}

@test "one path prints its kinds one per line, by name, by first line, and through the kinds made of others" {
  run "${kit_checkout}/bin/kit" kinds README.md
  [ "$status" -eq 0 ]
  [ "$output" = $'any\nmarkdown\ntext' ]
  mkdir bin && printf '#!/usr/bin/env bash\necho\n' > bin/tool
  run "${kit_checkout}/bin/kit" kinds bin/tool
  [ "$output" = $'any\nbash\ntext' ]
  run "${kit_checkout}/bin/kit" kinds bin/other
  [ "$output" = 'any' ]
  run "${kit_checkout}/bin/kit" kinds bin
  [ "$status" -eq 0 ]
  [ "$output" = 'any' ]
}

@test "several paths print a line each, --json an object each, and a relative path starts where the command was typed" {
  mkdir -p .github/workflows docs
  run "${kit_checkout}/bin/kit" kinds justfile .github/workflows/ci.yml image.png
  [ "${lines[0]}" = $'justfile\tany just text' ]
  [ "${lines[1]}" = $'.github/workflows/ci.yml\tany text workflows yaml' ]
  [ "${lines[2]}" = $'image.png\tany' ]
  run "${kit_checkout}/bin/kit" kinds --json notes.txt a.json
  [ "$(jq -sc . <<<"$output")" = '[{"path":"notes.txt","kinds":["any","text"]},{"path":"a.json","kinds":["any","json","text"]}]' ]
  cd docs
  run "${kit_checkout}/bin/kit" kinds --json page.md
  [ "$(jq -c .path <<<"$output")" = '"docs/page.md"' ]
}

@test "the kinds are compiled once into the git directory, and compiled afresh when a kind changes" {
  run "${kit_checkout}/bin/kit" kinds a.toml
  [ "$output" = 'any' ]
  [ "$(find .git/kit -name 'kinds-*.awk' | wc -l)" -eq 1 ]
  mkdir -p .kit/packages/toml/files/repo/.kit/kinds
  printf 'description: "x"\ndepends:\n  - kit\n' > .kit/packages/toml/package.yml
  printf 'name: toml\npaths:\n  - "**/*.toml"\n' > .kit/packages/toml/files/repo/.kit/kinds/toml.yml
  "${kit_checkout}/bin/kit" packages add toml >/dev/null 2>&1
  run "${kit_checkout}/bin/kit" kinds a.toml
  [ "$output" = $'any\ntoml' ]
  [ "$(find .git/kit -name 'kinds-*.awk' | wc -l)" -eq 1 ]
  printf 'name: toml\npaths:\n  - "**/*.tml"\n' > .kit/packages/toml/files/repo/.kit/kinds/toml.yml
  run "${kit_checkout}/bin/kit" kinds a.toml
  [ "$output" = 'any' ]
}

@test "a word that names a kind filters the listing, any other word is a path, and the two do not mix" {
  run "${kit_checkout}/bin/kit" kinds bash
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 1 ]
  [[ "${lines[0]}" == "bash "*".kit/kinds/bash.yml"* ]]
  run "${kit_checkout}/bin/kit" kinds --json bash json
  [ "$(jq -r .name <<<"$output" | paste -s -d ' ' -)" = "bash json" ]
  # a word shaped like a kind's name that names no kind is a path, and so is one of any other shape
  run "${kit_checkout}/bin/kit" kinds nonesuch
  [ "$status" -eq 0 ]
  [ "$output" = 'any' ]
  run "${kit_checkout}/bin/kit" kinds bash README.md
  [ "$status" -eq 2 ]
  [[ "$output" == *"kit kinds takes kind names or paths, not both: kinds bash; paths README.md; run: kit help kinds"* ]]
}

@test "a repository with no kinds says so in one dim line, lists nothing as JSON, and finds no kind for a path" {
  fresh-repo bare >/dev/null
  run "${kit_checkout}/bin/kit" kinds
  [ "$status" -eq 0 ]
  [ "$output" = "• no kinds under .kit/kinds/; kit install brings the kit package's" ]
  run "${kit_checkout}/bin/kit" kinds --json
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_checkout}/bin/kit" kinds foo.sh
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run "${kit_checkout}/bin/kit" kinds --json foo.sh bar.md
  [ "$status" -eq 0 ]
  [ "$(jq -sc 'map(.kinds)' <<<"$output")" = '[[],[]]' ]
}

# ----------------------------------------------------------------------------------------------------------------------
