#!/usr/bin/env bats
# facts.bats - the tree fact: the files git holds, keyed by path, as the policy engines read them
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
