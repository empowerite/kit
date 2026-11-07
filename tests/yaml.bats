#!/usr/bin/env bats
# yaml.bats - the kit's files are YAML, read through yq: every form YAML allows reads as YAML says
# ----------------------------------------------------------------------------------------------------------------------

bats_require_minimum_version 1.5.0

load helper

setup() {
  kit_bash_modules=yaml
  fresh-repo
}

@test "a block scalar is read whole" {
  printf 'text: |\n  line one\n  line two\nnext: x\n' > in.yml
  run kit-bash 'yaml:value in.yml text'
  [ "$status" -eq 0 ]
  [ "$output" = $'line one\nline two' ]
}

@test "a list is read as items, written as a flow list or one item per line" {
  printf 'flow: [kit, style/bash, "quoted item"]\nblock:\n  - kit\n  - "style/bash"\n' > in.yml
  run kit-bash 'yaml:list in.yml flow'
  [ "$status" -eq 0 ]
  [ "$output" = $'kit\nstyle/bash\nquoted item' ]
  run kit-bash 'yaml:list in.yml block'
  [ "$output" = $'kit\nstyle/bash' ]
}

@test "a single-quoted value is read literally, a double-quoted one unescaped, and an unquoted one as written" {
  printf 'pattern: %s\ntext: "a\\tb"\nplain: hello world\ndefault: true\nversion: 1.0\n' "'^\\s+#'" > in.yml
  run kit-bash 'yaml:value in.yml pattern'
  [ "$output" = '^\s+#' ]
  run kit-bash 'yaml:value in.yml text'
  [ "$output" = $'a\tb' ]
  run kit-bash 'yaml:value in.yml plain'
  [ "$output" = 'hello world' ]
  run kit-bash 'yaml:value in.yml default'
  [ "$output" = true ]
  run kit-bash 'yaml:value in.yml version'
  [ "$output" = 1.0 ]
}

@test "a comment after an unquoted scalar is dropped, and a # inside quotes is kept" {
  printf 'key: value # a note\nhash: "a # b"\nitems:\n  - one # first\n  - "two # second"\n' > in.yml
  run kit-bash 'yaml:value in.yml key'
  [ "$output" = value ]
  run kit-bash 'yaml:value in.yml hash'
  [ "$output" = 'a # b' ]
  run kit-bash 'yaml:list in.yml items'
  [ "$output" = $'one\ntwo # second' ]
}

