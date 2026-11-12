# Using mise

[mise](https://mise.jdx.dev) pins each tool by version in `mise.toml`, and resolves each pin in `mise.lock`. For how
kit finds and chooses a manager, see [toolchains.md](../toolchains.md).

## Set up

1. Install mise, such as with `brew install mise`, and activate it in your shell profile: `eval "$(mise activate
   zsh)"`. See [mise's getting started](https://mise.jdx.dev/getting-started.html).
2. Pin the tools under `[tools]` in `mise.toml`, and turn the lock on:

   ```toml
   [settings]
   lockfile = true

   [tools]
   jq = "1.8"
   shellcheck = "0.11"
   ```

3. Run `kit tools install`. It trusts the file, then runs `mise install`.
4. Commit `mise.toml` and `mise.lock`.

## What kit reads

- **Usable**: mise runs offline, and `mise.toml`, `.mise.toml`, or `mise.local.toml` is here. A stale shim or a
  broken install is not usable.
- **Active**: one of those files is here, and mise's tools are on `PATH`. kit does not care how they got there:
  `mise activate`, `mise env`, a harness, or a runner's setup.
- **Pins**: each tool's version, as `mise.lock` resolves it.

Active, in more detail:

- `mise activate` sets `MISE_SHELL`. The others put mise's directories on `PATH`, and kit asks `mise bin-paths`.
- Shims alone do not count. They sit on `PATH` in every directory, set up once per machine, not for this repository.
- The answer is cached with the rest. `PATH`, `MISE_ENV`, and mise's files and locks are part of the key.

| `kit tools` | What mise runs |
| --- | --- |
| `install` | `mise install`, never `mise trust`: that is yours to run, once you have read the file |
| `update` | `mise upgrade` |
| `--outdated` | `mise outdated --json` |

## Backends

A tool's name can carry a backend. `npm:ajv-cli` installs from npm, and `conda:parallel` installs GNU parallel from
conda-forge, which packages it with its own perl.

kit drops the backend when it matches a tool by name. So `npm:bats` counts as `bats`, and brew does not fill in for
it; see [brew fills in](using-brew.md#brew-fills-in).

## A desk and a runner

mise reads `mise.<env>.toml` on top of `mise.toml` when `MISE_ENV=<env>` is set. A repository can split its tools
that way:

- `mise.toml` holds the tools the gate runs. Every desk and runner reads it.
- `mise.desk.toml` adds the tools only a desk runs, such as lefthook. A desk exports `MISE_ENV=desk` in its shell
  profile.
- The CI workflow sets `MISE_ENV=ci`, so a `mise.ci.toml` would hold the tools only a runner needs.

## Hydrating on cd

mise can run `kit hydrate` when a shell enters the repository, so a fresh worktree is complete before its first
command:

```toml
[hooks]
enter = "kit hydrate --quick"
```

- The hook runs only in a shell with `mise activate`. mise's documentation marks hooks experimental.
- `[env]` can put `kit/bin` on `PATH` too: `_.path = ["kit/bin"]`.
- A desk that uses direnv instead gets both from the `.envrc`; see [hydrating on cd](../toolchains.md#hydrating-on-cd).
  mise has deprecated `use mise`, its direnv integration, so the `.envrc` does not activate mise.

## On a runner

Set `ci_toolchain: mise` under `answers:` in `kit.yml`. The workflow puts mise on `PATH` with
`jdx/mise-action`, without installing. Then `kit tools install` installs the pinned tools; see
[packages/github/ci.md](../packages/github/ci.md).
