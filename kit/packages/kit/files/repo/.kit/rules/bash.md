---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.sh"
  - "**/*.bash"
content:
  - "^#!.*/(env +)?(ba)?sh( |$)"
lint: shellcheck --shell=bash --external-sources
templates:
  - variant: script
    executable: true
  - variant: module
---
# Bash

Applies to:

- `*.sh` and `*.bash` files
- recipe bodies under `#!/usr/bin/env bash`
- `run:` blocks in workflows

## Shape

1. An executed script opens with these lines, in order:
   1. `#!/usr/bin/env bash`
   2. a `# <file> - <purpose>` banner under 120 columns
   3. a line of dashes to column 120
   4. a blank line
   5. an optional comment paragraph
   6. `set -o errexit -o nounset -o pipefail`
2. A sourced module opens with `# shellcheck shell=bash`, then the same banner. It sets no shell options, because
   they would change the caller's options.
3. The file ends with the same line of dashes.

## Style

1. `then`, `do`, and `else` go on their own line, never after a semicolon.
2. Spell `set` options out: `set -o errexit`, never `set -e`.
3. Use a tool's long options where every platform accepts them: `--raw-output`, `--field`, `--quiet`. Where BSD
   and GNU differ, use the short form, which both accept: `sed -n`, `head -n 1`, `mkdir -p`, `rm -f`, `cut -d`,
   `cp -R`, `find -type`, `read -r`, and `git -C`.
4. Arithmetic uses `(( ))`. String and file tests use `[[ ]]`.
5. Brace every variable longer than one character: `${name}`. `$1`, `$@`, `$?`, and `$#` stay bare.
6. A `case` label has both parentheses: `(vanilla)`. The last group has no `;;`.
7. Quote every expansion outside `[[ ]]` and `(( ))`.
8. Name a function in a sourced module `<module>:<kebab-name>`, where `<module>` is the file's basename. The
   loader, `1st.sh`, is the exception. Its verbs have bare names (`use`, `run`, `info`, `step`, `ok`, `section`,
   `warn`, `error`, and `summary`), and its helpers use the `use:`, `kit:`, `out:`, `guard:`, and `ere:` prefixes.
   `super.sh` adds the bare `super`.
9. Name a function private to one executed script in bare kebab-case.

## Behavior

1. An error message is one line. It names the remedy, if any. The same condition gives the same message everywhere.
2. A helper that refuses returns non-zero and lets the caller end the script. It does not call `exit` itself.
3. A secret never appears in `argv`. Pipe it: `printf '%s' "${secret}" | gh secret set NAME`.
4. Diagnostics go to stderr. Stdout is for data a caller can pipe.
5. A command that changes a remote, a branch, or a worktree is guarded by a check that names what it is about to
   do. If the change removes work, as a sweep of worktrees or branches does, the command also has `--dry-run`.
6. Read the default branch from `origin/HEAD`, with `main` as the fallback. Never hard-code it.
7. shellcheck passes with no directive. Where a directive is needed, its reason is on the same line.

## Comments

1. A comment states what the code does and any non-obvious reason for it. It contains no history and does not
   defend an obvious choice.
2. A pointer is `# see: <ref>`, on its own line after the comment it supports.
