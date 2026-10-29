---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - ".gitignore"
  - "**/.gitignore"
templates:
  - variant: file
  - variant: region
    path: .gitignore
---
# Ignore lists

Applies to `.gitignore` at any depth.

## Shape

The templates in `.kit/templates/gitignore/` hold the exact shape. A repository can replace them with templates of its
own.

1. A file the repository writes opens with `# .gitignore - untracked files and folders` and a blank line. A file
   the kit created that contains only the kit's region belongs to the kit until the repository adds to it.
2. Group the entries. Each group opens with `# <what produces it>; see: <ref>` and ends with a blank line.

## Entries

1. Every entry names a file the repository itself produces. Editor and operating-system files belong in the
   user's global ignore list, not here.
2. Put an entry in the deepest directory where it applies. The root `.gitignore` contains only patterns that apply
   across the whole repository.
3. A pattern that names a directory ends with `/`. A pattern anchored to the file's directory starts with `/`.
4. Ignore `/worktrees/` at the root. The workflow checks out a branch for each issue there.

## Adding an entry

Ask what produces the file. If nothing in the repository does, do not add it.
