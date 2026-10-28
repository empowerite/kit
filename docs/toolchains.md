# Toolchains

A toolchain is the tool manager that installs the tools the gate (the repository's own CI command) runs: mise,
Homebrew, or a Docker container image. `kit toolchain` installs, reports on, and updates those tools the same way
whichever manager the repository uses.

```sh
kit toolchain install
```

`bin/toolchain.sh` defines the interface, and each manager has a provider, `bin/toolchain-<name>.sh`, that
implements it. Every provider answers the same calls, so the root checkout (the desk), a CI runner, and the
delivered CI job report the same results whichever manager the repository uses.

## Declaring

A repository declares its toolchains in `.kit/settings.yml`, in order:

```yaml
toolchains:
  - provider: mise
  - provider: brew
    file: Brewfile
  - provider: docker
    file: toolchain.Dockerfile
    image: example/toolchain
```

- With no declaration, the kit detects toolchains from the files they read, in the order mise, brew, docker.
- `file:` names a different file for brew or docker.
- `image:` names the docker image. The default is `kit-toolchain`.
- A provider the kit does not have is refused.

| Provider | Detected from | Pins | Outdated when |
| --- | --- | --- | --- |
| `mise` | `mise.toml` or `.mise.toml` | each tool's version, resolved in `mise.lock` | `mise outdated` lists it |
| `brew` | `Brewfile` | a formula by name; a version only where the name contains it, `node@22` | `brew outdated` lists it |
| `docker` | `toolchain.Dockerfile` | each base image by digest, `FROM name:tag@sha256:…` | the registry's digest for the tag differs, or there is none |

## Commands

| Command | What it does |
| --- | --- |
| `kit toolchain list` | Prints each declared toolchain and the file it reads. |
| `kit toolchain install` | Installs every toolchain's pinned tools: `mise install`, `brew bundle install --no-upgrade`, or a `docker build` of the image. The image is tagged by the file's digest, so an unchanged file is never rebuilt. |
| `kit toolchain outdated` | Prints one line per tool that is behind its source. With `--check`, fails when there is one. |
| `kit toolchain update` | Moves every tool to its latest version and rewrites the pins: `mise upgrade`, `brew upgrade` of the Brewfile's outdated formulae, or new `FROM` digests in the Dockerfile. Commit the files it rewrote. |
| `kit toolchain path` | Prints the directories that put the tools first on `PATH`, for a runner's `GITHUB_PATH`. |
| `kit toolchain run <command>…` | Runs a command with the tools. |

`kit gate` runs the gate the same way:

- Normally, with every toolchain's directories first on `PATH`.
- When docker is declared, inside the image, with the checkout mounted at `/work` and run as the caller's user, so
  the files the gate writes belong to the caller. The image needs `bash`, because the gate runs as `bash -c`.

## On a runner

The delivered workflow runs `kit toolchain install`, then appends the output of `kit toolchain path` to the
runner's `PATH`.

- Its one setup action puts the `mise` binary on the runner when the repository has a `mise.toml`. The action's
  own install step is turned off.
- Homebrew and Docker are already on GitHub's hosted runners.
- The brew provider finds Homebrew at its own prefix when the runner leaves it off `PATH`.

`tests/toolchain.bats` tests each provider against a fake of its tool.
