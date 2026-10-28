# Toolchains

The gate runs tools, and which tool manager puts them on a desk or a runner is the repository's decision: mise,
Homebrew, or a container image. The kit asks every one the same questions through one seam, `toolchain:` in
`bin/toolchain.sh`, so a desk, a runner, and the delivered CI job say the same thing whichever manages the tools.

## Declaring

A repository declares its toolchains in `.kit/settings.yml`, in order, or the kit detects them from the files each
one reads:

```yaml
toolchains:
  - provider: mise
  - provider: brew
    file: Brewfile
  - provider: docker
    file: toolchain.Dockerfile
    image: example/toolchain
```

| Provider | Detected from | Pins | Outdated means |
| --- | --- | --- | --- |
| `mise` | `mise.toml` or `.mise.toml` | each tool's version, resolved in `mise.lock` | `mise outdated` names it |
| `brew` | `Brewfile` | a formula by name; a version only where the name carries it, `node@22` | `brew outdated` names it |
| `docker` | `toolchain.Dockerfile` | each base image by digest, `FROM name:tag@sha256:…` | the registry's digest for the tag differs, or there is none |

## The verbs

- `kit toolchain install` installs every declared toolchain's pinned tools: `mise install`, `brew bundle install
  --no-upgrade`, or a `docker build` of the image, tagged by the file's digest so an unchanged file is never rebuilt.
- `kit toolchain outdated` prints one line per tool behind its source, and `--check` fails when there is one.
- `kit toolchain update` moves every tool to its latest and rewrites the pins: `mise upgrade`, `brew upgrade` of the
  Brewfile's outdated formulae, or the Dockerfile's `FROM` digests. Commit what it rewrote.
- `kit toolchain path` prints the directories that put the tools first on `PATH`, for a runner's `GITHUB_PATH`.
- `kit toolchain run <command>…` runs a command with the tools.

`kit gate` runs the gate the same way: with every toolchain's path first on `PATH`, or, where docker is declared,
inside the image, the checkout mounted at `/work` and the caller's user, so what the gate writes is the caller's.
The image needs `bash`, since the gate runs as `bash -c`.

## On a runner

The delivered workflow runs `kit toolchain install` and appends `kit toolchain path` to the runner's `PATH`. Its
one setup action puts the `mise` binary on the runner where the repository has a `mise.toml`, with its own install
turned off; Homebrew and Docker are already on GitHub's hosted runners, and the brew provider finds Homebrew at
its own prefix where the runner leaves it off `PATH`.

`tests/toolchain.bats` drives each provider through a fake of its tool.
