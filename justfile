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

# The gate a pull request passes: the linters lefthook.yml declares, over every path, so a runner and a desk's
# pre-commit hook agree; the bats suite; and bin/kit check over this repository's installed copy of its own kit.
# Every tool comes from mise.toml.
#
#   just ci
[doc("🚦 Run the gate: the linters over every path, bats, kit check")]
ci:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  mise exec -- lefthook run pre-commit --all-files
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
