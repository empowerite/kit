---
# Added by kit's style-yaml package. To remove it, run kit package remove style-yaml.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - "**/*.yml"
  - "**/*.yaml"
templates:
  - variant: document
---
# YAML style

Applies to `*.yml` and `*.yaml`.

1. The file opens with a comment: what it configures, and the one thing a reader would not guess from the keys.
2. A comment states the non-obvious reason. It contains no history.
3. A pointer is `# see: <ref>`.
