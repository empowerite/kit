---
# Added by the kit package, which kit always installs.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - "**/*.sh"
  - "**/*.bash"
content:
  - "^#!.*/(env +)?(ba)?sh( |$)"
lint: shellcheck --shell=bash --external-sources
---
# Bash

Applies to:

- `*.sh` and `*.bash` files
- recipe bodies in a justfile that bash runs: `[script]` bodies, and shebang bodies under `#!/usr/bin/env bash`
- `run:` blocks in workflows

## Running

1. An executed script starts with `#!/usr/bin/env bash`, never `#!/bin/bash`. On macOS, `/bin/bash` is bash 3.2.
2. An executed script sets `errexit`, `nounset`, and `pipefail` before anything else, so a failure stops it rather
   than letting it report success.
3. A sourced file sets no shell options, because they would change its caller's options.
4. Where BSD and GNU tools differ, use the form both accept: `sed -n`, `head -n 1`, `mkdir -p`, `rm -f`, `cut -d`,
   `cp -R`, `find -type`, `read -r`, and `git -C`. A script must run the same on a desk and on a CI runner.

## Behavior

1. A secret never appears in `argv`. Pipe it: `printf '%s' "${secret}" | gh secret set NAME`.
2. Diagnostics go to stderr. Stdout is for data a caller can pipe.
3. A command that changes a remote, a branch, or a worktree is guarded by a check that names what it is about to
   do. If the change removes work, as a sweep of worktrees or branches does, the command also has `--dry-run`.
4. Read the default branch from `origin/HEAD`, with `main` as the fallback. Never hard-code it.
5. shellcheck passes with no directive. Where a directive is needed, its reason is on the same line.

## Modules on the kit's module path

A module, a file that `use` loads from the kit's bash module path, is a sourced file. Name each function in it
`<module>:<name>`, where `<module>` is the file's name without `.sh`, so every function says where it is defined.
