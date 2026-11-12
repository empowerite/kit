---
# kit:generated -- the POSIX shell rule, from the kit package
# see: "Changing what kit adds" in .kit/1st.md, to replace it
kind:
  - sh
lint: shellcheck --shell=sh
---
# POSIX shell

Applies to files of kind `sh`: a script whose first line is `#!/bin/sh` or `#!/usr/bin/env sh`.

Write POSIX shell only where bash cannot be counted on, as in a loader that must run before bash 4.4 is installed.
Everywhere else, write bash, under `.kit/rules/bash.md`.

## Running

1. An executed script starts with `#!/bin/sh`, and sets `errexit` and `nounset` before anything else. POSIX shells
   before 2024 have no `pipefail`, so check the status of each command in a pipeline that can fail.
2. Use POSIX alone. A script has no `[[ ]]`, arrays, `local`, `(( ))` commands, `$'…'`, `printf -v`, `source`, or
   `function`. `shellcheck --shell=sh` refuses each.
3. A function's name is a plain name, letters, digits, and `_`, because POSIX allows no other, so a function cannot
   be named `<module>:<name>` as a bash module's are.
4. Where BSD and GNU tools differ, use the form both accept, as the bash rule lists them. A tool outside POSIX, such
   as `readlink`, is used only with a comment that says every system it runs on has it.

## Behavior

1. Diagnostics go to stderr. Stdout is for data a caller can pipe.
2. shellcheck passes with no directive. Where a directive is needed, its reason is on the same line.
