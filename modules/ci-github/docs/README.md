# The ci-github module

Places `.github/workflows/kit-ci.yml`: one GitHub Actions job, `CI`, run for a pull request into the default branch. It
puts `kit` on `PATH`, from the kit's own checkout where `.kit/manifest` says `unreleased` and from the release the
manifest names otherwise, and runs `kit ci`: the gate, `kit check`, and `kit linked`, every one reported.

mise is used only where the repository has a `mise.toml`. A repository that pins its tools another way gives the
gate its tools its own way, and a repository on another CI calls `kit ci` from its own pipeline.

The workflow is a template with one key, `ci_runner`, the label the job runs on. Its default is `ubuntu-latest`;
a repository with runners of its own sets `ci_runner` under `answers:` in `.kit/settings.yml`.
