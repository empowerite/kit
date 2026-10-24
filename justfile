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

# The gate: shellcheck over every bash file, actionlint over every workflow, markdownlint over every markdown file,
# the bats suite, and bin/kit check over
# this repository's installed copy of its own kit. The runner runs it on every pull request, and a desk runs it the
# same way; the pre-commit hook is the narrower, faster check over the staged paths.
#
#   just ci
[doc("🚦 The gate: shellcheck, actionlint, markdownlint, bats, kit check")]
ci:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  mise exec -- shellcheck --shell=bash --external-sources bin/kit bin/*.sh
  mise exec -- actionlint
  mise exec -- markdownlint-cli2 '**/*.md'
  mise exec -- bats --print-output-on-failure tests/
  bin/kit check
  echo "✅ ci passed"

# Reinstall this repository's own kit from its source and stage the result. Here the checkout is the release, so an
# edit under tree/ or targets/ is an edit to the release, not a consumer's drift: the targets the manifest names
# are dropped and taken again, so the installed copies and .kit/manifest are rewritten from the source, and every
# path the old or the new manifest names is staged. The pre-commit hook runs this on a commit that touches the
# source; by hand it is for a desk without the hook.
#
#   just refresh
[doc("🔁 Reinstall this repository's own kit from tree/ and targets/, and stage the result")]
refresh:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  # a delivered path's line is a 64-hex digest, two spaces, the path; the version and targets lines are not
  readarray -t before < <(awk '$1 ~ /^[0-9a-f]{64}$/ { print $2 }' .kit/manifest)
  read -r -a targets <<<"$(sed -n 's/^targets *//p' .kit/manifest)"
  (( ${#targets[@]} == 0 )) || bin/kit target remove "${targets[@]}" >/dev/null
  rm -f .kit/manifest
  bin/kit install "${targets[@]}" >/dev/null
  readarray -t after < <(awk '$1 ~ /^[0-9a-f]{64}$/ { print $2 }' .kit/manifest)
  git add --all -- "${before[@]}" "${after[@]}" .kit/manifest
  echo "✅ refreshed and staged: ${#after[@]} delivered paths, targets ${targets[*]:-none}"

# ----------------------------------------------------------------------------------------------------------------------
