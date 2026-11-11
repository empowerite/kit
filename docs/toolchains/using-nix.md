# Using nix

[Nix](https://nixos.org) builds a dev shell from `flake.nix`, and `flake.lock` pins each of the flake's inputs by
revision. kit's nix toolchain works with flakes only. For how kit finds and chooses a toolchain, see
[toolchains.md](../toolchains.md).

## Set up

1. Install nix with flakes on. The [Determinate installer](https://docs.determinate.systems) turns them on:

   ```sh
   curl -fsSL https://install.determinate.systems/nix | sh -s -- install
   ```

   With the [upstream installer](https://nixos.org/download), turn them on yourself: add
   `experimental-features = nix-command flakes` to `~/.config/nix/nix.conf`.
2. Write a `flake.nix` with a dev shell:

   ```nix
   {
     inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
     outputs = { self, nixpkgs }:
       let pkgs = nixpkgs.legacyPackages.aarch64-darwin;
       in { devShells.aarch64-darwin.default = pkgs.mkShell { packages = [ pkgs.jq pkgs.shellcheck ]; }; };
   }
   ```

3. Run `git add flake.nix`. Nix ignores a `flake.nix` that git does not track.
4. Run `kit tools install`. It runs `nix develop --command true`, which builds the shell and writes `flake.lock`.
5. Commit `flake.nix` and `flake.lock`.

## What kit reads

- **Usable**: `nix` is on `PATH`, `flake.nix` names `devShells` or `devShell`, and flakes are on. kit greps the file
  and never evaluates it, so a comment naming `devShells` counts.
- **Active**: `IN_NIX_SHELL` is set, as inside `nix develop`, and `flake.nix` is here.
- **Pins**: each flake input by its locked revision. A tool's version is whatever the pinned nixpkgs holds.

Flakes are on where `builtins.getFlake` exists, and kit asks nix that. It is the one sign of flakes that upstream nix
and Determinate Nix share. Determinate Nix treats flakes as stable, so it does not list them in
`experimental-features`.

| `kit tools` | What nix runs |
| --- | --- |
| `install` | `nix develop --command true` |
| `update` | `nix flake update` |
| `--outdated` | `nix flake metadata --json` for each input's original reference. It can say only that a revision differs. |
| `path` | `nix print-dev-env --json`, which evaluates the flake |

nix pins inputs, not tools. So with a `Brewfile` on macOS, brew fills in for all of its tools; see
[brew fills in](using-brew.md#brew-fills-in).

## Entering the shell

- `nix develop` enters the default dev shell. `nix develop .#ci` enters one named `ci`.
- With direnv, install [nix-direnv](https://github.com/nix-community/nix-direnv) and add `use flake` to `.envrc`. It
  caches the shell under `.direnv/`, which `.gitignore` lists.
- A dev shell's `shellHook` can do what mise's enter hook does: put `kit/bin` on `PATH` and run
  `kit hydrate --quick`.

## On a runner

Set `ci_toolchain: nix` under `answers:` in `kit.yml`. The workflow installs nix with
`DeterminateSystems/nix-installer-action`. Then `kit tools install` builds the dev shell; see
[packages/github/ci.md](../packages/github/ci.md).

## This repository's flake

`flake.nix` ports `tools:` in `kit.yml`, which the kit toolchain installs:

- `devShells.ci` holds the tools the gate runs, every one but lefthook, as a runner installs them.
- `devShells.default` adds lefthook, which `kit.yml` keeps to a desk. `nix develop` and kit enter it.
- Both put `kit/bin` on `PATH` and hydrate `.kit/`.
- Neither holds the npm tools, ajv-cli and markdownlint-cli2. `package.json` pins them for every toolchain, and
  `kit tools install` runs `npm ci` with the shell's node.

Where it differs from the kit toolchain:

- The versions follow nixpkgs, so some sit a minor version off `kit.yml`'s pins. go is `go_1_27`, because nixpkgs'
  default `go` is older.
- An Intel Mac is left out. nixpkgs 26.11 dropped `x86_64-darwin`.

On a desk with both, kit and nix are usable, and kit uses the one recorded. `kit toolchains choose nix` switches.
