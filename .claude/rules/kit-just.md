---
paths:
  - "justfile"
  - "**/justfile"
  - "**/*.just"
---
# Just

Applies to `justfile` and `*.just`. A recipe body under `#!/usr/bin/env bash` is bash, and the bash rule
applies to it as well.

## Shape

1. A module file opens with `# <file> - see: https://just.systems/man/en/`, a rule of dashes to column 120, and
   a description of what the file is and who imports it.
2. The settings block follows: `set dotenv-load := true`, `set fallback := true` in a module (never in the
   root, which has no parent), `set positional-arguments := true`, and `set export := true`.
3. Every justfile carries the default-and-help pair:

   ```just
   # Default recipe: list this justfile's recipes.
   _: help

   # List recipes in this justfile.
   [no-exit-message]
   @help *options:
     just --justfile "{{source_file()}}" --list
   ```

4. A module the root takes with `mod <name> "<path>"` carries `[no-cd]` on every recipe, so its body runs from
   the invoking root rather than the module's directory.
5. The file closes with a rule of dashes.

## Recipes

1. Every recipe carries `[doc("<emoji> <short, action-first description>")]`. Related recipes share an emoji.
2. A comment block precedes every recipe: usage lines, then a paragraph of intent. It answers what the recipe
   does, when to run it, and what it produces.
3. A non-trivial body is `#!/usr/bin/env bash` with `set -o errexit -o nounset -o pipefail` as its first line.
4. Non-trivial logic lives in a bash module the recipe loads, not in the body. The body stays a few lines.
5. Recipe bodies use `${var}`, never `{{var}}` mixed in. `set export := true` makes parameters available.
6. Two-space indentation inside a body. Never a tab.
7. Recipe and module names are spelled out, never abbreviated. A second `mod` at the same target is an alias.
8. Comments wrap at 118 columns. Bodies do not wrap.

## Invocation

1. The idiom is the space form, `just kit issue 42`, never `just kit/issue` and never `kit::issue` outside a
   dependency list.
2. A recipe that mutates a remote or a worktree names what it is about to do before doing it.
