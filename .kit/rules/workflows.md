---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - ".github/workflows/*.yml"
  - ".github/workflows/*.yaml"
lint: actionlint
templates:
  - variant: workflow
---
# Workflows

Applies to GitHub Actions workflows under `.github/workflows/`. The YAML rule also applies to them.

## Shape

1. The file opens with a comment: what the workflow verifies or produces, when it runs, and the one thing a
   reader would not guess from the keys.
2. Keys are in the order GitHub documents them: `name`, `on`, `permissions`, `jobs`.
3. Use one job per scope and one step per policy. A job's key is the check name a ruleset requires, so renaming
   the job breaks the requirement.
4. Declare `permissions` at the top, granting the least the jobs need.

## Steps

1. Every `uses:` pins a full commit SHA, with the version as a trailing comment: `@<sha>  # v1.2.3`.
2. Every step has a `name`: the policy it enforces, in kebab-case.
3. A `run:` block is bash and follows the bash rule: `set -o errexit -o nounset -o pipefail` first.
4. A refusal prints `::error::<message>` naming the remedy, and returns non-zero. If a reader will act on the
   refusal, it also writes `### ❌ <Message>` and the remedy to `${GITHUB_STEP_SUMMARY}`.
5. A step whose result a reader looks for writes one `✅` line to the summary.
6. A later step that must run after an earlier refusal has `if: ${{ !cancelled() }}`.
7. A tool gets on `PATH` because the repository pins it and a step installs it, never through `curl` in a step.
   The one exception is the kit itself. The delivered job downloads the release that `.kit/manifest` names,
   because nothing else can install the kit first.
