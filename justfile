#!/usr/bin/env just --justfile
# justfile - see: https://just.systems/man/en/
# ----------------------------------------------------------------------------------------------------------------------
# The root justfile: the gate, and the refresh of this repository's own installed copy. The loop's commands are
# kit issue, rebase, tidy, and status, spelled that way and no other; mise.toml puts bin/ on PATH.

set dotenv-load := true
set positional-arguments := true

# A bash recipe body is hosted by the interpreter named here and marked [script], as the just rule asks. The
# interpreter is named bare so it resolves through the PATH built below; bash comes from the machine, 4.4 or
# later, as the kit itself requires.
set script-interpreter := ['bash']

# The mise-pinned toolchain's bin directories go first on PATH once, at parse time, so every recipe calls its
# tools bare and the pinned tool wins over any other copy: the same directories mise-action puts on the
# runner's PATH, so a desk and the runner cannot disagree about which tool ran. Idempotent by inspecting
# PATH rather than a sentinel, so a nested just adds nothing.
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

# The gate: kit lint over every tracked path, each rule's own linter, every module.yml and .kit/settings.yml
# against their schemas, the bats suite, and bin/kit check over this repository's installed copy of its own kit.
# The runner runs it on every pull request, and a desk runs it the same way; the pre-commit hook is the narrower,
# faster check over the staged paths.
#
#   just ci
[doc("🚦 The gate: kit lint, the schemas, bats, kit check")]
[script]
ci:
  set -o errexit -o nounset -o pipefail
  bin/kit lint --all-files
  ajv validate --spec=draft2020 -s modules/module.schema.json -d 'modules/*/module.yml'
  ajv validate --spec=draft2020 -s .kit/settings.schema.json -d .kit/settings.yml
  bats --timing --print-output-on-failure tests/
  bin/kit check
  echo "✅ ci passed"

# Bring this repository's own installed copies up to the index and stage the result. Here the checkout is the
# release, so an edit under modules/ is an edit to the release, not a consumer's drift. Everything is done
# against the index, what the commit will carry: the sources and the copies are exported from it, kit update
# runs over the export with HEAD's release as the base of every merge, the release the copies were last made
# from, and the export is staged back. A desk's unstaged text outside a region, or outside the kit's entries in
# a merged JSON file, is not staged with it, and an entry the source dropped is retired, since the base still
# carries it. The working tree then takes the same step, so a desk's copies agree with the index inside the
# kit's parts. The pre-commit hook runs this on a commit that touches the source; by hand it is for a desk
# without the hook.
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
  # The export is no repository of its own: the ceiling keeps git from finding this one above the temp directory.
  (cd "${work}/index" && GIT_CEILING_DIRECTORIES="${work}" KIT_BASE="${work}/base" bin/kit update >/dev/null)
  GIT_DIR=$(git rev-parse --absolute-git-dir) GIT_WORK_TREE="${work}/index" GIT_INDEX_FILE="${work}/index.file" \
    git -C "${work}/index" add --all
  git read-tree "$(GIT_INDEX_FILE="${work}/index.file" git write-tree)"
  KIT_BASE="${work}/base" "${work}/index/bin/kit" update >/dev/null
  modules=$(sed -n 's/^modules *//p' .kit/manifest)
  count=$(awk 'length($1) == 64' .kit/manifest | wc -l | tr -d ' ')
  echo "✅ refreshed from the index and staged: ${count} delivered paths, modules ${modules}"

# ----------------------------------------------------------------------------------------------------------------------
