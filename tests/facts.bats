#!/usr/bin/env bats
# facts.bats - the facts document: its moment, and the tree of files git holds, keyed by path, as the engines read it
# ----------------------------------------------------------------------------------------------------------------------

load helper

bats_require_minimum_version 1.5.0

setup() {
  # shellcheck disable=SC2034  # kit-bash reads it
  kit_bash_modules=facts
  fresh-repo >/dev/null
  git config user.name test
  git config user.email test@example.com
}

# commit-all makes one commit of everything staged.
commit-all() {
  git commit --quiet --message "$1"
}

@test "the tree is keyed by path, with type, executable, size, blob, and kinds" {
  mkdir bin
  printf '#!/bin/sh\n' > bin/run
  chmod +x bin/run
  printf '# hello\n' > README.md
  git add bin/run README.md
  commit-all "#1 start: files"
  run kit-bash 'facts:tree HEAD'
  [ "${status}" -eq 0 ]
  jq --exit-status '.["bin/run"].executable == true and .["bin/run"].type == "file"' <<<"${output}"
  jq --exit-status '.["README.md"].executable == false' <<<"${output}"
  jq --exit-status --argjson size "$(wc -c < README.md)" '.["README.md"].size == $size' <<<"${output}"
  jq --exit-status --arg blob "$(git rev-parse HEAD:README.md)" '.["README.md"].blob == $blob' <<<"${output}"
  jq --exit-status '.["README.md"].kinds == []' <<<"${output}"
}

@test "the kinds of a path come from the kinds files" {
  mkdir -p .kit/kinds
  printf 'name: markdown\npaths:\n  - "**/*.md"\n' > .kit/kinds/markdown.yml
  printf 'x\n' > a.md
  printf 'x\n' > b.txt
  git add a.md b.txt
  commit-all "#1 start: files"
  run kit-bash 'facts:tree HEAD'
  [ "${status}" -eq 0 ]
  jq --exit-status '.["a.md"].kinds == ["markdown"] and .["b.txt"].kinds == []' <<<"${output}"
}

@test "the tree from the index holds what is staged, not what is on disk" {
  printf 'x\n' > a.md
  git add a.md
  chmod +x a.md
  run kit-bash 'facts:tree'
  [ "${status}" -eq 0 ]
  jq --exit-status '.["a.md"].executable == false' <<<"${output}"
}

@test "a link and a submodule are typed, and never executable" {
  printf 'x\n' > a.md
  ln -s a.md ln
  git add a.md ln
  git update-index --add --cacheinfo "160000,$(git hash-object a.md),sub"
  commit-all "#1 start: files"
  run kit-bash 'facts:tree HEAD'
  [ "${status}" -eq 0 ]
  jq --exit-status '.ln.type == "link" and .ln.executable == false' <<<"${output}"
  jq --exit-status '.sub.type == "submodule" and .sub.executable == false and .sub.size == 0' <<<"${output}"
}

@test "a path with a space, an accent, or a newline comes through whole" {
  printf 'x\n' > 'a b.md'
  printf 'x\n' > 'é.md'
  printf 'x\n' > $'x\ny.md'
  git add --all
  run kit-bash 'facts:tree'
  [ "${status}" -eq 0 ]
  jq --exit-status 'has("a b.md") and has("é.md") and has("x\ny.md")' <<<"${output}"
  jq --exit-status '.["x\ny.md"].kinds == []' <<<"${output}"
}

@test "an ignored file is not in the tree" {
  printf 'out/\n' > .gitignore
  mkdir out
  printf 'x\n' > out/x
  git add --all
  run kit-bash 'facts:tree'
  [ "${status}" -eq 0 ]
  jq --exit-status 'has("out/x") | not' <<<"${output}"
  jq --exit-status 'has(".gitignore")' <<<"${output}"
}

