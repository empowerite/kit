---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - "**/*.yml"
  - "**/*.yaml"
---
# YAML

Applies to `*.yml` and `*.yaml`. Shape and Steps apply to GitHub Actions workflows, under `.github/workflows/`;
Comments apply to every YAML file.

## Shape

1. The file opens with a comment: what the workflow verifies or produces, when it runs, and the one thing a
   reader would not guess from the keys.
2. Two-space indentation. No tabs. No trailing whitespace.
3. Keys are in the order GitHub documents them: `name`, `on`, `permissions`, `jobs`.
4. One job per scope, one step per policy. A job's key is the check's name a ruleset requires, so renaming it
   breaks the requirement.
5. `permissions` is declared at the top and names the least the jobs need.

## Steps

1. Every `uses:` pins a full commit SHA with the version as a trailing comment: `@<sha>  # v1.2.3`.
2. A step has a `name` that is the policy it enforces, in kebab-case.
3. A `run:` block is bash and follows the bash rule: `set -o errexit -o nounset -o pipefail` first.
4. A refusal prints `::error::<message>` naming the remedy, and returns non-zero. A refusal a reader will act on
   also writes `### ❌ <Message>` and the remedy to `${GITHUB_STEP_SUMMARY}`.
5. A step whose verdict a reader looks for writes one `✅` line to the summary.
6. A later step that must run after an earlier refusal carries `if: ${{ !cancelled() }}`.
7. A tool is on `PATH` because `mise.toml` pins it and `jdx/mise-action` installs it, never by `curl` in a step.

## Comments

1. A comment states the non-obvious why. It carries no history.
2. A pointer is `# see: <ref>`.
