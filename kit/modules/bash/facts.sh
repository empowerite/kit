# shellcheck shell=bash
# facts.sh - the facts about the repository that a policy engine judges (sourced module)
# ----------------------------------------------------------------------------------------------------------------------
# The facts are JSON that kit policy hands to a policy engine, so a rule can judge more than a name. This module
# gathers them into one document, which kit facts prints. It holds, in this order:
#
#   - `moment`: commit or ci, the point the facts are judged at
#   - `branch` and `default_branch`; the branch is null on the default branch or a detached HEAD
#   - `commits`: at ci, those past the fork from origin's default branch, merges excluded, or null when the fork is not
#     here, as in a shallow clone; at commit, the one commit
#   - `change`: at ci, the open change on the branch, with its title, body, and author; at commit, always null
#   - `tree`: at ci, HEAD; at commit, the tree the commit will make
#   - `policy`: the policy settings, with the kit's defaults filled in
#
# A fact a moment cannot know is null, so a rule over a change's title never fires at commit.
#
# facts:tree prints the `tree`: the files git holds, as an object keyed by repository-relative path. Each entry has:
#
#   - `type`: file, link, or submodule
#   - `executable`: true for a file of mode 100755, and false for everything else
#   - `size`: the bytes of the blob, and 0 for a submodule
#   - `blob`: the object name of the blob, or of the commit a submodule is at
#   - `kinds`: the kinds of file the path is, as context:classify finds them, and [] where there are none
#   - `text`: the content of the blob, only for a file that matches a glob under `facts.read` in kit.yml
#
# The tree is read from a tree-ish when one is given, and otherwise from the index, so it holds what is staged and
# not what is on disk. A file git ignores is not in either. Every list from git is NUL-separated, so a path with a
# space, an accent, or a newline comes through whole. context:classify takes one path to a line, so a path with a
# newline has no kinds.
#
# The `text` is read from the blob in the tree, never from disk, so it is the content of the tree's moment. A binary
# blob, one with a NUL byte, and one over 65536 bytes get no `text`; kit warns once for each, naming the path. A glob
# that matches nothing is no fault and says nothing.

use context git policy yaml

# The most bytes of a blob that facts:tree gives as text.
declare -gri facts_text_limit=65536

# The moments a document can be for.
declare -gra facts_moments=(commit ci)

# facts:choices <key> [<answer>…] answers kit syntax --choices for kit facts' --moment: a closed set, the moments.
facts:choices() {
  [[ "$1" == --moment ]] || return 1
  if [[ "${2:-}" == --kind ]]
  then
    printf 'closed\n'
    return 0
  fi
  printf '%s\t%s\t\t\n' commit 'the tree the commit will make, as kit commit judges it' \
    ci 'HEAD, the commits, and the change, as kit policy judges them'
}

# facts:document <moment> [<tree-ish>] prints the document described in the file header. At ci, the tree is read from
# the tree-ish, HEAD by default, and is empty while HEAD has no commit. When the fork point is missing, as in a
# shallow clone, it warns and gives null for the commits instead of guessing. The change is read from the forge (GitHub or
# GitLab), and is null when there is no forge or no open change. At commit, the tree is the index, as facts:tree
# reads it; commits is empty and change is null, so the caller fills in the commit it makes.
facts:document() {
  local moment=$1 treeish=${2:-HEAD} default_branch branch commits='[]' change='null' rules number title body author
  local moments
  if ! list:among "${moment}" "${facts_moments[@]}"
  then
    printf -v moments '%s, ' "${facts_moments[@]}"
    error "'${moment}' is not a moment; the moments: ${moments%, }"
    return 2
  fi
  rules=$(policy:rules) || return 1
  default_branch=$(git:default-branch) || return 1
  branch=$(policy:branch)
  [[ "${branch}" != "${default_branch}" ]] || branch=''
  if [[ "${moment}" == ci && -n "${branch}" ]]
  then
    if git rev-parse --verify --quiet "origin/${default_branch}" >/dev/null \
      && git merge-base "origin/${default_branch}" HEAD >/dev/null 2>&1
    then
      commits=$(git log --no-merges --format='%H%x1f%s' "origin/${default_branch}..HEAD" \
        | jq --raw-input --slurp 'split("\n") | map(select(. != "") | split("\u001f") | {sha: .[0], subject: .[1]})')
    else
      commits=null
      warn "the fork from origin/${default_branch} is not here, so no commit is checked; fetch with full history"
    fi
    if use forge 2>/dev/null && number=$(forge:pr-here 2>/dev/null) && [[ -n "${number}" ]]
    then
      title=$(forge:pr-title "${number}" 2>/dev/null || true)
      body=$(forge:pr-body "${number}" 2>/dev/null || true)
      author=$(forge:pr-author "${number}" 2>/dev/null || true)
      change=$(jq --null-input --argjson number "${number}" --arg title "${title}" --arg body "${body}" \
        --arg author "${author}" \
        '{number: $number, title: $title, body: $body, author: (if $author == "" then null else $author end)}')
    fi
  fi
  # The tree can be large, so it reaches jq on stdin, never as an argument. Each stage's status is checked, since a
  # caller may not set pipefail.
  facts:document-tree "${moment}" "${treeish}" \
    | jq --arg moment "${moment}" --arg branch "${branch}" --arg default_branch "${default_branch}" \
      --argjson commits "${commits}" --argjson change "${change}" --argjson policy "${rules}" \
      '{moment: $moment, branch: (if $branch == "" then null else $branch end), default_branch: $default_branch,
        commits: $commits, change: $change, tree: ., policy: $policy}'
  local -a statuses=("${PIPESTATUS[@]}")
  (( statuses[0] == 0 && statuses[1] == 0 ))
}

