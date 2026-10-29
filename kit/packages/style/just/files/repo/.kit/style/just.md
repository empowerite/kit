---
# Added by kit's style/just package. To remove it, run kit package remove style/just.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - "justfile"
  - "**/justfile"
  - "**/*.just"
content:
  - "^#!.*/(env +)?just( |$)"
templates:
  - variant: root
    path: justfile
  - variant: module
---
# Just style

Applies to `justfile` and `*.just`. The rule, `.kit/rules/just.md`, also applies.

## Shape

The templates in `.kit/templates/just/` hold the exact shape. A repository can replace them with templates of its own.

1. Every justfile opens with `# <file> - see: https://just.systems/man/en/` and a description of what the file is
   and who imports it. The root justfile has the `#!/usr/bin/env just --justfile` shebang above those lines. A
   module does not.
2. The settings block comes next:
   - `set dotenv-load := true`
   - `set fallback := true`, only in a module, never in the root, because the root has no parent
   - `set positional-arguments := true`
   - `set script-interpreter := ['bash']`, with the interpreter named bare, so the `bash` first on `PATH` runs it

   `set export := true` comes after the default-and-help pair, so that the pair is identical in every justfile.
3. Where mise pins the tools, the root justfile puts them first on `PATH`, once, at parse time, from
   `mise bin-paths`. Every recipe then calls its tools by bare name:

   ````just
   export PATH := ```
     mise bin-paths 2>/dev/null | while IFS= read -r d
     do
       case ":${PATH}:" in (*":${d}:"*) ;; (*) printf '%s:' "${d}" ;; esac
     done
     ``` + env('PATH')
   ````

4. Every justfile has the default-and-help pair, with the file's own path in an exported variable:

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

5. In a module the root imports with `mod <name> "<path>"`, every recipe has `[no-cd]`. The recipe body then
   runs from the invoking root, not from the module's directory.

## Recipes

1. Every recipe except the default-and-help pair has `[doc("<emoji> <short, action-first description>")]`.
   Related recipes share an emoji.
2. A comment block precedes every recipe: a paragraph of intent, then the usage lines. The comment states what
   the recipe does, when to run it, and what it produces.
3. Write a non-trivial body as a `[script]` recipe, never a shebang recipe. Its first line is
   `set -o errexit -o nounset -o pipefail`. Call a tool in it by bare name, never through `mise exec`.
4. A body shows what runs. Move logic longer than a dozen lines into a bash script that the body calls.
5. Use no `{{...}}` interpolation in a comment either.
6. Indent a body with two spaces. Never use a tab.
7. Spell out recipe and module names. Never abbreviate them. A second `mod` at the same target is an alias.
8. Comments wrap at 118 columns. Bodies do not wrap.

## Invocation

1. Call a module's recipe with spaces: `just tools check`. Never write `just tools/check`, and never write
   `tools::check` outside a dependency list.
