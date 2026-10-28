---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.yml"
  - "**/*.yaml"
templates:
  - variant: document
---
# YAML

Applies to `*.yml` and `*.yaml`. A GitHub Actions workflow also follows the workflows rule.

## Shape

1. The file opens with a comment: what it configures, and the one thing a reader would not guess from the keys.
2. Indent with two spaces. No tabs. No trailing whitespace.
3. Quote a scalar that contains a colon followed by a space, or that starts with a reserved character.

## Comments

1. A comment states the non-obvious reason. It contains no history.
2. A pointer is `# see: <ref>`.
