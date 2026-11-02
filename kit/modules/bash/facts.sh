# shellcheck shell=bash
# facts.sh - the facts about the repository that a policy engine judges (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
#
# The facts are JSON that kit policy hands to a policy engine, so a rule can judge more than a name. This module
# gathers the ones that come from git; policy:input puts them in the document it builds.
#
# facts:tree prints the `tree`: the files git holds, as an object keyed by repository-relative path. Each entry has:
#
#   - `type`: file, link, or submodule
#   - `executable`: true for a file of mode 100755, and false for everything else
#   - `size`: the bytes of the blob, and 0 for a submodule
#   - `blob`: the object name of the blob, or of the commit a submodule is at
#   - `kinds`: the kinds of file the path is, as context:classify finds them, and [] where there are none
#
# The tree is read from a tree-ish when one is given, and otherwise from the index, so it holds what is staged and
# not what is on disk. A file git ignores is not in either. Every list from git is NUL-separated, so a path with a
# space, an accent, or a newline comes through whole. context:classify takes one path to a line, so a path with a
# newline has no kinds.

use context

# facts:tree [<tree-ish>] prints the tree as a JSON object. With a <tree-ish>, it reads that tree; without one, the
# index named by GIT_INDEX_FILE, or the default index. It runs from the repository root.
facts:tree() {
  local work
  work=$(mktemp -d) || return 1
  facts:tree-in "${work}" "$@"
  local -i status=$?
  rm -rf "${work}"
  return "${status}"
}

# facts:tree-in <dir> [<tree-ish>] does the work of facts:tree, keeping its lists in <dir>. entries.json is one object
# per path, from git's list; sizes holds one line per blob, in the order of entries.json; kinds is context:classify's
# answer for every path that fits on a line.
facts:tree-in() {
  local dir=$1
  # ls-tree prints "<mode> <type> <blob>\t<path>" and ls-files --stage "<mode> <blob> <stage>\t<path>"; jq reads both.
  if (( $# > 1 ))
  then
    git ls-tree -z -r --full-tree "$2" > "${dir}/list" || return 1
  else
    git ls-files -z --stage > "${dir}/list" || return 1
  fi
  jq --raw-input --slurp --compact-output --argjson tree "$(( $# > 1 ? 1 : 0 ))" '
    split("\u0000") | map(select(length > 0)
      | index("\t") as $tab | {meta: .[:$tab], path: .[$tab + 1:]}
      | (.meta | split(" ")) as $m
      | {path, mode: $m[0], blob: (if $tree == 1 then $m[2] else $m[1] end),
         stage: (if $tree == 1 then "0" else $m[2] end)}
      | select(.stage == "0"))' "${dir}/list" > "${dir}/entries.json" || return 1
  # Every blob but a submodule's commit gets its size from one git process.
  jq --raw-output '.[] | select(.mode != "160000") | .blob' "${dir}/entries.json" \
    | git cat-file --batch-check='%(objectsize)' > "${dir}/sizes" || return 1
  jq --join-output '.[] | select(.path | contains("\n") | not) | .path + "\n"' "${dir}/entries.json" \
    | context:classify > "${dir}/kinds" || return 1
  jq --null-input --compact-output --rawfile sizes "${dir}/sizes" --rawfile kinds "${dir}/kinds" \
    --slurpfile entries "${dir}/entries.json" '
    ($sizes | split("\n")) as $size
    | ($kinds | split("\n") | map(select(length > 0) | capture("^(?<path>.*)\t(?<kinds>[^\t]*)$"))
       | map({key: .path, value: (.kinds | split(" ") | map(select(length > 0)))}) | from_entries) as $kind
    | reduce ($entries[0] | to_entries[]) as $e ({n: 0, tree: {}};
        ($e.value) as $f
        | ($f.mode == "160000") as $sub
        | .tree[$f.path] = {
            type: (if $sub then "submodule" elif $f.mode == "120000" then "link" else "file" end),
            executable: ($f.mode == "100755"),
            size: (if $sub then 0 else ($size[.n] | tonumber) end),
            blob: $f.blob,
            kinds: ($kind[$f.path] // [])}
        | .n += (if $sub then 0 else 1 end))
    | .tree'
}
# ----------------------------------------------------------------------------------------------------------------------
