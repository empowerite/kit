# The ci-github module

Places `.github/workflows/kit-ci.yml`: one GitHub Actions job, `CI`, run for a pull request into the default branch. It
fetches full history, which `kit policy` reads; puts `kit` on `PATH`, from the kit's own checkout where `.kit/manifest`
says `unreleased` and from the release the manifest names otherwise; installs the gate's tools with `kit toolchain
install`, whichever of mise, Homebrew, or a container image manages them; and runs `kit ci`: the gate, `kit check`, `kit
linked`, and `kit policy`, every one reported.

Its one setup action, `jdx/mise-action` with its own install turned off, puts the `mise` binary on the runner where the
repository has a `mise.toml`; Homebrew and Docker are on GitHub's hosted runners already. A repository on another CI
calls `kit toolchain install` and `kit ci` from its own pipeline.

The workflow is a template with one key, `ci_runner`, the label the job runs on. Its default is `ubuntu-latest`;
a repository with runners of its own sets `ci_runner` under `answers:` in `.kit/settings.yml`.
