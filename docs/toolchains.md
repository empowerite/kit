# Toolchains

A toolchain is a tool manager: mise, Nix, Homebrew, a Docker image, a dev container, or a Linux system's package
manager. It pins the tools the gate runs, the gate being the repository's own CI command.

- kit finds the toolchains where it runs. `kit.yml` does not name one for a desk, because everyone who clones
  shares it. CI is the exception: the CI package's `ci_toolchain` answer names the runner's toolchain.
- `kit toolchains` lists what kit found, and records which one to use.
- `kit tools` lists, installs, and updates what the chosen toolchain pins, and runs commands with it. It does
  the same for the packages npm or uv pins on top.

```sh
kit toolchains
kit tools install
```

`kit/modules/bash/toolchain.sh` defines the seam ([architecture.md](architecture.md#forges-seams-and-providers)
says what a seam is). Each manager has a provider, `kit/modules/bash/toolchain/<name>.sh`, that answers its calls.

## Roles

A toolchain fills one role or both:

- **project**: it pins the repository's tools, such as the linters and the gate's own tools.
- **system**: it supplies what kit itself stands on: bash, git, jq, and yq.

A run uses exactly one project toolchain, and at most one system toolchain. kit combines two toolchains' tools on
`PATH` only where brew fills in; see [brew fills in](#brew-fills-in).

## Detection

Each provider answers two questions about the repository it runs in:

- **usable**: its tool is installed and on `PATH`. For the project role, its file is here too. The file counts
  whether committed, ignored, or not yet added.
- **active**: the run is already inside it, as in a shell where mise is activated. Its file must be here too, so a
  marker left over from another project's shell does not count.

Detection runs only when a verb needs it. A probe stats files and runs a tool for its version. It never reaches the
network, and never calls git.

| Provider | Roles | Usable when | Active when | Pins |
| --- | --- | --- | --- | --- |
| `mise` | project | `mise` runs, and `mise.toml`, `.mise.toml`, or `mise.local.toml` is here | one of its files is here, and `MISE_SHELL` is set or every directory `mise bin-paths` prints is on `PATH` | each tool's version, resolved in `mise.lock` |
| `nix` | project | `nix` is on `PATH` with flakes on, and `flake.nix` names `devShells` or `devShell` | `IN_NIX_SHELL` is set, and `flake.nix` is here | each flake input by its locked revision in `flake.lock` |
| `brew` | system; project with a `Brewfile`, on macOS | `brew` is on `PATH` | never | a formula by name; a version only in a name such as `node@22` |
| `docker` | project | `docker` runs, and `toolchain.Dockerfile` is here | never; it runs commands in its container | each base image by digest, `FROM name:tag@sha256:…` |
| `devcontainer` | project; system too while active | From outside, the `devcontainer` CLI is on `PATH`, and `.devcontainer/devcontainer.json` or `.devcontainer.json` is here. Inside, the file alone. | `REMOTE_CONTAINERS`, `CODESPACES`, or `DEVCONTAINER` is set, and the file is here | each feature in `devcontainer-lock.json` by its locked version |
| `apt` | system | `apt-get` is on `PATH` | never | nothing; it installs for the whole machine |
| `dnf` | system | `dnf` is on `PATH` | never | nothing; it installs for the whole machine |
| `pacman` | system | `pacman` is on `PATH` | never | nothing; it installs for the whole machine |

kit asks nix whether `builtins.getFlake` exists, the one sign of flakes that upstream nix and Determinate Nix share.
Determinate Nix treats flakes as stable and does not list them in `experimental-features`.

mise is active however its tools reached `PATH`: `mise activate`, `mise env`, a harness, or a runner's setup.

- `mise activate` sets `MISE_SHELL`. The others put mise's directories on `PATH`, and kit asks `mise bin-paths`.
- Shims alone do not count. They sit on `PATH` in every directory, set up once per machine, not for this repository.
- The answer is cached with the rest. `PATH`, `MISE_ENV`, and mise's files and locks are part of the key.

devcontainer:

- It is offered last, after docker.
- Inside the container it is active, so it is the one chosen for both roles.
- From outside, it runs commands in the container, as docker does. Inside, a command runs as is.
- The file allows comments, which kit does not parse. The pins come from the lock, so a feature not yet locked is
  not listed.

apt, dnf, and pacman:

- They fill the system role alone. They install for the whole machine, so they do not pin any repository's tools.
- Each names the command that installs a missing tool, such as `sudo apt-get install shellcheck`. kit never runs it
  on its own, and `kit tools install` only names it. `kit welcome` runs it only when you say now, in your terminal,
  so `sudo` can ask for a password.
- A package's name can differ from the tool's. Fedora names shellcheck `ShellCheck`; Arch names gh `github-cli`.
- A tool with no package there does not get a command. kit names the tool's site instead. Debian's and Arch's `yq` are
  not mikefarah's, which kit needs, so on apt kit names the site, and on pacman it names `go-yq`.
- With brew on `PATH` too, both are usable for the system. brew is offered first, and the choice is yours:
  `kit toolchains choose apt --system`.

## Choosing

For each role, the first of these that answers is the one:

1. An active toolchain. A run that names a different one is refused, and the refusal says which is active.
2. For the project, the one named for the run: `--toolchain <name>`, or `KIT_TOOLCHAIN=<name>`. One that is not
   usable here is refused; kit never falls back to another.
3. The one recorded earlier, while it is still usable here.
4. The only usable one.
5. With several usable, on a terminal, the person's answer, which kit records.

With several usable and no terminal, as for an agent, kit refuses. The refusal lists each candidate with the
command that picks it, such as `kit toolchains choose mise`. With none usable, kit runs **ambient**: it uses what
is on `PATH`.

`kit toolchains choose <name>` records the project choice, and `--system` records the system one. `ambient` is a
name too: it chooses not to use a toolchain.

## brew fills in

A `Brewfile` often lists what comes before the project toolchain: a newer bash, mise itself, direnv. So brew never
competes with another project toolchain:

- Only on macOS is the `Brewfile` read for the project. Elsewhere, as on a Linux runner, brew fills the system role
  alone.
- Where another project toolchain is usable, brew is not a candidate. `kit toolchains choose brew` is refused.
- Where it is the only one usable, brew is the project toolchain, as any other is.
- Otherwise it fills in for the chosen one: it acts on the `Brewfile`'s tools that toolchain does not pin.

Filling in:

- A tool counts as pinned when its name matches without a backend, a path, or a version. mise's `npm:bats` and
  brew's `user/tap/bats@1` are both `bats`. nix pins flake inputs, not tools, so brew fills in for all of its own.
- `kit tools` lists those tools in a table of their own, after the chosen toolchain's. `kit tools install` installs
  each one missing with `brew install`. `--outdated` and `update` read and move only those.
- `kit tools path`, `kit tools run`, and `kit gate` put brew's directories after the chosen toolchain's.
- docker and devcontainer run commands in a container, which brew's tools never reach. brew fills in for neither.

## Where things are kept

| What | Where | Shared with |
| --- | --- | --- |
| The system choice | `toolchain:` in `~/.kit/settings.yml` | every repository on the machine |
| The project choice, at the desk | `<git common dir>/kit/toolchain` | the desk and every worktree of the clone |
| The project choice, in a worktree | `<its git dir>/kit/toolchain`, in `.git/worktrees/` | that worktree alone |
| What detection found | `.kit/cache/toolchains.tsv`, keyed by `.kit/cache/toolchains.key` | nothing; each checkout has its own |

- Nothing here is committed. Each choice lives in the git directory, never in the working tree.
  `.kit/.gitignore` always lists `/cache/`.
- `git worktree remove` takes a worktree's own choice with it.
- A worktree takes the desk's choice while that toolchain is usable there. Its branch may lack the file, though.
  Then kit asks again, and keeps the answer in the worktree alone.
- The cache holds while `PATH`, where each tool resolves, each provider's files, and each activation marker are
  unchanged. A hit does not run any tool. Discarding the cache never discards a choice.

## `kit toolchains`

Lists each provider, one row for each role it fills. The columns mark whether it is usable, active, and chosen
here. ➕ marks brew where it fills in. `--json` prints one object per row, with `fills` for that.

## `kit tools`

Alone, it lists what the chosen project toolchain pins: each tool's name, the version pinned, and the version
installed. It never reaches the network, so `kit serve` can offer it to agents. docker asks its daemon whether the
image is built only when the daemon is local; with a remote one, no base shows as installed.

| Command | What it does |
| --- | --- |
| `kit tools` | Lists each pinned tool, and whether it is installed. `--json` prints one object per tool. |
| `kit tools --outdated` | Lists only the tools behind their latest version, with that version. This asks the network. `--check` fails when one is behind. A source it cannot read is named, and the run fails. |
| `kit tools install` | Installs the pinned tools: `mise install`, `brew bundle install --no-upgrade` (filling in, `brew install` of what is missing), a `docker build` of the image, or `devcontainer up`. docker tags its image by the file's digest, so an unchanged file is never rebuilt. |
| `kit tools update` | Moves every tool to its latest version, and rewrites the pins: `mise upgrade`, `brew upgrade` of the Brewfile's outdated formulae, new `FROM` digests, or `devcontainer upgrade`. Commit the files it rewrote. |
| `kit tools path` | Prints the directories that put the tools first on `PATH`, for a runner's `GITHUB_PATH`. |
| `kit tools run <command>…` | Runs a command with the tools: inside the container for docker, and for devcontainer from outside; otherwise with the tools first on `PATH`. |

Each takes `--toolchain <name>` to name the toolchain for that run.

Ambient, no toolchain pins anything. Then `kit tools` lists the tools the rules' linters need, and where each is
found on `PATH`, or `missing`.

- `kit tools install` does not install anything. With every tool on `PATH`, it says so and exits 0. Otherwise it fails,
  naming each missing tool, with the system toolchain's command that installs it.
- `kit tools update` refuses when no dependency provider is usable either: there is nothing pinned to update.
  With npm or uv usable, it updates what they pin. `kit tools --outdated` finds no tool behind.
- `kit tools path` prints only the dependency providers' directories, such as `node_modules/.bin`.
  `kit tools run` runs the command with those first on `PATH`, or as is when there are none.

## Dependencies

A toolchain pins tools, node and python among them. A language's package manager pins that language's packages on
top, such as markdownlint-cli2 in `package.json`. Each is a dependency provider, in
`kit/modules/bash/dependencies/<name>.sh`.

| Provider | Usable when | Installs with | Puts on `PATH` | Pins |
| --- | --- | --- | --- | --- |
| `npm` | `npm` is on `PATH`, and `package.json` and `package-lock.json` are here | `npm ci` | `node_modules/.bin` | what `package.json` names, at the lock's versions |
| `uv` | `uv` is on `PATH`, and `pyproject.toml` and `uv.lock` are here | `uv sync --frozen` | `.venv/bin` | every package in the lock but the project |

- A run uses one toolchain, but every usable dependency provider.
- A provider is looked for with the toolchain's directories first on `PATH`, so a node the toolchain pins counts.
- The lock is required. A manifest without its lock does not pin anything, so it is not usable. `kit welcome` names it.
- Only the manifest at the repository root counts. Workspaces are not read yet.
- `kit tools` acts on the toolchain first, then each provider: it lists, installs, updates, and asks what is behind.
  With several, each table is headed by the name of what it lists. A JSON row's `from` names it.
- `kit tools path`, `kit tools run`, and `kit gate` put the toolchain's directories first, then the providers'.
- `kit tools --outdated` asks `npm outdated` and `uv pip list --outdated`. `kit tools update` installs each npm
  package behind at its latest, and runs `uv lock --upgrade`, which stays within `pyproject.toml`'s bounds.
- `kit welcome` checks each provider whose manifest is here, at setup. Its fix is `kit tools install`.
- uv's lock is TOML, which kit reads with yq.

## The gate

`kit gate` runs the gate as `kit tools run` runs a command:

- Normally, with the chosen toolchain's directories first on `PATH`.
- With devcontainer chosen from outside, in the container, through `devcontainer exec`. Run `kit tools install` first,
  which brings the container up. Inside the container, the gate runs as is.
- With docker chosen, inside its image, with the checkout mounted at `/work`. It runs as the caller's user, so the
  files the gate writes belong to the caller. The image needs `bash`, because the gate runs as `bash -c`.

## Hydrating on cd

What kit delivers under `.kit/` is not committed. `kit hydrate` places it from the declaration in
`kit.yml`, and every verb that reads a delivered file does the same first. A repository that uses mise
can hydrate when a shell enters it, so a fresh worktree is complete before its first command:

```toml
[hooks]
enter = "kit hydrate --quick"
```

With `--quick`, the hook does not start or print anything when everything is in place, and does the whole job
otherwise. The hook runs only in a shell with `mise activate`. mise's documentation marks hooks experimental.

A desk that uses direnv gets the same from an `.envrc` of two lines, as this repository's is:

```sh
PATH_add kit/bin
kit hydrate --quick
```

- It does not put any tools on `PATH`. A toolchain that activates through direnv adds its own line, as nix does with
  `use flake`.
- An agent's subprocess shell and a CI runner have neither. There the verbs' own call and `kit ci` do the work.

## On a runner

The delivered workflow runs `kit tools install`, then appends the output of `kit tools path` to the runner's
`PATH`. [packages/github/ci.md](packages/github/ci.md) says what else the job sets up.

- A runner with several usable toolchains has no terminal to ask, so the run names one with `KIT_TOOLCHAIN`.
- The brew provider needs `brew` on `PATH`. `kit welcome` offers the `shellenv` line when brew is installed at a prefix
  but off `PATH`; a runner puts it on `PATH` in its own step.

`tests/toolchains.bats` tests detection and choosing. `tests/tools.bats` tests each provider against a fake of its
tool.
