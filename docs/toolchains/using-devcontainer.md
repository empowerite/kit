# Using a dev container

A [dev container](https://containers.dev) is configured by `.devcontainer/devcontainer.json`, or `.devcontainer.json`,
at the repository root. For how kit finds and chooses a manager, see [toolchains.md](../toolchains.md).

devcontainer fills the project role. Inside the container, it fills the system role too.

## Set up

1. Write `.devcontainer/devcontainer.json`, with the tools as an image or as features.
2. From outside the container, install the CLI: `npm install --global @devcontainers/cli`.
3. Run `kit tools install`. It runs `devcontainer up`, which brings the container up.
4. Commit the file, and `devcontainer-lock.json` beside it once the CLI writes one. Until then, kit lists no pins.

## What kit reads

- **Usable**: from outside, the `devcontainer` CLI is on `PATH` and the file is here. Inside, the file alone.
- **Active**: `REMOTE_CONTAINERS`, `CODESPACES`, or `DEVCONTAINER` is set, and the file is here.
- **Pins**: each feature in `devcontainer-lock.json`, by its locked version.

| `kit tools` | What kit does |
| --- | --- |
| `install` | `devcontainer up`, or nothing inside the container |
| `update` | `devcontainer upgrade` |
| `--outdated` | `devcontainer outdated`, which asks the registry |
| `run` | From outside, `devcontainer exec`; inside, the command as is |

- It is offered last, after docker.
- Inside the container it is active, so it is the one chosen for both roles.
- From outside, it runs commands in the container, as docker does. Run `kit tools install` first, so the container
  is up. `kit gate` runs the gate there too.
- The file allows comments, which kit does not parse. The pins come from the lock, so a feature not yet locked is
  not listed.
- brew never fills in for devcontainer, because its tools never reach the container.

## On a runner

Set `ci_toolchain: devcontainer` under `answers:` in `kit.yml`. The workflow installs the CLI with npm, so the
runner needs Node and npm. GitHub's hosted runners have both; see [packages/github/ci.md](../packages/github/ci.md).
