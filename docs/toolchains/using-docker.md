# Using a Docker image

A `toolchain.Dockerfile` at the repository root builds an image with the tools in it. kit runs each command in a
container of that image. For how kit finds and chooses a manager, see [toolchains.md](../toolchains.md).

docker fills the project role. It is never active: kit runs a command in its container instead.

## Set up

1. Install Docker with buildx, and check that `docker` runs.
2. Write `toolchain.Dockerfile`. Pin each base image by digest, and install bash, because kit runs commands as
   `bash -c`:

   ```dockerfile
   FROM debian:13@sha256:<digest>
   RUN apt-get update && apt-get install --yes bash jq shellcheck
   ```

3. Run `kit tools install`. It builds the image, named `kit-toolchain` and tagged with the file's digest.
4. Commit `toolchain.Dockerfile`.

## What kit reads

- **Usable**: `docker` runs, and `toolchain.Dockerfile` is here.
- **Pins**: each base image by digest, `FROM name:tag@sha256:…`.

| `kit tools` | What kit does |
| --- | --- |
| `install` | `docker build`, unless an image with the file's digest exists. An unchanged file is never rebuilt. |
| `update` | Rewrites each outdated base's digest, then builds. When the registry does not answer for a base, it fails and does not change any pin. |
| `--outdated` | Asks the registry for each tag's current digest. It lists every base pinned to an older digest, or not pinned. |
| `path` | Prints nothing: the tools are in the image. |
| `run` | Runs the command in a container. |

- `kit tools` asks whether the image is built only when the daemon is local. With a remote daemon, no base shows as
  installed.
- A command runs with the checkout mounted at `/work`, as the caller's user, so the files it writes belong to the
  caller. `kit gate` runs the gate that way.
- brew never fills in for docker, because its tools never reach the container.

## On a runner

GitHub's runners have docker. Set `ci_toolchain: docker` under `answers:` in `kit.yml`; see
[packages/github/ci.md](../packages/github/ci.md).
