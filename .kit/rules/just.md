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

Applies to `justfile` and `*.just`. A recipe body under `#!/usr/bin/env bash` is bash, and the bash rule
applies to it as well.

## Shape

1. Every justfile opens with `# <file> - see: https://just.systems/man/en/`, a rule of dashes to column 120,
   and a description of what the file is and who imports it. The root carries the `#!/usr/bin/env just
   --justfile` shebang above that; a module does not.
2. The settings block follows: `set dotenv-load := true`, `set fallback := true` in a module (never in the
   root, which has no parent), and `set positional-arguments := true`. `set export := true` comes after the
   default-and-help pair, so that pair reads the same in every justfile.
3. Every justfile carries the default-and-help pair, with the file's own path in an exported variable:

   ```just
   # This file's own path, assigned at parse time so the body below needs no interpolation.
   export <name>_justfile := source_file()

   # Default recipe: list this justfile's recipes.
   _: help

   # List recipes in this justfile.
   [no-exit-message]
   @help *options:
     just --justfile "${<name>_justfile}" --list
   ```

4. A module the root takes with `mod <name> "<path>"` carries `[no-cd]` on every recipe, so its body runs from
   the invoking root rather than the module's directory.
5. The file closes with a rule of dashes.

## Recipes

1. Every recipe but the default-and-help pair carries `[doc("<emoji> <short, action-first description>")]`.
   Related recipes share an emoji.
2. A comment block precedes every recipe: a paragraph of intent, then the usage lines. It answers what the
   recipe does, when to run it, and what it produces.
3. A non-trivial body is `#!/usr/bin/env bash` with `set -o errexit -o nounset -o pipefail` as its first line.
4. A body says what runs. Logic longer than a dozen lines lives in a bash script the body calls.
5. No `{{...}}` interpolation anywhere, in a body or a comment. A value just computes reaches a body as an exported
   variable, `export <name> := <expression>`, and `set export := true` exports every parameter, so a body reads
   `${name}` and nothing else.
6. Two-space indentation inside a body. Never a tab.
7. Recipe and module names are spelled out, never abbreviated. A second `mod` at the same target is an alias.
8. Comments wrap at 118 columns. Bodies do not wrap.

## Invocation

1. The idiom is the space form, `just tools check`, never `just tools/check` and never `tools::check` outside a
   dependency list.
2. A recipe that mutates a remote or a worktree names what it is about to do before doing it.
