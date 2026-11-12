# Using Homebrew

[Homebrew](https://brew.sh) installs formulae and casks. A `Brewfile` at the repository root lists them. For how kit
finds and chooses a manager, see [toolchains.md](../toolchains.md).

brew fills two roles:

- **system**, wherever `brew` is on `PATH`: it supplies what kit stands on, such as bash, jq, and yq.
- **project**, on macOS with a `Brewfile`. On Linux, as on a runner, kit does not read the `Brewfile`.

brew is never active: it is not a shell one enters.

## Set up

1. Install Homebrew; see [brew.sh](https://brew.sh). If `brew` is installed but not on `PATH`, `kit welcome` offers
   the `shellenv` line that puts it there.
2. List the tools in a `Brewfile`:

   ```ruby
   brew "bash"
   brew "jq"
   brew "yq"
   ```

3. Run `kit tools install`, which runs `brew bundle install --no-upgrade`.
4. Commit the `Brewfile`.

A `Brewfile` pins a formula by name only, so its version moves with Homebrew. A version is pinned only where the name
holds one, such as `node@22`.

| `kit tools` | What brew runs |
| --- | --- |
| `install` | `brew bundle install --no-upgrade`; filling in, `brew install` of what is missing |
| `update` | `brew upgrade` of the `Brewfile`'s outdated formulae |
| `--outdated` | `brew outdated --json=v2`, for the `Brewfile`'s formulae |

## brew fills in

A `Brewfile` often lists what comes before the tool manager: a newer bash, mise itself, direnv. So brew never
competes with another tool manager:

- Only on macOS is the `Brewfile` read for the project. Elsewhere, as on a Linux runner, brew fills the system role
  alone.
- Where another tool manager is usable, brew is not a candidate. `kit toolchains use brew` is refused.
- Where it is the only one usable, brew is the project's tool manager, as any other is.
- Otherwise it fills in for the chosen one: it acts on the `Brewfile`'s tools that toolchain does not pin.

Filling in:

- A tool counts as pinned when its name matches without a backend, a path, or a version. mise's `npm:bats` and
  brew's `user/tap/bats@1` are both `bats`. nix pins flake inputs, not tools, so brew fills in for all of its own.
- `kit tools` lists those tools in a table of their own, after the chosen toolchain's. `kit tools install` installs
  each one missing with `brew install`. `--outdated` and `update` read and move only those.
- `kit tools path`, `kit tools run`, and `kit gate` put brew's directories after the chosen toolchain's.
- docker and devcontainer run commands in a container, which brew's tools never reach. brew fills in for neither.
- `kit toolchains` marks brew with ➕ where it fills in.

## On a runner

GitHub's macOS and Linux runners have brew. The brew provider needs `brew` on `PATH`, so a runner where it is off
`PATH` puts it there in a step of its own.
