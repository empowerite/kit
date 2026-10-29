#!/usr/bin/env bats
# template.bats - a rule's template makes a new file, kit new file, and holds every file to its shape, kit lint
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

setup() {
  export kit_root NO_COLOR=1
  kit_root=$(cd "${BATS_TEST_DIRNAME}/../kit" && pwd)
  export KIT_HOME="${BATS_TEST_TMPDIR}/home" XDG_CACHE_HOME="${BATS_TEST_TMPDIR}/cache"
  mkdir -p "${KIT_HOME}" "${XDG_CACHE_HOME}"
  repo=$(mktemp -d)
  cd "${repo}"
  git init --quiet --initial-branch=main
  "${kit_root}/bin/kit" install 'style/*' >/dev/null 2>&1
}

teardown() {
  rm -rf "${repo}"
}

# in_template <command>… runs a command with the template module loaded, from the repository, as a verb would.
in_template() {
  bash -c 'source "${kit_root}/bash/1st.sh" && use template && "$@"' _ "$@"
}

# shape <path>… runs only the shape check of every rule governing the paths, as kit lint does, without the linters.
shape() {
  local path rule status=0
  for path in "$@"
  do
    while IFS= read -r rule
    do
      in_template template:check "${rule}" "${path}" || status=1
    done < <(bash -c 'source "${kit_root}/bash/1st.sh" && use context && context:rules "$1"' _ "${path}")
  done
  return "${status}"
}

@test "a new file is made at the root from its rule's first template, its holes filled, and made executable" {
  mkdir -p deep/er
  cd deep/er
  run "${kit_root}/bin/kit" new file bash/deploy.sh --purpose="push the build"
  [ "$status" -eq 0 ]
  [[ "$output" == *"made bash/deploy.sh from .kit/templates/bash/script"* ]]
  [[ "$output" == *".kit/rules/bash.md"* ]]
  cd "${repo}"
  [ -x bash/deploy.sh ]
  [ "$(sed -n 1,2p bash/deploy.sh)" = "$(printf '#!/usr/bin/env bash\n# deploy.sh - push the build')" ]
  [ "$(tail -c 1 bash/deploy.sh | od -An -c | tr -d ' ')" = '\n' ]
  [ "$(tail -n 1 bash/deploy.sh)" != "" ]
  grep --quiet 'bash/deploy.sh' "$(git rev-parse --git-common-dir)/kit/context.log"
}

@test "without a style package no template governs a file, and kit new file says where templates come from" {
  "${kit_root}/bin/kit" package remove style/bash >/dev/null 2>&1
  run "${kit_root}/bin/kit" new file bash/deploy.sh --purpose=x
  [ "$status" -eq 2 ]
  [[ "$output" == *"no template governs bash/deploy.sh: a style package delivers them"* ]]
  [ ! -e bash/deploy.sh ]
}

@test "every file kit new file makes has the shape of its own template" {
  "${kit_root}/bin/kit" new file bash/a.sh --purpose=a
  "${kit_root}/bin/kit" new file lib/b.sh --variant=module --purpose=b
  [ ! -x lib/b.sh ]
  "${kit_root}/bin/kit" new file justfile
  "${kit_root}/bin/kit" new file tools.just
  grep --quiet --fixed-strings 'export tools_justfile := source_file()' tools.just
  grep --quiet --fixed-strings '"${tools_justfile}"' tools.just
  "${kit_root}/bin/kit" new file docs/guide.md --title=Guide
  [ "$(cat docs/guide.md)" = "# Guide" ]
  "${kit_root}/bin/kit" new file conf/app.yml --purpose=app
  "${kit_root}/bin/kit" new file sub/.gitignore
  "${kit_root}/bin/kit" new file .github/workflows/check.yml --purpose=gate --workflow=check
  grep --quiet '^name: check$' .github/workflows/check.yml
  run shape bash/a.sh lib/b.sh justfile tools.just docs/guide.md conf/app.yml sub/.gitignore \
    .github/workflows/check.yml
  [ "$status" -eq 0 ]
}

