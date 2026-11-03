# The github/ci package

Delivers `.github/workflows/kit-ci.yml`: one GitHub Actions job, `CI`, that runs for each pull request into the
default branch. The job:

1. Fetches the full history, which `kit policy` reads.
2. Puts `kit` on `PATH`. Where `kit.yml` declares `unreleased`, it uses the kit's own checkout. Otherwise it
   downloads the release `kit.yml` declares with the `.sha256` published beside it, and unpacks the tarball only when
   its bytes have that digest.
3. Installs the gate's tools with `kit tools install`, through the toolchain `ci_toolchain` names: mise, nix,
   Homebrew, a Docker image, a dev container, or the tools already on the runner. The gate is the repository's own
   CI command, such as `just ci`.
4. Runs `kit ci`, which runs the gate, `kit check`, `kit linked`, and `kit policy`, and reports each one.

GitHub's hosted runners already have Homebrew and Docker. The job sets `KIT_TOOLCHAIN` to the `ci_toolchain` answer, so
kit uses that toolchain and refuses, never replaces, one the runner cannot use. One bootstrap step runs for the
toolchain named, and none for the others:

| `ci_toolchain` | Step | What it does |
| --- | --- | --- |
| `mise` | `mise-on-path` | `jdx/mise-action`, pinned by commit, with its own install turned off |
| `nix` | `nix-on-path` | `DeterminateSystems/nix-installer-action`, pinned by commit |
| `devcontainer` | `devcontainer-on-path` | `npm install --global @devcontainers/cli`, exact version; needs npm |
| `brew`, `docker`, `ambient` | none | the runner already has them, or the tools are on `PATH` |

The job needs `jq` and `yq` on the runner, which the hosted runners have; a runner of the repository's own needs both.

A repository on another CI system calls `kit tools install` and `kit ci` from its own pipeline.

The workflow is a template with three keys, all under `answers:` in `kit.yml`:

- `ci_runner` is the label of the runner the job runs on. The default is `ubuntu-latest`. A repository with its own
  runners sets it.
- `ci_timeout_minutes` is how long the job may run before GitHub stops it and fails it. The default is `"30"`.
  Without it, a hung step runs for GitHub's limit of six hours. Set it to about twice a usual run, in quotes:
  `ci_timeout_minutes: "20"`. This repository sets `"60"`.
- `ci_toolchain` is the toolchain the job installs with. The default is `ambient`: kit does not install anything. The
  job fails at the `toolchain` step, naming each tool the rules need that is missing, unless the runner already has
  them. To have CI install the tools, name a toolchain, such as `ci_toolchain: mise` or `nix`. This repository sets
  `mise`. `ci_toolchain: devcontainer` needs Node and npm on the runner; GitHub's hosted runners have both.
