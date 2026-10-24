---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.sh"
  - "**/*.bash"
  - "**/hooks/*"
---
# Bash

Applies to `*.sh`, `*.bash`, `hooks/*`, recipe bodies under `#!/usr/bin/env bash`, and `run:` blocks in
workflows.

## Shape

1. An executed script opens with `#!/usr/bin/env bash`, a `# <file> - <purpose>` banner, a rule of dashes to
   column 120, and a blank line, then `set -o errexit -o nounset -o pipefail`.
2. A sourced module opens with `# shellcheck shell=bash` and the same banner, and sets no shell options: they
   would change the caller's.
3. A git hook that must fail open sets `-o nounset -o pipefail` and no `errexit`, and says so in its banner.
4. The file closes with the same rule of dashes.

## Style

1. `then`, `do`, and `else` go on their own line, never after a semicolon.
2. Spell `set` options out: `set -o errexit`, never `set -e`.
3. Spell command options out: `--raw-output`, `--field`, `--delimiter`, never a one-letter flag in a script.
   Where BSD and GNU differ, use the form both accept: `sed -n`, `head -n 1`, `mkdir -p`.
4. Arithmetic uses `(( ))`. String and file tests use `[[ ]]`.
5. Brace every variable longer than one character: `${name}`. `$1`, `$@`, `$?`, and `$#` stay bare.
6. A `case` label carries both parentheses: `(vanilla)`. The last group has no `;;`.
7. Quote every expansion outside `[[ ]]` and `(( ))`.
8. A function in a sourced module is named `<module>:<kebab-name>`, the module being the file's basename.
9. A function private to one executed script is bare kebab-case.

## Behavior

1. An error message is one line, names the remedy first, and is the same wherever the same condition arises.
2. A helper that refuses returns non-zero and lets the caller end the script. It does not `exit` itself.
3. A secret never appears in `argv`. Pipe it: `printf '%s' "${secret}" | gh secret set NAME`.
4. Diagnostics go to stderr. Stdout carries data a caller can pipe.
5. A command that mutates a remote, a branch, or a worktree is guarded by a check that names what it is about
   to do, and a `--dry-run` where the mutation is a sweep.
6. The default branch is read from `origin/HEAD`, with `main` as the fallback, never hard-coded.
7. shellcheck passes with no directive, or the directive carries the reason on the same line.

## Comments

1. A comment states what the code does and the non-obvious why. It carries no history and defends no obvious
   choice.
2. A pointer is `# see: <ref>`, on its own line after the comment it supports.
