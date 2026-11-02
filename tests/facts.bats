#!/usr/bin/env bats
# facts.bats - the facts document: its moment, and the tree of files git holds, keyed by path, as the engines read it
# ----------------------------------------------------------------------------------------------------------------------

load helper

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
