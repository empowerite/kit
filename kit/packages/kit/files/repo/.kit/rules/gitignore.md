---
# kit:generated -- the gitignore rule, from the kit package
# see: "Changing what kit adds" in .kit/1st.md, to replace it
kind:
  - gitignore
---
# Ignore lists

Applies to `.gitignore` at any depth.

1. Ignore `/worktrees/` at the root. The workflow checks out a branch for each issue there.
2. A file the kit created that contains only the kit's region belongs to the kit until the repository adds to it.
