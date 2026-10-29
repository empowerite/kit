---
# Added by kit's style-workflows package. To remove it, run kit package remove style-workflows.
# To change it, do not edit it here: see "Changing what kit adds" in .kit/1st.md.
paths:
  - ".github/workflows/*.yml"
  - ".github/workflows/*.yaml"
templates:
  - variant: workflow
---
# Workflows style

Applies to GitHub Actions workflows under `.github/workflows/`. The rule, `.kit/rules/workflows.md`, also applies,
and so does YAML style when the repository takes it.

## Shape

1. The file opens with a comment: what the workflow verifies or produces, when it runs, and the one thing a
   reader would not guess from the keys.
2. Keys are in the order GitHub documents them: `name`, `on`, `permissions`, `jobs`.
3. Use one job per scope and one step per policy.

## Steps

1. Every step has a `name`: the policy it enforces, in kebab-case.
2. A refusal prints `::error::<message>` naming the remedy, and returns non-zero. If a reader will act on the
   refusal, it also writes `### ❌ <Message>` and the remedy to `${GITHUB_STEP_SUMMARY}`.
3. A step whose result a reader looks for writes one `✅` line to the summary.
