# Toolchains

A toolchain is a tool manager: mise, Homebrew, or a Docker container image. It pins the tools the gate runs, the
gate being the repository's own CI command.

- kit finds the toolchains where it runs. `kit.yml` never names one, because it is shared by everyone who clones.
- `kit toolchains` lists what kit found, and records which one to use.
- `kit tools` lists, installs, and updates what the chosen toolchain pins, and runs commands with it.

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

A run uses exactly one project toolchain, and at most one system toolchain. kit never combines two toolchains'
tools on `PATH`.

## Detection

Each provider answers two questions about the repository it runs in:

- **usable**: its tool is installed and on `PATH`, and its file is here. The file counts whether committed,
  ignored, or not yet added.
- **active**: the run is already inside it, as in a shell where mise is activated. Its file must be here too, so a
  marker left over from another project's shell does not count.

Detection runs only when a verb needs it. A probe stats files and runs a tool for its version. It never reaches the
network, and never calls git.

| Provider | Roles | Usable when | Active when | Pins |
| --- | --- | --- | --- | --- |
| `mise` | project | `mise` runs, and `mise.toml`, `.mise.toml`, or `mise.local.toml` is here | `MISE_SHELL` is set | each tool's version, resolved in `mise.lock` |
| `brew` | system; project with a `Brewfile` | `brew` is on `PATH` or at one of Homebrew's prefixes | never | a formula by name; a version only in a name such as `node@22` |
| `docker` | project | `docker` runs, and `toolchain.Dockerfile` is here | never; it runs commands in its container | each base image by digest, `FROM name:tag@sha256:…` |

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
name too: it chooses none.

## Where things are kept

| What | Where | Shared with |
| --- | --- | --- |
| The system choice | `toolchain:` in `~/.kit/settings.yml` | every repository on the machine |
| The project choice, at the desk | `<git common dir>/kit/toolchain` | the desk and every worktree of the clone |
| The project choice, in a worktree | `.kit/cache/toolchain` | that worktree alone |
| What detection found | `.kit/cache/toolchains.tsv`, keyed by `.kit/cache/toolchains.key` | nothing; each checkout has its own |

- Nothing here is committed. `.kit/.gitignore` always lists `/cache/`.
- A worktree takes the desk's choice while that toolchain is usable there. Its branch may lack the file, though.
  Then kit asks again, and keeps the answer in the worktree alone.
- The cache holds while `PATH`, where each tool resolves, each provider's files, and each activation marker are
  unchanged. A hit runs no tool. Discarding the cache never discards a choice.

## `kit toolchains`

Lists each provider, one row for each role it fills. The columns mark whether it is usable, active, and chosen
here. `--json` prints one object per row.

## `kit tools`

Alone, it lists what the chosen project toolchain pins: each tool's name, the version pinned, and the version
installed. It never reaches the network, so `kit serve` can offer it to agents. docker asks its daemon whether the
image is built only when the daemon is local; with a remote one, no base shows as installed.

| Command | What it does |
| --- | --- |
| `kit tools` | Lists each pinned tool, and whether it is installed. `--json` prints one object per tool. |
| `kit tools --outdated` | Lists only the tools behind their latest version, with that version. This asks the network. `--check` fails when one is behind. |
| `kit tools install` | Installs the pinned tools: `mise install`, `brew bundle install --no-upgrade`, or a `docker build` of the image. The image is tagged by the file's digest, so an unchanged file is never rebuilt. |
| `kit tools update` | Moves every tool to its latest version, and rewrites the pins: `mise upgrade`, `brew upgrade` of the Brewfile's outdated formulae, or new `FROM` digests. Commit the files it rewrote. |
| `kit tools path` | Prints the directories that put the tools first on `PATH`, for a runner's `GITHUB_PATH`. |
| `kit tools run <command>…` | Runs a command with the tools: inside the container for docker, otherwise with the tools first on `PATH`. |

Each takes `--toolchain <name>` to name the toolchain for that run.

Ambient, no toolchain pins anything. Then `kit tools` lists the tools the rules' linters need, and where each is
found on `PATH`, or `missing`.

- `kit tools install` installs nothing. With every tool on `PATH`, it says so and exits 0. Otherwise it fails,
  naming each missing tool, and the brew command when brew is the system toolchain.
- `kit tools update` refuses: there is nothing pinned to update. `kit tools --outdated` finds none behind.
- `kit tools path` prints nothing, and `kit tools run` runs the command as is.

## The gate

`kit gate` runs the gate as `kit tools run` runs a command:

- Normally, with the chosen toolchain's directories first on `PATH`.
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

With `--quick`, the hook starts nothing and says nothing when everything is in place, and does the whole job
otherwise. The hook runs only in a shell with `mise activate`. A desk that uses direnv instead gets the same from an
`.envrc` of two lines, `use mise` and `kit hydrate --quick`, as this repository's does. An agent's subprocess shell
and a CI runner have neither, and there the verbs' own call and `kit ci` do the work. mise's documentation marks
hooks experimental.

## On a runner

The delivered workflow runs `kit tools install`, then appends the output of `kit tools path` to the runner's
`PATH`. [packages/github/ci.md](packages/github/ci.md) says what else the job sets up.

- A runner with several usable toolchains has no terminal to ask, so the run names one with `KIT_TOOLCHAIN`.
- The brew provider finds Homebrew at its own prefix when the runner leaves it off `PATH`.

`tests/toolchains.bats` tests detection and choosing. `tests/tools.bats` tests each provider against a fake of its
tool.
