# The ci module

Places `.github/workflows/kit-ci.yml`: one job, `CI`, which runs the repository's `just ci` when its root justfile
has that recipe, runs `kit check`, and refuses a pull request not linked to an open issue of the repository.

The workflow is a template with one key, `ci_runner`, the label the job runs on. Its default is `ubuntu-latest`;
a repository with runners of its own puts `ci_runner <label>` in `.kit/answers`.
