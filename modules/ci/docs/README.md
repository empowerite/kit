# The ci module

Places `.github/workflows/kit-ci.yml`: one job, `CI`, run for a pull request into the default branch. After a
step that confirms `kit` is on `PATH`, three checks, each one kit command: `kit gate` runs the repository's
`just ci` and refuses when the root justfile has no `ci` recipe, `kit check` holds every delivered file to the
pinned release, and `kit linked` refuses a pull request not linked to an open issue of the repository. Each is a
verb a desk can run, so the runner does nothing a person cannot repeat.

The workflow is a template with one key, `ci_runner`, the label the job runs on. Its default is `ubuntu-latest`;
a repository with runners of its own sets `ci_runner` under `answers:` in `.kit/settings.yml`.
