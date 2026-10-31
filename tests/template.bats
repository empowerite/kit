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
  bash -c 'source "${kit_root}/modules/bash/1st.sh" && use template && "$@"' _ "$@"
}

# shape <path>… runs only the shape check of every rule governing the paths, as kit lint does, without the linters.
shape() {
  local path rule status=0
  for path in "$@"
  do
    while IFS= read -r rule
    do
      in_template template:check "${rule}" "${path}" || status=1
    done < <(bash -c 'source "${kit_root}/modules/bash/1st.sh" && use context && context:rules "$1"' _ "${path}")
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
  [ "$status" -eq 2 ]
  [[ "$output" == *"--variant 'nope' is not one of the choices; run: kit help new"* ]]
  run "${kit_root}/bin/kit" new file x --rule=nope
  [ "$status" -eq 2 ]
  [[ "$output" == *"--rule 'nope' is not one of the choices; run: kit help new"* ]]
  run "${kit_root}/bin/kit" new file bin/tool --rule=bash --purpose="no extension"
  [ "$status" -eq 0 ]
  [ "$(head -n 1 bin/tool)" = "#!/usr/bin/env bash" ]
}

@test "the most specific rule's template is chosen, and a path-limited variant is the default only on its path" {
  run in_template template:rules .github/workflows/check.yml
  [ "${lines[0]}" = ".kit/styles/workflows.md" ]
  [ "${lines[1]}" = ".kit/styles/yaml.md" ]
  run in_template template:applies .kit/styles/just.md justfile
  [ "$(cut -d $'\x1f' -f 1 <<<"$output" | paste -s -d ' ' -)" = "root module" ]
  run in_template template:applies .kit/styles/just.md tools.just
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

@test "a hole that is a line matches any run of lines, none included, one within a line any text on it, and unapply gives the values" {
  printf 'head «title» tail\n«body»\nend\n' > "${BATS_TEST_TMPDIR}/t"
  printf 'head anything at all tail\nend\n' > none
  printf 'head x tail\none\ntwo\nend\n' > two
  printf 'head x tail\nend\nmore\n' > after
  [ "$(in_template template:unapply "${BATS_TEST_TMPDIR}/t" none | jq -c .)" = '{"title":"anything at all","body":""}' ]
  [ "$(in_template template:unapply "${BATS_TEST_TMPDIR}/t" two | jq -c .)" = '{"title":"x","body":"one\ntwo\n"}' ]
  run ! in_template template:unapply "${BATS_TEST_TMPDIR}/t" after
  printf 'a.b «x»\n' > "${BATS_TEST_TMPDIR}/dot"
  printf 'aXb y\n' > dot
  run ! in_template template:unapply "${BATS_TEST_TMPDIR}/dot" dot
}

@test "every governed file in this repository renders back from what it gives its template" {
  local root path rule variant values file bound name failed=0 checked=0
  root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  cd "${root}"
  while IFS= read -r path
  do
    [[ -f "${path}" ]] || continue
    while IFS= read -r rule
    do
      while IFS=$'\x1f' read -r variant _ _
      do
        [[ -n "${variant}" ]] || continue
        file=.kit/templates/${rule}/${variant}
        values=$(in_template template:unapply "${file}" "${path}") || continue
        checked=$(( checked + 1 ))
        name=${path##*/}
        bound=$(jq -nc --arg p "${path}" --arg n "${name}" --arg s "${name%.*}" '{path: $p, name: $n, stem: $s}')
        if ! cmp --silent "${path}" <(in_template template:render "${file}" "$(jq -c --argjson b "${bound}" '. + $b' <<<"${values}")")
        then
          echo "does not render back: ${path} from ${file}"
          failed=$(( failed + 1 ))
        fi
      done < <(in_template template:applies ".kit/styles/${rule}.md" "${path}")
    done < <(in_template template:rules "${path}" | sed 's|.*/||; s|\.md$||' | sort -u)
  done < <(git ls-files | grep -vE '(^|/)\.kit/templates/|/regions/')
  echo "checked ${checked}"
  [ "${checked}" -ge 50 ]
  [ "${failed}" -eq 0 ]
}

@test "kit lint refuses a template with two holes and nothing between them" {
  mkdir -p .kit/packages/odd/files/repo/.kit/templates/notes .kit/packages/odd/files/repo/.kit/styles
  printf 'description: "x"\ndepends:\n  - kit\n' > .kit/packages/odd/package.yml
  printf -- '---\nkind:\n  - markdown\ntemplates:\n  - variant: twin\n    path: "**/*.notes.md"\n---\n# Notes\n' \
    > .kit/packages/odd/files/repo/.kit/styles/notes.md
  printf '# «title»\n«a»«b»\n' > .kit/packages/odd/files/repo/.kit/templates/notes/twin
  "${kit_root}/bin/kit" package add odd >/dev/null 2>&1
  printf '# x\nyz\n' > a.notes.md
  run "${kit_root}/bin/kit" lint a.notes.md
  [ "$status" -ne 0 ]
  [[ "$output" == *".kit/templates/notes/twin:2: two holes with nothing between them"* ]]
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
  [[ "$output" == *".kit/styles/yaml.md: the shape of its templates over 0 path(s)"* ]]
}

@test "any --name=value is a hole's value, whatever its case, and never the path" {
  run "${kit_root}/bin/kit" new file bash/up.sh --purpose=x --Odd=y
  [ "$status" -eq 0 ]
  [ -f bash/up.sh ]
  [ ! -e ./--Odd=y ]
}

@test "a seam's provider defines the seam's calls first, in name order, and kit lint refuses one that lacks a call" {
  local root
  root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  mkdir -p bash/toolchain
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' >> .kit/settings.yml
  "${kit_root}/bin/kit" new provider toolchain nix >/dev/null 2>&1
  run "${kit_root}/bin/kit" lint bash/toolchain/nix.sh
  [ "$status" -eq 0 ]
  sed -i.bak '/^toolchain\/nix:outdated() {$/,/^}$/d' bash/toolchain/nix.sh && rm bash/toolchain/nix.sh.bak
  run "${kit_root}/bin/kit" lint bash/toolchain/nix.sh
  [ "$status" -ne 0 ]
  [[ "$output" == *"bash/toolchain/nix.sh does not have the shape of the bash template (toolchain-provider)"* ]]
  [[ "$output" == *"toolchain-provider line 4: toolchain/«stem»:outdated()"* ]]
}

@test "each seam's header lists the calls its providers' template holds, beside any one provider alone implements" {
  local root seam header template
  root=$(cd "${BATS_TEST_DIRNAME}/.." && pwd)
  for seam in forge policy toolchain tracker
  do
    header=$(awk '/Every provider implements/ { on = 1; next } on && /^#$/ && n > 0 { exit }
      on && /^#   [a-z]/ && !/only .* implements it/ { sub(/^#   /, ""); print $1; n++ }' \
      "${root}/kit/modules/bash/${seam}.sh" | sort | paste -s -d ' ' -)
    template=$(sed -n -E "s|^${seam}/«stem»:([a-z-]+)\(\).*|\1|p" \
      "${root}/kit/packages/kit/files/repo/.kit/templates/bash/${seam}-provider" | sort | paste -s -d ' ' -)
    [ "${header}" = "${template}" ] || { echo "${seam}: header ${header} / template ${template}"; return 1; }
  done
}

# ----------------------------------------------------------------------------------------------------------------------
