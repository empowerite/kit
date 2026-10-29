---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.json"
  - "**/*.jsonc"
---
# JSON

Applies to `*.json` and `*.jsonc`.

1. A disabled linter rule has a comment that names the rule and the convention it conflicts with.
2. In a merged settings file, do not edit an entry the kit wrote. Change it through a release and `kit update`.
   The repository may edit everything else in the file.
