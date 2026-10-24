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

# Reinstall this repository's own kit from its source. Here the checkout is the release, so an edit to a file under
# tree/ or targets/ is an edit to the release, not a consumer's drift: the installed copies, .claude/settings.json
# (which carries only the kit's entries here), and .kit/manifest are rewritten from it. Run after changing any
# delivered file, and commit the result with the change.
#
#   just refresh
[doc("🔁 Reinstall this repository's own kit from tree/, targets/, and bin/kit")]
refresh:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  rm -f .kit/manifest .claude/settings.json
  bin/kit install claude

# ----------------------------------------------------------------------------------------------------------------------
