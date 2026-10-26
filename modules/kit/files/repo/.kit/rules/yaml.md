---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.yml"
  - "**/*.yaml"
---
# YAML

Applies to `*.yml` and `*.yaml`. A GitHub Actions workflow follows the workflows rule as well.

## Shape

1. The file opens with a comment: what the file configures and the one thing a reader would not guess from the
   keys.
2. Two-space indentation. No tabs. No trailing whitespace.
3. A scalar that holds a colon followed by a space, or opens with a reserved character, is quoted.

## Comments

1. A comment states the non-obvious why. It carries no history.
2. A pointer is `# see: <ref>`.
