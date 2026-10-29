---
# Added by kit's style/gitignore package. To remove it, run kit package remove style/gitignore.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - ".gitignore"
  - "**/.gitignore"
templates:
  - variant: file
  - variant: region
    path: .gitignore
---
# Ignore list style

Applies to `.gitignore` at any depth. The rule, `.kit/rules/gitignore.md`, also applies.

## Shape

The templates in `.kit/templates/gitignore/` hold the exact shape. A repository can replace them with templates of its
own.

1. A file the repository writes opens with `# .gitignore - untracked files and folders` and a blank line.
2. Group the entries. Each group opens with `# <what produces it>; see: <ref>` and ends with a blank line.

## Entries

1. Every entry names a file the repository itself produces. Editor and operating-system files belong in the
   user's global ignore list, not here.
2. Put an entry in the deepest directory where it applies. The root `.gitignore` contains only patterns that apply
   across the whole repository.

## Adding an entry

Ask what produces the file. If nothing in the repository does, do not add it.
