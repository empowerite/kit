#!/usr/bin/env just --justfile
# justfile - see: https://just.systems/man/en/
# ----------------------------------------------------------------------------------------------------------------------
# The root justfile: the gate (just ci) and just refresh, which rebuilds this repository's installed copy of its
# own kit. For the workflow, use kit issue, rebase, tidy, and status, not the raw git commands. mise.toml puts bin/
# on PATH.

set dotenv-load := true
set positional-arguments := true

# Recipes marked [script] run under this interpreter, as the just rule requires. It is named bare so it resolves
# through the PATH set below. bash comes from the machine and must be 4.4 or later, as for the kit itself.
set script-interpreter := ['bash']

# Put mise's pinned tool directories first on PATH, once, at parse time. Recipes then call each tool by its bare
# name and get the pinned version. mise-action puts the same directories on the runner's PATH, so a desk and the
# runner run the same tools. A directory already on PATH is not added again, so a nested just changes nothing.
export PATH := ```
  mise bin-paths 2>/dev/null | while IFS= read -r d
  do
    case ":${PATH}:" in (*":${d}:"*) ;; (*) printf '%s:' "${d}" ;; esac
  done
  ``` + env('PATH')

# This file's own path, assigned at parse time so the body below needs no interpolation.
export root_justfile := source_file()

# Default recipe: list this justfile's recipes.
_: help

# List recipes in this justfile.
[no-exit-message]
@help *options:
  just --justfile "${root_justfile}" --list

set export := true

# The gate. It runs, in order:
#
#   - kit lint over every tracked path
#   - ajv: every module.yml and .kit/settings.yml against its schema
#   - lychee: every link in the markdown, offline, including fragments
#   - the bats suite
#   - bin/kit check over this repository's installed copy of its own kit
#
# CI runs it on every pull request, and a desk runs it the same way. The pre-commit hook is a faster check over the
# staged paths only.
#
#   just ci
[doc("🚦 The gate: kit lint, the schemas, the links, bats, kit check")]
[script]
ci:
  set -o errexit -o nounset -o pipefail
  bin/kit lint --all-files
  ajv validate --spec=draft2020 -s modules/module.schema.json -d 'modules/*/module.yml'
  ajv validate --spec=draft2020 -s .kit/settings.schema.json -d .kit/settings.yml
  lychee --offline --include-fragments --no-progress README.md AGENTS.md docs modules .kit
  bats --timing --print-output-on-failure tests/
  bin/kit check
  echo "✅ ci passed"

# Rebuild this repository's installed copies of its own kit from the index, and stage the result. In this
# repository the checkout is the release, so an edit under modules/ is a change to the release, not drift.
#
# The recipe works on the index, which is what the commit will contain:
#
#   1. Export the index. Export HEAD's bin/ and modules/ as the base release, the one the copies were last built
#      from.
#   2. Run kit update over the exported index with that base, and stage the result.
#   3. Run kit update in the working tree too, so the desk's copies match the index in the parts kit owns.
#
# Unstaged edits outside a region, or outside kit's entries in a merged JSON file, are not staged. An entry the
# source removed is removed from the copy, because the base still has it. The pre-commit hook runs this when a
# commit touches the source. Run it by hand on a desk without the hook.
#
#   just refresh
[doc("🔁 Bring this repository's own installed copies up to the index, and stage the result")]
[script]
refresh:
  set -o errexit -o nounset -o pipefail
  work=$(mktemp -d)
  trap 'rm -rf "${work}"' EXIT
  mkdir "${work}/index" "${work}/base"
  git checkout-index --all --prefix="${work}/index/"
  git archive HEAD bin modules | tar -x -C "${work}/base"
  cp "$(git rev-parse --git-path index)" "${work}/index.file"
  # The export is not a git repository. The ceiling stops git from finding this one above the temp directory.
  (cd "${work}/index" && GIT_CEILING_DIRECTORIES="${work}" KIT_BASE="${work}/base" bin/kit update >/dev/null)
  GIT_DIR=$(git rev-parse --absolute-git-dir) GIT_WORK_TREE="${work}/index" GIT_INDEX_FILE="${work}/index.file" \
    git -C "${work}/index" add --all
  git read-tree "$(GIT_INDEX_FILE="${work}/index.file" git write-tree)"
  KIT_BASE="${work}/base" "${work}/index/bin/kit" update >/dev/null
  modules=$(sed -n 's/^modules *//p' .kit/manifest)
  count=$(awk 'length($1) == 64' .kit/manifest | wc -l | tr -d ' ')
  echo "✅ refreshed from the index and staged: ${count} delivered paths, modules ${modules}"

# ----------------------------------------------------------------------------------------------------------------------
