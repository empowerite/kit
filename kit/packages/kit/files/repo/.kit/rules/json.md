---
# kit:generated -- the json rule, from the kit package
# see: "Changing what kit adds" in .kit/1st.md, to replace it
kind:
  - json
---
# JSON

Applies to `*.json` and `*.jsonc`.

1. A disabled linter rule has a comment that names the rule and the convention it conflicts with.
2. In a merged settings file, do not edit an entry the kit wrote. Change it through a release and `kit update`.
   The repository may edit everything else in the file.
