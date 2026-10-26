#!/usr/bin/env just --justfile
# justfile - see: https://just.systems/man/en/
# ----------------------------------------------------------------------------------------------------------------------
# The root justfile: the gate, and the refresh of this repository's own installed copy. The loop's commands are
# kit issue, rebase, tidy, and status, spelled that way and no other; mise.toml puts bin/ on PATH.

set dotenv-load := true
set positional-arguments := true

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
# against their schemas, the bats suite, and bin/kit check over this repository's installed copy of its own kit. The runner runs it on every pull request, and a desk runs it the
# same way; the pre-commit hook is the narrower, faster check over the staged paths.
#
#   just ci
[doc("🚦 The gate: kit lint, the schemas, bats, kit check")]
ci:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  mise exec -- bin/kit lint --all-files
  mise exec -- ajv validate --spec=draft2020 -s modules/module.schema.json -d 'modules/*/module.yml'
  mise exec -- ajv validate --spec=draft2020 -s .kit/settings.schema.json -d .kit/settings.yml
  mise exec -- bats --print-output-on-failure tests/
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
refresh:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  paths() { awk 'length($1) == 64 { print substr($0, 67) }' .kit/manifest; }
  readarray -t before < <(paths)
  index=$(mktemp -d)
  trap 'rm -rf "${index}"' EXIT
  git checkout-index --all --prefix="${index}/"
  "${index}/bin/kit" update >/dev/null
  readarray -t after < <(paths)
  git add --all -- "${before[@]}" "${after[@]}" .kit/manifest
  echo "✅ refreshed from the index and staged: ${#after[@]} delivered paths, modules $(sed -n 's/^modules *//p' .kit/manifest)"

# ----------------------------------------------------------------------------------------------------------------------
