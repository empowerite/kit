---
# kit:generated -- the yaml style, from the style/yaml package
# see: kit packages remove style/yaml, or "Changing what kit adds" in .kit/1st.md
kind:
  - yaml
templates:
  - variant: document
---
# YAML style

Applies to `*.yml` and `*.yaml`.

1. The file opens with a comment: what it configures, and the one thing a reader would not guess from the keys.
2. A comment states the non-obvious reason. It contains no history.
3. A pointer is `# see: <ref>`.