@test "a missing key, a key that holds a list or a map, and a missing file all read as nothing" {
  printf 'packages:\n  - kit\nanswers:\n  ci_runner: x\nversion: unreleased\n' > in.yml
  run kit-bash 'yaml:value in.yml nothing; yaml:value in.yml packages; yaml:value in.yml answers
    yaml:list in.yml version'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
  run kit-bash 'yaml:value absent.yml version; yaml:list absent.yml packages; yaml:keys absent.yml'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "CRLF line endings read the same as LF" {
  printf 'version: unreleased\npackages:\n  - kit\n  - style/bash\nanswers:\n  ci_runner: x\n' > lf.yml
  printf 'trackers:\n  - prefix: "#"\n    provider: github\n' >> lf.yml
  sed 's/$/\r/' lf.yml > crlf.yml
  grep --quiet $'\r' crlf.yml
  for read in 'yaml:value FILE version' 'yaml:list FILE packages' 'yaml:map FILE answers' \
    'yaml:records FILE trackers prefix provider' 'yaml:keys FILE'
  do
    run kit-bash "${read//FILE/lf.yml}"
    lf=$output
    [ -n "${lf}" ]
    run kit-bash "${read//FILE/crlf.yml}"
    [ "$status" -eq 0 ]
    [ "$output" = "${lf}" ] || { echo "differs: ${read}: '${output}' vs '${lf}'"; return 1; }
  done
  run kit-bash 'yaml:list crlf.yml packages'
  [ "$output" = $'kit\nstyle/bash' ]
}

@test "an anchor and its alias resolve" {
  printf 'base: &base hello\nother: *base\nmap: &m\n  name: x\ncopy: *m\n' > in.yml
  run kit-bash 'yaml:value in.yml other'
  [ "$output" = hello ]
  run kit-bash 'yaml:map in.yml copy'
  [ "$output" = $'name\tx' ]
}

@test "a map prints each name and value tab-separated, unquoted, and a value that is empty as empty" {
  printf 'answers:\n  ci_runner: "ubuntu-latest"\n  count: 2\n  empty:\n  nested:\n    deeper: x\n' > in.yml
  run kit-bash 'yaml:map in.yml answers'
  [ "$status" -eq 0 ]
  [ "$output" = $'ci_runner\tubuntu-latest\ncount\t2\nempty\t' ]
}

@test "a record list prints the named fields in order, separated by the unit separator, a missing field empty" {
  printf 'expects:\n  - path: a\n    pattern: %s\n    remedy: r\n  - path: b\n  - not a record\n' "'x\\t+y'" > in.yml
  run kit-bash 'yaml:records in.yml expects path remedy pattern'
  [ "$status" -eq 0 ]
  [ "$output" = $'a\x1fr\x1fx\\t+y\nb\x1f\x1f' ]
  run kit-bash 'yaml:records in.yml expects path'
  [ "$output" = $'a\nb' ]
}

@test "yaml:keys prints the top-level keys in order" {
  printf 'version: 1\nlinked-exempt:\n  - x\nanswers:\n  a: b\n' > in.yml
  run kit-bash 'yaml:keys in.yml'
  [ "$status" -eq 0 ]
  [ "$output" = $'version\nlinked-exempt\nanswers' ]
}

@test "the frontmatter of a markdown file is read, and the body is not, whatever the line endings" {
  printf -- '---\nkind:\n  - bash\n  - just\nlint: shellcheck --shell=bash\n---\n' > rule.md
  printf '# Title\n\n---\n\nkey: not yaml\n' >> rule.md
  run kit-bash 'yaml:list rule.md kind'
  [ "$output" = $'bash\njust' ]
  run kit-bash 'yaml:value rule.md lint'
  [ "$output" = 'shellcheck --shell=bash' ]
  run kit-bash 'yaml:keys rule.md'
  [ "$output" = $'kind\nlint' ]
  run kit-bash 'yaml:value rule.md key'
  [ -z "$output" ]
  sed 's/$/\r/' rule.md > crlf.md
  run kit-bash 'yaml:list crlf.md kind'
  [ "$output" = $'bash\njust' ]
}

@test "a file yq cannot parse is refused in one line that names the file, and nothing is read from it" {
  printf 'packages:\n\t- kit\n' > tabs.yml
  run --separate-stderr kit-bash 'yaml:list tabs.yml packages'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [ "${#stderr_lines[@]}" -eq 1 ]
  [[ "$stderr" == *"tabs.yml"* ]]
  [[ "$stderr" == *"cannot start any token"* ]]
  printf 'version: "oops\n' > quote.yml
  run --separate-stderr kit-bash 'yaml:value quote.yml version'
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"quote.yml"* ]]
}

@test "yaml:json prints the document as JSON that jq accepts, the frontmatter alone for a markdown file" {
  printf 'version: unreleased\npackages: [kit, style/bash]\nanswers:\n  count: 2\n' > kit.yml
  run kit-bash 'yaml:json kit.yml'
  [ "$status" -eq 0 ]
  jq --exit-status '.version == "unreleased" and .packages == ["kit", "style/bash"]
    and .answers.count == 2' <<<"$output"
  printf -- '---\nkind:\n  - bash\n---\n# Title\n' > rule.md
  run kit-bash 'yaml:json rule.md'
  [ "$(jq --compact-output . <<<"$output")" = '{"kind":["bash"]}' ]
}

@test "without yq on PATH, the module refuses once, names the remedy, and every read returns 2" {
  printf 'answer: 42\n' > in.yml
  printf 'version: unreleased\n' > kit.yml
  # an empty temporary directory, so the module finds no results of other processes to sweep with the PATH gone
  mkdir tmp
  run env TMPDIR="${repo}/tmp" bash -c 'source "${kit_root}/modules/bash/1st.sh" && PATH=/nonexistent && use yaml
    yaml:value in.yml answer; echo "value=$?"; yaml:list in.yml answer; echo "list=$?"'
  [ "$status" -eq 0 ]
  [ "${#lines[@]}" -eq 3 ]
  [[ "${lines[0]}" == *"run: mise install"* ]]
  [[ "${lines[0]}" == *"brew install yq"* ]]
  [ "${lines[1]}" = value=2 ]
  [ "${lines[2]}" = list=2 ]
}

