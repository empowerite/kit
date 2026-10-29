---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - ".gitignore"
  - "**/.gitignore"
---
# Ignore lists

Applies to `.gitignore` at any depth.

1. Ignore `/worktrees/` at the root. The workflow checks out a branch for each issue there.
2. A file the kit created that contains only the kit's region belongs to the kit until the repository adds to it.
