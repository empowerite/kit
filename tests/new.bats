#!/usr/bin/env bats
# new.bats - kit new makes a rule, a style, a package, a template, a module, a seam's provider, and a verb
# ----------------------------------------------------------------------------------------------------------------------
# kit new file and kit new kind are tested in template.bats and package.bats. These tests make the rest in an empty
# repository with the kit installed, and the verb and the jq module in a copy of the kit's own source.

bats_require_minimum_version 1.5.0

load helper

setup_file() {
  world-repo installed style/bash style/markdown
}

setup() {
  world-copy-repo installed
}

kit() { "${kit_checkout}/bin/kit" "$@"; }

@test "a rule and a style for a kind go into the repository's own package, name the kind, and govern its files once active" {
  kit new kind toml --path='**/*.toml' >/dev/null 2>&1
  run kit new rule toml --lint='taplo lint'
  [ "$status" -eq 0 ]
  file=.kit/packages/local/files/repo/.kit/rules/toml.md
  [ "$(sed -n 2p "${file}")" = "# kit:generated -- the toml rule, from the local package" ]
  grep --quiet --line-regexp '  - toml' "${file}"
  grep --quiet --line-regexp 'lint: taplo lint' "${file}"
  run kit new style toml
  [ "$status" -eq 0 ]
  kit packages add local >/dev/null 2>&1
  run kit context Cargo.toml
  [ "$output" = $'.kit/rules/toml.md\n.kit/styles/toml.md' ]
  run kit lint .kit/packages/local/files/repo/.kit/rules/toml.md .kit/packages/local/files/repo/.kit/styles/toml.md
  [ "$status" -eq 0 ]
  run kit new rule toml
  [ "$status" -eq 1 ]
  [[ "$output" == *"is there already"* ]]
  run kit new rule ini
  [[ "$output" == *"no kind ini is defined yet; run: kit new kind ini"* ]]
}

@test "a package depends on kit and the packages named, and a template is listed in its package's style" {
  run kit new package team --description='The team rules' --depends=style/bash
  [ "$status" -eq 0 ]
  [[ "$output" == *"run: kit packages add team"* ]]
  [ "$(grep -v '^#' .kit/packages/team/package.yml | awk 'NF' | paste -s -d '|' -)" = \
    'description: "The team rules"|depends:|  - kit|  - style/bash' ]
  run kit packages add team
  [ "$status" -eq 0 ]
  kit new style notes --package=team >/dev/null 2>&1
  printf '# «title»\n\n«body»\n' > page.md
  run kit new template notes page --from=page.md --package=team
  [ "$status" -eq 0 ]
  [ "$(cat .kit/packages/team/files/repo/.kit/templates/notes/page)" = "$(cat page.md)" ]
  grep --quiet --line-regexp '  - variant: page' .kit/packages/team/files/repo/.kit/styles/notes.md
  run kit new template other x --package=team
  [[ "$output" == *"list it under templates: in a rule or style named other"* ]]
}

@test "a module and a seam's provider go into the repository's own directory on the module path, the provider with every call" {
  run kit new module notes --purpose='notes on the work'
  [ "$status" -eq 1 ]
  [[ "$output" == *"add one under bash_path in kit.yml"* ]]
  printf 'bash_path:\n  - kit:modules/bash/\n  - bash/\n' >> kit.yml
  mkdir bash
  run kit new module notes --purpose='notes on the work'
  [ "$status" -eq 0 ]
  [ "$(sed -n 2p bash/notes.sh)" = "# notes.sh - notes on the work (sourced module)" ]
  run kit new provider tracker jira
  [ "$status" -eq 0 ]
  for call in assign assignees create me state title url
  do
    grep --quiet "^tracker/jira:${call}() {" bash/tracker/jira.sh || { echo "no ${call}"; return 1; }
  done
  run kit lint bash/notes.sh bash/tracker/jira.sh
  [ "$status" -eq 0 ]
  run bash -c "source '${kit_checkout}/modules/bash/1st.sh'; use tracker/jira; tracker/jira:title 7"
  [ "$status" -eq 1 ]
  [[ "$output" == *"tracker/jira: title is not written yet"* ]]
  run kit new provider nonesuch x
  [ "$status" -eq 2 ]
  run kit new module --jq widgets --purpose=x
  [ "$status" -eq 1 ]
  [[ "$output" == *"a jq module comes from the release alone"* ]]
}

@test "in the kit's own source, a verb has the header kit help reads, and a jq module has its tests" {
  cp -R "${kit_checkout}" kit
  # the kit's own source tells ShellCheck where a verb's loader is, so the made verb lints as the kit's verbs do
  cp "${kit_checkout}/../.shellcheckrc" .
  kit/bin/kit hydrate >/dev/null 2>&1 || true
  run kit/bin/kit new verb frob --purpose='frobs the widgets' --group=files
  [ "$status" -eq 0 ]
  # the header is the verb template's, so kit lint passes it as made
  grep --quiet --line-regexp '# Usage:' kit/bin/kit-frob
  grep --quiet --line-regexp '#   kit frob' kit/bin/kit-frob
  grep --quiet --line-regexp '# Arguments:' kit/bin/kit-frob
  grep --quiet --line-regexp '# Exits 0 on success, and 2 on a usage error.' kit/bin/kit-frob
  run kit/bin/kit lint kit/bin/kit-frob
  [ "$status" -eq 0 ]
  run kit/bin/kit help frob
  [ "${lines[0]}" = "kit frob - frobs the widgets" ]
  run kit/bin/kit frob
  [ "$status" -eq 0 ]
  run kit/bin/kit frob extra
  [ "$status" -eq 2 ]
  run kit/bin/kit new verb frob2 --purpose=x --group=nowhere
  [ "$status" -eq 2 ]
  run kit/bin/kit new module widgets --jq --purpose='widgets, folded'
  [ "$status" -eq 0 ]
  run jq -L kit/modules/jq --run-tests tests/jq/widgets.test
  [ "$status" -eq 0 ]
  cd "${BATS_TEST_TMPDIR}" && git init --quiet other && cd other
  run "${kit_checkout}/bin/kit" new verb frob --purpose=x --group=files
  [ "$status" -eq 1 ]
  [[ "$output" == *"a verb is part of the kit's own source"* ]]
}

# ----------------------------------------------------------------------------------------------------------------------