@test "a hole left empty, a path that exists or leaves the root, and an unknown rule or variant are refused" {
  run "${kit_root}/bin/kit" new file docs/guide.md
  [ "$status" -eq 2 ]
  [[ "$output" == *"the document template of the markdown rule needs --title=<…>"* ]]
  [ ! -e docs/guide.md ]
  printf 'keep\n' > kept.sh
  run "${kit_root}/bin/kit" new file kept.sh --purpose=x
  [ "$status" -eq 1 ]
  [ "$(cat kept.sh)" = "keep" ]
  run "${kit_root}/bin/kit" new file ../out.sh --purpose=x
  [ "$status" -eq 2 ]
  run "${kit_root}/bin/kit" new file data.json
  [ "$status" -eq 2 ]
  [[ "$output" == *"no template governs data.json: a style package delivers them"* ]]
  run "${kit_root}/bin/kit" new file x.sh --variant=nope --purpose=x
  [[ "$output" == *"its templates are: script module"* ]]
  run "${kit_root}/bin/kit" new file x --rule=nope
  [[ "$output" == *"there is no rule nope; the rules are: bash gitignore json just markdown workflows yaml"* ]]
  run "${kit_root}/bin/kit" new file bin/tool --rule=bash --purpose="no extension"
  [ "$status" -eq 0 ]
  [ "$(head -n 1 bin/tool)" = "#!/usr/bin/env bash" ]
}

@test "the most specific rule's template is chosen, and a path-limited variant is the default only on its path" {
  run in_template template:rules .github/workflows/check.yml
  [ "${lines[0]}" = ".kit/style/workflows.md" ]
  [ "${lines[1]}" = ".kit/style/yaml.md" ]
  run in_template template:applies .kit/style/just.md justfile
  [ "$(cut -d $'\x1f' -f 1 <<<"$output" | paste -s -d ' ' -)" = "root module" ]
  run in_template template:applies .kit/style/just.md tools.just
  [ "$(cut -d $'\x1f' -f 1 <<<"$output")" = "module" ]
}

@test "the delivered templates impose no house style: a script passes with or without a line of dashes" {
  printf '#!/usr/bin/env bash\n# plain.sh - no dashes\n\nset -o errexit -o nounset -o pipefail\necho hi\n' > plain.sh
  printf '#!/usr/bin/env bash\n# dashed.sh - dashes\n# %s\n\nset -o errexit -o nounset -o pipefail\necho hi\n# %s\n' \
    "$(printf -- '-%.0s' {1..118})" "$(printf -- '-%.0s' {1..118})" > dashed.sh
  shape plain.sh dashed.sh
}

@test "a file off its template's shape is named with the line it leaves, and a hole named twice must agree" {
  "${kit_root}/bin/kit" new file bash/a.sh --purpose=a
  sed -i.bak '/^set -o/d' bash/a.sh
  run shape bash/a.sh
  [ "$status" -ne 0 ]
  [[ "$output" == *"bash/a.sh does not have the shape of the bash template (script, module)"* ]]
  [[ "$output" == *".kit/templates/bash/script line 4: set -o errexit -o nounset -o pipefail"* ]]
  "${kit_root}/bin/kit" new file tools.just
  sed -i.bak 's/"${tools_justfile}"/"${other_justfile}"/' tools.just
  run shape tools.just
  [ "$status" -ne 0 ]
}

@test "a hole that is a line matches any run of lines, none included, and one within a line any text on it" {
  printf 'head «title» tail\n«body»\nend\n' > "${BATS_TEST_TMPDIR}/t"
  printf 'head anything at all tail\nend\n' > none
  printf 'head x tail\none\ntwo\nend\n' > two
  printf 'head x tail\nend\nmore\n' > after
  in_template template:matches "${BATS_TEST_TMPDIR}/t" none
  in_template template:matches "${BATS_TEST_TMPDIR}/t" two
  run ! in_template template:matches "${BATS_TEST_TMPDIR}/t" after
  printf 'a.b «x»\n' > "${BATS_TEST_TMPDIR}/dot"
  printf 'aXb y\n' > dot
  run ! in_template template:matches "${BATS_TEST_TMPDIR}/dot" dot
}

@test "kit lint holds a governed file to its shape, and skips templates, regions, and generated paths" {
  printf '#!/usr/bin/env bash\necho hi\n' > loose.sh
  mkdir -p locks
  printf 'lockfileVersion: 9\n' > locks/a.yaml
  printf '/locks/** linguist-generated\n' > .gitattributes
  run "${kit_root}/bin/kit" lint loose.sh locks/a.yaml .kit/templates/bash/script
  [[ "$output" == *"loose.sh does not have the shape of the bash template"* ]]
  [[ "$output" != *"locks/a.yaml does not"* ]]
  [[ "$output" != *"templates/bash/script does not"* ]]
  [[ "$output" == *".kit/style/yaml.md: the shape of its templates over 0 path(s)"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
