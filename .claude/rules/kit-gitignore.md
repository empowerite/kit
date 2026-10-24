---
# Placed by kit install from .kit/rules/gitignore.md, not edited by hand.
paths:
  - ".gitignore"
  - "**/.gitignore"
---
# Ignore lists

Applies to `.gitignore` at any depth.

## Shape

1. The file opens with `# .gitignore - untracked files and folders`, a rule of dashes to column 120, and a
   blank line.
2. Entries are grouped. Each group opens with `# <what produces it>; see: <ref>` and is followed by a blank
   line.
3. The file closes with a rule of dashes.

## Entries

1. Every entry names a file the repository itself produces. Editor and operating-system files belong in the
   user's global ignore list, not here.
2. An entry lives in the deepest directory where it applies. The root carries only patterns that span the
   repository.
3. A pattern that names a directory ends with `/`. A pattern anchored to the file's directory starts with `/`.
4. `worktrees/` is ignored at the root: it is where the loop checks out a branch per issue.

## Adding one

Ask what produces the file. If nothing in the repository does, do not add it.
