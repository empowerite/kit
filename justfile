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
  bats --print-output-on-failure tests/
  bin/kit check
  echo "✅ ci passed"

# Bring this repository's own installed copies up to the index and stage the result. Here the checkout is the
# release, so an edit under modules/ is an edit to the release, not a consumer's drift. The copies are made from
# the index, what the commit will carry, never from the working tree, so a half-staged edit cannot put a copy
# into a commit that lacks its source. It is kit update at the same version: an untouched copy is replaced, a
# retired one removed, a region rewritten between its markers with the rest of its file left alone. The
# pre-commit hook runs this on a commit that touches the source; by hand it is for a desk without the hook.
#
#   just refresh
[doc("🔁 Bring this repository's own installed copies up to the index, and stage the result")]
[script]
refresh:
  set -o errexit -o nounset -o pipefail
  paths() { awk 'length($1) == 64 { print substr($0, 67) }' .kit/manifest; }
  readarray -t before < <(paths)
  index=$(mktemp -d)
  trap 'rm -rf "${index}"' EXIT
  git checkout-index --all --prefix="${index}/"
  "${index}/bin/kit" update >/dev/null
  readarray -t after < <(paths)
  git add --all -- "${before[@]}" "${after[@]}" .kit/manifest
  modules=$(sed -n 's/^modules *//p' .kit/manifest)
  echo "✅ refreshed from the index and staged: ${#after[@]} delivered paths, modules ${modules}"

# ----------------------------------------------------------------------------------------------------------------------