@test "kit facts prints the moment and the tree beside the naming facts" {
  printf '# hello\n' > README.md
  git add README.md
  commit-all "#1 start: a readme"
  printf 'x\n' > staged.md
  git add staged.md
  run "${kit_root}/bin/kit" facts
  [ "${status}" -eq 0 ]
  jq --exit-status '.moment == "ci" and (.tree | type == "object")' <<<"${output}"
  jq --exit-status 'keys_unsorted == ["moment", "branch", "default_branch", "commits", "change", "tree", "policy"]' \
    <<<"${output}"
  # at ci, the tree is HEAD, never the index
  jq --exit-status '(.tree | has("README.md")) and (.tree | has("staged.md") | not)' <<<"${output}"
  [ "$("${kit_root}/bin/kit" facts)" = "$("${kit_root}/bin/kit" policy --input)" ]
}

@test "kit facts --moment=push is refused, and the moments are its choices" {
  run "${kit_root}/bin/kit" facts --moment=push
  [ "${status}" -eq 2 ]
  [[ "${output}" == *"--moment 'push' is not one of the choices"* ]]
  run "${kit_root}/bin/kit" syntax facts --choices --moment
  [ "${status}" -eq 0 ]
  [ "$(cut -f 1 <<<"${output}" | tr '\n' ' ')" = "commit ci " ]
}

@test "a file under facts.read carries its text, and no other file does" {
  printf 'facts:\n  read:\n    - README.md\n' > kit.yml
  printf '# hello\n\nsecond line' > README.md
  printf 'x\n' > a.md
  git add kit.yml README.md a.md
  printf 'changed on disk\n' > README.md
  run kit-bash 'facts:tree'
  [ "${status}" -eq 0 ]
  jq --exit-status --arg text $'# hello\n\nsecond line' '.["README.md"].text == $text' <<<"${output}"
  jq --exit-status '.["a.md"] | has("text") | not' <<<"${output}"
}

@test "a binary, an oversized, or a missing file under facts.read has no text, and is named once" {
  printf 'facts:\n  read:\n    - bin.dat\n    - big.txt\n    - nothing/*.md\n' > kit.yml
  printf 'a\0b\n' > bin.dat
  head -c 65537 /dev/zero | tr '\0' 'x' > big.txt
  git add kit.yml bin.dat big.txt
  run --separate-stderr kit-bash 'facts:tree'
  [ "${status}" -eq 0 ]
  jq --exit-status '(.["bin.dat"] | has("text") | not) and (.["big.txt"] | has("text") | not)' <<<"${output}"
  [ "$(grep -c 'bin.dat' <<<"${stderr}")" -eq 1 ]
  [ "$(grep -c 'big.txt' <<<"${stderr}")" -eq 1 ]
  [ "$(grep -c . <<<"${stderr}")" -eq 2 ]
  [[ "${stderr}" == *binary* && "${stderr}" == *"65536"* ]]
}

@test "at ci, the commits past the base that cannot be read fail the facts, named, and never count as none" {
  git commit --quiet --allow-empty --message "#1 start: the first"
  git update-ref refs/remotes/origin/main HEAD
  git switch --quiet --create 7-fix-the-thing
  git commit --quiet --allow-empty --message "#7 cli: fix the thing"
  run --separate-stderr kit-bash 'git() { [[ "$1 $2" != "log --no-merges" ]] || return 1; command git "$@"; }
    facts:document ci'
  [ "${status}" -eq 1 ]
  [ -z "${output}" ]
  said="run: git log --no-merges origin/main..HEAD (the commits past the branch's base could not be read)"
  [[ "${stderr}" == *"${said}"* ]]
}

@test "kit.yml with a facts key the kit does not know is refused" {
  printf 'facts:\n  reed:\n    - x\n' > kit.yml
  kit_bash_modules="facts settings"
  run kit-bash 'settings:validate'
  [ "${status}" -eq 1 ]
  [[ "${output}" == *reed* ]]
  printf 'facts:\n  read:\n    - x\n' > kit.yml
  run kit-bash 'settings:validate'
  [ "${status}" -eq 0 ]
}
