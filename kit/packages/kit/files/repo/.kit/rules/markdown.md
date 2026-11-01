---
# kit:generated -- the markdown rule, from the kit package
# see: "Changing what kit adds" in .kit/1st.md, to replace it
kind:
  - markdown
---
# Markdown

Applies to `*.md`.

1. Describe the current state. Leave the history to git: no "previously", "used to", "replaced", or "ported from".
2. A relative link resolves from the file that contains it, and it must resolve.
3. `AGENTS.md` opens with its title, then the kit's region, the heading and two sentences a harness reads first,
   before anything of the repository's own. The `agents/generic` package's host template holds that shape, and
   `kit check` holds the file to it.
