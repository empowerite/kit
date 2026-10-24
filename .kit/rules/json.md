---
paths:
  - "**/*.json"
  - "**/*.jsonc"
---
# JSON

Applies to `*.json` and `*.jsonc`.

## Shape

1. Two-space indentation. Keys in a stable order: the order the consuming tool documents, else alphabetical.
2. One trailing newline. No trailing commas in `.json`.
3. Where the consuming tool reads JSONC, name the file `.jsonc` and comment every non-obvious setting. Where
   it reads plain JSON, the reason for a setting lives in the document that describes the file.

## Settings

1. A disabled linter rule carries a comment naming the rule and the convention it would fight.
2. A tool's settings file that the kit writes into is merged by key. The kit owns only the keys and entries
   it writes, and anything else in the file is the repository's.

## Comments

1. A comment states what the entry does and the non-obvious why. It carries no history.
2. A pointer is `// see: <ref>`.
