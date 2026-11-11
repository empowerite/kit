# flake.nix - the nix dev shells: the tools tools: in kit.yml pins, for a desk or runner that uses nix
# ----------------------------------------------------------------------------------------------------------------------
# A port of tools: in kit.yml for the nix toolchain; see: docs/toolchains/using-nix.md. Each tool is the nixpkgs
# package of that name, and flake.lock pins nixpkgs by revision, as tools.lock pins each tool's download. The versions
# follow nixpkgs, so they can sit a minor version off kit.yml's pins.
#
#   - devShells.ci holds the tools the gate runs: every tool but lefthook, as a runner installs them.
#   - devShells.default adds lefthook, as a desk installs it. `nix develop` enters it, and kit's nix toolchain installs
#     from it.
#
# Both put kit/bin on PATH and hydrate .kit/, as the .envrc does. git and bash come from the
# machine. Nix ignores a flake.nix git does not track, so a new file here needs `git add` before `nix develop`.
#
# The npm tools, ajv-cli and markdownlint-cli2, are pinned in package.json, which every toolchain shares; the shell
# brings node, and kit tools install runs npm ci.

{
  description = "kit: keep it trustworthy";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      # the platforms tools.lock covers, but an Intel Mac: nixpkgs 26.11 dropped x86_64-darwin
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" ];
      forEachSystem = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});

      # Why each tool is here: see tools: in kit.yml, which says it for each.
      gate = pkgs: [
        pkgs.actionlint
        pkgs.bats
        pkgs.gh
        pkgs.go_1_27
        pkgs.jq
        pkgs.just
        pkgs.lychee
        pkgs.nodejs_24
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
