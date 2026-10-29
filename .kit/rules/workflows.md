---
# Delivered by kit, not edited by hand. A change is a new release, then kit update.
paths:
  - ".github/workflows/*.yml"
  - ".github/workflows/*.yaml"
lint: actionlint
---
# Workflows

Applies to GitHub Actions workflows under `.github/workflows/`.

1. A job's key is the check name a ruleset requires, so renaming the job breaks the requirement.
2. Declare `permissions` at the top, granting the least the jobs need.
3. Every `uses:` pins a full commit SHA, with the version as a trailing comment: `@<sha>  # v1.2.3`.
4. A `run:` block is bash and follows the bash rule: it sets `errexit`, `nounset`, and `pipefail` first.
5. A later step that must run after an earlier refusal has `if: ${{ !cancelled() }}`, so one failure does not
   hide the others.
6. A tool gets on `PATH` because the repository pins it and a step installs it, never through `curl` in a step.
   The one exception is the kit itself. The delivered job downloads the release that `.kit/manifest` names,
   because nothing else can install the kit first.