@test "a yq that is not mikefarah's is refused the same way" {
  mkdir shims
  printf '#!/usr/bin/env bash\necho "yq 3.4.3"\n' > shims/yq && chmod +x shims/yq
  printf 'answer: 42\n' > in.yml
  run env PATH="${repo}/shims:${PATH}" bash -c 'source "${kit_root}/modules/bash/1st.sh" && use yaml
    yaml:value in.yml answer; echo "value=$?"'
  [ "${#lines[@]}" -eq 2 ]
  [[ "${lines[0]}" == *"yq 3.4.3"* ]]
  [[ "${lines[0]}" == *"run: mise install"* ]]
  [ "${lines[1]}" = value=2 ]
}

@test "a result is kept under XDG_CACHE_HOME and replayed without yq; a changed file is read again, at the same size and time" {
  # a yq that logs one line per read it is asked for, then steps out of PATH and hands the call to the yq behind it
  mkdir shims
  printf '#!/usr/bin/env bash\n[[ "$1" == --version ]] || echo run >> "%s/yq.log"\nPATH=${PATH#*:} exec yq "$@"\n' \
    "${repo}" > shims/yq
  chmod +x shims/yq
  printf 'answer: 42\n' > in.yml
  touch -r in.yml stamp
  run env PATH="${repo}/shims:${PATH}" bash -c 'source "${kit_root}/modules/bash/1st.sh" && use yaml
    yaml:value in.yml answer; yaml:value in.yml answer; echo "$(yaml:value in.yml answer)"'
  [ "$output" = $'42\n42\n42' ]
  [ "$(grep -c '' yq.log)" -eq 1 ]
  ls "${XDG_CACHE_HOME}/kit/yaml/" | grep --quiet '^value answer '
  printf 'answer: 43\n' > in.yml
  touch -r stamp in.yml
  run env PATH="${repo}/shims:${PATH}" bash -c 'source "${kit_root}/modules/bash/1st.sh" && use yaml
    yaml:value in.yml answer; yaml:value in.yml answer'
  [ "$output" = $'43\n43' ]
  [ "$(grep -c '' yq.log)" -eq 2 ]
}

@test "a kept result that is a link is neither replayed nor replaced" {
  printf 'answer: 42\n' > in.yml
  run kit-bash 'yaml:value in.yml answer'
  [ "$output" = 42 ]
  entry=$(ls "${XDG_CACHE_HOME}/kit/yaml/" | grep '^value answer ')
  [ -n "${entry}" ]
  printf 'answer: 42\n\0planted\n' > planted
  ln -sf "${repo}/planted" "${XDG_CACHE_HOME}/kit/yaml/${entry}"
  run kit-bash 'yaml:value in.yml answer'
  [ "$status" -eq 0 ]
  [ "$output" = 42 ]
  [ -L "${XDG_CACHE_HOME}/kit/yaml/${entry}" ]
  [ "$(tr '\0' '|' < planted)" = $'answer: 42\n|planted' ]
}

@test "kit hydrate reads a kit.yml with CRLF endings and a flow list, and kit check passes" {
  # kit install writes the root files hydrate never touches; the declaration is then rewritten and .kit/ rebuilt from it
  "${kit_root}/bin/kit" install >/dev/null 2>&1
  [ ! -e .kit/styles/bash.md ]
  printf 'version: unreleased\r\npackages: [kit, style/bash]\r\n' > kit.yml
  rm -rf .kit
  run "${kit_root}/bin/kit" hydrate
  [ "$status" -eq 0 ]
  [ -f .kit/1st.md ]
  [ -f .kit/styles/bash.md ]
  run "${kit_root}/bin/kit" check
  [ "$status" -eq 0 ]
}

@test "yaml:type prints the tag of a key's value: a list, a map, a scalar, and null for a key that is missing or empty" {
  printf 'list: [a]\nmap:\n  k: v\ntext: hello\nempty:\n' > in.yml
  run kit-bash 'yaml:type in.yml list; yaml:type in.yml map; yaml:type in.yml text; yaml:type in.yml empty
    yaml:type in.yml none'
  [ "$status" -eq 0 ]
  [ "$output" = $'!!seq\n!!map\n!!str\n!!null\n!!null' ]
  run kit-bash 'yaml:type absent.yml list'
  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

# ----------------------------------------------------------------------------------------------------------------------
