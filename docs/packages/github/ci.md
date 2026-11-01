# The github/ci package

Delivers `.github/workflows/kit-ci.yml`: one GitHub Actions job, `CI`, that runs for each pull request into the
default branch. The job:

1. Fetches the full history, which `kit policy` reads.
2. Puts `kit` on `PATH`. Where `kit.yml` declares `unreleased`, it uses the kit's own checkout. Otherwise it
   downloads the release `kit.yml` declares with the `.sha256` published beside it, and unpacks the tarball only when
   its bytes have that digest.
3. Installs the gate's tools with `kit toolchains install`, whether mise, Homebrew, or a container image manages
   them. The gate is the repository's own CI command, such as `just ci`.
4. Runs `kit ci`, which runs the gate, `kit check`, `kit linked`, and `kit policy`, and reports each one.

GitHub's hosted runners already have Homebrew and Docker. The workflow's one setup action, `jdx/mise-action`
with its own install turned off, puts `mise` on the runner when the repository has a `mise.toml` or `.mise.toml`.
The job needs `jq` and `yq` on the runner, which the hosted runners have; a runner of the repository's own needs both.

A repository on another CI system calls `kit toolchains install` and `kit ci` from its own pipeline.

The workflow is a template with one key, `ci_runner`: the label of the runner the job runs on. The default is
`ubuntu-latest`. A repository with its own runners sets `ci_runner` under `answers:` in `kit.yml`.