# facts:document-tree <moment> <tree-ish> prints the tree for the moment: the index at commit, and the tree-ish at
# ci. An empty object stands for HEAD before the first commit.
facts:document-tree() {
  if [[ "$1" == commit ]]
  then
    facts:tree
  elif [[ "$2" == HEAD ]] && ! git rev-parse --verify --quiet HEAD >/dev/null
  then
    printf '{}\n'
  else
    facts:tree "$2"
  fi
}

# facts:tree [<tree-ish>] prints the tree as a JSON object. With a <tree-ish>, it reads that tree; without one, the
# index named by GIT_INDEX_FILE, or the default index. It runs from the repository root.
facts:tree() {
  local work
  file:scratch work || return 1
  facts:tree-in "${work}" "$@"
  local -i status=$?
  rm -rf "${work}"
  return "${status}"
}

# facts:tree-in <dir> [<tree-ish>] does the work of facts:tree, keeping its lists in <dir>. entries.json is one object
# per path, from git's list; sizes holds one line per blob, in the order of entries.json; kinds is context:classify's
# answer for every path that fits on a line; texts.json is one {key, value} per file that facts:texts gives as text.
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
  facts:texts "${dir}" || return 1
  jq --null-input --compact-output --rawfile sizes "${dir}/sizes" --rawfile kinds "${dir}/kinds" \
    --slurpfile entries "${dir}/entries.json" --slurpfile texts "${dir}/texts.json" '
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
    | .tree
    | reduce $texts[] as $t (.; if has($t.key) then .[$t.key].text = $t.value else . end)'
}

# facts:texts <dir> writes <dir>/texts.json: one {key, value} per file whose path matches a glob under facts.read in
# kit.yml, the value being the blob read from git. It reads <dir>/entries.json. A link and a submodule have no text.
# A binary blob or one over the limit is skipped with a warning naming it.
facts:texts() {
  local dir=$1 settings glob path blob size matched doc
  local -a globs=()
  : > "${dir}/texts.json"
  settings=$(settings:file)
  [[ -f "${settings}" ]] || return 0
  doc=$(yaml:json "${settings}") || return 1
  list:read globs jq --raw-output '.facts.read // [] | .[]' <<<"${doc}" || return 1
  (( ${#globs[@]} > 0 )) || return 0
  while IFS= read -r -d '' path && IFS= read -r -d '' blob
  do
    matched=false
    for glob in "${globs[@]}"
    do
      if context:glob-matches "${path}" "${glob}"
      then
        matched=true
        break
      fi
    done
    [[ "${matched}" == true ]] || continue
    size=$(git cat-file -s "${blob}") || return 1
    if (( size > facts_text_limit ))
    then
      warn "facts.read: ${path} has no text; it is ${size} bytes, over the limit of ${facts_text_limit}"
      continue
    fi
    git cat-file blob "${blob}" > "${dir}/blob" || return 1
    if (( $(LC_ALL=C tr -d '\0' < "${dir}/blob" | wc -c) != size ))
    then
      warn "facts.read: ${path} has no text; it is binary"
      continue
    fi
    jq --null-input --compact-output --arg key "${path}" --rawfile value "${dir}/blob" '{key: $key, value: $value}' \
      >> "${dir}/texts.json" || return 1
  done < <(jq --join-output '.[] | select(.mode == "100644" or .mode == "100755") | .path, "\u0000", .blob, "\u0000"' \
    "${dir}/entries.json")
}
# ----------------------------------------------------------------------------------------------------------------------
