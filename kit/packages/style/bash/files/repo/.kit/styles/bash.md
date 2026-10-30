---
# Added by kit's style/bash package. To remove it, run kit package remove style/bash.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - "**/*.sh"
  - "**/*.bash"
content:
  - "^#!.*/(env +)?(ba)?sh( |$)"
templates:
  - variant: script
    executable: true
  - variant: module
---
# Bash style

Applies where the bash rule applies. The rule, `.kit/rules/bash.md`, says what a script must do. This file says
how it looks.

## Shape

The templates in `.kit/templates/bash/` hold the exact shape. `kit new file` makes a file from them, and `kit lint`
checks a file against them. A repository can replace them with templates of its own.

1. An executed script opens with these lines, in order:
   1. `#!/usr/bin/env bash`
   2. a `# <file> - <purpose>` banner under 120 columns; the purpose is in the indicative, what the file does, as
      `kit help` prints a verb's
   3. a blank line
   4. an optional comment paragraph
   5. `set -o errexit -o nounset -o pipefail`
2. A sourced module opens with `# shellcheck shell=bash`, then the same banner.

## Style

1. `then`, `do`, and `else` go on their own line, never after a semicolon.
2. Spell `set` options out: `set -o errexit`, never `set -e`.
3. Use a tool's long options where every platform accepts them: `--raw-output`, `--field`, `--quiet`.
4. Brace every variable longer than one character: `${name}`. `$1`, `$@`, `$?`, and `$#` stay bare.
5. A `case` label has both parentheses: `(vanilla)`. The last group has no `;;`.
6. Name a function private to one executed script in bare kebab-case.

## Behavior

1. An error message is one line. It names the remedy, if any. The same condition gives the same message everywhere.
2. A helper that refuses returns non-zero and lets the caller end the script. It does not call `exit` itself.

## Comments

1. A comment states what the code does and any non-obvious reason for it. It contains no history and does not
   defend an obvious choice.
2. A pointer is `# see: <ref>`, on its own line after the comment it supports.
