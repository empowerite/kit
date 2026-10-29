---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "justfile"
  - "**/justfile"
  - "**/*.just"
content:
  - "^#!.*/(env +)?just( |$)"
---
# Just

Applies to `justfile` and `*.just`. A `[script]` recipe body is bash, so the bash rule also applies to it.

1. A recipe runs the repository's pinned tools, the same ones its CI runs, never another copy that happens to be
   first on `PATH`.
2. A `[script]` body sets `errexit`, `nounset`, and `pipefail` first, so a failure stops the recipe.
3. Use no `{{...}}` interpolation in a body. A value interpolated into shell breaks on a space or a quote, and can
   run as a command. Pass a value that `just` computes as an exported variable, `export <name> := <expression>`,
   and refer to it in the body only as `${name}`. `set export := true` exports every parameter.
4. A recipe that changes a remote or a worktree names what it is about to do before doing it.
