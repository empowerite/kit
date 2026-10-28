---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.json"
  - "**/*.jsonc"
---
# JSON

Applies to `*.json` and `*.jsonc`.

## Shape

1. Indent with two spaces. Keep keys in a stable order: as the consuming tool documents, else alphabetical.
2. End with one trailing newline. No trailing commas in `.json`.
3. Where the consuming tool reads JSONC, name the file `.jsonc` and comment every non-obvious setting. Where it
   reads plain JSON, explain a setting in the document that describes the file.

## Settings

1. A disabled linter rule has a comment that names the rule and the convention it conflicts with.
2. In a merged settings file, do not edit an entry the kit wrote. Change it through a release and `kit update`.
   The repository may edit everything else in the file.

## Comments

1. A comment states what the entry does and any non-obvious reason for it. It contains no history.
2. A pointer is `// see: <ref>`.
