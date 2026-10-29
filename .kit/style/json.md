---
# Added by kit's style-json package. To remove it, run kit package remove style-json.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - "**/*.json"
  - "**/*.jsonc"
---
# JSON style

Applies to `*.json` and `*.jsonc`. The rule, `.kit/rules/json.md`, also applies.

1. Indent with two spaces. Keep keys in a stable order: as the consuming tool documents, else alphabetical.
2. Where the consuming tool reads JSONC, name the file `.jsonc` and comment every non-obvious setting. Where it
   reads plain JSON, explain a setting in the document that describes the file.
3. A comment states what the entry does and any non-obvious reason for it. It contains no history.
4. A pointer is `// see: <ref>`.
