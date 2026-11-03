# flake.nix - the nix dev shells: the tools mise.toml and mise.desk.toml pin, for a desk or runner that uses nix
# ----------------------------------------------------------------------------------------------------------------------
# A port of mise.toml for the nix toolchain; see: docs/toolchains.md. Each mise tool is the nixpkgs package of that
# name, and flake.lock pins nixpkgs by revision, as mise.lock pins each tool by version. The versions follow nixpkgs,
# so they can sit a minor version off mise's pins.
#
#   - devShells.ci is mise.toml: the tools the gate runs.
#   - devShells.default is mise.toml and mise.desk.toml: the gate's tools and lefthook. `nix develop` enters it, and
#     kit's nix toolchain installs from it.
#
# Both put kit/bin on PATH and hydrate .kit/, as mise.toml's [env] and enter hook do. git and bash come from the
# machine. Nix ignores a flake.nix git does not track, so a new file here needs `git add` before `nix develop`.
#
# nixpkgs does not package ajv-cli, so it is built here from nix/ajv-cli/, whose package-lock.json pins it and its
# dependencies.

{
  description = "kit: keep it trustworthy";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      # the platforms mise.toml's lockfile_platforms covers
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];
      forEachSystem = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      ajv-cli = pkgs: pkgs.buildNpmPackage {
        pname = "ajv-cli";
        version = "5.0.0";
        src = ./nix/ajv-cli;
        npmDepsHash = "sha256-iEYIsU4nhNk0X2khfTdfXpjktivN7Dlum5yUb9JleR0=";
        dontNpmBuild = true;
        postInstall = ''
          makeWrapper ${pkgs.nodejs}/bin/node $out/bin/ajv \
            --add-flags $out/lib/node_modules/kit-ajv-cli/node_modules/ajv-cli/dist/index.js
        '';
      };

      # Why each tool is here: see mise.toml, which says it for each.
      gate = pkgs: [
        pkgs.actionlint
        pkgs.bats
        pkgs.gh
        pkgs.go_1_27
        pkgs.jq
        pkgs.just
        pkgs.lychee
        (ajv-cli pkgs)
        pkgs.markdownlint-cli2
        pkgs.open-policy-agent
        pkgs.parallel
        pkgs.shellcheck
        pkgs.yq-go
      ];

      # kit/bin on PATH, from the repository root wherever the shell starts, then .kit/ hydrated, quietly when it is
      shellHook = ''
        root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
        export PATH="$root/kit/bin:$PATH"
        kit hydrate --quick
      '';
    in
    {
      devShells = forEachSystem (pkgs: {
        ci = pkgs.mkShell { packages = gate pkgs; inherit shellHook; };
        default = pkgs.mkShell { packages = gate pkgs ++ [ pkgs.lefthook ]; inherit shellHook; };
      });
    };
}
