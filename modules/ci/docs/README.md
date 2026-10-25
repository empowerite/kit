# The ci module

Places `.github/workflows/kit-ci.yml`: one job, `CI`, whose steps are one kit command each. `kit gate` runs the
repository's `just ci` when its root justfile has that recipe and reports a missing one as a skip, `kit check`
holds every delivered file to the pinned release, and `kit linked` refuses a pull request not linked to an open
issue of the repository. Each is a verb a desk can run, so the runner does nothing a person cannot repeat.

The workflow is a template with one key, `ci_runner`, the label the job runs on. Its default is `ubuntu-latest`;
a repository with runners of its own puts `ci_runner <label>` in `.kit/answers`.
