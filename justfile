#!/usr/bin/env just --justfile
# justfile - see: https://just.systems/man/en/
# ----------------------------------------------------------------------------------------------------------------------
# The root justfile: the kit's own module, taken the way a consumer takes it, and the gate.

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

[doc("📁 The kit's recipes: issue, tidy, status (invoke as `just kit <recipe>`)")]
mod kit ".kit/kit.just"

# The gate a pull request passes: shellcheck over every bash file, actionlint over every workflow, the bats
# suite over bin/kit, and bin/kit check over this repository's installed copy of its own kit. Every tool comes
# from mise.toml.
#
#   just ci
[doc("🚦 Run the gate: shellcheck, actionlint, bats, kit check")]
ci:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  mise exec -- shellcheck --shell=bash --external-sources bin/kit .kit/bash/*.sh .kit/hooks/*
  mise exec -- actionlint
  mise exec -- bats --print-output-on-failure tests/
  bin/kit check
  echo "✅ ci passed"

# Reinstall this repository's own kit from its source. Here the source under .kit/ is the payload, so an edit to
# a delivered file is an edit to the release, not a consumer's drift: the installed copies, .claude/settings.json
# (which carries only the kit's entries here), and .kit/manifest are rewritten from it. Run after changing any
# delivered file, and commit the result with the change.
#
#   just refresh
[doc("🔁 Reinstall this repository's own kit from .kit/ and bin/kit")]
refresh:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  rm -f .kit/manifest .claude/settings.json
  bin/kit install claude

# ----------------------------------------------------------------------------------------------------------------------
