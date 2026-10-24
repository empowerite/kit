# kit.just - see: https://just.systems/man/en/
# ----------------------------------------------------------------------------------------------------------------------
# The kit's just module. A consumer's root justfile takes it with `mod kit ".kit/kit.just"`, which gives
# `just kit issue`, `just kit tidy`, and `just kit status`. Each recipe carries [no-cd] so its body runs
# from the checkout it was invoked in, and the bash behind each lives in .kit/bash/, loaded through BASH_ENV.
#
# A root justfile that wants the bare spelling adds its own veneer:
#
#   issue *numbers:
#     just kit issue "$@"

set dotenv-load := true
set fallback := true
set positional-arguments := true

export BASH_ENV := source_directory() / "bash/1st.sh"

# This file's own path, assigned at parse time so the body below needs no interpolation.
export kit_justfile := source_file()

# Default recipe: list this justfile's recipes.
_: help

# List recipes in this justfile.
[no-exit-message]
@help *options:
  just --justfile "${kit_justfile}" --list

set export := true

# Start, or return to, work on an issue: a branch linked to it under Development, a worktree at
# worktrees/<N>-<slug>, a start commit, and a draft pull request. Further numbers are secondary issues the
# pull request closes. Safe to re-run; it never switches the checkout it runs from.
#
#   just kit issue 42        # start #42
#   just kit issue 42 43     # start #42, and close #43 with it
#   just kit issue           # ask for a title, create the issue, and start it (needs a terminal)
[doc("🌿 Start or resume an issue: linked branch, worktree, start commit, draft pull request")]
[no-cd]
issue *numbers:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  gh:issue "$@"

# Sweep the checkout back to pristine: remove worktrees whose pull request is merged or closed, prune the
# bookkeeping, and delete local branches whose upstream is gone. The default branch, a dirty worktree, and the
# worktree this runs in are never touched.
#
#   just kit tidy            # sweep
#   just kit tidy --dry-run  # say what a sweep would do
[doc("🧹 Remove merged worktrees and gone branches")]
[no-cd]
tidy *flags:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  git:tidy "$@"

# Show the main checkout and every worktree under worktrees/: branch, pull request, and dirty or clean. Reads
# only.
#
#   just kit status            # fetch --prune, then render
#   just kit status --no-fetch # render from the current remote-tracking refs
[doc("📊 Show every worktree with its branch, pull request, and state")]
[no-cd]
status *flags:
  #!/usr/bin/env bash
  set -o errexit -o nounset -o pipefail
  git:status "$@"

# ----------------------------------------------------------------------------------------------------------------------
