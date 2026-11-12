# Install

The README gives the Homebrew install. This page has the other ways to install kit, the toolchains that install a
repository's tools, what kit needs on the machine, how kit picks which release to run, and how trust works.

## What a release is

A release is a tarball with `bin/`, `modules/`, `packages/`, `libexec/` (the loader's helper and the `kittui`
binaries that `kit tui` runs), and a `version` file. Install one release on your machine, the way you install your
other tools.

The `kit` you install is a loader. In each repository it runs the release that repository asks for, so one installed
kit serves every repository, whatever release each one pins.

## Ways to install

### Homebrew

On macOS, Linux, or Windows Subsystem for Linux (WSL):

```sh
brew install empowerite/tap/kit
```

### mise

Add kit as a tool in your own `~/.config/mise/config.toml`:

```toml
[tools]
"github:empowerite/kit" = { version = "0.1.0", version_prefix = "release/" }
```

### By hand, with gh

This downloads the latest release, never a pre-release. For one version, run
`gh release download "release/${version}" ...` instead. Unpack it and put its `bin/` on `PATH`:

```sh
gh release download --repo empowerite/kit --pattern 'kit-*.tar.gz' --output kit.tar.gz
mkdir kit && tar --extract --gzip --file kit.tar.gz --directory kit --strip-components 1
export PATH="${PWD}/kit/bin:${PATH}"
```

### Windows

kit runs on Linux and macOS. On Windows, use WSL, which runs it as Linux. Install WSL with `wsl --install` in
PowerShell, then use any of the ways above inside it.

## Toolchains

Installing kit and installing a repository's tools are separate steps. The ways above install kit. A toolchain
installs the tools a repository's work runs with, such as its linters and the tools its gate runs. `kit toolchains`
lists the toolchains kit finds, and `kit tools install` installs what the chosen one pins. kit supports these
toolchains; see [toolchains.md](toolchains.md):

- [kit's own](toolchains/using-kit.md), with `tools:` in `kit.yml`
- [mise](toolchains/using-mise.md)
- [Nix](toolchains/using-nix.md), with flakes
- [Homebrew](toolchains/using-brew.md), with a Brewfile
- [a Docker image](toolchains/using-docker.md)
- [a dev container](toolchains/using-devcontainer.md)
- [apt, dnf, or pacman](toolchains/using-system-packages.md)
- [none](toolchains/using-no-toolchain.md): the tools already on the machine

## Check the install

- `kit version` prints the release.
- `kit help` lists the verbs.
- In a repository that has taken the kit, `kit check` says whether every delivered file matches the release.

## Requirements

- bash 4.4 or later. macOS ships bash 3.2; run `brew install bash`.
- git.
- `jq` 1.7 or later. Every verb reads its arguments through it.
- `yq` 4, mikefarah's. It reads `kit.yml` and every other YAML file. GitHub's hosted runners have `jq` and `yq`
  already.
- `curl` and `tar`. The loader uses them to download a pinned release, and `kit update` to download the previous
  release for a three-way merge when it is not cached. Each download is checked against the `.sha256` published
  beside the release before kit reads anything in it.
- The linters the delivered rules and styles run. `kit lint` fails, rather than skips, when one is missing, so a team
  needs them before its first commit:
  - `shellcheck`, for the bash rule
  - `actionlint`, for the workflows rule
  - `markdownlint-cli2`, for the markdown style, with the `style/markdown` package
- The code forge's command-line tool: `gh` for GitHub or `glab` for GitLab. The code forge is the service that hosts the
  repository. These verbs need it: `kit work`, `kit assign`, `kit unassign`, `kit close`, `kit gather`,
  `kit settle`, `kit flow`, `kit tidy`, `kit linked`, and `kit release`. `kit commit` uses it to close an issue its
  subject names, and `kit policy` uses it in CI to read the pull request.
- For an Azure DevOps tracker: `az`, the Azure CLI.

## Which release runs

kit uses the first of these that is set:

1. `kit --version=<release> …`, for this call and every kit call it makes in the same clone.
2. `KIT_VERSION=<release>` in the environment, which does the same.
3. `version:` in the `kit.yml` at the root of the clone you are in. Each worktree is its own root.
4. The kit you installed. This applies outside a clone, and in a clone that pins no version.

A release is named by its version, such as `0.1.0`. `--version=` and `KIT_VERSION` also take the path of a copy of a
release.

The loader keeps every release it runs in `~/.cache/kit/`. It downloads a version the first time a repository asks
for it, and checks it against the `.sha256` published beside it.

The loader refuses a kit option it does not know. A later kit may know it, so update the kit you installed.

## Trust

kit runs nothing in a clone until you trust it.

- `kit trust` shows what the clone asks kit to run: its pinned release, the bash it loads, and its gate. Then it asks
  you, and remembers your answer for the clone and all its worktrees.
- In a terminal, the first kit command in a clone you have not trusted asks the same question, and runs after a yes.
- `kit trust ~/Code/` trusts every clone under that folder.
- `kit trust --list` and `kit trust --revoke` read and change the records, which are plain lines in
  `~/.config/kit/trusted`.

Trusting a clone means you accept code from anyone who can push to it. kit does not check each pull; code review and
signed commits do.
